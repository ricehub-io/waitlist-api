package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *Database
}

// NewHandler instantiates HTTP handler with given DB wrapper instance.
func NewHandler(db *Database) *Handler {
	return &Handler{db}
}

func (h *Handler) CreateWaitlistEmail(c *gin.Context) {

}

func (h *Handler) GetFoundingCreatorStats(c *gin.Context) {
	// TODO: read it from somewhere
	const (
		total     = 10
		taken     = 3
		available = total - taken
	)
	c.JSON(http.StatusOK, gin.H{"total": total, "taken": taken, "available": available})
}

func (h *Handler) CreateFoundingCreator(c *gin.Context) {

}
