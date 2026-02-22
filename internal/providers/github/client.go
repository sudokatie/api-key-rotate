package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles GitHub API communication
type Client struct {
	token string
	http  *http.Client
	base  string
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
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
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.do(req)
		if err != nil {
			return nil, err
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
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.do(req)
		if err != nil {
			return nil, err
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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result secretsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Secrets, nil
}

// GetPublicKey retrieves the public key for encrypting secrets
func (c *Client) GetPublicKey(owner, repo string) (*PublicKey, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/secrets/public-key", c.base, owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.do(req)
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

func (c *Client) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 429 {
		// Rate limited - wait and retry
		resp.Body.Close()
		time.Sleep(5 * time.Second)
		return c.http.Do(req)
	}

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}
