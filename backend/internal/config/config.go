package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort          string
	DatabaseURL         string
	CORSAllowedOrigins  []string
	SessionCookieName   string
	SessionCookieSecure bool
	LNDClientMode       string
	LNDHost             string
	LNDRESTHost         string
	LNDMacaroon         string
	LNDMacaroonHex      string
	LNDTLS              string
	ServerSecret        string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		// DATABASE_URL intentionally has no fallback. Silently starting with a
		// local database can split financial state between environments.
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		CORSAllowedOrigins: splitEnvList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "pamojabuild_session"),
		// Production is secure by default. Local HTTP development must opt out
		// explicitly with SESSION_COOKIE_SECURE=false.
		SessionCookieSecure: getEnvAsBool("SESSION_COOKIE_SECURE", true),
		LNDClientMode:       getEnv("LND_CLIENT_MODE", "grpc"),
		LNDHost:             getEnv("LND_HOST", "localhost:10009"),
		LNDRESTHost:         getEnv("LND_REST_HOST", "https://localhost:8080"),
		LNDMacaroon:         getEnv("LND_MACAROON", ""),
		LNDMacaroonHex:      getEnv("LND_MACAROON_HEX", ""),
		LNDTLS:              getEnv("LND_TLS_PATH", ""),
		ServerSecret:        getEnv("SERVER_SECRET", "ledger-hmac-secret"),
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func splitEnvList(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
