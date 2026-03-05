package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Client handles Railway GraphQL API communication
type Client struct {
	token string
	http  *providers.RetryableHTTPClient
	base  string
}

// NewClient creates a new Railway API client
func NewClient(token string) *Client {
	cfg := providers.DefaultRetryConfig()
	cfg.InitialBackoff = 2 * time.Second

	return &Client{
		token: token,
		http:  providers.NewRetryableClient(30*time.Second, cfg),
		base:  "https://backboard.railway.app/graphql/v2",
	}
}

// Project represents a Railway project
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Environment represents a Railway environment
type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Variable represents a Railway environment variable
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GraphQL request/response types
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ListProjects returns all projects for the authenticated user
func (c *Client) ListProjects() ([]Project, error) {
	query := `
		query {
			me {
				projects {
					edges {
						node {
							id
							name
						}
					}
				}
			}
		}
	`

	resp, err := c.execute(query, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Me struct {
			Projects struct {
				Edges []struct {
					Node Project `json:"node"`
				} `json:"edges"`
			} `json:"projects"`
		} `json:"me"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}

	projects := make([]Project, len(result.Me.Projects.Edges))
	for i, edge := range result.Me.Projects.Edges {
		projects[i] = edge.Node
	}

	return projects, nil
}

// ListEnvironments returns all environments for a project
func (c *Client) ListEnvironments(projectID string) ([]Environment, error) {
	query := `
		query($projectId: String!) {
			project(id: $projectId) {
				environments {
					edges {
						node {
							id
							name
						}
					}
				}
			}
		}
	`

	vars := map[string]interface{}{"projectId": projectID}
	resp, err := c.execute(query, vars)
	if err != nil {
		return nil, err
	}

	var result struct {
		Project struct {
			Environments struct {
				Edges []struct {
					Node Environment `json:"node"`
				} `json:"edges"`
			} `json:"environments"`
		} `json:"project"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}

	envs := make([]Environment, len(result.Project.Environments.Edges))
	for i, edge := range result.Project.Environments.Edges {
		envs[i] = edge.Node
	}

	return envs, nil
}

// GetVariables returns all variables for a project environment
func (c *Client) GetVariables(projectID, environmentID string) (map[string]string, error) {
	query := `
		query($projectId: String!, $environmentId: String!) {
			variables(projectId: $projectId, environmentId: $environmentId)
		}
	`

	vars := map[string]interface{}{
		"projectId":     projectID,
		"environmentId": environmentID,
	}
	resp, err := c.execute(query, vars)
	if err != nil {
		return nil, err
	}

	var result struct {
		Variables map[string]string `json:"variables"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, err
	}

	return result.Variables, nil
}

// UpsertVariable creates or updates a variable
func (c *Client) UpsertVariable(projectID, environmentID, name, value string) error {
	query := `
		mutation($input: VariableUpsertInput!) {
			variableUpsert(input: $input)
		}
	`

	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"projectId":     projectID,
			"environmentId": environmentID,
			"name":          name,
			"value":         value,
		},
	}

	_, err := c.execute(query, vars)
	return err
}

// execute sends a GraphQL request
func (c *Client) execute(query string, variables map[string]interface{}) (*graphQLResponse, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.base, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// Set GetBody for retry support
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
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

	var result graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return &result, nil
}
