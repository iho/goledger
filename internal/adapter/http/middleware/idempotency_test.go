package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeIdempotencyStore struct {
	checkAndSetFn func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error)
	updateFn      func(ctx context.Context, key string, response []byte, ttl time.Duration) error
}

func TestIdempotencyMiddleware_IgnoresStoreErrors(t *testing.T) {
	var called bool
	store := &fakeIdempotencyStore{
		checkAndSetFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
			return false, nil, context.DeadlineExceeded
		},
	}
	mw := NewIdempotencyMiddleware(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(`{}`))
	req.Header.Set(IdempotencyKeyHeader, "key-err")
	rr := httptest.NewRecorder()

	mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if called {
		t.Fatalf("handler should not be called when store errors")
	}

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestIdempotencyMiddleware_DoesNotCacheFailedResponses(t *testing.T) {
	var updated bool
	store := &fakeIdempotencyStore{
		checkAndSetFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
			return false, nil, nil
		},
		updateFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) error {
			updated = true
			return nil
		},
	}
	mw := NewIdempotencyMiddleware(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(`{}`))
	req.Header.Set(IdempotencyKeyHeader, "key-fail")
	rr := httptest.NewRecorder()

	mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})).ServeHTTP(rr, req)

	if updated {
		t.Fatalf("expected error responses not to be cached")
	}
}

func (f *fakeIdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	if f.checkAndSetFn != nil {
		return f.checkAndSetFn(ctx, key, response, ttl)
	}
	return false, nil, nil
}

func (f *fakeIdempotencyStore) Update(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, key, response, ttl)
	}
	return nil
}

func TestIdempotencyMiddleware_SkipsNonMutatingRequests(t *testing.T) {
	store := &fakeIdempotencyStore{}
	mw := NewIdempotencyMiddleware(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil)
	rr := httptest.NewRecorder()

	called := false
	mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected next handler to be called")
	}
}

func TestIdempotencyMiddleware_ReturnsCachedResponse(t *testing.T) {
	store := &fakeIdempotencyStore{
		checkAndSetFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
			return true, []byte(`{"cached":true}`), nil
		},
	}
	mw := NewIdempotencyMiddleware(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(`{}`))
	req.Header.Set(IdempotencyKeyHeader, "key-123")
	rr := httptest.NewRecorder()

	mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be called when cached response exists")
	})).ServeHTTP(rr, req)

	if rr.Header().Get("X-Idempotency-Replay") != "true" {
		t.Fatalf("expected X-Idempotency-Replay header to be set")
	}

	if got := rr.Body.String(); got != `{"cached":true}` {
		t.Fatalf("unexpected cached body: %s", got)
	}
}

func TestIdempotencyMiddleware_StoresSuccessfulResponse(t *testing.T) {
	var updatedBody []byte
	store := &fakeIdempotencyStore{
		checkAndSetFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
			return false, nil, nil
		},
		updateFn: func(ctx context.Context, key string, response []byte, ttl time.Duration) error {
			updatedBody = append([]byte(nil), response...)
			return nil
		},
	}
	mw := NewIdempotencyMiddleware(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewBufferString(`{}`))
	req.Header.Set(IdempotencyKeyHeader, "key-456")
	rr := httptest.NewRecorder()

	mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: %d", rr.Code)
	}

	if string(updatedBody) != `{"ok":true}` {
		t.Fatalf("expected cached body to be stored, got %s", string(updatedBody))
	}
}
