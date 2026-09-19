package handler

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/archaditya/bytevault/internal/monitoring"
)

// HealthHandler holds dependencies for health-related endpoints.
type HealthHandler struct {
	dbPool *pgxpool.Pool
}

// NewHealthHandler constructs a HealthHandler with DB pool dependencies.
func NewHealthHandler(dbPool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{dbPool: dbPool}
}

// Health handles GET /api/v1/health (Lightweight Docker/Compose health probe)
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status":  "healthy",
		"service": "PushPort",
	})
}

// DeepHealth handles GET /api/v1/health/deep (Comprehensive DB, Pool, and Telemetry probe)
func (h *HealthHandler) DeepHealth(c echo.Context) error {
	ctx := c.Request().Context()
	dbStatus := "healthy"
	var dbLatencyMs float64

	if h.dbPool != nil {
		start := time.Now()
		var pingVal int
		err := h.dbPool.QueryRow(ctx, "SELECT 1").Scan(&pingVal)
		dbLatencyMs = float64(time.Since(start).Microseconds()) / 1000.0
		if err != nil {
			dbStatus = "unreachable: " + err.Error()
		}
	}

	snapshot := monitoring.GlobalTelemetry.GetSnapshot(h.dbPool)

	overall := "healthy"
	if dbStatus != "healthy" || snapshot.ErrorRatePct > 10.0 {
		overall = "degraded"
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status":    overall,
		"service":   "PushPort",
		"timestamp": time.Now().UTC(),
		"database": map[string]any{
			"status":     dbStatus,
			"latency_ms": dbLatencyMs,
			"pool":       snapshot.Database,
		},
		"telemetry": snapshot,
	})
}