package monitoring

import (
	"testing"
	"time"
)

func TestTelemetryWindowedRPSAndErrors(t *testing.T) {
	tracker := NewTelemetryTracker()

	// Record 60 successful requests
	for i := 0; i < 60; i++ {
		tracker.Record("/api/v1/files", 200, 10*time.Millisecond)
	}

	// Record 4 server errors (500)
	for i := 0; i < 4; i++ {
		tracker.Record("/api/v1/files", 500, 20*time.Millisecond)
	}

	// Record 6 client errors (400)
	for i := 0; i < 6; i++ {
		tracker.Record("/api/v1/auth/login", 401, 5*time.Millisecond)
	}

	mockQueueDepths := map[string]int64{
		"email":            2,
		"media_processing": 5,
	}

	snapshot := tracker.GetSnapshot(nil, mockQueueDepths)

	if snapshot.TotalRequests != 70 {
		t.Errorf("expected 70 total requests, got %d", snapshot.TotalRequests)
	}

	if snapshot.WindowRequests != 70 {
		t.Errorf("expected 70 window requests, got %d", snapshot.WindowRequests)
	}

	if snapshot.Success2xx != 60 {
		t.Errorf("expected 60 2xx requests, got %d", snapshot.Success2xx)
	}

	if snapshot.ServerErr5xx != 4 {
		t.Errorf("expected 4 5xx errors, got %d", snapshot.ServerErr5xx)
	}

	if snapshot.CurrentRPS <= 0 {
		t.Errorf("expected positive CurrentRPS, got %f", snapshot.CurrentRPS)
	}

	// 4 / 70 = 5.71%
	if snapshot.ErrorRatePct < 5.0 || snapshot.ErrorRatePct > 6.0 {
		t.Errorf("expected error rate around 5.7%%, got %f", snapshot.ErrorRatePct)
	}

	if snapshot.QueueDepths["media_processing"] != 5 {
		t.Errorf("expected queue depth 5 for media_processing, got %d", snapshot.QueueDepths["media_processing"])
	}
}
