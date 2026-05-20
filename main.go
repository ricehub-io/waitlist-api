package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/ricehub-io/waitlist-api/internal/logger"

	_ "github.com/ricehub-io/waitlist-api/docs"
	"github.com/ricehub-io/waitlist-api/internal/config"
	"github.com/ricehub-io/waitlist-api/internal/db"
	"github.com/ricehub-io/waitlist-api/internal/handlers"
	"github.com/ricehub-io/waitlist-api/internal/middlewares"
	"github.com/ricehub-io/waitlist-api/internal/storage"
)

// @title RiceHub Waitlist API
// @version 1.0.0
// @description REST API for RiceHub waitlist frontend.

// @host 127.0.0.1:3000
// @BasePath /

// @securityDefinitions.apikey AdminSecret
// @in header
// @name X-Admin-Secret

func main() {
	os.Exit(setup())
}

func setup() int {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cfg, err := config.NewConfig(ctx)
	if err != nil {
		log.Printf("Initializing config: %v", err)
		return 1
	}

	l, err := logger.Init(zap.InfoLevel, cfg.SentryDSN, cfg.Environment)
	if err != nil {
		log.Printf("Initializing logger: %v", err)
		return 1
	}
	defer logger.Sync(l)

	if err := run(l, cfg); err != nil {
		l.Error("Run failed", zap.Error(err))
		return 1
	}

	return 0
}

func run(l *zap.Logger, cfg *config.Config) error {
	// TODO: use signal and graceful shutdown

	if cfg.IsProd() {
		l.Info("Running in production mode")
	} else {
		l.Warn("Running in development mode")
	}

	if err := handlers.InitCustomValidation(); err != nil {
		return fmt.Errorf("initializing custom validation: %w", err)
	}

	if cfg.DiscordWebhookURL == "" {
		l.Info("Discord notifications disabled")
	}
	if !cfg.WithSentry() {
		l.Warn("Sentry error logging disabled")
	}

	db, err := db.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	storage, err := storage.NewStorage(cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3BaseURL)
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}

	h := handlers.NewHandler(l, cfg, db, storage)
	limiter := middlewares.NewIPRateLimiter(cfg.RateLimitPerMinute, cfg.RateLimitBurst, time.Hour)

	r, err := newRouter(l, cfg, h, limiter)
	if err != nil {
		return err
	}

	l.Sugar().Infof("Listening on %s", cfg.BindAddress)
	if err := r.Run(cfg.BindAddress); err != nil {
		return fmt.Errorf("running gin router: %w", err)
	}

	return nil
}

func newRouter(
	l *zap.Logger,
	cfg *config.Config,
	h *handlers.Handler,
	limiter *middlewares.IPRateLimiter,
) (*gin.Engine, error) {
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("setting trusted proxies: %w", err)
	}

	if cfg.WithSentry() {
		r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	}
	r.Use(
		cors.New(cors.Config{
			AllowOrigins: []string{cfg.CORSOrigin},
			AllowMethods: []string{"GET", "POST"},
			AllowHeaders: []string{"Origin", "Content-Type"},
		}),
		gin.Recovery(),
	)
	if cfg.WithSentry() {
		r.Use(middlewares.RequestLogger(l))
	}

	rl := middlewares.RateLimitMiddleware(limiter)

	wr := r.Group("/waitlist")
	wr.GET("", h.GetWaitlistEmailCount)
	wr.POST("", rl, h.CreateWaitlistEmail)

	fr := r.Group("/founders")
	fr.GET("", h.GetFoundingCreatorStats)
	fr.POST("", rl, h.CreateFoundingCreator)

	rr := r.Group("/rices")
	rr.GET("", h.GetPreviewRices)
	rr.POST("", middlewares.AdminMiddleware(cfg.AdminSecret), h.CreatePreviewRice)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
	))

	return r, nil
}
