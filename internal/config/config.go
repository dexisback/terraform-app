package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Environment     string
	HTTPAddr        string
	DatabaseURL     string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:     getEnv("APP_ENV", "development"),
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://todo:todo@localhost:5432/todo?sslmode=disable"),
		ShutdownTimeout: 10 * time.Second,
	}

	rawTimeout := getEnv("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout.String())
	timeout, err := time.ParseDuration(rawTimeout)
	if err != nil || timeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration: %q", rawTimeout)
	}
	cfg.ShutdownTimeout = timeout
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
