package model

import "encoding/json"

// --- Requests ---

type RecommendRequest struct {
	Lat             float64  `json:"lat"`
	Lng             float64  `json:"lng"`
	DurationMinutes int      `json:"duration_minutes"` // 30, 45, 60, 90, 120
	Pace            string   `json:"pace"`             // casual, moderate, fast
	RouteTypes      []string `json:"route_types"`      // loop, out_and_back, point_to_point
}

type WeatherRequest struct {
	RouteID   string `json:"route_id"`
	StartTime string `json:"start_time"` // ISO 8601
	Lat       float64
	Lng       float64
}

// --- Responses ---

type RecommendResponse struct {
	Routes []RouteRecommendation `json:"routes"`
}

type RouteRecommendation struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Type                 string      `json:"type"` // loop, out_and_back, point_to_point
	DistanceKm           float64     `json:"distance_km"`
	EstimatedDurationMin int         `json:"estimated_duration_min"`
	Difficulty           string      `json:"difficulty"` // easy, medium, hard
	Surface              string      `json:"surface"`    // asphalt, gravel, grass, mixed
	ElevationGainM       float64     `json:"elevation_gain_m"`
	Rating               int         `json:"rating"` // 1-5
	GeoJSON              *GeoJSON    `json:"geojson"`
	Waypoints            []Waypoint  `json:"waypoints"`
	NearbyPOIs           []POI       `json:"nearby_pois"`
	Weather              *Weather    `json:"weather"`
}

type Waypoint struct {
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Name string  `json:"name"`
}

type POI struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
	Type string  `json:"type"` // toilet, water, parking, shelter
}

type Weather struct {
	StartHour      string         `json:"start_hour"`
	EndHour        string         `json:"end_hour"`
	Hourly         []WeatherHour  `json:"hourly"`
	Summary        string         `json:"summary"`
	Recommendation string         `json:"recommendation"` // recommended, caution, not_recommended
	Alerts         []string       `json:"alerts"`
}

type WeatherHour struct {
	Time                     string  `json:"time"`
	TemperatureC             float64 `json:"temperature_c"`
	PrecipitationProbability float64 `json:"precipitation_probability"`
	WindSpeedKmh             float64 `json:"wind_speed_kmh"`
	UvIndex                  float64 `json:"uv_index"`
	Condition                string  `json:"condition"`
}

type GeoJSON struct {
	Type        string           `json:"type"`
	Coordinates [][]float64      `json:"coordinates"`
	Properties  *GeoJSONProperties `json:"properties,omitempty"`
}

type GeoJSONProperties struct {
	Name      string   `json:"name,omitempty"`
	Surface   string   `json:"surface,omitempty"`
	Surfaces  []string `json:"surfaces,omitempty"`
}

// --- External API models ---

type OverpassResponse struct {
	Elements []OverpassElement `json:"elements"`
}

type OverpassElement struct {
	Type     string             `json:"type"`
	ID       int64              `json:"id"`
	Lat      float64            `json:"lat,omitempty"`
	Lon      float64            `json:"lon,omitempty"`
	Center   *OverpassCenter    `json:"center,omitempty"`
	Tags     map[string]string  `json:"tags,omitempty"`
	Geometry []OverpassGeometry `json:"geometry,omitempty"`
	Nodes    []int64            `json:"nodes,omitempty"`
}

type OverpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type OverpassGeometry struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type OSRMResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance float64          `json:"distance"` // meters
		Duration float64          `json:"duration"` // seconds
		Geometry *json.RawMessage `json:"geometry"`
	} `json:"routes"`
}

type OpenMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Hourly    struct {
		Time                     []string  `json:"time"`
		Temperature2M            []float64 `json:"temperature_2m"`
		PrecipitationProbability []float64 `json:"precipitation_probability"`
		WindSpeed10M             []float64 `json:"wind_speed_10m"`
		UvIndex                  []float64 `json:"uv_index"`
	} `json:"hourly"`
}

type NominatimResponse []struct {
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}
