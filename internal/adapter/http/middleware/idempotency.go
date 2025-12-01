package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/iho/goledger/internal/usecase"
)

const (
	// IdempotencyKeyHeader is the header name for idempotency keys.
	IdempotencyKeyHeader = "Idempotency-Key"
	idempotencyTTL       = 24 * time.Hour
)

// IdempotencyMiddleware handles request idempotency using Redis.
type IdempotencyMiddleware struct {
	store usecase.IdempotencyStore
}

// NewIdempotencyMiddleware creates a new IdempotencyMiddleware.
func NewIdempotencyMiddleware(store usecase.IdempotencyStore) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{store: store}
}

// Wrap wraps an http.Handler with idempotency checking.
func (m *IdempotencyMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to mutating requests
		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if we have a cached response
		exists, cachedResponse, err := m.store.CheckAndSet(r.Context(), key, nil, idempotencyTTL)
		if err != nil {
			http.Error(w, "idempotency check failed", http.StatusInternalServerError)
			return
		}

		if exists && cachedResponse != nil && string(cachedResponse) != "processing" {
			// Return cached response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Idempotency-Replay", "true")
			w.Write(cachedResponse)
			return
		}

		// Capture response
		recorder := &responseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(recorder, r)

		// Store response for future idempotent requests
		if recorder.statusCode >= 200 && recorder.statusCode < 300 {
			m.store.Update(r.Context(), key, recorder.body.Bytes(), idempotencyTTL)
		}
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
