package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all environment configurations.
type Config struct {
	DBPath        string
	Port          string
	SessionSecret string
	AppEnv        string
}

// Load loads environment variables from a .env file and environment.
func Load() (*Config, error) {
	// Optional load .env file (ignore error if file not present)
	_ = godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "layerinvoice.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "layerinvoice-secret-key-change-me"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	return &Config{
		DBPath:        dbPath,
		Port:          port,
		SessionSecret: sessionSecret,
		AppEnv:        appEnv,
	}, nil
}

// GetEnvAsInt is a helper to read an environment variable as integer.
func GetEnvAsInt(name string, defaultVal int) int {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultVal
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultVal
	}
	return value
}
