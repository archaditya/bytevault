package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/middleware"
	"github.com/archaditya/bytevault/internal/service"
)

func generateTestToken(secret, userID, role string, perms map[string]bool) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":         userID,
		"role":        role,
		"permissions": perms,
		"typ":         "access",
		"iat":         now.Unix(),
		"exp":         now.Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func TestAuthMiddleware_QueryParamRestriction(t *testing.T) {
	jwtSecret := "super-secure-test-jwt-secret-32-chars-long"
	jwtCfg := config.JWTConfig{
		Secret:        jwtSecret,
		AccessExpiry:  "15m",
		RefreshExpiry: "168h",
	}
	authSvc := service.NewAuthService(nil, nil, nil, nil, nil, nil, jwtCfg, config.AuthConfig{})
	authMW := middleware.Auth(authSvc, nil, nil)

	handlerFunc := func(c echo.Context) error {
		return c.String(http.StatusOK, "success")
	}

	e := echo.New()

	// 1. General API endpoint with ?token= MUST be rejected (401)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/files?token="+generateTestToken(jwtSecret, "u1", "user", nil), nil)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	c1.SetPath("/api/v1/files")

	_ = authMW(handlerFunc)(c1)
	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 when token passed via query param on general endpoint, got %d", rec1.Code)
	}

	// 2. Download route with ?token= MUST be accepted (200)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/files/123/download?token="+generateTestToken(jwtSecret, "u1", "user", nil), nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/api/v1/files/:id/download")

	_ = authMW(handlerFunc)(c2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 when token passed on download route, got %d", rec2.Code)
	}

	// 3. Preview route with ?token= MUST be accepted (200)
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/files/123/preview?token="+generateTestToken(jwtSecret, "u1", "user", nil), nil)
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	c3.SetPath("/api/v1/files/:id/preview")

	_ = authMW(handlerFunc)(c3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 when token passed on preview route, got %d", rec3.Code)
	}
}

func TestRequirePermission_SuperAdminBypassAndChecks(t *testing.T) {
	e := echo.New()
	handlerFunc := func(c echo.Context) error {
		return c.String(http.StatusOK, "authorized")
	}

	reqPerm := middleware.RequirePermission("admin:telemetry")

	// 1. Super admin without explicit admin:telemetry permission MUST be permitted
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/telemetry", nil)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	c1.Set("role", "super_admin")
	c1.Set("permissions", map[string]bool{"admin:users": true})

	_ = reqPerm(handlerFunc)(c1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected super_admin to bypass permission check, got %d", rec1.Code)
	}

	// 2. Standard admin without admin:telemetry MUST be rejected (403)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/telemetry", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.Set("role", "admin")
	c2.Set("permissions", map[string]bool{"admin:users": true})

	_ = reqPerm(handlerFunc)(c2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected admin without permission to be forbidden (403), got %d", rec2.Code)
	}

	// 3. Role with admin:telemetry explicitly granted MUST be permitted
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/telemetry", nil)
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	c3.Set("role", "telemetry_viewer")
	c3.Set("permissions", map[string]bool{"admin:telemetry": true})

	_ = reqPerm(handlerFunc)(c3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected caller with permission to be authorized (200), got %d", rec3.Code)
	}
}

func TestAuthMiddleware_MFATokenRejectedOnProtectedEndpoint(t *testing.T) {
	jwtSecret := "super-secure-test-jwt-secret-32-chars-long"
	jwtCfg := config.JWTConfig{
		Secret:        jwtSecret,
		AccessExpiry:  "15m",
		RefreshExpiry: "168h",
	}
	authSvc := service.NewAuthService(nil, nil, nil, nil, nil, nil, jwtCfg, config.AuthConfig{})
	authMW := middleware.Auth(authSvc, nil, nil)

	handlerFunc := func(c echo.Context) error {
		return c.String(http.StatusOK, "should-not-reach-here")
	}

	e := echo.New()

	// Forge MFA verification token (carries "purpose": "mfa_verify" instead of "typ": "access")
	now := time.Now()
	mfaClaims := jwt.MapClaims{
		"sub":     "victim-user-123",
		"purpose": "mfa_verify",
		"iat":     now.Unix(),
		"exp":     now.Add(5 * time.Minute).Unix(),
	}
	mfaToken := jwt.NewWithClaims(jwt.SigningMethodHS256, mfaClaims)
	mfaTokenStr, err := mfaToken.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("failed to sign mfa token: %v", err)
	}

	// Calling /api/v1/auth/me with MFA token MUST result in HTTP 401 Unauthorized
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+mfaTokenStr)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/auth/me")

	_ = authMW(handlerFunc)(c)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 when using MFA token on protected endpoint, got %d", rec.Code)
	}
}

func TestIPExtractor_CFConnectingIPPriority(t *testing.T) {
	// Replicate the exact server IPExtractor configured in server.go
	ipExtractor := func(req *http.Request) string {
		if cfIP := req.Header.Get("CF-Connecting-IP"); cfIP != "" {
			return strings.TrimSpace(cfIP)
		}
		return echo.ExtractIPDirect()(req)
	}

	// 1. Spoofed X-Forwarded-For alongside trusted CF-Connecting-IP
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
	req1.Header.Set("CF-Connecting-IP", "203.0.113.195")
	req1.Header.Set("X-Forwarded-For", "198.51.100.1, 10.0.0.1")
	req1.RemoteAddr = "127.0.0.1:54321"

	extractedIP1 := ipExtractor(req1)
	if extractedIP1 != "203.0.113.195" {
		t.Fatalf("expected CF-Connecting-IP '203.0.113.195', got '%s'", extractedIP1)
	}

	// 2. Request without CF-Connecting-IP should fall back to direct connection (not trust spoofed XFF)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
	req2.Header.Set("X-Forwarded-For", "198.51.100.1")
	req2.RemoteAddr = "192.0.2.1:12345"

	extractedIP2 := ipExtractor(req2)
	if extractedIP2 != "192.0.2.1" {
		t.Fatalf("expected fallback to RemoteAddr '192.0.2.1', got '%s'", extractedIP2)
	}
}
