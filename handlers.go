package main

import (
	"log"
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
	var body CreateWaitlistEmailRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"errors": []string{
				"could not parse request body",
				err.Error(),
			},
		})
		return
	}

	row, err := h.db.InsertWaitlistEmail(c.Request.Context(), body.Email)
	if err != nil {
		log.Printf("could not insert waitlist email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": []string{"internal server error"},
		})
		return
	}
	log.Println(row)

	c.Status(http.StatusCreated)
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
