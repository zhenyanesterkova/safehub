package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/zhenyanesterkova/safehub/internal/server/services"
)

var (
	StatusInternalServerError = "something went wrong on the server..."
)

type AuthHandler struct {
	authService *services.AuthService
	logger      *services.LogService
}

func NewAuthHandler(
	authService *services.AuthService,
	logger *services.LogService,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req services.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenResponse, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		h.logger.Log.Errorf("Error during registration: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(tokenResponse)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req services.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenResponse, err := h.authService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		h.logger.Log.Errorf("Error during login: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(tokenResponse)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req services.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokenResponse, err := h.authService.RefreshToken(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidToken) ||
			errors.Is(err, services.ErrInvalidCredentials) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		h.logger.Log.Errorf("Error during refresh token: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(tokenResponse)
	if err != nil {
		h.logger.Log.Errorf("Error encoding response: %v", err)
		http.Error(w, StatusInternalServerError, http.StatusInternalServerError)
		return
	}
}
