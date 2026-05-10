///Users/ahmedhelmy/Desktop/FUE/MASTER'S/Semester 2/SE/proj/e-wallet-v2/ewallet/config/config.go
package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL                string
	JWTSecret                  string
	AccessTokenDurationMinutes int
	RefreshTokenDurationDays   int
	Port                       string
	Env                        string
}

// Load reads configuration from a .env file (if present) and environment variables.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	cfg := &Config{
		DatabaseURL:                mustGetEnv("DATABASE_URL"),
		JWTSecret:                  mustGetEnv("JWT_SECRET"),
		AccessTokenDurationMinutes: getEnvInt("ACCESS_TOKEN_DURATION_MINUTES", 15),
		RefreshTokenDurationDays:   getEnvInt("REFRESH_TOKEN_DURATION_DAYS", 7),
		Port:                       getEnv("PORT", "8080"),
		Env:                        getEnv("ENV", "development"),
	}

	return cfg
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("FATAL: required environment variable %q is not set", key)
	}
	return val
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
		log.Printf("WARNING: env var %q is not a valid integer, using default %d", key, defaultVal)
	}
	return defaultVal
}
