package flyio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles Fly.io API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new Fly.io API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://api.fly.io/v1",
	}
}

// App represents a Fly.io application
type App struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Organization struct {
		Slug string `json:"slug"`
	} `json:"organization"`
	Status string `json:"status"`
}

// Secret represents a Fly.io secret (note: values are not returned by API)
type Secret struct {
	Name      string `json:"name"`
	Digest    string `json:"digest"`
	CreatedAt string `json:"createdAt"`
}

// ListApps returns all apps for the authenticated user
func (c *Client) ListApps() ([]App, error) {
	req, err := http.NewRequest("GET", c.base+"/apps", nil)
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

	var result struct {
		Apps []App `json:"apps"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Apps, nil
}

// ListSecrets returns all secret names for an app
// Note: Fly.io does not return secret values for security
func (c *Client) ListSecrets(appName string) ([]Secret, error) {
	url := fmt.Sprintf("%s/apps/%s/secrets", c.base, appName)
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

	var secrets []Secret
	if err := json.NewDecoder(resp.Body).Decode(&secrets); err != nil {
		return nil, err
	}

	return secrets, nil
}

// SetSecrets sets one or more secrets for an app
func (c *Client) SetSecrets(appName string, secrets map[string]string) error {
	url := fmt.Sprintf("%s/apps/%s/secrets", c.base, appName)

	body, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
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

// UnsetSecrets removes secrets from an app
func (c *Client) UnsetSecrets(appName string, keys []string) error {
	url := fmt.Sprintf("%s/apps/%s/secrets", c.base, appName)

	body, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", url, bytes.NewReader(body))
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
