package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationCategory identifies the source of a validation error.
//
// This type distinguishes between different validation contexts to enable
// appropriate formatting and handling of validation failures.
type ValidationCategory string

const (
	// ValidationCategoryConfig indicates a configuration file validation error.
	ValidationCategoryConfig ValidationCategory = "config"

	// ValidationCategoryPreflight indicates a preflight check failure (missing command).
	ValidationCategoryPreflight ValidationCategory = "preflight"

	// ValidationCategoryPackage indicates a package-level validation error.
	ValidationCategoryPackage ValidationCategory = "package"
)

// ValidationError represents a configuration or preflight validation failure.
//
// This unified type replaces the separate ValidationError types that existed
// in pkg/config and pkg/preflight. The Category field distinguishes the source.
//
// Fields:
//   - Category: Source of validation ("config", "preflight", "package")
//   - Field: Name of the invalid field or setting
//   - Message: Description of what's wrong
//   - Expected: What the valid value should look like
//   - ValidKeys: List of valid options (for enum-like fields)
//   - DocSection: Link to documentation for this setting
//   - Command: For preflight errors, the command that failed
//   - Hint: Actionable hint for fixing the error
//
// Example:
//
//	return &ValidationError{
//	    Category:   ValidationCategoryConfig,
//	    Field:      "rules.npm.format",
//	    Message:    "invalid format type",
//	    Expected:   "one of: json, yaml, xml, raw",
//	    ValidKeys:  []string{"json", "yaml", "xml", "raw"},
//	    DocSection: "docs/user/configuration.md#format",
//	}
type ValidationError struct {
	// Category identifies the validation source.
	// Values: "config", "preflight", "package"
	Category ValidationCategory

	// Field is the name of the field that failed validation.
	Field string

	// Message describes what is wrong with the field.
	Message string

	// Expected describes what a valid value should look like.
	Expected string

	// ValidKeys lists valid options for enum-like fields.
	ValidKeys []string

	// DocSection links to documentation for this field.
	DocSection string

	// Command is the system command that failed (preflight only).
	Command string

	// Hint provides an actionable suggestion for fixing the error.
	Hint string
}

// Error implements the error interface.
//
// Formats the error message based on the Category. For preflight errors,
// includes command and resolution. For config errors, includes field and message.
//
// Returns:
//   - string: Formatted error message appropriate for the validation category
func (e *ValidationError) Error() string {
	var sb strings.Builder

	switch e.Category {
	case ValidationCategoryPreflight:
		if e.Command != "" {
			sb.WriteString(fmt.Sprintf("command not found: %s", e.Command))
			if e.Hint != "" {
				sb.WriteString(fmt.Sprintf("\n  Resolution: %s", e.Hint))
			} else {
				sb.WriteString(fmt.Sprintf("\n  Resolution: Ensure '%s' is installed and available in your PATH.", e.Command))
			}
			return sb.String()
		}
	case ValidationCategoryConfig:
		if e.Field != "" {
			sb.WriteString(fmt.Sprintf("%s: %s", e.Field, e.Message))
		} else {
			sb.WriteString(e.Message)
		}
		return sb.String()
	}

	// Default format
	if e.Field != "" {
		sb.WriteString(fmt.Sprintf("%s: %s", e.Field, e.Message))
	} else if e.Message != "" {
		sb.WriteString(e.Message)
	} else if e.Command != "" {
		sb.WriteString(fmt.Sprintf("command not found: %s", e.Command))
	}

	return sb.String()
}

// VerboseError returns a detailed error message with schema hints.
//
// Returns:
//   - string: Detailed error with expected values and documentation links
func (e *ValidationError) VerboseError() string {
	var sb strings.Builder

	// Base error
	sb.WriteString(e.Error())

	// Add expected value hint
	if e.Expected != "" {
		sb.WriteString(fmt.Sprintf("\n    Expected: %s", e.Expected))
	}

	// Add valid keys hint
	if len(e.ValidKeys) > 0 {
		sb.WriteString(fmt.Sprintf("\n    Valid keys: %s", strings.Join(e.ValidKeys, ", ")))
	}

	// Add documentation link
	if e.DocSection != "" {
		sb.WriteString(fmt.Sprintf("\n    See: docs/configuration.md#%s", e.DocSection))
	}

	// Add resolution hint
	if e.Hint != "" && e.Category != ValidationCategoryPreflight {
		sb.WriteString(fmt.Sprintf("\n    Hint: %s", e.Hint))
	}

	return sb.String()
}

// IsValidationError checks if err is a ValidationError and returns it.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - *ValidationError: The ValidationError if err is one, nil otherwise
//   - bool: true if err is a ValidationError
func IsValidationError(err error) (*ValidationError, bool) {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}

// NewConfigValidationError creates a ValidationError for configuration issues.
//
// Parameters:
//   - field: The field name that failed validation
//   - message: Description of the error
//
// Returns:
//   - *ValidationError: New validation error with config category
//
// Example:
//
//	err := errors.NewConfigValidationError("rules.npm.format", "invalid format type")
func NewConfigValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Category: ValidationCategoryConfig,
		Field:    field,
		Message:  message,
	}
}

// NewPreflightValidationError creates a ValidationError for preflight check failures.
//
// Parameters:
//   - command: The command that was not found
//   - hint: Resolution hint for installing the command
//
// Returns:
//   - *ValidationError: New validation error with preflight category
//
// Example:
//
//	err := errors.NewPreflightValidationError("npm", "Install Node.js: https://nodejs.org/")
func NewPreflightValidationError(command, hint string) *ValidationError {
	return &ValidationError{
		Category: ValidationCategoryPreflight,
		Command:  command,
		Hint:     hint,
	}
}

// NewPackageValidationError creates a ValidationError for package-level issues.
//
// Parameters:
//   - pkg: The package name that failed validation
//   - message: Description of the error
//   - hint: Resolution hint
//
// Returns:
//   - *ValidationError: New validation error with package category
//
// Example:
//
//	err := errors.NewPackageValidationError("lodash", "version constraint invalid", "Use semantic versioning")
func NewPackageValidationError(pkg, message, hint string) *ValidationError {
	return &ValidationError{
		Category: ValidationCategoryPackage,
		Field:    pkg,
		Message:  message,
		Hint:     hint,
	}
}
