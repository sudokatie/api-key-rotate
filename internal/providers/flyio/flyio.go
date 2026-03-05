package flyio

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func init() {
	providers.Register(New())
}

// Provider implements the providers.Provider interface for Fly.io
type Provider struct {
	client *Client
}

// New creates a new Fly.io provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "flyio"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("fly.io access token required")
	}
	p.client = NewClient(creds.Token)
	return nil
}

// Test verifies the provider credentials work
func (p *Provider) Test() error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}
	_, err := p.client.ListApps()
	return err
}

// Find locates a key across all Fly.io apps
// Note: Fly.io does not expose secret values via API, so Value will be empty
func (p *Provider) Find(keyName string) ([]providers.Location, error) {
	if p.client == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	var locations []providers.Location

	apps, err := p.client.ListApps()
	if err != nil {
		return nil, err
	}

	for _, app := range apps {
		// Skip suspended apps
		if app.Status == "suspended" {
			continue
		}

		secrets, err := p.client.ListSecrets(app.Name)
		if err != nil {
			continue // Skip apps we can't access
		}

		for _, secret := range secrets {
			if secret.Name == keyName {
				locations = append(locations, providers.Location{
					Type:        "flyio",
					Provider:    "flyio",
					Path:        fmt.Sprintf("%s/%s", app.Name, keyName),
					Project:     app.Name,
					Environment: app.Organization.Slug,
					Exists:      true,
					Value:       "", // Fly.io doesn't expose secret values
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

	// Path format: appName/secretName
	parts := strings.SplitN(location.Path, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid location path: %s", location.Path)
	}
	appName, secretName := parts[0], parts[1]

	return p.client.SetSecrets(appName, map[string]string{secretName: newValue})
}

// SupportsRollback returns false because Fly.io doesn't expose secret values
// We can set new values but can't restore old ones without knowing them
func (p *Provider) SupportsRollback() bool {
	return false
}

// Rollback is not supported for Fly.io
func (p *Provider) Rollback(location providers.Location, originalValue string) error {
	// Rollback only works if we have the original value stored externally
	if originalValue == "" {
		return fmt.Errorf("fly.io does not support rollback: original value unknown")
	}
	return p.Update(location, originalValue)
}
