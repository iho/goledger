package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(pool *pgxpool.Pool, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		pool:        pool,
		redisClient: redisClient,
	}
}

// Liveness returns 200 if the service is alive.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readiness returns 200 if the service is ready to accept traffic.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check PostgreSQL
	if err := h.pool.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "postgres unhealthy", err.Error())
		return
	}

	// Check Redis
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis unhealthy", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ready",
		"postgres": "ok",
		"redis":    "ok",
	})
}
