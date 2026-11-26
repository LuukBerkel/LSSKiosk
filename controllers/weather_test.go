package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWeatherAPIURLUsesAircraftCoordinates(t *testing.T) {
	wantLat := fmt.Sprintf("latitude=%.14f", aircraftLatitude)
	wantLon := fmt.Sprintf("longitude=%.14f", aircraftLongitude)
	if !strings.Contains(weatherAPIURL, wantLat) || !strings.Contains(weatherAPIURL, wantLon) {
		t.Fatalf("weatherAPIURL = %q, want aircraft coordinates (%s, %s)", weatherAPIURL, wantLat, wantLon)
	}
}

func TestFetchOpenMeteoForecastParsesCurrentFourHoursAndFourDays(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"current": {
				"time": "2026-11-26T16:30",
				"temperature_2m": 11.4,
				"apparent_temperature": 9.2,
				"relative_humidity_2m": 82,
				"surface_pressure": 1013.6,
				"precipitation": 0.4,
				"weather_code": 61,
				"wind_speed_10m": 22.0,
				"wind_direction_10m": 225.0
			},
			"hourly": {
				"time": [
					"2026-11-26T15:00",
					"2026-11-26T16:00",
					"2026-11-26T17:00",
					"2026-11-26T18:00",
					"2026-11-26T19:00",
					"2026-11-26T20:00"
				],
				"temperature_2m": [12.0, 11.4, 10.6, 9.8, 9.1, 8.5],
				"precipitation_probability": [10, 65, 50, 30, 15, 5],
				"weather_code": [3, 61, 80, 2, 1, 0]
			},
			"daily": {
				"time": [
					"2026-11-26",
					"2026-11-27",
					"2026-11-28",
					"2026-11-29",
					"2026-11-30"
				],
				"weather_code": [61, 2, 0, 3, 63],
				"temperature_2m_max": [12.0, 13.4, 14.1, 10.2, 9.6],
				"temperature_2m_min": [7.0, 5.6, 6.2, 4.8, 3.9],
				"precipitation_probability_max": [70, 20, 10, 40, 80]
			}
		}`)
	}))
	defer server.Close()

	now := time.Date(2026, 11, 26, 16, 35, 0, 0, amsterdamLocation())
	weather, hourly, daily, err := fetchOpenMeteoForecast(context.Background(), server.Client(), server.URL, now)
	if err != nil {
		t.Fatalf("fetchOpenMeteoForecast returned error: %v", err)
	}
	if weather.Temp != 11 || weather.FeelsLike != 9 || weather.Humidity != 82 || weather.Pressure != 1014 {
		t.Errorf("current weather = %+v, want temp 11, feelsLike 9, humidity 82, pressure 1014", weather)
	}
	if weather.WindSpeed != 4 || weather.WindDir != "ZW" || weather.WindKmh != 22 || weather.RainChance != 65 || weather.Condition != "Regen" {
		t.Errorf("wind/rain/condition = %+v, want 4 Bft ZW 22 km/u, 65%%, Regen", weather)
	}
	if len(hourly) != 4 || hourly[0].Hour != "16:00" || hourly[3].Hour != "19:00" {
		t.Fatalf("hourly forecast = %+v, want 4 hours starting at 16:00", hourly)
	}
	if len(daily) != 4 || daily[0].Day != "Morgen" || daily[0].TempMax != 13 || daily[0].TempMin != 6 || daily[3].Day != "Maandag" {
		t.Fatalf("daily forecast = %+v, want 4 days starting at Morgen through Maandag", daily)
	}
}
