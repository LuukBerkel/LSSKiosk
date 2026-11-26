package controllers

import (
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type WeatherController struct {
	beego.Controller
}

type weatherData struct {
	Temp       int
	FeelsLike  int
	WindSpeed  int
	WindDir    string
	WindKmh    int
	RainChance int
	Condition  string
}

type forecastData struct {
	Hour string
	Icon string
	Temp int
	Rain int
}

func (c *WeatherController) Get() {
	now := time.Now()
	c.Data["LastUpdated"] = now.Format("15:04")
	c.Data["Weather"] = weatherData{
		Temp:       18,
		FeelsLike:  17,
		WindSpeed:  3,
		WindDir:    "SW",
		WindKmh:    16,
		RainChance: 20,
		Condition:  "Gedeeltelijk bewolkt",
	}
	c.Data["Forecast"] = []forecastData{
		{Hour: now.Truncate(time.Hour).Format("15:04"), Icon: "🌤️", Temp: 18, Rain: 10},
		{Hour: now.Truncate(time.Hour).Add(time.Hour).Format("15:04"), Icon: "☀️", Temp: 19, Rain: 5},
		{Hour: now.Truncate(time.Hour).Add(2 * time.Hour).Format("15:04"), Icon: "⛅", Temp: 18, Rain: 15},
		{Hour: now.Truncate(time.Hour).Add(3 * time.Hour).Format("15:04"), Icon: "🌦️", Temp: 17, Rain: 30},
	}
	c.TplName = "weather.tpl"
}
