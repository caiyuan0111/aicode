package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// TokenStorer defines the interface for refresh token persistence operations.
type TokenStorer interface {
	Store(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (*StoredToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID int64) error
}

type StoredToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type TokenStore struct {
	db *sql.DB
}

func NewTokenStore(db *sql.DB) *TokenStore {
	return &TokenStore{db: db}
}

func (s *TokenStore) Store(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	now := time.Now().UTC().Format(time.RFC3339)
	exp := expiresAt.UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		userID, tokenHash, exp, now,
	)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (s *TokenStore) GetByHash(ctx context.Context, tokenHash string) (*StoredToken, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at FROM refresh_tokens WHERE token_hash = ?`,
		tokenHash,
	)

	var t StoredToken
	var expiresAt, createdAt string
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &expiresAt, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan token: %w", err)
	}

	t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &t, nil
}

func (s *TokenStore) Revoke(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete token: %w", err)
	}
	return nil
}

func (s *TokenStore) RevokeAllForUser(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete tokens for user: %w", err)
	}
	return nil
}
