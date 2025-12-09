package warnings

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetWarningWriterRestoresAndCaptures tests the behavior of SetWarningWriter.
//
// It verifies:
//   - Original writer is restored after calling restore function
//   - Warning messages are captured by the new writer
//   - nil writer defaults to os.Stderr
func TestSetWarningWriterRestoresAndCaptures(t *testing.T) {
	original := warnWriter

	var buf bytes.Buffer
	restore := SetWarningWriter(&buf)
	Warnf("test message\n")
	restore()

	assert.Equal(t, original, warnWriter)
	assert.Contains(t, buf.String(), "test message")

	restore = SetWarningWriter(nil)
	restore()
	assert.Equal(t, os.Stderr, warnWriter)
}

// TestWarningWriterReturnsCurrent tests the behavior of WarningWriter.
//
// It verifies:
//   - Returns the currently configured warning writer
//   - Reflects writer changes made by SetWarningWriter
//   - Returns to original writer after restore
func TestWarningWriterReturnsCurrent(t *testing.T) {
	original := warnWriter
	assert.Equal(t, original, WarningWriter())

	var buf bytes.Buffer
	restore := SetWarningWriter(&buf)
	assert.Equal(t, &buf, WarningWriter())
	restore()

	assert.Equal(t, original, WarningWriter())
}
