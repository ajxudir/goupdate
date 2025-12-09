package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Progress provides a simple progress indicator for long-running operations.
//
// Fields:
//   - writer: Destination for progress output (typically os.Stderr)
//   - total: Total number of steps in the operation
//   - current: Current step number
//   - message: Descriptive message displayed with the progress
//   - mu: Mutex to protect concurrent access to progress state
//   - enabled: Whether progress output is enabled
//   - lastWidth: Width of the last rendered progress line for proper clearing
type Progress struct {
	writer    io.Writer
	total     int
	current   int
	message   string
	mu        sync.Mutex
	enabled   bool
	lastWidth int
}

// NewProgress creates a new progress indicator and returns it.
//
// Parameters:
//   - writer: Destination for progress output (typically os.Stderr)
//   - total: Total number of steps in the operation
//   - message: Descriptive message to display (e.g., "Processing packages")
//
// Returns:
//   - *Progress: A new progress indicator initialized and enabled
func NewProgress(writer io.Writer, total int, message string) *Progress {
	return &Progress{
		writer:  writer,
		total:   total,
		message: message,
		enabled: true,
	}
}

// SetEnabled enables or disables progress output.
//
// This is useful for suppressing progress in non-interactive environments
// or when structured output formats are used.
//
// Parameters:
//   - enabled: true to enable progress output; false to disable
func (p *Progress) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// Increment advances the progress by one step and re-renders the display.
//
// It performs the following operations:
//   - Step 1: Locks mutex, increments counter, copies values, unlocks
//   - Step 2: Renders progress outside the critical section to prevent I/O deadlocks
//
// This method is thread-safe and can be called concurrently from multiple goroutines.
func (p *Progress) Increment() {
	p.mu.Lock()
	p.current++
	current := p.current
	total := p.total
	enabled := p.enabled
	p.mu.Unlock()

	if enabled && total > 0 {
		p.renderValues(current, total)
	}
}

// SetCurrent sets the current progress value to a specific step and re-renders.
//
// It performs the following operations:
//   - Step 1: Locks mutex, updates current value, copies values, unlocks
//   - Step 2: Renders progress outside the critical section to prevent I/O deadlocks
//
// Parameters:
//   - current: The step number to set (0 to total)
//
// This method is thread-safe.
func (p *Progress) SetCurrent(current int) {
	p.mu.Lock()
	p.current = current
	total := p.total
	enabled := p.enabled
	p.mu.Unlock()

	if enabled && total > 0 {
		p.renderValues(current, total)
	}
}

// Done marks the progress as complete and prints a newline.
//
// It performs the following operations:
//   - Step 1: Sets current to total to show 100% completion
//   - Step 2: Renders the final progress state
//   - Step 3: Prints a newline to move past the progress line
//
// This should be called when the operation completes successfully.
func (p *Progress) Done() {
	p.mu.Lock()
	p.current = p.total
	current := p.current
	total := p.total
	enabled := p.enabled
	p.mu.Unlock()

	if enabled && total > 0 {
		p.renderValues(current, total)
		_, _ = fmt.Fprintln(p.writer)
	}
}

// Clear clears the progress line from the display.
//
// This overwrites the current progress line with spaces and returns the cursor
// to the beginning. Useful when you need to print other content without the
// progress indicator interfering.
func (p *Progress) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.enabled && p.lastWidth > 0 {
		_, _ = fmt.Fprintf(p.writer, "\r%s\r", strings.Repeat(" ", p.lastWidth))
	}
}

// render renders progress using current struct values.
//
// It performs the following operations:
//   - Step 1: Locks mutex, copies values, unlocks
//   - Step 2: Calls renderValues with the copied values
//
// NOTE: This method is kept for backwards compatibility but should not be called
// while holding the lock. Use renderValues directly with copied values instead.
func (p *Progress) render() {
	p.mu.Lock()
	enabled := p.enabled
	total := p.total
	current := p.current
	p.mu.Unlock()

	if !enabled || total == 0 {
		return
	}
	p.renderValues(current, total)
}

// renderValues renders progress with the given values.
//
// It performs the following operations:
//   - Step 1: Calculates percentage from current and total
//   - Step 2: Formats the progress line with message and percentage
//   - Step 3: Locks mutex briefly to update lastWidth and pad if needed
//   - Step 4: Writes to the output writer and flushes if it's a file
//
// This method is safe to call without holding the lock for current/total,
// but uses the lock for lastWidth to prevent display corruption.
//
// Parameters:
//   - current: Current step number
//   - total: Total number of steps
func (p *Progress) renderValues(current, total int) {
	percentage := float64(current) / float64(total) * 100
	line := fmt.Sprintf("\r%s: %d/%d (%.0f%%)", p.message, current, total, percentage)

	// Lock only for lastWidth access to prevent display corruption
	p.mu.Lock()
	// Clear previous content if the new line is shorter
	if len(line) < p.lastWidth {
		padding := strings.Repeat(" ", p.lastWidth-len(line))
		line += padding
	}
	p.lastWidth = len(line)
	p.mu.Unlock()

	_, _ = fmt.Fprint(p.writer, line)

	// Flush stderr to ensure progress renders immediately in CI environments
	if f, ok := p.writer.(*os.File); ok {
		_ = f.Sync()
	}
}
