package scanner

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"
)

// UpdateKey updates a key in an env file while preserving formatting
func UpdateKey(path string, keyName string, newValue string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	updated, err := updateKeyInContent(string(content), keyName, newValue)
	if err != nil {
		return err
	}

	tempPath := fmt.Sprintf("%s.tmp.%s", path, randomString(8))
	if err := os.WriteFile(tempPath, []byte(updated), 0600); err != nil {
		return err
	}

	return os.Rename(tempPath, path)
}

func updateKeyInContent(content string, keyName string, newValue string) (string, error) {
	lines := strings.Split(content, "\n")
	found := false

	for i, line := range lines {
		entry, ok := parseLine(strings.TrimSpace(line), i+1, line)
		if !ok || entry.Key != keyName {
			continue
		}

		found = true
		lines[i] = reconstructLine(entry, newValue)
	}

	if !found {
		return "", fmt.Errorf("key %s not found", keyName)
	}

	return strings.Join(lines, "\n"), nil
}

func reconstructLine(entry EnvEntry, newValue string) string {
	var sb strings.Builder

	if entry.Exported {
		sb.WriteString("export ")
	}

	sb.WriteString(entry.Key)
	sb.WriteString("=")

	switch entry.QuoteStyle {
	case QuoteSingle:
		sb.WriteString("'")
		sb.WriteString(newValue)
		sb.WriteString("'")
	case QuoteDouble:
		sb.WriteString(`"`)
		sb.WriteString(newValue)
		sb.WriteString(`"`)
	default:
		sb.WriteString(newValue)
	}

	return sb.String()
}

// BackupFile creates a backup of the file
func BackupFile(path string) (string, error) {
	backupPath := fmt.Sprintf("%s.bak.%d", path, time.Now().Unix())

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return "", err
	}

	return backupPath, nil
}

// RestoreBackup restores a backup file
func RestoreBackup(backupPath string, originalPath string) error {
	return os.Rename(backupPath, originalPath)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
