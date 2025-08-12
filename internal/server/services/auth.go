package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhenyanesterkova/safehub/internal/models"
	"github.com/zhenyanesterkova/safehub/internal/server/storage"
	"github.com/zhenyanesterkova/safehub/internal/shared/crypto"
)

const (
	minPasswordLength = 6
	minUsernameLength = 3
	maxUsernameLength = 50
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// AuthClaims представляет claims для JWT токена
type AuthClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService предоставляет сервисы аутентификации и авторизации
type AuthService struct {
	userRepo   storage.UserRepository
	jwtSecret  []byte
	tokenTTL   time.Duration
	refreshTTL time.Duration
	crypto     *crypto.CryptoService
}

// NewAuthService создает новый экземпляр AuthService
func NewAuthService(
	userRepo storage.UserRepository,
	jwtSecret string,
	tokenTTL,
	refreshTTL time.Duration,
	cryptoSvc *crypto.CryptoService,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtSecret:  []byte(jwtSecret),
		tokenTTL:   tokenTTL,
		refreshTTL: refreshTTL,
		crypto:     cryptoSvc,
	}
}

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=minUsernameLength,max=maxUsernameLength"`
	Password string `json:"password" validate:"required,min=minPasswordLength"`
	Email    string `json:"email" validate:"required,email"`
}

// LoginRequest представляет запрос на аутентификацию
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// TokenResponse представляет ответ с токенами
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// RefreshRequest представляет запрос на обновление токена
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error) {
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, models.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	salt, err := s.crypto.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	hashedPassword, err := s.hashPassword(req.Password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Salt:         string(salt),
		Email:        req.Email,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.generateTokens(user)
}

// Login аутентифицирует пользователя
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	hashedPassword, err := s.hashPassword(req.Password, []byte(user.Salt))
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if hashedPassword != user.PasswordHash {
		return nil, ErrInvalidCredentials
	}

	user.LastLoginAt = time.Now()
	if err := s.userRepo.UpdateLastLoginAt(ctx, user.ID, user.LastLoginAt); err != nil {
		log.Printf("Warning: failed to update last login time: %v", err)
	}

	return s.generateTokens(user)
}

// RefreshToken обновляет access token используя refresh token
func (s *AuthService) RefreshToken(ctx context.Context, req RefreshRequest) (*TokenResponse, error) {
	token, err := jwt.ParseWithClaims(req.RefreshToken, &AuthClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.RegisteredClaims.Subject != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID from string var: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.generateTokens(user)
}

// ValidateToken валидирует access token и возвращает пользователя
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*models.User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.RegisteredClaims.Subject != "access" {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID from string var: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// ChangePassword изменяет пароль пользователя
func (s *AuthService) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	hashedOldPassword, err := s.hashPassword(oldPassword, []byte(user.Salt))
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if hashedOldPassword != user.PasswordHash {
		return ErrInvalidCredentials
	}

	salt, err := s.crypto.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	hashedNewPassword, err := s.hashPassword(newPassword, []byte(salt))
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hashedNewPassword)
	user.Salt = string(salt)
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// generateTokens генерирует access и refresh токены
func (s *AuthService) generateTokens(user *models.User) (*TokenResponse, error) {
	now := time.Now()
	accessExpiresAt := now.Add(s.tokenTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)

	accessClaims := &AuthClaims{
		UserID:   user.ID.String(),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "access",
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := &AuthClaims{
		UserID:   user.ID.String(),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "refresh",
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiresAt,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) hashPassword(password string, salt []byte) (string, error) {
	saltedPassword := append([]byte(password), salt...)

	hash, err := bcrypt.GenerateFromPassword(saltedPassword, bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}
