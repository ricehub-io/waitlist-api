package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type multipartField struct {
	name  string
	value string
}

func postMultipart(
	t *testing.T,
	path string,
	fields []multipartField,
	fileName, fileContentType string,
	fileContent []byte,
	adminSecret string,
) (int, map[string]any) {
	t.Helper()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for _, f := range fields {
		require.NoError(t, w.WriteField(f.name, f.value))
	}

	if fileContent != nil {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="thumbnail"; filename="%s"`, fileName))
		h.Set("Content-Type", fileContentType)
		part, err := w.CreatePart(h)
		require.NoError(t, err)
		_, err = part.Write(fileContent)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, testServer.URL+path, &b)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Admin-Secret", adminSecret)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer closeSilent(resp.Body)

	body := map[string]any{}
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	return resp.StatusCode, body
}

func getJSON(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(testServer.URL + path)
	require.NoError(t, err)
	defer closeSilent(resp.Body)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return resp.StatusCode, body
}

func postForm(t *testing.T, path string, values url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := http.PostForm(testServer.URL+path, values)
	require.NoError(t, err)
	defer closeSilent(resp.Body)

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

// -- GET /rices --
func TestGetPreviewRices_Empty(t *testing.T) {
	resetDB(t)
	resp, err := http.Get(testServer.URL + "/rices")
	require.NoError(t, err)
	defer closeSilent(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Empty(t, body)
}

func TestGetPreviewRices_WithRows(t *testing.T) {
	resetDB(t)
	price := 9.99
	seedPreviewRice(t, "Arch Rice", "arch.png", nil)
	seedPreviewRice(t, "Paid Rice", "paid.png", &price)

	resp, err := http.Get(testServer.URL + "/rices")
	require.NoError(t, err)
	defer closeSilent(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body, 2)

	titlesSet := map[string]bool{}
	for _, r := range body {
		titlesSet[r["title"].(string)] = true
		assert.Contains(t, r, "id")
		assert.Contains(t, r, "thumbnailUrl")
		assert.Contains(t, r, "downloadCount")
		assert.Contains(t, r, "starCount")
		assert.Contains(t, r, "tags")
	}
	assert.True(t, titlesSet["Arch Rice"])
	assert.True(t, titlesSet["Paid Rice"])
}

func TestGetPreviewRices_PriceNullableField(t *testing.T) {
	resetDB(t)
	price := 2.50
	seedPreviewRice(t, "Free Rice", "free.png", nil)
	seedPreviewRice(t, "Paid Rice", "paid.png", &price)

	resp, err := http.Get(testServer.URL + "/rices")
	require.NoError(t, err)
	defer closeSilent(resp.Body)

	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body, 2)

	for _, r := range body {
		switch r["title"] {
		case "Free Rice":
			assert.Nil(t, r["price"])
		case "Paid Rice":
			assert.InDelta(t, 2.50, r["price"].(float64), 0.001)
		}
	}
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

// -- POST /rices --
func riceFields(title, starCount, downloadCount string, tags []string) []multipartField {
	mpf := make([]multipartField, 0, 3+len(tags))

	mpf = append(mpf, multipartField{name: "title", value: title})
	mpf = append(mpf, multipartField{name: "starCount", value: starCount})
	mpf = append(mpf, multipartField{name: "downloadCount", value: downloadCount})

	for _, tag := range tags {
		mpf = append(mpf, multipartField{name: "tags", value: tag})
	}

	return mpf
}

func TestCreatePreviewRice_Unauthorized(t *testing.T) {
	resetDB(t)
	fields := riceFields("Im Unathorized", "5", "7", []string{"test"})
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), "")
	assert.Equal(t, http.StatusUnauthorized, status)
	assertErrorContains(t, body, "unauthorized")
}

func TestCreatePreviewRice_HappyPath(t *testing.T) {
	resetDB(t)
	fields := riceFields("My Rice", "10", "5", []string{"test"})
	status, _ := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusCreated, status)

	var count int
	err := testDB.pool.QueryRow(
		context.Background(),
		"SELECT COUNT(*) FROM preview_rices WHERE title = 'My Rice'",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestCreatePreviewRice_HappyPathWithPriceAndTags(t *testing.T) {
	resetDB(t)
	fields := append(riceFields("Paid Rice", "3", "7", []string{"test"}),
		multipartField{name: "price", value: "9.99"},
		multipartField{name: "tags", value: "dark"},
		multipartField{name: "tags", value: "minimal"},
	)
	status, _ := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusCreated, status)

	var price float64
	var tags []string
	err := testDB.pool.QueryRow(
		context.Background(),
		"SELECT price, tags FROM preview_rices WHERE title = 'Paid Rice'",
	).Scan(&price, &tags)
	require.NoError(t, err)
	assert.InDelta(t, 9.99, price, 0.001)
	assert.ElementsMatch(t, []string{"test", "dark", "minimal"}, tags)
}

func TestCreatePreviewRice_MissingTitle(t *testing.T) {
	resetDB(t)
	fields := []multipartField{
		{name: "starCount", value: "0"},
		{name: "downloadCount", value: "0"},
	}
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "Title")
}

func TestCreatePreviewRice_InvalidPrice(t *testing.T) {
	resetDB(t)
	fields := append(riceFields("Bad Price Rice", "0", "0", []string{"test"}),
		multipartField{name: "price", value: "not-a-price"},
	)
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "price")
}

func TestCreatePreviewRice_ZeroPrice(t *testing.T) {
	resetDB(t)
	fields := append(riceFields("Zero Price Rice", "0", "0", []string{"test"}),
		multipartField{name: "price", value: "0"},
	)
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "Price")
}

func TestCreatePreviewRice_InvalidStarCount(t *testing.T) {
	resetDB(t)
	fields := []multipartField{
		{name: "title", value: "Rice"},
		{name: "starCount", value: "abc"},
		{name: "downloadCount", value: "0"},
	}
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "abc")
}

func TestCreatePreviewRice_NegativeDownloadCount(t *testing.T) {
	resetDB(t)
	fields := []multipartField{
		{name: "title", value: "Rice"},
		{name: "starCount", value: "0"},
		{name: "downloadCount", value: "-1"},
	}
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "DownloadCount")
}

func TestCreatePreviewRice_MissingThumbnail(t *testing.T) {
	resetDB(t)
	fields := riceFields("No Thumb Rice", "0", "0", []string{"test"})
	status, body := postMultipart(t, "/rices", fields, "", "", nil, testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "Thumbnail")
}

func TestCreatePreviewRice_MissingTags(t *testing.T) {
	resetDB(t)
	fields := []multipartField{
		{name: "title", value: "No Tags Rice"},
		{name: "starCount", value: "0"},
		{name: "downloadCount", value: "0"},
	}
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "Tags")
}

func TestCreatePreviewRice_InvalidThumbnailType(t *testing.T) {
	resetDB(t)
	fields := riceFields("Bad Type Rice", "0", "0", []string{"test"})
	status, body := postMultipart(t, "/rices", fields, "thumb.gif", "image/gif", []byte("GIF89a"), testCfg.AdminSecret)
	assert.Equal(t, http.StatusBadRequest, status)
	assertErrorContains(t, body, "thumbnail must be")
}

func TestCreatePreviewRice_DuplicateTitle(t *testing.T) {
	resetDB(t)
	seedPreviewRice(t, "Existing Rice", "existing.png", nil)
	fields := riceFields("Existing Rice", "0", "0", []string{"test"})
	status, body := postMultipart(t, "/rices", fields, "thumb.png", "image/png", validPNG(), testCfg.AdminSecret)
	assert.Equal(t, http.StatusConflict, status)
	assertErrorContains(t, body, "already exists")
}
