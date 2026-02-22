package providers

// Provider defines the interface for key storage services
type Provider interface {
	Name() string
	Configure(creds Credentials) error
	Test() error
	Find(keyName string) ([]Location, error)
	Update(location Location, newValue string) error
	SupportsRollback() bool
	Rollback(location Location, originalValue string) error
}

// Location represents where a key is stored
type Location struct {
	Type        string // "local", "vercel", "github"
	Provider    string // Provider name
	Path        string // Full path or identifier
	Project     string // Project name (cloud providers)
	Environment string // Environment name (cloud providers)
	Exists      bool
	Value       string // Current value (for rollback)
}

// Credentials holds authentication for a provider
type Credentials struct {
	Token     string
	APIKey    string
	Username  string
	Password  string
	ExtraData map[string]string
}
