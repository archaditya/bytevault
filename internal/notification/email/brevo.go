package email

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
)

//go:embed template.html
var emailTemplateFS embed.FS

var emailTmpl *template.Template

func init() {
	emailTmpl = template.Must(template.ParseFS(emailTemplateFS, "template.html"))
}

// EmailData holds template variables for the unified email template.
type EmailData struct {
	Heading  string
	Message  string
	OTP      string
	OTPColor string
	Body     string
	Footer   string
}

// BrevoClient sends transactional emails via Brevo (Sendinblue) REST API v3.
type BrevoClient struct {
	apiKey      string
	senderName  string
	senderEmail string
	httpClient  *http.Client
}

func NewBrevoClient(cfg config.BrevoConfig) *BrevoClient {
	return &BrevoClient{
		apiKey:      cfg.APIKey,
		senderName:  cfg.SenderName,
		senderEmail: cfg.SenderEmail,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// brevoRequest is the JSON body for Brevo's POST /v3/smtp/email endpoint.
type brevoRequest struct {
	Sender      brevoContact   `json:"sender"`
	To          []brevoContact `json:"to"`
	Subject     string         `json:"subject"`
	HTMLContent string         `json:"htmlContent"`
}

type brevoContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// SendOTP sends an OTP verification email.
func (b *BrevoClient) SendOTP(ctx context.Context, toEmail, toName, otp string) error {
	var buf bytes.Buffer
	emailTmpl.Execute(&buf, EmailData{
		Heading:  "Verify your email",
		Message:  fmt.Sprintf("Hi %s, use this code to verify your PushPort account:", toName),
		OTP:      otp,
		OTPColor: "#a78bfa",
		Footer:   "This code expires in 10 minutes. If you didn't request this, ignore this email.",
	})
	return b.send(ctx, toEmail, toName, "PushPort — Verify your email", buf.String())
}


// SendPasswordReset sends a password reset OTP email.
func (b *BrevoClient) SendPasswordReset(ctx context.Context, toEmail, toName, otp string) error {
	var buf bytes.Buffer
	emailTmpl.Execute(&buf, EmailData{
		Heading:  "Reset your password",
		Message:  fmt.Sprintf("Hi %s, use this code to reset your PushPort password:", toName),
		OTP:      otp,
		OTPColor: "#fbbf24",
		Footer:   "This code expires in 10 minutes. If you didn't request this, ignore this email.",
	})
	return b.send(ctx, toEmail, toName, "PushPort — Reset your password", buf.String())
}

// SendGeneric sends a general notification email.
func (b *BrevoClient) SendGeneric(ctx context.Context, toEmail, toName, subject, htmlBody string) error {
	return b.send(ctx, toEmail, toName, subject, htmlBody)
}

// send makes the actual HTTP call to Brevo.
func (b *BrevoClient) send(ctx context.Context, toEmail, toName, subject, html string) error {
	payload := brevoRequest{
		Sender:      brevoContact{Name: b.senderName, Email: b.senderEmail},
		To:          []brevoContact{{Name: toName, Email: toEmail}},
		Subject:     subject,
		HTMLContent: html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal brevo request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create brevo request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", b.apiKey)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("brevo HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Log.Error().
			Int("status", resp.StatusCode).
			Str("body", string(respBody)).
			Msg("Brevo API error")
		return fmt.Errorf("brevo API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.Log.Info().Str("to", toEmail).Str("subject", subject).Msg("Email sent via Brevo")
	return nil
}
