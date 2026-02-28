package providers

import (
	"math"
	"math/rand"
	"net/http"
	"time"
)

// RetryConfig defines retry behavior for HTTP requests
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retries (default: 3)
	InitialBackoff  time.Duration // Initial backoff duration (default: 1s)
	MaxBackoff      time.Duration // Maximum backoff duration (default: 30s)
	BackoffFactor   float64       // Multiplier for each retry (default: 2.0)
	Jitter          float64       // Random jitter factor 0-1 (default: 0.1)
	RetryOn5xx      bool          // Retry on 5xx errors (default: true)
	RetryOnRateLimit bool         // Retry on 429 (default: true)
}

// DefaultRetryConfig returns sensible defaults per spec
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:       3,
		InitialBackoff:   1 * time.Second,
		MaxBackoff:       30 * time.Second,
		BackoffFactor:    2.0,
		Jitter:           0.1,
		RetryOn5xx:       true,
		RetryOnRateLimit: true,
	}
}

// RetryableHTTPClient wraps http.Client with retry logic
type RetryableHTTPClient struct {
	client *http.Client
	config RetryConfig
}

// NewRetryableClient creates a new client with retry support
func NewRetryableClient(timeout time.Duration, config RetryConfig) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		client: &http.Client{Timeout: timeout},
		config: config,
	}
}

// Do executes a request with retry logic
// The caller must close the response body on success
func (c *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Clone request for retry (body must be re-readable)
		clonedReq := cloneRequest(req)

		resp, err := c.client.Do(clonedReq)
		if err != nil {
			lastErr = err
			if attempt < c.config.MaxRetries {
				c.sleep(attempt)
				continue
			}
			return nil, lastErr
		}

		// Check if we should retry
		if c.shouldRetry(resp.StatusCode) && attempt < c.config.MaxRetries {
			resp.Body.Close()
			lastResp = resp
			c.sleep(attempt)
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

// shouldRetry determines if a status code should trigger a retry
func (c *RetryableHTTPClient) shouldRetry(statusCode int) bool {
	// Rate limit
	if statusCode == 429 && c.config.RetryOnRateLimit {
		return true
	}

	// 5xx server errors
	if statusCode >= 500 && statusCode < 600 && c.config.RetryOn5xx {
		return true
	}

	return false
}

// sleep calculates and performs the backoff sleep
func (c *RetryableHTTPClient) sleep(attempt int) {
	backoff := c.calculateBackoff(attempt)
	time.Sleep(backoff)
}

// calculateBackoff computes exponential backoff with jitter
func (c *RetryableHTTPClient) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initial * factor^attempt
	backoff := float64(c.config.InitialBackoff) * math.Pow(c.config.BackoffFactor, float64(attempt))

	// Cap at max
	if backoff > float64(c.config.MaxBackoff) {
		backoff = float64(c.config.MaxBackoff)
	}

	// Add jitter: +/- jitter%
	if c.config.Jitter > 0 {
		jitter := backoff * c.config.Jitter * (2*rand.Float64() - 1)
		backoff += jitter
	}

	return time.Duration(backoff)
}

// cloneRequest creates a copy of the request
// Note: For requests with bodies, the caller should use GetBody
func cloneRequest(req *http.Request) *http.Request {
	clone := req.Clone(req.Context())

	// If the original request has a GetBody function, use it to get a fresh body
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err == nil {
			clone.Body = body
		}
	}

	return clone
}
