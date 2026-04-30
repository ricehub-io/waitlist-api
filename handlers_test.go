package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -- HELPERS --
func silentClose(c io.Closer) {
	_ = c.Close()
}

func getJSON(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(testServer.URL + path)
	require.NoError(t, err)
	defer silentClose(resp.Body)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return resp.StatusCode, body
}

func postForm(t *testing.T, path string, values url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := http.PostForm(testServer.URL+path, values)
	require.NoError(t, err)
	defer silentClose(resp.Body)

	body := map[string]any{}
	b, _ := io.ReadAll(resp.Body)
	if len(b) > 0 {
		_ = json.Unmarshal(b, &body)
	}
	return resp.StatusCode, body
}

func assertErrorContains(t *testing.T, body map[string]any, substr string) {
	t.Helper()
	errs, ok := body["errors"].([]any)
	require.True(t, ok, "expected 'errors' array in response body, got: %v", body)
	for _, e := range errs {
		if strings.Contains(fmt.Sprint(e), substr) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got %v", substr, errs)
}

// -- GET /waitlist --
func TestGetWaitlistEmailCount_Empty(t *testing.T) {
	resetDB(t)
	status, body := getJSON(t, "/waitlist")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, float64(0), body["count"])
}

func TestGetWaitlistEmailCount_WithRows(t *testing.T) {
	resetDB(t)
	seedWaitlist(t, "a@x.com", "b@x.com", "c@x.com")
	status, body := getJSON(t, "/waitlist")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, float64(3), body["count"])
}

// -- POST /waitlist --
func TestCreateWaitlistEmail_HappyPath(t *testing.T) {
	resetDB(t)
	status, _ := postForm(t, "/waitlist", url.Values{"email": {"alice@example.com"}})
	assert.Equal(t, http.StatusCreated, status)

	var count int
	err := testDB.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM waitlist_emails WHERE email = 'alice@example.com'",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCreateWaitlistEmail_MissingEmail(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/waitlist", url.Values{})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateWaitlistEmail_InvalidFormat(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/waitlist", url.Values{"email": {"not-an-email"}})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateWaitlistEmail_DuplicateExact(t *testing.T) {
	resetDB(t)
	seedWaitlist(t, "alice@x.com")
	status, body := postForm(t, "/waitlist", url.Values{"email": {"alice@x.com"}})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "already on waitlist")
}

func TestCreateWaitlistEmail_DuplicateCaseInsensitive(t *testing.T) {
	resetDB(t)
	seedWaitlist(t, "Alice@X.com")
	status, body := postForm(t, "/waitlist", url.Values{"email": {"alice@x.com"}})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "already on waitlist")
}

// -- GET /founders --
func TestGetFoundingCreatorStats_Default(t *testing.T) {
	resetDB(t)
	status, body := getJSON(t, "/founders")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, float64(10), body["slotsTotal"])
	assert.Equal(t, float64(0), body["slotsTaken"])
	assert.Equal(t, float64(10), body["slotsAvailable"])
}

func TestGetFoundingCreatorStats_SomeTaken(t *testing.T) {
	resetDB(t)
	_, err := testDB.pool.Exec(
		context.Background(),
		"UPDATE settings SET slots_taken = 3 WHERE id = 1",
	)
	require.NoError(t, err)
	status, body := getJSON(t, "/founders")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, float64(10), body["slotsTotal"])
	assert.Equal(t, float64(3), body["slotsTaken"])
	assert.Equal(t, float64(7), body["slotsAvailable"])
}

func TestGetFoundingCreatorStats_Full(t *testing.T) {
	resetDB(t)
	_, err := testDB.pool.Exec(
		context.Background(),
		"UPDATE settings SET slots_taken = 10 WHERE id = 1",
	)
	require.NoError(t, err)
	status, body := getJSON(t, "/founders")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, float64(0), body["slotsAvailable"])
}

// -- POST /founders --
func TestCreateFoundingCreator_HappyPath(t *testing.T) {
	resetDB(t)
	status, _ := postForm(t, "/founders", url.Values{
		"username":    {"alice"},
		"email":       {"alice@x.com"},
		"dotfilesUrl": {"https://github.com/alice/dotfiles"},
	})
	assert.Equal(t, http.StatusCreated, status)

	var count int
	err := testDB.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM founder_applicants WHERE username = 'alice'",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCreateFoundingCreator_MissingUsername(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_UsernameTooShort(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"abc"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_UsernameMinBoundary(t *testing.T) {
	resetDB(t)
	status, _ := postForm(t, "/founders", url.Values{
		"username":    {"abcd"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusCreated, status)
}

func TestCreateFoundingCreator_UsernameMaxBoundary(t *testing.T) {
	resetDB(t)
	status, _ := postForm(t, "/founders", url.Values{
		"username":    {strings.Repeat("a", 14)},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusCreated, status)
}

func TestCreateFoundingCreator_UsernameTooLong(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {strings.Repeat("a", 15)},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_UsernameNonAlphanumDash(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"a-b"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_UsernameNonAlphanumSpace(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"al ce"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_InvalidEmail(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"alice"},
		"email":       {"not-an-email"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_InvalidURL(t *testing.T) {
	resetDB(t)
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"alice"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"not-a-url"},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "errors")
}

func TestCreateFoundingCreator_DuplicateUsernameExact(t *testing.T) {
	resetDB(t)
	seedFounder(t, "alice", "a@x.com", "https://x.com")
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"alice"},
		"email":       {"b@y.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "username is already taken")
}

func TestCreateFoundingCreator_DuplicateUsernameCaseInsensitive(t *testing.T) {
	resetDB(t)
	seedFounder(t, "Alice", "a@x.com", "https://x.com")
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"alice"},
		"email":       {"b@y.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "username is already taken")
}

func TestCreateFoundingCreator_DuplicateEmailExact(t *testing.T) {
	resetDB(t)
	seedFounder(t, "alice", "a@x.com", "https://x.com")
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"bobby"},
		"email":       {"a@x.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "email is already taken")
}

func TestCreateFoundingCreator_DuplicateEmailCaseInsensitive(t *testing.T) {
	resetDB(t)
	seedFounder(t, "alice", "a@x.com", "https://x.com")
	status, body := postForm(t, "/founders", url.Values{
		"username":    {"bobby"},
		"email":       {"A@X.com"},
		"dotfilesUrl": {"https://x.com"},
	})
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "email is already taken")
}
