package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	CORSOrigin     string
	StorageBaseURL string
}

// NewConfig loads .env file and parses it into new config struct.
// Returns error if file could not be loaded.
// Exits if any required environment variable is missing.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	port := getOptEnv("PORT", "3000")
	return &Config{
		Port:           port,
		DatabaseURL:    getEnv("DATABASE_URL"),
		CORSOrigin:     getOptEnv("CORS_ORIGIN", "http://127.0.0.1:5173"),
		StorageBaseURL: getOptEnv("STORAGE_BASE_URL", "http://127.0.0.1:"+port+"/storage"),
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
