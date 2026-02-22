package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSimple(t *testing.T) {
	entries, err := ParseEnvContent("KEY=value")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "KEY", entries[0].Key)
	assert.Equal(t, "value", entries[0].Value)
	assert.Equal(t, QuoteNone, entries[0].QuoteStyle)
}

func TestParseDoubleQuoted(t *testing.T) {
	entries, err := ParseEnvContent(`KEY="hello world"`)
	require.NoError(t, err)
	assert.Equal(t, "hello world", entries[0].Value)
	assert.Equal(t, QuoteDouble, entries[0].QuoteStyle)
}

func TestParseSingleQuoted(t *testing.T) {
	entries, err := ParseEnvContent(`KEY='hello world'`)
	require.NoError(t, err)
	assert.Equal(t, "hello world", entries[0].Value)
	assert.Equal(t, QuoteSingle, entries[0].QuoteStyle)
}

func TestParseExported(t *testing.T) {
	entries, err := ParseEnvContent("export KEY=value")
	require.NoError(t, err)
	assert.True(t, entries[0].Exported)
}

func TestParseInlineComment(t *testing.T) {
	entries, err := ParseEnvContent("KEY=value # this is a comment")
	require.NoError(t, err)
	assert.Equal(t, "value", entries[0].Value)
}

func TestParseSkipsComments(t *testing.T) {
	entries, err := ParseEnvContent("# comment\nKEY=value")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 2, entries[0].Line)
}

func TestParseValueWithEquals(t *testing.T) {
	entries, err := ParseEnvContent("URL=postgres://user:pass@host/db?opt=val")
	require.NoError(t, err)
	assert.Equal(t, "postgres://user:pass@host/db?opt=val", entries[0].Value)
}

func TestParseEscapedQuotes(t *testing.T) {
	entries, err := ParseEnvContent(`KEY="hello \"world\""`)
	require.NoError(t, err)
	assert.Equal(t, `hello \"world\"`, entries[0].Value)
}

func TestParseEmptyValue(t *testing.T) {
	entries, err := ParseEnvContent("KEY=")
	require.NoError(t, err)
	assert.Equal(t, "", entries[0].Value)
}

func TestParseMultipleLines(t *testing.T) {
	content := `
# Database
DB_HOST=localhost
DB_PORT=5432

# API Keys
export API_KEY="secret123"
`
	entries, err := ParseEnvContent(content)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, "DB_HOST", entries[0].Key)
	assert.Equal(t, "DB_PORT", entries[1].Key)
	assert.Equal(t, "API_KEY", entries[2].Key)
}

func TestParseEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	content := "KEY1=value1\nKEY2=value2\n"
	err := os.WriteFile(envPath, []byte(content), 0644)
	require.NoError(t, err)

	entries, err := ParseEnvFile(envPath)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, envPath, entries[0].FilePath)
}

func TestParseEnvFileNotFound(t *testing.T) {
	_, err := ParseEnvFile("/nonexistent/path/.env")
	assert.Error(t, err)
}

func TestParseLineNumbers(t *testing.T) {
	content := "# line 1\n# line 2\nKEY=value"
	entries, err := ParseEnvContent(content)
	require.NoError(t, err)
	assert.Equal(t, 3, entries[0].Line)
}

func TestParseRawLine(t *testing.T) {
	content := "  KEY=value  "
	entries, err := ParseEnvContent(content)
	require.NoError(t, err)
	assert.Equal(t, "  KEY=value  ", entries[0].RawLine)
}
