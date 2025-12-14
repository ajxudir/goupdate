package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CHAOS TESTS FOR CONFIGURATION VALIDATION
// =============================================================================
//
// These tests deliberately try to break the configuration system by providing
// malicious, malformed, or edge-case inputs. The goal is to ensure the config
// loader and validator handle unexpected inputs gracefully without:
// - Panicking
// - Memory exhaustion (YAML bomb, huge files)
// - ReDoS (complex regex patterns)
// - Security bypasses (path traversal, absolute paths)
// - Undefined behavior on edge cases
//
// =============================================================================

// -----------------------------------------------------------------------------
// YAML PARSING EDGE CASES
// -----------------------------------------------------------------------------

// TestChaos_YAMLParsing_MalformedInputs tests YAML parsing with malformed inputs.
func TestChaos_YAMLParsing_MalformedInputs(t *testing.T) {
	testCases := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{"empty", "", false},
		{"whitespace_only", "   \n\t\n   ", true}, // Tab character in whitespace causes YAML error
		{"just_comment", "# This is just a comment", false},
		{"unclosed_bracket", "rules: [npm", true},
		{"unclosed_brace", "rules: {npm:", true},
		{"missing_colon", "rules npm", true},
		{"invalid_indentation", "rules:\n  npm:\n test: true", true},
		{"tab_indent", "rules:\n\tnpm:\n\t\tinclude: ['*.json']", true}, // Go YAML library rejects raw tabs
		{"null_value", "rules: null", false},
		{"boolean_as_map", "rules: true", true},
		{"number_as_map", "rules: 123", true},
		{"array_as_map", "rules: [1, 2, 3]", true},
		{"duplicate_keys", "rules:\n  npm:\n    include: ['a']\n  npm:\n    include: ['b']", true}, // Go YAML library rejects duplicate keys
		{"anchor_reference", "rules:\n  npm: &anchor\n    include: ['*.json']\n  npm2: *anchor", false},
		{"invalid_anchor", "rules:\n  npm: *undefined_anchor", true},
		{"multiline_string", "rules:\n  npm:\n    include: |\n      *.json\n      *.js", true}, // include expects array
		{"empty_map_key", "rules:\n  '':\n    include: ['*.json']", false},
		{"numeric_map_key", "rules:\n  123:\n    include: ['*.json']", false},
		{"special_char_key", "rules:\n  'npm:@scope':\n    include: ['*.json']", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfigData([]byte(tc.yaml))

			if tc.expectError {
				assert.Error(t, err, "should error for malformed YAML: %s", tc.name)
			} else {
				assert.NoError(t, err, "should not error for valid YAML: %s", tc.name)
			}

			// Should never panic
			if cfg != nil {
				t.Logf("Parsed config for %s: rules=%d", tc.name, len(cfg.Rules))
			}
		})
	}
}

// TestChaos_YAMLParsing_NestedDepth tests deeply nested YAML structures.
func TestChaos_YAMLParsing_NestedDepth(t *testing.T) {
	// Create deeply nested structure
	t.Run("deep_nesting_100_levels", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("rules:\n")
		for i := 0; i < 100; i++ {
			sb.WriteString(strings.Repeat("  ", i+1))
			sb.WriteString("nested:\n")
		}

		_, err := loadConfigData([]byte(sb.String()))
		// Should not panic, may error due to type mismatch
		t.Logf("Deep nesting result: %v", err)
	})

	t.Run("wide_map_1000_keys", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("rules:\n")
		for i := 0; i < 1000; i++ {
			// Generate unique key using full number
			sb.WriteString("  rule_")
			sb.WriteString(strings.Repeat("0", 4-len(fmt.Sprintf("%d", i))))
			sb.WriteString(fmt.Sprintf("%d", i))
			sb.WriteString(":\n    include: ['*.json']\n")
		}

		cfg, err := loadConfigData([]byte(sb.String()))
		assert.NoError(t, err, "should handle wide maps")
		if cfg != nil {
			t.Logf("Loaded %d rules", len(cfg.Rules))
			assert.Equal(t, 1000, len(cfg.Rules), "should have 1000 rules")
		}
	})
}

// -----------------------------------------------------------------------------
// SECURITY POLICY CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_Security_PathTraversal tests various path traversal bypass attempts.
func TestChaos_Security_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	parentDir := filepath.Dir(tmpDir)

	// Create a parent config
	parentConfig := filepath.Join(parentDir, "parent-test-config.yml")
	require.NoError(t, os.WriteFile(parentConfig, []byte("rules: {}"), 0644))
	defer os.Remove(parentConfig)

	testCases := []struct {
		name        string
		extends     []string
		allowTraversal bool
		expectError bool
		errorContain string
	}{
		{"single_dotdot", []string{"../parent-test-config.yml"}, false, true, "path traversal not allowed"},
		{"double_dotdot", []string{"../../parent-test-config.yml"}, false, true, "path traversal not allowed"},
		{"dotdot_with_dir", []string{"subdir/../parent-test-config.yml"}, false, true, "path traversal not allowed"},
		{"dotdot_in_middle", []string{"dir/../parent-test-config.yml"}, false, true, "path traversal not allowed"},
		{"hidden_dotdot", []string{"./subdir/../../parent-test-config.yml"}, false, true, "path traversal not allowed"},
		{"urlencoded_dotdot", []string{"%2e%2e/parent-test-config.yml"}, false, false, ""},  // Not URL-decoded
		{"unicode_dotdot", []string{"\u002e\u002e/parent-test-config.yml"}, false, true, "path traversal not allowed"},  // Unicode periods
		{"allowed_with_flag", []string{"../parent-test-config.yml"}, true, false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Extends: tc.extends,
				Rules:   make(map[string]PackageManagerCfg),
				Security: &SecurityCfg{
					AllowPathTraversal: tc.allowTraversal,
				},
			}
			cfg.SetRootConfig(true)

			_, err := processExtendsSecure(cfg, tmpDir, cfg)

			if tc.expectError {
				assert.Error(t, err, "should error for: %s", tc.name)
				if err != nil && tc.errorContain != "" {
					assert.Contains(t, err.Error(), tc.errorContain,
						"error should contain expected message for: %s", tc.name)
				}
			} else {
				// May still error if file doesn't exist, but not for path traversal
				if err != nil {
					assert.NotContains(t, err.Error(), "path traversal",
						"should not be a path traversal error for: %s", tc.name)
				}
			}
		})
	}
}

// TestChaos_Security_AbsolutePaths tests absolute path security bypasses.
func TestChaos_Security_AbsolutePaths(t *testing.T) {
	tmpDir := t.TempDir()
	testConfig := filepath.Join(tmpDir, "test.yml")
	require.NoError(t, os.WriteFile(testConfig, []byte("rules: {}"), 0644))

	testCases := []struct {
		name          string
		extends       []string
		allowAbsolute bool
		expectError   bool
		errorContain  string
	}{
		{"absolute_path", []string{testConfig}, false, true, "absolute paths not allowed"},
		{"root_path", []string{"/etc/goupdate.yml"}, false, true, "absolute paths not allowed"},
		{"allowed_absolute", []string{testConfig}, true, false, ""},
		// Windows-style paths (if on Windows)
		{"drive_letter", []string{"C:\\config.yml"}, false, false, ""}, // Not absolute on Unix
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Extends: tc.extends,
				Rules:   make(map[string]PackageManagerCfg),
				Security: &SecurityCfg{
					AllowAbsolutePaths: tc.allowAbsolute,
				},
			}
			cfg.SetRootConfig(true)

			_, err := processExtendsSecure(cfg, tmpDir, cfg)

			if tc.expectError {
				assert.Error(t, err, "should error for: %s", tc.name)
				if err != nil && tc.errorContain != "" {
					assert.Contains(t, err.Error(), tc.errorContain)
				}
			}
		})
	}
}

// TestChaos_Security_CyclicExtends tests cyclic extends detection.
func TestChaos_Security_CyclicExtends(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("direct_cycle_A_extends_A", func(t *testing.T) {
		configA := filepath.Join(tmpDir, "a.yml")
		require.NoError(t, os.WriteFile(configA, []byte(`extends: ["a.yml"]
rules: {}`), 0644))

		cfg, err := LoadConfig(configA, tmpDir)
		assert.Error(t, err, "should detect self-reference cycle")
		if err != nil {
			assert.Contains(t, err.Error(), "cyclic", "error should mention cyclic")
		}
		_ = cfg
	})

	t.Run("direct_cycle_A_B_A", func(t *testing.T) {
		configA := filepath.Join(tmpDir, "cycle-a.yml")
		configB := filepath.Join(tmpDir, "cycle-b.yml")

		require.NoError(t, os.WriteFile(configA, []byte(`extends: ["cycle-b.yml"]
rules: {}`), 0644))
		require.NoError(t, os.WriteFile(configB, []byte(`extends: ["cycle-a.yml"]
rules: {}`), 0644))

		cfg, err := LoadConfig(configA, tmpDir)
		assert.Error(t, err, "should detect A->B->A cycle")
		if err != nil {
			assert.Contains(t, err.Error(), "cyclic", "error should mention cyclic")
		}
		_ = cfg
	})

	t.Run("indirect_cycle_A_B_C_A", func(t *testing.T) {
		configA := filepath.Join(tmpDir, "chain-a.yml")
		configB := filepath.Join(tmpDir, "chain-b.yml")
		configC := filepath.Join(tmpDir, "chain-c.yml")

		require.NoError(t, os.WriteFile(configA, []byte(`extends: ["chain-b.yml"]
rules: {}`), 0644))
		require.NoError(t, os.WriteFile(configB, []byte(`extends: ["chain-c.yml"]
rules: {}`), 0644))
		require.NoError(t, os.WriteFile(configC, []byte(`extends: ["chain-a.yml"]
rules: {}`), 0644))

		cfg, err := LoadConfig(configA, tmpDir)
		assert.Error(t, err, "should detect A->B->C->A cycle")
		if err != nil {
			assert.Contains(t, err.Error(), "cyclic", "error should mention cyclic")
		}
		_ = cfg
	})
}

// TestChaos_Security_FileSizeLimit tests config file size limits.
func TestChaos_Security_FileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("file_at_limit", func(t *testing.T) {
		// Create file just under limit (10MB - 1KB)
		content := strings.Repeat("# comment\n", 1024*1024) // ~10MB of comments
		configPath := filepath.Join(tmpDir, "large.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		_, err := loadConfigFileWithLimit(configPath, 11*1024*1024) // 11MB limit
		// Should succeed or fail on parsing, not size
		t.Logf("Large file (under limit) result: %v", err)
	})

	t.Run("file_over_limit", func(t *testing.T) {
		// Create file over default limit
		content := strings.Repeat("# comment\n", 2*1024*1024) // ~20MB of comments
		configPath := filepath.Join(tmpDir, "huge.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		_, err := loadConfigFileWithLimit(configPath, DefaultMaxConfigFileSize)
		assert.Error(t, err, "should reject file over size limit")
		if err != nil {
			assert.Contains(t, err.Error(), "too large")
		}
	})
}

// -----------------------------------------------------------------------------
// REGEX PATTERN CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_RegexPatterns_Malformed tests malformed regex patterns.
func TestChaos_RegexPatterns_Malformed(t *testing.T) {
	testCases := []struct {
		name    string
		pattern string
		valid   bool
	}{
		{"valid_simple", `\.json$`, true},    // Escaped dot for literal match
		{"glob_star_invalid", "*.json", false}, // Glob pattern, not valid regex (unanchored *)
		{"valid_regex", `^package\.json$`, true},
		{"unclosed_bracket", "[a-z", false},
		{"unclosed_paren", "(abc", false},
		{"invalid_escape", `\`, false},
		{"invalid_quantifier", "a**", false},
		{"invalid_range", "[z-a]", false},
		{"empty_alternation", "a|", true}, // Valid in Go regex
		{"nested_groups", "((((a))))", true},
		{"lookbehind", `(?<=foo)bar`, false}, // Go RE2 doesn't support lookbehind
		{"lookahead", `foo(?=bar)`, false},   // Go RE2 doesn't support lookahead either
		{"backreference", `(a)\1`, false},    // Go RE2 doesn't support backreferences
		{"unicode_property", `\p{L}`, true},  // Go supports unicode properties
		{"named_group", `(?P<name>abc)`, true},
		{"atomic_group", `(?>abc)`, false}, // Go doesn't support atomic groups
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := regexp.Compile(tc.pattern)
			if tc.valid {
				assert.NoError(t, err, "pattern should be valid: %s", tc.pattern)
			} else {
				assert.Error(t, err, "pattern should be invalid: %s", tc.pattern)
			}
		})
	}
}

// TestChaos_RegexPatterns_ReDoSVulnerable tests potentially ReDoS-vulnerable patterns.
func TestChaos_RegexPatterns_ReDoSVulnerable(t *testing.T) {
	// These patterns are known to cause exponential backtracking in some engines
	// Go's RE2 is designed to be ReDoS-resistant, so these should complete quickly
	dangerousPatterns := []struct {
		name    string
		pattern string
		input   string
	}{
		{"evil_regex_1", `(a+)+b`, strings.Repeat("a", 30)},
		{"evil_regex_2", `(a|aa)+b`, strings.Repeat("a", 30)},
		{"evil_regex_3", `(a|a?)+b`, strings.Repeat("a", 30)},
		{"evil_regex_4", `(.*a){10}`, strings.Repeat("a", 30)},
		{"polynomial", `a*a*a*a*b`, strings.Repeat("a", 100)},
	}

	for _, tc := range dangerousPatterns {
		t.Run(tc.name, func(t *testing.T) {
			re, err := regexp.Compile(tc.pattern)
			if err != nil {
				t.Logf("Pattern %s is invalid in Go: %v", tc.name, err)
				return
			}

			// Should complete quickly due to RE2
			done := make(chan bool, 1)
			go func() {
				re.MatchString(tc.input)
				done <- true
			}()

			select {
			case <-done:
				t.Logf("Pattern %s completed (RE2 is ReDoS-resistant)", tc.name)
			case <-time.After(2 * time.Second):
				t.Errorf("Pattern %s took too long - potential ReDoS", tc.name)
			}
		})
	}
}

// TestChaos_RegexPatterns_LongPatterns tests very long regex patterns.
func TestChaos_RegexPatterns_LongPatterns(t *testing.T) {
	t.Run("pattern_at_complexity_limit", func(t *testing.T) {
		// Pattern at default complexity limit (1000 chars)
		pattern := strings.Repeat("a", DefaultMaxRegexComplexity)
		_, err := regexp.Compile(pattern)
		assert.NoError(t, err, "pattern at limit should be valid")
	})

	t.Run("pattern_over_complexity_limit", func(t *testing.T) {
		// Pattern over complexity limit
		pattern := strings.Repeat("a", DefaultMaxRegexComplexity*2)

		cfg := &Config{}
		maxComplexity := cfg.GetMaxRegexComplexity()

		if len(pattern) > maxComplexity {
			t.Logf("Pattern length %d exceeds complexity limit %d", len(pattern), maxComplexity)
		}
	})
}

// -----------------------------------------------------------------------------
// FIELD VALUE CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_FieldValues_SpecialCharacters tests special characters in config values.
func TestChaos_FieldValues_SpecialCharacters(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{"null_byte", "test\x00value"},
		{"newline", "test\nvalue"},
		{"carriage_return", "test\rvalue"},
		{"tab", "test\tvalue"},
		{"unicode_bom", "\xef\xbb\xbftest"},
		{"unicode_rtl", "test\u202evalue"},
		{"unicode_null", "test\u0000value"},
		{"emoji", "test\U0001F600value"},
		{"high_unicode", "test\U0010FFFFvalue"},
		{"shell_injection", "$(whoami)"},
		{"command_substitution", "`whoami`"},
		{"semicolon", "test;whoami"},
		{"pipe", "test|whoami"},
		{"redirect", "test>file"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			yaml := `rules:
  npm:
    include:
      - "` + tc.value + `"`

			cfg, err := loadConfigData([]byte(yaml))
			// Should not panic
			if err != nil {
				t.Logf("Error for %s: %v", tc.name, err)
			} else if cfg != nil && cfg.Rules["npm"].Include != nil {
				t.Logf("Include pattern for %s: %q", tc.name, cfg.Rules["npm"].Include[0])
			}
		})
	}
}

// TestChaos_FieldValues_VeryLongStrings tests very long string values.
func TestChaos_FieldValues_VeryLongStrings(t *testing.T) {
	t.Run("long_include_pattern", func(t *testing.T) {
		longPattern := strings.Repeat("a", 10000)
		yaml := `rules:
  npm:
    include:
      - "` + longPattern + `"`

		cfg, err := loadConfigData([]byte(yaml))
		assert.NoError(t, err, "should handle long patterns")
		if cfg != nil && cfg.Rules["npm"].Include != nil {
			assert.Equal(t, len(longPattern), len(cfg.Rules["npm"].Include[0]))
		}
	})

	t.Run("many_patterns", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("rules:\n  npm:\n    include:\n")
		for i := 0; i < 1000; i++ {
			sb.WriteString("      - 'pattern_")
			sb.WriteString(strings.Repeat("x", 100))
			sb.WriteString("'\n")
		}

		cfg, err := loadConfigData([]byte(sb.String()))
		assert.NoError(t, err, "should handle many patterns")
		if cfg != nil && cfg.Rules["npm"].Include != nil {
			assert.Equal(t, 1000, len(cfg.Rules["npm"].Include))
		}
	})
}

// -----------------------------------------------------------------------------
// GROUP CONFIGURATION CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_Groups_EdgeCases tests edge cases in group configuration.
// Groups in this config system are defined within rules as map[string][]string
// where the key is the group name and the value is a list of package names.
func TestChaos_Groups_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{
			name: "empty_group_name",
			yaml: `rules:
  npm:
    groups:
      '':
        - react
        - vue`,
			expectError: false,
		},
		{
			name: "numeric_group_name",
			yaml: `rules:
  npm:
    groups:
      123:
        - react
        - vue`,
			expectError: false,
		},
		{
			name: "special_char_group_name",
			yaml: `rules:
  npm:
    groups:
      'group:with:colons':
        - react
        - vue`,
			expectError: false,
		},
		{
			name: "empty_package_list",
			yaml: `rules:
  npm:
    groups:
      mygroup: []`,
			expectError: false,
		},
		{
			name: "single_package",
			yaml: `rules:
  npm:
    groups:
      mygroup:
        - react`,
			expectError: false,
		},
		{
			name: "many_groups",
			yaml: `rules:
  npm:
    groups:
      group1:
        - pkg1
      group2:
        - pkg2
      group3:
        - pkg3`,
			expectError: false,
		},
		{
			name: "unicode_group_name",
			yaml: `rules:
  npm:
    groups:
      '日本語':
        - react`,
			expectError: false,
		},
		{
			name: "very_long_group_name",
			yaml: `rules:
  npm:
    groups:
      'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa':
        - react`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfigData([]byte(tc.yaml))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if cfg != nil && cfg.Rules != nil {
				if npmRule, ok := cfg.Rules["npm"]; ok {
					t.Logf("Groups loaded for %s: %d", tc.name, len(npmRule.Groups))
				}
			}
		})
	}
}

// TestChaos_Groups_Membership_InvalidReferences tests invalid group references.
func TestChaos_Groups_Membership_InvalidReferences(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("reference_nonexistent_group", func(t *testing.T) {
		yaml := `rules:
  npm:
    include: ['*.json']
    group: nonexistent_group`

		configPath := filepath.Join(tmpDir, "invalid-group.yml")
		require.NoError(t, os.WriteFile(configPath, []byte(yaml), 0644))

		_, err := LoadConfig(configPath, tmpDir)
		// Should either error on invalid group or ignore unknown fields
		t.Logf("Invalid group reference result: %v", err)
	})
}

// -----------------------------------------------------------------------------
// SYSTEM TESTS CONFIGURATION CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_SystemTests_MalformedCommands tests malformed system test commands.
func TestChaos_SystemTests_MalformedCommands(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{
			name: "empty_command",
			yaml: `system_tests:
  run_mode: after_all
  tests:
    - name: empty
      commands: ''
rules: {}`,
		},
		{
			name: "null_command",
			yaml: `system_tests:
  run_mode: after_all
  tests:
    - name: null
      commands: null
rules: {}`,
		},
		{
			name: "multiline_command",
			yaml: `system_tests:
  run_mode: after_all
  tests:
    - name: multiline
      commands: |
        echo "line1"
        echo "line2"
rules: {}`,
		},
		{
			name: "special_chars_in_command",
			yaml: `system_tests:
  run_mode: after_all
  tests:
    - name: special
      commands: 'echo $HOME && whoami | head -1'
rules: {}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfigData([]byte(tc.yaml))
			// Should not panic during parsing
			if err != nil {
				t.Logf("Parse error for %s: %v", tc.name, err)
			} else if cfg != nil && cfg.SystemTests != nil {
				t.Logf("System tests loaded for %s: %d tests", tc.name, len(cfg.SystemTests.Tests))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// INCREMENTAL CONFIGURATION CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_Incremental_EdgeCases tests edge cases in incremental config.
func TestChaos_Incremental_EdgeCases(t *testing.T) {
	testCases := []struct {
		name string
		yaml string
	}{
		{
			name: "empty_incremental",
			yaml: `incremental: []
rules: {}`,
		},
		{
			name: "many_incremental_steps",
			yaml: `incremental: ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10']
rules: {}`,
		},
		{
			name: "incremental_with_special_chars",
			yaml: `incremental: ['minor:@scope/pkg', 'patch:pkg-with-dash']
rules: {}`,
		},
		{
			name: "incremental_with_glob",
			yaml: `incremental: ['minor:*', 'patch:**/*']
rules: {}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfigData([]byte(tc.yaml))
			assert.NoError(t, err, "should parse incremental config")
			if cfg != nil {
				t.Logf("Incremental steps for %s: %d", tc.name, len(cfg.Incremental))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// VALIDATION CHAOS TESTS
// -----------------------------------------------------------------------------

// TestChaos_Validation_UnknownFields tests handling of unknown fields.
func TestChaos_Validation_UnknownFields(t *testing.T) {
	testCases := []struct {
		name            string
		yaml            string
		expectError     bool
		expectWarning   bool
	}{
		{
			name: "unknown_root_field",
			yaml: `unknown_field: value
rules: {}`,
			expectError: true,
		},
		{
			name: "unknown_rule_field",
			yaml: `rules:
  npm:
    include: ['*.json']
    unknown_nested: value`,
			expectError: true,
		},
		{
			name: "typo_extends",
			yaml: `extend: ['base.yml']
rules: {}`,
			expectError: true,
		},
		{
			name: "typo_include",
			yaml: `rules:
  npm:
    includ: ['*.json']`,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidateConfigFile([]byte(tc.yaml))

			if tc.expectError {
				assert.True(t, result.HasErrors(), "should have validation errors for: %s", tc.name)
				if result.HasErrors() {
					t.Logf("Validation errors for %s: %s", tc.name, result.ErrorMessages())
				}
			}
		})
	}
}

// TestChaos_Validation_TypeCoercion tests type coercion edge cases.
func TestChaos_Validation_TypeCoercion(t *testing.T) {
	testCases := []struct {
		name        string
		yaml        string
		expectError bool
	}{
		{
			name: "string_as_boolean",
			yaml: `rules:
  npm:
    include: ['*.json']
security:
  allow_path_traversal: 'true'`,
			expectError: true, // String not boolean
		},
		{
			name: "number_as_boolean",
			yaml: `rules:
  npm:
    include: ['*.json']
security:
  allow_path_traversal: 1`,
			expectError: true, // Number not boolean
		},
		{
			name: "boolean_yes",
			yaml: `rules:
  npm:
    include: ['*.json']
security:
  allow_path_traversal: yes`,
			expectError: false, // YAML 'yes' is boolean true
		},
		{
			name: "boolean_on",
			yaml: `rules:
  npm:
    include: ['*.json']
security:
  allow_path_traversal: on`,
			expectError: false, // YAML 'on' is boolean true
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := loadConfigData([]byte(tc.yaml))

			if tc.expectError {
				// May error on parse or on validation
				if err == nil && cfg != nil && cfg.Security != nil {
					t.Logf("Coerced allow_path_traversal for %s: %v", tc.name, cfg.Security.AllowPathTraversal)
				}
			} else {
				assert.NoError(t, err, "should not error for: %s", tc.name)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// HELPER FUNCTIONS
// -----------------------------------------------------------------------------
