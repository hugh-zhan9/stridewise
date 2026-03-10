package main

import (
	"context"
	"flag"
	"log"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"stridewise/backend/internal/config"
	fitconnector "stridewise/backend/internal/connector/fit"
	garminconnector "stridewise/backend/internal/connector/garmin"
	gpxconnector "stridewise/backend/internal/connector/gpx"
	keepconnector "stridewise/backend/internal/connector/keep"
	nikeconnector "stridewise/backend/internal/connector/nike"
	stravaconnector "stridewise/backend/internal/connector/strava"
	tcxconnector "stridewise/backend/internal/connector/tcx"
	"stridewise/backend/internal/storage"
	syncjob "stridewise/backend/internal/sync"
	"stridewise/backend/internal/task"
	"stridewise/backend/internal/worker"
)

func main() {
	confPath := flag.String("conf", "config/config.yaml", "config path")
	flag.Parse()

	cfg, err := config.Load(*confPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN)
	if err != nil {
		log.Fatalf("connect postgres failed: %v", err)
	}
	defer pool.Close()

	store := storage.NewPostgresStore(pool)
	processor := syncjob.NewProcessor(store, map[string]syncjob.Connector{
		"keep":   keepconnector.NewLive(cfg.Keep.PhoneNumber, cfg.Keep.Password, "", nil),
		"strava": stravaconnector.New(cfg.Strava.DataFile),
		"garmin": garminconnector.New(cfg.Garmin.DataFile),
		"nike":   nikeconnector.New(cfg.Nike.DataFile),
		"gpx":    gpxconnector.New(cfg.GPX.DataFile),
		"tcx":    tcxconnector.New(cfg.TCX.DataFile),
		"fit":    fitconnector.New(cfg.FIT.DataFile),
	})
	worker.SetProcessor(processor)

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
		asynq.Config{Concurrency: cfg.Asynq.Concurrency},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeSyncJob, worker.HandleSyncJob)

	if err := server.Run(mux); err != nil {
		log.Fatalf("worker run failed: %v", err)
	}
}
