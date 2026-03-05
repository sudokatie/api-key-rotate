package keyring

import (
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const serviceName = "api-key-rotate"

// envVarNames maps provider names to their environment variable names
var envVarNames = map[string]string{
	"vercel":   "VERCEL_TOKEN",
	"github":   "GITHUB_TOKEN",
	"railway":  "RAILWAY_TOKEN",
	"supabase": "SUPABASE_ACCESS_TOKEN",
}

// Set stores a token for a provider in the system keychain
func Set(provider string, token string) error {
	return keyring.Set(serviceName, provider, token)
}

// Get retrieves a token for a provider
// Checks environment variables first, then falls back to system keychain
func Get(provider string) (string, error) {
	// Check environment variable first
	if envVar, ok := envVarNames[strings.ToLower(provider)]; ok {
		if token := os.Getenv(envVar); token != "" {
			return token, nil
		}
	}

	// Fall back to keyring
	return keyring.Get(serviceName, provider)
}

// Delete removes a token for a provider from the system keychain
func Delete(provider string) error {
	return keyring.Delete(serviceName, provider)
}

// Exists checks if a token exists for a provider
// Returns true if found in environment or keychain
func Exists(provider string) bool {
	// Check environment variable first
	if envVar, ok := envVarNames[strings.ToLower(provider)]; ok {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// Check keyring
	_, err := keyring.Get(serviceName, provider)
	return err == nil
}

// Source returns where the token was found: "env", "keyring", or ""
func Source(provider string) string {
	// Check environment variable first
	if envVar, ok := envVarNames[strings.ToLower(provider)]; ok {
		if os.Getenv(envVar) != "" {
			return "env"
		}
	}

	// Check keyring
	_, err := keyring.Get(serviceName, provider)
	if err == nil {
		return "keyring"
	}

	return ""
}
