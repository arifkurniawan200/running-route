package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arifkurniawan200/running-route/config"
	"github.com/arifkurniawan200/running-route/internal/model"
)

type Overpass struct {
	baseURL string
	client  *http.Client
}

func NewOverpass(cfg *config.Config) *Overpass {
	return &Overpass{
		baseURL: cfg.OverpassURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Overpass) QueryParksAndPaths(lat, lng, radiusM float64) (*model.OverpassResponse, error) {
	query := fmt.Sprintf(`[out:json];
(
  node["leisure"="park"](around:%v,%v,%v);
  way["leisure"="park"](around:%v,%v,%v);
  way["highway"="footway"](around:%v,%v,%v);
  way["highway"="pedestrian"](around:%v,%v,%v);
  way["leisure"="track"]["sport"="running"](around:%v,%v,%v);
  way["leisure"="track"](around:%v,%v,%v);
);
out geom;`, radiusM, lat, lng, radiusM, lat, lng, radiusM, lat, lng, radiusM, lat, lng, radiusM, lat, lng, radiusM, lat, lng)

	return c.doRequest(query)
}

func (o *Overpass) doRequest(query string) (*model.OverpassResponse, error) {
	formData := url.Values{}
	formData.Set("data", query)

	req, err := http.NewRequest("POST", o.baseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("overpass request create failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "RunningRouteRecommender/1.0 (arif@arif-kurniawan.site)")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("overpass request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("overpass read failed: %w", err)
	}

	// Check if Overpass returned HTML (rate limit / error page)
	if isHTML(body) {
		return nil, fmt.Errorf("overpass returned HTML (rate limited?): %s", string(body[:min(len(body), 200)]))
	}

	var result model.OverpassResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("overpass decode failed: %w (body: %s)", err, string(body[:min(len(body), 200)]))
	}

	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isHTML checks if response body starts with '<' (HTML error page)
func isHTML(body []byte) bool {
	return len(body) > 0 && body[0] == '<'
}

// --- Open-Meteo Elevation Client ---

type ElevationClient struct {
	baseURL string
	client  *http.Client
}

func NewElevationClient(cfg *config.Config) *ElevationClient {
	return &ElevationClient{
		baseURL: cfg.OpenMeteoURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *ElevationClient) GetElevation(lats, lngs []float64) (*model.OpenMeteoElevationResponse, error) {
	latStr := make([]string, len(lats))
	lngStr := make([]string, len(lngs))
	for i := range lats {
		latStr[i] = fmt.Sprintf("%.6f", lats[i])
		lngStr[i] = fmt.Sprintf("%.6f", lngs[i])
	}

	params := url.Values{}
	params.Set("latitude", strings.Join(latStr, ","))
	params.Set("longitude", strings.Join(lngStr, ","))
	u := fmt.Sprintf("%s/elevation?%s", c.baseURL, params.Encode())

	resp, err := c.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("elevation request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elevation read failed: %w", err)
	}

	if isHTML(body) {
		return nil, fmt.Errorf("elevation rate limited")
	}

	var result model.OpenMeteoElevationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("elevation decode failed: %w", err)
	}

	return &result, nil
}

// --- OSRM Client ---

type OSRM struct {
	baseURL string
	client  *http.Client
}

func NewOSRM(cfg *config.Config) *OSRM {
	return &OSRM{
		baseURL: cfg.OSRMURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *OSRM) RouteFoot(startLat, startLng, endLat, endLng float64) (*model.OSRMResponse, error) {
	u := fmt.Sprintf("%s/route/v1/foot/%v,%v;%v,%v?geometries=geojson&overview=full&steps=true&annotations=true",
		c.baseURL, startLng, startLat, endLng, endLat)

	resp, err := c.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("osrm request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("osrm read failed: %w", err)
	}

	var result model.OSRMResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("osrm decode failed: %w", err)
	}

	return &result, nil
}

// --- Open-Meteo Client ---

type OpenMeteo struct {
	baseURL string
	client  *http.Client
}

func NewOpenMeteo(cfg *config.Config) *OpenMeteo {
	return &OpenMeteo{
		baseURL: cfg.OpenMeteoURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *OpenMeteo) GetForecast(lat, lng float64, startHour, endHour string) (*model.OpenMeteoResponse, error) {
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%.4f", lat))
	params.Set("longitude", fmt.Sprintf("%.4f", lng))
	params.Set("hourly", "temperature_2m,precipitation_probability,wind_speed_10m,uv_index")
	params.Set("timezone", "auto")
	if startHour != "" {
		params.Set("start_hour", startHour)
	}
	if endHour != "" {
		params.Set("end_hour", endHour)
	}

	u := fmt.Sprintf("%s/forecast?%s", c.baseURL, params.Encode())

	resp, err := c.client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("open-meteo request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("open-meteo read failed: %w", err)
	}

	var result model.OpenMeteoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("open-meteo decode failed: %w", err)
	}

	return &result, nil
}

// --- Nominatim Client ---

type Nominatim struct {
	baseURL string
	client  *http.Client
}

func NewNominatim(cfg *config.Config) *Nominatim {
	return &Nominatim{
		baseURL: cfg.NominatimURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Nominatim) ReverseGeocode(lat, lng float64) (*model.NominatimResponse, error) {
	params := url.Values{}
	params.Set("lat", fmt.Sprintf("%.6f", lat))
	params.Set("lon", fmt.Sprintf("%.6f", lng))
	params.Set("format", "json")

	u := fmt.Sprintf("%s/reverse?%s", c.baseURL, params.Encode())

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "RunningRouteRecommender/1.0 (arif@arif-kurniawan.site)")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim reverse request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nominatim reverse read failed: %w", err)
	}

	if isHTML(body) {
		return nil, fmt.Errorf("nominatim rate limited: %s", string(body[:min(len(body), 200)]))
	}

	var result model.NominatimResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("nominatim reverse decode failed: %w", err)
	}

	return &result, nil
}

func (c *Nominatim) Search(query string) (*model.NominatimResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("limit", "5")

	u := fmt.Sprintf("%s/search?%s", c.baseURL, params.Encode())

	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("User-Agent", "RunningRouteRecommender/1.0 (arif@arif-kurniawan.site)")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("nominatim search read failed: %w", err)
	}

	if isHTML(body) {
		return nil, fmt.Errorf("nominatim rate limited: %s", string(body[:min(len(body), 200)]))
	}

	var result model.NominatimResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("nominatim search decode failed: %w", err)
	}

	return &result, nil
}
