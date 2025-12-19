package verbose

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEnableDisable tests the behavior of Enable and Disable functions.
//
// It verifies:
//   - Disable sets enabled state to false
//   - Enable sets enabled state to true
//   - IsEnabled returns correct state
func TestEnableDisable(t *testing.T) {
	// Reset state
	Disable()
	assert.False(t, IsEnabled())

	Enable()
	assert.True(t, IsEnabled())

	Disable()
	assert.False(t, IsEnabled())
}

// TestSetWriter tests the behavior of SetWriter.
//
// It verifies:
//   - Writer can be set and messages are written to it
//   - nil writer parameter is ignored
//   - Verbose messages include [DEBUG] prefix
func TestSetWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	Enable()
	Printf("test message")
	Disable()

	assert.Contains(t, buf.String(), "[DEBUG] test message")

	// Test nil writer is ignored
	SetWriter(nil)
	buf.Reset()
	Enable()
	Printf("another message")
	Disable()
	assert.Contains(t, buf.String(), "[DEBUG] another message")
}

// TestPrintf tests the behavior of Printf.
//
// It verifies:
//   - No output when verbose is disabled
//   - Formatted output appears when verbose is enabled
//   - Format string and arguments are properly interpolated
func TestPrintf(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	Printf("should not appear")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	Printf("test %s %d", "arg", 42)
	Disable()

	assert.Contains(t, buf.String(), "[DEBUG] test arg 42")
}

// TestInfo tests the behavior of Info.
//
// It verifies:
//   - No output when verbose is disabled
//   - Message appears with [DEBUG] prefix when enabled
func TestInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	Info("should not appear")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	Info("info message")
	Disable()

	assert.Contains(t, buf.String(), "[DEBUG] info message")
}

// TestInfof tests the behavior of Infof.
//
// It verifies:
//   - No output when verbose is disabled
//   - Formatted message appears when enabled
func TestInfof(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	Infof("should not %s", "appear")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	Infof("info %s %d", "formatted", 123)
	Disable()

	assert.Contains(t, buf.String(), "[DEBUG] info formatted 123")
}

func TestWithDocRef(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	WithDocRef("config", "should not appear")
	assert.Empty(t, buf.String())

	// Known topic
	Enable()
	WithDocRef("config", "config issue")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] config issue")
	assert.Contains(t, output, "Configuration")
	assert.Contains(t, output, "docs/configuration.md")

	// Unknown topic - just prints message
	buf.Reset()
	Enable()
	WithDocRef("unknown-topic", "unknown topic message")
	output = buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] unknown topic message")
	assert.NotContains(t, output, "📖")
}

func TestWithDocRefAllTopics(t *testing.T) {
	topics := []string{"config", "rules", "lock", "outdated", "update", "groups", "cli", "architecture"}

	for _, topic := range topics {
		buf := &bytes.Buffer{}
		SetWriter(buf)
		Enable()
		WithDocRef(topic, "test message")
		Disable()

		assert.Contains(t, buf.String(), "[DEBUG] test message", "topic: %s", topic)
		assert.Contains(t, buf.String(), "📖", "topic: %s should have doc reference", topic)
	}
}

func TestConfigHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	ConfigHelp("npm", "issue", "solution")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	ConfigHelp("npm", "missing field", "add the field")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Rule 'npm': missing field")
	assert.Contains(t, output, "Solution: add the field")
	assert.Contains(t, output, "docs/configuration.md#rules")
}

func TestUnsupportedHelp(t *testing.T) {
	testCases := []struct {
		feature  string
		expected string
	}{
		{"lock", "lock_files:"},
		{"installed", "lock_files:"},
		{"outdated", "outdated:"},
		{"versions", "outdated:"},
		{"update", "update:"},
	}

	for _, tc := range testCases {
		t.Run(tc.feature, func(t *testing.T) {
			buf := &bytes.Buffer{}
			SetWriter(buf)

			// When disabled, no output
			Disable()
			UnsupportedHelp("test-rule", tc.feature)
			assert.Empty(t, buf.String())

			// When enabled, output appears
			Enable()
			UnsupportedHelp("test-rule", tc.feature)
			output := buf.String()
			Disable()

			assert.Contains(t, output, "Rule 'test-rule' does not support")
			assert.Contains(t, output, tc.expected)
			assert.Contains(t, output, "docs/configuration.md#rules")
		})
	}
}

func TestCommandExec(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	CommandExec("npm install", "/path/to/dir")
	assert.Empty(t, buf.String())

	// When enabled at Debug level, output appears
	Enable()
	SetLevel(2) // Debug level
	CommandExec("npm install", "/path/to/dir")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Executing: npm install")
	assert.Contains(t, output, "Working dir: /path/to/dir")
}

func TestCommandResult(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	CommandResult("npm install", 0, "output")
	assert.Empty(t, buf.String())

	// Success case
	Enable()
	CommandResult("npm install", 0, "success output")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "Command succeeded: npm install")
	assert.Contains(t, output, "success output")

	// Failure case
	buf.Reset()
	Enable()
	CommandResult("npm install", 1, "error output")
	output = buf.String()
	Disable()

	assert.Contains(t, output, "Command failed (exit 1): npm install")
	assert.Contains(t, output, "error output")

	// Empty output
	buf.Reset()
	Enable()
	CommandResult("npm install", 0, "")
	output = buf.String()
	Disable()

	assert.Contains(t, output, "Command succeeded")
	assert.NotContains(t, output, "|")

	// Multi-line output (more than 5 lines)
	buf.Reset()
	Enable()
	multiLine := strings.Join([]string{"line1", "line2", "line3", "line4", "line5", "line6", "line7"}, "\n")
	CommandResult("npm install", 0, multiLine)
	output = buf.String()
	Disable()

	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "line3")
	assert.Contains(t, output, "more lines")
	assert.NotContains(t, output, "line6") // Should be truncated
}

func TestConfigLoaded(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	ConfigLoaded("/path/to/config.yml", []string{"default"})
	assert.Empty(t, buf.String())

	// With extends
	Enable()
	ConfigLoaded("/path/to/config.yml", []string{"default", "base"})
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Config loaded: /path/to/config.yml")
	assert.Contains(t, output, "Extends: [default base]")

	// Without extends
	buf.Reset()
	Enable()
	ConfigLoaded("/path/to/config.yml", nil)
	output = buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Config loaded: /path/to/config.yml")
	assert.NotContains(t, output, "Extends:")
}

func TestPackageFiltered(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	PackageFiltered("lodash", "ignored by config")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	PackageFiltered("lodash", "ignored by config")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Package 'lodash' filtered: ignored by config")
}

func TestVersionSelected(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	VersionSelected("lodash", "4.17.0", "4.17.21", "latest compatible")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	VersionSelected("lodash", "4.17.0", "4.17.21", "latest compatible")
	output := buf.String()
	Disable()

	assert.Contains(t, output, "[DEBUG] Version selected for 'lodash': 4.17.0 → 4.17.21 (latest compatible)")
}

func TestTruncate(t *testing.T) {
	// Short string - no truncation
	assert.Equal(t, "short", truncate("short", 10))

	// Exact length - no truncation
	assert.Equal(t, "exact", truncate("exact", 5))

	// Long string - truncated
	assert.Equal(t, "this is a l...", truncate("this is a long string", 14))

	// Very short maxLen
	assert.Equal(t, "...", truncate("test", 3))
}
