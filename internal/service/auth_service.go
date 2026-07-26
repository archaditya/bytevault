package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
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
	userRepo         *repository.UserRepository
	sessionRepo      *repository.SessionRepository
	roleRepo         *repository.RoleRepository
	activityRepo     *repository.ActivityRepository
	authProviderRepo *repository.AuthProviderRepository
	notifService     *NotificationService
	jwtCfg           config.JWTConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	roleRepo *repository.RoleRepository,
	activityRepo *repository.ActivityRepository,
	authProviderRepo *repository.AuthProviderRepository,
	notifService *NotificationService,
	jwtCfg config.JWTConfig,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		roleRepo:         roleRepo,
		activityRepo:     activityRepo,
		authProviderRepo: authProviderRepo,
		notifService:     notifService,
		jwtCfg:           jwtCfg,
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
		Email:     email,
		Password:  &hashedPassword,
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

	// Generate and send OTP for email verification
	if s.notifService != nil {
		if err := s.notifService.GenerateAndSendOTP(ctx, created.ID, created.Email, firstName, "registration"); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to send registration verification OTP")
		}
	}

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

	// Trigger OTP if user is not verified yet so they receive a fresh code on login
	if !user.IsVerified && s.notifService != nil {
		if err := s.notifService.GenerateAndSendOTP(ctx, user.ID, user.Email, *user.FirstName, "registration"); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to auto-send verification OTP on login")
		}
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

// GoogleLogin verifies a Google ID token and seamlessly links or creates the user.
func (s *AuthService) GoogleLogin(
	ctx context.Context,
	idToken string,
	reqFirstName *string,
	reqLastName *string,
	reqAvatarURL *string,
	userAgent *string,
	ip *string,
) (*model.User, *TokenPair, error) {
	// 1. Verify token with Google's tokeninfo endpoint
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify Google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("invalid Google token")
	}

	body, _ := io.ReadAll(resp.Body)
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Google token: %w", err)
	}

	rawEmail, _ := claims["email"].(string)
	googleSub, _ := claims["sub"].(string)
	if rawEmail == "" || googleSub == "" {
		return nil, nil, fmt.Errorf("Google token missing email or sub claim")
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(rawEmail))

	// Extract profile fields from ID Token claims
	var firstName, lastName, avatarURL string
	if val, ok := claims["given_name"].(string); ok {
		firstName = val
	}
	if val, ok := claims["family_name"].(string); ok {
		lastName = val
	}
	if val, ok := claims["picture"].(string); ok {
		avatarURL = val
	}

	// Fallback to splitting full "name" claim
	if firstName == "" {
		if fullName, ok := claims["name"].(string); ok && fullName != "" {
			parts := strings.Split(fullName, " ")
			firstName = parts[0]
			if len(parts) > 1 {
				lastName = strings.Join(parts[1:], " ")
			}
		}
	}

	// 2. Fallback to frontend-provided fields if claims were empty
	if firstName == "" && reqFirstName != nil {
		firstName = *reqFirstName
	}
	if lastName == "" && reqLastName != nil {
		lastName = *reqLastName
	}
	if avatarURL == "" && reqAvatarURL != nil {
		avatarURL = *reqAvatarURL
	}

	var user *model.User

	// STEP A: Look for existing Google OAuth link in auth_providers table
	existingLink, linkErr := s.authProviderRepo.FindByProviderAndID(ctx, "google", googleSub)
	if linkErr == nil && existingLink != nil {
		// OAuth link found — fetch linked user
		user, err = s.userRepo.FindByID(ctx, existingLink.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("linked user account not found: %w", err)
		}
	} else {
		// STEP B: Look up user by normalized email (case-insensitive)
		user, err = s.userRepo.FindByEmail(ctx, normalizedEmail)
		if err != nil {
			if !errors.Is(err, repository.ErrUserNotFound) {
				return nil, nil, err
			}

			// STEP C: User doesn't exist — create new user account
			tempPass := make([]byte, 16)
			if _, randErr := rand.Read(tempPass); randErr != nil {
				return nil, nil, fmt.Errorf("failed to generate secure temp password: %w", randErr)
			}
			tempPassword := hex.EncodeToString(tempPass)
			hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(tempPassword), 14)
			hashedPassword := string(hashedBytes)

			user = &model.User{
				Email:      normalizedEmail,
				Password:   &hashedPassword,
				FirstName:  &firstName,
				LastName:   &lastName,
				AvatarURL:  &avatarURL,
				IsVerified: true, // Google OAuth emails are verified
			}

			user, err = s.userRepo.Create(ctx, user)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create Google user: %w", err)
			}

			defaultRole, roleErr := s.roleRepo.FindByName(ctx, "user")
			if roleErr == nil {
				s.roleRepo.AssignRoleToUser(ctx, user.ID, defaultRole.ID, nil)
				user.RoleName = defaultRole.Name
				user.Permissions = defaultRole.Permissions
			}
		} else {
			// STEP D: Existing email/pass user found — Link Google OAuth to this user!
			updated := false
			if (user.FirstName == nil || *user.FirstName == "") && firstName != "" {
				user.FirstName = &firstName
				updated = true
			}
			if (user.LastName == nil || *user.LastName == "") && lastName != "" {
				user.LastName = &lastName
				updated = true
			}
			if (user.AvatarURL == nil || *user.AvatarURL == "") && avatarURL != "" {
				user.AvatarURL = &avatarURL
				updated = true
			}
			if !user.IsVerified {
				user.IsVerified = true
				updated = true
			}

			if updated {
				s.userRepo.UpdateDetails(ctx, user.ID, user.FirstName, user.LastName, nil, &user.IsVerified, nil, nil)
			}
		}

		// Save link in auth_providers table so future logins match instantly
		_, _ = s.authProviderRepo.Create(ctx, &model.AuthProvider{
			UserID:         user.ID,
			Provider:       "google",
			ProviderUserID: googleSub,
			Email:          &normalizedEmail,
		})
	}

	if user.RoleName == "" {
		role, roleErr := s.roleRepo.GetUserRole(ctx, user.ID)
		if roleErr == nil {
			user.RoleName = role.Name
			user.Permissions = role.Permissions
		}
	}

	// Log activity
	s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &user.ID,
		Action:       "user.google_login",
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

// --- Private helpers ---

// CreateSessionForVerifiedUser creates an authenticated session for a user who just completed email verification.
// This allows auto-login after OTP verification without requiring a separate login step.
func (s *AuthService) CreateSessionForVerifiedUser(ctx context.Context, userID, role string, permissions map[string]bool, userAgent, ip *string) (*TokenPair, error) {
	// Log activity
	s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &userID,
		Action:       "user.email_verified",
		ResourceType: strPtr("session"),
		IPAddress:    ip,
		UserAgent:    userAgent,
	})

	return s.createSession(ctx, userID, role, permissions, userAgent, ip)
}

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
