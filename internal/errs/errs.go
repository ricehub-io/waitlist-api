// Package implementing named error types so sentry can show correct error titles ^.^
package errs

import "fmt"

// -- DB ERROR
type DBError struct {
	Op    string
	Cause error
}

func NewDBError(op string, cause error) *DBError {
	return &DBError{op, cause}
}

func (e *DBError) Error() string {
	return fmt.Sprintf("db: %s: %v", e.Op, e.Cause)
}

func (e *DBError) Unwrap() error {
	return e.Cause
}

// -- DISCORD WEBHOOK ERROR
type DiscordSendWebhookError struct{ Cause error }

func (e *DiscordSendWebhookError) Error() string {
	return fmt.Sprintf("discord send webhook: %v", e.Cause)
}

func (e *DiscordSendWebhookError) Unwrap() error {
	return e.Cause
}

// -- OPEN THUMBNAIL ERROR
type OpenThumbnailError struct{ Cause error }

func (e *OpenThumbnailError) Error() string {
	return fmt.Sprintf("open thumbnail: %v", e.Cause)
}

func (e *OpenThumbnailError) Unwrap() error {
	return e.Cause
}

// -- STORAGE UPLOAD FILE (THUMBNAIL) ERROR
type UploadThumbnailError struct{ Cause error }

func (e *UploadThumbnailError) Error() string {
	return fmt.Sprintf("storage upload file: thumbnail: %v", e.Cause)
}

func (e *UploadThumbnailError) Unwrap() error {
	return e.Cause
}
