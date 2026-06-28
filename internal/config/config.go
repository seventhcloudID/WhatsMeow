package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port       int
	APIKey     string
	DBPath     string
	WebhookURL string
	LogLevel   string
}

func Load() Config {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	return Config{
		Port:       port,
		APIKey:     getEnv("API_KEY", "changeme"),
		DBPath:     getEnv("DB_PATH", "data/session.db"),
		WebhookURL: getEnv("WEBHOOK_URL", ""),
		LogLevel:   getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
