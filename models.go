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

// -- HTTP REQUESTS/RESPONSES --
type CreateWaitlistEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}
