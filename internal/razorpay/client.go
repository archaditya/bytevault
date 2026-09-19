package razorpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.razorpay.com/v1"

// Client is a pure Go HTTP client for Razorpay Subscriptions & Payments API.
type Client struct {
	keyID         string
	keySecret     string
	webhookSecret string
	httpClient    *http.Client
}

// NewClient creates a new Razorpay API client.
func NewClient(keyID, keySecret, webhookSecret string) *Client {
	return &Client{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// PlanItem represents the item within a Razorpay plan creation request.
type PlanItem struct {
	Name        string `json:"name"`
	Amount      int    `json:"amount"` // in paise
	Currency    string `json:"currency"`
	Description string `json:"description,omitempty"`
}

// CreatePlanRequest payload for POST /v1/plans
type CreatePlanRequest struct {
	Period   string   `json:"period"`   // "daily", "weekly", "monthly", "yearly"
	Interval int      `json:"interval"` // e.g. 1
	Item     PlanItem `json:"item"`
	Notes    map[string]string `json:"notes,omitempty"`
}

// PlanResponse returned by Razorpay for plan endpoints.
type PlanResponse struct {
	ID       string   `json:"id"`
	Entity   string   `json:"entity"`
	Interval int      `json:"interval"`
	Period   string   `json:"period"`
	Item     PlanItem `json:"item"`
}

// CreateSubscriptionRequest payload for POST /v1/subscriptions
type CreateSubscriptionRequest struct {
	PlanID         string            `json:"plan_id"`
	TotalCount     int               `json:"total_count"` // e.g. 120 cycles
	Quantity       int               `json:"quantity"`
	CustomerNotify int               `json:"customer_notify"` // 1 = Razorpay notifies, 0 = silent
	CustomerID     string            `json:"customer_id,omitempty"`
	StartAt        int64             `json:"start_at,omitempty"`
	ExpireBy       int64             `json:"expire_by,omitempty"`
	Notes          map[string]string `json:"notes,omitempty"`
}

// SubscriptionResponse returned by Razorpay for subscription operations.
type SubscriptionResponse struct {
	ID                 string            `json:"id"`
	Entity             string            `json:"entity"`
	PlanID             string            `json:"plan_id"`
	CustomerID         string            `json:"customer_id"`
	Status             string            `json:"status"` // created, authenticated, active, pending, halted, cancelled, completed, paused, expired
	CurrentStart       int64             `json:"current_start"`
	CurrentEnd         int64             `json:"current_end"`
	EndedAt            int64             `json:"ended_at"`
	Quantity           int               `json:"quantity"`
	ChargeAt           int64             `json:"charge_at"`
	StartAt            int64             `json:"start_at"`
	EndAt              int64             `json:"end_at"`
	AuthAttempts       int               `json:"auth_attempts"`
	TotalCount         int               `json:"total_count"`
	PaidCount          int               `json:"paid_count"`
	RemainingCount     int               `json:"remaining_count"`
	ShortURL           string            `json:"short_url"`
	HasScheduledChange bool              `json:"has_scheduled_changes"`
	ScheduleChangeAt   string            `json:"schedule_change_at"`
	Notes              map[string]string `json:"notes"`
}

// PaymentResponse represents payment details from GET /v1/payments/:id
type PaymentResponse struct {
	ID          string            `json:"id"`
	Entity      string            `json:"entity"`
	Amount      int               `json:"amount"`
	Currency    string            `json:"currency"`
	Status      string            `json:"status"` // captured, failed, refunded
	OrderID     string            `json:"order_id"`
	InvoiceID   string            `json:"invoice_id"`
	Method      string            `json:"method"`
	Email       string            `json:"email"`
	Contact     string            `json:"contact"`
	Description string            `json:"description"`
	ErrorCode   string            `json:"error_code"`
	ErrorDesc   string            `json:"error_description"`
	CreatedAt   int64             `json:"created_at"`
	Notes       map[string]string `json:"notes"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}

	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("razorpay error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// CreatePlan creates a recurring billing plan in Razorpay.
func (c *Client) CreatePlan(ctx context.Context, req CreatePlanRequest) (*PlanResponse, error) {
	var resp PlanResponse
	if err := c.doRequest(ctx, http.MethodPost, "/plans", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateSubscription initiates a subscription for a given plan.
func (c *Client) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*SubscriptionResponse, error) {
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/subscriptions", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FetchSubscription retrieves subscription details by ID.
func (c *Client) FetchSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResponse, error) {
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodGet, "/subscriptions/"+subscriptionID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSubscription updates subscription plan (immediate or at cycle end).
// scheduleChangeAt can be "now" or "cycle_end".
func (c *Client) UpdateSubscription(ctx context.Context, subscriptionID, newPlanID, scheduleChangeAt string) (*SubscriptionResponse, error) {
	payload := map[string]interface{}{
		"plan_id":            newPlanID,
		"schedule_change_at": scheduleChangeAt,
	}
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodPatch, "/subscriptions/"+subscriptionID, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelSubscription cancels a subscription either immediately or at cycle end.
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string, cancelAtCycleEnd bool) (*SubscriptionResponse, error) {
	cancelAt := 0
	if cancelAtCycleEnd {
		cancelAt = 1
	}
	payload := map[string]interface{}{
		"cancel_at_cycle_end": cancelAt,
	}
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/subscriptions/"+subscriptionID+"/cancel", payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PauseSubscription pauses an active subscription.
func (c *Client) PauseSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResponse, error) {
	payload := map[string]interface{}{
		"pause_at": "now",
	}
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/subscriptions/"+subscriptionID+"/pause", payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResumeSubscription resumes a paused subscription.
func (c *Client) ResumeSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResponse, error) {
	payload := map[string]interface{}{
		"resume_at": "now",
	}
	var resp SubscriptionResponse
	if err := c.doRequest(ctx, http.MethodPost, "/subscriptions/"+subscriptionID+"/resume", payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// FetchPayment retrieves payment details by payment ID.
func (c *Client) FetchPayment(ctx context.Context, paymentID string) (*PaymentResponse, error) {
	var resp PaymentResponse
	if err := c.doRequest(ctx, http.MethodGet, "/payments/"+paymentID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// VerifyWebhookSignature verifies that the webhook payload was signed by Razorpay.
func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	if c.webhookSecret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// VerifySubscriptionPaymentSignature verifies the checkout modal signature:
// signature = HMAC-SHA256(payment_id + "|" + subscription_id, secret)
func (c *Client) VerifySubscriptionPaymentSignature(paymentID, subscriptionID, signature string) bool {
	if c.keySecret == "" || signature == "" {
		return false
	}
	data := paymentID + "|" + subscriptionID
	mac := hmac.New(sha256.New, []byte(c.keySecret))
	mac.Write([]byte(data))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

type CustomerResponse struct {
	ID      string `json:"id"`
	Entity  string `json:"entity"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Contact string `json:"contact"`
}

// CreateCustomer creates or registers a customer in Razorpay for recurring subscriptions and saved cards.
func (c *Client) CreateCustomer(ctx context.Context, name, email, contact string) (*CustomerResponse, error) {
	reqBody := map[string]string{
		"name":  name,
		"email": email,
	}
	if contact != "" {
		reqBody["contact"] = contact
	}
	var resp CustomerResponse
	if err := c.doRequest(ctx, http.MethodPost, "/customers", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

