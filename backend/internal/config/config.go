package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort     string
	DatabaseURL    string
	JWTSecret      string
	LNDClientMode  string
	LNDHost        string
	LNDRESTHost    string
	LNDMacaroon    string
	LNDMacaroonHex string
	LNDTLS         string
	ServerSecret   string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		// DATABASE_URL intentionally has no fallback. Silently starting with a
		// local database can split financial state between environments.
		DatabaseURL: getEnv("DATABASE_URL", ""),
		// Authentication must never silently start with a shared development
		// secret. Router construction fails unless deployment provides one.
		JWTSecret:      getEnv("JWT_SECRET", ""),
		LNDClientMode:  getEnv("LND_CLIENT_MODE", "grpc"),
		LNDHost:        getEnv("LND_HOST", "localhost:10009"),
		LNDRESTHost:    getEnv("LND_REST_HOST", "https://localhost:8080"),
		LNDMacaroon:    getEnv("LND_MACAROON", ""),
		LNDMacaroonHex: getEnv("LND_MACAROON_HEX", ""),
		LNDTLS:         getEnv("LND_TLS_PATH", ""),
		ServerSecret:   getEnv("SERVER_SECRET", "ledger-hmac-secret"),
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
