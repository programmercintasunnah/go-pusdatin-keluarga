package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]string)
var broadcast = make(chan Message)

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
	Target   string `json:"target,omitempty"`
	Type     string `json:"type"` // "broadcast" | "private" | "group"
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, _ := upgrader.Upgrade(w, r, nil)
	defer ws.Close()
	clients[ws] = ""

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}

		if clients[ws] == "" {
			clients[ws] = msg.Username
		}
		broadcast <- msg
	}
}

func HandleMessages() {
	for {
		msg := <-broadcast
		for client, username := range clients {
			switch msg.Type {
			case "broadcast":
				client.WriteJSON(msg)
			case "private":
				if username == msg.Target || username == msg.Username {
					client.WriteJSON(msg)
				}
			case "group":
				client.WriteJSON(msg)
			}
		}
		fmt.Println("📩", msg.Type, ":", msg.Username, "->", msg.Message)
	}
}
