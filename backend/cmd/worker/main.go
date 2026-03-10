package main

import (
	"context"
	"flag"
	"log"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/asyncjob"
	"stridewise/backend/internal/baseline"
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
	"stridewise/backend/internal/training"
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
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Addr})
	defer asynqClient.Close()
	processor := syncjob.NewProcessor(store, map[string]syncjob.Connector{
		"keep":   keepconnector.NewLive(cfg.Keep.PhoneNumber, cfg.Keep.Password, "", nil),
		"strava": stravaconnector.New(cfg.Strava.DataFile),
		"garmin": garminconnector.New(cfg.Garmin.DataFile),
		"nike":   nikeconnector.New(cfg.Nike.DataFile),
		"gpx":    gpxconnector.New(cfg.GPX.DataFile),
		"tcx":    tcxconnector.New(cfg.TCX.DataFile),
		"fit":    fitconnector.New(cfg.FIT.DataFile),
	})
	processor.SetBaselineEnqueuer(asyncjob.NewBaselineEnqueuer(store, asynqClient))
	worker.SetSyncProcessor(processor)
	worker.SetTrainingProcessor(training.NewProcessor(store))

	var summarizer ai.Summarizer
	if cfg.AI.Provider == "openai" {
		summarizer = ai.NewOpenAISummarizer(ai.OpenAIConfig{
			APIKey:      cfg.AI.OpenAI.APIKey,
			BaseURL:     cfg.AI.OpenAI.BaseURL,
			Model:       cfg.AI.OpenAI.Model,
			TimeoutMs:   cfg.AI.OpenAI.TimeoutMs,
			MaxTokens:   cfg.AI.OpenAI.MaxTokens,
			Temperature: cfg.AI.OpenAI.Temperature,
		})
	}
	baselineProcessor := baseline.NewProcessor(store)
	baselineProcessor.SetSummarizer(summarizer)
	worker.SetBaselineProcessor(baselineProcessor)

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
		asynq.Config{Concurrency: cfg.Asynq.Concurrency},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeSyncJob, worker.HandleSyncJob)
	mux.HandleFunc(task.TypeTrainingRecalc, worker.HandleTrainingRecalc)
	mux.HandleFunc(task.TypeBaselineRecalc, worker.HandleBaselineRecalc)

	if err := server.Run(mux); err != nil {
		log.Fatalf("worker run failed: %v", err)
	}
}
