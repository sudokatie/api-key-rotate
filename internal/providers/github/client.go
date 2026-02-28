package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles GitHub API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	// GitHub rate limit varies, use reasonable backoff
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://api.github.com",
	}
}

// Repo represents a GitHub repository
type Repo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// Secret represents a GitHub Actions secret
type Secret struct {
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PublicKey for encrypting secrets
type PublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

type secretsResponse struct {
	Secrets    []Secret `json:"secrets"`
	TotalCount int      `json:"total_count"`
}

// ListUserRepos returns all repos for the authenticated user
func (c *Client) ListUserRepos() ([]Repo, error) {
	var allRepos []Repo
	page := 1

	for {
		url := fmt.Sprintf("%s/user/repos?per_page=100&page=%d", c.base, page)
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

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

// ListOrgRepos returns all repos for an organization
func (c *Client) ListOrgRepos(org string) ([]Repo, error) {
	var allRepos []Repo
	page := 1

	for {
		url := fmt.Sprintf("%s/orgs/%s/repos?per_page=100&page=%d", c.base, org, page)
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

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

// ListSecrets returns all secrets for a repository
func (c *Client) ListSecrets(owner, repo string) ([]Secret, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/secrets", c.base, owner, repo)

	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 404 means no secrets or no access - return empty
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result secretsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Secrets, nil
}

// GetPublicKey retrieves the public key for encrypting secrets
func (c *Client) GetPublicKey(owner, repo string) (*PublicKey, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/secrets/public-key", c.base, owner, repo)

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

	var key PublicKey
	if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
		return nil, err
	}

	return &key, nil
}

// UpdateSecret creates or updates a secret
func (c *Client) UpdateSecret(owner, repo, secretName, encryptedValue, keyID string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/secrets/%s", c.base, owner, repo, secretName)

	body, _ := json.Marshal(map[string]string{
		"encrypted_value": encryptedValue,
		"key_id":          keyID,
	})

	req, err := c.newRequest("PUT", url, body)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 201 Created or 204 No Content are both success
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update secret: %s", string(body))
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
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Set GetBody for retry support
	if body != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}

	return req, nil
}
