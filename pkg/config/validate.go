package config

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/ajxudir/goupdate/pkg/verbose"
	"gopkg.in/yaml.v3"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field      string
	Message    string
	Expected   string // Expected type or schema hint
	ValidKeys  string // Valid keys for this context
	DocSection string // Documentation section reference
}

// Error returns the error message string.
//
// This implements the error interface for ValidationError.
//
// Returns:
//   - string: formatted error message with field name if available
func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// VerboseError returns a detailed error message with schema hints.
//
// This provides additional context including expected types, valid keys,
// and documentation references to help users fix the error.
//
// Returns:
//   - string: detailed error message with schema information and documentation links
func (e ValidationError) VerboseError() string {
	var sb strings.Builder
	if e.Field != "" {
		sb.WriteString(fmt.Sprintf("%s: %s", e.Field, e.Message))
	} else {
		sb.WriteString(e.Message)
	}
	if e.Expected != "" {
		sb.WriteString(fmt.Sprintf("\n    Expected: %s", e.Expected))
	}
	if e.ValidKeys != "" {
		sb.WriteString(fmt.Sprintf("\n    Valid keys: %s", e.ValidKeys))
	}
	if e.DocSection != "" {
		sb.WriteString(fmt.Sprintf("\n    📖 See: docs/configuration.md#%s", e.DocSection))
	}
	return sb.String()
}

// ValidationResult holds the results of configuration validation.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []string
}

// HasErrors returns true if there are any validation errors.
//
// Returns:
//   - bool: true if validation found errors, false otherwise
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ErrorMessages returns all error messages as a formatted string.
//
// This formats all validation errors into a single multi-line string
// suitable for displaying to users.
//
// Returns:
//   - string: formatted error messages, or empty string if no errors
func (r *ValidationResult) ErrorMessages() string {
	if len(r.Errors) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, "  - "+e.Error())
	}
	return "Configuration validation failed:\n" + strings.Join(msgs, "\n")
}

// VerboseErrorMessages returns detailed error messages with schema hints.
//
// This is like ErrorMessages but includes additional context such as
// expected types, valid keys, and documentation references.
//
// Returns:
//   - string: detailed formatted error messages, or empty string if no errors
func (r *ValidationResult) VerboseErrorMessages() string {
	if len(r.Errors) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, "  - "+e.VerboseError())
	}
	return "Configuration validation failed:\n" + strings.Join(msgs, "\n")
}

// Schema information for validation errors
var configSchema = map[string]schemaInfo{
	"Config": {
		fields: "extends, working_dir, rules, exclude_versions, groups, incremental, system_tests",
		doc:    "configuration",
	},
	"PackageManagerCfg": {
		fields: "enabled, manager, include, exclude, groups, format, fields, ignore, exclude_versions, constraint_mapping, latest_mapping, package_overrides, extraction, outdated, update, lock_files, self_pinning, metadata, incremental",
		doc:    "rules",
	},
	"OutdatedCfg": {
		fields: "commands, env, format, extraction, versioning, exclude_versions, exclude_version_patterns, timeout_seconds",
		doc:    "outdated",
	},
	"UpdateCfg": {
		fields: "commands, env, group, timeout_seconds",
		doc:    "update",
	},
	"LockFileCfg": {
		fields: "files, format, extraction, commands, env, timeout_seconds, command_extraction",
		doc:    "lock-files",
	},
	"ExtractionCfg": {
		fields: "pattern, path, name_attr, version_attr, name_element, version_element, dev_attr, dev_value, dev_element, dev_element_value",
		doc:    "extraction",
	},
	"OutdatedExtractionCfg": {
		fields: "pattern, json_key, yaml_key",
		doc:    "outdated",
	},
	"PackageOverrideCfg": {
		fields: "ignore, constraint, version, outdated, update",
		doc:    "package-overrides",
	},
	"VersioningCfg": {
		fields: "format, regex, sort",
		doc:    "versioning",
	},
	"LatestMappingCfg": {
		fields: "default, packages",
		doc:    "latest-mapping",
	},
	"GroupCfg": {
		fields: "packages (list of package names)",
		doc:    "groups",
	},
	"SystemTestsCfg": {
		fields: "tests, run_preflight, run_mode, stop_on_fail",
		doc:    "system-tests",
	},
	"SystemTestCfg": {
		fields: "name, commands, env, timeout_seconds, continue_on_fail",
		doc:    "system-tests",
	},
}

type schemaInfo struct {
	fields string
	doc    string
}

// ValidateConfigFile validates a YAML configuration file for syntax errors and unknown fields.
//
// This performs strict validation using KnownFields(true) to detect typos and
// unknown configuration options. It also validates required fields and constraints.
//
// Parameters:
//   - data: YAML configuration data as bytes
//
// Returns:
//   - *ValidationResult: validation result with any errors and warnings found
func ValidateConfigFile(data []byte) *ValidationResult {
	result := &ValidationResult{}

	verbose.Printf("Config validation: starting YAML parsing with strict field checking\n")

	// First, check for unknown fields using strict YAML parsing
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		verbose.Printf("Config validation FAILED: YAML decode error: %v\n", err)
		// Parse the error to provide better messages
		errMsg := err.Error()
		if strings.Contains(errMsg, "field") && strings.Contains(errMsg, "not found") {
			// Extract field name and type from error like "field foo not found in type config.Config"
			fieldName, typeName := extractFieldAndType(errMsg)
			lineNum := extractLineNumber(errMsg)

			verr := ValidationError{
				Message: fmt.Sprintf("unknown field '%s'", fieldName),
			}
			if lineNum > 0 {
				verr.Message = fmt.Sprintf("unknown field '%s' (line %d)", fieldName, lineNum)
			}

			// Add schema hints
			if schema, ok := configSchema[typeName]; ok {
				verr.ValidKeys = schema.fields
				verr.DocSection = schema.doc
			} else if typeName != "" {
				verr.Expected = fmt.Sprintf("valid field for %s", typeName)
			}

			// Suggest similar field if typo detected
			if suggestion := suggestSimilarField(fieldName, typeName); suggestion != "" {
				verr.Message += fmt.Sprintf(" (did you mean '%s'?)", suggestion)
			}

			result.Errors = append(result.Errors, verr)
		} else if strings.Contains(errMsg, "cannot unmarshal") {
			// Type mismatch errors - check before "yaml:" since these also contain "yaml:"
			result.Errors = append(result.Errors, ValidationError{
				Message:  errMsg,
				Expected: extractExpectedType(errMsg),
			})
		} else if strings.Contains(errMsg, "yaml:") {
			result.Errors = append(result.Errors, ValidationError{
				Message:    fmt.Sprintf("YAML syntax error: %s", errMsg),
				DocSection: "configuration",
			})
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Message: errMsg,
			})
		}
		return result
	}

	// Validate required fields and constraints
	verbose.Printf("Config validation: YAML parsed successfully, validating structure\n")
	validateConfigStruct(&cfg, result)

	if len(result.Errors) == 0 {
		verbose.Printf("Config validation PASSED: no errors found\n")
	} else {
		verbose.Printf("Config validation FAILED: %d errors found\n", len(result.Errors))
	}
	if len(result.Warnings) > 0 {
		verbose.Printf("Config validation: %d warnings\n", len(result.Warnings))
	}

	return result
}

// Validate validates a loaded Config struct.
//
// This validates the configuration structure for required fields,
// valid values, and logical consistency.
//
// Returns:
//   - *ValidationResult: validation result with any errors and warnings found
func (c *Config) Validate() *ValidationResult {
	result := &ValidationResult{}
	validateConfigStruct(c, result)
	return result
}

// validateConfigStruct validates the Config structure.
//
// This checks rules, groups, incremental packages, and system tests
// for validity and consistency.
//
// Parameters:
//   - cfg: the configuration to validate
//   - result: validation result to append errors and warnings to
func validateConfigStruct(cfg *Config, result *ValidationResult) {
	// Validate rules
	verbose.Printf("Config validation: checking %d rules\n", len(cfg.Rules))
	for ruleName, rule := range cfg.Rules {
		verbose.Printf("Config validation: validating rule %q\n", ruleName)
		validateRule(ruleName, &rule, result)
	}

	// Validate groups reference valid rules
	verbose.Printf("Config validation: checking %d top-level groups\n", len(cfg.Groups))
	for groupName := range cfg.Groups {
		if groupName == "" {
			verbose.Printf("Config validation ERROR: empty group name detected\n")
			result.Errors = append(result.Errors, ValidationError{
				Field:   "groups",
				Message: "group name cannot be empty",
			})
		}
	}

	// Validate incremental packages are not empty strings
	if len(cfg.Incremental) > 0 {
		verbose.Printf("Config validation: checking %d incremental packages\n", len(cfg.Incremental))
	}
	for i, pkg := range cfg.Incremental {
		if pkg == "" {
			verbose.Printf("Config validation ERROR: empty incremental package at index %d\n", i)
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("incremental[%d]", i),
				Message: "incremental package name cannot be empty",
			})
		}
	}

	// Validate system_tests configuration
	if cfg.SystemTests != nil {
		verbose.Printf("Config validation: checking system_tests configuration\n")
		validateSystemTests(cfg.SystemTests, result)
	}
}

// validateSystemTests validates system tests configuration.
//
// This checks that test names and commands are specified, run_mode is valid,
// and timeout values are positive.
//
// Parameters:
//   - st: the system tests configuration to validate
//   - result: validation result to append errors and warnings to
func validateSystemTests(st *SystemTestsCfg, result *ValidationResult) {
	// Validate run_mode if specified
	if st.RunMode != "" {
		verbose.Printf("Config validation: checking system_tests.run_mode=%q\n", st.RunMode)
		validModes := []string{SystemTestRunModeAfterEach, SystemTestRunModeAfterAll, SystemTestRunModeNone}
		isValid := false
		for _, m := range validModes {
			if st.RunMode == m {
				isValid = true
				break
			}
		}
		if !isValid {
			verbose.Printf("Config validation ERROR: invalid run_mode %q (valid: %v)\n", st.RunMode, validModes)
			result.Errors = append(result.Errors, ValidationError{
				Field:      "system_tests.run_mode",
				Message:    fmt.Sprintf("invalid run_mode '%s'", st.RunMode),
				Expected:   "after_each, after_all, or none",
				DocSection: "system-tests",
			})
		} else {
			verbose.Printf("Config validation: run_mode %q is valid\n", st.RunMode)
		}
	}

	// Validate tests
	if len(st.Tests) == 0 && (st.IsRunPreflight() || st.GetRunMode() != SystemTestRunModeNone) {
		verbose.Printf("Config validation WARNING: no tests defined but system tests are enabled\n")
		result.Warnings = append(result.Warnings, "system_tests: no tests defined but system tests are enabled")
	}

	verbose.Printf("Config validation: checking %d system tests\n", len(st.Tests))
	for i, test := range st.Tests {
		prefix := fmt.Sprintf("system_tests.tests[%d]", i)

		// Name is required
		if test.Name == "" {
			verbose.Printf("Config validation ERROR: %s.name is empty\n", prefix)
			result.Errors = append(result.Errors, ValidationError{
				Field:      prefix + ".name",
				Message:    "test name is required",
				DocSection: "system-tests",
			})
		} else {
			verbose.Printf("Config validation: validating test %q\n", test.Name)
		}

		// Commands is required
		if strings.TrimSpace(test.Commands) == "" {
			verbose.Printf("Config validation ERROR: %s.commands is empty\n", prefix)
			result.Errors = append(result.Errors, ValidationError{
				Field:      prefix + ".commands",
				Message:    "test commands cannot be empty",
				DocSection: "system-tests",
			})
		}

		// Timeout should be positive if specified
		if test.TimeoutSeconds < 0 {
			verbose.Printf("Config validation ERROR: %s.timeout_seconds=%d is negative\n", prefix, test.TimeoutSeconds)
			result.Errors = append(result.Errors, ValidationError{
				Field:    prefix + ".timeout_seconds",
				Message:  "timeout must be positive",
				Expected: "positive integer (seconds)",
			})
		}
	}
}

// validateRule validates a package manager rule configuration.
//
// This checks include patterns, groups, lock files, outdated config,
// and package overrides for validity.
//
// Parameters:
//   - name: the rule name
//   - rule: the rule configuration to validate
//   - result: validation result to append errors and warnings to
func validateRule(name string, rule *PackageManagerCfg, result *ValidationResult) {
	prefix := fmt.Sprintf("rules.%s", name)

	// manager is required for custom rules (but may be inherited from defaults)
	// We can't strictly require it here because it may come from extends

	// format is required if no lock_files defined
	// Similar to manager, may be inherited

	// Validate include patterns if specified
	if len(rule.Include) > 0 {
		verbose.Printf("Config validation: rule %q has %d include patterns\n", name, len(rule.Include))
	}
	for i, pattern := range rule.Include {
		if pattern == "" {
			verbose.Printf("Config validation ERROR: rule %q include[%d] is empty\n", name, i)
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.include[%d]", prefix, i),
				Message: "include pattern cannot be empty",
			})
		}
	}

	// Validate groups
	if len(rule.Groups) > 0 {
		verbose.Printf("Config validation: rule %q has %d groups\n", name, len(rule.Groups))
	}
	for groupName, group := range rule.Groups {
		if groupName == "" {
			verbose.Printf("Config validation ERROR: rule %q has empty group name\n", name)
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("%s.groups", prefix),
				Message: "group name cannot be empty",
			})
		}
		if len(group.Packages) == 0 {
			verbose.Printf("Config validation WARNING: rule %q group %q has no packages\n", name, groupName)
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s.groups.%s: group has no packages", prefix, groupName))
		} else {
			verbose.Printf("Config validation: rule %q group %q has %d packages\n", name, groupName, len(group.Packages))
		}
	}

	// Validate lock files
	if len(rule.LockFiles) > 0 {
		verbose.Printf("Config validation: rule %q has %d lock file configs\n", name, len(rule.LockFiles))
	}
	for i, lf := range rule.LockFiles {
		lfPrefix := fmt.Sprintf("%s.lock_files[%d]", prefix, i)
		if len(lf.Files) == 0 {
			verbose.Printf("Config validation ERROR: %s.files is empty\n", lfPrefix)
			result.Errors = append(result.Errors, ValidationError{
				Field:   lfPrefix + ".files",
				Message: "lock file must specify at least one file pattern",
			})
		}
		if lf.Format == "" && lf.Extraction == nil {
			verbose.Printf("Config validation ERROR: %s has no format or extraction\n", lfPrefix)
			result.Errors = append(result.Errors, ValidationError{
				Field:   lfPrefix,
				Message: "lock file must specify format or extraction",
			})
		}
	}

	// Validate outdated config
	if rule.Outdated != nil {
		verbose.Printf("Config validation: rule %q has outdated config\n", name)
		validateOutdated(prefix+".outdated", rule.Outdated, result)
	}

	// Validate package overrides
	if len(rule.PackageOverrides) > 0 {
		verbose.Printf("Config validation: rule %q has %d package overrides\n", name, len(rule.PackageOverrides))
	}
	for pkgName, override := range rule.PackageOverrides {
		if pkgName == "" {
			verbose.Printf("Config validation ERROR: rule %q has empty package_overrides key\n", name)
			result.Errors = append(result.Errors, ValidationError{
				Field:   prefix + ".package_overrides",
				Message: "package override key cannot be empty",
			})
		}
		validatePackageOverride(fmt.Sprintf("%s.package_overrides.%s", prefix, pkgName), &override, result)
	}
}

// validateOutdated validates outdated configuration.
//
// This checks that commands contain required placeholders and warns
// if the {{package}} placeholder is missing.
//
// Parameters:
//   - prefix: field path prefix for error messages
//   - outdated: the outdated configuration to validate
//   - result: validation result to append errors and warnings to
func validateOutdated(prefix string, outdated *OutdatedCfg, result *ValidationResult) {
	// Commands should be non-empty if specified
	if outdated.Commands != "" {
		// Check for required placeholders
		if !strings.Contains(outdated.Commands, "{{package}}") {
			verbose.Printf("Config validation WARNING: %s.commands missing {{package}} placeholder\n", prefix)
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s.commands: missing {{package}} placeholder", prefix))
		} else {
			verbose.Printf("Config validation: %s.commands has required placeholders\n", prefix)
		}
	}
}

// validatePackageOverride validates package override configuration.
//
// This warns if a constraint is specified but empty.
//
// Parameters:
//   - prefix: field path prefix for error messages
//   - override: the package override configuration to validate
//   - result: validation result to append errors and warnings to
func validatePackageOverride(prefix string, override *PackageOverrideCfg, result *ValidationResult) {
	// If constraint is specified, it should be non-empty
	if override.Constraint != nil && *override.Constraint == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s.constraint: empty constraint specified", prefix))
	}
}

// extractFieldAndType extracts the unknown field name and the type it was found in.
//
// This parses YAML error messages to extract the field name and type information
// for better error reporting.
//
// Parameters:
//   - errMsg: YAML error message
//
// Returns:
//   - field: the unknown field name
//   - typeName: the type name where the field was found
func extractFieldAndType(errMsg string) (field, typeName string) {
	// Error format: "yaml: unmarshal errors:\n  line X: field foo not found in type config.Type"
	parts := strings.Split(errMsg, "field ")
	if len(parts) >= 2 {
		fieldPart := parts[1]
		spaceIdx := strings.Index(fieldPart, " ")
		if spaceIdx > 0 {
			field = fieldPart[:spaceIdx]
		} else {
			field = fieldPart
		}
	}

	// Extract type name
	if idx := strings.Index(errMsg, "in type config."); idx >= 0 {
		typePart := errMsg[idx+len("in type config."):]
		if endIdx := strings.IndexAny(typePart, " \n"); endIdx > 0 {
			typeName = typePart[:endIdx]
		} else {
			typeName = typePart
		}
	}

	return field, typeName
}

// extractLineNumber extracts the line number from a YAML error message.
//
// This uses regex to find "line X:" patterns in YAML error messages.
//
// Parameters:
//   - errMsg: YAML error message
//
// Returns:
//   - int: the line number, or 0 if not found
func extractLineNumber(errMsg string) int {
	// Pattern: "line X:" in the error message
	re := regexp.MustCompile(`line (\d+):`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) >= 2 {
		var lineNum int
		_, _ = fmt.Sscanf(matches[1], "%d", &lineNum)
		return lineNum
	}
	return 0
}

// extractExpectedType extracts the expected type from unmarshal errors.
//
// This parses "cannot unmarshal X into Y" error messages to extract
// the expected type Y.
//
// Parameters:
//   - errMsg: YAML unmarshal error message
//
// Returns:
//   - string: the expected type name, or empty string if not found
func extractExpectedType(errMsg string) string {
	// Pattern: "cannot unmarshal !!X into Y"
	if idx := strings.Index(errMsg, "into "); idx >= 0 {
		typePart := errMsg[idx+5:]
		if endIdx := strings.IndexAny(typePart, " \n"); endIdx > 0 {
			return typePart[:endIdx]
		}
		return typePart
	}
	return ""
}

// commonTypos maps common typos to correct field names
var commonTypos = map[string]map[string]string{
	"Config": {
		"rule":                "rules",
		"extend":              "extends",
		"working-dir":         "working_dir",
		"workingDir":          "working_dir",
		"exclude_version":     "exclude_versions",
		"group":               "groups",
		"incremental_package": "incremental",
	},
	"PackageManagerCfg": {
		"enable":              "enabled",
		"includes":            "include",
		"excludes":            "exclude",
		"group":               "groups",
		"field":               "fields",
		"lock_file":           "lock_files",
		"lockFiles":           "lock_files",
		"lockFile":            "lock_files",
		"package_override":    "package_overrides",
		"packageOverrides":    "package_overrides",
		"exclude_version":     "exclude_versions",
		"constraint_map":      "constraint_mapping",
		"constraintMapping":   "constraint_mapping",
		"latest_map":          "latest_mapping",
		"latestMapping":       "latest_mapping",
		"self-pinning":        "self_pinning",
		"selfPinning":         "self_pinning",
		"incremental_package": "incremental",
	},
	"OutdatedCfg": {
		"command":                 "commands",
		"timeout":                 "timeout_seconds",
		"timeoutSeconds":          "timeout_seconds",
		"exclude_version":         "exclude_versions",
		"exclude_version_pattern": "exclude_version_patterns",
		"excludeVersionPatterns":  "exclude_version_patterns",
	},
	"UpdateCfg": {
		"lock_commands":  "commands",
		"lock_command":   "commands",
		"lockCommands":   "commands",
		"timeout":        "timeout_seconds",
		"timeoutSeconds": "timeout_seconds",
	},
	"LockFileCfg": {
		"file":               "files",
		"command":            "commands",
		"command_extraction": "command_extraction",
	},
	"ExtractionCfg": {
		"name-attr":       "name_attr",
		"nameAttr":        "name_attr",
		"version-attr":    "version_attr",
		"versionAttr":     "version_attr",
		"name-element":    "name_element",
		"nameElement":     "name_element",
		"version-element": "version_element",
		"versionElement":  "version_element",
	},
	"OutdatedExtractionCfg": {
		"json-key": "json_key",
		"jsonKey":  "json_key",
		"yaml-key": "yaml_key",
		"yamlKey":  "yaml_key",
	},
	"PackageOverrideCfg": {
		"ignored": "ignore",
	},
	"SystemTestsCfg": {
		"test":          "tests",
		"runPreflight":  "run_preflight",
		"run-preflight": "run_preflight",
		"runMode":       "run_mode",
		"run-mode":      "run_mode",
		"stopOnFail":    "stop_on_fail",
		"stop-on-fail":  "stop_on_fail",
	},
	"SystemTestCfg": {
		"command":          "commands",
		"timeout":          "timeout_seconds",
		"timeoutSeconds":   "timeout_seconds",
		"timeout-seconds":  "timeout_seconds",
		"continueOnFail":   "continue_on_fail",
		"continue-on-fail": "continue_on_fail",
	},
}

// suggestSimilarField returns a suggested field name if the input looks like a typo.
//
// This checks common typos and naming convention differences (kebab-case vs snake_case)
// to suggest corrections for unknown fields.
//
// Parameters:
//   - field: the unknown field name
//   - typeName: the type name where the field was found
//
// Returns:
//   - string: suggested correct field name, or empty string if no suggestion
func suggestSimilarField(field, typeName string) string {
	// Check common typos for this type
	if typos, ok := commonTypos[typeName]; ok {
		if suggestion, found := typos[field]; found {
			return suggestion
		}
	}

	// Check if camelCase vs snake_case
	if strings.Contains(field, "-") {
		// Try converting kebab-case to snake_case
		snakeCase := strings.ReplaceAll(field, "-", "_")
		if schema, ok := configSchema[typeName]; ok {
			if strings.Contains(schema.fields, snakeCase) {
				return snakeCase
			}
		}
	}

	return ""
}

// extractUnknownField extracts just the field name (for backwards compatibility).
//
// This is a convenience wrapper around extractFieldAndType that only returns
// the field name. Kept for backwards compatibility.
//
// Parameters:
//   - errMsg: YAML error message
//
// Returns:
//   - string: the unknown field name
func extractUnknownField(errMsg string) string {
	field, _ := extractFieldAndType(errMsg)
	return field
}

// ValidateConfigFileStrict is like ValidateConfigFile but treats warnings as errors.
//
// This provides the strictest validation mode where even warnings will cause
// validation to fail.
//
// Parameters:
//   - data: YAML configuration data as bytes
//
// Returns:
//   - *ValidationResult: validation result with warnings converted to errors
func ValidateConfigFileStrict(data []byte) *ValidationResult {
	result := ValidateConfigFile(data)
	// Convert warnings to errors
	for _, w := range result.Warnings {
		result.Errors = append(result.Errors, ValidationError{Message: w})
	}
	result.Warnings = nil
	return result
}
