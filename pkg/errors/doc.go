// Package errors provides unified error types and display for goupdate.
//
// This package consolidates all error handling into a single location:
//   - ExitError: Command exit with specific exit code
//   - PartialSuccessError: Some operations succeeded, some failed
//   - ValidationError: Configuration or preflight validation failures
//   - UnsupportedError: Operations not supported for specific packages
//
// Error Display:
//
// The package provides consistent error formatting with actionable hints:
//
//	errors.PrintErrorWithHints(os.Stderr, errs, verbose)
//
// Error Checking:
//
// Use the Is* functions to check error types:
//
//	if exitErr, ok := errors.IsExitError(err); ok {
//	    os.Exit(exitErr.Code)
//	}
//
// Exit Codes:
//
// Standard exit codes are defined for scripting integration:
//   - ExitSuccess (0): All operations completed successfully
//   - ExitPartialFailure (1): Some operations failed
//   - ExitFailure (2): All operations failed or critical error
//   - ExitConfigError (3): Configuration or validation error
package errors
