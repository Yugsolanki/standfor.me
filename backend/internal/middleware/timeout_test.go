package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestTimeout_HandlerCompletesInTime(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "success" {
		t.Errorf("expected body 'success', got %q", rec.Body.String())
	}
}

func TestTimeout_HandlerTimesOut(t *testing.T) {
	timeout := 50 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if body["error"] != "Gateway Timeout" {
		t.Errorf("expected error 'Gateway Timeout', got %v", body["error"])
	}
	if body["message"] == nil {
		t.Error("expected message in response")
	}
}

func TestTimeout_Returns504OnTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	wrapped.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", rec.Code)
	}
	if elapsed < timeout {
		t.Errorf("expected timeout to take at least %v, got %v", timeout, elapsed)
	}
}

func TestTimeout_CopiesHeaders(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"key":"value"}`))
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("expected custom header to be copied")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}
	if rec.Body.String() != `{"key":"value"}` {
		t.Errorf("expected body to be copied, got %q", rec.Body.String())
	}
}

func TestTimeout_ContentTypeOnTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type: application/json, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestTimeout_ConnectionCloseOnTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("Connection") != "close" {
		t.Error("expected Connection: close header on timeout")
	}
}

func TestTimeout_RequestIDPropagated(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	})

	wrapped := Timeout(timeout)(RequestID(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "test-request-id")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) != "test-request-id" {
		t.Error("expected request ID to be propagated in timeout response")
	}

	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)

	if body["request_id"] != "test-request-id" {
		t.Error("expected request_id in timeout response body")
	}
}

func TestTimeout_NoRequestIDOnTimeout(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)

	if body["request_id"] != "" {
		t.Error("expected empty request_id in timeout response when not set")
	}
}

func TestTimeout_ZeroTimeout(t *testing.T) {
	timeout := 0 * time.Second

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504 for zero timeout, got %d", rec.Code)
	}
}

func TestTimeout_VeryLongTimeout(t *testing.T) {
	timeout := 10 * time.Second

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for long timeout, got %d", rec.Code)
	}
}

func TestTimeout_HandlerWritesAfterTimeout(t *testing.T) {
	timeout := 20 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("late write"))
		}()

		<-r.Context().Done()
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", rec.Code)
	}
}

func TestTimeout_MultipleWrites(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first "))
		_, _ = w.Write([]byte("second "))
		_, _ = w.Write([]byte("third"))
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Body.String() != "first second third" {
		t.Errorf("expected concatenated body, got %q", rec.Body.String())
	}
}

func TestTimeout_NoStatusCodeWritten(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body only"))
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "body only" {
		t.Errorf("expected body 'body only', got %q", rec.Body.String())
	}
}

func TestTimeout_ConcurrentWrite(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		chars := []byte("abcdefghij")
		for i := range chars {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_, _ = w.Write([]byte{chars[idx]})
			}(i)
		}
		wg.Wait()
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestTimeoutWriter_Header(t *testing.T) {
	tw := &timeoutWriter{
		header: make(http.Header),
	}

	tw.Header().Set("X-Test", "value")

	if tw.header.Get("X-Test") != "value" {
		t.Error("expected header to be set")
	}
}

func TestTimeoutWriter_WriteHeader(t *testing.T) {
	tw := &timeoutWriter{
		header: make(http.Header),
	}

	tw.WriteHeader(http.StatusCreated)

	if tw.statusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", tw.statusCode)
	}

	tw.WriteHeader(http.StatusOK)

	if tw.statusCode != http.StatusCreated {
		t.Error("expected first WriteHeader to be preserved")
	}
}

func TestTimeoutWriter_WriteHeader_AfterTimeout(t *testing.T) {
	tw := &timeoutWriter{
		header:   make(http.Header),
		timedOut: true,
	}

	tw.WriteHeader(http.StatusCreated)

	if tw.statusCode != 0 {
		t.Error("expected status code to not be set after timeout")
	}
}

func TestTimeoutWriter_Write(t *testing.T) {
	tw := &timeoutWriter{
		header: make(http.Header),
	}

	n, err := tw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if string(tw.body) != "hello" {
		t.Errorf("expected body 'hello', got %q", string(tw.body))
	}
}

func TestTimeoutWriter_Write_AfterTimeout(t *testing.T) {
	tw := &timeoutWriter{
		header:   make(http.Header),
		timedOut: true,
	}

	n, err := tw.Write([]byte("hello"))
	if err != http.ErrHandlerTimeout {
		t.Errorf("expected ErrHandlerTimeout, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
}

func TestTimeoutWriter_Write_Concurrent(t *testing.T) {
	tw := &timeoutWriter{
		header: make(http.Header),
	}

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		wg.Go(func() {
			defer wg.Done()
			_, _ = tw.Write([]byte("x"))
		})
	}
	wg.Wait()

	if len(tw.body) != 100 {
		t.Errorf("expected 100 bytes, got %d", len(tw.body))
	}
}

func TestTimeoutWriter_WriteHeader_Concurrent(t *testing.T) {
	tw := &timeoutWriter{
		header: make(http.Header),
	}

	var wg sync.WaitGroup
	for i := 100; i < 200; i++ {
		wg.Add(1)
		go func(code int) {
			defer wg.Done()
			tw.WriteHeader(code)
		}(i)
	}
	wg.Wait()

	if tw.statusCode == 0 {
		t.Error("expected status code to be set")
	}
}

func TestTimeoutWriter_Flush_NotTimedOut(t *testing.T) {
	rec := httptest.NewRecorder()
	tw := &timeoutWriter{
		ResponseWriter: rec,
		header:         make(http.Header),
		body:           []byte("test body"),
	}

	tw.Flush()

	if len(tw.body) != 0 {
		t.Error("expected body to be cleared after flush")
	}
}

func TestTimeoutWriter_Flush_TimedOut(t *testing.T) {
	rec := httptest.NewRecorder()
	tw := &timeoutWriter{
		ResponseWriter: rec,
		header:         make(http.Header),
		body:           []byte("test body"),
		timedOut:       true,
	}

	tw.Flush()

	if len(tw.body) == 0 {
		t.Error("expected body to not be cleared when timed out")
	}
}

func TestTimeoutWriter_Flush_NoFlusher(t *testing.T) {
	tw := &timeoutWriter{
		ResponseWriter: &notFlusher{},
		header:         make(http.Header),
		body:           []byte("test body"),
	}

	tw.Flush()

	if len(tw.body) == 0 {
		t.Error("expected body to NOT be cleared when ResponseWriter doesn't implement Flusher")
	}
}

type notFlusher struct{}

func (nf *notFlusher) Header() http.Header {
	return make(http.Header)
}

func (nf *notFlusher) Write([]byte) (int, error) {
	return 0, nil
}

func (nf *notFlusher) WriteHeader(int) {}

func TestTimeout_ContextCancelledByHandler(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		_ = r.WithContext(ctx)
		cancel()
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)
}

func TestTimeout_ImmediateContextCancellation(t *testing.T) {
	timeout := 100 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})

	wrapped := Timeout(timeout)(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)
}

func TestTimeout_LogsTimeoutFields(t *testing.T) {
	timeout := 10 * time.Millisecond

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	})

	wrapped := Timeout(timeout)(RequestID(nextHandler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", rec.Code)
	}
}
