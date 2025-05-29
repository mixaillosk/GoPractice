//Var 23 -> 3

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Структура для ответа геокодера
type GeocodingResponse struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
		Country   string  `json:"country"`
	} `json:"results"`
}

// Структура для текущей погоды
type WeatherResponse struct {
	Current struct {
		Temperature      float64 `json:"temperature_2m"`
		WindSpeed        float64 `json:"wind_speed_10m"`
		ApparentTemp     float64 `json:"apparent_temperature"`
		RelativeHumidity int     `json:"relative_humidity_2m"`
		Time             string  `json:"time"`
	} `json:"current"`
}

// Функция получения координат по названию города
func getCoordinates(city string) (float64, float64, error) {
	encodedCity := url.QueryEscape(city)
	url := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=5 ", encodedCity)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "WeatherApp/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
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

	// Выведем список подходящих городов, если их больше одного
	if len(data.Results) > 1 {
		fmt.Println("\nНайдено несколько городов с таким названием:")
		for i, result := range data.Results {
			fmt.Printf("%d. %s, %s\n", i+1, result.Name, result.Country)
		}

		fmt.Print("Введите номер нужного города: ")
		var choice int
		_, err := fmt.Scan(&choice)
		if err != nil || choice < 1 || choice > len(data.Results) {
			return 0, 0, fmt.Errorf("некорректный выбор")
		}
		return data.Results[choice-1].Latitude, data.Results[choice-1].Longitude, nil
	}

	// Если только один вариант
	return data.Results[0].Latitude, data.Results[0].Longitude, nil
}

// Функция получения данных о погоде
// func getWeather(lat, lon float64) (WeatherResponse, error) {
// 	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m ,wind_speed_10m,apparent_temperature,relative_humidity_2m",
// 		lat, lon)

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return WeatherResponse{}, err
// 	}

// 	// Имитируем реальный браузер
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0 Safari/537.36")
// 	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
// 	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
// 	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
// 	req.Header.Set("Connection", "keep-alive")
// 	req.Header.Set("Upgrade-Insecure-Requests", "1")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return WeatherResponse{}, fmt.Errorf("ошибка при запросе погоды: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return WeatherResponse{}, fmt.Errorf("не удалось прочитать тело ответа: %v", err)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return WeatherResponse{}, fmt.Errorf("ошибка от API: код %d, тело: %s", resp.StatusCode, body)
// 	}

// 	var data WeatherResponse
// 	if err := json.Unmarshal(body, &data); err != nil {
// 		return WeatherResponse{}, fmt.Errorf("ошибка при парсинге JSON: %v", err)
// 	}

// 	return data, nil
// }

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
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Введите название города: ")
	city, _ := reader.ReadString('\n')
	city = strings.TrimSpace(city)

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
