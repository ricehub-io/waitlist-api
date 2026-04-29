package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type Handler struct {
	db *Database
}

// NewHandler instantiates HTTP handler with given DB wrapper instance.
func NewHandler(db *Database) *Handler {
	return &Handler{db}
}

// CreateWaitlistEmail godoc
// @Summary Add email to waitlist
// @Description Adds a new email address to the waitlist
// @Tags waitlist
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "Email address"
// @Success 201 "Created"
// @Failure 400 {object} APIError "Bad request"
// @Failure 409 {object} APIError "Email already on waitlist"
// @Failure 500 {object} APIError "Internal server error"
// @Router /waitlist [post]
func (h *Handler) CreateWaitlistEmail(c *gin.Context) {
	var body CreateWaitlistEmailRequest
	if err := c.ShouldBind(&body); err != nil {
		sendErrors(c, http.StatusBadRequest, "could not parse request body", err.Error())
		return
	}

	if err := h.db.InsertWaitlistEmail(c.Request.Context(), body.Email); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			sendErrors(c, http.StatusConflict, "this email address is already on waitlist!")
			return
		}

		log.Printf("could not insert waitlist email: %v", err)
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusCreated)
}

// GetFoundingCreatorStats godoc
// @Summary Get founding creator slot statistics
// @Description Returns total, taken, and available founding creator slots
// @Tags founders
// @Produce json
// @Success 200 {object} GetFoundingCreatorStatsResponse
// @Failure 500 {object} APIError "Internal server error"
// @Router /founders [get]
func (h *Handler) GetFoundingCreatorStats(c *gin.Context) {
	stats, err := h.db.FetchSlotStats(c.Request.Context())
	if err != nil {
		log.Printf("could not fetch slot stats: %v", err)
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	dto := GetFoundingCreatorStatsResponse{
		SlotsTotal:     stats.SlotsTotal,
		SlotsTaken:     stats.SlotsTaken,
		SlotsAvailable: stats.SlotsTotal - stats.SlotsTaken,
	}

	c.JSON(http.StatusOK, dto)
}

// CreateFoundingCreator godoc
// @Summary Register as founding creator
// @Description Submit application to become a founding creator
// @Tags founders
// @Accept x-www-form-urlencoded
// @Produce json
// @Param username formData string true "Username (4-14 alphanumeric characters)"
// @Param email formData string true "Email address"
// @Param dotfilesUrl formData string true "URL to dotfiles repository or r/unixporn post"
// @Success 201 "Created"
// @Failure 400 {object} APIError "Bad request"
// @Failure 409 {object} APIError "Username or email already taken"
// @Failure 500 {object} APIError "Internal server error"
// @Router /founders [post]
func (h *Handler) CreateFoundingCreator(c *gin.Context) {
	var body CreateFoundingCreatorRequest
	if err := c.ShouldBind(&body); err != nil {
		sendErrors(c, http.StatusBadRequest, "could not parse request body", err.Error())
		return
	}

	if err := h.db.InsertFoundingApplicant(
		c.Request.Context(),
		body.Username, body.Email, body.DotfilesURL,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			// TODO: be more precise - let the user know whether its username or email
			sendErrors(c, http.StatusConflict, "this username or email is already taken!")
			return
		}

		log.Printf("could not insert founding applicant: %v", err)
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusCreated)
}

// -- HELPERS --
type APIError struct {
	Errors []string
}

// sendErrors sends HTTP response with given status code and errors.
// Does not abort the connection, you have to return/abort manually.
func sendErrors(c *gin.Context, status int, errs ...string) {
	c.JSON(status, gin.H{"errors": errs})
}
