// Package systemtest provides system test execution for validating application health
// before, during, and after dependency updates.
package systemtest

import (
	"fmt"
	"strings"
	"time"
)

// TestResult represents the result of a single system test execution.
type TestResult struct {
	// Name is the test identifier.
	Name string

	// Passed indicates whether the test passed.
	Passed bool

	// Duration is how long the test took to execute.
	Duration time.Duration

	// Error contains the error message if the test failed.
	Error error

	// Output contains the test command output (stdout/stderr).
	Output string

	// ContinueOnFail indicates if the update process should continue despite failure.
	ContinueOnFail bool
}

// Result represents the aggregate result of running all system tests.
type Result struct {
	// Tests contains results for each individual test.
	Tests []TestResult

	// Phase indicates when the tests were run (preflight, after_each, after_all).
	Phase string

	// TotalDuration is the total time for all tests.
	TotalDuration time.Duration
}

// Passed returns true if all tests passed and returns false if any test failed.
//
// Returns:
//   - bool: true if all tests passed; false if any test failed or no tests were run
func (r *Result) Passed() bool {
	for _, t := range r.Tests {
		if !t.Passed {
			return false
		}
	}
	return true
}

// PassedCount returns the number of tests that passed.
//
// Returns:
//   - int: Count of tests with Passed=true
func (r *Result) PassedCount() int {
	count := 0
	for _, t := range r.Tests {
		if t.Passed {
			count++
		}
	}
	return count
}

// FailedCount returns the number of tests that failed.
//
// Returns:
//   - int: Count of tests with Passed=false
func (r *Result) FailedCount() int {
	count := 0
	for _, t := range r.Tests {
		if !t.Passed {
			count++
		}
	}
	return count
}

// CriticalFailures returns tests that failed and are marked as critical (ContinueOnFail=false).
//
// Critical failures should halt the update process to prevent system instability.
//
// Returns:
//   - []TestResult: Slice of failed tests where ContinueOnFail is false; empty if no critical failures
func (r *Result) CriticalFailures() []TestResult {
	var failures []TestResult
	for _, t := range r.Tests {
		if !t.Passed && !t.ContinueOnFail {
			failures = append(failures, t)
		}
	}
	return failures
}

// HasCriticalFailure returns true if any critical test (ContinueOnFail=false) failed.
//
// Returns:
//   - bool: true if at least one critical test failed; false otherwise
func (r *Result) HasCriticalFailure() bool {
	return len(r.CriticalFailures()) > 0
}

// FailedTests returns all tests that failed regardless of ContinueOnFail setting.
//
// Returns:
//   - []TestResult: Slice of all failed tests; empty if no tests failed
func (r *Result) FailedTests() []TestResult {
	var failures []TestResult
	for _, t := range r.Tests {
		if !t.Passed {
			failures = append(failures, t)
		}
	}
	return failures
}

// Summary returns a brief summary string of the test results.
//
// Returns:
//   - string: One-line summary showing passed/failed counts (e.g., "All 5 system tests passed" or "3/5 system tests passed (2 failed)")
func (r *Result) Summary() string {
	total := len(r.Tests)
	passed := r.PassedCount()
	failed := r.FailedCount()

	if failed == 0 {
		return fmt.Sprintf("All %d system tests passed", total)
	}
	return fmt.Sprintf("%d/%d system tests passed (%d failed)", passed, total, failed)
}

// FormatResults returns a formatted string showing all test results including passing tests.
//
// Use FormatResultsQuiet for minimal output (only shows on failure).
//
// Returns:
//   - string: Multi-line formatted output with test phase, individual test status, and durations
func (r *Result) FormatResults() string {
	return r.formatResults(true)
}

// FormatResultsQuiet returns formatted results only if there are failures.
//
// Returns:
//   - string: Formatted output showing only failed tests; empty string if all tests passed
func (r *Result) FormatResultsQuiet() string {
	if r.Passed() {
		return ""
	}
	return r.formatResults(false)
}

// formatResults is the internal implementation for formatting test results.
//
// It performs the following operations:
//   - Step 1: Build formatted header with test phase
//   - Step 2: Iterate through tests and format each result with icon and duration
//   - Step 3: Show error details for failed tests
//
// Parameters:
//   - showPassing: When true, all tests are shown; when false, only failures are shown
//
// Returns:
//   - string: Formatted multi-line output with test results
func (r *Result) formatResults(showPassing bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("System Tests (%s)\n", r.Phase))
	sb.WriteString(strings.Repeat("─", 60) + "\n")

	for _, t := range r.Tests {
		// In quiet mode, skip passing tests
		if !showPassing && t.Passed {
			continue
		}

		icon := "✓"
		if !t.Passed {
			icon = "✗"
		}
		durationStr := formatDuration(t.Duration)
		sb.WriteString(fmt.Sprintf("  %s %-40s [%s]\n", icon, t.Name, durationStr))

		if !t.Passed && t.Error != nil {
			// Show first line of error
			errLines := strings.Split(t.Error.Error(), "\n")
			if len(errLines) > 0 {
				sb.WriteString(fmt.Sprintf("    └─ %s\n", errLines[0]))
			}
		}
	}

	sb.WriteString(strings.Repeat("─", 60) + "\n")
	return sb.String()
}

// formatDuration formats a duration for display in human-readable format.
//
// Parameters:
//   - d: Duration to format
//
// Returns:
//   - string: Duration formatted as milliseconds (e.g., "500ms") if less than 1 second, otherwise as seconds (e.g., "2.5s")
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// Phase constants for test execution timing.
const (
	PhasePreflight = "Preflight"
	PhaseAfterEach = "After Update"
	PhaseAfterAll  = "Validation"
)
