package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
}

var cfg *Config

func Load() error {
	// Load .env file if exists (ignore error if not exists)
	_ = godotenv.Load()

	cfg = &Config{
		Port: getEnv("PORT", "3000"),
	}

	return nil
}

func Get() *Config {
	if cfg == nil {
		Load()
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
