package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
	"github.com/Yugsolanki/standfor-me/internal/pkg/requestid"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestRecoverer_NoPanic(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "success" {
		t.Errorf("expected body 'success', got %q", rec.Body.String())
	}
}

func TestRecoverer_Panic_String(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("intentional panic")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body["error"] != "Internal Server Error" {
		t.Errorf("expected error 'Internal Server Error', got %v", body["error"])
	}
	if body["message"] == nil {
		t.Error("expected message in response")
	}
	if body["request_id"] == nil {
		t.Error("expected request_id in response")
	}
}

func TestRecoverer_Panic_Error(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("error panic"))
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/error-panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestRecoverer_Panic_Integer(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(42)
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/int-panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestRecoverer_Panic_Nil(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/nil-panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestRecoverer_Panic_Struct(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(struct{ Msg string }{Msg: "struct panic"})
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/struct-panic", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestRecoverer_LogsPanic(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic for logging")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/log-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Error("expected log to contain 'panic recovered'")
	}
	if !strings.Contains(logOutput, "test panic for logging") {
		t.Error("expected log to contain panic value")
	}
	if !strings.Contains(logOutput, "request_id") {
		t.Error("expected log to contain request_id")
	}
}

func TestRecoverer_LogsStackTrace(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic with stack")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/stack-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "stack_trace") {
		t.Error("expected log to contain stack_trace")
	}
}

func TestRecoverer_AddsCanonicalFields(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cf := logger.NewCanonicalFields()
		ctx := logger.WithCanonicalFields(r.Context(), cf)
		_ = r.WithContext(ctx)
		panic("panic for canonical fields")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/canonical-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)
}

func TestRecoverer_RequestIDInResponse(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic for request ID")
	})

	wrapped := RequestID(Recoverer(log)(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/reqid-test", nil)
	req.Header.Set(requestid.RequestIDHeader, "custom-request-id")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get(requestid.RequestIDHeader) != "custom-request-id" {
		t.Error("expected request ID in response header")
	}

	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)

	if body["request_id"] != "custom-request-id" {
		t.Error("expected request_id in response body")
	}
}

func TestRecoverer_NoRequestID(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic without request ID")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/no-reqid-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get(requestid.RequestIDHeader) != "" {
		t.Error("expected empty request ID header when not set")
	}
}

func TestRecoverer_ContentType(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic for content type")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/content-type-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type: application/json, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestRecoverer_VariousHTTPMethods(t *testing.T) {
	log := testLogger()
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("panic")
		})

		wrapped := Recoverer(log)(nextHandler)

		req := httptest.NewRequest(method, "/", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("method %s: expected 500, got %d", method, rec.Code)
		}
	}
}

func TestRecoverer_TruncatesLargeStack(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	var recursiveFunc func(depth int)
	recursiveFunc = func(depth int) {
		if depth > 100 {
			panic("deep stack panic")
		}
		recursiveFunc(depth + 1)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recursiveFunc(0)
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/truncate-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if len(logOutput) > maxStackSize {
		t.Error("expected log output to be truncated to maxStackSize")
	}
}

func TestRecoverer_MultiplePanics(t *testing.T) {
	log := testLogger()

	for i := range 3 {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("multiple panic")
		})

		wrapped := Recoverer(log)(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("iteration %d: expected 500, got %d", i, rec.Code)
		}
	}
}

func TestRecoverer_WritesResponseBody(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic without writing first")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/body-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if body["error"] != "Internal Server Error" {
		t.Errorf("expected error message in body, got %v", body)
	}
}

func TestRecoverer_IoDiscardPool(t *testing.T) {
	log := testLogger()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("testing pool")
	})

	wrapped := Recoverer(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/pool-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)
}
