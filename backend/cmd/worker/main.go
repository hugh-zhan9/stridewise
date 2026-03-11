package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"stridewise/backend/internal/ability"
	"stridewise/backend/internal/ai"
	"stridewise/backend/internal/asyncjob"
	"stridewise/backend/internal/baseline"
	"stridewise/backend/internal/config"
	fitconnector "stridewise/backend/internal/connector/fit"
	garminconnector "stridewise/backend/internal/connector/garmin"
	gpxconnector "stridewise/backend/internal/connector/gpx"
	keepconnector "stridewise/backend/internal/connector/keep"
	nikeconnector "stridewise/backend/internal/connector/nike"
	"stridewise/backend/internal/recommendation"
	stravaconnector "stridewise/backend/internal/connector/strava"
	tcxconnector "stridewise/backend/internal/connector/tcx"
	"stridewise/backend/internal/storage"
	syncjob "stridewise/backend/internal/sync"
	"stridewise/backend/internal/task"
	"stridewise/backend/internal/training"
	"stridewise/backend/internal/weather"
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
	baselineEnqueuer := asyncjob.NewBaselineEnqueuer(store, asynqClient)
	processor := syncjob.NewProcessor(store, map[string]syncjob.Connector{
		"keep":   keepconnector.NewLive(cfg.Keep.PhoneNumber, cfg.Keep.Password, "", nil),
		"strava": stravaconnector.New(cfg.Strava.DataFile),
		"garmin": garminconnector.New(cfg.Garmin.DataFile),
		"nike":   nikeconnector.New(cfg.Nike.DataFile),
		"gpx":    gpxconnector.New(cfg.GPX.DataFile),
		"tcx":    tcxconnector.New(cfg.TCX.DataFile),
		"fit":    fitconnector.New(cfg.FIT.DataFile),
	})
	processor.SetBaselineEnqueuer(baselineEnqueuer)
	processor.SetAbilityEnqueuer(asyncjob.NewAbilityLevelEnqueuer(store, asynqClient))
	worker.SetSyncProcessor(processor)

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

	weatherProvider := buildWeatherProvider(cfg)
	recProcessor := buildRecommendationProcessor(store, weatherProvider, cfg)
	worker.SetTrainingProcessor(buildTrainingProcessor(store, baselineProcessor, recProcessor))

	var abilityLeveler ai.AbilityLeveler
	if cfg.AI.Provider == "openai" {
		abilityLeveler = ai.NewOpenAIAbilityLeveler(ai.OpenAIConfig{
			APIKey:      cfg.AI.OpenAI.APIKey,
			BaseURL:     cfg.AI.OpenAI.BaseURL,
			Model:       cfg.AI.OpenAI.Model,
			TimeoutMs:   cfg.AI.OpenAI.TimeoutMs,
			MaxTokens:   cfg.AI.OpenAI.MaxTokens,
			Temperature: cfg.AI.OpenAI.Temperature,
		})
	}
	abilityProcessor := ability.NewProcessor(store, abilityLeveler)
	worker.SetAbilityProcessor(abilityProcessor)

	go runNightlyScheduler(context.Background(), store, baselineEnqueuer, time.Now)

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.Redis.Addr},
		asynq.Config{Concurrency: cfg.Asynq.Concurrency},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(task.TypeSyncJob, worker.HandleSyncJob)
	mux.HandleFunc(task.TypeTrainingRecalc, worker.HandleTrainingRecalc)
	mux.HandleFunc(task.TypeBaselineRecalc, worker.HandleBaselineRecalc)
	mux.HandleFunc(task.TypeAbilityLevelCalc, worker.HandleAbilityLevelCalc)

	if err := server.Run(mux); err != nil {
		log.Fatalf("worker run failed: %v", err)
	}
}

func buildWeatherProvider(cfg *config.Config) weather.Provider {
	mockProvider := weather.NewMockProvider(weather.SnapshotInput{
		TemperatureC:      20,
		FeelsLikeC:        20,
		Humidity:          0.5,
		WindSpeedMS:       2,
		PrecipitationProb: 0.1,
		AQI:               50,
		UVIndex:           3,
	})
	if cfg == nil || cfg.Weather.QWeather.APIKey == "" || cfg.Weather.QWeather.APIHost == "" {
		return mockProvider
	}
	return weather.NewQWeatherProvider(weather.QWeatherConfig{
		APIKey:    cfg.Weather.QWeather.APIKey,
		APIHost:   cfg.Weather.QWeather.APIHost,
		TimeoutMs: cfg.Weather.QWeather.TimeoutMs,
	})
}

func buildRecommendationProcessor(store *storage.PostgresStore, provider weather.Provider, cfg *config.Config) *recommendation.Processor {
	var recommender ai.Recommender
	if cfg != nil && cfg.AI.Provider == "openai" {
		recommender = ai.NewOpenAIRecommender(ai.OpenAIConfig{
			APIKey:      cfg.AI.OpenAI.APIKey,
			BaseURL:     cfg.AI.OpenAI.BaseURL,
			Model:       cfg.AI.OpenAI.Model,
			TimeoutMs:   cfg.AI.OpenAI.TimeoutMs,
			MaxTokens:   cfg.AI.OpenAI.MaxTokens,
			Temperature: cfg.AI.OpenAI.Temperature,
		})
	}
	processor := recommendation.NewProcessor(store, provider, recommender)
	if cfg != nil {
		processor.SetAIInfo(cfg.AI.Provider, cfg.AI.OpenAI.Model)
	}
	return processor
}

func buildTrainingProcessor(store training.AsyncJobStore, baseline training.BaselineRecalculator, rec training.RecommendationService) *training.Processor {
	return training.NewProcessor(store, baseline, rec)
}
