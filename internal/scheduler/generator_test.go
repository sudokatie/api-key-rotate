package scheduler

import (
	"encoding/hex"
	"testing"
)

func TestRandomGeneratorHex(t *testing.T) {
	g := NewRandomGenerator(32, "hex")

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Hex encoding of 32 bytes = 64 characters
	if len(value) != 64 {
		t.Errorf("expected 64 char hex string, got %d", len(value))
	}

	// Should be valid hex
	_, err = hex.DecodeString(value)
	if err != nil {
		t.Errorf("invalid hex: %v", err)
	}
}

func TestRandomGeneratorBase64(t *testing.T) {
	g := NewRandomGenerator(32, "base64")

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Base64 of 32 bytes with padding
	expectedLen := 44 // ceil(32 / 3) * 4
	if len(value) != expectedLen {
		t.Errorf("expected %d char base64 string, got %d", expectedLen, len(value))
	}
}

func TestRandomGeneratorBase64URL(t *testing.T) {
	g := NewRandomGenerator(32, "base64url")

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should not contain + or /
	for _, c := range value {
		if c == '+' || c == '/' {
			t.Errorf("base64url should not contain + or /, found in: %s", value)
			break
		}
	}
}

func TestRandomGeneratorDefaultLength(t *testing.T) {
	g := NewRandomGenerator(0, "hex")

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Default 32 bytes = 64 hex chars
	if len(value) != 64 {
		t.Errorf("expected default 64 char hex string, got %d", len(value))
	}
}

func TestRandomGeneratorUniqueness(t *testing.T) {
	g := NewRandomGenerator(32, "hex")

	values := make(map[string]bool)
	for i := 0; i < 100; i++ {
		value, err := g.Generate()
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		if values[value] {
			t.Error("generated duplicate value")
		}
		values[value] = true
	}
}

func TestStaticGenerator(t *testing.T) {
	g := NewStaticGenerator("fixed-value")

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if value != "fixed-value" {
		t.Errorf("expected 'fixed-value', got %q", value)
	}

	// Should return same value every time
	value2, _ := g.Generate()
	if value != value2 {
		t.Error("static generator should return same value")
	}
}

func TestPrefixedGenerator(t *testing.T) {
	inner := NewStaticGenerator("inner-value")
	g := NewPrefixedGenerator("prefix_", inner)

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "prefix_inner-value"
	if value != expected {
		t.Errorf("expected %q, got %q", expected, value)
	}
}

func TestPrefixedGeneratorWithRandom(t *testing.T) {
	inner := NewRandomGenerator(16, "hex")
	g := NewPrefixedGenerator("sk_", inner)

	value, err := g.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should start with prefix
	if len(value) < 3 || value[:3] != "sk_" {
		t.Errorf("expected to start with 'sk_', got %q", value)
	}

	// Should have correct total length: 3 + 32 (16 bytes hex)
	if len(value) != 35 {
		t.Errorf("expected 35 chars, got %d", len(value))
	}
}
