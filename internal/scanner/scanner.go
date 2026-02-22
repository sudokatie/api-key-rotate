package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scan walks directories looking for env files
func Scan(paths []string, excludes []string, patterns []string) ([]string, error) {
	var results []string
	seen := make(map[string]bool)

	for _, root := range paths {
		expanded := expandHome(root)

		err := filepath.WalkDir(expanded, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if d.IsDir() {
				name := d.Name()
				for _, pattern := range excludes {
					if matched, _ := filepath.Match(pattern, name); matched {
						return filepath.SkipDir
					}
				}
				return nil
			}

			name := d.Name()
			for _, pattern := range patterns {
				if matched, _ := filepath.Match(pattern, name); matched {
					absPath, _ := filepath.Abs(path)
					if !seen[absPath] {
						seen[absPath] = true
						results = append(results, absPath)
					}
					break
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// FindKey searches files for a specific key
func FindKey(files []string, keyName string) ([]LocalLocation, error) {
	var locations []LocalLocation

	for _, file := range files {
		entries, err := ParseEnvFile(file)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.Key == keyName {
				locations = append(locations, LocalLocation{
					Path:  file,
					Line:  entry.Line,
					Value: entry.Value,
				})
			}
		}
	}

	return locations, nil
}

// FindAllKeys returns all unique key names from scanned files
func FindAllKeys(files []string) ([]string, error) {
	seen := make(map[string]bool)
	var keys []string

	for _, file := range files {
		entries, err := ParseEnvFile(file)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !seen[entry.Key] {
				seen[entry.Key] = true
				keys = append(keys, entry.Key)
			}
		}
	}

	return keys, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
