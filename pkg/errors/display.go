package errors

import (
	"fmt"
	"io"
	"strings"
)

// PrintErrorWithHints prints errors with actionable hints to the writer.
//
// This is the single implementation for error display across all commands.
// It formats errors consistently and looks up hints for each error.
//
// Parameters:
//   - w: Writer to output to (typically os.Stderr)
//   - errs: Slice of errors to display
//   - verbose: If true, includes additional details for validation errors
//
// Output format:
//
//	Error: <error message>
//	  Hint: <actionable hint if available>
//
// Example:
//
//	errors.PrintErrorWithHints(os.Stderr, collectedErrors, verbose)
func PrintErrorWithHints(w io.Writer, errs []error, verbose bool) {
	if len(errs) == 0 {
		return
	}

	for _, err := range errs {
		printSingleError(w, err, verbose)
	}
}

// printSingleError prints a single error with appropriate formatting.
//
// This function determines the error type and dispatches to the appropriate
// formatter. It handles ValidationError, UnsupportedError, PartialSuccessError,
// and standard errors differently.
//
// Parameters:
//   - w: Writer to output to
//   - err: The error to print
//   - verbose: If true, includes detailed information
func printSingleError(w io.Writer, err error, verbose bool) {
	if err == nil {
		return
	}

	// Check for validation errors - format specially
	if ve, ok := IsValidationError(err); ok {
		printValidationError(w, ve, verbose)
		return
	}

	// Check for unsupported errors - format specially
	if ue, ok := IsUnsupportedError(err); ok {
		printUnsupportedError(w, ue, verbose)
		return
	}

	// Check for partial success errors - format specially
	if pse, ok := IsPartialSuccess(err); ok {
		printPartialSuccessError(w, pse, verbose)
		return
	}

	// Standard error with hint lookup
	enhanced := EnhanceErrorWithHint(err)
	_, _ = fmt.Fprintf(w, "Error: %s\n", enhanced)
}

// printValidationError prints a validation error with appropriate detail level.
//
// In verbose mode, prints the full VerboseError with expected values and hints.
// Otherwise, prints the standard Error message.
//
// Parameters:
//   - w: Writer to output to
//   - err: The validation error to print
//   - verbose: If true, includes expected values and documentation links
func printValidationError(w io.Writer, err *ValidationError, verbose bool) {
	if verbose {
		_, _ = fmt.Fprintf(w, "Validation Error: %s\n", err.VerboseError())
	} else {
		_, _ = fmt.Fprintf(w, "Validation Error: %s\n", err.Error())
	}
}

// printUnsupportedError prints an unsupported operation error.
//
// In verbose mode and when package information is available, prints detailed
// information including package, operation, and reason.
//
// Parameters:
//   - w: Writer to output to
//   - err: The unsupported error to print
//   - verbose: If true, includes package details when available
func printUnsupportedError(w io.Writer, err *UnsupportedError, verbose bool) {
	if verbose && err.Package != "" {
		_, _ = fmt.Fprintf(w, "Unsupported: %s - %s (%s)\n", err.Package, err.Operation, err.Reason)
	} else {
		_, _ = fmt.Fprintf(w, "Unsupported: %s\n", err.Error())
	}
}

// printPartialSuccessError prints partial success details.
//
// Prints a summary of succeeded and failed operations. In verbose mode,
// also prints detailed information about each failed operation with hints.
//
// Parameters:
//   - w: Writer to output to
//   - err: The partial success error to print
//   - verbose: If true, includes detailed failure information with hints
func printPartialSuccessError(w io.Writer, err *PartialSuccessError, verbose bool) {
	_, _ = fmt.Fprintf(w, "Partial Success: %s\n", err.Error())
	if verbose && len(err.Errors) > 0 {
		_, _ = fmt.Fprintf(w, "  Failed operations:\n")
		for _, e := range err.Errors {
			_, _ = fmt.Fprintf(w, "    - %s\n", EnhanceErrorWithHint(e))
		}
	}
}

// FormatValidationError formats a ValidationError for display.
//
// Parameters:
//   - err: The validation error to format
//
// Returns:
//   - string: Formatted error message with field, message, and valid options
//
// Example output:
//
//	Config validation error in 'rules.npm.format':
//	  invalid format type
//	  Expected: one of: json, yaml, xml, raw
//	  See: docs/user/configuration.md#format
func FormatValidationError(err *ValidationError) string {
	if err == nil {
		return ""
	}
	return err.VerboseError()
}

// FormatUnsupportedError formats an UnsupportedError with guidance.
//
// Parameters:
//   - err: The unsupported error to format
//
// Returns:
//   - string: Formatted message explaining why and what to do
func FormatUnsupportedError(err *UnsupportedError) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(err.Error())

	// Add guidance based on reason
	switch {
	case strings.Contains(err.Reason, "floating"):
		sb.WriteString("\n  Guidance: Floating constraints cannot be updated automatically. Pin to a specific version.")
	case strings.Contains(err.Reason, "no lock"):
		sb.WriteString("\n  Guidance: Run the package manager's install command to generate a lock file.")
	case strings.Contains(err.Reason, "no outdated"):
		sb.WriteString("\n  Guidance: Configure an outdated command in your config file.")
	}

	return sb.String()
}

// FormatErrorsWithHints formats multiple errors with hints for display.
//
// Parameters:
//   - errs: Slice of errors to format
//
// Returns:
//   - string: Formatted error messages, each prefixed with an error indicator
//
// Example output:
//
//	Error: failed to parse config
//	  Hint: Check file syntax: Validate JSON/YAML syntax using a linter
//	Error: command not found: npm
//	  Resolution: Install Node.js: https://nodejs.org/
func FormatErrorsWithHints(errs []error) string {
	if len(errs) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, err := range errs {
		sb.WriteString("\u274C " + EnhanceErrorWithHint(err) + "\n")
	}
	return sb.String()
}

// FormatValidationErrors formats multiple validation errors.
//
// Parameters:
//   - errs: Slice of validation errors
//   - verbose: If true, includes detailed information
//
// Returns:
//   - string: Formatted validation errors
func FormatValidationErrors(errs []*ValidationError, verbose bool) string {
	if len(errs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Validation failed:\n")

	for _, err := range errs {
		if verbose {
			sb.WriteString(fmt.Sprintf("  - %s\n", err.VerboseError()))
		} else {
			sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
		}
	}

	return sb.String()
}

// ValidationResult holds the results of validation operations.
//
// This replaces the separate ValidationResult types from pkg/config
// and pkg/preflight with a unified type.
//
// Fields:
//   - Errors: Slice of validation errors
//   - Warnings: Slice of warning messages
type ValidationResult struct {
	// Errors contains all validation errors encountered.
	Errors []*ValidationError

	// Warnings contains non-fatal warning messages.
	Warnings []string
}

// HasErrors returns true if there are any validation errors.
//
// Returns:
//   - bool: true if the result contains one or more validation errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
//
// Returns:
//   - bool: true if the result contains one or more warning messages
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// AddError adds a validation error to the result.
//
// Parameters:
//   - err: The validation error to add to the errors list
func (r *ValidationResult) AddError(err *ValidationError) {
	r.Errors = append(r.Errors, err)
}

// AddWarning adds a warning message to the result.
//
// Parameters:
//   - msg: The warning message to add to the warnings list
func (r *ValidationResult) AddWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// ErrorMessage returns a formatted error message for all validation errors.
//
// Returns:
//   - string: Formatted error messages, or empty string if no errors
func (r *ValidationResult) ErrorMessage() string {
	if len(r.Errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Validation failed:\n")
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}

// VerboseErrorMessage returns detailed error messages with hints.
//
// Returns:
//   - string: Detailed error messages with hints, or empty string if no errors
func (r *ValidationResult) VerboseErrorMessage() string {
	if len(r.Errors) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Validation failed:\n")
	for _, err := range r.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.VerboseError()))
	}
	return sb.String()
}

// PrintTo writes validation results to the given writer.
//
// Parameters:
//   - w: Writer to output to
//   - verbose: If true, includes detailed error information
func (r *ValidationResult) PrintTo(w io.Writer, verbose bool) {
	if len(r.Warnings) > 0 {
		for _, warning := range r.Warnings {
			_, _ = fmt.Fprintf(w, "Warning: %s\n", warning)
		}
	}

	if len(r.Errors) > 0 {
		if verbose {
			_, _ = fmt.Fprint(w, r.VerboseErrorMessage())
		} else {
			_, _ = fmt.Fprint(w, r.ErrorMessage())
		}
	}
}

// NewValidationResult creates a new empty ValidationResult.
//
// Initializes the Errors and Warnings slices to empty (non-nil) slices.
//
// Returns:
//   - *ValidationResult: New validation result with empty error and warning slices
//
// Example:
//
//	result := errors.NewValidationResult()
//	result.AddError(validationErr)
//	if result.HasErrors() {
//	    return result
//	}
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Errors:   make([]*ValidationError, 0),
		Warnings: make([]string, 0),
	}
}
