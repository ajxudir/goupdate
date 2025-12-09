package errors

import (
	"bytes"
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExitCodes tests the exit code constants.
//
// It verifies that:
//   - ExitSuccess equals 0
//   - ExitPartialFailure equals 1
//   - ExitFailure equals 2
//   - ExitConfigError equals 3
func TestExitCodes(t *testing.T) {
	assert.Equal(t, 0, ExitSuccess)
	assert.Equal(t, 1, ExitPartialFailure)
	assert.Equal(t, 2, ExitFailure)
	assert.Equal(t, 3, ExitConfigError)
}

// TestExitError tests the ExitError struct and its methods.
//
// It verifies that:
//   - Error() returns the Message field when set
//   - Error() returns wrapped error message when Err is set
//   - Error() returns "exit code N" when neither is set
//   - Unwrap() returns the wrapped error
func TestExitError(t *testing.T) {
	t.Run("with message", func(t *testing.T) {
		err := &ExitError{Code: ExitFailure, Message: "test message"}
		assert.Equal(t, "test message", err.Error())
		assert.Equal(t, ExitFailure, err.Code)
	})

	t.Run("with wrapped error", func(t *testing.T) {
		innerErr := stderrors.New("inner error")
		err := &ExitError{Code: ExitConfigError, Err: innerErr}
		assert.Equal(t, "inner error", err.Error())
		assert.Equal(t, ExitConfigError, err.Code)
		assert.Equal(t, innerErr, err.Unwrap())
	})

	t.Run("with neither", func(t *testing.T) {
		err := &ExitError{Code: ExitPartialFailure}
		assert.Contains(t, err.Error(), "exit code 1")
	})
}

// TestNewExitError tests the NewExitError constructor.
//
// Parameters:
//   - code: Exit code value
//   - err: Error to wrap
//
// It verifies that:
//   - Code and Err fields are set correctly
func TestNewExitError(t *testing.T) {
	innerErr := stderrors.New("test error")
	err := NewExitError(ExitConfigError, innerErr)

	assert.Equal(t, ExitConfigError, err.Code)
	assert.Equal(t, innerErr, err.Err)
}

// TestNewExitErrorf tests the NewExitErrorf constructor.
//
// Parameters:
//   - code: Exit code value
//   - format: Printf-style format string
//   - args: Format arguments
//
// It verifies that:
//   - Code is set correctly
//   - Message is formatted properly
func TestNewExitErrorf(t *testing.T) {
	err := NewExitErrorf(ExitFailure, "failed: %s", "reason")

	assert.Equal(t, ExitFailure, err.Code)
	assert.Equal(t, "failed: reason", err.Message)
}

// TestGetExitCode tests the GetExitCode function.
//
// Parameters:
//   - err: Error to extract exit code from
//
// It verifies that:
//   - Nil error returns ExitSuccess
//   - ExitError returns its Code
//   - Wrapped ExitError returns its Code
//   - Plain error returns ExitFailure
func TestGetExitCode(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.Equal(t, ExitSuccess, GetExitCode(nil))
	})

	t.Run("ExitError", func(t *testing.T) {
		err := NewExitError(ExitConfigError, stderrors.New("test"))
		assert.Equal(t, ExitConfigError, GetExitCode(err))
	})

	t.Run("wrapped ExitError", func(t *testing.T) {
		inner := NewExitError(ExitPartialFailure, stderrors.New("test"))
		wrapped := stderrors.Join(stderrors.New("wrapper"), inner)
		assert.Equal(t, ExitPartialFailure, GetExitCode(wrapped))
	})

	t.Run("plain error", func(t *testing.T) {
		err := stderrors.New("plain error")
		assert.Equal(t, ExitFailure, GetExitCode(err))
	})
}

// TestEnhanceErrorWithHint tests the EnhanceErrorWithHint function.
//
// Parameters:
//   - err: Error to enhance with contextual hints
//
// It verifies that:
//   - Nil error returns empty string
//   - Matching patterns return error message with hint
//   - Non-matching patterns return error message only
//   - Various error patterns (JSON, network, permission, etc.) are handled
func TestEnhanceErrorWithHint(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		assert.Equal(t, "", EnhanceErrorWithHint(nil))
	})

	t.Run("matching pattern", func(t *testing.T) {
		err := stderrors.New("failed to parse JSON file")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "failed to parse")
		assert.Contains(t, result, "💡")
		assert.Contains(t, result, "Check file syntax")
	})

	t.Run("lock install drifted", func(t *testing.T) {
		err := stderrors.New("lock install drifted: expected 1.0.0, found 1.0.1")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "lock install drifted")
		assert.Contains(t, result, "npm install")
	})

	t.Run("command timeout", func(t *testing.T) {
		err := stderrors.New("command timed out after 30 seconds")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "command timed out")
		assert.Contains(t, result, "--no-timeout")
	})

	t.Run("no matching pattern", func(t *testing.T) {
		err := stderrors.New("some random error")
		result := EnhanceErrorWithHint(err)
		assert.Equal(t, "some random error", result)
		assert.NotContains(t, result, "💡")
	})

	t.Run("network error", func(t *testing.T) {
		err := stderrors.New("network connection failed")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "network")
		assert.Contains(t, result, "internet connection")
	})

	t.Run("permission denied", func(t *testing.T) {
		err := stderrors.New("open file: permission denied")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "permission denied")
		assert.Contains(t, result, "permissions")
	})

	t.Run("404 error", func(t *testing.T) {
		err := stderrors.New("HTTP 404: package not found")
		result := EnhanceErrorWithHint(err)
		assert.Contains(t, result, "404")
		assert.Contains(t, result, "package name")
	})
}

// TestFormatErrorsWithHints tests the FormatErrorsWithHints function.
//
// Parameters:
//   - errs: Slice of errors to format
//
// It verifies that:
//   - Empty slice returns empty string
//   - Multiple errors are formatted with error icons
func TestFormatErrorsWithHints(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		result := FormatErrorsWithHints(nil)
		assert.Equal(t, "", result)
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := []error{
			stderrors.New("failed to parse JSON"),
			stderrors.New("network error"),
		}
		result := FormatErrorsWithHints(errs)
		assert.Contains(t, result, "❌")
		assert.Contains(t, result, "failed to parse")
		assert.Contains(t, result, "network")
	})
}

// TestPartialSuccessError tests the PartialSuccessError struct and NewPartialSuccessError constructor.
//
// Parameters:
//   - succeeded: Number of successful operations
//   - failed: Number of failed operations
//   - errs: Slice of individual errors
//
// It verifies that:
//   - Fields are set correctly
//   - Error() returns formatted summary
func TestPartialSuccessError(t *testing.T) {
	errs := []error{
		stderrors.New("error 1"),
		stderrors.New("error 2"),
	}
	err := NewPartialSuccessError(5, 2, errs)

	assert.Equal(t, 5, err.Succeeded)
	assert.Equal(t, 2, err.Failed)
	assert.Equal(t, errs, err.Errors)
	assert.Equal(t, "5 succeeded, 2 failed", err.Error())
}

// TestIsExitError tests the IsExitError type assertion helper.
//
// Parameters:
//   - err: Error to check
//
// It verifies that:
//   - Nil error returns false, nil
//   - ExitError returns true with the error
//   - Wrapped ExitError returns true with the error
//   - Non-ExitError returns false, nil
func TestIsExitError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		exitErr, ok := IsExitError(nil)
		assert.False(t, ok)
		assert.Nil(t, exitErr)
	})

	t.Run("ExitError", func(t *testing.T) {
		err := NewExitError(ExitConfigError, stderrors.New("config error"))
		exitErr, ok := IsExitError(err)
		assert.True(t, ok)
		assert.NotNil(t, exitErr)
		assert.Equal(t, ExitConfigError, exitErr.Code)
	})

	t.Run("wrapped ExitError", func(t *testing.T) {
		inner := NewExitError(ExitPartialFailure, stderrors.New("test"))
		wrapped := stderrors.Join(stderrors.New("wrapper"), inner)
		exitErr, ok := IsExitError(wrapped)
		assert.True(t, ok)
		assert.NotNil(t, exitErr)
		assert.Equal(t, ExitPartialFailure, exitErr.Code)
	})

	t.Run("non-ExitError", func(t *testing.T) {
		err := stderrors.New("plain error")
		exitErr, ok := IsExitError(err)
		assert.False(t, ok)
		assert.Nil(t, exitErr)
	})
}

// TestIsPartialSuccess tests the IsPartialSuccess type assertion helper.
//
// Parameters:
//   - err: Error to check
//
// It verifies that:
//   - Nil error returns false, nil
//   - PartialSuccessError returns true with the error
//   - Non-PartialSuccessError returns false, nil
func TestIsPartialSuccess(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		pse, ok := IsPartialSuccess(nil)
		assert.False(t, ok)
		assert.Nil(t, pse)
	})

	t.Run("PartialSuccessError", func(t *testing.T) {
		err := NewPartialSuccessError(3, 1, []error{stderrors.New("failed")})
		pse, ok := IsPartialSuccess(err)
		assert.True(t, ok)
		assert.NotNil(t, pse)
		assert.Equal(t, 3, pse.Succeeded)
	})

	t.Run("non-PartialSuccessError", func(t *testing.T) {
		err := stderrors.New("plain error")
		pse, ok := IsPartialSuccess(err)
		assert.False(t, ok)
		assert.Nil(t, pse)
	})
}

// TestUnsupportedError tests the UnsupportedError struct and related functions.
//
// It verifies that:
//   - NewUnsupportedError creates error with correct fields
//   - Error() includes package name when set
//   - Error() works without package name
//   - IsUnsupportedError identifies the error type
//   - IsUnsupported returns boolean for error type check
func TestUnsupportedError(t *testing.T) {
	t.Run("NewUnsupportedError", func(t *testing.T) {
		err := NewUnsupportedError("update", "no lock file found", "lodash")
		assert.NotNil(t, err)
		assert.Equal(t, "lodash", err.Package)
		assert.Equal(t, "update", err.Operation)
		assert.Equal(t, "no lock file found", err.Reason)
	})

	t.Run("Error message with package", func(t *testing.T) {
		err := &UnsupportedError{
			Package:   "react",
			Operation: "outdated",
			Reason:    "no lock file",
		}
		assert.Contains(t, err.Error(), "react")
		assert.Contains(t, err.Error(), "outdated")
	})

	t.Run("Error message without package", func(t *testing.T) {
		err := &UnsupportedError{
			Operation: "update",
			Reason:    "not supported",
		}
		msg := err.Error()
		assert.Contains(t, msg, "update")
		assert.Contains(t, msg, "not supported")
	})

	t.Run("Error message with only reason", func(t *testing.T) {
		// Covers the branch where Package="" and Operation=""
		err := &UnsupportedError{
			Reason: "only reason provided",
		}
		msg := err.Error()
		assert.Equal(t, "only reason provided", msg)
	})

	t.Run("IsUnsupportedError", func(t *testing.T) {
		err := NewUnsupportedError("pkg", "op", "reason")
		ue, ok := IsUnsupportedError(err)
		assert.True(t, ok)
		assert.NotNil(t, ue)
	})

	t.Run("IsUnsupported with nil", func(t *testing.T) {
		assert.False(t, IsUnsupported(nil))
	})

	t.Run("IsUnsupported with UnsupportedError", func(t *testing.T) {
		err := NewUnsupportedError("pkg", "op", "reason")
		assert.True(t, IsUnsupported(err))
	})

	t.Run("IsUnsupported with other error", func(t *testing.T) {
		err := stderrors.New("plain error")
		assert.False(t, IsUnsupported(err))
	})
}

// TestValidationError tests the ValidationError struct and related functions.
//
// It verifies that:
//   - Config validation errors include field and message
//   - Preflight validation errors include command
//   - Package validation errors include package name and hint
//   - VerboseError includes Expected, ValidKeys, and DocSection
//   - IsValidationError identifies the error type
func TestValidationError(t *testing.T) {
	t.Run("config validation error", func(t *testing.T) {
		err := NewConfigValidationError("rules.npm.format", "invalid format")
		assert.NotNil(t, err)
		assert.Equal(t, ValidationCategoryConfig, err.Category)
		assert.Contains(t, err.Error(), "rules.npm.format")
		assert.Contains(t, err.Error(), "invalid format")
	})

	t.Run("preflight validation error", func(t *testing.T) {
		err := NewPreflightValidationError("npm", "npm is required for JavaScript packages")
		assert.NotNil(t, err)
		assert.Equal(t, ValidationCategoryPreflight, err.Category)
		assert.Contains(t, err.Error(), "npm")
	})

	t.Run("package validation error", func(t *testing.T) {
		err := NewPackageValidationError("lodash", "invalid version format", "use semver")
		assert.NotNil(t, err)
		assert.Equal(t, ValidationCategoryPackage, err.Category)
	})

	t.Run("VerboseError with expected", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "format",
			Message:  "invalid",
			Expected: "json or yaml",
		}
		verbose := err.VerboseError()
		assert.Contains(t, verbose, "Expected: json or yaml")
	})

	t.Run("VerboseError with ValidKeys", func(t *testing.T) {
		err := &ValidationError{
			Category:  ValidationCategoryConfig,
			Field:     "type",
			Message:   "invalid type",
			ValidKeys: []string{"prod", "dev", "peer"},
		}
		verbose := err.VerboseError()
		assert.Contains(t, verbose, "Valid keys:")
	})

	t.Run("VerboseError with DocSection", func(t *testing.T) {
		err := &ValidationError{
			Category:   ValidationCategoryConfig,
			Field:      "rules",
			Message:    "no rules defined",
			DocSection: "docs/configuration.md",
		}
		verbose := err.VerboseError()
		assert.Contains(t, verbose, "docs/configuration.md")
	})

	t.Run("IsValidationError", func(t *testing.T) {
		err := NewConfigValidationError("field", "message")
		ve, ok := IsValidationError(err)
		assert.True(t, ok)
		assert.NotNil(t, ve)
	})

	t.Run("IsValidationError with nil", func(t *testing.T) {
		ve, ok := IsValidationError(nil)
		assert.False(t, ok)
		assert.Nil(t, ve)
	})

	t.Run("IsValidationError with other error", func(t *testing.T) {
		err := stderrors.New("plain error")
		ve, ok := IsValidationError(err)
		assert.False(t, ok)
		assert.Nil(t, ve)
	})
}

// TestValidationResult tests the ValidationResult struct and related functions.
//
// It verifies that:
//   - NewValidationResult creates empty result
//   - AddError adds errors and HasErrors reflects state
//   - AddWarning adds warnings and HasWarnings reflects state
//   - ErrorMessage returns formatted error summary
//   - VerboseErrorMessage includes detailed information
func TestValidationResult(t *testing.T) {
	t.Run("NewValidationResult", func(t *testing.T) {
		result := NewValidationResult()
		assert.NotNil(t, result)
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("AddError and HasErrors", func(t *testing.T) {
		result := NewValidationResult()
		assert.False(t, result.HasErrors())

		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "test error",
		})
		assert.True(t, result.HasErrors())
		assert.Len(t, result.Errors, 1)
	})

	t.Run("AddWarning and HasWarnings", func(t *testing.T) {
		result := NewValidationResult()
		assert.False(t, result.HasWarnings())

		result.AddWarning("test warning")
		assert.True(t, result.HasWarnings())
		assert.Len(t, result.Warnings, 1)
	})

	t.Run("ErrorMessage with no errors returns empty", func(t *testing.T) {
		result := NewValidationResult()
		msg := result.ErrorMessage()
		assert.Empty(t, msg)
	})

	t.Run("ErrorMessage", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "field1",
			Message:  "error 1",
		})
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "field2",
			Message:  "error 2",
		})

		msg := result.ErrorMessage()
		assert.Contains(t, msg, "error 1")
		assert.Contains(t, msg, "error 2")
	})

	t.Run("VerboseErrorMessage with no errors returns empty", func(t *testing.T) {
		result := NewValidationResult()
		msg := result.VerboseErrorMessage()
		assert.Empty(t, msg)
	})

	t.Run("VerboseErrorMessage", func(t *testing.T) {
		result := NewValidationResult()
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "test error",
			Expected: "expected value",
		})

		msg := result.VerboseErrorMessage()
		assert.Contains(t, msg, "test")
		assert.Contains(t, msg, "Expected:")
	})
}

// TestHints tests the hint registration and retrieval functions.
//
// It verifies that:
//   - GetHint returns empty for unknown patterns
//   - GetHintForCommand returns registered command hints
//   - RegisterHint adds new patterns
//   - RegisterCommandHint adds command-specific hints
func TestHints(t *testing.T) {
	t.Run("GetHint returns empty for unknown pattern", func(t *testing.T) {
		err := stderrors.New("completely unknown error xyz123")
		hint := GetHint(err)
		assert.Empty(t, hint)
	})

	t.Run("GetHint returns nil for nil error", func(t *testing.T) {
		hint := GetHint(nil)
		assert.Empty(t, hint)
	})

	t.Run("GetHint matches permission denied", func(t *testing.T) {
		err := stderrors.New("failed: permission denied")
		hint := GetHint(err)
		assert.Contains(t, hint, "permissions")
	})

	t.Run("GetHint matches 404 pattern", func(t *testing.T) {
		err := stderrors.New("got 404 from registry")
		hint := GetHint(err)
		assert.Contains(t, hint, "package name")
	})

	t.Run("GetHintForCommand", func(t *testing.T) {
		// Register a test hint
		RegisterCommandHint("test-cmd-unique", "Check your test-cmd-unique setup")

		hint := GetHintForCommand("test-cmd-unique")
		assert.Contains(t, hint, "Check your test-cmd-unique setup")
	})

	t.Run("RegisterHint", func(t *testing.T) {
		RegisterHint("unique_test_pattern_xyz", "This is a test hint", "Resolution here")
		err := stderrors.New("unique_test_pattern_xyz happened")
		hint := GetHint(err)
		assert.Contains(t, hint, "This is a test hint")
	})
}

// Tests for display.go

// TestPrintErrorWithHints tests the PrintErrorWithHints function.
//
// Parameters:
//   - w: Writer for output
//   - errs: Slice of errors to print
//   - verbose: Whether to include detailed information
//
// It verifies that:
//   - Empty errors produce no output
//   - Single and multiple errors are formatted correctly
//   - Validation, unsupported, and partial success errors are handled
//   - Verbose mode includes additional details
//   - Nil errors in slice are skipped
func TestPrintErrorWithHints(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		var buf bytes.Buffer
		PrintErrorWithHints(&buf, []error{}, false)
		assert.Empty(t, buf.String())
	})

	t.Run("single error", func(t *testing.T) {
		var buf bytes.Buffer
		err := stderrors.New("test error")
		PrintErrorWithHints(&buf, []error{err}, false)
		assert.Contains(t, buf.String(), "Error: test error")
	})

	t.Run("multiple errors", func(t *testing.T) {
		var buf bytes.Buffer
		errs := []error{
			stderrors.New("error 1"),
			stderrors.New("error 2"),
		}
		PrintErrorWithHints(&buf, errs, false)
		output := buf.String()
		assert.Contains(t, output, "error 1")
		assert.Contains(t, output, "error 2")
	})

	t.Run("with validation error", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewConfigValidationError("field", "validation failed")
		PrintErrorWithHints(&buf, []error{err}, false)
		assert.Contains(t, buf.String(), "Validation Error")
	})

	t.Run("with validation error verbose", func(t *testing.T) {
		var buf bytes.Buffer
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "test error",
			Expected: "expected value",
		}
		PrintErrorWithHints(&buf, []error{err}, true)
		assert.Contains(t, buf.String(), "Expected")
	})

	t.Run("with unsupported error", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewUnsupportedError("update", "not supported", "pkg")
		PrintErrorWithHints(&buf, []error{err}, false)
		assert.Contains(t, buf.String(), "Unsupported")
	})

	t.Run("with unsupported error verbose", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewUnsupportedError("update", "not supported", "pkg")
		PrintErrorWithHints(&buf, []error{err}, true)
		output := buf.String()
		assert.Contains(t, output, "pkg")
		assert.Contains(t, output, "update")
	})

	t.Run("with partial success error", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewPartialSuccessError(3, 1, []error{stderrors.New("failed op")})
		PrintErrorWithHints(&buf, []error{err}, false)
		assert.Contains(t, buf.String(), "Partial Success")
	})

	t.Run("with partial success error verbose", func(t *testing.T) {
		var buf bytes.Buffer
		err := NewPartialSuccessError(3, 1, []error{stderrors.New("failed op")})
		PrintErrorWithHints(&buf, []error{err}, true)
		output := buf.String()
		assert.Contains(t, output, "Partial Success")
		assert.Contains(t, output, "Failed operations")
	})

	t.Run("nil error is skipped", func(t *testing.T) {
		var buf bytes.Buffer
		PrintErrorWithHints(&buf, []error{nil}, false)
		assert.Empty(t, buf.String())
	})
}

// TestFormatValidationError tests the FormatValidationError function.
//
// Parameters:
//   - err: ValidationError to format
//
// It verifies that:
//   - Nil error returns empty string
//   - Valid error is formatted with field and message
func TestFormatValidationError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FormatValidationError(nil)
		assert.Empty(t, result)
	})

	t.Run("with error", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "invalid",
		}
		result := FormatValidationError(err)
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "invalid")
	})
}

// TestFormatUnsupportedError tests the FormatUnsupportedError function.
//
// Parameters:
//   - err: UnsupportedError to format
//
// It verifies that:
//   - Nil error returns empty string
//   - Floating constraint includes guidance
//   - No lock file includes lock file guidance
//   - No outdated command includes command guidance
//   - Generic reasons don't include guidance
func TestFormatUnsupportedError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FormatUnsupportedError(nil)
		assert.Empty(t, result)
	})

	t.Run("floating constraint", func(t *testing.T) {
		err := &UnsupportedError{
			Package:   "pkg",
			Operation: "update",
			Reason:    "floating constraint",
		}
		result := FormatUnsupportedError(err)
		assert.Contains(t, result, "Guidance")
		assert.Contains(t, result, "Pin to a specific version")
	})

	t.Run("no lock file", func(t *testing.T) {
		err := &UnsupportedError{
			Package:   "pkg",
			Operation: "update",
			Reason:    "no lock file found",
		}
		result := FormatUnsupportedError(err)
		assert.Contains(t, result, "lock file")
	})

	t.Run("no outdated command", func(t *testing.T) {
		err := &UnsupportedError{
			Package:   "pkg",
			Operation: "check",
			Reason:    "no outdated command configured",
		}
		result := FormatUnsupportedError(err)
		assert.Contains(t, result, "outdated command")
	})

	t.Run("generic reason", func(t *testing.T) {
		err := &UnsupportedError{
			Package:   "pkg",
			Operation: "update",
			Reason:    "some other reason",
		}
		result := FormatUnsupportedError(err)
		assert.NotContains(t, result, "Guidance")
	})
}

// TestFormatValidationErrors tests the FormatValidationErrors function.
//
// Parameters:
//   - errs: Slice of ValidationError to format
//   - verbose: Whether to include detailed information
//
// It verifies that:
//   - Empty slice returns empty string
//   - Non-verbose mode shows basic error info
//   - Verbose mode includes Expected field
func TestFormatValidationErrors(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := FormatValidationErrors([]*ValidationError{}, false)
		assert.Empty(t, result)
	})

	t.Run("non-verbose", func(t *testing.T) {
		errs := []*ValidationError{
			{Category: ValidationCategoryConfig, Field: "field1", Message: "error1"},
			{Category: ValidationCategoryConfig, Field: "field2", Message: "error2"},
		}
		result := FormatValidationErrors(errs, false)
		assert.Contains(t, result, "Validation failed")
		assert.Contains(t, result, "error1")
		assert.Contains(t, result, "error2")
	})

	t.Run("verbose", func(t *testing.T) {
		errs := []*ValidationError{
			{Category: ValidationCategoryConfig, Field: "field", Message: "error", Expected: "expected value"},
		}
		result := FormatValidationErrors(errs, true)
		assert.Contains(t, result, "Expected")
	})
}

// TestValidationResultPrintTo tests the PrintTo method of ValidationResult.
//
// Parameters:
//   - w: Writer for output
//   - verbose: Whether to include detailed information
//
// It verifies that:
//   - Empty result produces no output
//   - Warnings are printed with "Warning:" prefix
//   - Errors are printed in non-verbose and verbose modes
//   - Both warnings and errors are printed together
func TestValidationResultPrintTo(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		var buf bytes.Buffer
		result := NewValidationResult()
		result.PrintTo(&buf, false)
		assert.Empty(t, buf.String())
	})

	t.Run("with warnings only", func(t *testing.T) {
		var buf bytes.Buffer
		result := NewValidationResult()
		result.AddWarning("test warning")
		result.PrintTo(&buf, false)
		assert.Contains(t, buf.String(), "Warning: test warning")
	})

	t.Run("with errors non-verbose", func(t *testing.T) {
		var buf bytes.Buffer
		result := NewValidationResult()
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "test error",
		})
		result.PrintTo(&buf, false)
		assert.Contains(t, buf.String(), "test error")
	})

	t.Run("with errors verbose", func(t *testing.T) {
		var buf bytes.Buffer
		result := NewValidationResult()
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "test error",
			Expected: "expected value",
		})
		result.PrintTo(&buf, true)
		assert.Contains(t, buf.String(), "Expected")
	})

	t.Run("with warnings and errors", func(t *testing.T) {
		var buf bytes.Buffer
		result := NewValidationResult()
		result.AddWarning("warning message")
		result.AddError(&ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "field",
			Message:  "error message",
		})
		result.PrintTo(&buf, false)
		output := buf.String()
		assert.Contains(t, output, "Warning: warning message")
		assert.Contains(t, output, "error message")
	})
}

// Tests for validation.go - additional coverage for Error() method

// TestValidationErrorAllBranches tests all branches of ValidationError.Error().
//
// It verifies that:
//   - Preflight category with command shows "command not found"
//   - Preflight with hint includes "Resolution" text
//   - Preflight without command falls through to default
//   - Config category with field shows "field: message"
//   - Config without field shows message only
//   - Package category with field shows "field: message"
//   - Default with message only shows message
//   - Default with command only shows "command not found"
//   - VerboseError with hint shows "Hint:" text
func TestValidationErrorAllBranches(t *testing.T) {
	t.Run("preflight with command and hint", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryPreflight,
			Command:  "npm",
			Hint:     "Install Node.js",
		}
		msg := err.Error()
		assert.Contains(t, msg, "command not found: npm")
		assert.Contains(t, msg, "Resolution: Install Node.js")
	})

	t.Run("preflight with command no hint", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryPreflight,
			Command:  "npm",
		}
		msg := err.Error()
		assert.Contains(t, msg, "command not found: npm")
		assert.Contains(t, msg, "Ensure 'npm' is installed")
	})

	t.Run("preflight without command", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryPreflight,
			Field:    "some-field",
			Message:  "some message",
		}
		msg := err.Error()
		// Falls through to default
		assert.Contains(t, msg, "some-field: some message")
	})

	t.Run("config with field", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "rules.npm",
			Message:  "invalid format",
		}
		msg := err.Error()
		assert.Contains(t, msg, "rules.npm: invalid format")
	})

	t.Run("config without field", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Message:  "general error",
		}
		msg := err.Error()
		assert.Equal(t, "general error", msg)
	})

	t.Run("package category with field", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryPackage,
			Field:    "lodash",
			Message:  "version error",
		}
		msg := err.Error()
		assert.Contains(t, msg, "lodash: version error")
	})

	t.Run("default with message only", func(t *testing.T) {
		err := &ValidationError{
			Message: "standalone message",
		}
		msg := err.Error()
		assert.Equal(t, "standalone message", msg)
	})

	t.Run("default with command only", func(t *testing.T) {
		err := &ValidationError{
			Command: "some-cmd",
		}
		msg := err.Error()
		assert.Contains(t, msg, "command not found: some-cmd")
	})

	t.Run("verbose with hint for non-preflight", func(t *testing.T) {
		err := &ValidationError{
			Category: ValidationCategoryConfig,
			Field:    "test",
			Message:  "error",
			Hint:     "Try this fix",
		}
		verbose := err.VerboseError()
		assert.Contains(t, verbose, "Hint: Try this fix")
	})
}
