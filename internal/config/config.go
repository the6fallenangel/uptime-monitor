package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment    string
	DatabaseURL    string
	Port           string
	JWTSecret      string
	FrontendOrigin string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	AlertFrom      string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Environment:    getEnv("ENVIRONMENT", "development"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		SMTPHost:       getEnv("SMTP_HOST", ""),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		AlertFrom:      getEnv("ALERT_FROM", ""),
		SMTPPort:       getEnvInt("SMTP_PORT", 8080),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if v, err := strconv.Atoi(value); err == nil {
			return v
		}
	}
	return fallback
}
