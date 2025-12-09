package display

import (
	"io"
	"os"

	"github.com/ajxudir/goupdate/pkg/output"
)

// Progress re-exports output.Progress for convenience.
// Use NewProgress or NewStderrProgress to create instances.
type Progress = output.Progress

// NewProgress creates a progress indicator for long-running operations.
//
// Progress indicators show the current state of batch operations,
// updating in place on the terminal. They are thread-safe and can
// be updated from concurrent goroutines.
//
// Parameters:
//   - w: Writer to output progress to (typically os.Stderr)
//   - total: Total number of items to process
//   - message: Message prefix shown before the progress (e.g., "Processing")
//
// Returns:
//   - *Progress: A new progress indicator ready for use
//
// Example:
//
//	progress := display.NewProgress(os.Stderr, 10, "Checking packages")
//	for i := 0; i < 10; i++ {
//	    // ... do work ...
//	    progress.Increment()
//	}
//	progress.Done()
func NewProgress(w io.Writer, total int, message string) *Progress {
	return output.NewProgress(w, total, message)
}

// NewStderrProgress creates a progress indicator that writes to stderr.
//
// This is a convenience wrapper around NewProgress that uses os.Stderr
// as the output writer, which is the most common use case.
//
// Parameters:
//   - total: Total number of items to process
//   - message: Message prefix shown before the progress
//
// Returns:
//   - *Progress: A new progress indicator writing to stderr
//
// Example:
//
//	progress := display.NewStderrProgress(len(packages), "Updating")
//	for _, pkg := range packages {
//	    updatePackage(pkg)
//	    progress.Increment()
//	}
//	progress.Done()
func NewStderrProgress(total int, message string) *Progress {
	return output.NewProgress(os.Stderr, total, message)
}

// NewDisabledProgress creates a progress indicator that produces no output.
//
// Use this when progress output should be suppressed, such as when
// running in non-interactive mode, JSON output mode, or during tests.
//
// Parameters:
//   - total: Total number of items (still tracked internally)
//   - message: Message (unused but required for interface consistency)
//
// Returns:
//   - *Progress: A disabled progress indicator
//
// Example:
//
//	var progress *display.Progress
//	if interactive {
//	    progress = display.NewStderrProgress(total, "Processing")
//	} else {
//	    progress = display.NewDisabledProgress(total, "Processing")
//	}
func NewDisabledProgress(total int, message string) *Progress {
	p := output.NewProgress(io.Discard, total, message)
	p.SetEnabled(false)
	return p
}

// ProgressConfig holds configuration for creating progress indicators.
//
// Fields:
//   - Writer: Output destination (defaults to os.Stderr if nil)
//   - Total: Total items to process
//   - Message: Progress message prefix
//   - Enabled: Whether progress is displayed (defaults to true)
//
// Example:
//
//	config := display.ProgressConfig{
//	    Total:   len(packages),
//	    Message: "Scanning files",
//	    Enabled: !quietMode,
//	}
//	progress := display.NewProgressFromConfig(config)
type ProgressConfig struct {
	// Writer is the output destination.
	// If nil, defaults to os.Stderr.
	Writer io.Writer

	// Total is the total number of items to process.
	Total int

	// Message is the prefix shown before the progress.
	Message string

	// Enabled controls whether progress is displayed.
	// Defaults to true.
	Enabled bool
}

// NewProgressFromConfig creates a progress indicator from configuration.
//
// This provides more control over progress indicator behavior than
// the simpler NewProgress function.
//
// Parameters:
//   - config: Configuration for the progress indicator
//
// Returns:
//   - *Progress: A configured progress indicator
//
// Example:
//
//	progress := display.NewProgressFromConfig(display.ProgressConfig{
//	    Writer:  os.Stderr,
//	    Total:   100,
//	    Message: "Processing files",
//	    Enabled: showProgress,
//	})
func NewProgressFromConfig(config ProgressConfig) *Progress {
	w := config.Writer
	if w == nil {
		w = os.Stderr
	}

	p := output.NewProgress(w, config.Total, config.Message)
	p.SetEnabled(config.Enabled)
	return p
}

// WithProgress executes a function while showing progress.
//
// This is a convenience wrapper that creates a progress indicator,
// passes it to the function, and ensures Done() is called on completion.
//
// Parameters:
//   - w: Writer for progress output
//   - total: Total items to process
//   - message: Progress message prefix
//   - fn: Function to execute, receives progress indicator
//
// Returns:
//   - error: Any error returned by the function
//
// Example:
//
//	err := display.WithProgress(os.Stderr, len(packages), "Updating", func(p *display.Progress) error {
//	    for _, pkg := range packages {
//	        if err := updatePackage(pkg); err != nil {
//	            return err
//	        }
//	        p.Increment()
//	    }
//	    return nil
//	})
func WithProgress(w io.Writer, total int, message string, fn func(*Progress) error) error {
	p := NewProgress(w, total, message)
	defer p.Done()
	return fn(p)
}

// WithProgressConditional executes a function with optional progress.
//
// If enabled is true, shows progress; otherwise uses a disabled progress
// indicator that still tracks counts but produces no output.
//
// Parameters:
//   - w: Writer for progress output
//   - total: Total items to process
//   - message: Progress message prefix
//   - enabled: Whether to show progress output
//   - fn: Function to execute
//
// Returns:
//   - error: Any error returned by the function
//
// Example:
//
//	err := display.WithProgressConditional(os.Stderr, len(packages), "Checking",
//	    !quietMode, func(p *display.Progress) error {
//	        // ... processing ...
//	    })
func WithProgressConditional(w io.Writer, total int, message string, enabled bool, fn func(*Progress) error) error {
	var p *Progress
	if enabled {
		p = NewProgress(w, total, message)
	} else {
		p = NewDisabledProgress(total, message)
	}
	defer p.Done()
	return fn(p)
}
