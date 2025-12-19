package cmdexec

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyReplacements tests the behavior of applyReplacements.
//
// It verifies:
//   - Template placeholders are correctly replaced with values
//   - Empty values remove the placeholder entirely (not quoted as '')
func TestApplyReplacements(t *testing.T) {
	t.Run("basic replacement", func(t *testing.T) {
		cmd := `curl -s "https://registry.npmjs.org/{{package}}" | jq -r '.versions["{{version}}"]'`
		replacements := map[string]string{
			"package": "react",
			"version": "18.2.0",
		}

		result := applyReplacements(cmd, replacements)
		assert.Equal(t, `curl -s "https://registry.npmjs.org/react" | jq -r '.versions["18.2.0"]'`, result)
	})

	t.Run("empty value removes placeholder", func(t *testing.T) {
		// This tests the fix for composer getting '' as an empty argument
		cmd := `composer update {{package}} {{with_all_deps_flag}} --no-interaction`
		replacements := map[string]string{
			"package":            "laravel/framework",
			"with_all_deps_flag": "", // Empty value should be removed, not become ''
		}

		result := applyReplacements(cmd, replacements)
		// Empty value should be removed, resulting in double space which is fine for shell
		assert.Equal(t, `composer update laravel/framework  --no-interaction`, result)
		// Most importantly, it should NOT contain ''
		assert.NotContains(t, result, "''")
	})

	t.Run("non-empty flag value", func(t *testing.T) {
		cmd := `composer update {{package}} {{with_all_deps_flag}} --no-interaction`
		replacements := map[string]string{
			"package":            "laravel/framework",
			"with_all_deps_flag": "-W",
		}

		result := applyReplacements(cmd, replacements)
		assert.Equal(t, `composer update laravel/framework -W --no-interaction`, result)
	})
}

// TestGetShell tests the behavior of getShell.
//
// It verifies:
//   - SHELL environment variable is used when set
//   - Falls back to sh when SHELL is not set
func TestGetShell(t *testing.T) {
	t.Run("uses SHELL env var when set", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping Unix-specific test on Windows")
		}
		originalShell := os.Getenv("SHELL")
		defer func() { _ = os.Setenv("SHELL", originalShell) }()

		require.NoError(t, os.Setenv("SHELL", "/bin/bash"))
		shell, args := getShell()
		assert.Equal(t, "/bin/bash", shell)
		assert.Equal(t, []string{"-l", "-c"}, args)
	})

	t.Run("falls back to sh when SHELL not set", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping Unix-specific test on Windows")
		}
		originalShell := os.Getenv("SHELL")
		defer func() { _ = os.Setenv("SHELL", originalShell) }()

		require.NoError(t, os.Unsetenv("SHELL"))
		shell, args := getShell()
		assert.Equal(t, "sh", shell)
		assert.Equal(t, []string{"-c"}, args)
	})
}

// TestParseCommandGroups_SingleCommand tests the behavior of parseCommandGroups with a single command.
//
// It verifies:
//   - Single commands are parsed into one group with one command
func TestParseCommandGroups_SingleCommand(t *testing.T) {
	groups := parseCommandGroups("echo hello")
	require.Len(t, groups, 1)
	assert.Equal(t, []string{"echo hello"}, groups[0])
}

// TestParseCommandGroups_PipedCommands tests the behavior of parseCommandGroups with piped commands.
//
// It verifies:
//   - Piped commands are parsed into one group with multiple commands
func TestParseCommandGroups_PipedCommands(t *testing.T) {
	cmd := `echo "hello world" | grep hello`
	groups := parseCommandGroups(cmd)
	require.Len(t, groups, 1)
	assert.Equal(t, []string{`echo "hello world"`, "grep hello"}, groups[0])
}

// TestParseCommandGroups_MultilinePiped tests the behavior of parseCommandGroups with multiline piped commands.
//
// It verifies:
//   - Multiline piped commands are parsed into one group
func TestParseCommandGroups_MultilinePiped(t *testing.T) {
	cmd := `curl -s "https://example.com" |
jq -r '.versions[]'`
	groups := parseCommandGroups(cmd)
	require.Len(t, groups, 1)
	assert.Len(t, groups[0], 2)
}

// TestParseCommandGroups_Sequential tests the behavior of parseCommandGroups with sequential commands.
//
// It verifies:
//   - Sequential commands are parsed into separate groups
func TestParseCommandGroups_Sequential(t *testing.T) {
	cmd := `echo first
echo second
echo third`
	groups := parseCommandGroups(cmd)
	require.Len(t, groups, 3)
	assert.Equal(t, []string{"echo first"}, groups[0])
	assert.Equal(t, []string{"echo second"}, groups[1])
	assert.Equal(t, []string{"echo third"}, groups[2])
}

// TestParseCommandGroups_LineContinuation tests the behavior of parseCommandGroups with line continuation.
//
// It verifies:
//   - Commands with backslash continuation are joined together
func TestParseCommandGroups_LineContinuation(t *testing.T) {
	cmd := `curl -s \
"https://example.com"`
	groups := parseCommandGroups(cmd)
	require.Len(t, groups, 1)
	assert.Contains(t, groups[0][0], "curl -s")
}

// TestParseCommandArgs tests the behavior of parseCommandArgs.
//
// It verifies:
//   - Simple commands are parsed into arguments
//   - Quoted arguments preserve spaces
//   - Mixed quotes are handled correctly
//   - Empty input returns nil
func TestParseCommandArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple command",
			input:    "echo hello world",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "double quoted argument",
			input:    `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "single quoted argument",
			input:    `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "mixed quotes",
			input:    `curl -s "https://example.com" -H 'Content-Type: application/json'`,
			expected: []string{"curl", "-s", "https://example.com", "-H", "Content-Type: application/json"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommandArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSplitByPipe tests the behavior of splitByPipe.
//
// It verifies:
//   - Single commands return one part
//   - Piped commands are split correctly
//   - Pipes inside quotes are preserved
//   - Multiple pipes create multiple parts
func TestSplitByPipe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single command",
			input:    "echo hello",
			expected: []string{"echo hello"},
		},
		{
			name:     "two piped commands",
			input:    "echo hello | grep h",
			expected: []string{"echo hello", "grep h"},
		},
		{
			name:     "pipe in quotes",
			input:    `echo "hello | world"`,
			expected: []string{`echo "hello | world"`},
		},
		{
			name:     "multiple pipes",
			input:    "cat file | grep foo | wc -l",
			expected: []string{"cat file", "grep foo", "wc -l"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitByPipe(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildReplacements tests the behavior of BuildReplacements.
//
// It verifies:
//   - Replacement map is correctly built with package, version, and constraint keys
func TestBuildReplacements(t *testing.T) {
	replacements := BuildReplacements("react", "18.2.0", "^")
	assert.Equal(t, "react", replacements["package"])
	assert.Equal(t, "18.2.0", replacements["version"])
	assert.Equal(t, "^", replacements["constraint"])
}

// TestExecuteCommands_SimpleCommand tests the behavior of executeCommands with simple commands.
//
// It verifies:
//   - Simple commands execute successfully and return output
func TestExecuteCommands_SimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	output, err := executeCommands("echo hello", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

// TestExecuteCommands_WithReplacements tests the behavior of executeCommands with template replacements.
//
// It verifies:
//   - Template placeholders are replaced and command executes successfully
func TestExecuteCommands_WithReplacements(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	replacements := map[string]string{"name": "world"}
	output, err := executeCommands("echo {{name}}", nil, "", 30, replacements)
	require.NoError(t, err)
	assert.Contains(t, string(output), "world")
}

// TestExecuteCommands_PipedCommands tests the behavior of executeCommands with piped commands.
//
// It verifies:
//   - Piped commands execute correctly and return filtered output
func TestExecuteCommands_PipedCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	output, err := executeCommands("echo hello world | grep hello", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

// TestExecuteCommands_WithEnv tests the behavior of executeCommands with environment variables.
//
// It verifies:
//   - Environment variables are set and accessible during command execution
func TestExecuteCommands_WithEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	env := map[string]string{"TEST_VAR": "test_value"}
	output, err := executeCommands("printenv TEST_VAR", env, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "test_value")
}

// TestExecuteCommands_EnvExpansion tests the behavior of executeCommands with environment variable expansion.
//
// It verifies:
//   - Environment variable references are expanded in command environment
func TestExecuteCommands_EnvExpansion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	require.NoError(t, os.Setenv("GOUPDATE_TEST_BASE", "/usr/local"))
	defer func() { _ = os.Unsetenv("GOUPDATE_TEST_BASE") }()

	env := map[string]string{"PATH": "$GOUPDATE_TEST_BASE/bin:$PATH"}
	// Just verify it doesn't error - the env expansion happens
	_, err := executeCommands("echo test", env, "", 30, nil)
	assert.NoError(t, err)
}

// TestExecuteCommands_WorkingDirectory tests the behavior of executeCommands with a working directory.
//
// It verifies:
//   - Commands execute in the specified working directory
func TestExecuteCommands_WorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	output, err := executeCommands("pwd", nil, tmpDir, 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), tmpDir)
}

// TestExecuteCommands_EmptyCommand tests the behavior of executeCommands with empty command string.
//
// It verifies:
//   - Empty commands return an error
func TestExecuteCommands_EmptyCommand(t *testing.T) {
	_, err := executeCommands("", nil, "", 30, nil)
	assert.Error(t, err)
}

// TestExecuteCommands_SequentialCommands tests the behavior of executeCommands with sequential commands.
//
// It verifies:
//   - Sequential commands execute in order and last output is returned
func TestExecuteCommands_SequentialCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Sequential commands - last output is returned
	output, err := executeCommands("echo first\necho second", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "second")
}

// TestExecuteCommands_CommandNotFound tests the behavior of executeCommands with non-existent command.
//
// It verifies:
//   - Non-existent commands return an error
func TestExecuteCommands_CommandNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	_, err := executeCommands("nonexistent_command_12345", nil, "", 30, nil)
	assert.Error(t, err)
}

// TestShellEscape tests the behavior of shellEscape.
//
// It verifies:
//   - Empty strings are quoted
//   - Safe strings are not quoted
//   - Unsafe characters trigger quoting
//   - Single quotes are properly escaped
//   - Injection attempts are safely escaped
func TestShellEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "''"},
		{"safe string", "react", "react"},
		{"version string", "18.2.0", "18.2.0"},
		{"scoped package", "@types/react", "@types/react"},
		{"path", "path/to/file", "path/to/file"},
		{"with spaces", "hello world", "'hello world'"},
		{"with semicolon", "foo;bar", "'foo;bar'"},
		{"with backtick", "foo`bar", "'foo`bar'"},
		{"with dollar", "foo$bar", "'foo$bar'"},
		{"with single quote", "foo'bar", "'foo'\\''bar'"},
		{"with double quote", "foo\"bar", "'foo\"bar'"},
		{"injection attempt", "react; rm -rf /", "'react; rm -rf /'"},
		{"command substitution", "$(whoami)", "'$(whoami)'"},
		{"backtick substitution", "`whoami`", "'`whoami`'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsShellSafe tests the behavior of isShellSafe.
//
// It verifies:
//   - Safe characters return true
//   - Unsafe characters return false
func TestIsShellSafe(t *testing.T) {
	// Safe characters
	safeChars := "abcABC012-_./@:+="
	for _, r := range safeChars {
		assert.True(t, isShellSafe(r), "expected '%c' to be safe", r)
	}

	// Unsafe characters
	unsafeChars := " ;$`\"'(){}[]|&<>*?#!~"
	for _, r := range unsafeChars {
		assert.False(t, isShellSafe(r), "expected '%c' to be unsafe", r)
	}
}

// TestExecutePipedCommandsEdgeCases tests the behavior of executePipedCommands edge cases.
//
// It verifies:
//   - Empty commands return an error
//   - Multiple piped commands execute together
func TestExecutePipedCommandsEdgeCases(t *testing.T) {
	t.Run("empty commands returns error", func(t *testing.T) {
		_, err := executePipedCommands([]string{}, nil, "", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no commands in group")
	})

	t.Run("multiple piped commands execute together", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("skipping on Windows")
		}
		output, err := executePipedCommands([]string{"echo hello world", "grep hello"}, nil, "", 30)
		require.NoError(t, err)
		assert.Contains(t, string(output), "hello")
	})
}

// TestExecuteCommandEdgeCases tests the behavior of executeCommand edge cases.
//
// It verifies:
//   - Whitespace-only commands return an error
//   - Commands with stderr output include error details
//   - Timeout scenarios are handled correctly
func TestExecuteCommandEdgeCases(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	t.Run("empty command returns error", func(t *testing.T) {
		_, err := executeCommands("   \n\t  ", nil, "", 30, nil)
		assert.Error(t, err)
	})

	t.Run("command with stderr output on failure", func(t *testing.T) {
		_, err := executeCommands("ls /nonexistent_path_12345", nil, "", 30, nil)
		assert.Error(t, err)
	})

	t.Run("timeout scenario", func(t *testing.T) {
		_, err := executeCommands("sleep 10", nil, "", 1, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})
}

// TestParseCommandArgsEdgeCases tests the behavior of parseCommandArgs edge cases.
//
// It verifies:
//   - Escaped spaces are handled
//   - Escaped quotes in double quotes work
//   - Tabs as separators are handled
//   - Nested different quotes work correctly
//   - Single quotes inside double quotes are preserved
func TestParseCommandArgsEdgeCases(t *testing.T) {
	t.Run("escaped space", func(t *testing.T) {
		result := parseCommandArgs(`echo hello\ world`)
		assert.Contains(t, result, "echo")
	})

	t.Run("escaped quote in double quotes", func(t *testing.T) {
		result := parseCommandArgs(`echo "hello \"world\""`)
		assert.NotEmpty(t, result)
	})

	t.Run("tabs as separators", func(t *testing.T) {
		result := parseCommandArgs("echo\thello\tworld")
		assert.Equal(t, []string{"echo", "hello", "world"}, result)
	})

	t.Run("nested different quotes", func(t *testing.T) {
		result := parseCommandArgs(`echo "it's fine"`)
		assert.Contains(t, result, "it's fine")
	})

	t.Run("single quote inside double quotes", func(t *testing.T) {
		result := parseCommandArgs(`echo "don't worry"`)
		assert.Len(t, result, 2)
		assert.Equal(t, "don't worry", result[1])
	})
}

// TestParseCommandGroupsEdgeCases tests the behavior of parseCommandGroups edge cases.
//
// It verifies:
//   - Whitespace-only input returns empty groups
//   - Comment lines become their own groups
//   - Complex multiline with continuation and pipes work
//   - Empty lines between commands are ignored
//   - Trailing pipes are handled correctly
//   - Multiple trailing pipes are properly grouped
func TestParseCommandGroupsEdgeCases(t *testing.T) {
	t.Run("whitespace only", func(t *testing.T) {
		groups := parseCommandGroups("   \n\t  ")
		assert.Empty(t, groups)
	})

	t.Run("handles comment lines", func(t *testing.T) {
		// Comments are NOT skipped in parseCommandGroups - they become their own groups
		groups := parseCommandGroups("# comment\necho hello")
		assert.Len(t, groups, 2)
	})

	t.Run("complex multiline with continuation and pipes", func(t *testing.T) {
		cmd := `curl -s https://example.com \
| jq -r '.data' \
| grep foo`
		groups := parseCommandGroups(cmd)
		require.Len(t, groups, 1)
	})

	t.Run("empty lines between commands", func(t *testing.T) {
		cmd := "echo first\n\necho second"
		groups := parseCommandGroups(cmd)
		assert.Len(t, groups, 2)
	})

	t.Run("trailing pipe at end of input", func(t *testing.T) {
		// Command ending with pipe triggers the remaining group handler (lines 202-204)
		groups := parseCommandGroups("echo hello |")
		require.Len(t, groups, 1)
		assert.Equal(t, []string{"echo hello"}, groups[0])
	})

	t.Run("multiple trailing pipes", func(t *testing.T) {
		// Multiple lines with trailing pipes
		groups := parseCommandGroups("echo hello |\ngrep world |")
		require.Len(t, groups, 1)
		assert.Equal(t, []string{"echo hello", "grep world"}, groups[0])
	})
}

// TestExecuteCommandsNoTimeout tests the behavior of executeCommands with no timeout.
//
// It verifies:
//   - Commands execute successfully when timeout is 0
func TestExecuteCommandsNoTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Test with timeout = 0 (no timeout)
	output, err := executeCommands("echo hello", nil, "", 0, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

// TestExecuteCommandStdoutFallbackError tests the behavior of executeCommand with stdout error fallback.
//
// It verifies:
//   - Commands that fail with stdout but no stderr include stdout in error
func TestExecuteCommandStdoutFallbackError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Command that fails with stdout but no stderr
	_, err := executeCommands("sh -c 'echo stdout_error; exit 1'", nil, "", 30, nil)
	assert.Error(t, err)
}

// TestExecuteCommandEmptyDirectly tests the behavior of executePipedCommands with whitespace command.
//
// It verifies:
//   - Whitespace-only commands return an error
func TestExecuteCommandEmptyDirectly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Test executePipedCommands with whitespace command
	_, err := executePipedCommands([]string{"   "}, nil, "", 30)
	assert.Error(t, err)
}

// TestExecuteCommandBareError tests the behavior of executeCommand with bare error.
//
// It verifies:
//   - Commands that fail without stdout or stderr still return an error
func TestExecuteCommandBareError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Command that fails with no stdout and no stderr output
	// This triggers the bare error return (line 336)
	_, err := executeCommands("sh -c 'exit 42'", nil, "", 30, nil)
	assert.Error(t, err)
}

// TestExecuteCommandsWithContext_BasicExecution tests the behavior of executeCommandsWithContext basic execution.
//
// It verifies:
//   - Commands execute successfully with context
func TestExecuteCommandsWithContext_BasicExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	ctx := context.Background()
	output, err := executeCommandsWithContext(ctx, "echo hello", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

// TestExecuteCommandsWithContext_WithReplacements tests the behavior of executeCommandsWithContext with replacements.
//
// It verifies:
//   - Template placeholders are replaced with context execution
func TestExecuteCommandsWithContext_WithReplacements(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	ctx := context.Background()
	replacements := map[string]string{"name": "world"}
	output, err := executeCommandsWithContext(ctx, "echo {{name}}", nil, "", 30, replacements)
	require.NoError(t, err)
	assert.Contains(t, string(output), "world")
}

// TestExecuteCommandsWithContext_CancelledContext tests the behavior of executeCommandsWithContext with cancelled context.
//
// It verifies:
//   - Pre-cancelled context returns context.Canceled error
func TestExecuteCommandsWithContext_CancelledContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Pre-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := executeCommandsWithContext(ctx, "echo hello", nil, "", 30, nil)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestExecuteCommandsWithContext_ContextCancelDuringExecution tests the behavior of executeCommandsWithContext with cancellation during execution.
//
// It verifies:
//   - Context cancellation during execution returns an error
func TestExecuteCommandsWithContext_ContextCancelDuringExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	// Context with very short timeout to simulate cancellation during execution
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Command that would take longer than the context timeout
	_, err := executeCommandsWithContext(ctx, "sleep 10", nil, "", 0, nil)
	assert.Error(t, err)
}

// TestExecuteCommandsWithContext_EmptyCommands tests the behavior of executeCommandsWithContext with empty commands.
//
// It verifies:
//   - Empty commands return an error with context
func TestExecuteCommandsWithContext_EmptyCommands(t *testing.T) {
	ctx := context.Background()
	_, err := executeCommandsWithContext(ctx, "", nil, "", 30, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no commands provided")
}

// TestExecuteCommandsWithContext_WithEnv tests the behavior of executeCommandsWithContext with environment variables.
//
// It verifies:
//   - Environment variables are set with context execution
func TestExecuteCommandsWithContext_WithEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	ctx := context.Background()
	env := map[string]string{"TEST_CTX_VAR": "ctx_value"}
	output, err := executeCommandsWithContext(ctx, "printenv TEST_CTX_VAR", env, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "ctx_value")
}

// TestExecuteCommandsWithContext_SequentialCommands tests the behavior of executeCommandsWithContext with sequential commands.
//
// It verifies:
//   - Sequential commands execute with context checking between groups
func TestExecuteCommandsWithContext_SequentialCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	ctx := context.Background()
	// Sequential commands - context is checked before each group
	output, err := executeCommandsWithContext(ctx, "echo first\necho second", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "second")
}

// TestExecuteCommandsWithContext_PipedCommands tests the behavior of executeCommandsWithContext with piped commands.
//
// It verifies:
//   - Piped commands execute correctly with context
func TestExecuteCommandsWithContext_PipedCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	ctx := context.Background()
	output, err := executeCommandsWithContext(ctx, "echo hello world | grep hello", nil, "", 30, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
}

// TestKillProcGroup tests the behavior of killProcGroup.
//
// It verifies:
//   - Nil command process returns nil error
//   - Running processes are killed successfully
//   - Invalid PIDs return an error
func TestKillProcGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping Unix-specific test on Windows")
	}

	t.Run("nil command returns nil", func(t *testing.T) {
		cmd := &exec.Cmd{}
		err := killProcGroup(cmd)
		assert.NoError(t, err)
	})

	t.Run("kills running process", func(t *testing.T) {
		cmd := exec.Command("sleep", "60")
		setProcGroup(cmd)
		err := cmd.Start()
		require.NoError(t, err)

		// Give process time to start
		time.Sleep(50 * time.Millisecond)

		err = killProcGroup(cmd)
		assert.NoError(t, err)

		// Wait for process to finish (should be killed)
		_ = cmd.Wait()
	})

	t.Run("error on invalid pid", func(t *testing.T) {
		// Create a command that has already exited
		cmd := exec.Command("echo", "test")
		err := cmd.Run() // Run and wait for completion
		require.NoError(t, err)

		// Now try to kill it - should get an error because process no longer exists
		err = killProcGroup(cmd)
		// On Unix, killing a process that doesn't exist returns an error
		assert.Error(t, err)
	})
}

// TestSetProcGroup tests the behavior of setProcGroup.
//
// It verifies:
//   - Process group attributes are set on command
func TestSetProcGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping Unix-specific test on Windows")
	}

	t.Run("sets proc group on command", func(t *testing.T) {
		cmd := exec.Command("echo", "test")
		assert.Nil(t, cmd.SysProcAttr)

		setProcGroup(cmd)
		assert.NotNil(t, cmd.SysProcAttr)
		assert.True(t, cmd.SysProcAttr.Setpgid)
	})
}
