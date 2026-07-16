package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	AlertFrom   string
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
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.SMTPHost = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		cfg.SMTPPort, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("SMTP_USER"); v != "" {
		cfg.SMTPUser = v
	}
	if v := os.Getenv("SMTP_PASS"); v != "" {
		cfg.SMTPPass = v
	}
	if v := os.Getenv("ALERT_FROM"); v != "" {
		cfg.AlertFrom = v
	}
	return cfg
}
