package main

import (
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

// -- DATABASE MODELS --
type PreviewRice struct {
	ID            uuid.UUID
	Title         string
	Price         *float64
	ThumbnailPath string
	DownloadCount int
	StarCount     int
	Tags          []string
	CreatedAt     time.Time
}
type PreviewRices []PreviewRice

type WaitlistEmail struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}

type FounderApplicant struct {
	ID          uuid.UUID
	Username    string
	Email       string
	DotfilesURL string
	CreatedAt   time.Time
}

type SlotStats struct {
	SlotsTotal int
	SlotsTaken int
}

// -- HTTP REQUESTS/RESPONSES --
type GetPreviewRicesResponse struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Price         *float64  `json:"price,omitempty"`
	ThumbnailURL  string    `json:"thumbnailUrl"`
	DownloadCount int       `json:"downloadCount"`
	StarCount     int       `json:"starCount"`
	Tags          []string  `json:"tags"`
}

// ToResponse converts given preview rice models to http-friendly response.
// storageBaseURL must not end with a leading slash.
func (rices PreviewRices) ToResponse(storageBaseURL string) []GetPreviewRicesResponse {
	resp := make([]GetPreviewRicesResponse, len(rices))

	for i, r := range rices {
		resp[i] = GetPreviewRicesResponse{
			ID:            r.ID,
			Title:         r.Title,
			Price:         r.Price,
			ThumbnailURL:  storageBaseURL + "/" + r.ThumbnailPath,
			DownloadCount: r.DownloadCount,
			StarCount:     r.StarCount,
			Tags:          r.Tags,
		}
	}

	return resp
}

type GetWaitlistEmailCountResponse struct {
	Count int `json:"count"`
}

type CreateWaitlistEmailRequest struct {
	Email   string `form:"email" binding:"required,email"`
	Website string `form:"website"`
}

type GetFoundingCreatorStatsResponse struct {
	SlotsTotal     int `json:"slotsTotal"`
	SlotsTaken     int `json:"slotsTaken"`
	SlotsAvailable int `json:"slotsAvailable"`
}

type CreateFoundingCreatorRequest struct {
	Username    string `form:"username" binding:"required,min=4,max=14,alphanum"`
	Email       string `form:"email" binding:"required,email"`
	DotfilesURL string `form:"dotfilesUrl" binding:"required,url"`
	Website     string `form:"website"`
}

type CreatePreviewRiceRequest struct {
	Title         string                `form:"title" binding:"required,min=4,max=32,ricetitle"`
	Price         *float64              `form:"price" binding:"omitempty,gt=0"`
	StarCount     int                   `form:"starCount" binding:"gte=0"`
	DownloadCount int                   `form:"downloadCount" binding:"gte=0"`
	Tags          []string              `form:"tags" binding:"required"`
	Thumbnail     *multipart.FileHeader `form:"thumbnail" binding:"required"`
}
