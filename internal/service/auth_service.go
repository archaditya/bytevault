package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/adityakkpk/bytevault/internal/config"
	"github.com/adityakkpk/bytevault/internal/model"
	"github.com/adityakkpk/bytevault/internal/repository"
)

var (
	ErrEmailExists       = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// TokenPair holds access + refresh tokens returned to client
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// AuthService handles registration, login, token generation
type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	jwtCfg      config.JWTConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	jwtCfg config.JWTConfig,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtCfg:      jwtCfg,
	}
}

// Register creates a new user and returns tokens
func (s *AuthService) Register(ctx context.Context, email, password, firstName, lastName string) (*model.User, *TokenPair, error) {
	// Check if email already taken (active user)
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("failed to check email: %w", err)
	}

	// Hash password with bcrypt
	// bcrypt.GenerateFromPassword takes []byte and a cost factor (14 = secure)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashedBytes)

	// Create user
	user := &model.User{
		Email:     email,
		Password:  &hashedPassword,
		FirstName: &firstName,
		LastName:  &lastName,
	}

	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	tokens, err := s.createSession(ctx, created.ID, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	return created, tokens, nil
}

// Login verifies credentials and returns tokens
func (s *AuthService) Login(ctx context.Context, email, password string, userAgent, ip *string) (*model.User, *TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	// User has no password (OAuth-only user)
	if user.Password == nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Compare password with bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.createSession(ctx, user.ID, userAgent, ip)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshTokens generates new tokens from a valid refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Hash the token to look it up in DB
	tokenHash := hashToken(refreshToken)

	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, ErrInvalidToken
	}

	// Delete old session, create new one
	s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)

	tokens, err := s.createSession(ctx, session.UserID, session.UserAgent, session.IPAddress)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// Logout deletes the session for a refresh token
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
}

// ValidateAccessToken parses and validates a JWT access token
// Returns the user ID from the token claims
func (s *AuthService) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", ErrInvalidToken
	}

	return userID, nil
}

// --- Private helpers ---

// createSession generates tokens and stores session in DB
func (s *AuthService) createSession(ctx context.Context, userID string, userAgent, ip *string) (*TokenPair, error) {
	// Parse expiry durations from config strings
	accessDuration, err := time.ParseDuration(s.jwtCfg.AccessExpiry)
	if err != nil {
		accessDuration = 15 * time.Minute
	}
	refreshDuration, err := time.ParseDuration(s.jwtCfg.RefreshExpiry)
	if err != nil {
		refreshDuration = 7 * 24 * time.Hour
	}

	// Generate JWT access token
	now := time.Now()
	accessClaims := jwt.MapClaims{
		"sub": userID,
		"iat": now.Unix(),
		"exp": now.Add(accessDuration).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString([]byte(s.jwtCfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate random refresh token (32 bytes = 64 hex chars)
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshString := hex.EncodeToString(refreshBytes)

	// Store session with HASHED refresh token
	session := &repository.Session{
		UserID:           userID,
		RefreshTokenHash: hashToken(refreshString),
		UserAgent:        userAgent,
		IPAddress:        ip,
		ExpiresAt:        now.Add(refreshDuration),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessString,
		RefreshToken: refreshString, // Raw token goes to client
		ExpiresAt:    now.Add(accessDuration).Unix(),
	}, nil
}

// hashToken creates SHA-256 hash of a token for secure DB storage
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}