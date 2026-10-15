package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName            string
	AppEnv             string
	AppPort            string
	DatabaseURL        string
	JWTSecret          string
	JWTExpirationHours int
	DefaultTaxRateBps  int64 // Basis points: 1100 = 11.00%
}

func LoadConfig() (*Config, error) {
	// Load .env if present; ignore error if not found in production
	_ = godotenv.Load()

	appName := getEnv("APP_NAME", "Go Fiber POS Backend")
	appEnv := getEnv("APP_ENV", "development")
	appPort := getEnv("APP_PORT", "8080")
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:1234@localhost:5432/pos_db?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "pos-super-secret-key-change-in-production-2026")

	jwtExpHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil || jwtExpHours <= 0 {
		jwtExpHours = 24
	}

	taxRateBps, err := strconv.ParseInt(getEnv("DEFAULT_TAX_RATE_BPS", "1100"), 10, 64)
	if err != nil || taxRateBps < 0 {
		taxRateBps = 1100 // 11%
	}

	return &Config{
		AppName:            appName,
		AppEnv:             appEnv,
		AppPort:            appPort,
		DatabaseURL:        databaseURL,
		JWTSecret:          jwtSecret,
		JWTExpirationHours: jwtExpHours,
		DefaultTaxRateBps:  taxRateBps,
	}, nil
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
