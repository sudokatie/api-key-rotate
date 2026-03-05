package supabase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles Supabase Management API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new Supabase API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://api.supabase.com",
	}
}

// Project represents a Supabase project
type Project struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Region         string `json:"region"`
	Status         string `json:"status"`
}

// Secret represents a Supabase project secret
type Secret struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ListProjects returns all projects for the authenticated user
func (c *Client) ListProjects() ([]Project, error) {
	req, err := http.NewRequest("GET", c.base+"/v1/projects", nil)
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

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// ListSecrets returns all secrets for a project
func (c *Client) ListSecrets(projectRef string) ([]Secret, error) {
	url := fmt.Sprintf("%s/v1/projects/%s/secrets", c.base, projectRef)
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

// UpsertSecrets creates or updates secrets (bulk operation)
func (c *Client) UpsertSecrets(projectRef string, secrets []Secret) error {
	url := fmt.Sprintf("%s/v1/projects/%s/secrets", c.base, projectRef)

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

// DeleteSecret removes a secret from a project
func (c *Client) DeleteSecret(projectRef, name string) error {
	url := fmt.Sprintf("%s/v1/projects/%s/secrets", c.base, projectRef)

	// DELETE uses a body with the secret names to delete
	body, err := json.Marshal([]string{name})
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
