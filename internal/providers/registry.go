package providers

import (
	"sync"
)

var (
	registry = make(map[string]Provider)
	mu       sync.RWMutex
)

// Register adds a provider to the registry
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get retrieves a provider by name
func Get(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns all registered providers
func All() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Provider, 0, len(registry))
	for _, p := range registry {
		result = append(result, p)
	}
	return result
}

// Names returns names of all registered providers
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Clear removes all providers (for testing)
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Provider)
}
