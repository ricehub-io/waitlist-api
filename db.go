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
func (db *Database) InsertWaitlistEmail(ctx context.Context, email string) (WaitlistEmail, error) {
	const query = "INSERT INTO waitlist_emails (email) VALUES ($1) RETURNING *"

	row, err := rowToStruct[WaitlistEmail](ctx, db.pool, query, email)
	if err != nil {
		return WaitlistEmail{}, fmt.Errorf("insert waitlist email: %w", err)
	}

	return row, nil
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
