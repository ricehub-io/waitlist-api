package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	return &Database{pool}, nil
}

// Close closes all database connections and internal pool. (Blocking)
func (db *Database) Close() {
	db.pool.Close()
}

// InsertWaitlistEmail inserts a new row with given email to waitlist_emails table.
func (db *Database) InsertWaitlistEmail(ctx context.Context, email string) error {
	const query = "INSERT INTO waitlist_emails (email) VALUES ($1)"
	_, err := db.pool.Exec(ctx, query, email)
	return fmt.Errorf("insert waitlist email: %w", err)
}

// InsertFoundingApplicant inserts a new row with given values to founder_applicants table.
func (db *Database) InsertFoundingApplicant(
	ctx context.Context,
	username, email, dotfilesURL string,
) error {
	const query = `
	INSERT INTO founder_applicants (username, email, dotfiles_url)
	VALUES ($1, $2, $3)
	`
	_, err := db.pool.Exec(ctx, query, username, email, dotfilesURL)
	if err != nil {
		return fmt.Errorf("insert founding applicant: %w", err)
	}
	return nil
}

func (db *Database) FetchSlotStats(ctx context.Context) (SlotStats, error) {
	const query = "SELECT slots_total, slots_taken FROM settings LIMIT 1"
	s, err := rowToStruct[SlotStats](ctx, db.pool, query)
	if err != nil {
		return s, fmt.Errorf("fetch slot stats: %w", err)
	}
	return s, nil
}

// -- HELPERS --
type DatabaseExecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// rowToStruct executes given query using the provided
// database executor (pool/tx), scanning the result into a struct.
// Returns a wrapped error if either query or scanning failed.
func rowToStruct[T any](
	ctx context.Context,
	exec DatabaseExecutor,
	query string,
	args ...any,
) (T, error) {
	var zero T

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return zero, fmt.Errorf("db executor query: %w", err)
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
	if err != nil {
		return zero, fmt.Errorf("pgx collect one row: %w", err)
	}

	return row, nil
}
