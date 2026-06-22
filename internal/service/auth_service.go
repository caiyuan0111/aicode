package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caiyuan0111/aicode/internal/config"
	"github.com/caiyuan0111/aicode/internal/model"
	"github.com/caiyuan0111/aicode/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
)

type AuthService struct {
	userStore  store.UserStorer
	tokenStore store.TokenStorer
	cfg        *config.Config
}

func NewAuthService(userStore store.UserStorer, tokenStore store.TokenStorer, cfg *config.Config) *AuthService {
	return &AuthService{
		userStore:  userStore,
		tokenStore: tokenStore,
		cfg:        cfg,
	}
}

// Register creates a new user account. Returns a generic error on duplicate email.
func (s *AuthService) Register(ctx context.Context, email, password string) error {
	if err := validateEmail(email); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}

	// Check if email already exists
	if _, err := s.userStore.GetByEmail(ctx, email); err == nil {
		// User exists — return generic error to prevent enumeration
		return errors.New("registration failed")
	} else if !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("check email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := s.userStore.Create(ctx, email, string(hash)); err != nil {
		// Check for unique constraint violation (race condition)
		if strings.Contains(err.Error(), "UNIQUE") {
			return errors.New("registration failed")
		}
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// Login authenticates a user and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*model.TokenPair, error) {
	if err := validateEmail(email); err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := validatePassword(password); err != nil {
		return nil, ErrInvalidCredentials
	}

	u, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(ctx, u.ID, u.Email)
}

// Refresh rotates a valid refresh token, revoking the old one and issuing new tokens.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	// Parse and validate the refresh JWT
	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(s.cfg.JWTSecret), nil
		},
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}

	// Check the refresh token hasn't been revoked
	tokenHash := hashToken(refreshToken)
	stored, err := s.tokenStore.GetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("get token: %w", err)
	}

	if time.Now().After(stored.ExpiresAt) {
		// Clean up expired token
		s.tokenStore.Revoke(ctx, tokenHash)
		return nil, ErrInvalidToken
	}

	// Revoke the old refresh token (rotation)
	if err := s.tokenStore.Revoke(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke token: %w", err)
	}

	return s.generateTokens(ctx, claims.UserID, claims.Email)
}

// Me returns the user without the password hash.
func (s *AuthService) Me(ctx context.Context, userID int64) (*model.User, error) {
	u, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *AuthService) generateTokens(ctx context.Context, userID int64, email string) (*model.TokenPair, error) {
	now := time.Now()

	// Access token
	accessExpiry := now.Add(s.cfg.AccessTokenExpiry)
	accessClaims := &model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("%d-%d", userID, now.UnixNano()),
		},
		UserID: userID,
		Email:  email,
		Type:   "access",
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token
	refreshExpiry := now.Add(s.cfg.RefreshTokenExpiry)
	refreshClaims := &model.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			Subject:   fmt.Sprintf("%d", userID),
			ID:        fmt.Sprintf("%d-%d", userID, now.UnixNano()),
		},
		UserID: userID,
		Email:  email,
		Type:   "refresh",
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	// Store refresh token hash
	if err := s.tokenStore.Store(ctx, userID, hashToken(refreshToken), refreshExpiry); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.AccessTokenExpiry.Seconds()),
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(email, "@") {
		return errors.New("invalid email format")
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return errors.New("invalid email format")
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}
