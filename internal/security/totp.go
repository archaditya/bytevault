package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	TOTPPeriod = 30
	TOTPDigits = 6
)

// GenerateTOTPSecret creates a random 16-character Base32 secret key for TOTP
func GenerateTOTPSecret() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// GenerateTOTPCode computes the 6-digit TOTP code for a given secret at time t
func GenerateTOTPCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	counter := uint64(t.Unix() / TOTPPeriod)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	code := (uint32(hash[offset]&0x7f)<<24 |
		uint32(hash[offset+1])<<16 |
		uint32(hash[offset+2])<<8 |
		uint32(hash[offset+3])) % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// VerifyTOTPCode validates a 6-digit code against the secret (±1 time step window)
func VerifyTOTPCode(secret, code string) bool {
	cleanCode := strings.TrimSpace(code)
	if len(cleanCode) != TOTPDigits {
		return false
	}

	now := time.Now()
	for _, offset := range []int64{-1, 0, 1} {
		t := now.Add(time.Duration(offset*TOTPPeriod) * time.Second)
		expected, err := GenerateTOTPCode(secret, t)
		if err == nil && expected == cleanCode {
			return true
		}
	}
	return false
}

// GenerateQRCodeURI returns the otpauth:// URI for authenticator apps (Google Authenticator, Authy)
func GenerateQRCodeURI(secret, userEmail string) string {
	issuer := "PushPort"
	label := fmt.Sprintf("%s:%s", issuer, userEmail)
	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s&period=30&digits=6",
		url.PathEscape(label),
		strings.ToUpper(strings.TrimSpace(secret)),
		url.QueryEscape(issuer),
	)
}
