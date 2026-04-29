package main

import (
	"time"

	"github.com/google/uuid"
)

// -- DATABASE MODELS --
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
type CreateWaitlistEmailRequest struct {
	Email string `form:"email" binding:"required,email"`
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
}
