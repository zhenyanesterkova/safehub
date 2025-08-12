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

// AuthHandler обрабатывает запросы аутентификации и авторизации
type AuthHandler struct {
	authService *services.AuthService
	logger      *services.LogService
}

// NewAuthHandler создает новый экземпляр AuthHandler
func NewAuthHandler(
	authService *services.AuthService,
	logger *services.LogService,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register регистрирует нового пользователя в системе
// @Summary Регистрация нового пользователя
// @Description Создает нового пользователя и возвращает JWT токены для аутентификации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.RegisterRequest true "Данные для регистрации"
// @Success 201 {object} services.TokenResponse "Успешная регистрация с JWT токенами"
// @Failure 400 {string} "Некорректный формат запроса"
// @Failure 409 {string} "Пользователь уже существует"
// @Failure 500 {string} "Внутренняя ошибка сервера"
// @Router /api/v1/auth/register [post]
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

// Login аутентифицирует пользователя по логину и паролю
// @Summary Аутентификация пользователя
// @Description Проверяет учетные данные пользователя и возвращает JWT токены
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "Учетные данные пользователя"
// @Success 200 {object} services.TokenResponse "Успешная аутентификация с JWT токенами"
// @Failure 400 {string} "Некорректный формат запроса"
// @Failure 401 {string} "Неверные учетные данные"
// @Failure 500 {string} "Внутренняя ошибка сервера"
// @Router /api/v1/auth/login [post]
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

// Refresh обновляет access токен используя refresh токен
// @Summary Обновление access токена
// @Description Обновляет истекший access токен используя действующий refresh токен
// @Tags auth
// @Accept json
// @Produce json
// @Param request body services.RefreshRequest true "Refresh токен для обновления"
// @Success 200 {object} services.TokenResponse "Новые JWT токены"
// @Failure 400 {string} "Некорректный формат запроса"
// @Failure 401 {string} "Недействительный refresh токен"
// @Failure 500 {string} "Внутренняя ошибка сервера"
// @Router /api/v1/auth/refresh [post]
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
