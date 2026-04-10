package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPayloadLimit_AllowsNonBodyMethods(t *testing.T) {
	mw := PayloadLimit(1024)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	methods := []string{http.MethodGet, http.MethodDelete, http.MethodHead}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("method %s: expected status 200, got %d", method, rec.Code)
		}
	}
}

func TestPayloadLimit_RejectsPostOverLimit(t *testing.T) {
	mw := PayloadLimit(10) // 10 bytes limit

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("this is more than 10 bytes"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type: application/json")
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if body["error"] != "Payload Too Large" {
		t.Errorf("expected error 'Payload Too Large', got %v", body["error"])
	}
	if body["message"] == nil {
		t.Error("expected message in response")
	}
}

func TestPayloadLimit_AllowsPostUnderLimit(t *testing.T) {
	mw := PayloadLimit(100) // 100 bytes limit

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("small body"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimit_PutMethod(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPut, "/test", strings.NewReader("this is more than 10 bytes"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}

func TestPayloadLimit_PatchMethod(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader("this is more than 10 bytes"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}

func TestPayloadLimit_ZeroContentLength(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.ContentLength = 0
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimit_NegativeContentLength(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.ContentLength = -1
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimit_ConnectionCloseHeader(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("this is more than 10 bytes"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Connection") != "close" {
		t.Error("expected Connection: close header")
	}
}

func TestPayloadLimitByRoute_MatchesPattern(t *testing.T) {
	routeLimits := map[string]int64{
		"/api/v1/upload": 100,
		"/api/v1/avatar": 50,
	}

	mw := PayloadLimitByRoute(routeLimits, 10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/test", strings.NewReader(strings.Repeat("x", 50)))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for 50 bytes under 100 limit, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/upload/test", strings.NewReader(strings.Repeat("x", 101)))
	rec = httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413 for 101 bytes over 100 limit, got %d", rec.Code)
	}
}

func TestPayloadLimitByRoute_UsesDefaultLimit(t *testing.T) {
	routeLimits := map[string]int64{
		"/api/v1/upload": 100,
	}

	mw := PayloadLimitByRoute(routeLimits, 10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/other/path", strings.NewReader("12345678901"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}

func TestPayloadLimitByRoute_FirstMatchWins(t *testing.T) {
	routeLimits := map[string]int64{
		"/api/v1/upload": 100,
		"/api/v1":        50,
	}

	mw := PayloadLimitByRoute(routeLimits, 10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/test", strings.NewReader(strings.Repeat("x", 80)))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimitByRoute_AllowsNonBodyMethods(t *testing.T) {
	routeLimits := map[string]int64{
		"/api/v1/upload": 100,
	}

	mw := PayloadLimitByRoute(routeLimits, 10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/upload/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{-1, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{2 * 1024 * 1024 * 1024, "2.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestPayloadLimit_RequestIDInResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := RequestID(PayloadLimit(10)(handler))

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("this is more than 10 bytes"))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if body["request_id"] == nil {
		t.Error("expected request_id in response")
	}
}

func TestPayloadLimit_AllowsEmptyBodyPost(t *testing.T) {
	mw := PayloadLimit(10)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.ContentLength = 0
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimit_AtExactLimit(t *testing.T) {
	mw := PayloadLimit(5)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("12345"))
	req.ContentLength = 5
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPayloadLimit_OneOverLimit(t *testing.T) {
	mw := PayloadLimit(5)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("123456"))
	req.ContentLength = 6
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", rec.Code)
	}
}
