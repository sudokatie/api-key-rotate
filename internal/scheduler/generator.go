package scheduler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// RandomGenerator generates random key values
type RandomGenerator struct {
	// Length is the number of bytes of randomness
	Length int
	// Format is the output format (hex, base64, base64url)
	Format string
}

// NewRandomGenerator creates a random key generator
func NewRandomGenerator(length int, format string) *RandomGenerator {
	if length <= 0 {
		length = 32
	}
	if format == "" {
		format = "hex"
	}
	return &RandomGenerator{
		Length: length,
		Format: format,
	}
}

// Generate creates a new random key value
func (g *RandomGenerator) Generate() (string, error) {
	bytes := make([]byte, g.Length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	switch g.Format {
	case "hex":
		return hex.EncodeToString(bytes), nil
	case "base64":
		return base64.StdEncoding.EncodeToString(bytes), nil
	case "base64url":
		return base64.URLEncoding.EncodeToString(bytes), nil
	default:
		return hex.EncodeToString(bytes), nil
	}
}

// StaticGenerator always returns the same value (useful for testing)
type StaticGenerator struct {
	Value string
}

// NewStaticGenerator creates a static key generator
func NewStaticGenerator(value string) *StaticGenerator {
	return &StaticGenerator{Value: value}
}

// Generate returns the static value
func (g *StaticGenerator) Generate() (string, error) {
	return g.Value, nil
}

// PrefixedGenerator adds a prefix to generated keys
type PrefixedGenerator struct {
	Prefix string
	Inner  KeyGenerator
}

// NewPrefixedGenerator creates a prefixed key generator
func NewPrefixedGenerator(prefix string, inner KeyGenerator) *PrefixedGenerator {
	return &PrefixedGenerator{
		Prefix: prefix,
		Inner:  inner,
	}
}

// Generate creates a prefixed key value
func (g *PrefixedGenerator) Generate() (string, error) {
	value, err := g.Inner.Generate()
	if err != nil {
		return "", err
	}
	return g.Prefix + value, nil
}
