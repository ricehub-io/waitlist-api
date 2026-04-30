package main

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitlistEmailCount_Empty(t *testing.T) {
	resetDB(t)
	count, err := testDB.WaitlistEmailCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int(0), count)
}

func TestWaitlistEmailCount_WithRows(t *testing.T) {
	resetDB(t)
	seedWaitlist(t, "a@x.com", "b@x.com")
	count, err := testDB.WaitlistEmailCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int(2), count)
}

func TestInsertWaitlistEmail_Success(t *testing.T) {
	resetDB(t)
	err := testDB.InsertWaitlistEmail(context.Background(), "new@x.com")
	require.NoError(t, err)
}

func TestInsertWaitlistEmail_DuplicateReturnsUniqueViolation(t *testing.T) {
	resetDB(t)
	seedWaitlist(t, "dup@x.com")
	err := testDB.InsertWaitlistEmail(context.Background(), "dup@x.com")
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr), "expected *pgconn.PgError, got %T: %v", err, err)
	assert.Equal(t, pgerrcode.UniqueViolation, pgErr.Code)
}

func TestInsertFoundingApplicant_Success(t *testing.T) {
	resetDB(t)
	err := testDB.InsertFoundingApplicant(context.Background(), "alice", "alice@x.com", "https://x.com")
	require.NoError(t, err)
}

func TestInsertFoundingApplicant_DuplicateUsernameConstraintName(t *testing.T) {
	resetDB(t)
	require.NoError(t, testDB.InsertFoundingApplicant(context.Background(), "alice", "a@x.com", "https://x.com"))

	err := testDB.InsertFoundingApplicant(context.Background(), "alice", "b@y.com", "https://x.com")
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, pgerrcode.UniqueViolation, pgErr.Code)
	assert.Equal(t, "founder_applicants_username_key", pgErr.ConstraintName,
		"constraint name must match the value checked in CreateFoundingCreator handler")
}

func TestInsertFoundingApplicant_DuplicateEmailConstraintName(t *testing.T) {
	resetDB(t)
	require.NoError(t, testDB.InsertFoundingApplicant(context.Background(), "alice", "a@x.com", "https://x.com"))

	err := testDB.InsertFoundingApplicant(context.Background(), "bob", "a@x.com", "https://x.com")
	require.Error(t, err)

	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, pgerrcode.UniqueViolation, pgErr.Code)
	assert.Equal(t, "founder_applicants_email_key", pgErr.ConstraintName,
		"constraint name must match the value checked in CreateFoundingCreator handler")
}

func TestFetchSlotStats(t *testing.T) {
	resetDB(t)
	stats, err := testDB.FetchSlotStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10, stats.SlotsTotal)
	assert.Equal(t, 0, stats.SlotsTaken)
}

func TestFetchSlotStats_AfterUpdate(t *testing.T) {
	resetDB(t)
	_, err := testDB.pool.Exec(context.Background(),
		"UPDATE settings SET slots_total = 20, slots_taken = 5 WHERE id = 1")
	require.NoError(t, err)

	stats, err := testDB.FetchSlotStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 20, stats.SlotsTotal)
	assert.Equal(t, 5, stats.SlotsTaken)
}
