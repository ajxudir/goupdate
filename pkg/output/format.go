// Package output provides formatters for exporting command results in various formats.
// It supports CSV, JSON, and XML output formats as alternatives to the default table display.
package output

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Format represents the output format type.
type Format string

const (
	// FormatTable is the default terminal table output.
	FormatTable Format = "table"
	// FormatCSV outputs data as comma-separated values.
	FormatCSV Format = "csv"
	// FormatJSON outputs data as JSON.
	FormatJSON Format = "json"
	// FormatXML outputs data as XML.
	FormatXML Format = "xml"
)

// ParseFormat parses a format string into a Format type.
//
// The parsing is case-insensitive. Valid values are "csv", "json", and "xml".
// Any unrecognized format returns FormatTable as the default.
//
// Parameters:
//   - s: Format string to parse (e.g., "csv", "JSON", "XmL")
//
// Returns:
//   - Format: The parsed format, or FormatTable if unrecognized
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "csv":
		return FormatCSV
	case "json":
		return FormatJSON
	case "xml":
		return FormatXML
	default:
		return FormatTable
	}
}

// IsStructuredFormat returns true if the format requires structured output (not table).
//
// Structured formats (CSV, JSON, XML) are typically used for machine consumption
// and require different data collection than the interactive table format.
//
// Parameters:
//   - f: The format to check
//
// Returns:
//   - bool: true if format is CSV, JSON, or XML; false for table format
func IsStructuredFormat(f Format) bool {
	return f == FormatCSV || f == FormatJSON || f == FormatXML
}

// Formatter handles writing data in a specific format.
//
// Fields:
//   - format: The output format (CSV, JSON, XML, or Table)
//   - writer: Destination for formatted output
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new formatter for the given format and writer.
//
// Parameters:
//   - format: The desired output format
//   - writer: Destination for formatted output
//
// Returns:
//   - *Formatter: A new formatter instance ready to write data
func NewFormatter(format Format, writer io.Writer) *Formatter {
	return &Formatter{
		format: format,
		writer: writer,
	}
}

// Format returns the current format.
//
// Returns:
//   - Format: The format this formatter is configured to use
func (f *Formatter) Format() Format {
	return f.format
}

// WriteCSV writes data as CSV to the output writer.
//
// It performs the following operations:
//   - Step 1: Creates a CSV writer
//   - Step 2: Writes the header row
//   - Step 3: Writes all data rows
//   - Step 4: Flushes the buffer and returns any errors
//
// Note: csv.Writer buffers all writes and only reports errors via Error() after Flush().
//
// Parameters:
//   - headers: Column headers for the CSV
//   - rows: Data rows, each row should have the same number of columns as headers
//
// Returns:
//   - error: When write or flush fails, returns the underlying error; otherwise returns nil
func (f *Formatter) WriteCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(f.writer)

	_ = w.Write(headers)
	for _, row := range rows {
		_ = w.Write(row)
	}

	w.Flush()
	return w.Error()
}

// WriteJSON writes data as compact JSON to the output writer.
//
// The output is compact (single line) for easy parsing by tools.
//
// Parameters:
//   - data: Data structure to encode as JSON (must be marshallable)
//
// Returns:
//   - error: When encoding fails, returns the underlying error; otherwise returns nil
func (f *Formatter) WriteJSON(data interface{}) error {
	encoder := json.NewEncoder(f.writer)
	return encoder.Encode(data)
}

// WriteXML writes data as XML to the output writer.
//
// It performs the following operations:
//   - Step 1: Writes the XML header (<?xml version="1.0"?>)
//   - Step 2: Encodes the data with 2-space indentation
//   - Step 3: Adds a trailing newline
//
// Parameters:
//   - data: Data structure to encode as XML (must be marshallable and have xml tags)
//
// Returns:
//   - error: When encoding fails, returns the underlying error; otherwise returns nil
func (f *Formatter) WriteXML(data interface{}) error {
	_, _ = fmt.Fprint(f.writer, xml.Header)
	encoder := xml.NewEncoder(f.writer)
	encoder.Indent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(f.writer)
	return nil
}
