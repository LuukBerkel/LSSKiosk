package controllers

import (
	"fmt"
	"math"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type EnergyController struct {
	beego.Controller
}

type energyData struct {
	GasUsageTodayM3         float64
	GasUsageWeekM3          float64
	ElectricityUsageWeekKwh float64
	GridFlowWatts           int
	WeekRangeLabel          string
	TemperatureLine         string
	SolarLine               string
	GasDays                 []gasDayPoint
	GridPoints              []energyChartPoint
	DaylightWindows         []daylightWindow
	SunriseMarkers          []float64
	SunsetMarkers           []float64
}

type gasDayPoint struct {
	DateLabel    string
	GasUsageM3   float64
	X            float64
	BarX         float64
	BarY         float64
	BarHeight    float64
	TemperatureY float64
}

type energyChartPoint struct {
	HourLabel    string
	X            float64
	ImportY      float64
	ImportHeight float64
	ExportHeight float64
	BarX         float64
}

type daylightWindow struct {
	X     float64
	Width float64
}

func (c *EnergyController) Get() {
	now := time.Now()
	currentHour := now.Truncate(time.Hour)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -6)

	const (
		chartLeft  = 72.0
		chartWidth = 816.0
		barWidth   = 14.0
	)

	weekHours := int(currentHour.Sub(weekStart).Hours())
	gasUsageToday := 0.0
	electricityUsageWeek := 0.0
	currentUsageKW := 0.0
	currentProductionKW := 0.0
	dailyGasUsage := make([]float64, 7)
	dailyTemperatureTotal := make([]float64, 7)
	dailySampleCount := make([]int, 7)
	dailyTemperature := []float64{10, 9, 10, 12, 11, 8, 7}
	dailyGasFactor := []float64{1.05, 1, 1.08, 1.02, 0.98, 0.85, 0.9}
	dailySolarFactor := []float64{0.75, 0.9, 0.8, 1, 0.95, 0.85, 0.7}

	for hourOffset := 0; hourOffset <= weekHours; hourOffset++ {
		sampleTime := weekStart.Add(time.Duration(hourOffset) * time.Hour)
		weekday := (int(sampleTime.Weekday()) + 6) % 7
		hourOfDay := float64(sampleTime.Hour())

		morningPeak := math.Exp(-math.Pow((hourOfDay-7)/2.2, 2))
		eveningPeak := math.Exp(-math.Pow((hourOfDay-19)/3, 2))
		gasRate := (0.055 + 0.19*morningPeak + 0.12*eveningPeak) * dailyGasFactor[weekday]
		electricityUsageWeek += electricityConsumptionForHour(hourOfDay, weekday)
		outdoorTemp := dailyTemperature[weekday] + 5.5*math.Sin(2*math.Pi*(hourOfDay-8)/24)

		dayIndex := 0
		for dayIndex < len(dailyGasUsage)-1 && !sampleTime.Before(weekStart.AddDate(0, 0, dayIndex+1)) {
			dayIndex++
		}
		dailyGasUsage[dayIndex] += gasRate
		dailyTemperatureTotal[dayIndex] += outdoorTemp
		dailySampleCount[dayIndex]++
		if sampleTime.Year() == today.Year() && sampleTime.YearDay() == today.YearDay() {
			gasUsageToday += gasRate
		}
		if sampleTime.Equal(currentHour) {
			currentUsageKW = electricityConsumptionForHour(hourOfDay, weekday)
			currentProductionKW = solarProductionForHour(hourOfDay) * dailySolarFactor[weekday]
		}
	}

	gasDays := make([]gasDayPoint, len(dailyGasUsage))
	gasUsageWeek := 0.0
	temperatureValues := make([]float64, len(dailyTemperatureTotal))
	dailyBarWidth := chartWidth / float64(len(dailyGasUsage)) * 0.58
	for day := range dailyGasUsage {
		gasUsageWeek += dailyGasUsage[day]
		averageTemperature := dailyTemperatureTotal[day] / float64(dailySampleCount[day])
		x := chartLeft + (float64(day)+0.5)*chartWidth/float64(len(dailyGasUsage))
		barHeight := dailyGasUsage[day] / 4 * 232
		gasDays[day] = gasDayPoint{
			DateLabel:    dutchDateLabel(weekStart.AddDate(0, 0, day)),
			GasUsageM3:   dailyGasUsage[day],
			X:            x,
			BarX:         x - dailyBarWidth/2,
			BarY:         280 - barHeight,
			BarHeight:    barHeight,
			TemperatureY: 280 - (averageTemperature+5)/30*232,
		}
		temperatureValues[day] = averageTemperature
	}
	temperatureLine := smoothPolyline(
		temperatureValues,
		gasDays[0].X,
		gasDays[len(gasDays)-1].X-gasDays[0].X,
		func(value float64) float64 {
			return 280 - (value+5)/30*232
		},
	)

	gridPoints := make([]energyChartPoint, 0, 25)
	solarValues := make([]float64, 0, 25)
	for hour := 0; hour <= 24; hour++ {
		hourValue := float64(hour)
		x := chartLeft + hourValue/24*chartWidth
		solarIntensity := 0.0
		if hour >= 6 && hour <= 18 {
			solarIntensity = 900 * math.Sin(math.Pi*(hourValue-6)/12)
		}
		buildingLoad := 1.65 + 0.55*math.Exp(-math.Pow((hourValue-19)/3, 2)) +
			0.35*math.Exp(-math.Pow((hourValue-7)/2, 2))
		netGridKW := buildingLoad - solarIntensity/300
		importKW := math.Max(netGridKW, 0)
		exportKW := math.Max(-netGridKW, 0)

		gridPoints = append(gridPoints, energyChartPoint{
			HourLabel:    hourValueLabel(hour),
			X:            x,
			ImportY:      166 - importKW*18,
			ImportHeight: importKW * 18,
			ExportHeight: exportKW * 18,
			BarX:         x - barWidth/2,
		})
		solarValues = append(solarValues, solarIntensity)
	}
	solarLine := smoothPolyline(solarValues, chartLeft, chartWidth, func(value float64) float64 {
		return 275 - value*0.215
	})

	sunriseX := chartLeft + 6.0/24*chartWidth
	sunsetX := chartLeft + 18.0/24*chartWidth
	gridFlowWatts := 300
	if currentProductionKW < currentUsageKW {
		gridFlowWatts = -300
	}
	c.Data["LastUpdated"] = now.Format("15:04")
	c.Data["Energy"] = energyData{
		GasUsageTodayM3:         gasUsageToday,
		GasUsageWeekM3:          gasUsageWeek,
		ElectricityUsageWeekKwh: electricityUsageWeek,
		GridFlowWatts:           gridFlowWatts,
		WeekRangeLabel: fmt.Sprintf(
			"Uurmetingen · %s–%s (tot %s)",
			dutchDateLabel(weekStart),
			dutchDateLabel(today),
			currentHour.Format("15:04"),
		),
		TemperatureLine: temperatureLine,
		SolarLine:       solarLine,
		GasDays:         gasDays,
		GridPoints:      gridPoints,
		DaylightWindows: []daylightWindow{{X: sunriseX, Width: sunsetX - sunriseX}},
		SunriseMarkers:  []float64{sunriseX},
		SunsetMarkers:   []float64{sunsetX},
	}
	c.TplName = "energy.tpl"
}

func dutchDateLabel(date time.Time) string {
	months := [...]string{"jan", "feb", "mrt", "apr", "mei", "jun", "jul", "aug", "sep", "okt", "nov", "dec"}
	return fmt.Sprintf("%02d %s", date.Day(), months[int(date.Month())-1])
}

func electricityConsumptionForHour(hour float64, weekday int) float64 {
	dailyLoadFactor := []float64{1.05, 1, 1.08, 1.02, 0.98, 0.85, 0.9}
	morningPeak := math.Exp(-math.Pow((hour-7)/2.5, 2))
	eveningPeak := math.Exp(-math.Pow((hour-19)/3, 2))
	return (0.65 + 0.65*morningPeak + 0.95*eveningPeak) * dailyLoadFactor[weekday]
}

func solarProductionForHour(hour float64) float64 {
	if hour < 6 || hour > 18 {
		return 0
	}
	return 3.8 * math.Sin(math.Pi*(hour-6)/12)
}

func hourValueLabel(hour int) string {
	if hour%3 != 0 && hour != 24 {
		return ""
	}
	return fmt.Sprintf("%02d", hour%24)
}

func smoothPolyline(values []float64, startX, width float64, transform func(float64) float64) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return fmt.Sprintf("%.1f,%.1f", startX, transform(values[0]))
	}

	const pointsPerInterval = 4
	totalPoints := (len(values)-1)*pointsPerInterval + 1
	points := make([]string, 0, totalPoints)
	for i := 0; i < len(values)-1; i++ {
		for step := 0; step < pointsPerInterval; step++ {
			t := float64(step) / pointsPerInterval
			smoothT := t * t * (3 - 2*t)
			value := values[i] + (values[i+1]-values[i])*smoothT
			pointIndex := i*pointsPerInterval + step
			x := startX + float64(pointIndex)/float64(totalPoints-1)*width
			points = append(points, fmt.Sprintf("%.1f,%.1f", x, transform(value)))
		}
	}
	points = append(points, fmt.Sprintf("%.1f,%.1f", startX+width, transform(values[len(values)-1])))
	return strings.Join(points, " ")
}
