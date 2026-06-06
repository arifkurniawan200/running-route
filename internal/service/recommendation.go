package service

import (
	"fmt"
	"math"

	"github.com/arifkurniawan200/running-route/internal/model"
)

type Recommendation struct {
	route   *Route
	weather *Weather
}

func NewRecommendation(route *Route, weather *Weather) *Recommendation {
	return &Recommendation{route: route, weather: weather}
}

func (s *Recommendation) GetRoutes(req *model.RecommendRequest) (*model.RecommendResponse, error) {
	// Determine target distance based on pace and duration
	paceKmh := paceToSpeed(req.Pace)
	targetDistanceKm := paceKmh * float64(req.DurationMinutes) / 60.0

	// Get nearby points (parks, footways, tracks)
	points, err := s.route.NearbyPoints(req.Lat, req.Lng, 3000)
	if err != nil {
		return nil, err
	}

	if len(points) < 2 {
		// Fallback: generate a simple loop
		return s.fallbackRoute(req, targetDistanceKm)
	}

	var recommendations []model.RouteRecommendation

	// Generate loop route
	loop, _, err := s.route.GenerateLoop(req.Lat, req.Lng, targetDistanceKm, points)
	if err == nil && loop != nil {
		// Attach weather
		weather, wErr := s.weather.GetForecast(req.Lat, req.Lng, 6)
		if wErr == nil {
			loop.Weather = weather
		}
		recommendations = append(recommendations, *loop)
	}

	// If we don't have enough, add fallback routes
	if len(recommendations) < 2 {
		fallbacks, _ := s.fallbackRoute(req, targetDistanceKm)
		if fallbacks != nil {
			recommendations = append(recommendations, fallbacks.Routes...)
		}
	}

	// Limit to top 3
	if len(recommendations) > 3 {
		recommendations = recommendations[:3]
	}

	return &model.RecommendResponse{Routes: recommendations}, nil
}

func (s *Recommendation) fallbackRoute(req *model.RecommendRequest, targetKm float64) (*model.RecommendResponse, error) {
	// Generate synthetic routes by projecting points around the user
	directions := []struct {
		name  string
		bearing float64
		offset float64
	}{
		{"Loop GBK", 0, 0.3},
		{"Rute Senayan", 120, 0.25},
		{"Jalur Sudirman", 45, 0.2},
	}

	var routes []model.RouteRecommendation
	for i, d := range directions {
		if i >= 3 {
			break
		}

		// Project a point at bearing + distance
		endLat, endLng := projectPoint(req.Lat, req.Lng, d.bearing, targetKm*d.offset)

		distance := targetKm * (0.5 + float64(i)*0.2)
		if distance < 2 {
			distance = 2
		}

		rating := 4 - i
		if rating < 2 {
			rating = 2
		}

		difficulty := "easy"
		if distance > 8 {
			difficulty = "medium"
		}

		surface := "asphalt"
		if i == 1 {
			surface = "gravel"
		}

		weather, _ := s.weather.GetForecast(req.Lat, req.Lng, 6)

		route := model.RouteRecommendation{
			ID:                   fmt.Sprintf("route-fallback-%d", i+1),
			Name:                 d.name,
			Type:                 "loop",
			DistanceKm:           math.Round(distance*10) / 10,
			EstimatedDurationMin: int(distance / paceToSpeed(req.Pace) * 60),
			Difficulty:           difficulty,
			Surface:              surface,
			ElevationGainM:       15 + float64(i)*10,
			Rating:               rating,
			GeoJSON:              &model.GeoJSON{Type: "LineString", Coordinates: [][]float64{}},
			Waypoints: []model.Waypoint{
				{Lat: req.Lat, Lng: req.Lng, Name: "Start"},
				{Lat: endLat, Lng: endLng, Name: "Finish"},
			},
			NearbyPOIs: []model.POI{
				{Name: "Toilet Umum", Lat: req.Lat + 0.001, Lng: req.Lng + 0.001, Type: "toilet"},
			},
			Weather: weather,
		}

		routes = append(routes, route)
	}

	return &model.RecommendResponse{Routes: routes}, nil
}

// GetWeather returns weather forecast for a specific location and hours ahead
func (s *Recommendation) GetWeather(lat, lng float64, hours int) (*model.Weather, error) {
	return s.weather.GetForecast(lat, lng, hours)
}

func paceToSpeed(pace string) float64 {
	switch pace {
	case "fast":
		return 13.3 // ~4:30 min/km
	case "moderate":
		return 10.9 // ~5:30 min/km
	default:
		return 8.5 // ~7:00 min/km casual
	}
}

func projectPoint(lat, lng, bearing, distKm float64) (float64, float64) {
	const R = 6371.0
	bearingRad := bearing * math.Pi / 180.0
	angDist := distKm / R

	latRad := lat * math.Pi / 180.0
	lngRad := lng * math.Pi / 180.0

	newLat := math.Asin(math.Sin(latRad)*math.Cos(angDist) +
		math.Cos(latRad)*math.Sin(angDist)*math.Cos(bearingRad))
	newLng := lngRad + math.Atan2(
		math.Sin(bearingRad)*math.Sin(angDist)*math.Cos(latRad),
		math.Cos(angDist)-math.Sin(latRad)*math.Sin(newLat),
	)

	return newLat * 180.0 / math.Pi, newLng * 180.0 / math.Pi
}
