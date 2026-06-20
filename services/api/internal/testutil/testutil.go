package testutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"homelink-monitor/services/api/internal/db"
	"homelink-monitor/services/api/internal/store"
)

func DB(t *testing.T) (*sql.DB, *store.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	database, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database, store.New(database)
}
