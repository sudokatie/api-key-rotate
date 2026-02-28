package vercel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles Vercel API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new Vercel API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	// Vercel rate limit is 100/min, so reasonable backoff
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://api.vercel.com",
	}
}

// Project represents a Vercel project
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EnvVar represents a Vercel environment variable
type EnvVar struct {
	ID        string   `json:"id"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"` // production, preview, development
	Type      string   `json:"type"`   // plain, encrypted, secret
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

type projectsResponse struct {
	Projects   []Project `json:"projects"`
	Pagination struct {
		Next int `json:"next"`
	} `json:"pagination"`
}

type envsResponse struct {
	Envs []EnvVar `json:"envs"`
}

// ListProjects returns all projects
func (c *Client) ListProjects() ([]Project, error) {
	var allProjects []Project

	url := c.base + "/v9/projects?limit=100"
	for url != "" {
		req, err := c.newRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var result projectsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		allProjects = append(allProjects, result.Projects...)

		// Handle pagination
		if result.Pagination.Next > 0 {
			url = fmt.Sprintf("%s/v9/projects?limit=100&until=%d", c.base, result.Pagination.Next)
		} else {
			url = ""
		}
	}

	return allProjects, nil
}

// GetEnvVars returns all environment variables for a project
func (c *Client) GetEnvVars(projectID string) ([]EnvVar, error) {
	url := fmt.Sprintf("%s/v10/projects/%s/env", c.base, projectID)

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result envsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Envs, nil
}

// UpdateEnvVar updates an environment variable
func (c *Client) UpdateEnvVar(projectID, envID string, value string) error {
	url := fmt.Sprintf("%s/v10/projects/%s/env/%s", c.base, projectID, envID)

	body, _ := json.Marshal(map[string]string{"value": value})
	req, err := c.newRequest("PATCH", url, body)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update env var: %s", string(body))
	}

	return nil
}

// newRequest creates a new HTTP request with auth headers
func (c *Client) newRequest(method, url string, body []byte) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// Set GetBody for retry support
	if body != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}

	return req, nil
}
