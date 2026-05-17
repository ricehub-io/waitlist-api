package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/ricehub-io/waitlist-api/docs"
	"github.com/ricehub-io/waitlist-api/internal/config"
	"github.com/ricehub-io/waitlist-api/internal/db"
	"github.com/ricehub-io/waitlist-api/internal/handlers"
	"github.com/ricehub-io/waitlist-api/internal/middlewares"
	"github.com/ricehub-io/waitlist-api/internal/storage"
)

// @title RiceHub Waitlist API
// @version 1.5.0
// @description REST API for RiceHub waitlist frontend.

// @host 127.0.0.1:3000
// @BasePath /

// @securityDefinitions.apikey AdminSecret
// @in header
// @name X-Admin-Secret

func main() {
	if err := run(); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}

func run() error {
	if err := handlers.InitCustomValidation(); err != nil {
		return fmt.Errorf("initializing custom validation: %w", err)
	}

	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	db, err := db.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	storage, err := storage.NewStorage(cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3BaseURL)
	if err != nil {
		return fmt.Errorf("new storage: %w", err)
	}

	h := handlers.NewHandler(cfg, db, storage)
	limiter := middlewares.NewIPRateLimiter(cfg.RateLimitPerMinute, cfg.RateLimitBurst, time.Hour)

	r, err := newRouter(cfg, h, limiter)
	if err != nil {
		return err
	}

	if err := r.Run(":" + cfg.Port); err != nil {
		return fmt.Errorf("running gin router: %w", err)
	}

	return nil
}

func newRouter(
	cfg *config.Config,
	h *handlers.Handler,
	limiter *middlewares.IPRateLimiter,
) (*gin.Engine, error) {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("setting trusted proxies: %w", err)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.CORSOrigin},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

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
