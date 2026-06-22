package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/caiyuan0111/aicode/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRequest is the JSON body for registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the JSON body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest is the JSON body for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// HandleRegister handles POST /api/auth/register
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authService.Register(r.Context(), req.Email, req.Password); err != nil {
		errorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, nil)
}

// HandleLogin handles POST /api/auth/login
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tokens, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			errorJSON(w, http.StatusUnauthorized, err.Error())
		} else {
			errorJSON(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// HandleRefresh handles POST /api/auth/refresh
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		errorJSON(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	tokens, err := h.authService.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// HandleMe handles GET /api/me
func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())
	if userID == 0 {
		errorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.authService.Me(r.Context(), userID)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}
