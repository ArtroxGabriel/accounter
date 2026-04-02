package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Port                string // HTTP port (default: "8080")
	DatabasePath        string // SQLite file path (default: "accounter.db")
	BearerToken         string // Auth token (required)
	TelegramToken       string // Telegram bot API token (required)
	TelegramAllowedChat string // Optional: restrict bot to one chat ID
	LogLevel            string // "debug", "info", "warn", "error" (default: "info")
	Environment         string // "development" or "production" (default: "development")
	Timezone            string // IANA timezone for display (default: "America/Sao_Paulo")
}

var (
	ErrMissingBearerToken   = errors.New("missing BEARER_TOKEN environment variable")
	ErrMissingTelegramToken = errors.New("missing TELEGRAM_TOKEN environment variable")
)

// Load reads configuration from environment variables and .env file.
func Load() (Config, error) {
	// Attempt to load .env file. Failure is non-fatal for production usage.
	_ = godotenv.Load()

	cfg := Config{
		Port:                getEnv("PORT", "8080"),
		DatabasePath:        getEnv("DATABASE_PATH", "accounter.db"),
		BearerToken:         os.Getenv("BEARER_TOKEN"),
		TelegramToken:       os.Getenv("TELEGRAM_TOKEN"),
		TelegramAllowedChat: os.Getenv("TELEGRAM_ALLOWED_CHAT_ID"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		Timezone:            getEnv("TIMEZONE", "America/Sao_Paulo"),
	}

	if cfg.BearerToken == "" {
		return Config{}, ErrMissingBearerToken
	}
	if cfg.TelegramToken == "" {
		return Config{}, ErrMissingTelegramToken
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
