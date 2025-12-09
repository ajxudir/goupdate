package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateConfigFile_ValidConfig tests the behavior of ValidateConfigFile with valid config.
//
// It verifies:
//   - Valid config passes validation without errors
func TestValidateConfigFile_ValidConfig(t *testing.T) {
	yaml := `
rules:
  npm:
    manager: js
    include: ["**/package.json"]
    format: json
`
	result := ValidateConfigFile([]byte(yaml))
	assert.False(t, result.HasErrors(), "Valid config should not have errors")
}

// TestValidateConfigFile_UnknownField tests the behavior of ValidateConfigFile with unknown fields.
//
// It verifies:
//   - Unknown fields are detected and reported
func TestValidateConfigFile_UnknownField(t *testing.T) {
	yaml := `
rules:
  npm:
    manager: js
    badfield: value
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors(), "Should detect unknown field")
	assert.Contains(t, result.Errors[0].Message, "unknown field")
}

// TestValidateConfigFile_UnknownFieldWithSchemaHints tests the behavior of ValidateConfigFile with schema hints for unknown fields.
//
// It verifies:
//   - Unknown fields provide helpful schema hints in error messages
func TestValidateConfigFile_UnknownFieldWithSchemaHints(t *testing.T) {
	yaml := `
rules:
  npm:
    outdated:
      command: "npm view"
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors(), "Should detect unknown field 'command'")

	err := result.Errors[0]
	assert.Contains(t, err.Message, "command")

	// Check verbose error contains schema hints
	verbose := err.VerboseError()
	assert.Contains(t, verbose, "commands", "Should suggest 'commands' as valid key")
}

// TestValidateConfigFile_TypoSuggestion tests the behavior of ValidateConfigFile with typo suggestions.
//
// It verifies:
//   - Common typos are detected and correct field names are suggested
func TestValidateConfigFile_TypoSuggestion(t *testing.T) {
	yaml := `
rule:
  npm:
    manager: js
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	// Should suggest 'rules' instead of 'rule'
	assert.Contains(t, result.Errors[0].Message, "rules")
}

// TestValidateConfigFile_YamlSyntaxError tests the behavior of ValidateConfigFile with YAML syntax errors.
//
// It verifies:
//   - YAML syntax errors are detected and reported
func TestValidateConfigFile_YamlSyntaxError(t *testing.T) {
	yaml := `
rules:
  npm:
    manager: [invalid yaml
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	assert.Contains(t, strings.ToLower(result.Errors[0].Message), "yaml")
}

// TestValidateConfigFile_TypeMismatchError tests the behavior of ValidateConfigFile with type mismatch errors.
//
// It verifies:
//   - Type mismatches are detected and reported
func TestValidateConfigFile_TypeMismatchError(t *testing.T) {
	// This should trigger "cannot unmarshal" error
	yaml := `
rules:
  npm:
    enabled: not_a_boolean
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	// The error should be about type mismatch
	assert.NotEmpty(t, result.Errors)
}

// TestValidateConfigFile_GenericError tests the behavior of ValidateConfigFile with generic errors.
//
// It verifies:
//   - Generic errors are handled and reported
func TestValidateConfigFile_GenericError(t *testing.T) {
	// Empty file returns EOF error which doesn't contain common patterns
	// This tests the else branch for non-specific errors
	result := ValidateConfigFile([]byte(""))
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Message, "EOF")
}

// TestExtractFieldAndType tests the behavior of extractFieldAndType.
//
// It verifies:
//   - Field names and type names are correctly extracted from error messages
func TestExtractFieldAndType(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		wantField string
		wantType  string
	}{
		{
			name:      "unknown field in Config",
			errMsg:    "yaml: unmarshal errors:\n  line 5: field rule not found in type config.Config",
			wantField: "rule",
			wantType:  "Config",
		},
		{
			name:      "unknown field in PackageManagerCfg",
			errMsg:    "yaml: unmarshal errors:\n  line 10: field command not found in type config.OutdatedCfg",
			wantField: "command",
			wantType:  "OutdatedCfg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, typeName := extractFieldAndType(tt.errMsg)
			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantType, typeName)
		})
	}
}

// TestExtractLineNumber tests the behavior of extractLineNumber.
//
// It verifies:
//   - Line numbers are correctly extracted from error messages
//   - Missing line numbers return 0
func TestExtractLineNumber(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   int
	}{
		{
			name:   "line number present",
			errMsg: "yaml: unmarshal errors:\n  line 15: field foo not found",
			want:   15,
		},
		{
			name:   "no line number",
			errMsg: "yaml: unmarshal errors: some error",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLineNumber(tt.errMsg)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSuggestSimilarField tests the behavior of suggestSimilarField.
//
// It verifies:
//   - Common typos are mapped to correct field names
//   - Unknown fields return empty suggestions
func TestSuggestSimilarField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		typeName string
		want     string
	}{
		{
			name:     "rule -> rules",
			field:    "rule",
			typeName: "Config",
			want:     "rules",
		},
		{
			name:     "command -> commands",
			field:    "command",
			typeName: "OutdatedCfg",
			want:     "commands",
		},
		{
			name:     "includes -> include",
			field:    "includes",
			typeName: "PackageManagerCfg",
			want:     "include",
		},
		{
			name:     "no suggestion",
			field:    "completely_wrong",
			typeName: "Config",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestSimilarField(tt.field, tt.typeName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidationError_VerboseError tests the behavior of ValidationError.VerboseError.
//
// It verifies:
//   - Verbose error messages include all available information
//   - Expected types, valid keys, and doc sections are included
func TestValidationError_VerboseError(t *testing.T) {
	err := ValidationError{
		Field:      "rules.npm.outdated.command",
		Message:    "unknown field 'command'",
		Expected:   "string",
		ValidKeys:  "commands, env, format, extraction",
		DocSection: "outdated",
	}

	verbose := err.VerboseError()

	assert.Contains(t, verbose, "rules.npm.outdated.command")
	assert.Contains(t, verbose, "unknown field 'command'")
	assert.Contains(t, verbose, "Expected: string")
	assert.Contains(t, verbose, "Valid keys: commands, env, format, extraction")
	assert.Contains(t, verbose, "docs/configuration.md#outdated")
}

// TestValidationResult_VerboseErrorMessages tests the behavior of ValidationResult.VerboseErrorMessages.
//
// It verifies:
//   - Multiple validation errors are formatted correctly with verbose details
func TestValidationResult_VerboseErrorMessages(t *testing.T) {
	result := &ValidationResult{
		Errors: []ValidationError{
			{
				Message:   "unknown field 'foo'",
				ValidKeys: "bar, baz",
			},
		},
	}

	verbose := result.VerboseErrorMessages()
	assert.Contains(t, verbose, "unknown field 'foo'")
	assert.Contains(t, verbose, "Valid keys: bar, baz")
}

// TestValidateConfigStruct_EmptyGroupName tests the behavior of validateConfigStruct with empty group names.
//
// It verifies:
//   - Empty group names are detected and reported as errors
func TestValidateConfigStruct_EmptyGroupName(t *testing.T) {
	cfg := &Config{
		Groups: map[string]GroupCfg{
			"": {Packages: []string{"pkg1"}},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Message, "group name cannot be empty")
}

// TestValidateConfigStruct_EmptyIncremental tests the behavior of validateConfigStruct with empty incremental package names.
//
// It verifies:
//   - Empty incremental package names are detected and reported as errors
func TestValidateConfigStruct_EmptyIncremental(t *testing.T) {
	cfg := &Config{
		Incremental: []string{"valid", "", "also-valid"},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Message, "incremental package name cannot be empty")
}

// TestValidateSystemTests_ValidConfig tests the behavior of system tests validation with valid config.
//
// It verifies:
//   - Valid system tests configuration passes validation
func TestValidateSystemTests_ValidConfig(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			RunMode: SystemTestRunModeAfterAll,
			Tests: []SystemTestCfg{
				{
					Name:           "unit-tests",
					Commands:       "npm test",
					TimeoutSeconds: 120,
				},
			},
		},
	}

	result := cfg.Validate()
	assert.False(t, result.HasErrors(), "Valid system_tests config should not have errors")
}

// TestValidateSystemTests_InvalidRunMode tests the behavior of system tests validation with invalid run mode.
//
// It verifies:
//   - Invalid run mode values are detected and reported with valid options
func TestValidateSystemTests_InvalidRunMode(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			RunMode: "invalid_mode",
			Tests: []SystemTestCfg{
				{
					Name:     "test1",
					Commands: "echo hello",
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())

	var foundError bool
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "invalid run_mode") {
			foundError = true
			assert.Contains(t, err.Expected, "after_each")
			assert.Contains(t, err.Expected, "after_all")
			assert.Contains(t, err.Expected, "none")
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid run_mode")
}

// TestValidateSystemTests_EmptyTestName tests the behavior of system tests validation with empty test names.
//
// It verifies:
//   - Empty test names are detected and reported as errors
func TestValidateSystemTests_EmptyTestName(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			Tests: []SystemTestCfg{
				{
					Name:     "", // Empty name
					Commands: "npm test",
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())

	var foundError bool
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "test name is required") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about empty test name")
}

// TestValidateSystemTests_EmptyCommands tests the behavior of system tests validation with empty commands.
//
// It verifies:
//   - Empty test commands are detected and reported as errors
func TestValidateSystemTests_EmptyCommands(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			Tests: []SystemTestCfg{
				{
					Name:     "test1",
					Commands: "", // Empty commands
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())

	var foundError bool
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "test commands cannot be empty") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about empty commands")
}

// TestValidateSystemTests_WhitespaceOnlyCommands tests the behavior of system tests validation with whitespace-only commands.
//
// It verifies:
//   - Whitespace-only commands are detected and reported as errors
func TestValidateSystemTests_WhitespaceOnlyCommands(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			Tests: []SystemTestCfg{
				{
					Name:     "test1",
					Commands: "   \n\t   ", // Whitespace only
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())

	var foundError bool
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "test commands cannot be empty") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about whitespace-only commands")
}

// TestValidateSystemTests_NegativeTimeout tests the behavior of system tests validation with negative timeout.
//
// It verifies:
//   - Negative timeout values are detected and reported as errors
func TestValidateSystemTests_NegativeTimeout(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			Tests: []SystemTestCfg{
				{
					Name:           "test1",
					Commands:       "npm test",
					TimeoutSeconds: -1, // Negative timeout
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())

	var foundError bool
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "timeout must be positive") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about negative timeout")
}

// TestValidateSystemTests_NoTestsDefined_Warning tests the behavior of system tests validation with no tests defined.
//
// It verifies:
//   - Warning is issued when no tests are defined but system tests are enabled
func TestValidateSystemTests_NoTestsDefined_Warning(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			Tests: []SystemTestCfg{}, // Empty tests list
		},
	}

	result := cfg.Validate()
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "no tests defined")
}

// TestValidateSystemTests_NoTestsWithNoneMode_NoWarning tests the behavior of system tests validation with no tests and none mode.
//
// It verifies:
//   - No warning when tests list is empty and run_mode is none
func TestValidateSystemTests_NoTestsWithNoneMode_NoWarning(t *testing.T) {
	falseVal := false
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			RunMode:      SystemTestRunModeNone,
			RunPreflight: &falseVal,
			Tests:        []SystemTestCfg{}, // Empty tests list
		},
	}

	result := cfg.Validate()
	// Should not have warning when run_mode is "none" and run_preflight is false
	assert.Empty(t, result.Warnings)
}

// TestValidateSystemTests_MultipleErrors tests the behavior of system tests validation with multiple errors.
//
// It verifies:
//   - Multiple validation errors are all detected and reported
func TestValidateSystemTests_MultipleErrors(t *testing.T) {
	cfg := &Config{
		SystemTests: &SystemTestsCfg{
			RunMode: "invalid",
			Tests: []SystemTestCfg{
				{
					Name:           "",
					Commands:       "",
					TimeoutSeconds: -5,
				},
				{
					Name:     "valid-test",
					Commands: "npm test",
				},
			},
		},
	}

	result := cfg.Validate()
	assert.True(t, result.HasErrors())
	// Should have multiple errors: invalid run_mode, empty name, empty commands, negative timeout
	assert.GreaterOrEqual(t, len(result.Errors), 4)
}

// TestValidateConfigFile_SystemTestsTypos tests the behavior of ValidateConfigFile with system tests typos.
//
// It verifies:
//   - Typos in system tests configuration are detected
func TestValidateConfigFile_SystemTestsTypos(t *testing.T) {
	yaml := `
system_tests:
  run_mode: after_all
  test:
    - name: test1
      command: "npm test"
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors(), "Should detect typos in system_tests")
}

// TestValidateConfigFile_ValidSystemTests tests the behavior of ValidateConfigFile with valid system tests.
//
// It verifies:
//   - Valid system tests configuration passes validation
func TestValidateConfigFile_ValidSystemTests(t *testing.T) {
	yaml := `
rules:
  npm:
    manager: js
    include: ["**/package.json"]
    format: json

system_tests:
  run_preflight: true
  run_mode: after_all
  stop_on_fail: true
  tests:
    - name: unit-tests
      commands: npm test
      timeout_seconds: 120
    - name: e2e-tests
      commands: |
        npm run build
        npm run test:e2e
      timeout_seconds: 300
      continue_on_fail: true
      env:
        CI: "true"
`
	result := ValidateConfigFile([]byte(yaml))
	assert.False(t, result.HasErrors(), "Valid config with system_tests should not have errors: %v", result.Errors)
}

// TestSystemTestsCfg_IsRunPreflight tests the behavior of SystemTestsCfg.IsRunPreflight.
//
// It verifies:
//   - Default value is true
//   - Explicit true value works
//   - Explicit false value works
func TestSystemTestsCfg_IsRunPreflight(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *SystemTestsCfg
		expected bool
	}{
		{
			name:     "default (nil)",
			cfg:      &SystemTestsCfg{},
			expected: true,
		},
		{
			name: "explicit true",
			cfg: &SystemTestsCfg{
				RunPreflight: &trueVal,
			},
			expected: true,
		},
		{
			name: "explicit false",
			cfg: &SystemTestsCfg{
				RunPreflight: &falseVal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.IsRunPreflight())
		})
	}
}

// TestSystemTestsCfg_IsStopOnFail tests the behavior of SystemTestsCfg.IsStopOnFail.
//
// It verifies:
//   - Default value is true
//   - Explicit true value works
//   - Explicit false value works
func TestSystemTestsCfg_IsStopOnFail(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		cfg      *SystemTestsCfg
		expected bool
	}{
		{
			name:     "default (nil)",
			cfg:      &SystemTestsCfg{},
			expected: true,
		},
		{
			name: "explicit true",
			cfg: &SystemTestsCfg{
				StopOnFail: &trueVal,
			},
			expected: true,
		},
		{
			name: "explicit false",
			cfg: &SystemTestsCfg{
				StopOnFail: &falseVal,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.IsStopOnFail())
		})
	}
}

// TestSystemTestsCfg_GetRunMode tests the behavior of SystemTestsCfg.GetRunMode.
//
// It verifies:
//   - Default value is after_all
//   - Different run modes are returned correctly
func TestSystemTestsCfg_GetRunMode(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *SystemTestsCfg
		expected string
	}{
		{
			name:     "default (empty)",
			cfg:      &SystemTestsCfg{},
			expected: "after_all",
		},
		{
			name: "after_each",
			cfg: &SystemTestsCfg{
				RunMode: SystemTestRunModeAfterEach,
			},
			expected: "after_each",
		},
		{
			name: "after_all",
			cfg: &SystemTestsCfg{
				RunMode: SystemTestRunModeAfterAll,
			},
			expected: "after_all",
		},
		{
			name: "none",
			cfg: &SystemTestsCfg{
				RunMode: SystemTestRunModeNone,
			},
			expected: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.cfg.GetRunMode())
		})
	}
}

// TestValidationError_Error tests the behavior of ValidationError.Error.
//
// It verifies:
//   - Error message without field shows message only
//   - Error message with field shows field and message
func TestValidationError_Error(t *testing.T) {
	t.Run("without field", func(t *testing.T) {
		err := ValidationError{
			Message: "test error message",
		}
		assert.Equal(t, "test error message", err.Error())
	})

	t.Run("with field", func(t *testing.T) {
		err := ValidationError{
			Field:   "rules.npm.format",
			Message: "invalid format",
		}
		assert.Equal(t, "rules.npm.format: invalid format", err.Error())
	})
}

// TestValidationResult_ErrorMessages tests the behavior of ValidationResult.ErrorMessages.
//
// It verifies:
//   - Multiple errors are formatted correctly
//   - Error messages include header
func TestValidationResult_ErrorMessages(t *testing.T) {
	result := &ValidationResult{
		Errors: []ValidationError{
			{Message: "error 1"},
			{Message: "error 2"},
		},
	}

	msgs := result.ErrorMessages()
	assert.Contains(t, msgs, "error 1")
	assert.Contains(t, msgs, "error 2")
	assert.Contains(t, msgs, "Configuration validation failed")
}

// TestValidateConfigFileStrict tests the behavior of ValidateConfigFileStrict.
//
// It verifies:
//   - Warnings are converted to errors in strict mode
//   - Invalid configs return errors
func TestValidateConfigFileStrict(t *testing.T) {
	t.Run("converts warnings to errors", func(t *testing.T) {
		yaml := `
rules:
  npm:
    manager: js
    include: ["**/package.json"]
    format: json
    outdated:
      commands: echo test
`
		result := ValidateConfigFileStrict([]byte(yaml))
		// In strict mode: warnings should be converted to errors
		// The missing {{package}} placeholder warning becomes an error
		assert.True(t, result.HasErrors(), "strict mode should have errors when warnings exist")
		assert.Empty(t, result.Warnings, "strict mode should have no warnings (converted to errors)")
		// Verify that the placeholder warning was converted to an error
		foundPlaceholderError := false
		for _, err := range result.Errors {
			if strings.Contains(err.Message, "{{package}}") {
				foundPlaceholderError = true
				break
			}
		}
		assert.True(t, foundPlaceholderError, "should have converted placeholder warning to error")
	})

	t.Run("returns errors for invalid config", func(t *testing.T) {
		yaml := `
rules:
  npm:
    badfield: value
`
		result := ValidateConfigFileStrict([]byte(yaml))
		assert.True(t, result.HasErrors())
	})
}

// TestValidateOutdated tests the behavior of validateOutdated.
//
// It verifies:
//   - Missing package placeholder generates warning
//   - Placeholder presence prevents warning
//   - Empty commands don't generate warning
func TestValidateOutdated(t *testing.T) {
	t.Run("warns on missing package placeholder", func(t *testing.T) {
		result := &ValidationResult{}
		outdated := &OutdatedCfg{
			Commands: "npm view",
		}
		validateOutdated("rules.npm.outdated", outdated, result)
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "{{package}}")
	})

	t.Run("no warning when commands has placeholder", func(t *testing.T) {
		result := &ValidationResult{}
		outdated := &OutdatedCfg{
			Commands: "npm view {{package}}",
		}
		validateOutdated("rules.npm.outdated", outdated, result)
		assert.Empty(t, result.Warnings)
	})

	t.Run("no warning when commands is empty", func(t *testing.T) {
		result := &ValidationResult{}
		outdated := &OutdatedCfg{
			Commands: "",
		}
		validateOutdated("rules.npm.outdated", outdated, result)
		assert.Empty(t, result.Warnings)
	})
}

// TestValidatePackageOverride tests the behavior of validatePackageOverride.
//
// It verifies:
//   - Empty constraint generates warning
//   - Non-empty constraint doesn't generate warning
//   - Nil constraint doesn't generate warning
func TestValidatePackageOverride(t *testing.T) {
	t.Run("warns on empty constraint", func(t *testing.T) {
		result := &ValidationResult{}
		emptyConstraint := ""
		override := &PackageOverrideCfg{
			Constraint: &emptyConstraint,
		}
		validatePackageOverride("rules.npm.package_overrides.lodash", override, result)
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "empty constraint")
	})

	t.Run("no warning for non-empty constraint", func(t *testing.T) {
		result := &ValidationResult{}
		constraint := "^"
		override := &PackageOverrideCfg{
			Constraint: &constraint,
		}
		validatePackageOverride("rules.npm.package_overrides.lodash", override, result)
		assert.Empty(t, result.Warnings)
	})

	t.Run("no warning for nil constraint", func(t *testing.T) {
		result := &ValidationResult{}
		override := &PackageOverrideCfg{
			Constraint: nil,
		}
		validatePackageOverride("rules.npm.package_overrides.lodash", override, result)
		assert.Empty(t, result.Warnings)
	})
}

// TestExtractExpectedType tests the behavior of extractExpectedType.
//
// It verifies:
//   - Expected type is extracted from unmarshal errors
//   - Missing type information returns empty string
func TestExtractExpectedType(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   string
	}{
		{
			name:   "extracts type from unmarshal error",
			errMsg: "yaml: cannot unmarshal !!str `hello` into bool",
			want:   "bool",
		},
		{
			name:   "extracts type with newline",
			errMsg: "yaml: cannot unmarshal !!seq into string\nmore text",
			want:   "string",
		},
		{
			name:   "no 'into' in message",
			errMsg: "yaml: some other error",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractExpectedType(tt.errMsg)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtractUnknownField tests the behavior of extractUnknownField.
//
// It verifies:
//   - Unknown field names are extracted from error messages
func TestExtractUnknownField(t *testing.T) {
	// Use same format as TestExtractFieldAndType tests
	errMsg := "yaml: unmarshal errors:\n  line 5: field rule not found in type config.Config"
	field := extractUnknownField(errMsg)
	assert.Equal(t, "rule", field)
}

// TestSuggestSimilarFieldKebabCase tests the behavior of suggestSimilarField with kebab-case.
//
// It verifies:
//   - Kebab-case fields are converted to snake_case suggestions
func TestSuggestSimilarFieldKebabCase(t *testing.T) {
	// Test kebab-case to snake_case conversion
	suggestion := suggestSimilarField("working-dir", "Config")
	assert.Equal(t, "working_dir", suggestion)
}

// TestExtractFieldAndTypeEdgeCases tests the behavior of extractFieldAndType with edge cases.
//
// It verifies:
//   - Field at end of string is handled
//   - Missing field keyword is handled
//   - Type without trailing content is handled
func TestExtractFieldAndTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		wantField string
		wantType  string
	}{
		{
			name:      "field at end of string",
			errMsg:    "yaml: unmarshal errors: field badfield",
			wantField: "badfield",
			wantType:  "",
		},
		{
			name:      "no field keyword",
			errMsg:    "yaml: unmarshal errors: some other error",
			wantField: "",
			wantType:  "",
		},
		{
			name:      "type without trailing content",
			errMsg:    "yaml: field foo not found in type config.PackageManagerCfg",
			wantField: "foo",
			wantType:  "PackageManagerCfg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, typeName := extractFieldAndType(tt.errMsg)
			assert.Equal(t, tt.wantField, field)
			assert.Equal(t, tt.wantType, typeName)
		})
	}
}

// TestValidationErrorVerboseErrorEmpty tests the behavior of ValidationError.VerboseError with minimal content.
//
// It verifies:
//   - Simple error messages work without extra sections
func TestValidationErrorVerboseErrorEmpty(t *testing.T) {
	err := ValidationError{
		Message: "simple error",
	}

	verbose := err.VerboseError()
	// Should just contain the message without extra sections
	assert.Contains(t, verbose, "simple error")
	assert.NotContains(t, verbose, "Expected:")
	assert.NotContains(t, verbose, "Valid keys:")
	assert.NotContains(t, verbose, "See:")
}

// TestValidateRuleEdgeCases tests the behavior of validateRule with edge cases.
//
// It verifies:
//   - Package overrides with empty constraints generate warnings
//   - Outdated configs with missing placeholders generate warnings
//   - Update configs are validated
//   - Empty include patterns generate errors
//   - Empty group names generate errors
//   - Groups with no packages generate warnings
//   - Lock files without file patterns generate errors
//   - Lock files without format or extraction generate errors
//   - Empty package override keys generate errors
func TestValidateRuleEdgeCases(t *testing.T) {
	t.Run("rule with package_overrides validation", func(t *testing.T) {
		emptyConstraint := ""
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					PackageOverrides: map[string]PackageOverrideCfg{
						"lodash": {
							Constraint: &emptyConstraint,
						},
					},
				},
			},
		}
		result := cfg.Validate()
		// Should have warning about empty constraint
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "empty constraint")
	})

	t.Run("rule with outdated validation", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					Outdated: &OutdatedCfg{
						Commands: "npm view", // Missing {{package}}
					},
				},
			},
		}
		result := cfg.Validate()
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "{{package}}")
	})

	t.Run("rule with update validation", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					Update: &UpdateCfg{
						Commands: "npm install",
					},
				},
			},
		}
		result := cfg.Validate()
		// Update validation is minimal - no errors expected
		assert.False(t, result.HasErrors())
	})

	t.Run("empty include pattern", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json", ""},
					Format:  "json",
				},
			},
		}
		result := cfg.Validate()
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Errors[0].Message, "include pattern cannot be empty")
	})

	t.Run("empty group name in rule", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					Groups: map[string]GroupCfg{
						"": {Packages: []string{"pkg1"}},
					},
				},
			},
		}
		result := cfg.Validate()
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Errors[0].Message, "group name cannot be empty")
	})

	t.Run("group with no packages warning", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					Groups: map[string]GroupCfg{
						"empty-group": {Packages: []string{}},
					},
				},
			},
		}
		result := cfg.Validate()
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "group has no packages")
	})

	t.Run("lock file without files error", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					LockFiles: []LockFileCfg{
						{Format: "json", Files: []string{}},
					},
				},
			},
		}
		result := cfg.Validate()
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Errors[0].Message, "lock file must specify at least one file pattern")
	})

	t.Run("lock file without format or extraction error", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					LockFiles: []LockFileCfg{
						{Files: []string{"package-lock.json"}},
					},
				},
			},
		}
		result := cfg.Validate()
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Errors[0].Message, "lock file must specify format or extraction")
	})

	t.Run("empty package override key error", func(t *testing.T) {
		cfg := &Config{
			Rules: map[string]PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Format:  "json",
					PackageOverrides: map[string]PackageOverrideCfg{
						"": {Ignore: true},
					},
				},
			},
		}
		result := cfg.Validate()
		assert.True(t, result.HasErrors())
		assert.Contains(t, result.Errors[0].Message, "package override key cannot be empty")
	})
}

// TestSuggestSimilarFieldMoreCases tests the behavior of suggestSimilarField with additional cases.
//
// It verifies:
//   - Unknown types return empty suggestions
//   - Unknown fields in known types return empty suggestions
//   - Kebab-case conversion works via schema
//   - Non-existent kebab-case fields return empty suggestions
func TestSuggestSimilarFieldMoreCases(t *testing.T) {
	t.Run("unknown type returns empty", func(t *testing.T) {
		suggestion := suggestSimilarField("foo", "UnknownType")
		assert.Equal(t, "", suggestion)
	})

	t.Run("unknown field in known type returns empty", func(t *testing.T) {
		suggestion := suggestSimilarField("unknown_field", "Config")
		assert.Equal(t, "", suggestion)
	})

	t.Run("kebab-case to snake_case via schema", func(t *testing.T) {
		// "exclude-versions" is NOT in commonTypos but "exclude_versions" IS in configSchema
		suggestion := suggestSimilarField("exclude-versions", "Config")
		assert.Equal(t, "exclude_versions", suggestion)
	})

	t.Run("kebab-case not in schema returns empty", func(t *testing.T) {
		// A kebab-case field that doesn't exist even after conversion
		suggestion := suggestSimilarField("non-existent-field", "Config")
		assert.Equal(t, "", suggestion)
	})
}

// TestValidationResultNoErrors tests the behavior of ValidationResult with no errors.
//
// It verifies:
//   - HasErrors returns false when no errors exist
//   - ErrorMessages returns empty string
//   - VerboseErrorMessages returns empty string
func TestValidationResultNoErrors(t *testing.T) {
	result := &ValidationResult{}
	assert.False(t, result.HasErrors())
	assert.Empty(t, result.ErrorMessages())
	assert.Empty(t, result.VerboseErrorMessages())
}

// TestValidationErrorEmptyField tests the behavior of ValidationError with empty field.
//
// It verifies:
//   - Error message is returned when field is empty
func TestValidationErrorEmptyField(t *testing.T) {
	err := ValidationError{
		Message: "test error",
	}
	// Error() should return Message
	assert.Equal(t, "test error", err.Error())
}

// TestValidateConfigFile_TypeMismatchRulesAsString tests the behavior of ValidateConfigFile with type mismatch.
//
// It verifies:
//   - Type mismatch for rules field is detected
func TestValidateConfigFile_TypeMismatchRulesAsString(t *testing.T) {
	// This YAML has a type mismatch - rules should be a map but we give it a string
	yaml := `
rules: "this should be a map"
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	assert.Contains(t, result.Errors[0].Message, "cannot unmarshal")
}

// TestValidateConfigFile_UnknownFieldWithTypeName tests the behavior of ValidateConfigFile with unknown field and type name.
//
// It verifies:
//   - Unknown fields in nested types are detected with type information
func TestValidateConfigFile_UnknownFieldWithTypeName(t *testing.T) {
	// Unknown field at a nested type that isn't in configSchema but typeName is extracted
	yaml := `
rules:
  npm:
    versioning:
      unknownversioning: test
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	err := result.Errors[0]
	assert.Contains(t, err.Message, "unknown field")
}

// TestValidateConfigFile_UnknownFieldInOverrideType tests the behavior of ValidateConfigFile with unknown field in override type.
//
// It verifies:
//   - Unknown fields in override types include type information in error
func TestValidateConfigFile_UnknownFieldInOverrideType(t *testing.T) {
	// Unknown field in a type not in configSchema (OutdatedOverrideCfg)
	// This hits the typeName != "" branch where Expected is set
	yaml := `
rules:
  npm:
    package_overrides:
      lodash:
        outdated:
          badfield: test
`
	result := ValidateConfigFile([]byte(yaml))
	assert.True(t, result.HasErrors())
	err := result.Errors[0]
	assert.Contains(t, err.Message, "unknown field")
	// The Expected field should be set with the type name
	assert.Contains(t, err.Expected, "OutdatedOverrideCfg")
}

// TestExtractFieldAndType_WithSpaceAfterType tests the behavior of extractFieldAndType with space after type.
//
// It verifies:
//   - Type names followed by space or newline are extracted correctly
func TestExtractFieldAndType_WithSpaceAfterType(t *testing.T) {
	// Test error message where type name is followed by space/newline
	errMsg := "yaml: unmarshal errors:\n  line 5: field foo not found in type config.OutdatedCfg more text"
	field, typeName := extractFieldAndType(errMsg)
	assert.Equal(t, "foo", field)
	assert.Equal(t, "OutdatedCfg", typeName)
}
