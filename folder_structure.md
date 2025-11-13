test-backend/
│
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env
│
├── main.go
│
├── config/
│   └── db.go
│
├── ingest/
│   └── ingest.go
│
├── api/
│   ├── weather_handler.go
│   └── chat_handler.go
│
├── db/
│   └── init.sql
│
└── DESIGN.md   ← berisi mini system design untuk 1 juta request/hari

🚀 Jalankan!
docker-compose up --build


Lalu buka:

🌤 http://localhost:8080/api/weather/current
📜 http://localhost:8080/api/weather/history
💬 WebSocket Chat: ws://localhost:8080/ws
