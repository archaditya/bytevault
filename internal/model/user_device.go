package model

import "time"

// UserDevice stores FCM tokens for push notifications.
// Same pattern as UsersFcmTokens in win-win-api.
//
// Each device a user logs in from gets its own entry.
// When you want to send a push notification, you query all
// active devices for that user and send to each FCM token.
type UserDevice struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	FcmToken   string     `json:"fcm_token"`
	DeviceType string     `json:"device_type"` // "web", "android", "ios"
	DeviceID   *string    `json:"device_id"`
	IsActive   bool       `json:"is_active"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}