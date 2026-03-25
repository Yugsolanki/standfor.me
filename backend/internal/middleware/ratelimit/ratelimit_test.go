// internal/middleware/ratelimit/ratelimit_test.go
package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// testRedisClient creates a Redis client for testing.
// Uses REDIS_URL env var or defaults to localhost:6379.
func testRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   15, // Use DB 15 for tests to avoid conflicts
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}

	// Clean test database
	client.FlushDB(ctx)

	t.Cleanup(func() {
		client.FlushDB(context.Background())
		err := client.Close()
		if err != nil {
			t.Errorf("failed to close redis client: %v", err)
		}
	})

	return client
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNew_Validation(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "nil client",
			config:  Config{Limit: 10, Window: time.Minute},
			wantErr: true,
		},
		{
			name:    "zero limit",
			config:  Config{Limit: 0, Window: time.Minute},
			wantErr: true,
		},
		{
			name:    "negative limit",
			config:  Config{Limit: -1, Window: time.Minute},
			wantErr: true,
		},
		{
			name:    "zero window",
			config:  Config{Limit: 10, Window: 0},
			wantErr: true,
		},
		{
			name:    "valid sliding window config",
			config:  Config{Limit: 10, Window: time.Minute, Strategy: SlidingWindow},
			wantErr: false,
		},
		{
			name:    "valid fixed window config",
			config:  Config{Limit: 100, Window: time.Hour, Strategy: FixedWindow},
			wantErr: false,
		},
		{
			name:    "valid token bucket config",
			config:  Config{Limit: 50, Window: time.Minute, Strategy: TokenBucket, BurstLimit: 10},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c *redis.Client
			if tt.name != "nil client" {
				c = client
			}
			_, err := New(c, tt.config, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlidingWindow_BasicRateLimit(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    5,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_sw",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// First 5 requests should succeed
	for i := range 5 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}

		remaining := rec.Header().Get(HeaderRateLimitRemaining)
		expected := fmt.Sprintf("%d", 5-i-1)
		if remaining != expected {
			t.Errorf("request %d: expected remaining=%s, got %s", i+1, expected, remaining)
		}
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("6th request: expected 429, got %d", rec.Code)
	}

	// Check response body
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["error"] != "rate limit exceeded" {
		t.Errorf("expected error='rate limit exceeded', got %v", body["error"])
	}

	// Verify Retry-After header is set
	if rec.Header().Get(HeaderRetryAfter) == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestFixedWindow_BasicRateLimit(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    3,
		Window:   10 * time.Second,
		Strategy: FixedWindow,
		Prefix:   "test_fw",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 3 requests should succeed
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: expected 429, got %d", rec.Code)
	}
}

func TestTokenBucket_BasicRateLimit(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:      10,          // 10 tokens per window
		Window:     time.Minute, // refill window
		Strategy:   TokenBucket,
		BurstLimit: 3, // bucket capacity = 3
		Prefix:     "test_tb",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 3 requests should succeed (bucket capacity is 3)
	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 4th request should be rate limited (bucket empty)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("4th request: expected 429, got %d", rec.Code)
	}
}

func TestDifferentClients_IndependentLimits(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    2,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_independent",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Client A: 2 requests
	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("client A request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Client A: should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.100:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("client A 3rd request: expected 429, got %d", rec.Code)
	}

	// Client B: should still be allowed
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.200:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("client B request: expected 200, got %d", rec.Code)
	}
}

func TestSkipFunc(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    1,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_skip",
		SkipFunc: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the limit
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should succeed")
	}

	// Health check should bypass rate limiting
	for i := range 10 {
		req = httptest.NewRequest(http.MethodGet, "/health", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("health check %d should bypass rate limit, got %d", i+1, rec.Code)
		}
	}
}

func TestCustomExceededHandler(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	customCalled := false
	limiter, err := New(client, Config{
		Limit:    1,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_custom_handler",
		ExceededHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			customCalled = true
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("slow down!"))
		}),
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Trigger custom handler
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !customCalled {
		t.Error("custom exceeded handler was not called")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
	if rec.Body.String() != "slow down!" {
		t.Errorf("expected 'slow down!', got %s", rec.Body.String())
	}
}

func TestKeyByHeader(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    2,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_header_key",
		KeyFunc:  KeyByHeader("X-API-Key"),
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two different API keys should have independent limits
	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-API-Key", "key-alpha")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("key-alpha request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// key-alpha should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "key-alpha")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("key-alpha 3rd request: expected 429, got %d", rec.Code)
	}

	// key-beta should still work
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "key-beta")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("key-beta request: expected 200, got %d", rec.Code)
	}
}

func TestProxyHeaders(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:      2,
		Window:     time.Minute,
		Strategy:   SlidingWindow,
		Prefix:     "test_proxy",
		TrustProxy: true,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Requests from different real IPs through the same proxy should be independent
	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234" // Proxy address
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("client 1 request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Client 1 should be limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("client 1 3rd request: expected 429, got %d", rec.Code)
	}

	// Client 2 (different real IP) should still work
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.2")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("client 2 request: expected 200, got %d", rec.Code)
	}
}

func TestXRealIP(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:      1,
		Window:     time.Minute,
		Strategy:   SlidingWindow,
		Prefix:     "test_xrealip",
		TrustProxy: true,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// X-Real-IP takes precedence
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "198.51.100.1")
	req.Header.Set("X-Forwarded-For", "198.51.100.2")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first request should succeed")
	}

	// Same X-Real-IP should be limited
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "198.51.100.1")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}
}

func TestFallbackToAllow(t *testing.T) {
	// Use a Redis client pointing to a non-existent server
	badClient := redis.NewClient(&redis.Options{
		Addr:        "localhost:59999", // Unlikely to be a real Redis
		DialTimeout: 100 * time.Millisecond,
	})
	defer func(badClient *redis.Client) {
		err := badClient.Close()
		if err != nil {
			t.Errorf("failed to close redis client: %v", err)
		}
	}(badClient)

	logger := testLogger()

	t.Run("fallback_allow", func(t *testing.T) {
		limiter, err := New(badClient, Config{
			Limit:           10,
			Window:          time.Minute,
			Strategy:        SlidingWindow,
			Prefix:          "test_fallback_allow",
			FallbackToAllow: true,
		}, logger)
		if err != nil {
			t.Fatalf("failed to create limiter: %v", err)
		}

		handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 with fallback-to-allow, got %d", rec.Code)
		}
	})

	t.Run("fallback_deny", func(t *testing.T) {
		limiter, err := New(badClient, Config{
			Limit:           10,
			Window:          time.Minute,
			Strategy:        SlidingWindow,
			Prefix:          "test_fallback_deny",
			FallbackToAllow: false,
		}, logger)
		if err != nil {
			t.Fatalf("failed to create limiter: %v", err)
		}

		handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429 with fallback-to-deny, got %d", rec.Code)
		}
	})
}

func TestHeaders_Disabled(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	enabled := false
	limiter, err := New(client, Config{
		Limit:         5,
		Window:        time.Minute,
		Strategy:      SlidingWindow,
		Prefix:        "test_no_headers",
		EnableHeaders: &enabled,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(HeaderRateLimitLimit) != "" {
		t.Error("expected no rate limit headers when disabled")
	}
	if rec.Header().Get(HeaderRateLimitRemaining) != "" {
		t.Error("expected no rate limit remaining header when disabled")
	}
}

func TestConcurrentRequests(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limit := 50
	limiter, err := New(client, Config{
		Limit:    limit,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_concurrent",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var (
		allowed atomic.Int64
		denied  atomic.Int64
		wg      sync.WaitGroup
	)

	totalRequests := 100
	wg.Add(totalRequests)

	for range totalRequests {
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			switch rec.Code {
			case http.StatusOK:
				allowed.Add(1)
			case http.StatusTooManyRequests:
				denied.Add(1)
			}
		}()
	}

	wg.Wait()

	totalAllowed := int(allowed.Load())
	totalDenied := int(denied.Load())

	t.Logf("Concurrent test: allowed=%d, denied=%d (limit=%d, total=%d)",
		totalAllowed, totalDenied, limit, totalRequests)

	if totalAllowed > limit {
		t.Errorf("allowed %d requests, but limit is %d", totalAllowed, limit)
	}
	if totalAllowed+totalDenied != totalRequests {
		t.Errorf("total responses (%d) don't match total requests (%d)",
			totalAllowed+totalDenied, totalRequests)
	}
}

func TestChiRouterIntegration(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	// Global rate limiter
	globalLimiter, err := New(client, Config{
		Limit:    100,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_chi_global",
		SkipFunc: func(r *http.Request) bool {
			return r.URL.Path == "/health"
		},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create global limiter: %v", err)
	}

	// Strict route-specific limiter
	authLimiter, err := New(client, Config{
		Limit:    3,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_chi_auth",
		KeyFunc:  KeyByRoute(false),
	}, logger)
	if err != nil {
		t.Fatalf("failed to create auth limiter: %v", err)
	}

	r := chi.NewRouter()
	r.Use(globalLimiter.Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	})

	r.Group(func(r chi.Router) {
		r.Use(authLimiter.Handler)
		r.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("logged in"))
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	httpClient := &http.Client{Timeout: 5 * time.Second}

	// Health check should always work (skipped from rate limiting)
	for i := range 5 {
		resp, err := httpClient.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		err = resp.Body.Close()
		if err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("health check %d: expected 200, got %d", i+1, resp.StatusCode)
		}
	}

	// Auth endpoint should be limited to 3 requests
	for i := range 3 {
		resp, err := httpClient.Post(ts.URL+"/auth/login", "application/json", nil) // # govet
		if err != nil {
			t.Fatalf("auth request failed: %v", err)
		}
		err = resp.Body.Close()
		if err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("auth request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
	}

	// 4th auth request should be rate limited
	resp, err := httpClient.Post(ts.URL+"/auth/login", "application/json", nil)
	if err != nil {
		t.Fatalf("auth request failed: %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("4th auth request: expected 429, got %d", resp.StatusCode)
	}
}

func TestReset(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    2,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_reset",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	for range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 before reset, got %d", rec.Code)
	}

	// Reset the limit
	err = limiter.Reset(context.Background(), "ip:10.0.0.1")
	if err != nil {
		t.Fatalf("reset failed: %v", err)
	}

	// Should work again
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after reset, got %d", rec.Code)
	}
}

func TestStatus(t *testing.T) {
	client := testRedisClient(t)
	logger := testLogger()

	limiter, err := New(client, Config{
		Limit:    5,
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "test_status",
	}, logger)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 3 requests
	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.99:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Check status
	result, err := limiter.Status(context.Background(), "ip:10.0.0.99")
	if err != nil {
		t.Fatalf("status check failed: %v", err)
	}

	if !result.Allowed {
		t.Error("expected Allowed=true with 2 remaining")
	}
	if result.Remaining != 2 {
		t.Errorf("expected Remaining=2, got %d", result.Remaining)
	}
	if result.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", result.Limit)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		xff        string
		trustProxy bool
		wantIP     string
	}{
		{
			name:       "basic remote addr with port",
			remoteAddr: "192.168.1.1:12345",
			trustProxy: false,
			wantIP:     "192.168.1.1",
		},
		{
			name:       "ipv6 remote addr",
			remoteAddr: "[::1]:12345",
			trustProxy: false,
			wantIP:     "::1",
		},
		{
			name:       "xff trusted",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.1, 70.41.3.18, 150.172.238.178",
			trustProxy: true,
			wantIP:     "203.0.113.1",
		},
		{
			name:       "xff not trusted",
			remoteAddr: "10.0.0.1:1234",
			xff:        "203.0.113.1",
			trustProxy: false,
			wantIP:     "10.0.0.1",
		},
		{
			name:       "x-real-ip takes precedence",
			remoteAddr: "10.0.0.1:1234",
			xRealIP:    "198.51.100.1",
			xff:        "203.0.113.1",
			trustProxy: true,
			wantIP:     "198.51.100.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			got := extractIP(req, tt.trustProxy)
			if got != tt.wantIP {
				t.Errorf("extractIP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}

// BenchmarkSlidingWindow measures the performance of sliding window rate limiting.
func BenchmarkSlidingWindow(b *testing.B) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   15,
	})
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			b.Errorf("failed to close redis client: %v", err)
		}
	}(client)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}
	client.FlushDB(ctx)

	limiter, err := New(client, Config{
		Limit:    1000000, // High limit to avoid 429s during benchmark
		Window:   time.Minute,
		Strategy: SlidingWindow,
		Prefix:   "bench_sw",
	}, slog.Default())
	if err != nil {
		b.Fatalf("failed to create limiter: %v", err)
	}

	handler := limiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}
	})
}
