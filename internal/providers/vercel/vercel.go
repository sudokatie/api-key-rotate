package vercel

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Provider implements the providers.Provider interface for Vercel
type Provider struct {
	client *Client
}

// New creates a new Vercel provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "vercel"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("vercel token required")
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

// Find locates a key across all Vercel projects
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
		envVars, err := p.client.GetEnvVars(proj.ID)
		if err != nil {
			continue // Skip projects we can't access
		}

		for _, env := range envVars {
			if env.Key == keyName {
				locations = append(locations, providers.Location{
					Type:        "vercel",
					Provider:    "vercel",
					Path:        fmt.Sprintf("%s/%s", proj.ID, env.ID),
					Project:     proj.Name,
					Environment: strings.Join(env.Target, ","),
					Exists:      true,
					Value:       env.Value,
				})
			}
		}
	}

	return locations, nil
}

// Update changes a key's value
func (p *Provider) Update(loc providers.Location, newValue string) error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}

	parts := strings.Split(loc.Path, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid location path: %s", loc.Path)
	}

	projectID, envID := parts[0], parts[1]
	return p.client.UpdateEnvVar(projectID, envID, newValue)
}

// SupportsRollback returns true as Vercel supports updating values
func (p *Provider) SupportsRollback() bool {
	return true
}

// Rollback restores the original value
func (p *Provider) Rollback(loc providers.Location, originalValue string) error {
	return p.Update(loc, originalValue)
}

func init() {
	providers.Register(New())
}
