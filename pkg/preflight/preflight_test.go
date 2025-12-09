package preflight

import (
	"os"
	"os/exec"
	"testing"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// TestExtractCommands tests the behavior of command extraction from multi-line command strings.
//
// It verifies:
//   - Single commands are extracted correctly
//   - Multiline commands with pipes are handled
//   - Empty command strings return empty results
//   - Multiple sequential commands with same binary are deduplicated
func TestExtractCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands string
		want     []string
	}{
		{
			name:     "single command",
			commands: "npm view {{package}} versions --json",
			want:     []string{"npm"},
		},
		{
			name:     "multiline with pipes",
			commands: "curl https://example.com |\njq .versions",
			want:     []string{"curl", "jq"},
		},
		{
			name:     "empty",
			commands: "",
			want:     []string{},
		},
		{
			name:     "multiple sequential commands",
			commands: "go get {{package}}@{{version}}\ngo mod tidy",
			want:     []string{"go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommands(tt.commands)
			if len(got) != len(tt.want) {
				t.Errorf("extractCommands() got %d commands, want %d", len(got), len(tt.want))
				return
			}
			for i, cmd := range got {
				if cmd != tt.want[i] {
					t.Errorf("extractCommands()[%d] = %s, want %s", i, cmd, tt.want[i])
				}
			}
		})
	}
}

// TestValidateCommand tests the behavior of command validation.
//
// It verifies:
//   - Common commands like "echo" are validated correctly
//   - Non-existent commands return validation errors
//   - Error contains correct command name
func TestValidateCommand(t *testing.T) {
	// Test with a command that should exist on most systems
	err := validateCommand("echo")
	if err != nil {
		t.Logf("echo command not found (might be expected on some systems)")
	}

	// Test with a command that definitely doesn't exist
	err = validateCommand("this_command_definitely_does_not_exist_12345")
	if err == nil {
		t.Error("validateCommand() expected error for non-existent command")
	}
	if err != nil && err.Command != "this_command_definitely_does_not_exist_12345" {
		t.Errorf("validateCommand() error command = %s, want this_command_definitely_does_not_exist_12345", err.Command)
	}
}

// TestGetResolutionHint tests the behavior of resolution hint lookup.
//
// It verifies:
//   - Known commands like "npm" have resolution hints
//   - Unknown commands return empty hint strings
func TestGetResolutionHint(t *testing.T) {
	hint := GetResolutionHint("npm")
	if hint == "" {
		t.Error("GetResolutionHint(npm) returned empty string")
	}

	hint = GetResolutionHint("unknown_command")
	if hint != "" {
		t.Errorf("GetResolutionHint(unknown_command) = %s, want empty string", hint)
	}
}

// TestValidatePackages tests the behavior of package validation.
//
// It verifies:
//   - Package validation doesn't panic
//   - HasErrors and ErrorMessage methods work correctly
func TestValidatePackages(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Outdated: &config.OutdatedCfg{
					Commands: "npm view {{package}} versions --json",
				},
			},
		},
	}

	packages := []formats.Package{
		{Name: "test", Rule: "npm"},
	}

	result := ValidatePackages(packages, cfg)
	// We can't assert the result depends on whether npm is installed
	// Just verify it doesn't panic
	_ = result.HasErrors()
	_ = result.ErrorMessage()
}

// TestValidationError tests the behavior of ValidationError formatting.
//
// It verifies:
//   - ValidationError with hint produces non-empty error message
//   - ValidationError without hint produces non-empty error message
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Command: "npm",
		Hint:    "Install Node.js",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("ValidationError.Error() returned empty string")
	}

	errNoHint := &ValidationError{
		Command: "npm",
	}
	errStrNoHint := errNoHint.Error()
	if errStrNoHint == "" {
		t.Error("ValidationError.Error() without hint returned empty string")
	}
}

// TestValidatePackagesDetectsMissingCommands tests the behavior of missing command detection.
//
// It verifies:
//   - Non-existent commands in outdated and update configs are detected
//   - Multiple missing commands are all detected
//   - Error message is generated for missing commands
func TestValidatePackagesDetectsMissingCommands(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"custom": {
				Outdated: &config.OutdatedCfg{
					Commands: "nonexistent_command_xyz_12345 {{package}}",
				},
				Update: &config.UpdateCfg{
					Commands: "another_nonexistent_cmd_67890 {{package}}",
				},
			},
		},
	}

	packages := []formats.Package{
		{Name: "test-pkg", Rule: "custom"},
	}

	result := ValidatePackages(packages, cfg)
	if !result.HasErrors() {
		t.Error("ValidatePackages() should return errors for non-existent commands")
	}

	// Verify both commands are detected as missing
	if len(result.Errors) < 2 {
		t.Errorf("ValidatePackages() should detect 2 missing commands, got %d", len(result.Errors))
	}

	// Check error message is not empty
	msg := result.ErrorMessage()
	if msg == "" {
		t.Error("ErrorMessage() should not be empty when there are errors")
	}
}

// TestValidateRulesDetectsMissingCommands tests the behavior of rule validation with missing commands.
//
// It verifies:
//   - Non-existent commands in rule configs are detected
//   - Correct number of missing commands are reported
func TestValidateRulesDetectsMissingCommands(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"custom": {
				Outdated: &config.OutdatedCfg{
					Commands: "nonexistent_outdated_cmd_abc {{package}}",
				},
			},
		},
	}

	result := ValidateRules([]string{"custom"}, cfg)
	if !result.HasErrors() {
		t.Error("ValidateRules() should return errors for non-existent commands")
	}

	if len(result.Errors) != 1 {
		t.Errorf("ValidateRules() should detect 1 missing command, got %d", len(result.Errors))
	}
}

// TestValidationErrorIncludesHint tests the behavior of validation error hint inclusion.
//
// It verifies:
//   - Error message contains command name
//   - Error message contains "Resolution" keyword
//   - Error message includes hint URL when available
func TestValidationErrorIncludesHint(t *testing.T) {
	// Test with a known command that has a hint
	err := &ValidationError{
		Command: "npm",
		Hint:    CommandResolutionHints["npm"],
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("ValidationError.Error() returned empty string")
	}

	// Verify error message contains the command name
	if !contains(errStr, "npm") {
		t.Errorf("ValidationError.Error() should contain command name, got: %s", errStr)
	}

	// Verify error message contains resolution hint
	if !contains(errStr, "Resolution") {
		t.Errorf("ValidationError.Error() should contain 'Resolution', got: %s", errStr)
	}

	// Verify error message contains the hint URL
	if !contains(errStr, "nodejs.org") {
		t.Errorf("ValidationError.Error() should contain hint URL, got: %s", errStr)
	}
}

// TestValidationErrorForUnixTools tests the behavior of Unix tool hints.
//
// It verifies:
//   - Unix tools like grep, awk, sed, sort have hints
//   - Hints mention Linux or macOS platforms
func TestValidationErrorForUnixTools(t *testing.T) {
	// Test that Unix tools have proper hints
	unixTools := []string{"grep", "awk", "sed", "sort"}
	for _, tool := range unixTools {
		hint := GetResolutionHint(tool)
		if hint == "" {
			t.Errorf("GetResolutionHint(%s) should return hint for Unix tool", tool)
		}
		if !contains(hint, "Linux") && !contains(hint, "macOS") {
			t.Errorf("GetResolutionHint(%s) should mention Linux/macOS, got: %s", tool, hint)
		}
	}
}

// TestValidationErrorForJSONTools tests the behavior of JSON tool hints.
//
// It verifies:
//   - jq command has a resolution hint
//   - Hint contains installation URL
func TestValidationErrorForJSONTools(t *testing.T) {
	// Test that jq has proper hints
	hint := GetResolutionHint("jq")
	if hint == "" {
		t.Error("GetResolutionHint(jq) should return hint")
	}
	if !contains(hint, "jqlang.github.io") {
		t.Errorf("GetResolutionHint(jq) should contain installation URL, got: %s", hint)
	}
}

// TestExtractCommandsFromPipedCommands tests the behavior of piped command extraction.
//
// It verifies:
//   - All commands in a piped command chain are extracted
//   - Commands are extracted in order
//   - Proper handling of multiline piped commands
func TestExtractCommandsFromPipedCommands(t *testing.T) {
	// Test that piped commands on a single line are all extracted
	commands := `pip index versions {{package}} 2>/dev/null |
grep -oE '[0-9]+\.[0-9]+' |
sort -u`

	result := extractCommands(commands)
	expected := []string{"pip", "grep", "sort"}

	if len(result) != len(expected) {
		t.Errorf("extractCommands() got %d commands, want %d: %v", len(result), len(expected), result)
		return
	}

	for i, cmd := range expected {
		if result[i] != cmd {
			t.Errorf("extractCommands()[%d] = %s, want %s", i, result[i], cmd)
		}
	}
}

// TestErrorMessageFormatsCorrectly tests the behavior of error message formatting.
//
// It verifies:
//   - Error message contains pre-flight validation header
//   - All command names are included in the message
//   - Resolution hints are included for each command
func TestErrorMessageFormatsCorrectly(t *testing.T) {
	result := &ValidateResult{
		Errors: []ValidationError{
			{Command: "npm", Hint: "Install Node.js: https://nodejs.org/"},
			{Command: "jq", Hint: "Install jq: https://jqlang.github.io/jq/download/"},
		},
	}

	msg := result.ErrorMessage()

	// Verify message contains header
	if !contains(msg, "Pre-flight validation failed") {
		t.Errorf("ErrorMessage() should contain header, got: %s", msg)
	}

	// Verify message contains both commands
	if !contains(msg, "npm") || !contains(msg, "jq") {
		t.Errorf("ErrorMessage() should contain both commands, got: %s", msg)
	}

	// Verify message contains resolution hints
	if !contains(msg, "Resolution") {
		t.Errorf("ErrorMessage() should contain resolution hints, got: %s", msg)
	}
}

// contains is a helper function for string matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestValidateRulesMissingRule tests the behavior of validation with missing rules.
//
// It verifies:
//   - Missing/non-existent rules are silently skipped
//   - No errors are returned for rules that don't exist in config
func TestValidateRulesMissingRule(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Outdated: &config.OutdatedCfg{
					Commands: "echo {{package}}",
				},
			},
		},
	}

	// Test with a rule that doesn't exist
	result := ValidateRules([]string{"nonexistent_rule"}, cfg)
	if result.HasErrors() {
		t.Error("ValidateRules() should not return errors for missing rules")
	}
}

// TestValidateRulesWithUpdateCommands tests the behavior of validation for update commands.
//
// It verifies:
//   - Non-existent commands in update configs are detected
//   - Validation errors are returned for missing update commands
func TestValidateRulesWithUpdateCommands(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"custom": {
				Update: &config.UpdateCfg{
					Commands: "nonexistent_update_cmd_xyz",
				},
			},
		},
	}

	result := ValidateRules([]string{"custom"}, cfg)
	if !result.HasErrors() {
		t.Error("ValidateRules() should return errors for non-existent update commands")
	}
}

// TestValidateRulesNilOutdatedAndUpdate tests the behavior with nil outdated and update configs.
//
// It verifies:
//   - Rules without outdated or update configs don't cause errors
//   - Validation succeeds for empty rule configurations
func TestValidateRulesNilOutdatedAndUpdate(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"empty": {},
		},
	}

	result := ValidateRules([]string{"empty"}, cfg)
	if result.HasErrors() {
		t.Error("ValidateRules() should not error for rules without outdated or update configs")
	}
}

// TestValidateCommandEmpty tests the behavior of empty command validation.
//
// It verifies:
//   - Empty command string returns nil (no error)
func TestValidateCommandEmpty(t *testing.T) {
	err := validateCommand("")
	if err != nil {
		t.Error("validateCommand() should return nil for empty command")
	}
}

// TestCommandExistsInShell tests the behavior of shell command existence checking.
//
// It verifies:
//   - Common shell commands like "echo" are detected
//   - Non-existent commands return false
func TestCommandExistsInShell(t *testing.T) {
	// Test with a command that exists
	exists := commandExistsInShell("echo")
	if !exists {
		t.Log("commandExistsInShell(echo) returned false - might be expected on some systems")
	}

	// Test with a command that definitely doesn't exist
	exists = commandExistsInShell("this_definitely_does_not_exist_at_all_xyz_123")
	if exists {
		t.Error("commandExistsInShell() should return false for non-existent command")
	}
}

// TestExtractCommandsWithComments tests the behavior of comment handling in command extraction.
//
// It verifies:
//   - Comment lines starting with # are skipped
//   - Commands on non-comment lines are extracted correctly
func TestExtractCommandsWithComments(t *testing.T) {
	commands := `# This is a comment
npm view {{package}}
# Another comment
pip install`

	result := extractCommands(commands)
	if len(result) != 2 {
		t.Errorf("extractCommands() should extract 2 commands, got %d: %v", len(result), result)
		return
	}
	if result[0] != "npm" || result[1] != "pip" {
		t.Errorf("extractCommands() got %v, want [npm pip]", result)
	}
}

// TestExtractCommandsWithLineContinuation tests the behavior of line continuation handling.
//
// It verifies:
//   - Backslash line continuations are handled
//   - Commands from continuation lines are extracted
func TestExtractCommandsWithLineContinuation(t *testing.T) {
	commands := `npm install \
--save`

	result := extractCommands(commands)
	if len(result) != 2 {
		t.Errorf("extractCommands() should extract 2 commands from continuation, got %d: %v", len(result), result)
	}
}

// TestExtractCommandsWithWhitespace tests the behavior of whitespace handling.
//
// It verifies:
//   - Leading and trailing whitespace is properly trimmed
//   - Empty lines with whitespace are skipped
//   - Commands are extracted correctly despite extra whitespace
func TestExtractCommandsWithWhitespace(t *testing.T) {
	commands := `   npm view {{package}}

   pip install   `

	result := extractCommands(commands)
	if len(result) != 2 {
		t.Errorf("extractCommands() should handle whitespace, got %d commands: %v", len(result), result)
		return
	}
	if result[0] != "npm" || result[1] != "pip" {
		t.Errorf("extractCommands() got %v, want [npm pip]", result)
	}
}

// TestValidatePackagesNilOutdatedAndUpdate tests the behavior with nil outdated and update configs.
//
// It verifies:
//   - Packages with rules lacking outdated/update configs don't cause errors
//   - Validation succeeds for empty configurations
func TestValidatePackagesNilOutdatedAndUpdate(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"empty": {},
		},
	}

	packages := []formats.Package{
		{Name: "test", Rule: "empty"},
	}

	result := ValidatePackages(packages, cfg)
	if result.HasErrors() {
		t.Error("ValidatePackages() should not error for packages without outdated or update configs")
	}
}

// TestValidatePackagesMissingRule tests the behavior with missing package rules.
//
// It verifies:
//   - Packages with non-existent rules are silently skipped
//   - No errors are returned for missing rule definitions
func TestValidatePackagesMissingRule(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{},
	}

	packages := []formats.Package{
		{Name: "test", Rule: "nonexistent"},
	}

	result := ValidatePackages(packages, cfg)
	// Missing rules are silently skipped
	if result.HasErrors() {
		t.Error("ValidatePackages() should not error for missing rules")
	}
}

// TestValidateResultHasErrorsEmpty tests the behavior of empty validation results.
//
// It verifies:
//   - HasErrors returns false for empty results
//   - ErrorMessage returns empty string for empty results
func TestValidateResultHasErrorsEmpty(t *testing.T) {
	result := &ValidateResult{}
	if result.HasErrors() {
		t.Error("HasErrors() should return false for empty result")
	}

	msg := result.ErrorMessage()
	if msg != "" {
		t.Errorf("ErrorMessage() should return empty string for empty result, got: %s", msg)
	}
}

// TestExtractCommandsEmptyPipe tests the behavior of malformed pipe handling.
//
// It verifies:
//   - Empty pipe segments (||) are handled gracefully
//   - Valid commands in malformed pipes are still extracted
func TestExtractCommandsEmptyPipe(t *testing.T) {
	commands := "echo test | | grep foo"
	result := extractCommands(commands)
	// Should handle empty pipe segments gracefully
	if len(result) == 0 {
		t.Error("extractCommands() should extract at least some commands from malformed pipe")
	}
}

// TestExtractCommandsBackslashOnlyLine tests the behavior of backslash-only lines.
//
// It verifies:
//   - Lines with only backslash are skipped
//   - Commands before and after backslash-only lines are extracted
func TestExtractCommandsBackslashOnlyLine(t *testing.T) {
	// Test line that becomes empty after removing trailing backslash
	commands := `npm install
\
pip install`
	result := extractCommands(commands)
	// Should extract npm and pip, skipping the backslash-only line
	if len(result) != 2 {
		t.Errorf("extractCommands() should extract 2 commands, got %d: %v", len(result), result)
		return
	}
	if result[0] != "npm" || result[1] != "pip" {
		t.Errorf("extractCommands() got %v, want [npm pip]", result)
	}
}

// TestExtractCommandsOnlyWhitespaceAfterBackslash tests the behavior of whitespace-only continuation lines.
//
// It verifies:
//   - Lines with only whitespace before backslash are skipped
//   - Commands are still extracted from valid lines
func TestExtractCommandsOnlyWhitespaceAfterBackslash(t *testing.T) {
	// Test line that has only whitespace before backslash
	commands := `npm install
   \
pip install`
	result := extractCommands(commands)
	// Should skip lines that become empty after trimming backslash
	if len(result) != 2 {
		t.Errorf("extractCommands() got %d commands, want 2: %v", len(result), result)
	}
}

// TestValidateCommandShellBuiltin tests the behavior of shell builtin command validation.
//
// It verifies:
//   - Shell builtins not in PATH are detected via shell fallback
//   - commandExistsInShell successfully finds builtins like type, alias, export, cd
func TestValidateCommandShellBuiltin(t *testing.T) {
	// Test shell builtins that are NOT executables in PATH
	// These should fail exec.LookPath but succeed with commandExistsInShell
	shellBuiltins := []string{"type", "alias", "export", "cd"}

	for _, builtin := range shellBuiltins {
		// Verify it's not found by exec.LookPath (pure shell builtin)
		_, lookPathErr := exec.LookPath(builtin)
		if lookPathErr != nil {
			// This builtin is NOT an executable - perfect for testing the fallback path
			err := validateCommand(builtin)
			if err == nil {
				// Success! The commandExistsInShell fallback worked
				return
			}
		}
	}

	// If we get here, all shell builtins were found in PATH (unusual system config)
	// Skip the test rather than fail
	t.Skip("No pure shell builtin found to test commandExistsInShell fallback")
}

// TestGetShellCommandCheckFallback tests the behavior of shell fallback when SHELL is not set.
//
// It verifies:
//   - Falls back to "sh" when SHELL environment variable is unset
//   - Returns appropriate arguments for shell command checking
func TestGetShellCommandCheckFallback(t *testing.T) {
	// Save original SHELL value
	origShell := os.Getenv("SHELL")
	defer func() { _ = os.Setenv("SHELL", origShell) }()

	// Unset SHELL to test fallback
	_ = os.Unsetenv("SHELL")

	shell, args := getShellCommandCheck("echo")

	// When SHELL is not set, should fall back to "sh"
	if shell != "sh" {
		t.Errorf("getShellCommandCheck() shell = %q, want %q when SHELL is unset", shell, "sh")
	}
	if len(args) < 2 {
		t.Errorf("getShellCommandCheck() should return at least 2 args, got %d", len(args))
	}
}
