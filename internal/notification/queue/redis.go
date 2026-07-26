package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/archaditya/bytevault/internal/config"
	"github.com/archaditya/bytevault/internal/logger"
)

// Job types for the notification queue.
const (
	JobTypeEmail    = "email"
	JobTypePush     = "push"
	JobTypeInApp    = "in_app"
	JobTypeMediaProcess = "media_process"
)

// Priority levels.
const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"
)

// Delivery statuses.
const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
)

// Queue names.
const (
	QueueEmail = "bytevault:queue:email"
	QueuePush  = "bytevault:queue:push"
	QueueInApp = "bytevault:queue:in_app"
	QueueMediaProcessing = "bytevault:queue:media_processing"
)

// PubSub channel for real-time fan-out.
const ChannelNotifications = "bytevault:notifications"

// Job is a single unit of work pushed to a Redis queue.
type Job struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	UserID         string         `json:"user_id"`
	Payload        map[string]any `json:"payload"`
	Priority       string         `json:"priority"`
	DeliveryStatus string         `json:"delivery_status"`
	CreatedAt      time.Time      `json:"created_at"`
	Retries        int            `json:"retries"`
	MaxRetries     int            `json:"max_retries"`
	LastError      string         `json:"last_error,omitempty"`
}

// RedisQueue provides queue operations backed by Redis lists and pub/sub.
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue creates a Redis client and verifies the connection.
func NewRedisQueue(cfg config.RedisConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Log.Info().Str("addr", cfg.Addr()).Msg("Redis connected")
	return &RedisQueue{client: client}, nil
}

// Enqueue pushes a job to the appropriate queue and publishes a notification on pub/sub.
func (q *RedisQueue) Enqueue(ctx context.Context, job *Job) error {
	if job.Priority == "" {
		job.Priority = PriorityNormal
	}
	if job.DeliveryStatus == "" {
		job.DeliveryStatus = DeliveryPending
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Choose queue by job type + priority (high priority gets separate queue)
	queueName := QueueInApp
	switch job.Type {
	case JobTypeEmail:
		queueName = QueueEmail
	case JobTypePush:
		queueName = QueuePush
	}
	if job.Priority == PriorityHigh {
		queueName = queueName + ":high"
	}

	if err := q.client.LPush(ctx, queueName, data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	q.client.Publish(ctx, ChannelNotifications, data)
	return nil
}

// Dequeue blocks until a job is available on the given queue (BRPOP).
// timeout of 0 means block forever.
func (q *RedisQueue) Dequeue(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
	result, err := q.client.BRPop(ctx, timeout, queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // timeout, no job
		}
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	// result[0] = queue name, result[1] = data
	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// DequeueWithPriority checks the high-priority queue first, then falls back to normal.
func (q *RedisQueue) DequeueWithPriority(ctx context.Context, queueName string, timeout time.Duration) (*Job, error) {
	// Try high-priority first (non-blocking)
	result, err := q.client.RPop(ctx, queueName+":high").Result()
	if err == nil {
		var job Job
		if err := json.Unmarshal([]byte(result), &job); err != nil {
			return nil, fmt.Errorf("failed to unmarshal high-priority job: %w", err)
		}
		return &job, nil
	}

	// Fall back to normal queue (blocking)
	return q.Dequeue(ctx, queueName, timeout)
}

// EnqueueMediaJob pushes a thumbnail/media processing task to Redis queue
func (q *RedisQueue) EnqueueMediaJob(ctx context.Context, fileID, userID, storageKey, contentType string) error {
	job := &Job{
		ID:        fmt.Sprintf("media-%s-%d", fileID, time.Now().UnixNano()),
		Type:      JobTypeMediaProcess,
		UserID:    userID,
		Payload: map[string]any{
			"file_id":      fileID,
			"user_id":      userID,
			"storage_key":  storageKey,
			"content_type": contentType,
		},
		Priority:       PriorityNormal,
		DeliveryStatus: DeliveryPending,
		CreatedAt:      time.Now(),
		MaxRetries:     3,
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal media job: %w", err)
	}

	return q.client.RPush(ctx, QueueMediaProcessing, data).Err()
}

// DequeueMediaJob fetches the next media task from Redis queue (blocking with timeout)
func (q *RedisQueue) DequeueMediaJob(ctx context.Context, timeout time.Duration) (*Job, error) {
	res, err := q.client.BLPop(ctx, timeout, QueueMediaProcessing).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Queue empty
		}
		return nil, err
	}

	if len(res) < 2 {
		return nil, fmt.Errorf("invalid Redis BLPop payload")
	}

	var job Job
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal media job: %w", err)
	}

	return &job, nil
}

// Subscribe listens to the notification pub/sub channel.
func (q *RedisQueue) Subscribe(ctx context.Context) *redis.PubSub {
	return q.client.Subscribe(ctx, ChannelNotifications)
}

// Close shuts down the Redis client.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}


// StoreOTP saves an OTP code in Redis with a TTL. Key format: bytevault:otp:{email}:{purpose}
func (q *RedisQueue) StoreOTP(ctx context.Context, email, purpose, otp string, ttl time.Duration) error {
	key := fmt.Sprintf("bytevault:otp:%s:%s", email, purpose)
	data, _ := json.Marshal(map[string]any{
		"otp":      otp,
		"attempts": 0,
	})
	return q.client.Set(ctx, key, data, ttl).Err()
}

// VerifyOTP checks the OTP from Redis. Returns nil on success, error otherwise.
func (q *RedisQueue) VerifyOTP(ctx context.Context, email, purpose, otp string, maxAttempts int) error {
	key := fmt.Sprintf("bytevault:otp:%s:%s", email, purpose)
	val, err := q.client.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("OTP not found or expired")
	}

	var stored map[string]any
	if err := json.Unmarshal([]byte(val), &stored); err != nil {
		return fmt.Errorf("corrupted OTP data")
	}

	attempts := int(stored["attempts"].(float64))
	if attempts >= maxAttempts {
		q.client.Del(ctx, key)
		return fmt.Errorf("maximum validation attempts exceeded")
	}

	storedOTP, _ := stored["otp"].(string)
	if storedOTP != otp {
		stored["attempts"] = attempts + 1
		data, _ := json.Marshal(stored)
		ttl := q.client.TTL(ctx, key).Val()
		q.client.Set(ctx, key, data, ttl)
		return fmt.Errorf("invalid verification code")
	}

	// Valid — delete from Redis
	q.client.Del(ctx, key)
	return nil
}

// DeleteOTP removes an OTP key (e.g. after successful verification).
func (q *RedisQueue) DeleteOTP(ctx context.Context, email, purpose string) {
	key := fmt.Sprintf("bytevault:otp:%s:%s", email, purpose)
	q.client.Del(ctx, key)
}

