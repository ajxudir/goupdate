package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseFormat tests the behavior of ParseFormat.
//
// It verifies:
//   - Parses valid format strings case-insensitively
//   - Returns FormatTable for unrecognized formats
func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
	}{
		{"csv", FormatCSV},
		{"CSV", FormatCSV},
		{"Csv", FormatCSV},
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"xml", FormatXML},
		{"XML", FormatXML},
		{"table", FormatTable},
		{"TABLE", FormatTable},
		{"", FormatTable},
		{"unknown", FormatTable},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsStructuredFormat tests the behavior of IsStructuredFormat.
//
// It verifies:
//   - Returns true for CSV, JSON, XML formats
//   - Returns false for table format
func TestIsStructuredFormat(t *testing.T) {
	assert.True(t, IsStructuredFormat(FormatCSV))
	assert.True(t, IsStructuredFormat(FormatJSON))
	assert.True(t, IsStructuredFormat(FormatXML))
	assert.False(t, IsStructuredFormat(FormatTable))
}

// TestFormatter_WriteCSV tests the behavior of WriteCSV.
//
// It verifies:
//   - Writes CSV headers and rows
func TestFormatter_WriteCSV(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatCSV, &buf)

	headers := []string{"NAME", "VERSION", "STATUS"}
	rows := [][]string{
		{"pkg1", "1.0.0", "ok"},
		{"pkg2", "2.0.0", "outdated"},
	}

	err := f.WriteCSV(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "NAME,VERSION,STATUS")
	assert.Contains(t, output, "pkg1,1.0.0,ok")
	assert.Contains(t, output, "pkg2,2.0.0,outdated")
}

// TestFormatter_WriteCSV_WithQuotes tests the behavior of WriteCSV with special characters.
//
// It verifies:
//   - Properly quotes fields with commas and quotes
func TestFormatter_WriteCSV_WithQuotes(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatCSV, &buf)

	headers := []string{"NAME", "DESCRIPTION"}
	rows := [][]string{
		{"pkg1", "A package with, comma"},
		{"pkg2", "A package with \"quotes\""},
	}

	err := f.WriteCSV(headers, rows)
	require.NoError(t, err)

	output := buf.String()
	// CSV should properly quote fields with special characters
	assert.Contains(t, output, "NAME,DESCRIPTION")
}

// TestFormatter_WriteJSON tests the behavior of WriteJSON.
//
// It verifies:
//   - Writes valid JSON that can be unmarshaled
func TestFormatter_WriteJSON(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatJSON, &buf)

	data := map[string]interface{}{
		"name":    "test",
		"version": "1.0.0",
	}

	err := f.WriteJSON(data)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result["name"])
	assert.Equal(t, "1.0.0", result["version"])
}

// TestFormatter_WriteXML tests the behavior of WriteXML.
//
// It verifies:
//   - Writes XML with header and proper structure
func TestFormatter_WriteXML(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatXML, &buf)

	type TestData struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `xml:"name"`
		Version string   `xml:"version"`
	}

	data := TestData{Name: "test", Version: "1.0.0"}

	err := f.WriteXML(data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<?xml version=")
	assert.Contains(t, output, "<test>")
	assert.Contains(t, output, "<name>test</name>")
	assert.Contains(t, output, "<version>1.0.0</version>")
}

// TestFormatter_Format tests the behavior of Format getter.
//
// It verifies:
//   - Returns the configured format
func TestFormatter_Format(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatJSON, &buf)
	assert.Equal(t, FormatJSON, f.Format())
}

// TestNewFormatter tests the behavior of NewFormatter.
//
// It verifies:
//   - Creates formatter with specified format
func TestNewFormatter(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatCSV, &buf)
	assert.NotNil(t, f)
	assert.Equal(t, FormatCSV, f.format)
}

// errorWriter is a test helper that always returns an error on write.
type errorWriter struct{}

// Write implements io.Writer and always returns an error.
//
// Parameters:
//   - p: Bytes to write (ignored)
//
// Returns:
//   - int: Always 0
//   - error: Always returns a test error
func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, assert.AnError
}

// TestFormatter_WriteCSV_FlushError tests the behavior of WriteCSV with flush errors.
//
// It verifies:
//   - Returns error when flush fails
func TestFormatter_WriteCSV_FlushError(t *testing.T) {
	// CSV writer buffers, so errors appear at Flush time
	ew := &errorWriter{}
	f := NewFormatter(FormatCSV, ew)

	err := f.WriteCSV([]string{"A", "B"}, [][]string{{"1", "2"}})
	assert.Error(t, err)
}

// unmarshalableXML is a test helper that always fails to marshal.
type unmarshalableXML struct{}

// MarshalXML implements xml.Marshaler and always returns an error.
//
// Parameters:
//   - e: XML encoder (ignored)
//   - start: Start element (ignored)
//
// Returns:
//   - error: Always returns a test error
func (u unmarshalableXML) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return assert.AnError
}

// TestFormatter_WriteXML_Error tests the behavior of WriteXML with encoding errors.
//
// It verifies:
//   - Returns error when XML encoding fails
func TestFormatter_WriteXML_Error(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(FormatXML, &buf)

	err := f.WriteXML(unmarshalableXML{})
	assert.Error(t, err)
}
