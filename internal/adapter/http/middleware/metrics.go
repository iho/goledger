
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)
)

// Metrics middleware records HTTP metrics.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := &metricsRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)

		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type metricsRecorder struct {
	http.ResponseWriter

	statusCode int
}

func (r *metricsRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// normalizePath normalizes URL paths to avoid high cardinality.
func normalizePath(path string) string {
	// Normalize paths with IDs to reduce cardinality
	// /api/v1/accounts/01ABC123 -> /api/v1/accounts/:id
	switch {
	case len(path) > 20 && path[:17] == "/api/v1/accounts/":
		if len(path) > 17 && path[17] != '/' {
			suffix := ""
			for i := 17; i < len(path); i++ {
				if path[i] == '/' {
					suffix = path[i:]
					break
				}
			}

			return "/api/v1/accounts/:id" + suffix
		}

	case len(path) > 21 && path[:18] == "/api/v1/transfers/":
		if len(path) > 18 && path[18] != '/' {
			suffix := ""
			for i := 18; i < len(path); i++ {
				if path[i] == '/' {
					suffix = path[i:]
					break
				}
			}

			return "/api/v1/transfers/:id" + suffix
		}
	}

	return path
}
