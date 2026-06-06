package config

import (
	"os"
	"time"
)

type Config struct {
	Port              string
	OverpassURL       string
	OSRMURL           string
	OpenMeteoURL      string
	NominatimURL      string
	CacheTTL          time.Duration // default TTL for cache entries
	OSMCacheTTL       time.Duration // OSM data changes rarely
	WeatherCacheTTL   time.Duration // weather updates every 30m
	SearchRadiusM     float64       // search radius for nearby places (meters)
	DefaultPaceMinPerKm float64     // default running pace in min/km
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		OverpassURL:    getEnv("OVERPAST_URL", "https://overpass-api.de/api/interpreter"),
		OSRMURL:        getEnv("OSRM_URL", "https://router.project-osrm.org"),
		OpenMeteoURL:   getEnv("OPEN_METEO_URL", "https://api.open-meteo.com/v1"),
		NominatimURL:   getEnv("NOMINATIM_URL", "https://nominatim.openstreetmap.org"),
		CacheTTL:       30 * time.Minute,
		OSMCacheTTL:    7 * 24 * time.Hour,
		WeatherCacheTTL: 30 * time.Minute,
		SearchRadiusM:  3000,
		DefaultPaceMinPerKm: 7.0, // ~8.5 km/h casual pace
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
