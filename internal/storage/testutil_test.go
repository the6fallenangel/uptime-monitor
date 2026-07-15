package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStorage(t *testing.T) *PostgresStorage {
	t.Helper()

	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		baseURL = "postgres://postgres:password@localhost:5432/uptime_monitor?sslmode=disable"
	}

	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

	adminPool, err := pgxpool.New(ctx, baseURL)
	if err != nil {
		t.Fatalf("connecting to postgres: %v", err)
	}

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA %q`, schema)); err != nil {
		adminPool.Close()
		t.Fatalf("creating test schema: %v", err)
	}

	t.Cleanup(func() {
		adminPool.Exec(ctx, fmt.Sprintf(`DROP SCHEMA %q CASCADE`, schema))
		adminPool.Close()
	})

	connString := fmt.Sprintf("%s&search_path=%s", baseURL, schema)
	store, err := NewPostgresStorage(ctx, connString)
	if err != nil {
		t.Fatalf("creating test storage: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	return store
}
