package middleware

import (
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/iho/goledger/internal/domain"
)

// RequestMeta populates the request context with attribution data (request
// ID, client IP, user agent) so downstream use cases can stamp audit rows
// with it. Must run after chi's RequestID and RealIP middleware.
func RequestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := domain.RequestMeta{
			RequestID: chimiddleware.GetReqID(r.Context()),
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
		}
		ctx := domain.ContextWithRequestMeta(r.Context(), meta)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
