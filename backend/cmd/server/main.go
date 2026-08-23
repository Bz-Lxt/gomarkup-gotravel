package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotravel/internal/config"
	"gotravel/internal/db"
	"gotravel/internal/geo"
	"gotravel/internal/handler"
	"gotravel/internal/logger"
	"gotravel/internal/middleware"
	"gotravel/internal/redisx"
	"gotravel/internal/repository"
	"gotravel/internal/routegraph"
	"gotravel/internal/seed"
	"gotravel/internal/service"
	"gotravel/internal/simulator"
	"gotravel/internal/ws"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.L.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		logger.L.Error("migrate", "err", err)
		os.Exit(1)
	}
	repo := repository.New(pool)
	if err := seed.Run(ctx, repo); err != nil {
		logger.L.Error("seed", "err", err)
		os.Exit(1)
	}

	rdb, rerr := redisx.Connect(cfg.RedisAddr)
	idx := geo.Index(geo.NewRTreeIndex())
	geoName := "rtree"
	if cfg.GeoIndexBackend == "redis" && rerr == nil {
		idx = geo.NewRedisGeoIndex(rdb)
		geoName = "redis"
	} else if cfg.GeoIndexBackend == "redis" && rerr != nil {
		logger.L.Warn("geo fallback rtree", "err", rerr)
	}

	hub := ws.NewHub()
	hub.Geo = geo.NewEngine(idx, 2000, 60*time.Second)

	var dist routegraph.Provider = routegraph.HaversineProvider{}
	if cfg.RouteProvider == "osrm" {
		dist = routegraph.OSRMProvider{BaseURL: cfg.OSRMBaseURL}
	}

	auth := &service.Auth{Repo: repo, Secret: []byte(cfg.JWTSecret)}
	team := &service.Team{Repo: repo}
	trip := &service.Trip{Repo: repo, Team: team, DistProv: dist}
	sess := service.NewSession(repo, trip)
	photo := &service.Photo{Repo: repo, Trip: trip, UploadDir: cfg.UploadDir}
	alert := &service.Alert{Repo: repo, Session: sess, Hub: hub}
	sim := simulator.New(hub)

	hub.OnPos = func(sessionID, userID int64, lat, lng, speed, acc, heading float64, ts time.Time) {
		sess.RecordPos(context.Background(), sessionID, userID, lat, lng, speed, acc, heading, ts)
	}
	hub.OnAck = func(sessionID, userID, alertID int64) {
		_ = alert.Ack(context.Background(), userID, alertID)
	}
	hub.OnLag = func(sessionID int64, a *geo.Alert) {
		_, _ = alert.PersistLaggard(context.Background(), sessionID, a)
	}

	api := &handler.API{
		Cfg: cfg, Auth: auth, Team: team, Trip: trip, Sess: sess,
		Photo: photo, Alert: alert, Hub: hub, Sim: sim, GeoName: geoName,
	}
	_ = os.MkdirAll(cfg.UploadDir, 0o755)
	routes := api.Routes()
	mux := http.NewServeMux()
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))
	mux.Handle("/", routes)

	h := middleware.Recover(middleware.Access(middleware.CORS(cfg.CORSOrigin)(mux)))
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: h, ReadHeaderTimeout: 8 * time.Second}

	go func() {
		logger.L.Info("listen", "addr", cfg.HTTPAddr, "geo", geoName)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	hub.Stop()
	c, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = srv.Shutdown(c)
}
