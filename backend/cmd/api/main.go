package main

import (
	"context"
	"flag"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"stridewise/backend/internal/config"
	"stridewise/backend/internal/server"
	"stridewise/backend/internal/storage"
	"stridewise/backend/internal/weather"
)

type syncJobStoreAdapter struct {
	store *storage.PostgresStore
}

func (a syncJobStoreAdapter) GetSyncJob(ctx context.Context, jobID string) (server.SyncJob, error) {
	job, err := a.store.GetSyncJob(ctx, jobID)
	if err != nil {
		return server.SyncJob{}, err
	}
	return server.SyncJob{
		JobID:        job.JobID,
		UserID:       job.UserID,
		Source:       job.Source,
		Status:       job.Status,
		RetryCount:   job.RetryCount,
		ErrorMessage: job.ErrorMessage,
	}, nil
}

func (a syncJobStoreAdapter) RetrySyncJob(ctx context.Context, jobID string) (server.SyncJob, error) {
	job, err := a.store.RetrySyncJob(ctx, jobID)
	if err != nil {
		return server.SyncJob{}, err
	}
	return server.SyncJob{
		JobID:        job.JobID,
		UserID:       job.UserID,
		Source:       job.Source,
		Status:       job.Status,
		RetryCount:   job.RetryCount,
		ErrorMessage: job.ErrorMessage,
	}, nil
}

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
	jobAdapter := syncJobStoreAdapter{store: store}
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Redis.Addr})
	defer asynqClient.Close()

	mockProvider := weather.NewMockProvider(weather.SnapshotInput{
		TemperatureC:      20,
		FeelsLikeC:        20,
		Humidity:          0.5,
		WindSpeedMS:       2,
		PrecipitationProb: 0.1,
		AQI:               50,
		UVIndex:           3,
	})

	httpSrv := server.NewHTTPServer(
		cfg.Server.HTTP.Addr,
		cfg.Security.InternalToken,
		store,
		jobAdapter,
		jobAdapter,
		asynqClient,
		store,
		store,
		mockProvider,
	)
	app := kratos.New(
		kratos.Name("stridewise-api"),
		kratos.Server(httpSrv),
	)

	if err := app.Run(); err != nil {
		log.Fatalf("run app failed: %v", err)
	}
}
