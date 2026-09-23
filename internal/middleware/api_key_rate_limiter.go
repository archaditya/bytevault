package middleware

import (
	"sync"
	"time"
)

type keyRateLimitEntry struct {
	count       int
	windowStart time.Time
}

type APIKeyRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*keyRateLimitEntry
}

func NewAPIKeyRateLimiter() *APIKeyRateLimiter {
	limiter := &APIKeyRateLimiter{
		entries: make(map[string]*keyRateLimitEntry),
	}
	// Periodically purge expired windows
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			limiter.cleanup()
		}
	}()
	return limiter
}

func (l *APIKeyRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-2 * time.Minute)
	for k, v := range l.entries {
		if v.windowStart.Before(cutoff) {
			delete(l.entries, k)
		}
	}
}

// Allow evaluates whether an incoming request from an API key is within the 1-minute window rate limit.
// Returns (allowed bool, remaining int, retryAfterSeconds int)
func (l *APIKeyRateLimiter) Allow(keyID string, limitPerMin int) (bool, int, int) {
	if limitPerMin <= 0 {
		limitPerMin = 60
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, exists := l.entries[keyID]
	if !exists || now.Sub(entry.windowStart) >= time.Minute {
		l.entries[keyID] = &keyRateLimitEntry{
			count:       1,
			windowStart: now,
		}
		return true, limitPerMin - 1, int(time.Minute.Seconds())
	}

	remainingTime := int((time.Minute - now.Sub(entry.windowStart)).Seconds())
	if remainingTime < 1 {
		remainingTime = 1
	}

	if entry.count >= limitPerMin {
		return false, 0, remainingTime
	}

	entry.count++
	remaining := limitPerMin - entry.count
	return true, remaining, remainingTime
}
