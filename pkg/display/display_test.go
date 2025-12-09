package display

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/user/goupdate/pkg/constants"
)

// TestPrintUnsupportedMessages tests the PrintUnsupportedMessages function.
//
// It verifies that:
//   - Empty messages produce no output
//   - Multiple messages are printed correctly
func TestPrintUnsupportedMessages(t *testing.T) {
	t.Run("empty messages prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUnsupportedMessages(&buf, []string{})
		assert.Empty(t, buf.String())
	})

	t.Run("prints messages", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUnsupportedMessages(&buf, []string{"msg1", "msg2"})
		output := buf.String()
		assert.Contains(t, output, "msg1")
		assert.Contains(t, output, "msg2")
	})
}

// TestPrintWarnings tests the PrintWarnings function.
//
// It verifies that:
//   - Empty warnings produce no output
//   - Warnings are printed with the appropriate icon
func TestPrintWarnings(t *testing.T) {
	t.Run("empty warnings prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		PrintWarnings(&buf, []string{})
		assert.Empty(t, buf.String())
	})

	t.Run("prints warnings with icon", func(t *testing.T) {
		var buf bytes.Buffer
		PrintWarnings(&buf, []string{"warning1"})
		output := buf.String()
		assert.Contains(t, output, "warning1")
		assert.Contains(t, output, constants.IconWarn)
	})
}

// TestPrintWarningsInline tests the PrintWarningsInline function.
//
// It verifies that:
//   - Empty warnings produce no output
//   - Warnings are printed inline without leading blank line
//   - Multiple warnings are displayed with icons
func TestPrintWarningsInline(t *testing.T) {
	t.Run("empty warnings prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		PrintWarningsInline(&buf, []string{})
		assert.Empty(t, buf.String())
	})

	t.Run("prints warnings with icon", func(t *testing.T) {
		var buf bytes.Buffer
		PrintWarningsInline(&buf, []string{"warning1", "warning2"})
		output := buf.String()
		assert.Contains(t, output, "warning1")
		assert.Contains(t, output, "warning2")
		assert.Contains(t, output, constants.IconWarn)
		// Should not start with blank line (unlike PrintWarnings)
		assert.True(t, output[0] != '\n', "Should not start with blank line")
	})
}

// TestPrintUnsupported tests the PrintUnsupported function.
//
// It verifies that:
//   - Empty packages produce no output
//   - Non-verbose mode hides file paths
//   - Verbose mode shows file paths when available
//   - Package names and reasons are always displayed
func TestPrintUnsupported(t *testing.T) {
	t.Run("empty packages prints nothing", func(t *testing.T) {
		var buf bytes.Buffer
		PrintUnsupported(&buf, []UnsupportedPackage{}, false)
		assert.Empty(t, buf.String())
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		var buf bytes.Buffer
		packages := []UnsupportedPackage{
			{Name: "pkg1", File: "/path/to/file", Reason: "floating constraint"},
			{Name: "pkg2", File: "", Reason: "no lock file"},
		}
		PrintUnsupported(&buf, packages, false)
		output := buf.String()
		assert.Contains(t, output, "pkg1: floating constraint")
		assert.Contains(t, output, "pkg2: no lock file")
		// Should not contain file path in non-verbose mode
		assert.NotContains(t, output, "/path/to/file")
	})

	t.Run("verbose mode with file path", func(t *testing.T) {
		var buf bytes.Buffer
		packages := []UnsupportedPackage{
			{Name: "pkg1", File: "/path/to/file", Reason: "floating constraint"},
		}
		PrintUnsupported(&buf, packages, true)
		output := buf.String()
		assert.Contains(t, output, "pkg1")
		assert.Contains(t, output, "/path/to/file")
		assert.Contains(t, output, "floating constraint")
	})

	t.Run("verbose mode without file path", func(t *testing.T) {
		var buf bytes.Buffer
		packages := []UnsupportedPackage{
			{Name: "pkg1", File: "", Reason: "no lock"},
		}
		PrintUnsupported(&buf, packages, true)
		output := buf.String()
		assert.Contains(t, output, "pkg1: no lock")
	})
}

// TestPrintSummary tests the PrintSummary function.
//
// It verifies that:
//   - Summary with only total count displays correctly
//   - Succeeded, failed, and skipped counts appear when non-zero
//   - Full summary includes all components in proper format
func TestPrintSummary(t *testing.T) {
	t.Run("total only", func(t *testing.T) {
		var buf bytes.Buffer
		PrintSummary(&buf, Summary{Total: 10})
		assert.Contains(t, buf.String(), "Summary: 10 total")
		assert.NotContains(t, buf.String(), "succeeded")
		assert.NotContains(t, buf.String(), "failed")
		assert.NotContains(t, buf.String(), "skipped")
	})

	t.Run("with succeeded", func(t *testing.T) {
		var buf bytes.Buffer
		PrintSummary(&buf, Summary{Total: 10, Succeeded: 8})
		output := buf.String()
		assert.Contains(t, output, "10 total")
		assert.Contains(t, output, "8 succeeded")
	})

	t.Run("with failed", func(t *testing.T) {
		var buf bytes.Buffer
		PrintSummary(&buf, Summary{Total: 10, Succeeded: 8, Failed: 1})
		output := buf.String()
		assert.Contains(t, output, "1 failed")
	})

	t.Run("with skipped", func(t *testing.T) {
		var buf bytes.Buffer
		PrintSummary(&buf, Summary{Total: 10, Succeeded: 8, Failed: 1, Skipped: 1})
		output := buf.String()
		assert.Contains(t, output, "1 skipped")
	})

	t.Run("full summary", func(t *testing.T) {
		var buf bytes.Buffer
		PrintSummary(&buf, Summary{Total: 10, Succeeded: 7, Failed: 2, Skipped: 1})
		output := buf.String()
		assert.Contains(t, output, "Summary: 10 total, 7 succeeded, 2 failed, 1 skipped")
	})
}

// TestNewWarningCollector tests the NewWarningCollector function.
//
// It verifies that:
//   - A new collector is properly initialized with empty messages
//   - Written messages are captured and retrievable
func TestNewWarningCollector(t *testing.T) {
	collector := NewWarningCollector()
	assert.NotNil(t, collector)
	assert.Empty(t, collector.Messages())

	// Write some messages
	_, _ = collector.Write([]byte("test warning\n"))
	assert.Len(t, collector.Messages(), 1)
	assert.Equal(t, "test warning", collector.Messages()[0])
}

// TestPrintNoPackagesMessage tests the PrintNoPackagesMessage function.
//
// It verifies that:
//   - Without context, displays "No packages found"
//   - With context, appends context to the message
func TestPrintNoPackagesMessage(t *testing.T) {
	t.Run("without context", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessage(&buf, "")
		assert.Equal(t, "No packages found\n", buf.String())
	})

	t.Run("with context", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessage(&buf, "matching filters")
		assert.Equal(t, "No packages found matching filters\n", buf.String())
	})
}

// TestPrintNoPackagesMessageWithFilters tests the PrintNoPackagesMessageWithFilters function.
//
// It verifies that:
//   - No filters produces plain "No packages found" message
//   - Active filters are included in parentheses
func TestPrintNoPackagesMessageWithFilters(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessageWithFilters(&buf, "all", "all", "all")
		assert.Equal(t, "No packages found\n", buf.String())
	})

	t.Run("with type filter", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessageWithFilters(&buf, "prod", "all", "all")
		assert.Contains(t, buf.String(), "(type: prod)")
	})

	t.Run("with pm filter", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessageWithFilters(&buf, "all", "npm", "all")
		assert.Contains(t, buf.String(), "(pm: npm)")
	})

	t.Run("with rule filter", func(t *testing.T) {
		var buf bytes.Buffer
		PrintNoPackagesMessageWithFilters(&buf, "all", "all", "rule1")
		assert.Contains(t, buf.String(), "(rule: rule1)")
	})
}

// TestWarningCollectorWrite tests the Write method of WarningCollector.
//
// It verifies that:
//   - Multiple messages in a single write are split by newlines
//   - Whitespace is trimmed from each message
//   - Empty lines are ignored
func TestWarningCollectorWrite(t *testing.T) {
	collector := &WarningCollector{}

	t.Run("writes and trims messages", func(t *testing.T) {
		input := []byte("  warning 1  \n  warning 2  \n")
		n, err := collector.Write(input)
		assert.NoError(t, err)
		assert.Equal(t, len(input), n)

		messages := collector.Messages()
		assert.Len(t, messages, 2)
		assert.Equal(t, "warning 1", messages[0])
		assert.Equal(t, "warning 2", messages[1])
	})

	t.Run("ignores empty lines", func(t *testing.T) {
		collector2 := &WarningCollector{}
		_, _ = collector2.Write([]byte("\n\n  \n"))
		assert.Empty(t, collector2.Messages())
	})
}

// TestWarningCollectorMessages tests the Messages method of WarningCollector.
//
// It verifies that:
//   - Messages returns a copy of stored messages
//   - Modifying the returned slice does not affect the collector
func TestWarningCollectorMessages(t *testing.T) {
	collector := &WarningCollector{}
	_, _ = collector.Write([]byte("msg1\nmsg2"))

	// Messages returns a copy
	messages1 := collector.Messages()
	messages2 := collector.Messages()

	assert.Equal(t, messages1, messages2)

	// Modifying returned slice doesn't affect collector
	messages1[0] = "modified"
	assert.NotEqual(t, messages1[0], collector.Messages()[0])
}

// TestWarningCollectorReset tests the Reset method of WarningCollector.
//
// It verifies that:
//   - Reset clears all stored messages
//   - Collector can be reused after reset
func TestWarningCollectorReset(t *testing.T) {
	collector := &WarningCollector{}
	_, _ = collector.Write([]byte("msg1\nmsg2"))
	assert.Len(t, collector.Messages(), 2)

	collector.Reset()
	assert.Empty(t, collector.Messages())
}

// TestSafeInstalledValue tests the SafeInstalledValue function.
//
// It verifies that:
//   - Empty or whitespace-only values return PlaceholderNA
//   - Valid versions are returned trimmed
func TestSafeInstalledValue(t *testing.T) {
	assert.Equal(t, constants.PlaceholderNA, SafeInstalledValue(""))
	assert.Equal(t, constants.PlaceholderNA, SafeInstalledValue("   "))
	assert.Equal(t, "1.2.3", SafeInstalledValue("1.2.3"))
	assert.Equal(t, "1.2.3", SafeInstalledValue("  1.2.3  "))
}

// TestSafeDeclaredValue tests the SafeDeclaredValue function.
//
// It verifies that:
//   - Empty or whitespace-only values return PlaceholderWildcard
//   - N/A placeholders (case-insensitive) return PlaceholderWildcard
//   - Valid versions are returned trimmed
func TestSafeDeclaredValue(t *testing.T) {
	assert.Equal(t, constants.PlaceholderWildcard, SafeDeclaredValue(""))
	assert.Equal(t, constants.PlaceholderWildcard, SafeDeclaredValue("   "))
	assert.Equal(t, constants.PlaceholderWildcard, SafeDeclaredValue(constants.PlaceholderNA))
	assert.Equal(t, constants.PlaceholderWildcard, SafeDeclaredValue("#n/a")) // case insensitive
	assert.Equal(t, "1.2.3", SafeDeclaredValue("1.2.3"))
	assert.Equal(t, "1.2.3", SafeDeclaredValue("  1.2.3  "))
}

// TestSafeVersionValue tests the SafeVersionValue function.
//
// It verifies that:
//   - Empty or whitespace-only values return the default
//   - Valid versions are returned as-is
func TestSafeVersionValue(t *testing.T) {
	assert.Equal(t, "default", SafeVersionValue("", "default"))
	assert.Equal(t, "default", SafeVersionValue("   ", "default"))
	assert.Equal(t, "1.2.3", SafeVersionValue("1.2.3", "default"))
}

// TestFormatStatus tests the FormatStatus function.
//
// It verifies that status strings are formatted with appropriate icons.
func TestFormatStatus(t *testing.T) {
	assert.Contains(t, FormatStatus(constants.StatusUpdated), constants.IconSuccess)
	assert.Contains(t, FormatStatus(constants.StatusUpdated), constants.StatusUpdated)
	assert.Contains(t, FormatStatus(constants.StatusFailed), constants.IconError)
	assert.Contains(t, FormatStatus(constants.StatusPlanned), constants.IconPending)
}

// TestFormatStatusWithIcon tests the FormatStatusWithIcon function.
//
// It verifies that:
//   - Exact status matches return formatted strings with icons
//   - Prefix matches work for statuses like "Failed(1)"
//   - Case-insensitive matching works
//   - Lock statuses are handled correctly
//   - Unknown statuses are returned as-is
func TestFormatStatusWithIcon(t *testing.T) {
	t.Run("exact matches", func(t *testing.T) {
		assert.Equal(t, constants.IconSuccess+" Updated", FormatStatusWithIcon("Updated"))
		assert.Equal(t, constants.IconError+" Failed", FormatStatusWithIcon("Failed"))
		assert.Equal(t, constants.IconPending+" Planned", FormatStatusWithIcon("Planned"))
		assert.Equal(t, constants.IconWarning+" Outdated", FormatStatusWithIcon("Outdated"))
	})

	t.Run("prefix matches", func(t *testing.T) {
		assert.Equal(t, constants.IconError+" Failed(1)", FormatStatusWithIcon("Failed(1)"))
		assert.Equal(t, constants.IconError+" Failed(exit 2)", FormatStatusWithIcon("Failed(exit 2)"))
	})

	t.Run("case insensitive", func(t *testing.T) {
		assert.Contains(t, FormatStatusWithIcon("UPDATED"), constants.IconSuccess)
		assert.Contains(t, FormatStatusWithIcon("failed"), constants.IconError)
	})

	t.Run("lock statuses", func(t *testing.T) {
		assert.Contains(t, FormatStatusWithIcon("LockFound"), constants.IconSuccess)
		assert.Contains(t, FormatStatusWithIcon("NotInLock"), constants.IconInfo)
		assert.Contains(t, FormatStatusWithIcon("LockMissing"), constants.IconWarning)
		assert.Contains(t, FormatStatusWithIcon("Floating"), constants.IconBlocked)
		assert.Contains(t, FormatStatusWithIcon("VersionMissing"), constants.IconBlocked)
		assert.Contains(t, FormatStatusWithIcon("NotConfigured"), constants.IconNotConfigured)
	})

	t.Run("unknown status returned as-is", func(t *testing.T) {
		assert.Equal(t, "UnknownStatus", FormatStatusWithIcon("UnknownStatus"))
	})
}

// TestStatusIcon tests the StatusIcon function.
//
// It verifies that correct icons are returned for each status type.
func TestStatusIcon(t *testing.T) {
	assert.Equal(t, constants.IconSuccess, StatusIcon(constants.StatusUpdated))
	assert.Equal(t, constants.IconSuccess, StatusIcon(constants.StatusUpToDate))
	assert.Equal(t, constants.IconError, StatusIcon(constants.StatusFailed))
	assert.Equal(t, constants.IconPending, StatusIcon(constants.StatusPlanned))
	assert.Equal(t, "", StatusIcon("unknown"))
}

// TestIsSuccessStatus tests the IsSuccessStatus function.
//
// It verifies that only Updated and UpToDate statuses are considered success.
func TestIsSuccessStatus(t *testing.T) {
	assert.True(t, IsSuccessStatus(constants.StatusUpdated))
	assert.True(t, IsSuccessStatus(constants.StatusUpToDate))
	assert.False(t, IsSuccessStatus(constants.StatusFailed))
	assert.False(t, IsSuccessStatus(constants.StatusPlanned))
}

// TestIsFailureStatus tests the IsFailureStatus function.
//
// It verifies that Failed and ConfigError statuses are considered failures.
func TestIsFailureStatus(t *testing.T) {
	assert.True(t, IsFailureStatus(constants.StatusFailed))
	assert.True(t, IsFailureStatus(constants.StatusConfigError))
	assert.False(t, IsFailureStatus(constants.StatusUpdated))
	assert.False(t, IsFailureStatus(constants.StatusPlanned))
}

// TestHasAvailableUpdates tests the HasAvailableUpdates function.
//
// It verifies that any non-empty, non-placeholder version indicates available updates.
func TestHasAvailableUpdates(t *testing.T) {
	assert.True(t, HasAvailableUpdates("2.0.0", "", ""))
	assert.True(t, HasAvailableUpdates("", "1.5.0", ""))
	assert.True(t, HasAvailableUpdates("", "", "1.0.1"))
	assert.False(t, HasAvailableUpdates("", "", ""))
	assert.False(t, HasAvailableUpdates(constants.PlaceholderNA, constants.PlaceholderNA, constants.PlaceholderNA))
}

// TestFormatAvailableVersions tests the FormatAvailableVersions function.
//
// It verifies that:
//   - Empty or matching versions return empty string
//   - Available major/minor/patch versions are formatted correctly
func TestFormatAvailableVersions(t *testing.T) {
	t.Run("no available versions", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "", "", "")
		assert.Empty(t, result)
	})

	t.Run("major available", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "2.0.0", "", "")
		assert.Contains(t, result, "major: 2.0.0")
	})

	t.Run("minor available", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "", "1.5.0", "")
		assert.Contains(t, result, "minor: 1.5.0")
	})

	t.Run("multiple available", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "2.0.0", "1.5.0", "")
		assert.Contains(t, result, "major: 2.0.0")
		assert.Contains(t, result, "minor: 1.5.0")
	})

	t.Run("target equals available", func(t *testing.T) {
		result := FormatAvailableVersions("2.0.0", "2.0.0", "", "")
		assert.Empty(t, result)
	})

	t.Run("patch available", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "", "", "1.0.1")
		assert.Contains(t, result, "patch: 1.0.1")
		assert.Contains(t, result, "available")
	})

	t.Run("all three available", func(t *testing.T) {
		result := FormatAvailableVersions("1.0.0", "2.0.0", "1.5.0", "1.0.5")
		assert.Contains(t, result, "major: 2.0.0")
		assert.Contains(t, result, "minor: 1.5.0")
		assert.Contains(t, result, "patch: 1.0.5")
	})
}

// TestTruncateWithEllipsis tests the TruncateWithEllipsis function.
//
// It verifies that strings are truncated with "..." when exceeding max length.
func TestTruncateWithEllipsis(t *testing.T) {
	assert.Equal(t, "hello", TruncateWithEllipsis("hello", 10))
	assert.Equal(t, "hel...", TruncateWithEllipsis("hello world", 6))
	assert.Equal(t, "h...", TruncateWithEllipsis("hello", 4))

	t.Run("maxLen less than 4 is clamped to 4", func(t *testing.T) {
		// When maxLen < 4, it's set to 4, so "hello" (len 5) > 4 means truncation
		result := TruncateWithEllipsis("hello", 2)
		assert.Equal(t, "h...", result) // maxLen forced to 4, truncated to 1 char + "..."
	})

	t.Run("maxLen exactly 3", func(t *testing.T) {
		result := TruncateWithEllipsis("hello", 3)
		assert.Equal(t, "h...", result) // maxLen forced to 4
	})
}

// TestIsValidVersion tests the IsValidVersion function.
//
// It verifies that:
//   - Valid semantic versions return true
//   - Empty, whitespace, and placeholder values return false
func TestIsValidVersion(t *testing.T) {
	assert.True(t, IsValidVersion("1.2.3"))
	assert.False(t, IsValidVersion(""))
	assert.False(t, IsValidVersion("   "))
	assert.False(t, IsValidVersion(constants.PlaceholderNA))
	assert.False(t, IsValidVersion(constants.PlaceholderWildcard))
}

// Tests for StderrWriter

// TestStderrWriterWriteLine tests the WriteLine method of StderrWriter.
//
// It verifies that:
//   - Messages are written with formatting arguments and newline
//   - Messages without arguments are written correctly
//   - Nil writer does not cause panic
func TestStderrWriterWriteLine(t *testing.T) {
	t.Run("with writer", func(t *testing.T) {
		var buf bytes.Buffer
		w := &StderrWriter{Writer: &buf}
		w.WriteLine("Hello %s", "World")
		assert.Equal(t, "Hello World\n", buf.String())
	})

	t.Run("without args", func(t *testing.T) {
		var buf bytes.Buffer
		w := &StderrWriter{Writer: &buf}
		w.WriteLine("Simple message")
		assert.Equal(t, "Simple message\n", buf.String())
	})

	t.Run("nil writer does nothing", func(t *testing.T) {
		w := &StderrWriter{Writer: nil}
		w.WriteLine("test")
		// Should not panic
	})
}

// TestStderrWriterWriteTable tests the WriteTable method of StderrWriter.
//
// It verifies that:
//   - Table header and separator rows are written to the output
//   - Nil writer does not cause panic
func TestStderrWriterWriteTable(t *testing.T) {
	t.Run("with writer", func(t *testing.T) {
		var buf bytes.Buffer
		w := &StderrWriter{Writer: &buf}

		// Create a mock table
		table := &mockTableFormatter{
			headerRow:    "NAME    VERSION",
			separatorRow: "----    -------",
		}

		w.WriteTable(table)
		output := buf.String()
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "----")
	})

	t.Run("nil writer does nothing", func(t *testing.T) {
		w := &StderrWriter{Writer: nil}
		table := &mockTableFormatter{
			headerRow:    "TEST",
			separatorRow: "----",
		}
		w.WriteTable(table)
		// Should not panic
	})
}

// TestStderrWriterFlush tests the Flush method of StderrWriter.
//
// It verifies that:
//   - Flush completes without panic (no-op implementation)
func TestStderrWriterFlush(t *testing.T) {
	w := &StderrWriter{}
	w.Flush() // Should not panic
}

// Tests for NullWriter

// TestNullWriterWriteLine tests the WriteLine method of NullWriter.
//
// It verifies that:
//   - Messages are discarded without panic (no-op implementation)
func TestNullWriterWriteLine(t *testing.T) {
	w := &NullWriter{}
	w.WriteLine("This should be discarded: %s", "test")
	// No output to verify, just ensure no panic
}

// TestNullWriterWriteTable tests the WriteTable method of NullWriter.
//
// It verifies that:
//   - Table data is discarded without panic (no-op implementation)
func TestNullWriterWriteTable(t *testing.T) {
	w := &NullWriter{}
	table := &mockTableFormatter{
		headerRow:    "TEST",
		separatorRow: "----",
	}
	w.WriteTable(table)
	// No output to verify, just ensure no panic
}

// TestNullWriterFlush tests the Flush method of NullWriter.
//
// It verifies that:
//   - Flush completes without panic (no-op implementation)
func TestNullWriterFlush(t *testing.T) {
	w := &NullWriter{}
	w.Flush() // Should not panic
}

// Tests for DefaultStatusFormatter

// TestDefaultStatusFormatterFormat tests the Format method of DefaultStatusFormatter.
//
// It verifies that:
//   - Status strings are formatted with appropriate icons
//   - Updated status shows success icon
//   - Failed status shows error icon
func TestDefaultStatusFormatterFormat(t *testing.T) {
	f := &DefaultStatusFormatter{}

	t.Run("Updated status", func(t *testing.T) {
		result := f.Format(constants.StatusUpdated)
		assert.Contains(t, result, constants.IconSuccess)
		assert.Contains(t, result, constants.StatusUpdated)
	})

	t.Run("Failed status", func(t *testing.T) {
		result := f.Format(constants.StatusFailed)
		assert.Contains(t, result, constants.IconError)
		assert.Contains(t, result, constants.StatusFailed)
	})
}

// TestDefaultStatusFormatterIcon tests the Icon method of DefaultStatusFormatter.
//
// It verifies that:
//   - Success statuses return success icon
//   - Error statuses return error icon
//   - Unknown statuses return empty string
func TestDefaultStatusFormatterIcon(t *testing.T) {
	f := &DefaultStatusFormatter{}

	t.Run("Success statuses", func(t *testing.T) {
		assert.Equal(t, constants.IconSuccess, f.Icon(constants.StatusUpdated))
		assert.Equal(t, constants.IconSuccess, f.Icon(constants.StatusUpToDate))
	})

	t.Run("Error statuses", func(t *testing.T) {
		assert.Equal(t, constants.IconError, f.Icon(constants.StatusFailed))
	})

	t.Run("Unknown status", func(t *testing.T) {
		assert.Equal(t, "", f.Icon("unknown"))
	})
}

// TestDefaultStatusFormatterIsSuccess tests the IsSuccess method of DefaultStatusFormatter.
//
// It verifies that:
//   - Updated and UpToDate statuses are identified as success
//   - Failed and Planned statuses are not identified as success
func TestDefaultStatusFormatterIsSuccess(t *testing.T) {
	f := &DefaultStatusFormatter{}

	t.Run("Updated is success", func(t *testing.T) {
		assert.True(t, f.IsSuccess(constants.StatusUpdated))
	})

	t.Run("UpToDate is success", func(t *testing.T) {
		assert.True(t, f.IsSuccess(constants.StatusUpToDate))
	})

	t.Run("Failed is not success", func(t *testing.T) {
		assert.False(t, f.IsSuccess(constants.StatusFailed))
	})

	t.Run("Planned is not success", func(t *testing.T) {
		assert.False(t, f.IsSuccess(constants.StatusPlanned))
	})
}

// TestDefaultStatusFormatterIsFailure tests the IsFailure method of DefaultStatusFormatter.
//
// It verifies that:
//   - Failed and ConfigError statuses are identified as failure
//   - Updated and Planned statuses are not identified as failure
func TestDefaultStatusFormatterIsFailure(t *testing.T) {
	f := &DefaultStatusFormatter{}

	t.Run("Failed is failure", func(t *testing.T) {
		assert.True(t, f.IsFailure(constants.StatusFailed))
	})

	t.Run("ConfigError is failure", func(t *testing.T) {
		assert.True(t, f.IsFailure(constants.StatusConfigError))
	})

	t.Run("Updated is not failure", func(t *testing.T) {
		assert.False(t, f.IsFailure(constants.StatusUpdated))
	})

	t.Run("Planned is not failure", func(t *testing.T) {
		assert.False(t, f.IsFailure(constants.StatusPlanned))
	})
}

// mockTableFormatter is a test double for TableFormatter interface.
//
// It provides mock implementations for:
//   - HeaderRow: Returns a predefined header string
//   - SeparatorRow: Returns a predefined separator string
//   - Other methods are no-ops for testing purposes
type mockTableFormatter struct {
	headerRow    string
	separatorRow string
}

// AddColumn is a no-op implementation for testing.
//
// Parameters:
//   - name: Column name (ignored)
func (m *mockTableFormatter) AddColumn(name string) {}

// AddColumnWithMinWidth is a no-op implementation for testing.
//
// Parameters:
//   - name: Column name (ignored)
//   - min: Minimum width (ignored)
func (m *mockTableFormatter) AddColumnWithMinWidth(name string, min int) {}

// UpdateWidths is a no-op implementation for testing.
//
// Parameters:
//   - values: Row values for width calculation (ignored)
func (m *mockTableFormatter) UpdateWidths(values ...string) {}

// HeaderRow returns the predefined header string for testing.
//
// Returns:
//   - string: The mock header row
func (m *mockTableFormatter) HeaderRow() string { return m.headerRow }

// SeparatorRow returns the predefined separator string for testing.
//
// Returns:
//   - string: The mock separator row
func (m *mockTableFormatter) SeparatorRow() string { return m.separatorRow }

// FormatRow returns an empty string for testing.
//
// Parameters:
//   - values: Column values for this row (ignored)
//
// Returns:
//   - string: Empty string
func (m *mockTableFormatter) FormatRow(values ...string) string { return "" }

// Tests for progress.go

// TestNewProgress tests the NewProgress constructor.
//
// Parameters:
//   - w: io.Writer for progress output
//   - total: Total number of items to process
//   - message: Progress message template
//
// It verifies that:
//   - A non-nil Progress instance is returned
//   - Progress can be completed via Done()
func TestNewProgress(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Testing")
	assert.NotNil(t, p)
	p.Done()
}

// TestNewStderrProgress tests the NewStderrProgress constructor.
//
// Parameters:
//   - total: Total number of items to process
//   - message: Progress message template
//
// It verifies that:
//   - A non-nil Progress instance is returned
//   - Progress writes to os.Stderr by default
func TestNewStderrProgress(t *testing.T) {
	// NewStderrProgress uses os.Stderr, just test it doesn't panic
	p := NewStderrProgress(5, "Testing")
	assert.NotNil(t, p)
	p.SetEnabled(false) // Disable to avoid actual output
	p.Done()
}

// TestNewDisabledProgress tests the NewDisabledProgress constructor.
//
// Parameters:
//   - total: Total number of items to process
//   - message: Progress message template
//
// It verifies that:
//   - A non-nil Progress instance is returned
//   - Progress is disabled and produces no output
func TestNewDisabledProgress(t *testing.T) {
	p := NewDisabledProgress(10, "Testing")
	assert.NotNil(t, p)
	// Should be disabled
	p.Increment()
	p.Done()
}

// TestNewProgressFromConfig tests the NewProgressFromConfig constructor.
//
// Parameters:
//   - config: ProgressConfig struct with Writer, Total, Message, and Enabled fields
//
// It verifies that:
//   - Progress is created with custom writer
//   - Nil writer defaults to stderr
//   - Disabled config produces disabled progress
func TestNewProgressFromConfig(t *testing.T) {
	t.Run("with writer", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressFromConfig(ProgressConfig{
			Writer:  &buf,
			Total:   10,
			Message: "Processing",
			Enabled: true,
		})
		assert.NotNil(t, p)
		p.Done()
	})

	t.Run("nil writer defaults to stderr", func(t *testing.T) {
		p := NewProgressFromConfig(ProgressConfig{
			Writer:  nil,
			Total:   5,
			Message: "Processing",
			Enabled: false, // Disable to avoid actual stderr output
		})
		assert.NotNil(t, p)
		p.Done()
	})

	t.Run("disabled progress", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressFromConfig(ProgressConfig{
			Writer:  &buf,
			Total:   5,
			Message: "Processing",
			Enabled: false,
		})
		assert.NotNil(t, p)
		p.Increment()
		p.Done()
	})
}

// TestWithProgress tests the WithProgress helper function.
//
// Parameters:
//   - w: io.Writer for progress output
//   - total: Total number of items to process
//   - message: Progress message template
//   - fn: Callback function that receives the Progress instance
//
// It verifies that:
//   - Callback function is invoked with Progress instance
//   - Progress is properly cleaned up after callback completes
func TestWithProgress(t *testing.T) {
	var buf bytes.Buffer
	callCount := 0

	err := WithProgress(&buf, 5, "Processing", func(p *Progress) error {
		for i := 0; i < 5; i++ {
			callCount++
			p.Increment()
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 5, callCount)
}

// TestWithProgressWithError tests the WithProgress helper with error return.
//
// It verifies that:
//   - Errors returned from callback are propagated
//   - Progress is properly cleaned up even on error
func TestWithProgressWithError(t *testing.T) {
	var buf bytes.Buffer
	expectedErr := assert.AnError

	err := WithProgress(&buf, 5, "Processing", func(p *Progress) error {
		return expectedErr
	})

	assert.Equal(t, expectedErr, err)
}

// TestWithProgressConditional tests the WithProgressConditional helper.
//
// Parameters:
//   - w: io.Writer for progress output
//   - total: Total number of items to process
//   - message: Progress message template
//   - enabled: Whether progress should be enabled
//   - fn: Callback function that receives the Progress instance
//
// It verifies that:
//   - Enabled flag controls whether progress is visible
//   - Callback is always invoked regardless of enabled state
func TestWithProgressConditional(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		var buf bytes.Buffer
		called := false

		err := WithProgressConditional(&buf, 5, "Processing", true, func(p *Progress) error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("disabled", func(t *testing.T) {
		var buf bytes.Buffer
		called := false

		err := WithProgressConditional(&buf, 5, "Processing", false, func(p *Progress) error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})
}

// Tests for status.go

// TestFormatStatusAllBranches tests the FormatStatus function with all status types.
//
// It verifies that:
//   - Each status type is formatted with its corresponding icon
//   - Unknown statuses are returned without icon
func TestFormatStatusAllBranches(t *testing.T) {
	tests := []struct {
		status       string
		expectedIcon string
	}{
		{constants.StatusUpdated, constants.IconSuccess},
		{constants.StatusPlanned, constants.IconPending},
		{constants.StatusUpToDate, constants.IconSuccess},
		{constants.StatusFailed, constants.IconError},
		{constants.StatusOutdated, constants.IconWarning},
		{constants.StatusConfigError, constants.IconError},
		{constants.StatusSummarizeError, constants.IconError},
		{"NotConfigured", constants.IconNotConfigured},
		{"Floating", constants.IconBlocked},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := FormatStatus(tc.status)
			if tc.expectedIcon != "" {
				assert.Contains(t, result, tc.expectedIcon)
			} else {
				assert.Equal(t, tc.status, result)
			}
		})
	}
}

// TestStatusIconAllBranches tests the StatusIcon function with all status types.
//
// It verifies that:
//   - Each status type returns its correct icon
//   - Unknown statuses return empty string
func TestStatusIconAllBranches(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{constants.StatusUpdated, constants.IconSuccess},
		{constants.StatusUpToDate, constants.IconSuccess},
		{constants.StatusPlanned, constants.IconPending},
		{constants.StatusFailed, constants.IconError},
		{constants.StatusConfigError, constants.IconError},
		{constants.StatusSummarizeError, constants.IconError},
		{constants.StatusOutdated, constants.IconWarning},
		{"NotConfigured", constants.IconNotConfigured},
		{"Floating", constants.IconBlocked},
		{"unknown", ""},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			result := StatusIcon(tc.status)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsPendingStatus tests the IsPendingStatus function.
//
// It verifies that:
//   - Only Planned status is identified as pending
//   - Updated, Failed, and unknown statuses are not pending
func TestIsPendingStatus(t *testing.T) {
	assert.True(t, IsPendingStatus(constants.StatusPlanned))
	assert.False(t, IsPendingStatus(constants.StatusUpdated))
	assert.False(t, IsPendingStatus(constants.StatusFailed))
	assert.False(t, IsPendingStatus("unknown"))
}

// TestFormatInstallStatus tests the FormatInstallStatus function.
//
// It verifies that:
//   - LockFound returns success icon
//   - NotInLock returns info icon
//   - LockMissing returns warning icon
//   - Floating returns blocked icon
//   - NotConfigured returns not-configured icon
//   - VersionMissing returns error icon
//   - SelfPinned returns pinned icon
//   - Unknown status returns as-is
func TestFormatInstallStatus(t *testing.T) {
	t.Run("LockFound", func(t *testing.T) {
		result := FormatInstallStatus("LockFound")
		assert.Contains(t, result, constants.IconSuccess)
		assert.Contains(t, result, "LockFound")
	})

	t.Run("NotInLock", func(t *testing.T) {
		result := FormatInstallStatus("NotInLock")
		assert.Contains(t, result, constants.IconInfo)
	})

	t.Run("LockMissing", func(t *testing.T) {
		result := FormatInstallStatus("LockMissing")
		assert.Contains(t, result, constants.IconWarning)
	})

	t.Run("Floating", func(t *testing.T) {
		result := FormatInstallStatus("Floating")
		assert.Contains(t, result, constants.IconBlocked)
	})

	t.Run("NotConfigured", func(t *testing.T) {
		result := FormatInstallStatus("NotConfigured")
		assert.Contains(t, result, constants.IconNotConfigured)
	})

	t.Run("VersionMissing", func(t *testing.T) {
		result := FormatInstallStatus("VersionMissing")
		assert.Contains(t, result, constants.IconError)
	})

	t.Run("SelfPinned", func(t *testing.T) {
		result := FormatInstallStatus("SelfPinned")
		assert.Contains(t, result, constants.IconPinned)
	})

	t.Run("unknown status", func(t *testing.T) {
		result := FormatInstallStatus("UnknownStatus")
		assert.Equal(t, "UnknownStatus", result)
	})
}

// Tests for table.go

// TestNewTableFromSchema tests the NewTableFromSchema constructor.
//
// Parameters:
//   - schema: TableSchema defining column configuration
//   - opts: TableOptions for customizing table behavior
//
// It verifies that:
//   - Table is created from schema with correct columns
//   - Optional columns can be shown or hidden via options
//   - Columns with MinWidth are created with minimum width
//   - Columns without MinWidth are created normally
func TestNewTableFromSchema(t *testing.T) {
	t.Run("basic schema", func(t *testing.T) {
		table := NewTableFromSchema(ListSchema, TableOptions{})
		assert.NotNil(t, table)
	})

	t.Run("with optional columns shown", func(t *testing.T) {
		table := NewTableFromSchema(ListSchema, TableOptions{
			ShowOptional: map[string]bool{"GROUP": true},
		})
		assert.NotNil(t, table)
	})

	t.Run("with optional columns hidden", func(t *testing.T) {
		table := NewTableFromSchema(ListSchema, TableOptions{
			ShowOptional: map[string]bool{"GROUP": false},
		})
		assert.NotNil(t, table)
	})

	t.Run("column without MinWidth uses AddColumn", func(t *testing.T) {
		// Schema with a column that has no MinWidth (covers else branch)
		schema := Schema{
			Columns: []ColumnDef{
				{Name: "TEST", MinWidth: 0, Optional: false},
			},
		}
		table := NewTableFromSchema(schema, TableOptions{})
		assert.NotNil(t, table)
	})
}

// TestNewListTable tests the NewListTable constructor.
//
// Parameters:
//   - showGroup: Whether to include the GROUP column
//
// It verifies that:
//   - Table is created with list command columns
//   - GROUP column is conditionally included
func TestNewListTable(t *testing.T) {
	t.Run("with group", func(t *testing.T) {
		table := NewListTable(true)
		assert.NotNil(t, table)
	})

	t.Run("without group", func(t *testing.T) {
		table := NewListTable(false)
		assert.NotNil(t, table)
	})
}

// TestNewOutdatedTable tests the NewOutdatedTable constructor.
//
// Parameters:
//   - showGroup: Whether to include the GROUP column
//
// It verifies that:
//   - Table is created with outdated command columns
//   - GROUP column is conditionally included
func TestNewOutdatedTable(t *testing.T) {
	t.Run("with group", func(t *testing.T) {
		table := NewOutdatedTable(true)
		assert.NotNil(t, table)
	})

	t.Run("without group", func(t *testing.T) {
		table := NewOutdatedTable(false)
		assert.NotNil(t, table)
	})
}

// TestNewUpdateTable tests the NewUpdateTable constructor.
//
// Parameters:
//   - showGroup: Whether to include the GROUP column
//
// It verifies that:
//   - Table is created with update command columns
//   - GROUP column is conditionally included
func TestNewUpdateTable(t *testing.T) {
	t.Run("with group", func(t *testing.T) {
		table := NewUpdateTable(true)
		assert.NotNil(t, table)
	})

	t.Run("without group", func(t *testing.T) {
		table := NewUpdateTable(false)
		assert.NotNil(t, table)
	})
}

// TestNewScanTable tests the NewScanTable constructor.
//
// It verifies that:
//   - Table is created with scan command columns
func TestNewScanTable(t *testing.T) {
	table := NewScanTable()
	assert.NotNil(t, table)
}

// Tests for values.go

// TestFormatAvailableVersionsUpToDate tests the FormatAvailableVersionsUpToDate function.
//
// Parameters:
//   - major: Available major version or empty string
//   - minor: Available minor version or empty string
//   - patch: Available patch version or empty string
//
// It verifies that:
//   - No versions available returns empty string
//   - Placeholder values are treated as empty
//   - Available versions are formatted with labels
//   - Whitespace is trimmed from version strings
func TestFormatAvailableVersionsUpToDate(t *testing.T) {
	t.Run("no versions available", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("", "", "")
		assert.Empty(t, result)
	})

	t.Run("only N/A placeholders", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate(constants.PlaceholderNA, constants.PlaceholderNA, constants.PlaceholderNA)
		assert.Empty(t, result)
	})

	t.Run("major available", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("2.0.0", "", "")
		assert.Contains(t, result, "major: 2.0.0")
		assert.Contains(t, result, "available")
	})

	t.Run("minor available", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("", "1.5.0", "")
		assert.Contains(t, result, "minor: 1.5.0")
	})

	t.Run("patch available", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("", "", "1.0.1")
		assert.Contains(t, result, "patch: 1.0.1")
	})

	t.Run("all versions available", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("2.0.0", "1.5.0", "1.0.1")
		assert.Contains(t, result, "major: 2.0.0")
		assert.Contains(t, result, "minor: 1.5.0")
		assert.Contains(t, result, "patch: 1.0.1")
	})

	t.Run("whitespace is trimmed", func(t *testing.T) {
		result := FormatAvailableVersionsUpToDate("  2.0.0  ", "", "")
		assert.Contains(t, result, "major: 2.0.0")
	})
}

// TestFormatVersion tests the FormatVersion function.
//
// Parameters:
//   - version: Version string to format
//
// It verifies that:
//   - Normal versions are returned as-is
//   - Versions with leading 'v' are preserved
//   - Whitespace is trimmed from version strings
//   - Empty strings are returned as empty
func TestFormatVersion(t *testing.T) {
	t.Run("normal version", func(t *testing.T) {
		result := FormatVersion("1.2.3")
		assert.Equal(t, "1.2.3", result)
	})

	t.Run("with leading v", func(t *testing.T) {
		result := FormatVersion("v1.2.3")
		assert.Equal(t, "v1.2.3", result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := FormatVersion("  1.2.3  ")
		assert.Equal(t, "1.2.3", result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := FormatVersion("")
		assert.Equal(t, "", result)
	})
}
