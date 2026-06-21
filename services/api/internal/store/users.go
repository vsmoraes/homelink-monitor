package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"homelink-monitor/services/api/internal/domain"
)

type UserWithPassword struct {
	domain.User
	PasswordHash string
}

func (s *Store) UserCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash, role string, now time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO users(username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		username, passwordHash, role, ts(now), ts(now))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Users(ctx context.Context) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, role, created_at, updated_at FROM users ORDER BY username ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		user, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

func (s *Store) UserByUsername(ctx context.Context, username string) (*UserWithPassword, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, password_hash, role, created_at, updated_at FROM users WHERE username = ?`, username)
	return scanUserWithPassword(row)
}

func (s *Store) UserByID(ctx context.Context, id int64) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, username, role, created_at, updated_at FROM users WHERE id = ?`, id)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (s *Store) UpdateUser(ctx context.Context, id int64, username, role string, passwordHash *string, now time.Time) error {
	if passwordHash != nil {
		_, err := s.db.ExecContext(ctx, `UPDATE users SET username = ?, role = ?, password_hash = ?, updated_at = ? WHERE id = ?`,
			username, role, *passwordHash, ts(now), id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE users SET username = ?, role = ?, updated_at = ? WHERE id = ?`,
		username, role, ts(now), id)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) CreateSession(ctx context.Context, token string, userID int64, now, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions(token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash(token), userID, ts(now), ts(expiresAt))
	return err
}

func (s *Store) UserBySession(ctx context.Context, token string, now time.Time) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT u.id, u.username, u.role, u.created_at, u.updated_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`, tokenHash(token), ts(now))
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash(token))
	return err
}

func (s *Store) DeleteSessionsForUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, ts(now))
	return err
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func scanUser(row scanner) (*domain.User, error) {
	var user domain.User
	var created, updated string
	if err := row.Scan(&user.ID, &user.Username, &user.Role, &created, &updated); err != nil {
		return nil, err
	}
	createdAt, err := parseTS(created)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTS(updated)
	if err != nil {
		return nil, err
	}
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt
	return &user, nil
}

func scanUserRows(row scanner) (*domain.User, error) { return scanUser(row) }

func scanUserWithPassword(row scanner) (*UserWithPassword, error) {
	var user UserWithPassword
	var created, updated string
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	createdAt, err := parseTS(created)
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTS(updated)
	if err != nil {
		return nil, err
	}
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt
	return &user, nil
}
