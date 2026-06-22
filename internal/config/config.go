package config

import (
	"os"
	"time"
)

type Config struct {
	Port               string
	JWTSecret          string
	DBPath             string
	ServiceName        string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		DBPath:             getEnv("DB_PATH", "data.db"),
		ServiceName:        getEnv("SERVICE_NAME", "aicode"),
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
