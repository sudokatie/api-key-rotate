package scanner

// QuoteStyle indicates how a value was quoted
type QuoteStyle int

const (
	QuoteNone QuoteStyle = iota
	QuoteSingle
	QuoteDouble
)

// EnvEntry represents a key-value pair found in an env file
type EnvEntry struct {
	Key        string
	Value      string
	Line       int
	QuoteStyle QuoteStyle
	Exported   bool
	RawLine    string
	FilePath   string
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
