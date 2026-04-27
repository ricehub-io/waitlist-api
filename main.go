package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

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
	fr.POST("", h.CreateWaitlistEmail)

	return fmt.Errorf("gin run: %w", r.Run(":"+cfg.Port))
}
