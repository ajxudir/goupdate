// Package cmdexec provides command execution utilities for goupdate.
// It supports multiline commands with piped (|) and sequential (newline) execution,
// environment variable configuration, and templated arguments.
package cmdexec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/user/goupdate/pkg/warnings"
)

// getShell returns the user's shell and args to run a command.
//
// This function checks the SHELL environment variable first (Unix systems),
// and falls back to platform-specific defaults if not set. Using the user's
// shell ensures that aliases and shell configurations are available during
// command execution.
//
// Returns:
//   - shell: The path to the shell executable
//   - args: The shell arguments needed to execute a command string
func getShell() (shell string, args []string) {
	// Check SHELL environment variable first (Unix)
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-l", "-c"}
	}

	// Platform-specific fallback
	return getDefaultShell()
}

// ExecuteFunc is the function signature for command execution.
//
// This type defines the signature for functions that execute commands
// with environment configuration, working directory, timeout, and template
// replacements.
//
// Parameters:
//   - commands: Multiline command string to execute
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//   - replacements: Template variable replacements (e.g., {{package}} -> actual package name)
//
// Returns:
//   - []byte: Combined stdout output from the commands
//   - error: Any error that occurred during execution
type ExecuteFunc func(commands string, env map[string]string, dir string, timeoutSeconds int, replacements map[string]string) ([]byte, error)

// ExecuteWithContextFunc is the function signature for context-aware command execution.
//
// This type defines the signature for functions that execute commands with
// context support, allowing cancellation of long-running operations.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - commands: Multiline command string to execute
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//   - replacements: Template variable replacements (e.g., {{package}} -> actual package name)
//
// Returns:
//   - []byte: Combined stdout output from the commands
//   - error: Any error that occurred during execution, including context cancellation
type ExecuteWithContextFunc func(ctx context.Context, commands string, env map[string]string, dir string, timeoutSeconds int, replacements map[string]string) ([]byte, error)

// Execute is the default command execution function.
//
// This variable holds the implementation used for command execution throughout
// the application. It can be replaced with a mock implementation for testing.
var Execute ExecuteFunc = executeCommands

// ExecuteWithContext is the context-aware command execution function.
//
// This variable holds the context-aware implementation used for command execution.
// It allows callers to cancel long-running operations and can be replaced with
// a mock implementation for testing.
var ExecuteWithContext ExecuteWithContextFunc = executeCommandsWithContext

// executeCommands executes a multiline command string.
//
// This function supports multiple command execution features:
// - Piped commands: lines ending with | are joined and piped together
// - Sequential commands: separate lines run sequentially
// - Line continuation: lines ending with \ are joined with the next line
// - Environment variables from env map
// - Template replacements (e.g., {{package}}, {{version}})
//
// Parameters:
//   - commands: Multiline command string to execute
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//   - replacements: Template variable replacements applied to the command string
//
// Returns:
//   - []byte: Output from the last executed command group
//   - error: Error from the first failed command, or nil if all succeeded
func executeCommands(commands string, env map[string]string, dir string, timeoutSeconds int, replacements map[string]string) ([]byte, error) {
	if strings.TrimSpace(commands) == "" {
		return nil, fmt.Errorf("no commands provided")
	}

	// Apply template replacements
	cmd := applyReplacements(commands, replacements)

	// Parse into command groups (piped commands are one group, sequential are separate)
	groups := parseCommandGroups(cmd)

	var lastOutput []byte
	for _, group := range groups {
		output, err := executePipedCommands(group, env, dir, timeoutSeconds)
		if err != nil {
			return output, err
		}
		lastOutput = output
	}

	return lastOutput, nil
}

// executeCommandsWithContext executes a multiline command string with context support.
//
// The context allows callers to cancel long-running operations. This function supports:
// - Piped commands: lines ending with | are joined and piped together
// - Sequential commands: separate lines run sequentially
// - Line continuation: lines ending with \ are joined with the next line
// - Environment variables from env map
// - Template replacements (e.g., {{package}}, {{version}})
// - Context cancellation checks before each command group
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - commands: Multiline command string to execute
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//   - replacements: Template variable replacements applied to the command string
//
// Returns:
//   - []byte: Output from the last executed command group
//   - error: Error from the first failed command or context cancellation, nil if all succeeded
func executeCommandsWithContext(ctx context.Context, commands string, env map[string]string, dir string, timeoutSeconds int, replacements map[string]string) ([]byte, error) {
	if strings.TrimSpace(commands) == "" {
		return nil, fmt.Errorf("no commands provided")
	}

	// Check if context is already cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Apply template replacements
	cmd := applyReplacements(commands, replacements)

	// Parse into command groups (piped commands are one group, sequential are separate)
	groups := parseCommandGroups(cmd)

	var lastOutput []byte
	for _, group := range groups {
		// Check context before each command group
		if ctx.Err() != nil {
			return lastOutput, ctx.Err()
		}
		output, err := executePipedCommandsWithContext(ctx, group, env, dir, timeoutSeconds)
		if err != nil {
			return output, err
		}
		lastOutput = output
	}

	return lastOutput, nil
}

// applyReplacements applies template replacements to the command string.
//
// Template placeholders in the format {{key}} are replaced with their corresponding
// values from the replacements map. All values are shell-escaped to prevent command
// injection vulnerabilities.
//
// Parameters:
//   - commands: Command string containing template placeholders
//   - replacements: Map of template keys to replacement values
//
// Returns:
//   - string: Command string with all placeholders replaced and values shell-escaped
func applyReplacements(commands string, replacements map[string]string) string {
	result := commands
	for key, value := range replacements {
		placeholder := "{{" + key + "}}"
		// Shell escape the value to prevent injection
		escapedValue := shellEscape(value)
		result = strings.ReplaceAll(result, placeholder, escapedValue)
	}
	return result
}

// shellEscape escapes a string for safe use in shell commands.
//
// This function wraps values in single quotes and properly escapes any single quotes
// within the value. It handles special characters that could cause shell injection or
// parsing issues. Safe characters (alphanumeric, dash, underscore, etc.) are returned
// unquoted for readability.
//
// Parameters:
//   - s: String to escape for shell usage
//
// Returns:
//   - string: Shell-safe escaped string, either quoted or unquoted if safe
func shellEscape(s string) string {
	// If the string is empty, return empty quotes
	if s == "" {
		return "''"
	}

	// Check if the string needs escaping
	// Safe characters: alphanumeric, dash, underscore, dot, slash, at, colon, plus
	needsEscape := false
	for _, r := range s {
		if !isShellSafe(r) {
			needsEscape = true
			break
		}
	}

	if !needsEscape {
		return s
	}

	// Use single quotes for escaping (simplest and safest)
	// Single quotes preserve everything literally except single quotes themselves
	// For single quotes in the string, we close the quote, add escaped single quote, reopen
	var escaped strings.Builder
	escaped.WriteRune('\'')
	for _, r := range s {
		if r == '\'' {
			// End current quote, add escaped quote, start new quote
			escaped.WriteString("'\\''")
		} else {
			escaped.WriteRune(r)
		}
	}
	escaped.WriteRune('\'')
	return escaped.String()
}

// isShellSafe returns true if the character is safe to use unquoted in shell.
//
// Safe characters include alphanumerics and a limited set of special characters
// (dash, underscore, dot, slash, at, colon, plus, equal) that don't require quoting.
//
// Parameters:
//   - r: Rune (character) to check
//
// Returns:
//   - bool: true if the character is safe to use unquoted, false otherwise
func isShellSafe(r rune) bool {
	// Safe: alphanumeric, dash, underscore, dot, slash, at, colon, plus, equal
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.' ||
		r == '/' || r == '@' || r == ':' ||
		r == '+' || r == '='
}

// parseCommandGroups parses a multiline command string into groups.
//
// This function splits a command string into groups for execution:
// - Lines ending with \ (backslash) are joined with the next line
// - Lines ending with | are part of a pipe chain (one group)
// - Lines with inline pipes (e.g., "cmd1 | cmd2") are split and grouped
// - Other newlines separate sequential command groups
//
// Parameters:
//   - commands: Multiline command string to parse
//
// Returns:
//   - [][]string: Array of command groups, where each group is an array of piped commands
func parseCommandGroups(commands string) [][]string {
	// Normalize line endings for cross-platform compatibility (CRLF -> LF)
	normalized := strings.ReplaceAll(commands, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	var groups [][]string
	var currentGroup []string
	var currentCmd strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Handle line continuation with backslash
		if strings.HasSuffix(trimmed, "\\") {
			currentCmd.WriteString(strings.TrimSuffix(trimmed, "\\"))
			currentCmd.WriteString(" ")
			continue
		}

		// Add any accumulated continuation
		currentCmd.WriteString(trimmed)
		fullLine := strings.TrimSpace(currentCmd.String())
		currentCmd.Reset()

		// Check if this line ends with a pipe (continuation to next line)
		if strings.HasSuffix(fullLine, "|") {
			// This is part of a pipe chain
			// Remove trailing pipe and add to current group
			pipeCmd := strings.TrimSuffix(fullLine, "|")
			pipeCmd = strings.TrimSpace(pipeCmd)
			if pipeCmd != "" {
				currentGroup = append(currentGroup, pipeCmd)
			}
			continue
		}

		// Check if line contains inline pipes
		if strings.Contains(fullLine, " | ") || strings.Contains(fullLine, "\t|\t") {
			// Split by pipe
			parts := splitByPipe(fullLine)
			if len(parts) > 1 {
				currentGroup = append(currentGroup, parts...)
				groups = append(groups, currentGroup)
				currentGroup = nil
				continue
			}
		}

		// This is a standalone command or end of pipe chain
		currentGroup = append(currentGroup, fullLine)

		// Always flush the current group after a non-pipe command
		if len(currentGroup) > 0 {
			groups = append(groups, currentGroup)
			currentGroup = nil
		}
	}

	// Handle any remaining commands
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// splitByPipe splits a command line by pipe operators, respecting quotes.
//
// This function intelligently splits commands by pipe (|) characters while
// preserving pipes that appear inside quoted strings. It handles both single
// and double quotes.
//
// Parameters:
//   - line: Command line string to split
//
// Returns:
//   - []string: Array of command parts separated by pipes
func splitByPipe(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Handle quotes
		if (r == '"' || r == '\'') && (i == 0 || runes[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			}
			current.WriteRune(r)
			continue
		}

		// Check for pipe outside quotes
		if !inQuote && r == '|' {
			part := strings.TrimSpace(current.String())
			if part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	// Add final part
	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}

// executePipedCommands executes a group of piped commands.
//
// Commands are run through the user's shell to support aliases and shell configurations.
// For multiple commands, they are joined with pipes and executed as a single shell command.
// This is a convenience wrapper around executePipedCommandsWithContext using background context.
//
// Parameters:
//   - commands: Array of command strings to pipe together
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//
// Returns:
//   - []byte: Combined stdout output from the piped commands
//   - error: Any error that occurred during execution
func executePipedCommands(commands []string, env map[string]string, dir string, timeoutSeconds int) ([]byte, error) {
	return executePipedCommandsWithContext(context.Background(), commands, env, dir, timeoutSeconds)
}

// executePipedCommandsWithContext executes a group of piped commands with context support.
//
// The provided context is used as the parent for any timeout context, allowing
// callers to cancel operations externally. For multiple commands, they are joined
// with pipes and executed as a single shell command for efficiency and proper
// shell feature handling.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - commands: Array of command strings to pipe together
//   - env: Environment variables to set for the commands
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (0 for no timeout)
//
// Returns:
//   - []byte: Combined stdout output from the piped commands
//   - error: Any error that occurred during execution or context cancellation
func executePipedCommandsWithContext(ctx context.Context, commands []string, env map[string]string, dir string, timeoutSeconds int) ([]byte, error) {
	if len(commands) == 0 {
		return nil, fmt.Errorf("no commands in group")
	}

	// Use the provided context as parent, adding timeout if specified
	if timeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	// Build environment
	environ := os.Environ()
	for key, value := range env {
		// Expand any environment variable references in the value
		expandedValue := os.ExpandEnv(value)
		environ = append(environ, fmt.Sprintf("%s=%s", key, expandedValue))
	}

	// For a single command, execute directly through shell
	if len(commands) == 1 {
		return executeCommand(ctx, commands[0], environ, dir, timeoutSeconds)
	}

	// For piped commands, join them with pipes and run as single shell command
	// This is more efficient and handles shell features properly
	pipelineCmd := strings.Join(commands, " | ")
	return executeCommand(ctx, pipelineCmd, environ, dir, timeoutSeconds)
}

// executeCommand executes a single command string through the user's shell.
//
// This function runs the command through the user's shell (obtained via getShell),
// ensuring aliases and shell configurations are available. The command runs in its
// own process group so all child processes can be terminated on timeout, preventing
// orphaned processes.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - cmdStr: Command string to execute
//   - environ: Full environment variable array for the command
//   - dir: Working directory for command execution
//   - timeoutSeconds: Maximum execution time in seconds (used for error messages)
//
// Returns:
//   - []byte: Combined stdout output from the command
//   - error: Any error that occurred during execution, including timeout errors
func executeCommand(ctx context.Context, cmdStr string, environ []string, dir string, timeoutSeconds int) ([]byte, error) {
	if strings.TrimSpace(cmdStr) == "" {
		return nil, fmt.Errorf("empty command")
	}

	shell, shellArgs := getShell()
	args := append(shellArgs, cmdStr)

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = environ
	if dir != "" {
		cmd.Dir = dir
	}

	// Run command in its own process group so we can kill all children on timeout
	setProcGroup(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded && timeoutSeconds > 0 {
			// Kill entire process group to ensure no orphaned child processes
			if killErr := killProcGroup(cmd); killErr != nil {
				warnings.Warnf("Warning: failed to kill process group on timeout: %v\n", killErr)
			}
			warnings.Warnf("command timed out after %d seconds\n", timeoutSeconds)
			return nil, fmt.Errorf("command timed out after %d seconds: %w", timeoutSeconds, err)
		}

		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg != "" {
			return nil, fmt.Errorf("%w: %s", err, errMsg)
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}

// parseCommandArgs parses a command string into arguments, respecting quotes.
//
// This function splits a command string into individual arguments while properly
// handling quoted strings (both single and double quotes) and escape sequences.
// Quoted strings are treated as single arguments even if they contain spaces.
//
// Parameters:
//   - cmdStr: Command string to parse into arguments
//
// Returns:
//   - []string: Array of parsed command arguments
func parseCommandArgs(cmdStr string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range cmdStr {
		// Handle escape sequences
		if r == '\\' && i+1 < len(cmdStr) {
			next := rune(cmdStr[i+1])
			if next == '"' || next == '\'' || next == '\\' || next == ' ' {
				current.WriteRune(next)
				continue
			}
		}

		// Handle quotes
		if (r == '"' || r == '\'') && (i == 0 || cmdStr[i-1] != '\\') {
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			} else {
				current.WriteRune(r)
			}
			continue
		}

		// Handle spaces outside quotes
		if !inQuote && (r == ' ' || r == '\t') {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	// Add final argument
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// BuildReplacements creates a replacement map for common template variables.
//
// This is a convenience function that creates a map with standard template
// variable keys (package, version, constraint) for use with command execution.
//
// Parameters:
//   - pkg: Package name to use for {{package}} template
//   - version: Version string to use for {{version}} template
//   - constraint: Version constraint to use for {{constraint}} template
//
// Returns:
//   - map[string]string: Map of template keys to replacement values
func BuildReplacements(pkg, version, constraint string) map[string]string {
	return map[string]string{
		"package":    pkg,
		"version":    version,
		"constraint": constraint,
	}
}
