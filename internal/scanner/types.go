package scanner

// EnvEntry represents a key-value pair found in an env file
type EnvEntry struct {
	Key      string
	Value    string
	FilePath string
	Line     int
	Quoted   bool   // Whether the value was quoted
	Comment  string // Any trailing comment
}

// ScanResult represents the result of scanning a directory
type ScanResult struct {
	Entries []EnvEntry
	Files   []string
	Errors  []error
}

// KeyPattern defines a pattern to match API keys
type KeyPattern struct {
	Name     string // Pattern name (e.g., "openai", "stripe")
	Pattern  string // Regex pattern
	Provider string // Associated provider
}
