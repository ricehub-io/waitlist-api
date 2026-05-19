package handlers

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // init jpeg support
	_ "image/png"  // init png support
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"

	"github.com/ricehub-io/waitlist-api/internal/config"
	"github.com/ricehub-io/waitlist-api/internal/db"
	"github.com/ricehub-io/waitlist-api/internal/discord"
	"github.com/ricehub-io/waitlist-api/internal/models"
	"github.com/ricehub-io/waitlist-api/internal/storage"
)

var allowedImgExt = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
}

type Handler struct {
	logger  *zap.Logger
	cfg     *config.Config
	db      *db.Database
	storage storage.Storer
}

// NewHandler instantiates HTTP handler with given DB wrapper instance.
func NewHandler(
	logger *zap.Logger,
	cfg *config.Config,
	db *db.Database,
	storage storage.Storer,
) *Handler {
	return &Handler{logger, cfg, db, storage}
}

// GetPreviewRices godoc
// @Summary Get all preview rices
// @Description Returns a list of all preview rices ordered by creation date
// @Tags rices
// @Produce json
// @Success 200 {array} models.GetPreviewRicesResponse
// @Failure 500 {object} APIError "Internal server error"
// @Router /rices [get]
func (h *Handler) GetPreviewRices(c *gin.Context) {
	rices, err := h.db.FetchPreviewRices(c.Request.Context())
	if err != nil {
		h.logger.Error("Could not fetch preview rices", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	baseURL := h.cfg.S3BaseURL + "/" + h.cfg.S3MediaBucket
	c.JSON(http.StatusOK, rices.ToResponse(baseURL))
}

// GetWaitlistEmailCount godoc
// @Summary Get amount of users in waitlist
// @Description Returns total count
// @Tags waitlist
// @Produce json
// @Success 200 {object} models.GetWaitlistEmailCountResponse
// @Failure 500 {object} APIError "Internal server error"
// @Router /waitlist [get]
func (h *Handler) GetWaitlistEmailCount(c *gin.Context) {
	count, err := h.db.WaitlistEmailCount(c.Request.Context())
	if err != nil {
		h.logger.Error("Could not fetch waitlist count", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.JSON(http.StatusOK, models.GetWaitlistEmailCountResponse{Count: count})
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
	var body models.CreateWaitlistEmailRequest
	if err := c.ShouldBind(&body); err != nil {
		sendErrors(c, http.StatusBadRequest, "could not parse request body", err.Error())
		return
	}

	if body.Website != "" {
		c.Status(http.StatusCreated)
		return
	}

	if err := h.db.InsertWaitlistEmail(c.Request.Context(), body.Email); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			sendErrors(c, http.StatusConflict, "this email address is already on waitlist!")
			return
		}

		h.logger.Error("Could not insert waitlist email", zap.Error(err))
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
// @Success 200 {object} models.GetFoundingCreatorStatsResponse
// @Failure 500 {object} APIError "Internal server error"
// @Router /founders [get]
func (h *Handler) GetFoundingCreatorStats(c *gin.Context) {
	stats, err := h.db.FetchSlotStats(c.Request.Context())
	if err != nil {
		h.logger.Error("Could not fetch slot stats", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	dto := models.GetFoundingCreatorStatsResponse{
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
	var body models.CreateFoundingCreatorRequest
	if err := c.ShouldBind(&body); err != nil {
		sendErrors(c, http.StatusBadRequest, "could not parse request body", err.Error())
		return
	}

	if body.Website != "" {
		c.Status(http.StatusCreated)
		return
	}

	if err := h.db.InsertFoundingApplicant(
		c.Request.Context(),
		body.Username, body.Email, body.DotfilesURL,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			switch pgErr.ConstraintName {
			case "founder_applicants_username_key":
				sendErrors(c, http.StatusConflict, "this username is already taken!")
			case "founder_applicants_email_key":
				sendErrors(c, http.StatusConflict, "this email is already taken!")
			default:
				sendErrors(c, http.StatusConflict, "this username or email is already taken!")
			}
			return
		}

		h.logger.Error("Could not insert founding applicant", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if h.cfg.DiscordWebhookURL != "" {
		if err := discord.SendWebhook(
			c.Request.Context(),
			h.cfg.DiscordWebhookURL,
			"New founding creator application has been received! @everyone",
		); err != nil {
			h.logger.Error("Failed to send discord webhook", zap.Error(err))
		}
	}

	c.Status(http.StatusCreated)
}

// CreatePreviewRice godoc
// @Summary Create a preview rice
// @Description Uploads thumbnail to S3 and stores rice metadata
// @Tags rices
// @Accept multipart/form-data
// @Produce json
// @Param X-Admin-Secret header string true "Admin secret key"
// @Param title formData string true "Rice title"
// @Param price formData number false "Price (e.g. 15.29, must be > 0)"
// @Param thumbnail formData file true "Thumbnail image (png, jpeg, webp)"
// @Param starCount formData int true "Star count"
// @Param downloadCount formData int true "Download count"
// @Param tags formData []string false "Tags"
// @Success 201 "Created"
// @Failure 400 {object} APIError "Bad request"
// @Failure 401 {object} APIError "Unauthorized"
// @Failure 409 {object} APIError "Title already exists"
// @Failure 500 {object} APIError "Internal server error"
// @Security AdminSecret
// @Router /rices [post]
func (h *Handler) CreatePreviewRice(c *gin.Context) {
	var body models.CreatePreviewRiceRequest
	if err := c.ShouldBind(&body); err != nil {
		sendErrors(c, http.StatusBadRequest, "could not parse request body", err.Error())
		return
	}

	mime, valid, err := validateMimeType(body.Thumbnail, allowedImgExt)
	if err != nil {
		sendErrors(c, http.StatusBadRequest, "could not validate thumbnail", err.Error())
		return
	}
	if !valid {
		sendErrors(c, http.StatusBadRequest, "thumbnail must be png, jpeg, or webp")
		return
	}

	thumbnailKey := fmt.Sprintf("thumbnails/%s.webp", uuid.New())
	f, err := openThumbnail(mime, body.Thumbnail)
	if err != nil {
		h.logger.Error("Could not open thumbnail file", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}
	defer closeSilent(f)

	if err := h.storage.UploadFile(
		c.Request.Context(),
		h.cfg.S3MediaBucket, thumbnailKey, f, "image/webp",
	); err != nil {
		h.logger.Error("Could not upload thumbnail", zap.Error(err))
		sendErrors(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.db.InsertPreviewRice(
		c.Request.Context(),
		body.Title, body.Price, thumbnailKey,
		body.StarCount, body.DownloadCount, body.Tags,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			sendErrors(c, http.StatusConflict, "a rice with this title already exists!")
			return
		}

		h.logger.Error("Could not insert preview rice", zap.Error(err))
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

func validateMimeType(fh *multipart.FileHeader, allowed map[string]struct{}) (string, bool, error) {
	f, err := fh.Open()
	if err != nil {
		return "", false, fmt.Errorf("file header open: %w", err)
	}
	defer closeSilent(f)

	buf := make([]byte, 512)
	if _, err := f.Read(buf); err != nil {
		return "", false, fmt.Errorf("file read: %w", err)
	}

	mimeType := http.DetectContentType(buf)
	_, exists := allowed[mimeType]
	return mimeType, exists, nil
}

func openThumbnail(mime string, fh *multipart.FileHeader) (io.ReadCloser, error) {
	if mime != "image/webp" {
		return imageToWebP(fh)
	}
	return fh.Open()
}

// tempFile wraps *os.File and removes it from disk on Close.
type tempFile struct{ *os.File }

func (t *tempFile) Close() error {
	name := t.Name()
	err := t.File.Close()
	_ = os.Remove(name)
	return err
}

func imageToWebP(fh *multipart.FileHeader) (*tempFile, error) {
	srcFile, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("file header open: %w", err)
	}
	defer closeSilent(srcFile)

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	outFile, err := os.CreateTemp("", "ricehub-thumbnail-*")
	if err != nil {
		return nil, fmt.Errorf("os create temp: %w", err)
	}

	if err := webp.Encode(outFile, img, &webp.Options{
		Lossless: false,
		Quality:  85.0,
	}); err != nil {
		_ = outFile.Close()
		_ = os.Remove(outFile.Name())
		return nil, fmt.Errorf("webp encode: %w", err)
	}

	if _, err := outFile.Seek(0, io.SeekStart); err != nil {
		_ = outFile.Close()
		_ = os.Remove(outFile.Name())
		return nil, fmt.Errorf("seek temp file: %w", err)
	}

	return &tempFile{outFile}, nil
}

func closeSilent(c io.Closer) {
	_ = c.Close()
}
