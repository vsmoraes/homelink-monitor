package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);`,
	`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
	`CREATE TABLE IF NOT EXISTS speed_tests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at TEXT NOT NULL,
		finished_at TEXT,
		download_mbps REAL,
		upload_mbps REAL,
		ping_ms REAL,
		jitter_ms REAL,
		server_name TEXT,
		server_location TEXT,
		success INTEGER NOT NULL,
		error TEXT
	);`,
	`CREATE TABLE IF NOT EXISTS latency_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		checked_at TEXT NOT NULL,
		target TEXT NOT NULL,
		latency_ms REAL,
		success INTEGER NOT NULL,
		error TEXT
	);`,
	`CREATE TABLE IF NOT EXISTS dns_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		checked_at TEXT NOT NULL,
		domain TEXT NOT NULL,
		resolver TEXT,
		duration_ms REAL,
		success INTEGER NOT NULL,
		error TEXT
	);`,
	`CREATE TABLE IF NOT EXISTS outages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at TEXT NOT NULL,
		ended_at TEXT,
		reason TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_speed_tests_started_at ON speed_tests(started_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_latency_checks_checked_at ON latency_checks(checked_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_dns_checks_checked_at ON dns_checks(checked_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_outages_started_at ON outages(started_at DESC);`,
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash TEXT NOT NULL UNIQUE,
		user_id INTEGER NOT NULL,
		created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);`,
}

func Open(ctx context.Context, path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxLifetime(time.Hour)
	if _, err := database.ExecContext(ctx, `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := Migrate(ctx, database); err != nil {
		_ = database.Close()
		return nil, err
	}
	return database, nil
}

func Migrate(ctx context.Context, database *sql.DB) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, migration := range migrations {
		version := i + 1
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&exists); err != nil && version != 1 {
			return err
		}
		if exists > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (?, ?)`, version, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
