package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Greetings from the Weather CLI Tool")
	fmt.Println("Please enter the desirable location to check weather: ")
	city := ""

	if scanner.Scan() {
		city = scanner.Text()
	}

	coords, err := getCoordinates(city)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	weather, err := getWeather(coords.Result[0].Latitude, coords.Result[0].Longitude)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("The temperature in %s is: %.1f°C \n", city, weather.CurrentWeather.Temperature)
	fmt.Printf("The windspeed in %s is: %.1f km/h\n", city, weather.CurrentWeather.Windspeed)
}
