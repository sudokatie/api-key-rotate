package github

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// EncryptSecret encrypts a secret using GitHub's public key
func EncryptSecret(publicKeyB64 string, secretValue string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}

	if len(publicKeyBytes) != 32 {
		return "", fmt.Errorf("invalid public key length: %d", len(publicKeyBytes))
	}

	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(secretValue), &publicKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("encrypt: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
