package mock

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

func TestMockProvider_Name(t *testing.T) {
	p := New("test-provider")
	assert.Equal(t, "test-provider", p.Name())
}

func TestMockProvider_Configure(t *testing.T) {
	p := New("test")

	// Success case
	err := p.Configure(providers.Credentials{Token: "token"})
	assert.NoError(t, err)

	// Error case
	p.SetConfigureError(errors.New("config failed"))
	err = p.Configure(providers.Credentials{})
	assert.Error(t, err)
}

func TestMockProvider_Test(t *testing.T) {
	p := New("test")

	err := p.Test()
	assert.NoError(t, err)

	p.SetTestError(errors.New("test failed"))
	err = p.Test()
	assert.Error(t, err)
}

func TestMockProvider_Find(t *testing.T) {
	p := New("test")

	// Empty result
	locs, err := p.Find("KEY")
	assert.NoError(t, err)
	assert.Empty(t, locs)

	// With result
	expected := []providers.Location{
		{Type: "mock", Path: "/test", Value: "secret"},
	}
	p.SetFindResult(expected)
	locs, err = p.Find("KEY")
	assert.NoError(t, err)
	assert.Equal(t, expected, locs)

	// Error case
	p.SetFindError(errors.New("find failed"))
	_, err = p.Find("KEY")
	assert.Error(t, err)
}

func TestMockProvider_Update(t *testing.T) {
	p := New("test")

	loc := providers.Location{Type: "mock", Path: "/test"}
	err := p.Update(loc, "new-value")
	assert.NoError(t, err)

	calls := p.UpdateCalls()
	assert.Len(t, calls, 1)
	assert.Equal(t, "new-value", calls[0].NewValue)

	// Error case
	p.SetUpdateError(errors.New("update failed"))
	err = p.Update(loc, "value")
	assert.Error(t, err)
}

func TestMockProvider_Rollback(t *testing.T) {
	p := New("test")

	loc := providers.Location{Type: "mock", Path: "/test"}
	err := p.Rollback(loc, "original")
	assert.NoError(t, err)

	calls := p.RollbackCalls()
	assert.Len(t, calls, 1)
	assert.Equal(t, "original", calls[0].OriginalValue)
}

func TestMockProvider_Reset(t *testing.T) {
	p := New("test")

	loc := providers.Location{}
	p.Update(loc, "v1")
	p.Update(loc, "v2")
	assert.Len(t, p.UpdateCalls(), 2)

	p.Reset()
	assert.Empty(t, p.UpdateCalls())
}

func TestMockProvider_SupportsRollback(t *testing.T) {
	p := New("test")
	assert.True(t, p.SupportsRollback())
}

func TestFailingProvider(t *testing.T) {
	p := FailingProvider("fail")
	err := p.Update(providers.Location{}, "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated")
}
