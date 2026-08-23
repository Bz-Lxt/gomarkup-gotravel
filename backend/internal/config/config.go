package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	RedisAddr        string
	JWTSecret        string
	GeoIndexBackend  string // rtree | redis
	RouteProvider    string // haversine | osrm
	GPSProvider      string // live | sim
	UploadDir        string
	CORSOrigin       string
	LogLevel         string
	OSRMBaseURL      string
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://gotravel:gotravel@127.0.0.1:5432/gotravel?sslmode=disable"),
		RedisAddr:       getenv("REDIS_ADDR", "127.0.0.1:6379"),
		JWTSecret:       getenv("JWT_SECRET", "gotravel-dev-secret-change-me"),
		GeoIndexBackend: strings.ToLower(getenv("GEO_INDEX_BACKEND", "rtree")),
		RouteProvider:   strings.ToLower(getenv("ROUTE_PROVIDER", "haversine")),
		GPSProvider:     strings.ToLower(getenv("GPS_PROVIDER", "sim")),
		UploadDir:       getenv("UPLOAD_DIR", "./uploads"),
		CORSOrigin:      getenv("CORS_ORIGIN", "http://localhost:27181"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		OSRMBaseURL:     getenv("OSRM_BASE_URL", ""),
	}
}
