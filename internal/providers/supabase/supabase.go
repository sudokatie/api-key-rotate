package supabase

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func init() {
	providers.Register(New())
}

// Provider implements the providers.Provider interface for Supabase
type Provider struct {
	client *Client
}

// New creates a new Supabase provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "supabase"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("supabase access token required")
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

// Find locates a key across all Supabase projects
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
		// Skip paused or inactive projects
		if proj.Status != "ACTIVE_HEALTHY" && proj.Status != "ACTIVE_UNHEALTHY" {
			continue
		}

		secrets, err := p.client.ListSecrets(proj.ID)
		if err != nil {
			continue // Skip projects we can't access
		}

		for _, secret := range secrets {
			if secret.Name == keyName {
				locations = append(locations, providers.Location{
					Type:        "supabase",
					Provider:    "supabase",
					Path:        fmt.Sprintf("%s/%s", proj.ID, keyName),
					Project:     proj.Name,
					Environment: proj.Region,
					Exists:      true,
					Value:       secret.Value,
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

	// Path format: projectRef/secretName
	parts := strings.SplitN(location.Path, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid location path: %s", location.Path)
	}
	projectRef, secretName := parts[0], parts[1]

	// Supabase upsert takes an array of secrets
	return p.client.UpsertSecrets(projectRef, []Secret{
		{Name: secretName, Value: newValue},
	})
}

// SupportsRollback returns true as Supabase supports updating secrets
func (p *Provider) SupportsRollback() bool {
	return true
}

// Rollback restores a key to its original value
func (p *Provider) Rollback(location providers.Location, originalValue string) error {
	return p.Update(location, originalValue)
}
