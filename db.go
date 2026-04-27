package main

import (
	"context"
	"time"

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
