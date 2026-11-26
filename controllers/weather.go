package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

const (
	weatherCacheTTL = 5 * time.Minute
)

var (
	weatherHTTPClient = &http.Client{Timeout: 8 * time.Second}
	weatherAPIURL     = fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.14f&longitude=%.14f"+
			"&current=temperature_2m,apparent_temperature,relative_humidity_2m,surface_pressure,precipitation,rain,snowfall,weather_code,wind_speed_10m,wind_direction_10m"+
			"&hourly=temperature_2m,relative_humidity_2m,surface_pressure,precipitation_probability,precipitation,rain,snowfall,weather_code,wind_speed_10m"+
			"&daily=weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max"+
			"&timezone=Europe%%2FAmsterdam&forecast_days=5",
		aircraftLatitude,
		aircraftLongitude,
	)

	weatherCacheMu    sync.Mutex
	weatherCacheEntry cachedWeather
)

type WeatherController struct {
	beego.Controller
}

type weatherData struct {
	Temp       int
	FeelsLike  int
	Humidity   int
	Pressure   int
	WindSpeed  int
	WindDir    string
	WindKmh    int
	RainChance int
	PrecipMm   float64
	Condition  string
	Icon       string
}

type forecastData struct {
	Hour string
	Icon string
	Temp int
	Rain int
}

type dailyForecastData struct {
	Day     string
	Icon    string
	TempMax int
	TempMin int
	Rain    int
}

type cachedWeather struct {
	Weather   weatherData
	Hourly    []forecastData
	Daily     []dailyForecastData
	ExpiresAt time.Time
}

type openMeteoResponse struct {
	Current struct {
		Time          string  `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		ApparentTemp  float64 `json:"apparent_temperature"`
		Humidity      float64 `json:"relative_humidity_2m"`
		Pressure      float64 `json:"surface_pressure"`
		Precipitation float64 `json:"precipitation"`
		WeatherCode   int     `json:"weather_code"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		Time          []string  `json:"time"`
		Temperature   []float64 `json:"temperature_2m"`
		RainChance    []int     `json:"precipitation_probability"`
		WeatherCode   []int     `json:"weather_code"`
	} `json:"hourly"`
	Daily struct {
		Time        []string  `json:"time"`
		WeatherCode []int     `json:"weather_code"`
		TempMax     []float64 `json:"temperature_2m_max"`
		TempMin     []float64 `json:"temperature_2m_min"`
		RainChance  []int     `json:"precipitation_probability_max"`
	} `json:"daily"`
}

func (c *WeatherController) Get() {
	loc := amsterdamLocation()
	now := time.Now().In(loc)

	weather, hourly, daily, err := getWeatherForecast(c.Ctx.Request.Context(), weatherHTTPClient, weatherAPIURL, now)
	if err != nil {
		logs.Warn("Unable to fetch Open-Meteo weather: %v", err)
		weather, hourly, daily = fallbackWeather(now)
	}

	c.Data["LastUpdated"] = now.Format("15:04")
	c.Data["Weather"] = weather
	c.Data["Forecast"] = hourly
	c.Data["DailyForecast"] = daily
	c.TplName = "weather.tpl"
}

func amsterdamLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		return time.FixedZone("CET", 3600)
	}
	return loc
}

func getWeatherForecast(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	now time.Time,
) (weatherData, []forecastData, []dailyForecastData, error) {
	weatherCacheMu.Lock()
	if time.Now().Before(weatherCacheEntry.ExpiresAt) && len(weatherCacheEntry.Hourly) > 0 {
		cached := weatherCacheEntry
		weatherCacheMu.Unlock()
		return cached.Weather, cached.Hourly, cached.Daily, nil
	}
	weatherCacheMu.Unlock()

	weather, hourly, daily, err := fetchOpenMeteoForecast(ctx, client, endpoint, now)
	if err != nil {
		return weatherData{}, nil, nil, err
	}

	weatherCacheMu.Lock()
	weatherCacheEntry = cachedWeather{
		Weather:   weather,
		Hourly:    hourly,
		Daily:     daily,
		ExpiresAt: time.Now().Add(weatherCacheTTL),
	}
	weatherCacheMu.Unlock()

	return weather, hourly, daily, nil
}

func fetchOpenMeteoForecast(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	now time.Time,
) (weatherData, []forecastData, []dailyForecastData, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return weatherData{}, nil, nil, fmt.Errorf("create Open-Meteo request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return weatherData{}, nil, nil, fmt.Errorf("request Open-Meteo forecast: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return weatherData{}, nil, nil, fmt.Errorf("Open-Meteo returned HTTP %d", response.StatusCode)
	}

	var data openMeteoResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&data); err != nil {
		return weatherData{}, nil, nil, fmt.Errorf("decode Open-Meteo response: %w", err)
	}

	loc := now.Location()
	currentHour := now.Truncate(time.Hour)
	hourly := make([]forecastData, 0, 4)
	currentHourRainChance := 0
	foundCurrentHour := false

	for i, rawTime := range data.Hourly.Time {
		parsedTime, parseErr := time.ParseInLocation("2006-01-02T15:04", rawTime, loc)
		if parseErr != nil {
			continue
		}
		if parsedTime.Before(currentHour) {
			continue
		}
		if !foundCurrentHour {
			foundCurrentHour = true
			if i < len(data.Hourly.RainChance) {
				currentHourRainChance = data.Hourly.RainChance[i]
			}
		}
		if len(hourly) >= 4 {
			break
		}
		temp := 0
		if i < len(data.Hourly.Temperature) {
			temp = int(math.Round(data.Hourly.Temperature[i]))
		}
		rain := 0
		if i < len(data.Hourly.RainChance) {
			rain = data.Hourly.RainChance[i]
		}
		code := 0
		if i < len(data.Hourly.WeatherCode) {
			code = data.Hourly.WeatherCode[i]
		}
		_, icon := wmoWeatherDescription(code)
		hourly = append(hourly, forecastData{
			Hour: parsedTime.Format("15:04"),
			Icon: icon,
			Temp: temp,
			Rain: rain,
		})
	}

	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
	daily := make([]dailyForecastData, 0, 4)
	for i, rawDate := range data.Daily.Time {
		parsedDate, parseErr := time.ParseInLocation("2006-01-02", rawDate, loc)
		if parseErr != nil {
			continue
		}
		if parsedDate.Before(tomorrow) {
			continue
		}
		if len(daily) >= 4 {
			break
		}
		code := 0
		if i < len(data.Daily.WeatherCode) {
			code = data.Daily.WeatherCode[i]
		}
		tempMax := 0
		if i < len(data.Daily.TempMax) {
			tempMax = int(math.Round(data.Daily.TempMax[i]))
		}
		tempMin := 0
		if i < len(data.Daily.TempMin) {
			tempMin = int(math.Round(data.Daily.TempMin[i]))
		}
		rain := 0
		if i < len(data.Daily.RainChance) {
			rain = data.Daily.RainChance[i]
		}
		_, icon := wmoWeatherDescription(code)
		label := dutchWeekday(parsedDate.Weekday())
		if len(daily) == 0 {
			label = "Morgen"
		}
		daily = append(daily, dailyForecastData{
			Day:     label,
			Icon:    icon,
			TempMax: tempMax,
			TempMin: tempMin,
			Rain:    rain,
		})
	}

	condition, icon := wmoWeatherDescription(data.Current.WeatherCode)
	windKmh := int(math.Round(data.Current.WindSpeed))
	weather := weatherData{
		Temp:       int(math.Round(data.Current.Temperature)),
		FeelsLike:  int(math.Round(data.Current.ApparentTemp)),
		Humidity:   int(math.Round(data.Current.Humidity)),
		Pressure:   int(math.Round(data.Current.Pressure)),
		WindSpeed:  kmhToBeaufort(data.Current.WindSpeed),
		WindDir:    degreesToCompassNL(data.Current.WindDirection),
		WindKmh:    windKmh,
		RainChance: currentHourRainChance,
		PrecipMm:   math.Round(data.Current.Precipitation*10) / 10,
		Condition:  condition,
		Icon:       icon,
	}

	return weather, hourly, daily, nil
}

func fallbackWeather(now time.Time) (weatherData, []forecastData, []dailyForecastData) {
	baseHour := now.Truncate(time.Hour)
	weather := weatherData{
		Temp:       18,
		FeelsLike:  17,
		Humidity:   70,
		Pressure:   1015,
		WindSpeed:  3,
		WindDir:    "ZW",
		WindKmh:    16,
		RainChance: 20,
		Condition:  "Gedeeltelijk bewolkt",
		Icon:       "⛅",
	}
	hourly := []forecastData{
		{Hour: baseHour.Format("15:04"), Icon: "🌤️", Temp: 18, Rain: 10},
		{Hour: baseHour.Add(time.Hour).Format("15:04"), Icon: "☀️", Temp: 19, Rain: 5},
		{Hour: baseHour.Add(2 * time.Hour).Format("15:04"), Icon: "⛅", Temp: 18, Rain: 15},
		{Hour: baseHour.Add(3 * time.Hour).Format("15:04"), Icon: "🌦️", Temp: 17, Rain: 30},
	}
	daily := make([]dailyForecastData, 0, 4)
	for offset := 1; offset <= 4; offset++ {
		day := now.AddDate(0, 0, offset)
		label := dutchWeekday(day.Weekday())
		if offset == 1 {
			label = "Morgen"
		}
		daily = append(daily, dailyForecastData{
			Day:     label,
			Icon:    "⛅",
			TempMax: 18,
			TempMin: 11,
			Rain:    20,
		})
	}
	return weather, hourly, daily
}

func wmoWeatherDescription(code int) (string, string) {
	switch code {
	case 0:
		return "Onbewolkt", "☀️"
	case 1:
		return "Licht bewolkt", "🌤️"
	case 2:
		return "Gedeeltelijk bewolkt", "⛅"
	case 3:
		return "Bewolkt", "☁️"
	case 45, 48:
		return "Mist", "🌫️"
	case 51, 53, 55, 56, 57:
		return "Motregen", "🌦️"
	case 61, 63, 65, 66, 67:
		return "Regen", "🌧️"
	case 71, 73, 75, 77:
		return "Sneeuw", "❄️"
	case 80, 81, 82:
		return "Buien", "🌦️"
	case 85, 86:
		return "Sneeuwbuien", "🌨️"
	case 95, 96, 99:
		return "Onweer", "⛈️"
	default:
		return "Wisselend bewolkt", "🌤️"
	}
}

func kmhToBeaufort(kmh float64) int {
	thresholds := []float64{1, 5.5, 11.5, 19.5, 28.5, 38.5, 49.5, 61.5, 74.5, 88.5, 102.5, 117.5}
	for bft, limit := range thresholds {
		if kmh < limit {
			return bft
		}
	}
	return 12
}

func degreesToCompassNL(deg float64) string {
	directions := []string{"N", "NO", "O", "ZO", "Z", "ZW", "W", "NW"}
	normalized := math.Mod(deg, 360)
	if normalized < 0 {
		normalized += 360
	}
	index := int(math.Round(normalized/45)) % 8
	return directions[index]
}

func dutchWeekday(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Maandag"
	case time.Tuesday:
		return "Dinsdag"
	case time.Wednesday:
		return "Woensdag"
	case time.Thursday:
		return "Donderdag"
	case time.Friday:
		return "Vrijdag"
	case time.Saturday:
		return "Zaterdag"
	case time.Sunday:
		return "Zondag"
	default:
		return ""
	}
}
