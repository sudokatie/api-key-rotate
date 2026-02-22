package github

import (
	"fmt"
	"strings"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// Provider implements the providers.Provider interface for GitHub
type Provider struct {
	client *Client
	orgs   []string
}

// New creates a new GitHub provider
func New() *Provider {
	return &Provider{}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "github"
}

// Configure sets up the provider with credentials
func (p *Provider) Configure(creds providers.Credentials) error {
	if creds.Token == "" {
		return fmt.Errorf("github token required")
	}
	p.client = NewClient(creds.Token)

	// Parse orgs from extra data
	if orgs, ok := creds.ExtraData["orgs"]; ok {
		p.orgs = strings.Split(orgs, ",")
	}

	return nil
}

// Test verifies the provider credentials work
func (p *Provider) Test() error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}
	_, err := p.client.ListUserRepos()
	return err
}

// Find locates a secret across all accessible repositories
func (p *Provider) Find(keyName string) ([]providers.Location, error) {
	if p.client == nil {
		return nil, fmt.Errorf("provider not configured")
	}

	var locations []providers.Location

	// Get user repos
	repos, err := p.client.ListUserRepos()
	if err != nil {
		return nil, err
	}

	// Add org repos
	for _, org := range p.orgs {
		orgRepos, err := p.client.ListOrgRepos(org)
		if err != nil {
			continue // Skip orgs we can't access
		}
		repos = append(repos, orgRepos...)
	}

	// Check each repo for the secret
	for _, repo := range repos {
		secrets, err := p.client.ListSecrets(repo.Owner.Login, repo.Name)
		if err != nil {
			continue // Skip repos we can't access
		}

		for _, secret := range secrets {
			if secret.Name == keyName {
				locations = append(locations, providers.Location{
					Type:     "github",
					Provider: "github",
					Path:     fmt.Sprintf("%s/%s", repo.FullName, secret.Name),
					Project:  repo.FullName,
					Exists:   true,
					// Note: GitHub secrets cannot be read, only written
					Value: "",
				})
			}
		}
	}

	return locations, nil
}

// Update changes a secret's value
func (p *Provider) Update(loc providers.Location, newValue string) error {
	if p.client == nil {
		return fmt.Errorf("provider not configured")
	}

	// Parse owner/repo/secret from path
	parts := strings.Split(loc.Path, "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid location path: %s", loc.Path)
	}

	owner, repo, secretName := parts[0], parts[1], parts[2]

	// Get public key for encryption
	pubKey, err := p.client.GetPublicKey(owner, repo)
	if err != nil {
		return fmt.Errorf("get public key: %w", err)
	}

	// Encrypt the secret
	encrypted, err := EncryptSecret(pubKey.Key, newValue)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	// Update the secret
	return p.client.UpdateSecret(owner, repo, secretName, encrypted, pubKey.KeyID)
}

// SupportsRollback returns false as GitHub secrets cannot be read
func (p *Provider) SupportsRollback() bool {
	return false
}

// Rollback is not supported for GitHub secrets
func (p *Provider) Rollback(loc providers.Location, originalValue string) error {
	return fmt.Errorf("github secrets do not support rollback - values cannot be read")
}

func init() {
	providers.Register(New())
}
