package keyring

import (
	"github.com/zalando/go-keyring"
)

const serviceName = "api-key-rotate"

// Set stores a token for a provider
func Set(provider string, token string) error {
	return keyring.Set(serviceName, provider, token)
}

// Get retrieves a token for a provider
func Get(provider string) (string, error) {
	return keyring.Get(serviceName, provider)
}

// Delete removes a token for a provider
func Delete(provider string) error {
	return keyring.Delete(serviceName, provider)
}

// Exists checks if a token exists for a provider
func Exists(provider string) bool {
	_, err := Get(provider)
	return err == nil
}
