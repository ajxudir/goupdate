package errors

import (
	"errors"
	"fmt"
)

// Exit codes for scripting integration.
// These codes allow scripts to distinguish between different failure modes.
const (
	// ExitSuccess indicates all operations completed successfully.
	ExitSuccess = 0

	// ExitPartialFailure indicates some operations failed but others succeeded.
	// Use --continue-on-fail to allow partial success instead of full rollback.
	ExitPartialFailure = 1

	// ExitFailure indicates all operations failed or a critical error occurred.
	// This includes: config errors, validation failures, complete update failures.
	ExitFailure = 2

	// ExitConfigError indicates a configuration or validation error.
	// The command could not proceed due to invalid config or missing requirements.
	ExitConfigError = 3
)

// ExitError represents a command termination with a specific exit code.
//
// Use this error when a command needs to exit with a non-zero status
// while providing context about what went wrong.
//
// Fields:
//   - Code: Exit code (use constants ExitSuccess, ExitError, ExitPartialSuccess)
//   - Message: Human-readable error message
//   - Err: Underlying error that caused this exit, may be nil
//
// Example:
//
//	return &ExitError{
//	    Code:    ExitFailure,
//	    Message: "failed to load config",
//	    Err:     err,
//	}
type ExitError struct {
	// Code is the exit code for the command.
	// Standard codes: 0=success, 1=partial failure, 2=failure, 3=config error.
	Code int

	// Message is a human-readable description of why the command failed.
	Message string

	// Err is the underlying error that caused this exit.
	// May be nil if no underlying error exists.
	Err error
}

// Error implements the error interface.
//
// Returns the Message field if set, otherwise returns the underlying error's
// message, or a default message with the exit code.
//
// Returns:
//   - string: The error message
func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

// Unwrap returns the underlying error for errors.Is/As support.
//
// This enables using errors.Is() and errors.As() to check the wrapped error.
//
// Returns:
//   - error: The underlying error, or nil if none exists
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewExitError creates an ExitError with the given code and underlying error.
//
// Parameters:
//   - code: Exit code (use ExitSuccess, ExitPartialFailure, ExitFailure, ExitConfigError)
//   - err: Underlying error, may be nil
//
// Returns:
//   - *ExitError: New exit error
//
// Example:
//
//	err := errors.NewExitError(errors.ExitConfigError, configErr)
func NewExitError(code int, err error) *ExitError {
	return &ExitError{Code: code, Err: err}
}

// NewExitErrorf creates an ExitError with the given code and formatted message.
//
// Parameters:
//   - code: Exit code
//   - format: Printf-style format string
//   - args: Format arguments
//
// Returns:
//   - *ExitError: New exit error with formatted message
//
// Example:
//
//	err := errors.NewExitErrorf(errors.ExitFailure, "failed to process %s", filename)
func NewExitErrorf(code int, format string, args ...interface{}) *ExitError {
	return &ExitError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// GetExitCode extracts the exit code from an error.
//
// If err is nil, returns ExitSuccess.
// If err is an ExitError, returns its code.
// Otherwise returns ExitFailure.
//
// Parameters:
//   - err: The error to extract code from
//
// Returns:
//   - int: Exit code
//
// Example:
//
//	code := errors.GetExitCode(err)
//	os.Exit(code)
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	return ExitFailure
}

// IsExitError checks if err is an ExitError and returns it.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - *ExitError: The ExitError if err is one, nil otherwise
//   - bool: true if err is an ExitError
//
// Example:
//
//	if exitErr, ok := errors.IsExitError(err); ok {
//	    os.Exit(exitErr.Code)
//	}
func IsExitError(err error) (*ExitError, bool) {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr, true
	}
	return nil, false
}

// PartialSuccessError indicates that some operations succeeded while others failed.
//
// This is used when processing multiple packages and some updates succeed
// while others fail. The command should exit with ExitPartialFailure.
//
// Fields:
//   - Succeeded: Count of successful operations
//   - Failed: Count of failed operations
//   - Errors: Slice of errors from failed operations
//
// Example:
//
//	if failCount > 0 && successCount > 0 {
//	    return &PartialSuccessError{
//	        Succeeded: successCount,
//	        Failed:    failCount,
//	        Errors:    collectedErrors,
//	    }
//	}
type PartialSuccessError struct {
	// Succeeded is the number of operations that completed successfully.
	Succeeded int

	// Failed is the number of operations that failed.
	Failed int

	// Errors contains all errors from failed operations.
	Errors []error
}

// Error implements the error interface.
//
// Returns a summary message in the format "X succeeded, Y failed".
//
// Returns:
//   - string: Summary of succeeded and failed operation counts
func (e *PartialSuccessError) Error() string {
	return fmt.Sprintf("%d succeeded, %d failed", e.Succeeded, e.Failed)
}

// NewPartialSuccessError creates a PartialSuccessError with the given counts and errors.
//
// Parameters:
//   - succeeded: Number of successful operations
//   - failed: Number of failed operations
//   - errs: Slice of errors from failed operations
//
// Returns:
//   - *PartialSuccessError: New partial success error
//
// Example:
//
//	err := errors.NewPartialSuccessError(5, 2, failedErrs)
func NewPartialSuccessError(succeeded, failed int, errs []error) *PartialSuccessError {
	return &PartialSuccessError{
		Succeeded: succeeded,
		Failed:    failed,
		Errors:    errs,
	}
}

// IsPartialSuccess checks if err is a PartialSuccessError and returns it.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - *PartialSuccessError: The PartialSuccessError if err is one, nil otherwise
//   - bool: true if err is a PartialSuccessError
//
// Example:
//
//	if pse, ok := errors.IsPartialSuccess(err); ok {
//	    fmt.Printf("%d succeeded, %d failed\n", pse.Succeeded, pse.Failed)
//	}
func IsPartialSuccess(err error) (*PartialSuccessError, bool) {
	var pse *PartialSuccessError
	if errors.As(err, &pse) {
		return pse, true
	}
	return nil, false
}

// UnsupportedError indicates an operation is not supported for a package.
//
// This replaces the separate UnsupportedError types from pkg/outdated
// and pkg/update. Use this when a package cannot be processed due to
// its configuration or state.
//
// Fields:
//   - Operation: The operation that was attempted ("outdated", "update")
//   - Reason: Why the operation is not supported
//   - Package: Name of the package
//
// Example:
//
//	return &UnsupportedError{
//	    Operation: "update",
//	    Reason:    "floating constraint",
//	    Package:   pkg.Name,
//	}
type UnsupportedError struct {
	// Operation is the attempted operation ("outdated" or "update").
	Operation string

	// Reason explains why the operation is not supported.
	Reason string

	// Package is the name of the affected package.
	Package string
}

// Error implements the error interface.
//
// Formats the error message based on available fields. If Package is set,
// includes it in the format "package: operation not supported: reason".
// Otherwise formats as "operation not supported: reason" or just the reason.
//
// Returns:
//   - string: Formatted error message
func (e *UnsupportedError) Error() string {
	if e.Package != "" {
		return fmt.Sprintf("%s: %s not supported: %s", e.Package, e.Operation, e.Reason)
	}
	if e.Operation != "" {
		return fmt.Sprintf("%s not supported: %s", e.Operation, e.Reason)
	}
	return e.Reason
}

// IsUnsupportedError checks if err is an UnsupportedError and returns it.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - *UnsupportedError: The UnsupportedError if err is one, nil otherwise
//   - bool: true if err is an UnsupportedError
//
// Example:
//
//	if ue, ok := errors.IsUnsupportedError(err); ok {
//	    fmt.Printf("Cannot %s %s: %s\n", ue.Operation, ue.Package, ue.Reason)
//	}
func IsUnsupportedError(err error) (*UnsupportedError, bool) {
	var ue *UnsupportedError
	if errors.As(err, &ue) {
		return ue, true
	}
	return nil, false
}

// IsUnsupported reports whether the error indicates an unsupported operation.
// This is a convenience function for checking UnsupportedError without getting the value.
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: true if err is an UnsupportedError
func IsUnsupported(err error) bool {
	_, ok := IsUnsupportedError(err)
	return ok
}

// NewUnsupportedError creates an UnsupportedError with the given details.
//
// Parameters:
//   - operation: The operation that was attempted
//   - reason: Why the operation is not supported
//   - pkg: Name of the package (optional)
//
// Returns:
//   - *UnsupportedError: New unsupported error
//
// Example:
//
//	err := errors.NewUnsupportedError("update", "floating constraint", "lodash")
func NewUnsupportedError(operation, reason, pkg string) *UnsupportedError {
	return &UnsupportedError{
		Operation: operation,
		Reason:    reason,
		Package:   pkg,
	}
}
