package main

import (
	"flag"
	"log"

	"github.com/hibiken/asynq"

	"stridewise/backend/internal/config"
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
