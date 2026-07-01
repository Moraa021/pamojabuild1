package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort    string
	DatabaseURL   string
	JWTSecret     string
	LNDHost       string
	LNDMacaroon   string
	LNDTLS        string
	ServerSecret  string // For ledger HMAC chaining
}

func Load() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://localhost:5432/pamoja?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		LNDHost:       getEnv("LND_HOST", "localhost:10009"),
		LNDMacaroon:   getEnv("LND_MACAROON", ""),
		LNDTLS:        getEnv("LND_TLS_PATH", ""),
		ServerSecret:  getEnv("SERVER_SECRET", "ledger-hmac-secret"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}