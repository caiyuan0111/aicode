package handler

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/caiyuan0111/aicode/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

type contextKey string

const (
	userIDKey contextKey = "user_id"
	emailKey  contextKey = "email"
)

func GetUserID(ctx context.Context) int64 {
	v, _ := ctx.Value(userIDKey).(int64)
	return v
}

func GetEmail(ctx context.Context) string {
	v, _ := ctx.Value(emailKey).(string)
	return v
}

// AuthMiddleware validates the JWT access token and injects user info into the context.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errorJSON(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				errorJSON(w, http.StatusUnauthorized, "malformed authorization header")
				return
			}
			tokenStr := parts[1]

			claims := &model.Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims,
				func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(jwtSecret), nil
				},
			)
			if err != nil || !token.Valid {
				errorJSON(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			if claims.Type != "access" {
				errorJSON(w, http.StatusUnauthorized, "invalid token type")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			ctx = context.WithValue(ctx, emailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RateLimitMiddleware limits the number of requests per IP.
type RateLimitMiddleware struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func NewRateLimitMiddleware() *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		rl.mu.Lock()
		lim, ok := rl.limiters[ip]
		if !ok {
			// 5 requests per minute with burst of 5
			lim = rate.NewLimiter(rate.Limit(5.0/60.0), 5)
			rl.limiters[ip] = lim
		}
		rl.mu.Unlock()

		if !lim.Allow() {
			errorJSON(w, http.StatusTooManyRequests, "too many requests, try again later")
			return
		}

		next.ServeHTTP(w, r)
	})
}
