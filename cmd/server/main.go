package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/arifkurniawan200/running-route/config"
	"github.com/arifkurniawan200/running-route/internal/cache"
	"github.com/arifkurniawan200/running-route/internal/client"
	"github.com/arifkurniawan200/running-route/internal/handler"
	"github.com/arifkurniawan200/running-route/internal/service"
)

func main() {
	cfg := config.Load()

	memCache := cache.NewMemory(cfg)
	defer memCache.Stop()

	overpass := client.NewOverpass(cfg)
	osrm := client.NewOSRM(cfg)
	openMeteo := client.NewOpenMeteo(cfg)
	elevation := client.NewElevationClient(cfg)
	nomination := client.NewNominatim(cfg)

	routeSvc := service.NewRoute(overpass, osrm, elevation)
	weatherSvc := service.NewWeather(openMeteo, memCache)
	recommendSvc := service.NewRecommendation(routeSvc, weatherSvc)
	geoAddrSvc := service.NewGeocoding(nomination)

	h := handler.New(recommendSvc, geoAddrSvc)

	mux := http.NewServeMux()
	h.Register(mux)

	// CORS middleware
	wrapped := corsMiddleware(mux)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: wrapped,
	}

	go func() {
		log.Printf("🚀 Running Route Recommender started on :%s", cfg.Port)
		log.Printf("   Caches: %s", memCache.Stats())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
