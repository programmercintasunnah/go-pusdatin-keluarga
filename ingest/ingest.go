package ingest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GeoResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

type WeatherResponse struct {
	Hourly struct {
		Temperature2m []float64 `json:"temperature_2m"`
		Time          []string  `json:"time"`
	} `json:"hourly"`
}

func IngestWeatherData(db *sql.DB) error {
	resp, err := http.Get("https://geocoding-api.open-meteo.com/v1/search?name=Jakarta")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var geo GeoResponse
	json.Unmarshal(body, &geo)
	if len(geo.Results) == 0 {
		return fmt.Errorf("Jakarta tidak ditemukan")
	}
	lat := geo.Results[0].Latitude
	lon := geo.Results[0].Longitude

	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&hourly=temperature_2m", lat, lon)
	resp2, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	var weather WeatherResponse
	json.Unmarshal(body2, &weather)

	if len(weather.Hourly.Temperature2m) > 0 {
		temp := weather.Hourly.Temperature2m[len(weather.Hourly.Temperature2m)-1]
		_, err := db.Exec(`
			INSERT INTO weather_data (city, temperature, weather_desc, collected_at)
			VALUES ($1, $2, $3, $4)
		`, "Jakarta", temp, "Temperature (°C)", time.Now())
		if err != nil {
			return err
		}
		fmt.Println("✅ Data cuaca disimpan:", temp)
	}
	return nil
}
