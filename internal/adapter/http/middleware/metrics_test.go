package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsMiddlewareRecordsRequest(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		path       string
		statusCode int
	}{
		{
			name:       "normalizes account path",
			method:     http.MethodGet,
			path:       "/api/v1/accounts/ABC123",
			statusCode: http.StatusTeapot,
		},
		{
			name:       "keeps non-matching path as-is",
			method:     http.MethodPost,
			path:       "/health",
			statusCode: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpRequestsTotal.Reset()
			httpRequestDuration.Reset()
			httpRequestsInFlight.Set(0)

			handlerCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(tc.statusCode)
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			Metrics(next).ServeHTTP(rr, req)

			if !handlerCalled {
				t.Fatalf("next handler was not invoked")
			}

			if got := testutil.ToFloat64(httpRequestsInFlight); got != 0 {
				t.Fatalf("expected in-flight gauge to return to 0, got %v", got)
			}

			normalized := normalizePath(tc.path)
			counter := httpRequestsTotal.WithLabelValues(tc.method, normalized, strconv.Itoa(tc.statusCode))
			if got := testutil.ToFloat64(counter); got != 1 {
				t.Fatalf("expected counter to be 1, got %v", got)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "account path without suffix",
			input:    "/api/v1/accounts/ABC123",
			expected: "/api/v1/accounts/:id",
		},
		{
			name:     "account path with suffix",
			input:    "/api/v1/accounts/ABC123/entries",
			expected: "/api/v1/accounts/:id/entries",
		},
		{
			name:     "transfer path",
			input:    "/api/v1/transfers/XYZ789",
			expected: "/api/v1/transfers/:id",
		},
		{
			name:     "non-matching path",
			input:    "/api/v1/health",
			expected: "/api/v1/health",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePath(tc.input); got != tc.expected {
				t.Fatalf("normalizePath(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}
