package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles Render API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new Render API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://api.render.com/v1",
	}
}

// Service represents a Render service
type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ServiceResponse wraps the service in the API response
type ServiceResponse struct {
	Service Service `json:"service"`
}

// EnvVar represents a Render environment variable
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Owner represents the owner info in list response
type Owner struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Type  string `json:"type"`
}

// ListServicesResponse is the response from listing services
type ListServicesResponse struct {
	Service Service `json:"service"`
	Cursor  string  `json:"cursor"`
}

// ListServices returns all services for the authenticated user
func (c *Client) ListServices() ([]Service, error) {
	var allServices []Service
	cursor := ""

	for {
		url := c.base + "/services?limit=100"
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		c.setHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if err := c.checkError(resp); err != nil {
			return nil, err
		}

		var responses []ListServicesResponse
		if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
			return nil, err
		}

		for _, r := range responses {
			allServices = append(allServices, r.Service)
		}

		// Check if there are more pages
		if len(responses) < 100 {
			break
		}
		if len(responses) > 0 {
			cursor = responses[len(responses)-1].Cursor
		}
		if cursor == "" {
			break
		}
	}

	return allServices, nil
}

// GetEnvVars returns all environment variables for a service
func (c *Client) GetEnvVars(serviceID string) ([]EnvVar, error) {
	url := fmt.Sprintf("%s/services/%s/env-vars", c.base, serviceID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkError(resp); err != nil {
		return nil, err
	}

	var envVars []struct {
		EnvVar EnvVar `json:"envVar"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envVars); err != nil {
		return nil, err
	}

	result := make([]EnvVar, len(envVars))
	for i, ev := range envVars {
		result[i] = ev.EnvVar
	}

	return result, nil
}

// UpdateEnvVar updates an environment variable for a service
func (c *Client) UpdateEnvVar(serviceID, key, value string) error {
	url := fmt.Sprintf("%s/services/%s/env-vars/%s", c.base, serviceID, key)

	body, err := json.Marshal(map[string]string{"value": value})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	// Set GetBody for retry support
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkError(resp)
}

// setHeaders adds authorization and common headers
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
}

// checkError returns an error if the response indicates failure
func (c *Client) checkError(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
