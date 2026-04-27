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
		return fmt.Errorf("NewConfig: %w", err)
	}

	db, err := NewDatabase(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("NewDatabase: %w", err)
	}
	defer db.Close()

	h := NewHandler(db)

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.POST("/waitlist", h.CreateWaitlistEmail)

	fr := r.Group("/founders")
	fr.GET("/founders", h.GetFoundingCreatorStats)
	fr.POST("/founders", h.CreateWaitlistEmail)

	return r.Run(":" + cfg.Port)
}
