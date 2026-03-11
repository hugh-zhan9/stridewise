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
		DataFile    string `yaml:"data_file"`
		PhoneNumber string `yaml:"phone_number"`
		Password    string `yaml:"password"`
	} `yaml:"keep"`
	Strava struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"strava"`
	Garmin struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"garmin"`
	Nike struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"nike"`
	GPX struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"gpx"`
	TCX struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"tcx"`
	FIT struct {
		DataFile string `yaml:"data_file"`
	} `yaml:"fit"`
	Weather struct {
		QWeather struct {
			APIKey    string `yaml:"api_key"`
			APIHost   string `yaml:"api_host"`
			TimeoutMs int    `yaml:"timeout_ms"`
		} `yaml:"qweather"`
	} `yaml:"weather"`
	AI struct {
		Provider string `yaml:"provider"`
		OpenAI   struct {
			APIKey      string  `yaml:"api_key"`
			BaseURL     string  `yaml:"base_url"`
			Model       string  `yaml:"model"`
			TimeoutMs   int     `yaml:"timeout_ms"`
			MaxTokens   int     `yaml:"max_tokens"`
			Temperature float64 `yaml:"temperature"`
		} `yaml:"openai"`
	} `yaml:"ai"`
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
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = "openai"
	}
	return &cfg, nil
}
