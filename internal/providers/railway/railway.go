package railway

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func init() {
	providers.Register(New())
}

// Provider implements the providers.Provider interface for Railway
type Provider struct {
	client *Client
}

// New creates a new Railway provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "railway"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("railway API token required")
	}
	p.client = NewClient(creds.Token)
	return nil
}

// Test verifies the provider credentials work
func (p *Provider) Test() error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}
	_, err := p.client.ListProjects()
	return err
}

// Find locates a key across all Railway projects and environments
func (p *Provider) Find(keyName string) ([]providers.Location, error) {
	if p.client == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	var locations []providers.Location

	projects, err := p.client.ListProjects()
	if err != nil {
		return nil, err
	}

	for _, proj := range projects {
		envs, err := p.client.ListEnvironments(proj.ID)
		if err != nil {
			continue // Skip projects we can't access
		}

		for _, env := range envs {
			vars, err := p.client.GetVariables(proj.ID, env.ID)
			if err != nil {
				continue // Skip environments we can't access
			}

			if value, ok := vars[keyName]; ok {
				locations = append(locations, providers.Location{
					Type:        "railway",
					Provider:    "railway",
					Path:        fmt.Sprintf("%s/%s/%s", proj.ID, env.ID, keyName),
					Project:     proj.Name,
					Environment: env.Name,
					Exists:      true,
					Value:       value,
				})
			}
		}
	}

	return locations, nil
}

// Update changes a key's value
func (p *Provider) Update(location providers.Location, newValue string) error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}

	// Path format: projectID/environmentID/varName
	parts := strings.SplitN(location.Path, "/", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid location path: %s", location.Path)
	}
	projectID, environmentID, varName := parts[0], parts[1], parts[2]

	return p.client.UpsertVariable(projectID, environmentID, varName, newValue)
}

// SupportsRollback returns true as Railway supports updating variables
func (p *Provider) SupportsRollback() bool {
	return true
}

// Rollback restores a key to its original value
func (p *Provider) Rollback(location providers.Location, originalValue string) error {
	return p.Update(location, originalValue)
}
