package warnings

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	mu         sync.RWMutex
	warnWriter io.Writer = os.Stderr
)

// Warnf writes formatted warning messages to the configured warning writer.
//
// It performs the following operations:
//   - Acquires a read lock to safely access the warning writer
//   - Formats the message using the provided format string and arguments
//   - Writes the formatted message to the configured writer
//   - Releases the read lock
//
// Parameters:
//   - format: Printf-style format string for the warning message
//   - args: Variadic arguments to format into the string
//
// Returns:
//   - None
func Warnf(format string, args ...any) {
	mu.RLock()
	w := warnWriter
	mu.RUnlock()
	_, _ = fmt.Fprintf(w, format, args...)
}

// WarningWriter returns the currently configured warning writer.
//
// It performs the following operations:
//   - Acquires a read lock to ensure thread-safe access
//   - Reads the current warning writer value
//   - Releases the read lock
//
// Returns:
//   - io.Writer: The currently configured writer for warning messages
func WarningWriter() io.Writer {
	mu.RLock()
	defer mu.RUnlock()
	return warnWriter
}

// SetWarningWriter swaps the warning writer and returns a restore function.
//
// It performs the following operations:
//   - Acquires a write lock to ensure thread-safe modification
//   - Saves the previous warning writer for restoration
//   - Sets the new warning writer (defaults to os.Stderr if nil)
//   - Releases the write lock
//   - Returns a function that restores the previous writer when called
//
// Parameters:
//   - w: The new io.Writer to use; if nil, defaults to os.Stderr
//
// Returns:
//   - func(): A restore function that sets the writer back to the previous value
func SetWarningWriter(w io.Writer) func() {
	mu.Lock()
	defer mu.Unlock()

	previous := warnWriter
	if w == nil {
		warnWriter = os.Stderr
	} else {
		warnWriter = w
	}

	return func() {
		mu.Lock()
		defer mu.Unlock()
		warnWriter = previous
	}
}
