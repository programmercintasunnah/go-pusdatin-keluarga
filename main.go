package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"test-backend/api"
	"test-backend/config"
	"test-backend/ingest"
)

func main() {
	db := config.ConnectDB()
	defer db.Close()

	// Weather endpoints
	http.HandleFunc("/api/weather/current", api.GetCurrentWeather(db))
	http.HandleFunc("/api/weather/history", api.GetWeatherHistory(db))

	// Chat endpoints (WebSocket)
	http.HandleFunc("/ws", api.HandleConnections)
	go api.HandleMessages()

	// Scheduler for ingest every 15 min
	go func() {
		for {
			_ = ingest.IngestWeatherData(db)
			time.Sleep(15 * time.Minute)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
