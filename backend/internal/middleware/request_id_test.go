package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID to be set in context")
		}
		if !isValidUUID(id) {
			t.Errorf("expected valid UUID, got %q", id)
		}
	})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("next handler was not called")
	}

	respID := rec.Header().Get(RequestIDHeader)
	if respID == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	expectedID := "existing-request-id-12345"

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != expectedID {
			t.Errorf("expected %q, got %q", expectedID, id)
		}
	})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, expectedID)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	respID := rec.Header().Get(RequestIDHeader)
	if respID != expectedID {
		t.Errorf("expected response header %q, got %q", expectedID, respID)
	}
}

func TestRequestID_EmptyHeaderGeneratesNew(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	respID := rec.Header().Get(RequestIDHeader)
	if respID == "" {
		t.Error("expected request ID to be generated for empty header")
	}
	if !isValidUUID(respID) {
		t.Errorf("expected valid UUID for empty header case, got %q", respID)
	}
}

func TestRequestID_SetsResponseHeader(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Error("expected X-Request-ID header to be set on response")
	}
}

func TestRequestID_ContextPropagation(t *testing.T) {
	var capturedID string

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
	})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "test-id-context-123")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if capturedID != "test-id-context-123" {
		t.Errorf("expected context to have request ID, got %q", capturedID)
	}
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("expected empty string for empty context, got %q", id)
	}
}

func TestGetRequestID_NonStringValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey{}, 12345)
	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("expected empty string for non-string value, got %q", id)
	}
}

func TestGetRequestID_NilValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey{}, nil)
	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("expected empty string for nil value, got %q", id)
	}
}

func TestRequestID_MultipleRequests(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := RequestID(nextHandler)

	ids := make(map[string]bool)

	for i := range 10 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		id := rec.Header().Get(RequestIDHeader)
		if id == "" {
			t.Errorf("request %d: expected request ID", i)
		}
		if ids[id] {
			t.Errorf("request %d: duplicate request ID %q", i, id)
		}
		ids[id] = true
	}
}

func TestRequestID_ChainedMiddleware(t *testing.T) {
	var (
		firstID  string
		secondID string
	)

	firstMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			firstID = GetRequestID(r.Context())
			next.ServeHTTP(w, r)
		})
	}

	secondMiddleware := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secondID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		})
	}

	handler := RequestID(firstMiddleware(secondMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if firstID == "" || secondID == "" {
		t.Error("both middlewares should see the request ID")
	}
	if firstID != secondID {
		t.Errorf("request IDs should match between middlewares, got %q and %q", firstID, secondID)
	}
}

func TestRequestID_VariousHTTPMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	wrapped := RequestID(nextHandler)

	for _, method := range methods {
		req := httptest.NewRequest(method, "/", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Header().Get(RequestIDHeader) == "" {
			t.Errorf("method %s: expected request ID header", method)
		}
	}
}

func TestRequestID_RequestWithBody(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		w.Header().Set("X-Handler-ID", id)
	})

	wrapped := RequestID(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(RequestIDHeader, "body-request-id")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Handler-ID") != "body-request-id" {
		t.Error("handler should see the same request ID")
	}
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
