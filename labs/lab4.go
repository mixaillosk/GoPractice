//Var 23 -> 3

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type GeocodingResponse struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
		Country   string  `json:"country"`
	} `json:"results"`
}

type WeatherResponse struct {
	Current struct {
		Temperature      float64 `json:"temperature_2m"`
		WindSpeed        float64 `json:"wind_speed_10m"`
		ApparentTemp     float64 `json:"apparent_temperature"`
		RelativeHumidity int     `json:"relative_humidity_2m"`
		Time             string  `json:"time"`
	} `json:"current"`
}

func getCoordinates(city string) (float64, float64, error) {
	encodedCity := url.QueryEscape(city)
	url := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=5", encodedCity)

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("ошибка при запросе координат: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	var data GeocodingResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, 0, fmt.Errorf("ошибка при парсинге JSON: %v", err)
	}

	if len(data.Results) == 0 {
		return 0, 0, fmt.Errorf("город не найден")
	}

	if len(data.Results) > 1 {
		fmt.Println("\nНайдено несколько городов с таким названием:")
		for i, result := range data.Results {
			fmt.Printf("%d. %s, %s\n", i+1, result.Name, result.Country)
		}

		var choice int
		for {
			fmt.Print("Введите номер нужного города: ")
			_, err := fmt.Scan(&choice)
			if err != nil {
				fmt.Println("Ошибка ввода. Пожалуйста, введите число.")
				continue
			}
			if choice < 1 || choice > len(data.Results) {
				fmt.Println("Некорректный номер. Попробуйте снова.")
				continue
			}
			return data.Results[choice-1].Latitude, data.Results[choice-1].Longitude, nil
		}
	}

	// Если только один вариант
	return data.Results[0].Latitude, data.Results[0].Longitude, nil
}

func getWeather(lat, lon float64) (WeatherResponse, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,wind_speed_10m,apparent_temperature,relative_humidity_2m", lat, lon)

	resp, err := http.Get(url)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("ошибка при запросе погоды: %v", err)
	}
	defer resp.Body.Close()

	var data WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return WeatherResponse{}, fmt.Errorf("ошибка при парсинге JSON: %v", err)
	}

	return data, nil
}

func main() {
	fmt.Print("Введите название города: ")
	var city string
	fmt.Scanln(&city)

	// Получаем координаты
	lat, lon, err := getCoordinates(city)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	// Получаем данные о погоде
	weather, err := getWeather(lat, lon)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}

	// Выводим информацию
	fmt.Printf("\nТекущая погода в %.0f°N %.0f°E:\n", lat, lon)
	fmt.Printf("Температура: %.1f°C\n", weather.Current.Temperature)
	fmt.Printf("Скорость ветра: %.1f м/с\n", weather.Current.WindSpeed)
	fmt.Printf("Ощущаемая температура: %.1f°C\n", weather.Current.ApparentTemp)
	fmt.Printf("Влажность воздуха: %d%%\n", weather.Current.RelativeHumidity)
	fmt.Printf("Время измерения: %s\n", weather.Current.Time)
}
