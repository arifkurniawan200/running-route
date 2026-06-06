# 🏃 Running Route Recommender

**Find the best running routes near you — with real-time weather forecast ⛅**

> Gabungin OSM data + Open-Meteo weather + rute yang di-generated dari Go backend, ditampilkan di Leaflet.

---

## 📋 Daftar Isi

- [Arsitektur](#-arsitektur)
- [Tech Stack](#-tech-stack)
- [API Endpoints](#-api-endpoints)
- [Data Flow](#-data-flow)
- [Cara Instalasi & Run](#-cara-instalasi--run)
- [Frontend Repo](#-frontend)
- [Use Case Scenarios](#-use-case-scenarios)
- [Project Structure](#-project-structure)

---

## 🏗️ Arsitektur

```
┌─────────────┐     HTTPS      ┌──────────────────────┐
│   User      │ ──────────────▶│   Next.js Frontend   │
│  Browser /  │ ◀──────────────│   (react-leaflet)    │
│   Mobile    │     JSON       └──────────┬───────────┘
└─────────────┘                            │
                                           │ POST /api/v1/routes/recommend
                                           ▼
┌────────────────────────────────────────────────────────┐
│                  Golang Backend (:8080)                 │
│                                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐ │
│  │   Handler    │  │   Service   │  │    Clients      │ │
│  │  (Chi/HTTP)  │──▶   Layer    │──▶ (Overpass, OSRM, │ │
│  │              │  │             │  │  Open-Meteo,    │ │
│  └─────────────┘  └─────────────┘  │  Nominatim)     │ │
│                                     └─────────────────┘ │
│  ┌─────────────────────────────────────────────────┐   │
│  │           Redis Cache (in-memory fallback)      │   │
│  └─────────────────────────────────────────────────┘   │
└────────────────────────────────────────────────────────┘
           │                    │                  │
           ▼                    ▼                  ▼
   ┌─────────────┐    ┌──────────────┐    ┌──────────────┐
   │ OSM Overpass │    │  OSRM Router │    │  Open-Meteo  │
   │ parks, paths │    │ foot routing │    │  forecast 6h │
   │ & footways   │    │              │    │  (no API key)│
   └─────────────┘    └──────────────┘    └──────────────┘
```

---

## 🛠️ Tech Stack

### Backend (Golang)
| Komponen | Library | Fungsi |
|---|---|---|
| HTTP Router | `net/http` (Go 1.22+ native) | Route handling, CORS |
| Cache | In-memory map + TTL | Skip Redis untuk MVP |
| OSM Query | Overpass API | Cari park, footway, track |
| Routing Engine | OSRM API | Generate jalan kaki / lari |
| Weather | Open-Meteo API | Forecast 6 jam ke depan |
| Geocoding | Nominatim API | Reverse & search lokasi |

### Frontend (Next.js)
| Komponen | Library | Fungsi |
|---|---|---|
| Map Viewer | Leaflet.js + react-leaflet | Render OSM tiles |
| Route Line | GeoJSON polyline | Tampilin rute lari |
| Weather Strip | Custom React component | Cuaca per-jam, swipeable |
| UI Framework | Tailwind CSS + shadcn/ui | Dark theme modern |
| Marker Cluster | leaflet.markercluster | Cluster markers kalo banyak POI |

### External APIs (gratis, tanpa API key)
| API | Rate Limit | Data |
|---|---|---|
| OSM Overpass | ~10.000 req/hari | Park, footway, pedestrian, track |
| OSRM | ~1.000 req/menit | Foot & car routing |
| Open-Meteo | unlimited | Suhu, hujan, angin, UV |
| Nominatim | 1 req/detik | Geocoding & reverse geocode |

---

## 🔌 API Endpoints

### `POST /api/v1/routes/recommend`

Request:
```json
{
  "lat": -6.2088,
  "lng": 106.8456,
  "duration_minutes": 60,
  "pace": "casual",
  "route_types": ["loop", "out_and_back"]
}
```

Response:
```json
{
  "routes": [
    {
      "id": "loop-12345",
      "name": "Lari di GBK Park",
      "type": "loop",
      "distance_km": 8.5,
      "estimated_duration_min": 59,
      "difficulty": "easy",
      "surface": "asphalt",
      "elevation_gain_m": 25,
      "rating": 4,
      "geojson": { "type": "LineString", "coordinates": [[...]] },
      "waypoints": [
        { "lat": -6.2088, "lng": 106.8456, "name": "Start - GBK Park" },
        { "lat": -6.2150, "lng": 106.8500, "name": "Belok" }
      ],
      "nearby_pois": [
        { "name": "Toilet Umum", "lat": -6.2100, "lng": 106.8460, "type": "toilet" }
      ],
      "weather": {
        "hourly": [
          { "time": "06:00", "temperature_c": 26, "precipitation_probability": 10, "wind_speed_kmh": 8, "uv_index": 2, "condition": "cerah" }
        ],
        "summary": "Suhu 26-29°C, cocok lari! 🏃",
        "recommendation": "recommended",
        "alerts": []
      }
    }
  ]
}
```

### `GET /api/v1/routes/{id}/weather?lat=-6.2088&lng=106.8456&hours=6`

### `GET /api/v1/geocode/search?q=Stadion GBK Jakarta`

### `GET /api/v1/health`

---

## 🔄 Data Flow

```
1. User pilih lokasi + durasi lari
       │
       ▼
2. Backend query OSM Overpass API
   → cari park, footway, track dalam radius 3km
       │
       ▼
3. Generate candidate routes:
   - Loop: start → titik terjauh → balik
   - Out & Back: lurus, balik ke titik awal
   - Point-to-Point: A → B (kalau ada transport)
       │
       ▼
4. OSRM routing (foot profile)
   → dapet jarak real (meter), geometry GeoJSON
       │
       ▼
5. Open-Meteo forecast (6 jam ke depan)
   → suhu per-jam, probabilitas hujan, angin, UV
       │
       ▼
6. Rekomendasi score system:
   ✅ Recommended: cuaca cerah, suhu 20-30°C
   ⚠️ Caution: hujan ringan / suhu > 32°C
   ❌ Not Recommended: hujan lebat / heatwave
       │
       ▼
7. Return top 3 routes ke frontend
```

---

## 🚀 Cara Instalasi & Run

### Prerequisites
- Go 1.22+
- Node.js 18+ (untuk frontend, repo terpisah)

### Backend

```bash
# Clone
git clone https://github.com/arifkurniawan200/running-route.git
cd running-route

# Install dependencies
go mod tidy

# Run (default port 8080)
go run ./cmd/server/main.go

# Atau build & run binary
go build -o running-route ./cmd/server/
./running-route
```

### Environment Variables (.env)

| Variable | Default | Deskripsi |
|---|---|---|
| `PORT` | `8080` | Port HTTP server |
| `OVERPAST_URL` | `https://overpass-api.de/api/interpreter` | OSM Overpass endpoint |
| `OSRM_URL` | `https://router.project-osrm.org` | OSRM routing endpoint |
| `OPEN_METEO_URL` | `https://api.open-meteo.com/v1` | Weather forecast endpoint |
| `NOMINATIM_URL` | `https://nominatim.openstreetmap.org` | Geocoding endpoint |

### Verify

```bash
curl http://localhost:8080/api/v1/health
# → {"status":"ok","service":"running-route"}
```

### Testing dengan sample request

```bash
curl -X POST http://localhost:8080/api/v1/routes/recommend \
  -H 'Content-Type: application/json' \
  -d '{
    "lat": -6.2088,
    "lng": 106.8456,
    "duration_minutes": 60,
    "pace": "casual"
  }' | jq .
```

---

## 💻 Frontend

Frontend Next.js ada di **repo terpisah**: [arifkurniawan200/running-route-frontend](https://github.com/arifkurniawan200/running-route-frontend)

### Tech stack frontend:
- **Next.js 14** (App Router)
- **react-leaflet** — render OSM map tiles
- **shadcn/ui** — UI components (dark theme)
- **Tailwind CSS** — styling
- **Leaflet.markercluster** — marker optimization
- **GeoJSON** — route rendering di peta

### Halaman utama:
| Halaman | Route | Fungsi |
|---|---|---|
| `/` | Home | Cari lokasi, pilih durasi, liat rekomendasi |
| `/routes/:id` | Detail Rute | Weather strip, surface info, POI, detail sheet |

### Komponen utama:
```
components/
├── LocationPicker.tsx     # Search + autocomplete Nominatim
├── RunningMap.tsx          # Leaflet map + route polyline
├── RouteCard.tsx           # Route recommendation card
├── WeatherStrip.tsx        # Hourly weather bar (swipeable)
├── RouteDetailSheet.tsx    # Slide-up detail panel
└── DurationSelector.tsx    # 30m / 1j / 1.5j / 2j
```

---

## 📁 Project Structure

```
running-route/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, HTTP server, CORS
├── config/
│   └── config.go                # Environment config
├── internal/
│   ├── cache/
│   │   └── memory.go            # In-memory cache with TTL
│   ├── client/
│   │   └── clients.go           # Overpass, OSRM, Open-Meteo, Nominatim
│   ├── handler/
│   │   └── handler.go           # HTTP handlers, response serialization
│   ├── model/
│   │   └── model.go             # All structs & external API models
│   └── service/
│       ├── route.go             # Route finding + generation logic
│       ├── weather.go           # Weather fetch + condition classifier
│       ├── recommendation.go    # Scoring, ranking, fallback routes
│       └── geocoding.go         # Forward & reverse geocoding
├── go.mod
├── go.sum
└── README.md
```

---

## 🎯 Use Case Scenarios

### 1. 🏃 Runner Mingguan
> "Bang, hari Sabtu pagi jam 6 mau lari 1 jam di sekitar GBK, cuaca gimana?"
- Input: lokasi GBK, durasi 60 menit, pace casual
- Output: 3 rute loop + weather forecast 06:00–12:00
- Decision: pilih rute yang direkomendasi (cerah) ⭐

### 2. 🌤️ Cek Cuaca Sebelum Lari
> "Cuaca di Senayan jam 7 pagi besok gimana?"
- Pencet marker di map → muncul weather per-jam
- Kalau hujan ≥ 70% → saran: "tunda atau pake treadmill"

### 3. 🏞️ Cari Rute Baru
> "Bosen lari di tempat yang sama, ada rute baru di sekitar?"
- Backend auto-generate loop & out-back dari OSM data
- Surface info: mana yang aspal, mana gravel

### 4. 🚀 Interval Training
> "Mau sprint interval 30 menit di track atletik"
- Filter: `leisure=track` + `sport=running` → track atletik
- Short loop 400m ideal buat interval

---

## 🧠 Rekomendasi Skor Logic

| Factor | +1 Score | +0 Score | -1 Score |
|---|---|---|---|
| **Surface** | Asphalt | Mixed | Gravel/grass |
| **Distance** | 3-10km | <3km | >15km |
| **Park nearby** | Yes (park) | Footway only | No path |
| **Weather** | Cerah, <30°C | Berawan | Hujan/panas |

Max score: 5 ⭐⭐⭐⭐⭐

---

## 🚧 Future Improvements

- [ ] **Elevation profile** dari OSRM
- [ ] **Safety score** (street lighting, crime data)
- [ ] **Popular routes** (crowd-sourced via history)
- [ ] **PostGIS** untuk spatial query lebih advanced
- [ ] **Redis** untuk production-ready cache
- [ ] **WebSocket** untuk real-time tracking
- [ ] **Route sharing** (generate link / image)
- [ ] **Strava integration** (sync activity)

---

## 📝 Catatan

> **MVP ini DIBUAT TANPA**: Redis, PostGIS, Docker, Nginx, PM2.
> Semua external API (Overpass, OSRM, Open-Meteo) panggil langsung dari Go — **gratis, tanpa API key**.
> Cache pake in-memory Go map + TTL — cukup untuk development & low traffic.

---

Made with 🏃‍♂️ by Arif Kurniawan
