package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/caiyuan0111/aicode/internal/model"
)

var ErrNotFound = errors.New("record not found")

// UserStorer defines the interface for user persistence operations.
type UserStorer interface {
	Create(ctx context.Context, email, passwordHash string) (int64, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, email, passwordHash string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		email, passwordHash, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?`, email)

	var u model.User
	var createdAt, updatedAt string
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (*model.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = ?`, id)

	var u model.User
	var createdAt, updatedAt string
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}
