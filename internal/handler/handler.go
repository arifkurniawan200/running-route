package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/arifkurniawan200/running-route/internal/model"
	"github.com/arifkurniawan200/running-route/internal/service"
)

type Handler struct {
	recommend *service.Recommendation
	geocoding *service.Geocoding
}

func New(recommend *service.Recommendation, geocoding *service.Geocoding) *Handler {
	return &Handler{recommend: recommend, geocoding: geocoding}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/routes/recommend", h.RecommendRoutes)
	mux.HandleFunc("GET /api/v1/routes/{id}/weather", h.GetRouteWeather)
	mux.HandleFunc("GET /api/v1/geocode/search", h.GeocodeSearch)
	mux.HandleFunc("GET /api/v1/health", h.Health)
}

func (h *Handler) RecommendRoutes(w http.ResponseWriter, r *http.Request) {
	var req model.RecommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Lat == 0 || req.Lng == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng are required"})
		return
	}
	if req.DurationMinutes <= 0 {
		req.DurationMinutes = 60
	}
	if req.Pace == "" {
		req.Pace = "casual"
	}

	resp, err := h.recommend.GetRoutes(&req)
	if err != nil {
		log.Printf("recommend error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetRouteWeather(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	hoursStr := r.URL.Query().Get("hours")

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)
	hours, _ := strconv.Atoi(hoursStr)
	if hours <= 0 {
		hours = 6
	}

	if lat == 0 || lng == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng query params required"})
		return
	}

	weather, err := h.recommend.GetWeather(lat, lng, hours)
	if err != nil {
		log.Printf("weather error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, weather)
}

func (h *Handler) GeocodeSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q query param required"})
		return
	}

	result, err := h.geocoding.Search(q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "running-route"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
