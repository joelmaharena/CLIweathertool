package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

type SearchEntry struct {
	ID         int64  `json:"id"`
	City       string `json:"city"`
	Searchtime string `json:"search_time"`
}
type GeocodingResponse struct {
	Result []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

type WeatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		Windspeed   float64 `json:"windspeed"`
		Weathercode int16   `json:"weathercode"`
	} `json:"current_weather"`
}

func initDB() {
	var err error
	if err = godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Could not connect to Docker Postgres: ", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS search_history (
    id SERIAL PRIMARY KEY, 
    city TEXT, 
    search_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)

	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	fmt.Println("Succesfully connected to the Database!")

}

func getCoordinates(city string) (GeocodingResponse, error) {
	safeCity := url.QueryEscape(city)
	apiUrl := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", safeCity)
	res, err := http.Get(apiUrl)

	if err != nil {
		return GeocodingResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return GeocodingResponse{}, err
	}

	var response GeocodingResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return GeocodingResponse{}, err
	}

	if len(response.Result) == 0 {
		return response, fmt.Errorf("City '%s' not found", city)
	}

	return response, nil

}
func getWeather(latitude float64, longitude float64) (WeatherResponse, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true", latitude, longitude)

	res, err := http.Get(url)

	if err != nil {
		return WeatherResponse{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return WeatherResponse{}, err
	}

	var response WeatherResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return WeatherResponse{}, err
	}

	return response, nil

}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	city := r.URL.Query().Get("city")
	if city == "" {
		http.Error(w, "Please provide a city parameter such as ?city=London", http.StatusBadRequest)
		return
	}
	coords, err := getCoordinates(city)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Saving the city %s to the database...\n", city)
	_, dbErr := db.Exec("INSERT INTO search_history (city) VALUES ($1)", city)
	if dbErr != nil {
		fmt.Println("DB Error:", dbErr)
	}

	weather, err := getWeather(coords.Result[0].Latitude, coords.Result[0].Longitude)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var weatherDescription string
	switch weather.CurrentWeather.Weathercode {
	case 0:
		weatherDescription = "Clear Sky☀️"
	case 1, 2, 3:
		weatherDescription = "Cloudy ☁️"
	case 61, 63, 65:
		weatherDescription = "Rain🌧️"
	default:
		weatherDescription = "Unknown"
	}
	response := map[string]any{
		"city":               city,
		"temperature":        weather.CurrentWeather.Temperature,
		"windspeed":          weather.CurrentWeather.Windspeed,
		"weatherDescription": weatherDescription,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, city, search_time FROM search_history ORDER BY id DESC LIMIT 10")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []SearchEntry

	for rows.Next() {
		var entry SearchEntry
		if err := rows.Scan(&entry.ID, &entry.City, &entry.Searchtime); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		history = append(history, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func main() {
	initDB()

	http.HandleFunc("/weather", weatherHandler)
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/history", historyHandler)

	fmt.Println("Server started! Visit at http://localhost:8080/")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
