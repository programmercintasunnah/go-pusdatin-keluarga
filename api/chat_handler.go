package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn     *websocket.Conn
	username string
	groups   map[string]bool // groups yang diikuti user ini
}

var (
	clients   = make(map[*websocket.Conn]*Client)
	clientsMu sync.RWMutex
	broadcast = make(chan Message, 100)
	groups    = make(map[string]map[*websocket.Conn]bool) // groupName -> clients
	groupsMu  sync.RWMutex
)

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
	Target   string `json:"target,omitempty"` // untuk private/group
	Type     string `json:"type"`             // "broadcast" | "private" | "group" | "join_group"
	SentAt   string `json:"sent_at,omitempty"`
}

func HandleConnections(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("❌ Upgrade error: %v", err)
			return
		}
		defer ws.Close()

		// Buat client baru
		client := &Client{
			conn:   ws,
			groups: make(map[string]bool),
		}

		clientsMu.Lock()
		clients[ws] = client
		clientsMu.Unlock()

		log.Printf("✅ New connection. Total clients: %d", len(clients))

		// Baca messages dari client ini
		for {
			var msg Message
			err := ws.ReadJSON(&msg)
			if err != nil {
				log.Printf("❌ Read error: %v", err)
				clientsMu.Lock()
				delete(clients, ws)
				clientsMu.Unlock()

				// Remove from all groups
				groupsMu.Lock()
				for groupName := range client.groups {
					if groupClients, exists := groups[groupName]; exists {
						delete(groupClients, ws)
					}
				}
				groupsMu.Unlock()

				log.Printf("🔌 Client disconnected. Total clients: %d", len(clients))
				break
			}

			// Set username pertama kali
			if client.username == "" && msg.Username != "" {
				client.username = msg.Username
				log.Printf("👤 User registered: %s", msg.Username)
			}

			// Handle join group
			if msg.Type == "join_group" && msg.Target != "" {
				groupsMu.Lock()
				if groups[msg.Target] == nil {
					groups[msg.Target] = make(map[*websocket.Conn]bool)
				}
				groups[msg.Target][ws] = true
				client.groups[msg.Target] = true
				groupsMu.Unlock()

				log.Printf("👥 %s joined group: %s", client.username, msg.Target)
				continue
			}

			// Set timestamp
			msg.SentAt = time.Now().Format(time.RFC3339)
			msg.Username = client.username

			// Simpan ke database (kecuali join_group)
			if msg.Type != "join_group" {
				saveMessageToDB(db, msg)
			}

			// Kirim ke channel broadcast
			broadcast <- msg
		}
	}
}

func HandleMessages() {
	for msg := range broadcast {
		clientsMu.RLock()

		switch msg.Type {
		case "broadcast":
			// Kirim ke semua client
			for conn, _ := range clients {
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Printf("❌ Write error: %v", err)
					conn.Close()
					delete(clients, conn)
				}
			}
			log.Printf("📢 BROADCAST from %s: %s", msg.Username, msg.Message)

		case "private":
			// Kirim hanya ke target dan sender
			for conn, client := range clients {
				if client.username == msg.Target || client.username == msg.Username {
					err := conn.WriteJSON(msg)
					if err != nil {
						log.Printf("❌ Write error: %v", err)
						conn.Close()
						delete(clients, conn)
					}
				}
			}
			log.Printf("💬 PRIVATE %s -> %s: %s", msg.Username, msg.Target, msg.Message)

		case "group":
			// Kirim ke semua member group
			groupsMu.RLock()
			if groupClients, exists := groups[msg.Target]; exists {
				for conn := range groupClients {
					err := conn.WriteJSON(msg)
					if err != nil {
						log.Printf("❌ Write error: %v", err)
						conn.Close()
						delete(clients, conn)
					}
				}
				log.Printf("👥 GROUP [%s] from %s: %s", msg.Target, msg.Username, msg.Message)
			}
			groupsMu.RUnlock()
		}

		clientsMu.RUnlock()
	}
}

func saveMessageToDB(db *sql.DB, msg Message) {
	query := `INSERT INTO chat_messages (username, message, sent_at) VALUES ($1, $2, $3)`

	sentAt, err := time.Parse(time.RFC3339, msg.SentAt)
	if err != nil {
		sentAt = time.Now()
	}

	_, err = db.Exec(query, msg.Username, msg.Message, sentAt)
	if err != nil {
		log.Printf("❌ Failed to save message to DB: %v", err)
	}
}

// API untuk get chat history
func GetChatHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := `
			SELECT username, message, sent_at 
			FROM chat_messages 
			ORDER BY sent_at DESC 
			LIMIT 50
		`

		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		messages := []Message{}
		for rows.Next() {
			var msg Message
			var sentAt time.Time
			if err := rows.Scan(&msg.Username, &msg.Message, &sentAt); err != nil {
				continue
			}
			msg.SentAt = sentAt.Format(time.RFC3339)
			messages = append(messages, msg)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

// API untuk get active users
func GetActiveUsers(w http.ResponseWriter, r *http.Request) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	users := []string{}
	for _, client := range clients {
		if client.username != "" {
			users = append(users, client.username)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(users),
		"users": users,
	})
}
