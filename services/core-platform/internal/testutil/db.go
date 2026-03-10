package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func RequireDBConfig(t *testing.T) (databaseURL string, migrationsDir string) {
	t.Helper()
	databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	migrationsDir = strings.TrimSpace(os.Getenv("TEST_MIGRATIONS_DIR"))
	if databaseURL == "" || migrationsDir == "" {
		t.Skip("integration env is not set: TEST_DATABASE_URL and TEST_MIGRATIONS_DIR are required")
	}
	return databaseURL, migrationsDir
}

func OpenDB(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func ApplyMigrations(t *testing.T, db *sql.DB, migrationsDir string) {
	t.Helper()
	pattern := filepath.Join(migrationsDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no migration files found in %s", migrationsDir)
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}
}

func ResetData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		TRUNCATE TABLE
		  node_heartbeats,
		  audit_logs,
		  traffic_usage_hourly,
		  user_policy_overrides,
		  tenant_policies,
		  access_keys,
		  devices,
		  nodes,
		  admins,
		  users,
		  tenants
		RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		t.Fatalf("reset data: %v", err)
	}
}

func UniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
