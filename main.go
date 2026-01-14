package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

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

func main() {
	http.HandleFunc("/weather", weatherHandler)

	http.HandleFunc("/", homeHandler)

	fmt.Println("Server started! Visit at http://localhost:8080/weather?city=London")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}
