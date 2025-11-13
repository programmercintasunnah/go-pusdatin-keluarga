package main

import (
	"log"
	"net/http"
	"os"

	"pu5d4t1n-k3lu4r64-simple/api"
	"pu5d4t1n-k3lu4r64-simple/config"
	"pu5d4t1n-k3lu4r64-simple/ingest"
)

func main() {
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	db := config.ConnectDB()
	defer db.Close()

	if mode == "--ingest" {
		log.Println("🌦️ Running ingest mode...")
		if err := ingest.IngestWeatherData(db); err != nil {
			log.Println("❌ Gagal ingest:", err)
		} else {
			log.Println("✅ Ingest selesai.")
		}
		return
	}

	// Weather API endpoints
	http.HandleFunc("/api/weather/current", api.GetCurrentWeather(db))
	http.HandleFunc("/api/weather/history", api.GetWeatherHistory(db))

	// Chat endpoints
	http.HandleFunc("/ws", api.HandleConnections(db))
	http.HandleFunc("/api/chat/history", api.GetChatHistory(db))
	http.HandleFunc("/api/chat/users", api.GetActiveUsers)

	// Start message handler goroutine
	go api.HandleMessages()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port:", port)
	log.Println("📡 WebSocket endpoint: ws://localhost:" + port + "/ws")
	log.Println("🌐 Weather API: http://localhost:" + port + "/api/weather/current")
	log.Println("💬 Chat History: http://localhost:" + port + "/api/chat/history")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
