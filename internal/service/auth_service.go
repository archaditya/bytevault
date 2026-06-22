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

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// TokenClaims holds the decoded JWT claims
type TokenClaims struct {
	UserID      string
	Role        string
	Permissions map[string]bool
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type AuthService struct {
	userRepo     *repository.UserRepository
	sessionRepo  *repository.SessionRepository
	roleRepo     *repository.RoleRepository
	activityRepo *repository.ActivityRepository
	jwtCfg       config.JWTConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	roleRepo *repository.RoleRepository,
	activityRepo *repository.ActivityRepository,
	jwtCfg config.JWTConfig,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		roleRepo:     roleRepo,
		activityRepo: activityRepo,
		jwtCfg:       jwtCfg,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, firstName, lastName string, ip, ua *string) (*model.User, *TokenPair, error) {
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("failed to check email: %w", err)
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashedBytes)

	user := &model.User{
		Email:    email,
		Password: &hashedPassword,
		FirstName: &firstName,
		LastName:  &lastName,
	}

	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Auto-assign "user" role
	defaultRole, err := s.roleRepo.FindByName(ctx, "user")
	if err == nil {
		s.roleRepo.AssignRoleToUser(ctx, created.ID, defaultRole.ID, nil)
		created.RoleName = defaultRole.Name
		created.Permissions = defaultRole.Permissions
	}

	// Log activity
	s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &created.ID,
		Action:       "user.register",
		ResourceType: strPtr("user"),
		ResourceID:   strPtr(created.ID),
		IPAddress:    ip,
		UserAgent:    ua,
	})

	tokens, err := s.createSession(ctx, created.ID, created.RoleName, created.Permissions, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	return created, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string, userAgent, ip *string) (*model.User, *TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if user.Password == nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Get user's role and permissions
	role, err := s.roleRepo.GetUserRole(ctx, user.ID)
	if err == nil {
		user.RoleName = role.Name
		user.Permissions = role.Permissions
	}

	// Log activity
	s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &user.ID,
		Action:       "user.login",
		ResourceType: strPtr("session"),
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	tokens, err := s.createSession(ctx, user.ID, user.RoleName, user.Permissions, userAgent, ip)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, ErrInvalidToken
	}

	s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)

	// Get fresh role/permissions from DB (in case admin changed them)
	role, err := s.roleRepo.GetUserRole(ctx, session.UserID)
	roleName := "user"
	var perms map[string]bool
	if err == nil {
		roleName = role.Name
		perms = role.Permissions
	}

	tokens, err := s.createSession(ctx, session.UserID, roleName, perms, session.UserAgent, session.IPAddress)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	return s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
}

// ValidateAccessToken now returns full TokenClaims (userID + role + permissions)
func (s *AuthService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	userID, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)

	// Parse permissions from JWT claims
	perms := make(map[string]bool)
	if permRaw, ok := claims["permissions"].(map[string]any); ok {
		for k, v := range permRaw {
			if boolVal, ok := v.(bool); ok {
				perms[k] = boolVal
			}
		}
	}

	if userID == "" {
		return nil, ErrInvalidToken
	}

	return &TokenClaims{
		UserID:      userID,
		Role:        role,
		Permissions: perms,
	}, nil
}

// --- Private helpers ---

func (s *AuthService) createSession(ctx context.Context, userID, role string, permissions map[string]bool, userAgent, ip *string) (*TokenPair, error) {
	accessDuration, err := time.ParseDuration(s.jwtCfg.AccessExpiry)
	if err != nil {
		accessDuration = 15 * time.Minute
	}
	refreshDuration, err := time.ParseDuration(s.jwtCfg.RefreshExpiry)
	if err != nil {
		refreshDuration = 7 * 24 * time.Hour
	}

	now := time.Now()

	// JWT now includes role + permissions
	accessClaims := jwt.MapClaims{
		"sub":         userID,
		"role":        role,
		"permissions": permissions,
		"iat":         now.Unix(),
		"exp":         now.Add(accessDuration).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString([]byte(s.jwtCfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshString := hex.EncodeToString(refreshBytes)

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
		RefreshToken: refreshString,
		ExpiresAt:    now.Add(accessDuration).Unix(),
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// strPtr is a helper to create a pointer to a string.
// Needed because Go can't take the address of a literal: &"hello" doesn't work
func strPtr(s string) *string {
	return &s
}