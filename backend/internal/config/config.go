package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		HTTP struct {
			Addr string `yaml:"addr"`
		} `yaml:"http"`
	} `yaml:"server"`
	Security struct {
		InternalToken string `yaml:"internal_token"`
	} `yaml:"security"`
	Postgres struct {
		DSN string `yaml:"dsn"`
	} `yaml:"postgres"`
	Redis struct {
		Addr string `yaml:"addr"`
	} `yaml:"redis"`
	Asynq struct {
		Concurrency int `yaml:"concurrency"`
	} `yaml:"asynq"`
	Keep struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"keep"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Server.HTTP.Addr == "" {
		cfg.Server.HTTP.Addr = ":8000"
	}
	if cfg.Asynq.Concurrency == 0 {
		cfg.Asynq.Concurrency = 10
	}
	return &cfg, nil
}
