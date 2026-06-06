package service

import (
	"fmt"
	"time"

	"github.com/arifkurniawan200/running-route/internal/cache"
	"github.com/arifkurniawan200/running-route/internal/client"
	"github.com/arifkurniawan200/running-route/internal/model"
)

type Weather struct {
	client  *client.OpenMeteo
	cache   *cache.Memory
}

func NewWeather(client *client.OpenMeteo, cache *cache.Memory) *Weather {
	return &Weather{client: client, cache: cache}
}

func (s *Weather) GetForecast(lat, lng float64, hoursAhead int) (*model.Weather, error) {
	cacheKey := fmt.Sprintf("weather:%.4f:%.4f:%d", lat, lng, hoursAhead)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*model.Weather), nil
	}

	now := time.Now().UTC()
	startHour := now.Format("2006-01-02T15:00")
	endHour := now.Add(time.Duration(hoursAhead) * time.Hour).Format("2006-01-02T15:00")

	resp, err := s.client.GetForecast(lat, lng, startHour, endHour)
	if err != nil {
		return nil, fmt.Errorf("fetch weather failed: %w", err)
	}

	if len(resp.Hourly.Time) == 0 {
		return nil, fmt.Errorf("no weather data available")
	}

	var hourly []model.WeatherHour
	for i := range resp.Hourly.Time {
		condition := classifyCondition(
			resp.Hourly.PrecipitationProbability[i],
			resp.Hourly.Temperature2M[i],
			resp.Hourly.UvIndex[i],
		)
		hourly = append(hourly, model.WeatherHour{
			Time:                     resp.Hourly.Time[i],
			TemperatureC:             resp.Hourly.Temperature2M[i],
			PrecipitationProbability: resp.Hourly.PrecipitationProbability[i],
			WindSpeedKmh:             resp.Hourly.WindSpeed10M[i],
			UvIndex:                  resp.Hourly.UvIndex[i],
			Condition:                condition,
		})
	}

	// Generate summary
	maxTemp := resp.Hourly.Temperature2M[0]
	minTemp := resp.Hourly.Temperature2M[0]
	maxPrecip := resp.Hourly.PrecipitationProbability[0]
	for _, t := range resp.Hourly.Temperature2M {
		if t > maxTemp {
			maxTemp = t
		}
		if t < minTemp {
			minTemp = t
		}
	}
	for _, p := range resp.Hourly.PrecipitationProbability {
		if p > maxPrecip {
			maxPrecip = p
		}
	}

	recommendation := "recommended"
	var alerts []string
	if maxPrecip > 70 {
		recommendation = "not_recommended"
		alerts = append(alerts, "Potensi hujan tinggi")
	} else if maxPrecip > 40 {
		recommendation = "caution"
		alerts = append(alerts, "Kemungkinan hujan ringan")
	}
	if maxTemp > 32 {
		if recommendation == "recommended" {
			recommendation = "caution"
		}
		alerts = append(alerts, "Suhu panas >32°C")
	}

	summary := fmt.Sprintf("Suhu %.0f-%.0f°C", minTemp, maxTemp)
	if recommendation == "recommended" {
		summary += ", cocok lari! 🏃"
	} else if recommendation == "caution" {
		summary += ", perhatikan kondisi cuaca ⚠️"
	} else {
		summary += ", kurang ideal untuk lari 🌧️"
	}

	weather := &model.Weather{
		Hourly:         hourly,
		Summary:        summary,
		Recommendation: recommendation,
		Alerts:         alerts,
	}

	s.cache.Set(cacheKey, weather, 30*time.Minute)
	return weather, nil
}

func classifyCondition(precipProb, tempC, uvIndex float64) string {
	if precipProb > 70 {
		return "hujan"
	} else if precipProb > 40 {
		return "berawan"
	} else if uvIndex > 5 {
		return "cerah"
	}
	return "cerah"
}
