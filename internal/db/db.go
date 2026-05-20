package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ricehub-io/waitlist-api/internal/models"
)

type Database struct {
	pool *pgxpool.Pool
}

// NewDatabase creates a new database pool for given connection URL.
func NewDatabase(url string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}

	var two uint
	err = pool.QueryRow(ctx, "SELECT 2").Scan(&two)
	if err != nil {
		return nil, fmt.Errorf("test query scan: %w", err)
	}
	if two != 2 {
		return nil, fmt.Errorf("invalid test query result: got %d, expected 2", two)
	}

	return &Database{pool}, nil
}

// Close closes all database connections and internal pool. (Blocking)
func (db *Database) Close() {
	db.pool.Close()
}

func (db *Database) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *Database) InsertWaitlistEmail(ctx context.Context, email string) error {
	const query = "INSERT INTO waitlist_emails (email) VALUES ($1)"
	_, err := db.pool.Exec(ctx, query, email)
	return err
}

func (db *Database) InsertFoundingApplicant(
	ctx context.Context,
	username, email, dotfilesURL string,
) error {
	const query = `
	INSERT INTO founder_applicants (username, email, dotfiles_url)
	VALUES ($1, $2, $3)
	`
	_, err := db.pool.Exec(ctx, query, username, email, dotfilesURL)
	return err
}

func (db *Database) FetchSlotStats(ctx context.Context) (models.SlotStats, error) {
	const query = "SELECT slots_total, slots_taken FROM settings LIMIT 1"
	return rowToStruct[models.SlotStats](ctx, db.pool, query)
}

func (db *Database) FetchPreviewRices(ctx context.Context) (models.PreviewRices, error) {
	const query = `
	SELECT id, title, price, thumbnail_path, download_count, star_count, tags, created_at
	FROM preview_rices
	ORDER BY created_at DESC
	`
	return rowsToStruct[models.PreviewRice](ctx, db.pool, query)
}

func (db *Database) InsertPreviewRice(
	ctx context.Context,
	title string,
	price *float64,
	thumbnailPath string,
	starCount, downloadCount int,
	tags []string,
) error {
	const query = `
	INSERT INTO preview_rices (title, price, thumbnail_path, star_count, download_count, tags)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.pool.Exec(ctx, query, title, price, thumbnailPath, starCount, downloadCount, tags)
	return err
}

func (db *Database) WaitlistEmailCount(ctx context.Context) (count int, err error) {
	const query = "SELECT COUNT(*) FROM waitlist_emails"
	err = db.Pool().QueryRow(ctx, query).Scan(&count)
	return count, err
}

// -- HELPERS --
type dbExecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// rowToStruct executes given query using the provided database executor,
// scanning the result into a struct.
// Returns a wrapped error if either query or scanning failed.
func rowToStruct[T any](
	ctx context.Context,
	exec dbExecutor,
	query string,
	args ...any,
) (zero T, err error) {
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return zero, fmt.Errorf("query: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, fmt.Errorf("collect one row: %w", err)
	}

	return row, nil
}

// rowToStructs executes given query using the provided database executor,
// scanning the result into a struct slice.
// Returns a wrapped error if either query or scanning failed.
func rowsToStruct[T any](
	ctx context.Context,
	exec dbExecutor,
	query string,
	args ...any,
) (zero []T, err error) {
	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return zero, fmt.Errorf("query: %w", err)
	}

	cRows, err := pgx.CollectRows(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, fmt.Errorf("collect rows: %w", err)
	}

	return cRows, nil
}
