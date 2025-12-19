// Package verbose provides debug logging with documentation references.
package verbose

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Level represents the verbosity level for debug output.
type Level int

const (
	// LevelQuiet suppresses all debug output
	LevelQuiet Level = iota
	// LevelNormal is the default level with no debug output
	LevelNormal
	// LevelVerbose (-v) shows key decisions, summaries, and important actions
	LevelVerbose
	// LevelDebug (-vv) adds shell commands, drift checks, per-package details
	LevelDebug
	// LevelTrace (-vvv) shows full lists, all parsed packages, all tags
	LevelTrace
)

var (
	mu         sync.RWMutex
	enabled    bool
	suppressed bool // Temporarily suppress verbose output (for drift checks, etc.)
	level      Level = LevelVerbose // Default to verbose when enabled
	writer     io.Writer = os.Stderr
)

// Enable turns on verbose logging and allows debug messages to be printed.
//
// It performs the following operations:
//   - Acquires a write lock to ensure thread-safe modification
//   - Sets the enabled flag to true
//   - Releases the write lock
//
// Returns:
//   - None
func Enable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = true
}

// Disable turns off verbose logging and prevents debug messages from being printed.
//
// It performs the following operations:
//   - Acquires a write lock to ensure thread-safe modification
//   - Sets the enabled flag to false
//   - Releases the write lock
//
// Returns:
//   - None
func Disable() {
	mu.Lock()
	defer mu.Unlock()
	enabled = false
}

// Suppress temporarily suppresses verbose output without disabling it.
// Use this for operations that would produce excessive noise (e.g., drift checks).
// Call Unsuppress() when done to restore normal verbose output.
func Suppress() {
	mu.Lock()
	defer mu.Unlock()
	suppressed = true
}

// Unsuppress restores verbose output after a Suppress() call.
func Unsuppress() {
	mu.Lock()
	defer mu.Unlock()
	suppressed = false
}

// SetLevel sets the verbosity level.
// Level 1 = Verbose (-v), Level 2 = Debug (-vv), Level 3 = Trace (-vvv)
func SetLevel(l int) {
	mu.Lock()
	defer mu.Unlock()
	switch {
	case l <= 0:
		level = LevelVerbose
	case l == 1:
		level = LevelVerbose
	case l == 2:
		level = LevelDebug
	default:
		level = LevelTrace
	}
}

// GetLevel returns the current verbosity level.
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return level
}

// AtLevel returns true if the current level is at least the specified level.
func AtLevel(l Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled && !suppressed && level >= l
}

// IsDebug returns true if debug level (-vv) or higher is enabled.
func IsDebug() bool {
	return AtLevel(LevelDebug)
}

// IsTrace returns true if trace level (-vvv) is enabled.
func IsTrace() bool {
	return AtLevel(LevelTrace)
}

// IsSuppressed returns whether verbose output is currently suppressed.
func IsSuppressed() bool {
	mu.RLock()
	defer mu.RUnlock()
	return suppressed
}

// IsEnabled returns whether verbose logging is currently enabled.
//
// It performs the following operations:
//   - Acquires a read lock to ensure thread-safe access
//   - Reads the enabled flag value
//   - Releases the read lock
//
// Returns:
//   - bool: true if verbose logging is enabled, false otherwise
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

// SetWriter sets the output writer for verbose messages.
//
// It performs the following operations:
//   - Acquires a write lock to ensure thread-safe modification
//   - Updates the writer if the provided writer is not nil
//   - Releases the write lock
//
// Parameters:
//   - w: The io.Writer to use for output; if nil, the writer remains unchanged
//
// Returns:
//   - None
func SetWriter(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	if w != nil {
		writer = w
	}
}

// getWriter returns the current writer with proper locking for internal use.
//
// It performs the following operations:
//   - Acquires a read lock to ensure thread-safe access
//   - Reads the writer value
//   - Releases the read lock
//
// Returns:
//   - io.Writer: The currently configured output writer
func getWriter() io.Writer {
	mu.RLock()
	defer mu.RUnlock()
	return writer
}

// isEnabled returns whether verbose is enabled and not suppressed.
//
// It performs the following operations:
//   - Acquires a read lock to ensure thread-safe access
//   - Checks both enabled and suppressed flags
//   - Releases the read lock
//
// Returns:
//   - bool: true if verbose logging is enabled and not suppressed, false otherwise
func isEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled && !suppressed
}

// Printf prints a formatted verbose message if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Formats and prints the message with [DEBUG] prefix to the configured writer
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - format: Printf-style format string
//   - args: Variadic arguments to format into the string
//
// Returns:
//   - None
func Printf(format string, args ...any) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] "+format+"\n", args...)
	}
}

// Info prints an informational verbose message if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the message with [DEBUG] prefix to the configured writer
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - msg: The message string to print
//
// Returns:
//   - None
func Info(msg string) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] %s\n", msg)
	}
}

// Infof prints a formatted informational verbose message if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Formats and prints the message with [DEBUG] prefix to the configured writer
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - format: Printf-style format string
//   - args: Variadic arguments to format into the string
//
// Returns:
//   - None
func Infof(format string, args ...any) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] "+format+"\n", args...)
	}
}

// Debugf prints a formatted debug message if verbose is enabled.
// Use for shell commands, drift checks, per-package details.
func Debugf(format string, args ...any) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] "+format+"\n", args...)
	}
}

// Tracef prints a formatted trace message if verbose is enabled.
// Use for full version lists, all parsed packages, pattern details.
func Tracef(format string, args ...any) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] "+format+"\n", args...)
	}
}

// DocRef represents a documentation reference for a specific topic.
//
// It contains information to help users find relevant documentation
// when troubleshooting issues or configuring the tool.
//
// Fields:
//   - Topic: A human-readable name for the documentation topic
//   - DocPath: The relative path to the documentation file or section
//   - Hint: A brief description of what the documentation covers
type DocRef struct {
	Topic   string
	DocPath string
	Hint    string
}

// Common documentation references.
var docRefs = map[string]DocRef{
	"config": {
		Topic:   "Configuration",
		DocPath: "docs/configuration.md",
		Hint:    "See configuration guide for YAML schema and options",
	},
	"rules": {
		Topic:   "Rule Configuration",
		DocPath: "docs/configuration.md#rules",
		Hint:    "Define custom rules in .goupdate.yml",
	},
	"lock": {
		Topic:   "Lock File Support",
		DocPath: "docs/configuration.md#lock-files",
		Hint:    "Configure lock file parsing for installed version detection",
	},
	"outdated": {
		Topic:   "Outdated Detection",
		DocPath: "docs/configuration.md#outdated",
		Hint:    "Configure version fetching commands",
	},
	"update": {
		Topic:   "Update Configuration",
		DocPath: "docs/configuration.md#update",
		Hint:    "Configure update and lock commands",
	},
	"groups": {
		Topic:   "Package Groups",
		DocPath: "docs/configuration.md#groups",
		Hint:    "Group packages for atomic updates",
	},
	"cli": {
		Topic:   "CLI Reference",
		DocPath: "docs/cli.md",
		Hint:    "See all available commands and flags",
	},
	"architecture": {
		Topic:   "Architecture",
		DocPath: "docs/architecture/",
		Hint:    "Understand internal data flow and design",
	},
}

// WithDocRef prints a verbose message with a documentation reference if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the message with [DEBUG] prefix
//   - If the topic is found in docRefs, appends documentation reference and hint
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - topic: The documentation topic key (e.g., "config", "rules", "lock")
//   - message: The main message to print
//
// Returns:
//   - None
func WithDocRef(topic, message string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	ref, ok := docRefs[strings.ToLower(topic)]
	if ok {
		_, _ = fmt.Fprintf(w, "[DEBUG] %s\n", message)
		_, _ = fmt.Fprintf(w, "        📖 %s: %s\n", ref.Topic, ref.DocPath)
		_, _ = fmt.Fprintf(w, "        💡 %s\n", ref.Hint)
	} else {
		_, _ = fmt.Fprintf(w, "[DEBUG] %s\n", message)
	}
}

// ConfigHelp prints configuration help for a specific rule issue if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the rule name and issue description
//   - Prints the suggested solution
//   - Appends a documentation reference link
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - rule: The name of the configuration rule
//   - issue: A description of the problem or issue
//   - solution: A description of how to solve the issue
//
// Returns:
//   - None
func ConfigHelp(rule, issue, solution string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Rule '%s': %s\n", rule, issue)
	_, _ = fmt.Fprintf(w, "        Solution: %s\n", solution)
	_, _ = fmt.Fprintf(w, "        📖 See: docs/configuration.md#rules\n")
}

// UnsupportedHelp prints help for unsupported package manager features if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints a message indicating the feature is not supported
//   - Provides a YAML configuration example based on the feature type
//   - Appends a documentation reference link
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - rule: The name of the package manager rule
//   - feature: The unsupported feature name (e.g., "lock", "outdated", "update")
//
// Returns:
//   - None
func UnsupportedHelp(rule, feature string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Rule '%s' does not support '%s'\n", rule, feature)
	_, _ = fmt.Fprintf(w, "        To add support, configure in .goupdate.yml:\n")
	_, _ = fmt.Fprintf(w, "        \n")
	_, _ = fmt.Fprintf(w, "        rules:\n")
	_, _ = fmt.Fprintf(w, "          %s:\n", rule)
	switch feature {
	case "lock", "installed":
		_, _ = fmt.Fprintf(w, "            lock_files:\n")
		_, _ = fmt.Fprintf(w, "              - files: [\"your-lock-file.json\"]\n")
		_, _ = fmt.Fprintf(w, "                format: json\n")
		_, _ = fmt.Fprintf(w, "                extraction:\n")
		_, _ = fmt.Fprintf(w, "                  pattern: '\"(?P<n>[^\"]+)\":\\s*\"(?P<version>[^\"]+)\"'\n")
	case "outdated", "versions":
		_, _ = fmt.Fprintf(w, "            outdated:\n")
		_, _ = fmt.Fprintf(w, "              commands: |\n")
		_, _ = fmt.Fprintf(w, "                your-pm show {{package}} versions --json\n")
	case "update":
		_, _ = fmt.Fprintf(w, "            update:\n")
		_, _ = fmt.Fprintf(w, "              commands: |\n")
		_, _ = fmt.Fprintf(w, "                your-pm update {{package}}@{{version}}\n")
		_, _ = fmt.Fprintf(w, "              commands: |\n")
		_, _ = fmt.Fprintf(w, "                your-pm install\n")
	}
	_, _ = fmt.Fprintf(w, "        \n")
	_, _ = fmt.Fprintf(w, "        📖 See: docs/configuration.md#rules\n")
}

// CommandExec logs command execution details if verbose is enabled.
// Only logs working directory if it's not the default (".").
//
// Parameters:
//   - cmd: The command string being executed
//   - workDir: The working directory path for command execution
func CommandExec(cmd, workDir string) {
	if isEnabled() {
		w := getWriter()
		_, _ = fmt.Fprintf(w, "[DEBUG] Executing: %s\n", cmd)
		// Only log working directory if non-default
		if workDir != "" && workDir != "." {
			_, _ = fmt.Fprintf(w, "        Working dir: %s\n", workDir)
		}
	}
}

// CommandResult logs command execution results if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the command status (succeeded or failed) with exit code
//   - Truncates long command strings to 60 characters for readability
//   - If output is provided, prints up to 5 lines with truncation
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - cmd: The command string that was executed
//   - exitCode: The exit code returned by the command (0 for success)
//   - output: The command output (stdout/stderr)
//
// Returns:
//   - None
func CommandResult(cmd string, exitCode int, output string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	if exitCode == 0 {
		_, _ = fmt.Fprintf(w, "[DEBUG] Command succeeded: %s\n", truncate(cmd, 60))
	} else {
		_, _ = fmt.Fprintf(w, "[DEBUG] Command failed (exit %d): %s\n", exitCode, truncate(cmd, 60))
	}
	if output != "" && len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) > 5 {
			for _, line := range lines[:3] {
				_, _ = fmt.Fprintf(w, "        | %s\n", truncate(line, 100))
			}
			_, _ = fmt.Fprintf(w, "        | ... (%d more lines)\n", len(lines)-3)
		} else {
			for _, line := range lines {
				_, _ = fmt.Fprintf(w, "        | %s\n", truncate(line, 100))
			}
		}
	}
}

// ConfigLoaded logs which config file was loaded if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the path to the loaded configuration file
//   - If extended configs exist, prints their paths
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - path: The file path to the main configuration file that was loaded
//   - extended: A slice of paths to configuration files that were extended/inherited
//
// Returns:
//   - None
func ConfigLoaded(path string, extended []string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Config loaded: %s\n", path)
	if len(extended) > 0 {
		_, _ = fmt.Fprintf(w, "        Extends: %v\n", extended)
	}
}

// PackageFiltered logs when a package is filtered out if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the package name and the reason it was filtered
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - name: The name of the package that was filtered
//   - reason: The reason why the package was filtered out
//
// Returns:
//   - None
func PackageFiltered(name, reason string) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] Package '%s' filtered: %s\n", name, reason)
	}
}

// VersionSelected logs version selection details if enabled.
//
// It performs the following operations:
//   - Checks if verbose logging is enabled
//   - Prints the package name, current version, target version, and selection reason
//   - Does nothing if verbose logging is disabled
//
// Parameters:
//   - pkg: The name of the package
//   - current: The current version of the package
//   - target: The target version selected for the package
//   - reason: The reason why this version was selected
//
// Returns:
//   - None
func VersionSelected(pkg, current, target, reason string) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] Version selected for '%s': %s → %s (%s)\n", pkg, current, target, reason)
	}
}

// VersionCheck logs that a package is being checked for updates.
func VersionCheck(pkg, current, constraint string) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] Checking for updates: %s (current: %s, constraint: %q)\n", pkg, current, constraint)
	}
}

// VersionStrategy logs the versioning strategy being used.
func VersionStrategy(format, sort string) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] Versioning strategy: format=%q, sort=%q\n", format, sort)
	}
}

// VersionsRetrieved logs the raw versions retrieved from a registry.
func VersionsRetrieved(pkg string, versions []string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Parsed %d available versions for %s\n", len(versions), pkg)
	if len(versions) > 0 {
		if len(versions) <= 10 {
			_, _ = fmt.Fprintf(w, "[DEBUG] Raw versions for %s: %v\n", pkg, versions)
		} else {
			_, _ = fmt.Fprintf(w, "[DEBUG] Raw versions for %s: %v... (%d more)\n", pkg, versions[:10], len(versions)-10)
		}
	}
}

// VersionsExcluded logs versions that were excluded by patterns.
func VersionsExcluded(pkg string, before, after int, excluded []string) {
	if !isEnabled() || before == after {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Excluded %d versions (before: %d, after: %d)\n", before-after, before, after)
	if len(excluded) > 0 {
		if len(excluded) <= 10 {
			_, _ = fmt.Fprintf(w, "[DEBUG] Excluded versions for %s: %v\n", pkg, excluded)
		} else {
			_, _ = fmt.Fprintf(w, "[DEBUG] Excluded versions for %s: %v... (%d more)\n", pkg, excluded[:10], len(excluded)-10)
		}
	}
}

// VersionsFiltered logs the newer versions found after filtering.
func VersionsFiltered(pkg, current string, filtered []string) {
	if !isEnabled() {
		return
	}
	w := getWriter()
	_, _ = fmt.Fprintf(w, "[DEBUG] Found %d newer versions for %s (current: %s)\n", len(filtered), pkg, current)
	if len(filtered) > 0 {
		if len(filtered) <= 10 {
			_, _ = fmt.Fprintf(w, "[DEBUG] Newer versions for %s: %v\n", pkg, filtered)
		} else {
			_, _ = fmt.Fprintf(w, "[DEBUG] Newer versions for %s: %v... (%d more)\n", pkg, filtered[:10], len(filtered)-10)
		}
	}
}

// ConstraintFilter logs constraint filtering details.
func ConstraintFilter(pkg string, inputCount int, constraint, originalConstraint, reference string, major, minor, patch bool) {
	if isEnabled() {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] FilterVersionsByConstraint for %s: input=%d versions, constraint=%q (original=%q), reference=%s, flags={major=%v, minor=%v, patch=%v}\n",
			pkg, inputCount, constraint, originalConstraint, reference, major, minor, patch)
	}
}

// ConstraintFilterResult logs the result of constraint filtering.
func ConstraintFilterResult(pkg string, inputCount, outputCount int) {
	if !isEnabled() {
		return
	}
	filtered := inputCount - outputCount
	if filtered > 0 {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] FilterVersionsByConstraint for %s: filtered out %d versions, %d remaining\n", pkg, filtered, outputCount)
	} else {
		_, _ = fmt.Fprintf(getWriter(), "[DEBUG] FilterVersionsByConstraint for %s: all %d versions allowed\n", pkg, outputCount)
	}
}

// truncate shortens a string to the specified maximum length.
//
// It performs the following operations:
//   - Returns the original string if it's within the maxLen limit
//   - Truncates the string to maxLen-3 and appends "..." if it exceeds maxLen
//
// Parameters:
//   - s: The string to truncate
//   - maxLen: The maximum length for the returned string (must be at least 3)
//
// Returns:
//   - string: The original or truncated string with "..." suffix if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
