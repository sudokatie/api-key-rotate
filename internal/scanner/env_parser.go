package scanner

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

var lineRegex = regexp.MustCompile(`^(export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$`)

// ParseEnvFile parses a .env file and returns all entries
func ParseEnvFile(path string) ([]EnvEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	entries, err := parseReader(file)
	if err != nil {
		return nil, err
	}
	// Set FilePath for all entries
	for i := range entries {
		entries[i].FilePath = path
	}
	return entries, nil
}

// ParseEnvContent parses env content from a string
func ParseEnvContent(content string) ([]EnvEntry, error) {
	return parseReader(strings.NewReader(content))
}

func parseReader(r io.Reader) ([]EnvEntry, error) {
	var entries []EnvEntry
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()
		line := strings.TrimSpace(rawLine)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, ok := parseLine(line, lineNum, rawLine)
		if ok {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

func parseLine(line string, lineNum int, rawLine string) (EnvEntry, bool) {
	matches := lineRegex.FindStringSubmatch(line)
	if matches == nil {
		return EnvEntry{}, false
	}

	exported := strings.TrimSpace(matches[1]) != ""
	key := matches[2]
	valueRaw := matches[3]

	value, quoteStyle := parseValue(valueRaw)

	return EnvEntry{
		Key:        key,
		Value:      value,
		Line:       lineNum,
		QuoteStyle: quoteStyle,
		Exported:   exported,
		RawLine:    rawLine,
	}, true
}

func parseValue(raw string) (string, QuoteStyle) {
	raw = strings.TrimSpace(raw)

	if len(raw) == 0 {
		return "", QuoteNone
	}

	// Double quoted
	if strings.HasPrefix(raw, `"`) {
		end := findClosingQuote(raw[1:], '"')
		if end >= 0 {
			return raw[1 : end+1], QuoteDouble
		}
	}

	// Single quoted
	if strings.HasPrefix(raw, `'`) {
		end := findClosingQuote(raw[1:], '\'')
		if end >= 0 {
			return raw[1 : end+1], QuoteSingle
		}
	}

	// Unquoted - strip inline comment
	if idx := strings.Index(raw, " #"); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}

	return raw, QuoteNone
}

func findClosingQuote(s string, quote byte) int {
	escaped := false
	for i := 0; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		if s[i] == '\\' {
			escaped = true
			continue
		}
		if s[i] == quote {
			return i
		}
	}
	return -1
}
