package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/ricehub/waitlist-api/docs"
)

// @title RiceHub Waitlist API
// @version 1.0
// @description API for RiceHub waitlist frontend.

// @host 127.0.0.1:3000
// @BasePath /

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := NewConfig()
	if err != nil {
		return fmt.Errorf("new config: %w", err)
	}

	db, err := NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("new database: %w", err)
	}
	defer db.Close()

	h := NewHandler(db)

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("gin set trusted proxies: %w", err)
	}

	r.POST("/waitlist", h.CreateWaitlistEmail)

	fr := r.Group("/founders")
	fr.GET("", h.GetFoundingCreatorStats)
	fr.POST("", h.CreateFoundingCreator)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
	))

	if err := r.Run(":" + cfg.Port); err != nil {
		return fmt.Errorf("gin run: %w", err)
	}

	return nil
}
