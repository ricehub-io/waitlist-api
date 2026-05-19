package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Environment        string `env:"ENVIRONMENT, default=dev"`
	BindAddress        string `env:"BIND_ADDRESS, default=127.0.0.1:3000"`
	DatabaseURL        string `env:"DATABASE_URL, required"`
	CORSOrigin         string `env:"CORS_ORIGIN"`
	S3BaseURL          string `env:"S3_BASE_URL, required"`
	S3MediaBucket      string `env:"S3_MEDIA_BUCKET, required"`
	S3AccessKey        string `env:"S3_ACCESS_KEY, required"`
	S3SecretKey        string `env:"S3_SECRET_KEY, required"`
	AdminSecret        string `env:"ADMIN_SECRET, required"`
	RateLimitPerMinute int    `env:"RATE_LIMIT_PER_MINUTE, default=5"`
	RateLimitBurst     int    `env:"RATE_LIMIT_BURST, default=3"`
	DiscordWebhookURL  string `env:"DISCORD_WEBHOOK_URL"`
}

// NewConfig tries to load .env file in current working directory and parse it into a config struct.
// Returns error if file or variable could not be parsed.
func NewConfig(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading env file: %w", err)
	}

	if env := os.Getenv("ENVIRONMENT"); env == "" {
		doppEnv := os.Getenv("DOPPLER_ENVIRONMENT")
		if err := os.Setenv("ENVIRONMENT", doppEnv); err != nil {
			return nil, fmt.Errorf("setting 'ENVIRONMENT' env variable: %w", err)
		}
	}

	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, fmt.Errorf("processing config: %w", err)
	}

	if c.RateLimitBurst <= 0 {
		return nil, fmt.Errorf("variable 'RATE_LIMIT_PER_MINUTE' must be greater than zero")
	}

	if c.RateLimitBurst <= 0 {
		return nil, fmt.Errorf("variable 'RATE_LIMIT_BURST' must be greater than zero")
	}

	return &c, nil
}
