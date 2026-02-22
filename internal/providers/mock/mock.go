// Package mock provides a mock provider for testing
package mock

import (
	"fmt"

	"github.com/sudokatie/api-key-rotate/internal/providers"
)

// MockProvider is a configurable mock for testing
type MockProvider struct {
	name           string
	configureErr   error
	testErr        error
	findResult     []providers.Location
	findErr        error
	updateErr      error
	rollbackErr    error
	updateCalls    []UpdateCall
	rollbackCalls  []RollbackCall
}

// UpdateCall records an Update call
type UpdateCall struct {
	Location providers.Location
	NewValue string
}

// RollbackCall records a Rollback call
type RollbackCall struct {
	Location      providers.Location
	OriginalValue string
}

// New creates a new mock provider
func New(name string) *MockProvider {
	return &MockProvider{name: name}
}

// Name returns the provider name
func (m *MockProvider) Name() string {
	return m.name
}

// Configure configures the provider
func (m *MockProvider) Configure(creds providers.Credentials) error {
	return m.configureErr
}

// Test tests the connection
func (m *MockProvider) Test() error {
	return m.testErr
}

// Find finds locations for a key
func (m *MockProvider) Find(keyName string) ([]providers.Location, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.findResult, nil
}

// Update updates a location
func (m *MockProvider) Update(location providers.Location, newValue string) error {
	m.updateCalls = append(m.updateCalls, UpdateCall{location, newValue})
	return m.updateErr
}

// SupportsRollback returns true
func (m *MockProvider) SupportsRollback() bool {
	return true
}

// Rollback rolls back a location
func (m *MockProvider) Rollback(location providers.Location, originalValue string) error {
	m.rollbackCalls = append(m.rollbackCalls, RollbackCall{location, originalValue})
	return m.rollbackErr
}

// SetConfigureError sets the error for Configure
func (m *MockProvider) SetConfigureError(err error) {
	m.configureErr = err
}

// SetTestError sets the error for Test
func (m *MockProvider) SetTestError(err error) {
	m.testErr = err
}

// SetFindResult sets the result for Find
func (m *MockProvider) SetFindResult(locs []providers.Location) {
	m.findResult = locs
}

// SetFindError sets the error for Find
func (m *MockProvider) SetFindError(err error) {
	m.findErr = err
}

// SetUpdateError sets the error for Update
func (m *MockProvider) SetUpdateError(err error) {
	m.updateErr = err
}

// SetRollbackError sets the error for Rollback
func (m *MockProvider) SetRollbackError(err error) {
	m.rollbackErr = err
}

// UpdateCalls returns all Update calls made
func (m *MockProvider) UpdateCalls() []UpdateCall {
	return m.updateCalls
}

// RollbackCalls returns all Rollback calls made
func (m *MockProvider) RollbackCalls() []RollbackCall {
	return m.rollbackCalls
}

// Reset clears all call history
func (m *MockProvider) Reset() {
	m.updateCalls = nil
	m.rollbackCalls = nil
}

// FailingProvider returns a provider that fails on update
func FailingProvider(name string) *MockProvider {
	p := New(name)
	p.SetUpdateError(fmt.Errorf("simulated update failure"))
	return p
}
