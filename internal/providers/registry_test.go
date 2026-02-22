package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockProvider implements Provider interface for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Configure(creds Credentials) error {
	return nil
}
func (m *mockProvider) Test() error                 { return nil }
func (m *mockProvider) Find(keyName string) ([]Location, error) {
	return nil, nil
}
func (m *mockProvider) Update(location Location, newValue string) error {
	return nil
}
func (m *mockProvider) SupportsRollback() bool { return false }
func (m *mockProvider) Rollback(location Location, originalValue string) error {
	return nil
}

func TestRegister(t *testing.T) {
	Clear()
	defer Clear()

	p := &mockProvider{name: "test-provider"}
	Register(p)

	got, ok := Get("test-provider")
	assert.True(t, ok)
	assert.Equal(t, "test-provider", got.Name())
}

func TestGetNotFound(t *testing.T) {
	Clear()
	defer Clear()

	_, ok := Get("nonexistent")
	assert.False(t, ok)
}

func TestAll(t *testing.T) {
	Clear()
	defer Clear()

	Register(&mockProvider{name: "provider1"})
	Register(&mockProvider{name: "provider2"})

	all := All()
	assert.Len(t, all, 2)
}

func TestNames(t *testing.T) {
	Clear()
	defer Clear()

	Register(&mockProvider{name: "vercel"})
	Register(&mockProvider{name: "github"})

	names := Names()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "vercel")
	assert.Contains(t, names, "github")
}

func TestClear(t *testing.T) {
	Clear()

	Register(&mockProvider{name: "test"})
	assert.Len(t, All(), 1)

	Clear()
	assert.Len(t, All(), 0)
}

func TestRegisterOverwrite(t *testing.T) {
	Clear()
	defer Clear()

	p1 := &mockProvider{name: "same-name"}
	p2 := &mockProvider{name: "same-name"}

	Register(p1)
	Register(p2)

	// Should have only one entry
	assert.Len(t, All(), 1)
}
