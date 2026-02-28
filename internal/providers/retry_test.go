package providers

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 1*time.Second, cfg.InitialBackoff)
	assert.Equal(t, 30*time.Second, cfg.MaxBackoff)
	assert.Equal(t, 2.0, cfg.BackoffFactor)
	assert.Equal(t, 0.1, cfg.Jitter)
	assert.True(t, cfg.RetryOn5xx)
	assert.True(t, cfg.RetryOnRateLimit)
}

func TestRetryableClient_Success(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewRetryableClient(5*time.Second, DefaultRetryConfig())
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestRetryableClient_RetryOn429(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialBackoff = 10 * time.Millisecond // Fast for tests
	cfg.Jitter = 0

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount))
}

func TestRetryableClient_RetryOn5xx(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialBackoff = 10 * time.Millisecond
	cfg.Jitter = 0

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}

func TestRetryableClient_MaxRetriesExhausted(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.MaxRetries = 2
	cfg.InitialBackoff = 10 * time.Millisecond
	cfg.Jitter = 0

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// After exhausting retries, we get the last response
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Equal(t, int32(3), atomic.LoadInt32(&callCount)) // 1 initial + 2 retries
}

func TestRetryableClient_NoRetryOn4xx(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialBackoff = 10 * time.Millisecond

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount)) // No retry
}

func TestRetryableClient_Disabled5xxRetry(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.RetryOn5xx = false
	cfg.InitialBackoff = 10 * time.Millisecond

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount)) // No retry
}

func TestRetryableClient_DisabledRateLimitRetry(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.RetryOnRateLimit = false
	cfg.InitialBackoff = 10 * time.Millisecond

	client := NewRetryableClient(5*time.Second, cfg)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount)) // No retry
}

func TestCalculateBackoff(t *testing.T) {
	cfg := RetryConfig{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0, // Disable jitter for predictable tests
	}
	client := &RetryableHTTPClient{config: cfg}

	// Attempt 0: 1s * 2^0 = 1s
	assert.Equal(t, 1*time.Second, client.calculateBackoff(0))

	// Attempt 1: 1s * 2^1 = 2s
	assert.Equal(t, 2*time.Second, client.calculateBackoff(1))

	// Attempt 2: 1s * 2^2 = 4s
	assert.Equal(t, 4*time.Second, client.calculateBackoff(2))

	// Attempt 3: 1s * 2^3 = 8s
	assert.Equal(t, 8*time.Second, client.calculateBackoff(3))
}

func TestCalculateBackoff_MaxCap(t *testing.T) {
	cfg := RetryConfig{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     5 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0,
	}
	client := &RetryableHTTPClient{config: cfg}

	// Attempt 10: 1s * 2^10 = 1024s, but capped at 5s
	assert.Equal(t, 5*time.Second, client.calculateBackoff(10))
}

func TestCalculateBackoff_WithJitter(t *testing.T) {
	cfg := RetryConfig{
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         0.1, // 10% jitter
	}
	client := &RetryableHTTPClient{config: cfg}

	// Run multiple times to verify jitter adds variance
	results := make(map[time.Duration]bool)
	for i := 0; i < 20; i++ {
		backoff := client.calculateBackoff(0)
		results[backoff] = true
		// Should be within 10% of 1s: 900ms to 1100ms
		assert.GreaterOrEqual(t, backoff, 900*time.Millisecond)
		assert.LessOrEqual(t, backoff, 1100*time.Millisecond)
	}

	// With jitter, we should see some variance
	assert.Greater(t, len(results), 1, "jitter should produce varied results")
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		retryOnRateLimit bool
		retryOn5xx     bool
		expected       bool
	}{
		{"429 with rate limit enabled", 429, true, true, true},
		{"429 with rate limit disabled", 429, false, true, false},
		{"500 with 5xx enabled", 500, true, true, true},
		{"502 with 5xx enabled", 502, true, true, true},
		{"503 with 5xx enabled", 503, true, true, true},
		{"504 with 5xx enabled", 504, true, true, true},
		{"500 with 5xx disabled", 500, true, false, false},
		{"200 success", 200, true, true, false},
		{"400 bad request", 400, true, true, false},
		{"401 unauthorized", 401, true, true, false},
		{"403 forbidden", 403, true, true, false},
		{"404 not found", 404, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &RetryableHTTPClient{
				config: RetryConfig{
					RetryOnRateLimit: tt.retryOnRateLimit,
					RetryOn5xx:       tt.retryOn5xx,
				},
			}
			assert.Equal(t, tt.expected, client.shouldRetry(tt.statusCode))
		})
	}
}
