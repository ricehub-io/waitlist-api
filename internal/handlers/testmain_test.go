package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/ricehub-io/waitlist-api/internal/config"
	"github.com/ricehub-io/waitlist-api/internal/db"
	"github.com/ricehub-io/waitlist-api/internal/middlewares"
)

var (
	testDB     *db.Database
	testServer *httptest.Server
)

type mockStorer struct{}

func (m *mockStorer) UploadFile(_ context.Context, _, _ string, _ io.Reader, _ string) error {
	return nil
}

var testCfg = &config.Config{
	S3BaseURL:     "http://localhost",
	S3MediaBucket: "test-bucket",
	CORSOrigin:    "http://localhost",
	AdminSecret:   "test-admin-secret",
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()

	if err := InitCustomValidation(); err != nil {
		panic("init custom validation: " + err.Error())
	}

	ctr, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("ricehub_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}
	defer func() { _ = ctr.Terminate(ctx) }()

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}

	testDB, err = db.NewDatabase(dsn)
	if err != nil {
		panic("new database: " + err.Error())
	}
	defer testDB.Close()

	if err := applyMigrations(ctx, testDB, "../../migrations"); err != nil {
		panic("apply migrations: " + err.Error())
	}

	h := NewHandler(zap.NewNop(), testCfg, testDB, &mockStorer{})
	limiter := middlewares.NewIPRateLimiter(100, 100, time.Hour)
	r := buildTestRouter(testCfg, h, limiter)
	testServer = httptest.NewServer(r)
	defer testServer.Close()

	return m.Run()
}

func buildTestRouter(
	cfg *config.Config,
	h *Handler,
	limiter *middlewares.IPRateLimiter,
) *gin.Engine {
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{cfg.CORSOrigin},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

	rl := middlewares.RateLimitMiddleware(limiter)

	wr := r.Group("/waitlist")
	wr.GET("", h.GetWaitlistEmailCount)
	wr.POST("", rl, h.CreateWaitlistEmail)

	fr := r.Group("/founders")
	fr.GET("", h.GetFoundingCreatorStats)
	fr.POST("", rl, h.CreateFoundingCreator)

	rr := r.Group("/rices")
	rr.GET("", h.GetPreviewRices)
	rr.POST("", middlewares.AdminMiddleware(cfg.AdminSecret), h.CreatePreviewRice)

	return r
}

func applyMigrations(ctx context.Context, database *db.Database, dir string) error {
	entries, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(entries)
	for _, path := range entries {
		sql, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := database.Pool().Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", path, err)
		}
	}
	return nil
}

func resetDB(t *testing.T) {
	t.Helper()
	_, err := testDB.Pool().Exec(
		context.Background(),
		`TRUNCATE waitlist_emails, founder_applicants, preview_rices;
		UPDATE settings SET slots_total = 10, slots_taken = 0 WHERE id = 1;`,
	)
	require.NoError(t, err)
}

func seedWaitlist(t *testing.T, emails ...string) {
	t.Helper()
	for _, email := range emails {
		_, err := testDB.Pool().Exec(
			context.Background(),
			"INSERT INTO waitlist_emails (email) VALUES ($1)", email,
		)
		require.NoError(t, err)
	}
}

func seedPreviewRice(t *testing.T, title, thumbnailPath string, price *float64) {
	t.Helper()
	_, err := testDB.Pool().Exec(
		context.Background(),
		"INSERT INTO preview_rices (title, thumbnail_path, price) VALUES ($1, $2, $3)",
		title, thumbnailPath, price,
	)
	require.NoError(t, err)
}

func seedFounder(t *testing.T, username, email, dotfilesURL string) {
	t.Helper()
	_, err := testDB.Pool().Exec(
		context.Background(),
		"INSERT INTO founder_applicants (username, email, dotfiles_url) VALUES ($1, $2, $3)",
		username, email, dotfilesURL,
	)
	require.NoError(t, err)
}
