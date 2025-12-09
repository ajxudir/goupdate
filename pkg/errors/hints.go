package errors

import (
	"strings"
)

// ErrorHint provides actionable resolution hints for common errors.
//
// Fields:
//   - Pattern: Substring to match in error message (case-insensitive)
//   - Hint: Brief description of the issue
//   - Resolution: Command or action to resolve the issue
type ErrorHint struct {
	// Pattern is a substring to match in error messages (case-insensitive).
	Pattern string

	// Hint is a brief description of the problem.
	Hint string

	// Resolution is a command or action to fix the problem.
	Resolution string
}

// CommandResolutionHints maps command names to installation instructions.
// Used for preflight validation errors when a required command is not found.
var CommandResolutionHints = map[string]string{
	// Package managers
	"npm":      "Install Node.js: https://nodejs.org/",
	"npx":      "Install Node.js: https://nodejs.org/",
	"node":     "Install Node.js: https://nodejs.org/",
	"yarn":     "Install Yarn: https://yarnpkg.com/getting-started/install",
	"pnpm":     "Install pnpm: https://pnpm.io/installation",
	"pip":      "Install Python: https://python.org/downloads/",
	"pip3":     "Install Python: https://python.org/downloads/",
	"python":   "Install Python: https://python.org/downloads/",
	"python3":  "Install Python: https://python.org/downloads/",
	"pipenv":   "Install pipenv: brew install pipenv (macOS), pipx install pipenv, or pip install --user pipenv",
	"go":       "Install Go: https://go.dev/dl/",
	"composer": "Install Composer: https://getcomposer.org/download/",
	"dotnet":   "Install .NET SDK: https://dotnet.microsoft.com/download",
	"nuget":    "Install NuGet CLI or .NET SDK: https://docs.microsoft.com/nuget/install-nuget-client-tools",
	"gem":      "Install Ruby: https://ruby-lang.org/en/downloads/",
	"bundle":   "Install Bundler: gem install bundler",
	"bundler":  "Install Bundler: gem install bundler",
	"cargo":    "Install Rust: https://rustup.rs/",
	"mvn":      "Install Maven: https://maven.apache.org/install.html",
	"gradle":   "Install Gradle: https://gradle.org/install/",

	// Common Unix tools
	"grep": "Unix tool - typically pre-installed on Linux/macOS",
	"awk":  "Unix tool - typically pre-installed on Linux/macOS",
	"sed":  "Unix tool - typically pre-installed on Linux/macOS",
	"sort": "Unix tool - typically pre-installed on Linux/macOS",
	"curl": "Install curl: https://curl.se/download.html (often pre-installed)",
	"wget": "Install wget: https://www.gnu.org/software/wget/ or use curl instead",

	// JSON/YAML processing tools
	"jq": "Install jq: https://jqlang.github.io/jq/download/ (JSON processor)",
	"yq": "Install yq: https://github.com/mikefarah/yq (YAML processor)",
}

// CommonErrorHints maps error patterns to actionable hints.
// These are used by EnhanceErrorWithHint to add context to errors.
var CommonErrorHints = []ErrorHint{
	{
		Pattern:    "failed to parse",
		Hint:       "Check file syntax",
		Resolution: "Validate JSON/YAML syntax using a linter or online validator",
	},
	{
		Pattern:    "lock install drifted",
		Hint:       "Lock file out of sync with manifest",
		Resolution: "Run the package manager's install/update command manually (e.g., npm install, go mod tidy)",
	},
	{
		Pattern:    "version mismatch after update",
		Hint:       "Manifest was not updated correctly",
		Resolution: "Check file permissions and verify the package exists in the manifest",
	},
	{
		Pattern:    "package .* missing after update",
		Hint:       "Package was removed during update",
		Resolution: "Verify the package still exists in the manifest file",
	},
	{
		Pattern:    "command timed out",
		Hint:       "Package manager command took too long",
		Resolution: "Use --no-timeout flag or increase timeout in config (timeout_seconds)",
	},
	{
		Pattern:    "group lock failed",
		Hint:       "Lock command failed for grouped packages",
		Resolution: "Check compatibility between grouped packages or update them individually",
	},
	{
		Pattern:    "failed to load config",
		Hint:       "Configuration file is invalid or not found",
		Resolution: "Run 'goupdate config --show-effective' to validate config, or 'goupdate config --init' to create one",
	},
	{
		Pattern:    "no such file or directory",
		Hint:       "File or directory not found",
		Resolution: "Verify the path exists and you have read permissions",
	},
	{
		Pattern:    "permission denied",
		Hint:       "Insufficient permissions",
		Resolution: "Check file permissions or run with appropriate privileges",
	},
	{
		Pattern:    "network",
		Hint:       "Network connectivity issue",
		Resolution: "Check internet connection and proxy settings",
	},
	{
		Pattern:    "ENOTFOUND",
		Hint:       "DNS resolution failed",
		Resolution: "Check network connectivity and DNS configuration",
	},
	{
		Pattern:    "ECONNREFUSED",
		Hint:       "Connection refused by server",
		Resolution: "Check if the registry/server is accessible and not blocked",
	},
	{
		Pattern:    "401",
		Hint:       "Authentication required",
		Resolution: "Configure authentication for the package registry",
	},
	{
		Pattern:    "403",
		Hint:       "Access forbidden",
		Resolution: "Check permissions and authentication credentials for the registry",
	},
	{
		Pattern:    "404",
		Hint:       "Package or version not found",
		Resolution: "Verify the package name and version exist in the registry",
	},
}

// GetHint returns an actionable hint for the given error.
//
// It searches the error message for known patterns in CommonErrorHints
// and returns a formatted hint if one matches.
//
// Parameters:
//   - err: The error to get a hint for
//
// Returns:
//   - string: The hint with resolution, or empty string if no hint found
//
// Example:
//
//	hint := errors.GetHint(err)
//	if hint != "" {
//	    fmt.Fprintf(os.Stderr, "Hint: %s\n", hint)
//	}
func GetHint(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())
	for _, hint := range CommonErrorHints {
		if strings.Contains(errStr, strings.ToLower(hint.Pattern)) {
			return hint.Hint + ": " + hint.Resolution
		}
	}

	return ""
}

// GetHintForCommand returns the installation hint for a command.
//
// Parameters:
//   - cmd: The command name (e.g., "go", "npm", "pip")
//
// Returns:
//   - string: Installation hint, or empty string if unknown command
func GetHintForCommand(cmd string) string {
	return CommandResolutionHints[cmd]
}

// RegisterHint adds a custom hint to the registry.
//
// This allows extending the hint system with project-specific patterns.
//
// Parameters:
//   - pattern: Lowercase substring to match in error messages
//   - hint: Brief description of the issue
//   - resolution: Actionable suggestion for fixing the error
func RegisterHint(pattern, hint, resolution string) {
	CommonErrorHints = append(CommonErrorHints, ErrorHint{
		Pattern:    pattern,
		Hint:       hint,
		Resolution: resolution,
	})
}

// RegisterCommandHint adds a command installation hint.
//
// Parameters:
//   - command: Command name (e.g., "mycommand")
//   - hint: Installation instructions
func RegisterCommandHint(command, hint string) {
	CommandResolutionHints[command] = hint
}

// EnhanceErrorWithHint adds actionable hints to an error message if a matching pattern is found.
//
// Parameters:
//   - err: The error to enhance
//
// Returns:
//   - string: Error message with hint appended if found, otherwise just the error message
//
// Example:
//
//	enhanced := errors.EnhanceErrorWithHint(err)
//	fmt.Fprintf(os.Stderr, "Error: %s\n", enhanced)
func EnhanceErrorWithHint(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()
	for _, hint := range CommonErrorHints {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(hint.Pattern)) {
			return errStr + "\n  \U0001F4A1 " + hint.Hint + ": " + hint.Resolution
		}
	}

	return errStr
}
