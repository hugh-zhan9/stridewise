package main

import (
	"flag"
	"log"

	"github.com/go-kratos/kratos/v2"

	"stridewise/backend/internal/config"
	"stridewise/backend/internal/server"
)

func main() {
	confPath := flag.String("conf", "config/config.yaml", "config path")
	flag.Parse()

	cfg, err := config.Load(*confPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, cfg.Security.InternalToken)
	app := kratos.New(
		kratos.Name("stridewise-api"),
		kratos.Server(httpSrv),
	)

	if err := app.Run(); err != nil {
		log.Fatalf("run app failed: %v", err)
	}
}
