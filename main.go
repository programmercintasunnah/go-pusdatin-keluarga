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

	// Jalankan server API
	http.HandleFunc("/api/weather/current", api.GetCurrentWeather(db))
	http.HandleFunc("/api/weather/history", api.GetWeatherHistory(db))
	http.HandleFunc("/ws", api.HandleConnections)
	go api.HandleMessages()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
