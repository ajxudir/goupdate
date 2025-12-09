package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/user/goupdate/pkg/constants"
)

// PrintWarnings prints warning messages to the writer.
//
// Formats each warning on its own line with a warning icon prefix.
// Does nothing if warnings slice is empty.
// Prints a blank line before the warnings for separation.
//
// Parameters:
//   - w: Writer to output to (typically os.Stderr)
//   - warnings: Slice of warning messages
//
// Example output:
//
//	<blank line>
//	Warning: Package foo has floating constraint
//	Warning: Unable to determine installed version for bar
func PrintWarnings(w io.Writer, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "%s %s\n", constants.IconWarn, warning)
	}
}

// PrintWarningsInline prints warning messages without a leading blank line.
//
// Same as PrintWarnings but without the leading blank line.
//
// Parameters:
//   - w: Writer to output to
//   - warnings: Slice of warning messages
func PrintWarningsInline(w io.Writer, warnings []string) {
	if len(warnings) == 0 {
		return
	}

	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "%s %s\n", constants.IconWarn, warning)
	}
}

// UnsupportedPackage represents a package that cannot be processed.
//
// Fields:
//   - Name: Package name
//   - File: Package file path
//   - Reason: Why the package is unsupported
type UnsupportedPackage struct {
	// Name is the package name.
	Name string

	// File is the path to the package file.
	File string

	// Reason explains why this package is unsupported.
	Reason string
}

// PrintUnsupported prints unsupported packages with reasons.
//
// Formats each unsupported package message on its own line.
// Does nothing if packages slice is empty.
// Prints a blank line before the messages for separation.
//
// Parameters:
//   - w: Writer to output to
//   - packages: Slice of unsupported packages
//   - verbose: If true, includes additional details (file path)
//
// Example output:
//
//	<blank line>
//	foo: floating constraint (cannot determine target version)
//	bar: no lock file configured
func PrintUnsupported(w io.Writer, packages []UnsupportedPackage, verbose bool) {
	if len(packages) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	for _, pkg := range packages {
		if verbose && pkg.File != "" {
			_, _ = fmt.Fprintf(w, "%s (%s): %s\n", pkg.Name, pkg.File, pkg.Reason)
		} else {
			_, _ = fmt.Fprintf(w, "%s: %s\n", pkg.Name, pkg.Reason)
		}
	}
}

// PrintUnsupportedMessages prints unsupported package messages.
//
// Simplified version that takes pre-formatted string messages.
// Does nothing if messages slice is empty.
// Prints a blank line before the messages for separation.
//
// Parameters:
//   - w: Writer to output to
//   - messages: Pre-formatted message strings
func PrintUnsupportedMessages(w io.Writer, messages []string) {
	if len(messages) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w)
	for _, message := range messages {
		_, _ = fmt.Fprintln(w, message)
	}
}

// Summary holds operation summary data.
//
// Fields:
//   - Total: Total items processed
//   - Succeeded: Items that succeeded
//   - Failed: Items that failed
//   - Skipped: Items that were skipped
type Summary struct {
	// Total is the total number of items processed.
	Total int

	// Succeeded is the number of successful operations.
	Succeeded int

	// Failed is the number of failed operations.
	Failed int

	// Skipped is the number of skipped operations.
	Skipped int
}

// PrintSummary prints an operation summary.
//
// Parameters:
//   - w: Writer to output to
//   - summary: Summary data to display
//
// Example output:
//
//	Summary: 10 total, 8 succeeded, 1 failed, 1 skipped
func PrintSummary(w io.Writer, summary Summary) {
	_, _ = fmt.Fprintf(w, "Summary: %d total", summary.Total)
	if summary.Succeeded > 0 {
		_, _ = fmt.Fprintf(w, ", %d succeeded", summary.Succeeded)
	}
	if summary.Failed > 0 {
		_, _ = fmt.Fprintf(w, ", %d failed", summary.Failed)
	}
	if summary.Skipped > 0 {
		_, _ = fmt.Fprintf(w, ", %d skipped", summary.Skipped)
	}
	_, _ = fmt.Fprintln(w)
}

// PrintNoPackagesMessage prints a "no packages found" message.
//
// Parameters:
//   - w: Writer to output to
//   - context: Context string describing what filters were used (optional)
//
// Example output:
//
//	No packages found
//	No packages found matching filters
func PrintNoPackagesMessage(w io.Writer, context string) {
	if context != "" {
		_, _ = fmt.Fprintf(w, "No packages found %s\n", context)
	} else {
		_, _ = fmt.Fprintln(w, "No packages found")
	}
}

// PrintNoPackagesMessageWithFilters prints a message when no packages are found with filter details.
//
// Parameters:
//   - w: Writer to output to
//   - typeFlag: Type filter value
//   - pmFlag: Package manager filter value
//   - ruleFlag: Rule filter value
//
// Example output:
//
//	No packages found (type: prod) (pm: npm) (rule: frontend)
func PrintNoPackagesMessageWithFilters(w io.Writer, typeFlag, pmFlag, ruleFlag string) {
	_, _ = fmt.Fprint(w, "No packages found")
	if typeFlag != constants.FilterAll && typeFlag != "" {
		_, _ = fmt.Fprintf(w, " (type: %s)", typeFlag)
	}
	if pmFlag != constants.FilterAll && pmFlag != "" {
		_, _ = fmt.Fprintf(w, " (pm: %s)", pmFlag)
	}
	if ruleFlag != constants.FilterAll && ruleFlag != "" {
		_, _ = fmt.Fprintf(w, " (rule: %s)", ruleFlag)
	}
	_, _ = fmt.Fprintln(w)
}

// WarningCollector captures warnings for deferred output.
//
// Implements io.Writer so it can be used as a warning sink.
// Warnings are collected and can be printed later using Messages().
//
// Example:
//
//	collector := &WarningCollector{}
//	// ... operations that may produce warnings ...
//	display.PrintWarnings(os.Stderr, collector.Messages())
type WarningCollector struct {
	messages []string
}

// Write implements io.Writer for capturing warning messages.
//
// Splits input on newlines and stores non-empty trimmed lines.
//
// Parameters:
//   - p: Byte slice containing warning message data
//
// Returns:
//   - int: Number of bytes written (always len(p))
//   - error: Always nil, never returns an error
func (c *WarningCollector) Write(p []byte) (int, error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			c.messages = append(c.messages, trimmed)
		}
	}
	return len(p), nil
}

// Messages returns a copy of all collected warning messages.
//
// Creates a defensive copy to prevent external modification of the internal slice.
//
// Returns:
//   - []string: Copy of all collected warning messages
func (c *WarningCollector) Messages() []string {
	copied := make([]string, len(c.messages))
	copy(copied, c.messages)
	return copied
}

// Reset clears all collected messages.
//
// Use this when you want to reuse the same collector for a new batch of warnings.
func (c *WarningCollector) Reset() {
	c.messages = nil
}

// NewWarningCollector creates a new WarningCollector.
//
// Returns:
//   - *WarningCollector: A new empty warning collector ready for use
//
// Example:
//
//	collector := display.NewWarningCollector()
//	warnings.SetOutput(collector)
//	// ... operations that may produce warnings ...
//	display.PrintWarnings(os.Stderr, collector.Messages())
func NewWarningCollector() *WarningCollector {
	return &WarningCollector{}
}
