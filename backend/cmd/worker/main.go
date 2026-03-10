package main

import (
	"context"
	"flag"
	"log"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"stridewise/backend/internal/config"
	keepconnector "stridewise/backend/internal/connector/keep"
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
		"keep": keepconnector.New(cfg.Keep.DataFile),
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
