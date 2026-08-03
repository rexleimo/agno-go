package openweather

import (
	"context"
	"testing"
)

func TestOpenWeatherToolkit_GetCurrentWeather(t *testing.T) {
	toolkit := New()

	// Test getting current weather
	result, err := toolkit.getCurrentWeather(context.Background(), map[string]interface{}{
		"location": "London",
		"units":    "metric",
	})

	// This might fail if API is down or demo mode doesn't work
	if err != nil {
		t.Logf("Expected API error in demo mode: %v", err)
		return
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	currentWeather, ok := resultMap["current_weather"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected current_weather object, got: %T", resultMap["current_weather"])
	}

	// Check required fields
	if _, ok := currentWeather["location"].(string); !ok {
		t.Error("Expected location in current weather")
	}
	if _, ok := currentWeather["temperature"].(float64); !ok {
		t.Error("Expected temperature in current weather")
	}
	if _, ok := currentWeather["weather"].(string); !ok {
		t.Error("Expected weather description in current weather")
	}
}

func TestOpenWeatherToolkit_GetWeatherForecast(t *testing.T) {
	toolkit := New()

	// Test getting weather forecast
	result, err := toolkit.getWeatherForecast(context.Background(), map[string]interface{}{
		"location": "London",
		"days":     2,
		"units":    "metric",
	})

	// This might fail if API is down or demo mode doesn't work
	if err != nil {
		t.Logf("Expected API error in demo mode: %v", err)
		return
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got: %T", result)
	}

	weatherForecast, ok := resultMap["weather_forecast"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected weather_forecast object, got: %T", resultMap["weather_forecast"])
	}

	// Check required fields
	if _, ok := weatherForecast["location"].(string); !ok {
		t.Error("Expected location in weather forecast")
	}
	if _, ok := weatherForecast["forecast_days"].(int); !ok {
		t.Error("Expected forecast_days in weather forecast")
	}
	if _, ok := weatherForecast["forecasts"].([]map[string]interface{}); !ok {
		t.Error("Expected forecasts array in weather forecast")
	}

	// Check forecast days
	forecastDays, _ := weatherForecast["forecast_days"].(int)
	if forecastDays != 2 {
		t.Errorf("Expected forecast_days 2, got %d", forecastDays)
	}
}

func TestOpenWeatherToolkit_New(t *testing.T) {
	toolkit := New()

	if toolkit == nil {
		t.Fatal("Expected toolkit to be created")
	}

	// Check that functions are registered
	functions := toolkit.Functions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	expectedFunctions := []string{"get_current_weather", "get_weather_forecast"}
	for _, expected := range expectedFunctions {
		found := false
		for _, function := range functions {
			if function.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected function '%s' not found", expected)
		}
	}
}