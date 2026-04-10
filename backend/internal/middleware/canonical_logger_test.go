package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/pkg/logger"
)

func TestCanonicalLogger_BasicRequest(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestCanonicalLogger_AddsCanonicalFieldsToContext(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cf := logger.GetCanonicalFields(r.Context())
		cf.Set("test_key", "test_value")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)
}

func TestCanonicalLogger_CapturesStatusCode(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "http_status=404") {
		t.Error("expected log to contain http_status=404")
	}
}

func TestCanonicalLogger_CapturesRequestMethod(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "http_method=POST") {
		t.Error("expected log to contain http_method=POST")
	}
}

func TestCanonicalLogger_CapturesPath(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "http_path=/api/v1/users/123") {
		t.Error("expected log to contain http_path")
	}
}

func TestCanonicalLogger_CapturesQueryString(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/search?q=hello&page=2", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "http_query") {
		t.Error("expected log to contain http_query")
	}
}

func TestCanonicalLogger_CapturesUserAgent(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 TestBrowser")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "user_agent") {
		t.Error("expected log to contain user_agent")
	}
}

func TestCanonicalLogger_LogsErrorFor5xx(t *testing.T) {
	level := determineLogLevel(500, map[string]any{})
	if level != slog.LevelError {
		t.Errorf("expected LevelError for 500, got %v", level)
	}
}

func TestCanonicalLogger_LogsWarningFor4xx(t *testing.T) {
	level := determineLogLevel(400, map[string]any{})
	if level != slog.LevelWarn {
		t.Errorf("expected LevelWarn for 400, got %v", level)
	}
}

func TestCanonicalLogger_LogsInfoFor2xx(t *testing.T) {
	level := determineLogLevel(200, map[string]any{})
	if level != slog.LevelInfo {
		t.Errorf("expected LevelInfo for 200, got %v", level)
	}
}

func TestCanonicalLogger_CapturesBytesWritten(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "bytes_written=9") {
		t.Error("expected log to contain bytes_written=9")
	}
}

func TestCanonicalLogger_CapturesDuration(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusNotFound)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "duration") {
		t.Error("expected log to contain duration")
	}
}

func TestCanonicalLogger_SkipsHealthCheckPaths(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	paths := []string{"/healthz", "/ready"}
	for _, path := range paths {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		logOutput := buf.String()
		if logOutput != "" {
			t.Errorf("expected no log output for path %s, got %q", path, logOutput)
		}
	}
}

func TestCanonicalLogger_ResponseRecorder_WriteHeaderOnce(t *testing.T) {
	rec := newResponseRecorder(httptest.NewRecorder())

	rec.WriteHeader(http.StatusOK)
	rec.WriteHeader(http.StatusNotFound)

	if rec.statusCode != http.StatusOK {
		t.Errorf("expected status code 200, got %d", rec.statusCode)
	}
}

func TestCanonicalLogger_ResponseRecorder_WriteOnce(t *testing.T) {
	rec := newResponseRecorder(httptest.NewRecorder())

	_, _ = rec.Write([]byte("first"))
	_, _ = rec.Write([]byte("second"))

	if rec.bytesWritten != 11 {
		t.Errorf("expected bytesWritten 11, got %d", rec.bytesWritten)
	}
}

func TestCanonicalLogger_ResponseRecorder_Unwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := newResponseRecorder(inner)

	if rec.Unwrap() != inner {
		t.Error("expected Unwrap to return the original ResponseWriter")
	}
}

func TestDetermineLogLevel_5xx(t *testing.T) {
	level := determineLogLevel(500, map[string]any{})
	if level != slog.LevelError {
		t.Errorf("expected LevelError for 500, got %v", level)
	}

	level = determineLogLevel(503, map[string]any{})
	if level != slog.LevelError {
		t.Errorf("expected LevelError for 503, got %v", level)
	}
}

func TestDetermineLogLevel_4xx(t *testing.T) {
	level := determineLogLevel(400, map[string]any{})
	if level != slog.LevelWarn {
		t.Errorf("expected LevelWarn for 400, got %v", level)
	}

	level = determineLogLevel(404, map[string]any{})
	if level != slog.LevelWarn {
		t.Errorf("expected LevelWarn for 404, got %v", level)
	}
}

func TestDetermineLogLevel_2xx(t *testing.T) {
	level := determineLogLevel(200, map[string]any{})
	if level != slog.LevelInfo {
		t.Errorf("expected LevelInfo for 200, got %v", level)
	}

	level = determineLogLevel(201, map[string]any{})
	if level != slog.LevelInfo {
		t.Errorf("expected LevelInfo for 201, got %v", level)
	}
}

func TestDetermineLogLevel_Panic(t *testing.T) {
	level := determineLogLevel(200, map[string]any{"panic": true})
	if level != slog.LevelError {
		t.Errorf("expected LevelError when panic is present, got %v", level)
	}
}

func TestShouldSkipLog_Errors(t *testing.T) {
	if shouldSkipLog(500, slog.LevelError, nil) {
		t.Error("should not skip error logs")
	}
	if shouldSkipLog(400, slog.LevelWarn, nil) {
		t.Error("should not skip warning logs")
	}
}

func TestShouldSkipLog_Non2xx(t *testing.T) {
	if shouldSkipLog(404, slog.LevelInfo, map[string]any{"http_path": "/test"}) {
		t.Error("should not skip non-2xx responses")
	}
}

func TestShouldSkipLog_HealthCheckPaths(t *testing.T) {
	if !shouldSkipLog(200, slog.LevelInfo, map[string]any{"http_path": "/healthz"}) {
		t.Error("should skip health check path")
	}
	if !shouldSkipLog(200, slog.LevelInfo, map[string]any{"http_path": "/ready"}) {
		t.Error("should skip readiness path")
	}
}

func TestCanonicalLogger_WritesResponseBody(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]string{"status": "ok"}
		_ = json.NewEncoder(w).Encode(resp)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Body.String() == "" {
		t.Error("expected response body to be written")
	}
}

func TestCanonicalLogger_HandlesPanic(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	recovered := Recoverer(log)
	wrapped := CanonicalLogger(log)(recovered(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic") {
		t.Error("expected log to contain panic info")
	}
}

func TestCanonicalLogger_WithRequestID(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestID(CanonicalLogger(log)(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, "custom-request-id")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "custom-request-id" {
		t.Error("expected X-Request-ID to match custom request ID")
	}
}

func TestCanonicalLogger_IPExtraction(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "client_ip") {
		t.Error("expected log to contain client_ip")
	}
}

func TestCanonicalLogger_AllMethods(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions}

	for _, method := range methods {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		wrapped := CanonicalLogger(log)(nextHandler)

		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("method %s: expected status 404, got %d", method, rec.Code)
		}
	}
}

func TestShouldSkipLog_2xxWithSampling(t *testing.T) {
	for range 10 {
		result := shouldSkipLog(200, slog.LevelInfo, map[string]any{"http_path": "/api/test"})
		if result {
			break
		}
	}
}

func TestCanonicalLogger_ClientIPForwarded(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := CanonicalLogger(log)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "client_ip") {
		t.Error("expected log to contain client_ip")
	}
}
