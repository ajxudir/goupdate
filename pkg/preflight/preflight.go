// Package preflight provides command validation before executing outdated or update operations.
package preflight

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// CommandResolutionHints maps command names to installation instructions.
//
// This map provides helpful hints for resolving missing commands during pre-flight validation.
// It includes common package managers, Unix tools, and specialized utilities like jq and yq.
//
// Keys are command names, values are human-readable installation instructions with URLs.
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

	// Common Unix tools (pre-installed on Linux/macOS)
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

// ValidationError represents a missing command with resolution hints.
//
// This error type is returned when a required command is not found in the system PATH
// or available as a shell alias/function. It provides installation hints to help users
// resolve the missing dependency.
//
// Fields:
//   - Command: The name of the missing command
//   - Hint: Installation instructions or URL for resolving the missing command (empty if no hint available)
type ValidationError struct {
	Command string
	Hint    string
}

// Error returns a formatted error message with resolution instructions.
//
// If a hint is available, it includes the hint in the resolution section.
// Otherwise, it provides generic guidance to check PATH or update the configuration.
//
// Returns:
//   - string: Formatted error message including command name and resolution instructions
func (e *ValidationError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("command not found: %s\n  Resolution: %s", e.Command, e.Hint)
	}
	// Clear default message for custom/unknown commands
	return fmt.Sprintf("command not found: %s\n  Resolution: Ensure '%s' is installed and available in your PATH.\n             If using a custom tool, install it or update your config to use an available alternative.", e.Command, e.Command)
}

// ValidateResult holds the result of pre-flight validation.
//
// This structure aggregates all validation errors and warnings discovered during
// pre-flight checks. It provides methods to check for errors and format error messages.
//
// Fields:
//   - Errors: List of validation errors for missing or unavailable commands
//   - Warnings: List of warning messages (currently unused, reserved for future use)
type ValidateResult struct {
	Errors   []ValidationError
	Warnings []string
}

// HasErrors returns true if there are validation errors.
//
// This method provides a convenient way to check if validation failed without
// inspecting the Errors slice directly.
//
// Returns:
//   - bool: true if there are one or more validation errors, false otherwise
func (r *ValidateResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ErrorMessage returns a formatted error message for all validation errors.
//
// This method consolidates all validation errors into a single, user-friendly
// message suitable for display. Each error is formatted with resolution hints.
//
// Returns:
//   - string: Formatted multi-line error message with header and list of errors; empty string if no errors
func (r *ValidateResult) ErrorMessage() string {
	if len(r.Errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Pre-flight validation failed:\n")
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// ValidatePackages checks that all required commands for the given packages are available.
//
// It performs the following operations:
//   - Extracts all commands from outdated and update configurations for each package's rule
//   - Validates that each unique command exists in the system PATH or as a shell alias
//   - Collects validation errors with resolution hints for missing commands
//
// Parameters:
//   - packages: List of packages to validate, each containing a rule name
//   - cfg: Configuration containing rule definitions with outdated and update commands
//
// Returns:
//   - *ValidateResult: Result containing any validation errors; never nil
func ValidatePackages(packages []formats.Package, cfg *config.Config) *ValidateResult {
	verbose.Debugf("Preflight: validating commands for %d packages", len(packages))
	result := &ValidateResult{}
	checkedCommands := make(map[string]bool)

	for _, p := range packages {
		ruleCfg, ok := cfg.Rules[p.Rule]
		if !ok {
			verbose.Tracef("Preflight: skipping package %q - rule %q not found in config", p.Name, p.Rule)
			continue
		}
		verbose.Tracef("Preflight: checking commands for package %q (rule: %s)", p.Name, p.Rule)

		// Check outdated commands
		if ruleCfg.Outdated != nil {
			commands := extractCommands(ruleCfg.Outdated.Commands)
			for _, cmd := range commands {
				if !checkedCommands[cmd] {
					checkedCommands[cmd] = true
					if err := validateCommand(cmd); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}

		// Check update commands
		if ruleCfg.Update != nil {
			commands := extractCommands(ruleCfg.Update.Commands)
			for _, cmd := range commands {
				if !checkedCommands[cmd] {
					checkedCommands[cmd] = true
					if err := validateCommand(cmd); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}
	}

	verbose.Debugf("Preflight: package validation complete - %d unique commands checked, %d errors", len(checkedCommands), len(result.Errors))
	return result
}

// ValidateRules checks that all required commands for the given rules are available.
//
// It performs the following operations:
//   - Extracts all commands from outdated and update configurations for each rule
//   - Validates that each unique command exists in the system PATH or as a shell alias
//   - Collects validation errors with resolution hints for missing commands
//
// Parameters:
//   - rules: List of rule names to validate
//   - cfg: Configuration containing rule definitions with outdated and update commands
//
// Returns:
//   - *ValidateResult: Result containing any validation errors; never nil
func ValidateRules(rules []string, cfg *config.Config) *ValidateResult {
	verbose.Debugf("Preflight: validating commands for %d rules", len(rules))
	result := &ValidateResult{}
	checkedCommands := make(map[string]bool)

	for _, ruleName := range rules {
		ruleCfg, ok := cfg.Rules[ruleName]
		if !ok {
			verbose.Tracef("Preflight: skipping rule %q - not found in config", ruleName)
			continue
		}
		verbose.Tracef("Preflight: checking commands for rule %q", ruleName)

		// Check outdated commands
		if ruleCfg.Outdated != nil {
			commands := extractCommands(ruleCfg.Outdated.Commands)
			for _, cmd := range commands {
				if !checkedCommands[cmd] {
					checkedCommands[cmd] = true
					if err := validateCommand(cmd); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}

		// Check update commands
		if ruleCfg.Update != nil {
			commands := extractCommands(ruleCfg.Update.Commands)
			for _, cmd := range commands {
				if !checkedCommands[cmd] {
					checkedCommands[cmd] = true
					if err := validateCommand(cmd); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}
	}

	verbose.Debugf("Preflight: rule validation complete - %d unique commands checked, %d errors", len(checkedCommands), len(result.Errors))
	return result
}

// extractCommands extracts all command names from a multiline commands string.
//
// It performs the following operations:
//   - Normalizes line endings for cross-platform compatibility (CRLF to LF)
//   - Skips empty lines and comment lines (starting with #)
//   - Handles line continuation backslashes
//   - Parses piped commands (separated by |)
//   - Extracts the first word from each command segment as the command name
//   - Deduplicates command names
//
// Parameters:
//   - commands: Multi-line string containing shell commands, possibly with pipes and line continuations
//
// Returns:
//   - []string: Unique list of command names in order of first appearance; empty slice if no commands found
func extractCommands(commands string) []string {
	var result []string
	seen := make(map[string]bool)

	trimmed := strings.TrimSpace(commands)
	if trimmed == "" {
		return result
	}

	// Normalize line endings for cross-platform compatibility (CRLF -> LF)
	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove line continuation backslash
		line = strings.TrimSuffix(line, "\\")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle piped commands on single line
		pipeParts := strings.Split(line, "|")
		for _, part := range pipeParts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Extract first word as command
			fields := strings.Fields(part)
			if len(fields) > 0 {
				cmd := fields[0]
				if !seen[cmd] {
					seen[cmd] = true
					result = append(result, cmd)
				}
			}
		}
	}

	return result
}

// validateCommand checks if a command exists in PATH or as a shell alias.
//
// It performs the following operations:
//   - Returns nil for empty command names
//   - First attempts exec.LookPath for fast binary lookup in PATH
//   - Falls back to shell-based check to detect aliases and shell functions
//   - Returns ValidationError with resolution hint if command is not found
//
// Parameters:
//   - cmd: The command name to validate (e.g., "npm", "jq", "grep")
//
// Returns:
//   - *ValidationError: Error with resolution hint if command not found; nil if command exists or cmd is empty
func validateCommand(cmd string) *ValidationError {
	if cmd == "" {
		return nil
	}

	verbose.Tracef("Preflight: checking command %q", cmd)

	// First try exec.LookPath (faster, finds binaries)
	if _, err := exec.LookPath(cmd); err == nil {
		verbose.Tracef("Preflight: command %q found in PATH", cmd)
		return nil
	}

	// Fall back to shell-based check to support aliases
	verbose.Tracef("Preflight: command %q not in PATH, checking shell aliases", cmd)
	if commandExistsInShell(cmd) {
		verbose.Tracef("Preflight: command %q found as shell alias/function", cmd)
		return nil
	}

	hint := CommandResolutionHints[cmd]
	if hint != "" {
		verbose.Printf("Preflight ERROR: command %q not found - hint: %s\n", cmd, hint)
	} else {
		verbose.Printf("Preflight ERROR: command %q not found (no resolution hint available)\n", cmd)
	}
	return &ValidationError{
		Command: cmd,
		Hint:    hint,
	}
}

// commandExistsInShell checks if a command exists through the user's shell.
//
// This function uses the shell's 'command -v' built-in to detect commands, aliases,
// and shell functions that exec.LookPath cannot find. It runs the check in a login
// shell to ensure proper initialization of aliases and functions.
//
// Parameters:
//   - cmd: The command name to check
//
// Returns:
//   - bool: true if the command exists in the shell environment, false otherwise
func commandExistsInShell(cmd string) bool {
	shell, args := getShellCommandCheck(cmd)
	checkCmd := exec.Command(shell, args...)
	return checkCmd.Run() == nil
}

// GetResolutionHint returns the installation hint for a command, if available.
//
// This function looks up installation instructions for common commands in the
// CommandResolutionHints map. It is useful for providing user-friendly guidance
// when a required command is missing.
//
// Parameters:
//   - cmd: The command name to look up (e.g., "npm", "jq", "python")
//
// Returns:
//   - string: Installation instructions with URL if available; empty string if no hint exists
func GetResolutionHint(cmd string) string {
	return CommandResolutionHints[cmd]
}
