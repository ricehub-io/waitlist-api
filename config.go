package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)


type Config struct {
	Port               string
	DatabaseURL        string
	CORSOrigin         string
	S3BaseURL          string
	S3MediaBucket      string
	S3AccessKey        string
	S3SecretKey        string
	AdminSecret        string
	RateLimitPerMinute int
	RateLimitBurst     int
}

// NewConfig loads .env file and parses it into new config struct.
// Returns error if file could not be loaded.
// Exits if any required environment variable is missing.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	return &Config{
		Port:               getOptEnv("PORT", "3000"),
		DatabaseURL:        getEnv("DATABASE_URL"),
		CORSOrigin:         getOptEnv("CORS_ORIGIN", "http://127.0.0.1:5173"),
		S3BaseURL:          getEnv("S3_BASE_URL"),
		S3MediaBucket:      getEnv("S3_MEDIA_BUCKET"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY"),
		S3SecretKey:        getEnv("S3_SECRET_KEY"),
		AdminSecret:        getEnv("ADMIN_SECRET"),
		RateLimitPerMinute: getOptEnvInt("RATE_LIMIT_PER_MINUTE", 5),
		RateLimitBurst:     getOptEnvInt("RATE_LIMIT_BURST", 3),
	}, nil
}

// getEnv fetches given environment variable, exiting if it's not set.
//
// Use it to get environment variables that are required and can't have a default value.
func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return val
}

// getOptEnv fetches an environment variable defaulting to given value if not set.
func getOptEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// getOptEnvInt fetches an integer environment variable defaulting to fallback if not set or invalid.
func getOptEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
