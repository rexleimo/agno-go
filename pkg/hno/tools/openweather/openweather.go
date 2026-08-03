package openweather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// OpenWeatherToolkit provides access to weather data
// This is a simplified implementation that provides basic weather information
// Note: In production, you would need to provide an API key

type OpenWeatherToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new OpenWeather toolkit
func New() *OpenWeatherToolkit {
	t := &OpenWeatherToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("openweather"),
	}

	// Register current weather function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_current_weather",
		Description: "Get current weather information for a location",
		Parameters: map[string]toolkit.Parameter{
			"location": {
				Type:        "string",
				Description: "City name or coordinates (e.g., 'London' or '51.5074,-0.1278')",
				Required:    true,
			},
			"units": {
				Type:        "string",
				Description: "Temperature units: 'metric' (Celsius), 'imperial' (Fahrenheit), or 'standard' (Kelvin)",
				Required:    false,
				Default:     "metric",
			},
		},
		Handler: t.getCurrentWeather,
	})

	// Register forecast function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_weather_forecast",
		Description: "Get weather forecast for a location",
		Parameters: map[string]toolkit.Parameter{
			"location": {
				Type:        "string",
				Description: "City name or coordinates (e.g., 'London' or '51.5074,-0.1278')",
				Required:    true,
			},
			"days": {
				Type:        "integer",
				Description: "Number of forecast days (1-5, default: 3)",
				Required:    false,
				Default:     3,
			},
			"units": {
				Type:        "string",
				Description: "Temperature units: 'metric' (Celsius), 'imperial' (Fahrenheit), or 'standard' (Kelvin)",
				Required:    false,
				Default:     "metric",
			},
		},
		Handler: t.getWeatherForecast,
	})

	return t
}

// OpenWeather API response structures
type CurrentWeatherResponse struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Sys struct {
		Country string `json:"country"`
		Sunrise int64  `json:"sunrise"`
		Sunset  int64  `json:"sunset"`
	} `json:"sys"`
	Name string `json:"name"`
}

type ForecastResponse struct {
	List []struct {
		Dt   int64 `json:"dt"`
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			TempMin   float64 `json:"temp_min"`
			TempMax   float64 `json:"temp_max"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Clouds struct {
			All int `json:"all"`
		} `json:"clouds"`
		DtTxt string `json:"dt_txt"`
	} `json:"list"`
	City struct {
		Name    string `json:"name"`
		Country string `json:"country"`
		Coord   struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"coord"`
	} `json:"city"`
}

// getCurrentWeather gets current weather for a location
func (o *OpenWeatherToolkit) getCurrentWeather(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, ok := args["location"].(string)
	if !ok {
		return nil, fmt.Errorf("location must be a string")
	}

	units := "metric"
	if unitsArg, ok := args["units"].(string); ok {
		units = unitsArg
	}

	// Build OpenWeather API URL
	baseURL := "https://api.openweathermap.org/data/2.5/weather"
	params := url.Values{}
	params.Add("q", location)
	params.Add("units", units)
	params.Add("appid", "demo") // Using demo mode - in production, use real API key

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenWeather API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenWeather API returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var weatherResponse CurrentWeatherResponse
	if err := json.Unmarshal(body, &weatherResponse); err != nil {
		return nil, fmt.Errorf("failed to parse OpenWeather response: %w", err)
	}

	// Convert to structured result
	weatherDescription := ""
	if len(weatherResponse.Weather) > 0 {
		weatherDescription = weatherResponse.Weather[0].Description
	}

	result := map[string]interface{}{
		"location":          weatherResponse.Name,
		"country":           weatherResponse.Sys.Country,
		"coordinates":       fmt.Sprintf("%.4f, %.4f", weatherResponse.Coord.Lat, weatherResponse.Coord.Lon),
		"temperature":       weatherResponse.Main.Temp,
		"feels_like":        weatherResponse.Main.FeelsLike,
		"temperature_min":   weatherResponse.Main.TempMin,
		"temperature_max":   weatherResponse.Main.TempMax,
		"pressure":          weatherResponse.Main.Pressure,
		"humidity":          weatherResponse.Main.Humidity,
		"weather":           weatherDescription,
		"wind_speed":        weatherResponse.Wind.Speed,
		"wind_direction":    weatherResponse.Wind.Deg,
		"cloudiness":        weatherResponse.Clouds.All,
		"units":             units,
		"sunrise":           weatherResponse.Sys.Sunrise,
		"sunset":            weatherResponse.Sys.Sunset,
	}

	return map[string]interface{}{
		"current_weather": result,
	}, nil
}

// getWeatherForecast gets weather forecast for a location
func (o *OpenWeatherToolkit) getWeatherForecast(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, ok := args["location"].(string)
	if !ok {
		return nil, fmt.Errorf("location must be a string")
	}

	days := 3
	if daysArg, ok := args["days"].(float64); ok {
		days = int(daysArg)
	}

	units := "metric"
	if unitsArg, ok := args["units"].(string); ok {
		units = unitsArg
	}

	// Build OpenWeather API URL
	baseURL := "https://api.openweathermap.org/data/2.5/forecast"
	params := url.Values{}
	params.Add("q", location)
	params.Add("units", units)
	params.Add("cnt", fmt.Sprintf("%d", days*8)) // 8 forecasts per day
	params.Add("appid", "demo")                  // Using demo mode - in production, use real API key

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenWeather API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenWeather API returned status %d", resp.StatusCode)
	}

	// Parse JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var forecastResponse ForecastResponse
	if err := json.Unmarshal(body, &forecastResponse); err != nil {
		return nil, fmt.Errorf("failed to parse OpenWeather forecast response: %w", err)
	}

	// Convert to structured results
	forecasts := make([]map[string]interface{}, 0)
	for _, item := range forecastResponse.List {
		weatherDescription := ""
		if len(item.Weather) > 0 {
			weatherDescription = item.Weather[0].Description
		}

		forecast := map[string]interface{}{
			"datetime":        item.DtTxt,
			"timestamp":       item.Dt,
			"temperature":     item.Main.Temp,
			"feels_like":      item.Main.FeelsLike,
			"temperature_min": item.Main.TempMin,
			"temperature_max": item.Main.TempMax,
			"pressure":        item.Main.Pressure,
			"humidity":        item.Main.Humidity,
			"weather":         weatherDescription,
			"wind_speed":      item.Wind.Speed,
			"wind_direction":  item.Wind.Deg,
			"cloudiness":      item.Clouds.All,
		}

		forecasts = append(forecasts, forecast)
	}

	result := map[string]interface{}{
		"location":  forecastResponse.City.Name,
		"country":   forecastResponse.City.Country,
		"coordinates": fmt.Sprintf("%.4f, %.4f", forecastResponse.City.Coord.Lat, forecastResponse.City.Coord.Lon),
		"forecast_days": days,
		"units":        units,
		"forecasts":    forecasts,
	}

	return map[string]interface{}{
		"weather_forecast": result,
	}, nil
}