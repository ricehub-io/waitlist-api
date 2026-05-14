package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/ricehub-io/waitlist-api/docs"
)

// @title RiceHub Waitlist API
// @version 1.3.0
// @description API for RiceHub waitlist frontend.

// @host 127.0.0.1:3000
// @BasePath /

// @securityDefinitions.apikey AdminSecret
// @in header
// @name X-Admin-Secret

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := initCustomValidation(); err != nil {
		return fmt.Errorf("init custom validation: %w", err)
	}

	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	db, err := NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	storage, err := NewStorage(cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3BaseURL)
	if err != nil {
		return fmt.Errorf("new storage: %w", err)
	}

	h := NewHandler(cfg, db, storage)

	r, err := newRouter(cfg, h)
	if err != nil {
		return err
	}

	if err := r.Run(":" + cfg.Port); err != nil {
		return fmt.Errorf("gin run: %w", err)
	}

	return nil
}

func newRouter(cfg *Config, h *Handler) (*gin.Engine, error) {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("gin set trusted proxies: %w", err)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.CORSOrigin},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

	wr := r.Group("/waitlist")
	wr.GET("", h.GetWaitlistEmailCount)
	wr.POST("", h.CreateWaitlistEmail)

	fr := r.Group("/founders")
	fr.GET("", h.GetFoundingCreatorStats)
	fr.POST("", h.CreateFoundingCreator)

	rr := r.Group("/rices")
	rr.GET("", h.GetPreviewRices)
	rr.POST("", adminMiddleware(cfg.AdminSecret), h.CreatePreviewRice)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
	))

	return r, nil
}

func initCustomValidation() error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("ricetitle", func(fl validator.FieldLevel) bool {
			re := regexp.MustCompile(`^[a-zA-Z0-9 '_-]+$`)
			return re.MatchString(fl.Field().String())
		}); err != nil {
			return fmt.Errorf("ricetitle validation: %w", err)
		}
	}

	return nil
}

func adminMiddleware(adminSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := c.GetHeader("X-Admin-Secret")
		if secret == "" || secret != adminSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"unauthorized"}})
			c.Abort()
			return
		}

		c.Next()
	}
}
