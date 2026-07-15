package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	godotenv.Load()

	cfg := Config{
		DatabaseURL: "postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable",
		Port:        "8080",
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	return cfg
}
