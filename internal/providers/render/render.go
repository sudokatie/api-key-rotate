package render

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func init() {
	providers.Register(New())
}

// Provider implements the providers.Provider interface for Render
type Provider struct {
	client *Client
}

// New creates a new Render provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "render"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("render API key required")
	}
	p.client = NewClient(creds.Token)
	return nil
}

// Test verifies the provider credentials work
func (p *Provider) Test() error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}
	_, err := p.client.ListServices()
	return err
}

// Find locates a key across all Render services
func (p *Provider) Find(keyName string) ([]providers.Location, error) {
	if p.client == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	var locations []providers.Location

	services, err := p.client.ListServices()
	if err != nil {
		return nil, err
	}

	for _, svc := range services {
		envVars, err := p.client.GetEnvVars(svc.ID)
		if err != nil {
			continue // Skip services we can't access
		}

		for _, ev := range envVars {
			if ev.Key == keyName {
				locations = append(locations, providers.Location{
					Type:        "render",
					Provider:    "render",
					Path:        fmt.Sprintf("%s/%s", svc.ID, keyName),
					Project:     svc.Name,
					Environment: svc.Type,
					Exists:      true,
					Value:       ev.Value,
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

	// Path format: serviceID/keyName
	parts := strings.SplitN(location.Path, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid location path: %s", location.Path)
	}
	serviceID, keyName := parts[0], parts[1]

	return p.client.UpdateEnvVar(serviceID, keyName, newValue)
}

// SupportsRollback returns true as Render supports updating env vars
func (p *Provider) SupportsRollback() bool {
	return true
}

// Rollback restores a key to its original value
func (p *Provider) Rollback(location providers.Location, originalValue string) error {
	return p.Update(location, originalValue)
}
