package vercel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Vercel API communication
type Client struct {
	token string
	http  *http.Client
	base  string
}

// NewClient creates a new Vercel API client
func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
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
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result projectsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.do(req)
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

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 429 {
		// Rate limited - wait and retry once
		resp.Body.Close()
		time.Sleep(2 * time.Second)
		return c.http.Do(req)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}
