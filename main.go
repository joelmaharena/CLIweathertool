package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func getCoordinates(city string) GeocodingResponse {
	safeCity := url.QueryEscape(city)
	apiUrl := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", safeCity)
	res, err := http.Get(apiUrl)

	if err != nil {
		log.Fatal("Failed response from the Geocoding server: ", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		log.Fatal("Could not read the response from Geocoding: ", err)
	}

	var response GeocodingResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal("Could not parse the json: ", err)
	}

	return response

}
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Greetings from the Weather CLI Tool")
	fmt.Println("Please enter the desirable location to check weather: ")
	input := ""
	if scanner.Scan() {
		input = scanner.Text()
	}
	var georesponse GeocodingResponse
	georesponse = getCoordinates(input)
	if len(georesponse.Result) == 0 {
		log.Fatal("Sorry that place doesn't seem to exist!")
	}
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true", georesponse.Result[0].Latitude, georesponse.Result[0].Longitude)

	res, err := http.Get(url)

	if err != nil {
		log.Fatal("Oh no! Fatal error:", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Could not read response:", err)
	}

	var response WeatherResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal("Could not parse the json: ", err)
	}

	fmt.Printf("The temperature in %s is: %.1f°C \n", input, response.CurrentWeather.Temperature)
	fmt.Printf("The windspeed in %s is: %.1f km/h\n", input, response.CurrentWeather.Windspeed)

}
