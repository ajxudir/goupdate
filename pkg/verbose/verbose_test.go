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
	CommandResult("npm install", 1, "output")
	assert.Empty(t, buf.String())

	// Success case - now silent (only failures are logged)
	Enable()
	CommandResult("npm install", 0, "success output")
	output := buf.String()
	Disable()

	assert.Empty(t, output, "Success should not produce output")

	// Failure case
	buf.Reset()
	Enable()
	CommandResult("npm install", 1, "error output")
	output = buf.String()
	Disable()

	assert.Contains(t, output, "Command failed (exit 1): npm install")
	assert.Contains(t, output, "error output")

	// Empty output on failure
	buf.Reset()
	Enable()
	CommandResult("npm install", 1, "")
	output = buf.String()
	Disable()

	assert.Contains(t, output, "Command failed")
	assert.NotContains(t, output, "|")

	// Multi-line output on failure (more than 5 lines should be truncated)
	buf.Reset()
	Enable()
	multiLine := strings.Join([]string{"line1", "line2", "line3", "line4", "line5", "line6", "line7"}, "\n")
	CommandResult("npm install", 1, multiLine)
	output = buf.String()
	Disable()

	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "line3")
	assert.Contains(t, output, "line4")
	assert.Contains(t, output, "line5")
	assert.NotContains(t, output, "line6") // Should be truncated
	assert.NotContains(t, output, "line7") // Should be truncated
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

func TestSuppressUnsuppress(t *testing.T) {
	// Start with known state
	Disable()
	Unsuppress()

	// Test Suppress
	assert.False(t, IsSuppressed())
	Suppress()
	assert.True(t, IsSuppressed())

	// Test Unsuppress
	Unsuppress()
	assert.False(t, IsSuppressed())
}

func TestSetLevelAndGetLevel(t *testing.T) {
	// Reset to known state
	Disable()

	tests := []struct {
		input    int
		expected Level
	}{
		{-1, LevelVerbose},
		{0, LevelVerbose},
		{1, LevelVerbose},
		{2, LevelDebug},
		{3, LevelTrace},
		{100, LevelTrace}, // anything > 3 is trace
	}

	for _, tt := range tests {
		SetLevel(tt.input)
		assert.Equal(t, tt.expected, GetLevel(), "SetLevel(%d) should set level to %d", tt.input, tt.expected)
	}
}

func TestAtLevel(t *testing.T) {
	// When disabled, AtLevel returns false
	Disable()
	SetLevel(3) // Trace level
	assert.False(t, AtLevel(LevelVerbose))

	// When enabled but suppressed, AtLevel returns false
	Enable()
	Suppress()
	assert.False(t, AtLevel(LevelVerbose))

	// When enabled and not suppressed
	Unsuppress()
	SetLevel(2) // Debug level
	assert.True(t, AtLevel(LevelVerbose), "Debug level should satisfy Verbose")
	assert.True(t, AtLevel(LevelDebug), "Debug level should satisfy Debug")
	assert.False(t, AtLevel(LevelTrace), "Debug level should not satisfy Trace")

	Disable()
}

func TestIsDebugIsTrace(t *testing.T) {
	// When disabled
	Disable()
	assert.False(t, IsDebug())
	assert.False(t, IsTrace())

	// When enabled at verbose level
	Enable()
	SetLevel(1)
	assert.False(t, IsDebug())
	assert.False(t, IsTrace())

	// When enabled at debug level
	SetLevel(2)
	assert.True(t, IsDebug())
	assert.False(t, IsTrace())

	// When enabled at trace level
	SetLevel(3)
	assert.True(t, IsDebug())
	assert.True(t, IsTrace())

	Disable()
}

func TestDebugfTracef(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	Debugf("debug message %d", 42)
	Tracef("trace message %s", "test")
	assert.Empty(t, buf.String())

	// When enabled, output appears
	Enable()
	Debugf("debug message %d", 42)
	assert.Contains(t, buf.String(), "[DEBUG] debug message 42")

	buf.Reset()
	Tracef("trace message %s", "test")
	assert.Contains(t, buf.String(), "[DEBUG] trace message test")

	Disable()
}

func TestVersionsExcluded(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	VersionsExcluded("lodash", []string{"4.17.20", "4.17.19"})
	assert.Empty(t, buf.String())

	// When empty list, no output even when enabled
	Enable()
	VersionsExcluded("lodash", []string{})
	assert.Empty(t, buf.String())

	// When enabled with versions
	VersionsExcluded("lodash", []string{"4.17.20", "4.17.19"})
	assert.Contains(t, buf.String(), "[DEBUG] Excluded versions for lodash")
	assert.Contains(t, buf.String(), "4.17.20")

	Disable()
}

func TestVersionsFiltered(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	// When disabled, no output
	Disable()
	VersionsFiltered("react", []string{"18.0.0", "17.0.2"})
	assert.Empty(t, buf.String())

	// When empty list, no output even when enabled
	Enable()
	VersionsFiltered("react", []string{})
	assert.Empty(t, buf.String())

	// When enabled with versions
	VersionsFiltered("react", []string{"18.0.0", "17.0.2"})
	assert.Contains(t, buf.String(), "[DEBUG] Newer versions for react")
	assert.Contains(t, buf.String(), "18.0.0")

	Disable()
}
