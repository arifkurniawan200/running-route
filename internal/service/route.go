package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/arifkurniawan200/running-route/internal/client"
	"github.com/arifkurniawan200/running-route/internal/model"
)

type Route struct {
	overpass  *client.Overpass
	osrm      *client.OSRM
	elevation *client.ElevationClient
}

func NewRoute(overpass *client.Overpass, osrm *client.OSRM, elevation *client.ElevationClient) *Route {
	return &Route{overpass: overpass, osrm: osrm, elevation: elevation}
}

// NearbyPoints finds parks, footways, and tracks near a location.
// Returns candidate start/end points for route generation.
func (s *Route) NearbyPoints(lat, lng float64, radiusM float64) ([]*Point, error) {
	osmData, err := s.overpass.QueryParksAndPaths(lat, lng, radiusM)
	if err != nil {
		return nil, fmt.Errorf("query osm failed: %w", err)
	}

	var points []*Point
	seen := make(map[int64]bool)

	for _, el := range osmData.Elements {
		if seen[el.ID] {
			continue
		}
		seen[el.ID] = true

		var elementLat, elementLng float64
		if el.Type == "node" {
			elementLat = el.Lat
			elementLng = el.Lon
		} else if el.Center != nil {
			elementLat = el.Center.Lat
			elementLng = el.Center.Lon
		} else if len(el.Geometry) > 0 {
			// Average of geometry points
			var sumLat, sumLng float64
			for _, g := range el.Geometry {
				sumLat += g.Lat
				sumLng += g.Lon
			}
			elementLat = sumLat / float64(len(el.Geometry))
			elementLng = sumLng / float64(len(el.Geometry))
		} else {
			continue
		}

		surface := el.Tags["surface"]
		if surface == "" {
			surface = "mixed"
		}

		name := el.Tags["name"]
		if name == "" {
			name = el.Tags["leisure"]
			if name == "" {
				name = el.Tags["highway"]
			}
			if name == "" {
				name = "Jalur Lari"
			}
		}

		points = append(points, &Point{
			Lat:     elementLat,
			Lng:     elementLng,
			Name:    name,
			Surface: surface,
			Type:    el.Tags["leisure"],
			OSMID:   el.ID,
		})
	}

	return points, nil
}

// CalculateRoute uses OSRM to get a walking route between two points.
func (s *Route) CalculateRoute(from, to *Point) (*model.OSRMResponse, error) {
	return s.osrm.RouteFoot(from.Lat, from.Lng, to.Lat, to.Lng)
}

// Point represents a candidate location for route generation.
type Point struct {
	Lat     float64
	Lng     float64
	Name    string
	Surface string
	Type    string
	OSMID   int64
}

// GenerateLoop creates a loop route by finding points at approximately
// the right distance from the user's location.
func (s *Route) GenerateLoop(lat, lng float64, targetDistanceKm float64, points []*Point) (*model.RouteRecommendation, *model.OSRMResponse, error) {
	if len(points) < 2 {
		return nil, nil, fmt.Errorf("not enough nearby points")
	}

	// Pick the best park or path
	best := points[0]
	bestDist := haversine(lat, lng, best.Lat, best.Lng)
	for _, p := range points {
		dist := haversine(lat, lng, p.Lat, p.Lng)
		if dist > 0.5 && dist < targetDistanceKm*0.4 && dist > bestDist {
			best = p
			bestDist = dist
		}
	}

	// Find a second point roughly opposite direction for a loop
	var second *Point
	var secondDist float64
	for _, p := range points {
		if p.OSMID == best.OSMID {
			continue
		}
		dist := haversine(best.Lat, best.Lng, p.Lat, p.Lng)
		if dist > targetDistanceKm*0.15 && dist < targetDistanceKm*0.5 {
			if second == nil || dist > secondDist {
				second = p
				secondDist = dist
			}
		}
	}

	if second == nil {
		second = points[len(points)/2]
	}

	// Calculate route via OSRM
	osrmResp, err := s.osrm.RouteFoot(best.Lat, best.Lng, second.Lat, second.Lng)
	if err != nil {
		return nil, nil, fmt.Errorf("osrm routing failed: %w", err)
	}

	if len(osrmResp.Routes) == 0 {
		return nil, nil, fmt.Errorf("no route found")
	}

	route := osrmResp.Routes[0]
	distanceKm := route.Distance / 1000.0

	// Build GeoJSON for the route
	geojson := parseOSRMGeometry(route.Geometry)

	difficulty := "easy"
	if distanceKm > 10 {
		difficulty = "medium"
	}
	if distanceKm > 15 {
		difficulty = "hard"
	}

	rating := 3
	if best.Surface == "asphalt" {
		rating++
	}
	if distanceKm > 3 {
		rating++
	}

	recommendation := &model.RouteRecommendation{
		ID:                   fmt.Sprintf("loop-%d", best.OSMID),
		Name:                 fmt.Sprintf("Lari di %s", best.Name),
		Type:                 "loop",
		DistanceKm:           math.Round(distanceKm*10) / 10,
		Difficulty:           difficulty,
		Surface:              best.Surface,
		Rating:               rating,
		GeoJSON:              geojson,
		EstimatedDurationMin: int(distanceKm * 60.0 / 8.5),
		Waypoints: []model.Waypoint{
			{Lat: best.Lat, Lng: best.Lng, Name: fmt.Sprintf("Start - %s", best.Name)},
			{Lat: second.Lat, Lng: second.Lng, Name: fmt.Sprintf("Belok - %s", second.Name)},
		},
		Steps:            parseOSRMSteps(osrmResp),
		ElevationProfile: sampleElevationProfile(osrmResp),
	}

	return recommendation, osrmResp, nil
}

// haversine calculates distance in km between two points
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLng := (lng2 - lng1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func parseOSRMGeometry(raw *json.RawMessage) *model.GeoJSON {
	// In production, parse the raw geometry from OSRM
	// For now return a simple GeoJSON
	return &model.GeoJSON{
		Type:        "LineString",
		Coordinates: [][]float64{},
	}
}

// Pool for sync.Pool to reduce allocations
var pointPool = sync.Pool{
	New: func() interface{} {
		return &Point{}
	},
}

// modifierToDirection converts OSRM maneuver modifier to Indonesian direction
func modifierToDirection(modifier string) string {
	switch modifier {
	case "left":
		return "↰ Kiri"
	case "right":
		return "↱ Kanan"
	case "sharp left":
		return "↰ Kiri Tajam"
	case "sharp right":
		return "↱ Kanan Tajam"
	case "slight left":
		return "↰ Serong Kiri"
	case "slight right":
		return "↱ Serong Kanan"
	case "straight":
		return "⬆ Lurus"
	case "uturn":
		return "↻ Putar Balik"
	default:
		return "⬆ Lurus"
	}
}

// maneuverTypeToText converts OSRM maneuver type to Indonesian
func maneuverTypeToText(mType string) string {
	switch mType {
	case "turn":
		return "Belok"
	case "new name":
		return "Lanjut"
	case "depart":
		return "Mulai"
	case "arrive":
		return "Sampai"
	case "merge":
		return "Gabung"
	case "on ramp":
		return "Masuk"
	case "off ramp":
		return "Keluar"
	case "fork":
		return "Percabangan"
	case "end of road":
		return "Ujung Jalan"
	case "continue":
		return "Lanjut"
	case "roundabout":
		return "Bundaran"
	case "rotary":
		return "Bundaran"
	case "roundabout turn":
		return "Bundaran"
	case "notification":
		return ""
	default:
		return mType
	}
}

// parseOSRMSteps converts OSRM step data to RouteStep models
func parseOSRMSteps(osrmResp *model.OSRMResponse) []model.RouteStep {
	if len(osrmResp.Routes) == 0 || len(osrmResp.Routes[0].Legs) == 0 {
		return nil
	}

	var steps []model.RouteStep
	for _, leg := range osrmResp.Routes[0].Legs {
		for _, s := range leg.Steps {
			dir := modifierToDirection(s.Maneuver.Modifier)
			inst := fmt.Sprintf("%s %s", maneuverTypeToText(s.Maneuver.Type), dir)
			if s.Name != "" {
				inst = fmt.Sprintf("%s ke %s", inst, s.Name)
			}

			loc := s.Maneuver.Location
			lat, lng := 0.0, 0.0
			if len(loc) >= 2 {
				lng = loc[0]
				lat = loc[1]
			}

			steps = append(steps, model.RouteStep{
				Instruction: inst,
				DistanceM:   math.Round(s.Distance),
				DurationS:   math.Round(s.Duration),
				Direction:   dir,
				StreetName:  s.Name,
				Lat:         lat,
				Lng:         lng,
			})
		}
	}
	return steps
}

// sampleElevationProfile extracts elevation sample points from OSRM geometry
func sampleElevationProfile(osrmResp *model.OSRMResponse) []model.ElevationPoint {
	if len(osrmResp.Routes) == 0 {
		return nil
	}

	geojson := osrmResp.Routes[0].Geometry
	if geojson == nil {
		return nil
	}

	var coords struct {
		Coordinates [][]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(*geojson, &coords); err != nil {
		return nil
	}

	// Sample ~20 points evenly along the route
	step := len(coords.Coordinates) / 20
	if step < 1 {
		step = 1
	}

	var profile []model.ElevationPoint
	totalDist := 0.0
	for i, c := range coords.Coordinates {
		if len(c) < 2 {
			continue
		}

		if i == 0 && len(coords.Coordinates) > 1 {
			// Add first point
		} else if i%step != 0 && i != len(coords.Coordinates)-1 {
			continue
		}

		if i > 0 {
			prev := coords.Coordinates[i-1]
			if len(prev) >= 2 {
				totalDist += haversine(prev[1], prev[0], c[1], c[0])
			}
		}

		profile = append(profile, model.ElevationPoint{
			DistanceKm: math.Round(totalDist*10) / 10,
			ElevationM: 0, // filled by elevation client later
			Lat:        c[1],
			Lng:        c[0],
		})
	}

	return profile
}
