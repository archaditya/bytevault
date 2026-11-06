package service_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/service"
)

func TestValidateAccessToken_MFAIsolationAndTypeCheck(t *testing.T) {
	jwtCfg := config.JWTConfig{
		Secret:        "super-secret-test-key-32-bytes-long!",
		AccessExpiry:  "15m",
		RefreshExpiry: "168h",
	}

	authSvc := service.NewAuthService(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		jwtCfg,
		config.AuthConfig{},
	)

	now := time.Now()

	// 1. Valid Access Token with "typ": "access"
	validClaims := jwt.MapClaims{
		"sub":         "user-123",
		"role":        "user",
		"permissions": map[string]any{"files:read": true},
		"typ":         "access",
		"iat":         now.Unix(),
		"exp":         now.Add(15 * time.Minute).Unix(),
	}
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims)
	validTokenStr, err := validToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		t.Fatalf("failed to sign valid token: %v", err)
	}

	claims, err := authSvc.ValidateAccessToken(validTokenStr)
	if err != nil {
		t.Fatalf("expected valid token to pass, got err: %v", err)
	}
	if claims.UserID != "user-123" || claims.Role != "user" || !claims.Permissions["files:read"] {
		t.Fatalf("unexpected claims from valid token: %+v", claims)
	}

	// 2. MFA Token with "purpose": "mfa_verify" (MUST be rejected by ValidateAccessToken)
	mfaClaims := jwt.MapClaims{
		"sub":     "user-123",
		"purpose": "mfa_verify",
		"iat":     now.Unix(),
		"exp":     now.Add(5 * time.Minute).Unix(),
	}
	mfaToken := jwt.NewWithClaims(jwt.SigningMethodHS256, mfaClaims)
	mfaTokenStr, err := mfaToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		t.Fatalf("failed to sign mfa token: %v", err)
	}

	_, err = authSvc.ValidateAccessToken(mfaTokenStr)
	if err != service.ErrInvalidToken {
		t.Fatalf("expected MFA token to be rejected with ErrInvalidToken, got: %v", err)
	}

	// 3. Token missing "typ": "access" (MUST be rejected)
	untypedClaims := jwt.MapClaims{
		"sub":  "user-123",
		"role": "user",
		"iat":  now.Unix(),
		"exp":  now.Add(15 * time.Minute).Unix(),
	}
	untypedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, untypedClaims)
	untypedTokenStr, err := untypedToken.SignedString([]byte(jwtCfg.Secret))
	if err != nil {
		t.Fatalf("failed to sign untyped token: %v", err)
	}

	_, err = authSvc.ValidateAccessToken(untypedTokenStr)
	if err != service.ErrInvalidToken {
		t.Fatalf("expected untyped token to be rejected with ErrInvalidToken, got: %v", err)
	}

	// 4. Token with invalid signing method (None / RSA instead of HS256)
	tamperedToken := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims)
	tamperedTokenStr, _ := tamperedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	_, err = authSvc.ValidateAccessToken(tamperedTokenStr)
	if err != service.ErrInvalidToken {
		t.Fatalf("expected unsigned/none-alg token to be rejected, got: %v", err)
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGoogleLogin_AudienceAndEmailVerifiedRejection(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	authCfg := config.AuthConfig{
		GoogleClientID: "expected-client-id.apps.googleusercontent.com",
	}
	authSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, nil,
		config.JWTConfig{Secret: "test-secret-32-chars-long-here-123", AccessExpiry: "15m", RefreshExpiry: "168h"},
		authCfg,
	)

	// 1. Audience mismatch: attacker minted Google token for a different OAuth app
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		mismatchedAudResp := `{"aud": "attacker-app.apps.googleusercontent.com", "email": "user@example.com", "sub": "g-12345", "email_verified": true}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(mismatchedAudResp)),
			Header:     make(http.Header),
		}, nil
	})

	_, _, err := authSvc.GoogleLogin(context.Background(), "fake-id-token", nil, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "audience mismatch") {
		t.Fatalf("expected error containing 'audience mismatch', got: %v", err)
	}

	// 2. Unverified email: Google ID token has email_verified == false
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		unverifiedEmailResp := `{"aud": "expected-client-id.apps.googleusercontent.com", "email": "unverified@example.com", "sub": "g-12345", "email_verified": false}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(unverifiedEmailResp)),
			Header:     make(http.Header),
		}, nil
	})

	_, _, err = authSvc.GoogleLogin(context.Background(), "fake-id-token", nil, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "email is not verified") {
		t.Fatalf("expected error containing 'email is not verified', got: %v", err)
	}

	// 3. Invalid Google token: tokeninfo endpoint returns HTTP 400
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error_description": "Invalid Value"}`)),
			Header:     make(http.Header),
		}, nil
	})

	_, _, err = authSvc.GoogleLogin(context.Background(), "invalid-token", nil, nil, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid Google token") {
		t.Fatalf("expected error containing 'invalid Google token', got: %v", err)
	}

	// 4. Missing GoogleClientID configuration (Firebase client): tokeninfo verification succeeds
	unconfiguredSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, nil,
		config.JWTConfig{Secret: "test-secret-32-chars-long-here-123", AccessExpiry: "15m", RefreshExpiry: "168h"},
		config.AuthConfig{}, // Empty GoogleClientID
	)
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		validResp := `{"aud": "some-id", "email": "valid@example.com", "sub": "g-123", "email_verified": true}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(validResp)),
			Header:     make(http.Header),
		}, nil
	})

	_, _, err = unconfiguredSvc.GoogleLogin(context.Background(), "any-token", nil, nil, nil, nil, nil)
	// Should not fail with audience error; fails only at userRepo (nil in this test)
	if err != nil && strings.Contains(err.Error(), "missing GoogleClientID") {
		t.Fatalf("did not expect error containing 'missing GoogleClientID', got: %v", err)
	}
}
