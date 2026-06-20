package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"homelink-monitor/services/api/internal/db"
)

func TestMigrateCreatesTables(t *testing.T) {
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	var count int
	err = database.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('settings','speed_tests','latency_checks','dns_checks','outages')`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Fatalf("expected 5 tables, got %d", count)
	}
}
