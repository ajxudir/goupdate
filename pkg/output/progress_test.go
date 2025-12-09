package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProgress_Basic tests the basic behavior of Progress.
//
// It verifies:
//   - Increments progress and shows percentage
func TestProgress_Basic(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Processing")

	p.Increment()
	p.Increment()
	p.Increment()

	output := buf.String()
	assert.Contains(t, output, "Processing")
	assert.Contains(t, output, "3/10")
	assert.Contains(t, output, "30%")
}

// TestProgress_Done tests the behavior of Done.
//
// It verifies:
//   - Marks progress as 100% complete
//   - Ends with newline
func TestProgress_Done(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 5, "Testing")

	p.Increment()
	p.Increment()
	p.Done()

	output := buf.String()
	assert.Contains(t, output, "5/5")
	assert.Contains(t, output, "100%")
	// Should end with newline
	assert.True(t, strings.HasSuffix(output, "\n"))
}

// TestProgress_SetCurrent tests the behavior of SetCurrent.
//
// It verifies:
//   - Sets progress to specific value and shows correct percentage
func TestProgress_SetCurrent(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 100, "Loading")

	p.SetCurrent(50)

	output := buf.String()
	assert.Contains(t, output, "50/100")
	assert.Contains(t, output, "50%")
}

// TestProgress_Clear tests the behavior of Clear.
//
// It verifies:
//   - Clears progress line with spaces and carriage return
func TestProgress_Clear(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Test")

	p.Increment()
	p.Clear()

	// After clear, the line should be overwritten with spaces
	output := buf.String()

	// Verify the output includes the initial render and clear pattern
	assert.NotEmpty(t, output, "output should not be empty")
	assert.Contains(t, output, "Test", "should contain original message")
	assert.Contains(t, output, "1/10", "should contain progress")

	// Clear should add carriage returns and spaces to overwrite
	assert.Contains(t, output, "\r", "should contain carriage return for clearing")

	// The output should end with spaces and carriage return from Clear()
	// Clear() writes: "\r" + spaces + "\r"
	assert.True(t, strings.HasSuffix(output, "\r"), "should end with carriage return from Clear()")
}

// TestProgress_Disabled tests the behavior when progress is disabled.
//
// It verifies:
//   - No output when progress is disabled
func TestProgress_Disabled(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Test")
	p.SetEnabled(false)

	p.Increment()
	p.Increment()
	p.Done()

	// Nothing should be written when disabled
	assert.Empty(t, buf.String())
}

// TestProgress_ZeroTotal tests the behavior with zero total.
//
// It verifies:
//   - Does not panic with zero total
//   - Produces no output
func TestProgress_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 0, "Test")

	// Should not panic with zero total
	p.Increment()
	p.Done()

	// Nothing meaningful should be rendered
	assert.Empty(t, buf.String())
}

// TestProgress_PercentageCalculation tests the behavior of percentage calculations.
//
// It verifies:
//   - Calculates correct percentages at different progress points
func TestProgress_PercentageCalculation(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 4, "Test")

	p.SetCurrent(1)
	output := buf.String()
	assert.Contains(t, output, "25%")

	p.SetCurrent(2)
	output = buf.String()
	assert.Contains(t, output, "50%")

	p.SetCurrent(3)
	output = buf.String()
	assert.Contains(t, output, "75%")

	p.SetCurrent(4)
	output = buf.String()
	assert.Contains(t, output, "100%")
}

// TestNewProgress tests the behavior of NewProgress.
//
// It verifies:
//   - Creates progress with correct initial state
func TestNewProgress(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Loading items")

	assert.NotNil(t, p)
	assert.Equal(t, 10, p.total)
	assert.Equal(t, 0, p.current)
	assert.Equal(t, "Loading items", p.message)
	assert.True(t, p.enabled)
}

// TestProgress_PaddingWhenLineShorter tests the behavior when progress line gets shorter.
//
// It verifies:
//   - Pads with spaces to clear previous longer line
func TestProgress_PaddingWhenLineShorter(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 100, "Processing")

	// First render with a longer line (larger percentage number)
	p.SetCurrent(99) // "Processing: 99/100 (99%)"
	initialLen := p.lastWidth
	assert.Greater(t, initialLen, 0, "initial render should set lastWidth")

	// Capture what 99/100 produces
	initialOutput := buf.String()
	assert.Contains(t, initialOutput, "99/100", "initial output should show 99/100")
	assert.Contains(t, initialOutput, "99%", "initial output should show 99%")

	// Now go back to a smaller number - this won't happen in practice but tests padding
	buf.Reset()
	p.current = 1 // Reset to 1 directly
	p.render()    // "Processing: 1/100 (1%)" - shorter line

	// The output should be padded to clear the previous longer line
	output := buf.String()

	// Verify the shorter line appears
	assert.Contains(t, output, "1/100", "padded output should show 1/100")
	assert.Contains(t, output, "1%", "padded output should show 1%")

	// Verify padding is applied - output length should be at least as long as initial
	assert.GreaterOrEqual(t, len(output), initialLen, "output should be padded to at least initial length")

	// Verify padding spaces exist at end (after percentage)
	// Since "1/100 (1%)" is shorter than "99/100 (99%)", there should be trailing spaces
	trimmedLen := len(strings.TrimRight(output, " "))
	assert.Less(t, trimmedLen, len(output), "output should have trailing padding spaces")
}

// TestProgress_ClearWithoutRender tests the behavior of Clear without prior render.
//
// It verifies:
//   - Clear without render produces no output
func TestProgress_ClearWithoutRender(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgress(&buf, 10, "Test")

	// Clear without any render should not write anything
	p.Clear()

	assert.Empty(t, buf.String())
}
