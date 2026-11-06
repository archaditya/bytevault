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
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/security"
)

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrMFARequired        = errors.New("mfa_required")
	ErrTooManyMFAAttempts = errors.New("too many failed MFA attempts: please try again later")
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
	authCfg          config.AuthConfig
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	roleRepo *repository.RoleRepository,
	activityRepo *repository.ActivityRepository,
	authProviderRepo *repository.AuthProviderRepository,
	notifService *NotificationService,
	jwtCfg config.JWTConfig,
	authCfg config.AuthConfig,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		roleRepo:         roleRepo,
		activityRepo:     activityRepo,
		authProviderRepo: authProviderRepo,
		notifService:     notifService,
		jwtCfg:           jwtCfg,
		authCfg:          authCfg,
	}
}

func (s *AuthService) SetAuthConfig(cfg config.AuthConfig) {
	s.authCfg = cfg
}

func (s *AuthService) Register(ctx context.Context, email, password, firstName, lastName string, ip, ua *string) (*model.User, *TokenPair, error) {
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("failed to check email: %w", err)
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

// dummyBcryptHash is a pre-computed bcrypt hash (cost 10) used to equalize response
// timing when an account is not found or has a nil password, preventing user enumeration.
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func (s *AuthService) Login(ctx context.Context, email, password string, userAgent, ip *string) (*model.User, *TokenPair, error) {
	user, err := s.userRepo.FindByEmailForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if user.Password == nil {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// 30-day recovery buffer: if user was pending deletion and logs in within 30 days, restore account
	if user.DeletedAt != nil {
		if time.Since(*user.DeletedAt) > 30*24*time.Hour || (user.Status != nil && *user.Status != "pending_deletion") {
			return nil, nil, ErrInvalidCredentials
		}

		if err := s.userRepo.RestoreUser(ctx, user.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to restore user account: %w", err)
		}
		user.DeletedAt = nil
		activeStatus := "active"
		user.Status = &activeStatus

		go func(uid string) {
			_ = s.activityRepo.Log(context.Background(), &model.ActivityLog{
				UserID:       &uid,
				Action:       "user.account_restored",
				ResourceType: strPtr("user"),
				IPAddress:    ip,
				UserAgent:    userAgent,
			})
		}(user.ID)
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

	// Check MFA: if enabled, return a short-lived MFA token instead of full session
	mfaEnabled, _, mfaErr := s.userRepo.GetMFAFields(ctx, user.ID)
	if mfaErr == nil && mfaEnabled {
		mfaToken, err := s.generateMFAToken(user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate MFA token: %w", err)
		}
		// Return user with mfa_token in the TokenPair (client checks mfa_required flag)
		return user, &TokenPair{
			AccessToken: mfaToken,
			ExpiresAt:   time.Now().Add(5 * time.Minute).Unix(),
		}, ErrMFARequired
	}

	// Log activity asynchronously so it doesn't add DB latency to login response
	go func() {
		_ = s.activityRepo.Log(context.Background(), &model.ActivityLog{
			UserID:       &user.ID,
			Action:       "user.login",
			ResourceType: strPtr("session"),
			IPAddress:    ip,
			UserAgent:    userAgent,
		})
	}()

	// Auto-upgrade older high-cost (e.g. cost 14) hashes to bcrypt.DefaultCost in background
	if user.Password != nil {
		cost, costErr := bcrypt.Cost([]byte(*user.Password))
		if costErr == nil && cost > bcrypt.DefaultCost {
			go func(uid string, pw string) {
				if upgraded, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost); err == nil {
					_ = s.userRepo.UpdatePassword(context.Background(), uid, string(upgraded))
				}
			}(user.ID, password)
		}
	}

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
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Security: Reject tokens issued with a specific purpose (e.g. MFA verification tokens)
	if purpose, hasPurpose := claims["purpose"]; hasPurpose && purpose != nil {
		return nil, ErrInvalidToken
	}

	// Security: Require explicit access token type
	tokenType, _ := claims["typ"].(string)
	if tokenType != "access" {
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
	// 1. Verify token with Google's tokeninfo endpoint using dedicated HTTP client with 5s timeout
	client := &http.Client{Timeout: 5 * time.Second}
	tokenURL := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Google verify request: %w", err)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify Google token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("invalid Google token")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read Google token response: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Google token: %w", err)
	}

	// Audience verification: ensure token was minted for our application when GoogleClientID is configured
	if s.authCfg.GoogleClientID != "" {
		aud, _ := claims["aud"].(string)
		if aud != s.authCfg.GoogleClientID {
			return nil, nil, fmt.Errorf("Google token audience mismatch")
		}
	}

	// Email verification: ensure email is verified by Google
	emailVerified := false
	if ev, ok := claims["email_verified"].(bool); ok {
		emailVerified = ev
	} else if evStr, ok := claims["email_verified"].(string); ok {
		emailVerified = strings.ToLower(evStr) == "true"
	}
	if !emailVerified {
		return nil, nil, fmt.Errorf("Google account email is not verified")
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

	if s.userRepo == nil {
		return nil, nil, fmt.Errorf("user repository not configured")
	}

	var user *model.User

	// STEP A: Look for existing Google OAuth link in auth_providers table
	if s.authProviderRepo != nil {
		existingLink, linkErr := s.authProviderRepo.FindByProviderAndID(ctx, "google", googleSub)
		if linkErr == nil && existingLink != nil {
			// OAuth link found — fetch linked user
			user, err = s.userRepo.FindByIDForAuth(ctx, existingLink.UserID)
			if err != nil {
				return nil, nil, fmt.Errorf("linked user account not found: %w", err)
			}
		}
	}

	if user == nil {
		// STEP B: Look up user by normalized email (case-insensitive)
		user, err = s.userRepo.FindByEmailForAuth(ctx, normalizedEmail)
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
			hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
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
			// STEP D: Existing user found
			// If within 30-day recovery buffer, seamlessly restore account
			if user.DeletedAt != nil {
				if time.Since(*user.DeletedAt) > 30*24*time.Hour || (user.Status != nil && *user.Status != "pending_deletion") {
					return nil, nil, ErrInvalidCredentials
				}
				if err := s.userRepo.RestoreUser(ctx, user.ID); err != nil {
					return nil, nil, fmt.Errorf("failed to restore user account: %w", err)
				}
				user.DeletedAt = nil
				activeStatus := "active"
				user.Status = &activeStatus
				go func(uid string) {
					_ = s.activityRepo.Log(context.Background(), &model.ActivityLog{
						UserID:       &uid,
						Action:       "user.account_restored",
						ResourceType: strPtr("user"),
						IPAddress:    ip,
						UserAgent:    userAgent,
					})
				}(user.ID)
			}
			// Link Google OAuth to this user!
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
				// Pre-hijack mitigation: genuine owner proved ownership via Google OAuth.
				// Invalidate all existing sessions and clear unverified password.
				user.Password = nil
				_ = s.sessionRepo.DeleteAllByUserID(ctx, user.ID)
				_ = s.userRepo.ClearPassword(ctx, user.ID)
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

// --- MFA Methods ---

// SetupMFA generates a new TOTP secret and returns the otpauth:// URI for QR scanning
func (s *AuthService) SetupMFA(ctx context.Context, userID string) (string, string, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	secret, err := security.GenerateTOTPSecret()
	if err != nil {
		return "", "", err
	}

	// Store secret but don't enable MFA yet (user must verify first)
	if err := s.userRepo.UpdateMFA(ctx, userID, false, &secret); err != nil {
		return "", "", fmt.Errorf("failed to store MFA secret: %w", err)
	}

	qrURI := security.GenerateQRCodeURI(secret, user.Email)
	return secret, qrURI, nil
}

// EnableMFA verifies the TOTP code and activates MFA for the user
func (s *AuthService) EnableMFA(ctx context.Context, userID, code string) error {
	// Rate limit check before verifying TOTP code
	if s.notifService != nil {
		attempts, err := s.notifService.CheckMFARateLimit(ctx, userID)
		if err == nil && attempts > 5 {
			return ErrTooManyMFAAttempts
		}
	}

	_, mfaSecret, err := s.userRepo.GetMFAFields(ctx, userID)
	if err != nil || mfaSecret == nil || *mfaSecret == "" {
		return fmt.Errorf("MFA not set up. Call /auth/mfa/setup first")
	}

	if !security.VerifyTOTPCode(*mfaSecret, code) {
		return fmt.Errorf("invalid TOTP code")
	}

	// Clear counter on success
	if s.notifService != nil {
		_ = s.notifService.ClearMFARateLimit(ctx, userID)
	}

	return s.userRepo.UpdateMFA(ctx, userID, true, mfaSecret)
}

// DisableMFA verifies the TOTP code and deactivates MFA for the user
func (s *AuthService) DisableMFA(ctx context.Context, userID, code string) error {
	// Rate limit check before verifying TOTP code
	if s.notifService != nil {
		attempts, err := s.notifService.CheckMFARateLimit(ctx, userID)
		if err == nil && attempts > 5 {
			return ErrTooManyMFAAttempts
		}
	}

	mfaEnabled, mfaSecret, err := s.userRepo.GetMFAFields(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to read MFA status: %w", err)
	}
	if !mfaEnabled || mfaSecret == nil {
		return fmt.Errorf("MFA is not enabled")
	}

	if !security.VerifyTOTPCode(*mfaSecret, code) {
		return fmt.Errorf("invalid TOTP code")
	}

	// Clear counter on success
	if s.notifService != nil {
		_ = s.notifService.ClearMFARateLimit(ctx, userID)
	}

	return s.userRepo.UpdateMFA(ctx, userID, false, nil)
}

// VerifyMFALogin validates the short-lived MFA token + TOTP code, then issues a full JWT session
func (s *AuthService) VerifyMFALogin(ctx context.Context, mfaToken, code string, userAgent, ip *string) (*model.User, *TokenPair, error) {
	userID, err := s.validateMFAToken(mfaToken)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	// Security: Rate limit failed TOTP attempts per user ID using Redis (max 5 attempts per 15 mins)
	if s.notifService != nil {
		attempts, err := s.notifService.CheckMFARateLimit(ctx, userID)
		if err == nil && attempts > 5 {
			return nil, nil, ErrTooManyMFAAttempts
		}
	}

	if s.userRepo == nil {
		return nil, nil, fmt.Errorf("user repository not configured")
	}

	_, mfaSecret, mfaErr := s.userRepo.GetMFAFields(ctx, userID)
	if mfaErr != nil || mfaSecret == nil {
		return nil, nil, fmt.Errorf("MFA not configured")
	}

	if !security.VerifyTOTPCode(*mfaSecret, code) {
		return nil, nil, fmt.Errorf("invalid TOTP code")
	}

	// Successful verification: clear attempt counter
	if s.notifService != nil {
		_ = s.notifService.ClearMFARateLimit(ctx, userID)
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	role, roleErr := s.roleRepo.GetUserRole(ctx, user.ID)
	if roleErr == nil {
		user.RoleName = role.Name
		user.Permissions = role.Permissions
	}

	s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &user.ID,
		Action:       "user.mfa_login",
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

// generateMFAToken creates a short-lived JWT (5 min) for the MFA verification step
func (s *AuthService) generateMFAToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":     userID,
		"purpose": "mfa_verify",
		"exp":     time.Now().Add(5 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.Secret))
}

// validateMFAToken parses and validates the short-lived MFA JWT
func (s *AuthService) validateMFAToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
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

	purpose, _ := claims["purpose"].(string)
	if purpose != "mfa_verify" {
		return "", ErrInvalidToken
	}

	userID, _ := claims["sub"].(string)
	if userID == "" {
		return "", ErrInvalidToken
	}

	return userID, nil
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

	// JWT now includes role + permissions + explicit token type
	accessClaims := jwt.MapClaims{
		"sub":         userID,
		"role":        role,
		"permissions": permissions,
		"typ":         "access",
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
