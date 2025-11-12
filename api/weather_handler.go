package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type WeatherData struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	WeatherDesc string  `json:"weather_desc"`
	CollectedAt string  `json:"collected_at"`
}

func GetCurrentWeather(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		row := db.QueryRow(`SELECT city, temperature, weather_desc, collected_at 
							FROM weather ORDER BY collected_at DESC LIMIT 1`)
		var data WeatherData
		row.Scan(&data.City, &data.Temperature, &data.WeatherDesc, &data.CollectedAt)
		json.NewEncoder(w).Encode(data)
	}
}

func GetWeatherHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query(`SELECT city, temperature, weather_desc, collected_at 
							 FROM weather ORDER BY collected_at DESC LIMIT 10`)
		defer rows.Close()

		var list []WeatherData
		for rows.Next() {
			var d WeatherData
			rows.Scan(&d.City, &d.Temperature, &d.WeatherDesc, &d.CollectedAt)
			list = append(list, d)
		}
		json.NewEncoder(w).Encode(list)
	}
}
