package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
}

func Load() Config {
	godotenv.Load()

	cfg := Config{
		DatabaseURL: "postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable",
		Port:        "8080",
		JWTSecret:   "dev-secret-change-me",
	}

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	return cfg
}
