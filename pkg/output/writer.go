package output

import (
	"fmt"
	"io"
)

// WriteScanResult writes scan results in the specified format.
//
// It performs the following operations:
//   - Step 1: Creates a formatter for the requested format
//   - Step 2: Writes the scan result using format-specific logic
//
// Parameters:
//   - w: Destination writer for the output
//   - format: Output format (FormatJSON, FormatXML, or FormatCSV)
//   - result: Scan result data to write
//
// Returns:
//   - error: When format is unsupported, returns an error; when write fails, returns the underlying error; otherwise returns nil
func WriteScanResult(w io.Writer, format Format, result *ScanResult) error {
	formatter := NewFormatter(format, w)

	switch format {
	case FormatJSON:
		return formatter.WriteJSON(result)
	case FormatXML:
		return formatter.WriteXML(result)
	case FormatCSV:
		return writeScanCSV(formatter, result)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// writeScanCSV writes scan results in CSV format using the formatter.
//
// Parameters:
//   - f: The formatter instance to use for CSV writing
//   - result: Scan result data containing file entries
//
// Returns:
//   - error: When CSV write fails; returns nil on success
func writeScanCSV(f *Formatter, result *ScanResult) error {
	headers := []string{"RULE", "PM", "FORMAT", "FILE", "STATUS", "ERROR"}
	rows := make([][]string, 0, len(result.Files))
	for _, entry := range result.Files {
		rows = append(rows, []string{entry.Rule, entry.PM, entry.Format, entry.File, entry.Status, entry.Error})
	}
	return f.WriteCSV(headers, rows)
}

// WriteListResult writes list results in the specified format.
//
// It performs the following operations:
//   - Step 1: Creates a formatter for the requested format
//   - Step 2: Writes the list result using format-specific logic
//
// Parameters:
//   - w: Destination writer for the output
//   - format: Output format (FormatJSON, FormatXML, or FormatCSV)
//   - result: List result data to write
//
// Returns:
//   - error: When format is unsupported, returns an error; when write fails, returns the underlying error; otherwise returns nil
func WriteListResult(w io.Writer, format Format, result *ListResult) error {
	formatter := NewFormatter(format, w)

	switch format {
	case FormatJSON:
		return formatter.WriteJSON(result)
	case FormatXML:
		return formatter.WriteXML(result)
	case FormatCSV:
		return writeListCSV(formatter, result)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// writeListCSV writes list results in CSV format using the formatter.
//
// Parameters:
//   - f: The formatter instance to use for CSV writing
//   - result: List result data containing package entries
//
// Returns:
//   - error: When CSV write fails; returns nil on success
func writeListCSV(f *Formatter, result *ListResult) error {
	headers := []string{"RULE", "PM", "TYPE", "CONSTRAINT", "VERSION", "INSTALLED", "STATUS", "GROUP", "NAME"}
	rows := make([][]string, 0, len(result.Packages))
	for _, pkg := range result.Packages {
		rows = append(rows, []string{
			pkg.Rule,
			pkg.PM,
			pkg.Type,
			pkg.Constraint,
			pkg.Version,
			pkg.InstalledVersion,
			pkg.Status,
			pkg.Group,
			pkg.Name,
		})
	}
	return f.WriteCSV(headers, rows)
}

// WriteOutdatedResult writes outdated results in the specified format.
//
// It performs the following operations:
//   - Step 1: Creates a formatter for the requested format
//   - Step 2: Writes the outdated result using format-specific logic
//
// Parameters:
//   - w: Destination writer for the output
//   - format: Output format (FormatJSON, FormatXML, or FormatCSV)
//   - result: Outdated result data to write
//
// Returns:
//   - error: When format is unsupported, returns an error; when write fails, returns the underlying error; otherwise returns nil
func WriteOutdatedResult(w io.Writer, format Format, result *OutdatedResult) error {
	formatter := NewFormatter(format, w)

	switch format {
	case FormatJSON:
		return formatter.WriteJSON(result)
	case FormatXML:
		return formatter.WriteXML(result)
	case FormatCSV:
		return writeOutdatedCSV(formatter, result)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// writeOutdatedCSV writes outdated results in CSV format using the formatter.
//
// Parameters:
//   - f: The formatter instance to use for CSV writing
//   - result: Outdated result data containing package version information
//
// Returns:
//   - error: When CSV write fails; returns nil on success
func writeOutdatedCSV(f *Formatter, result *OutdatedResult) error {
	headers := []string{"RULE", "PM", "TYPE", "CONSTRAINT", "VERSION", "INSTALLED", "MAJOR", "MINOR", "PATCH", "STATUS", "GROUP", "NAME", "ERROR"}
	rows := make([][]string, 0, len(result.Packages))
	for _, pkg := range result.Packages {
		rows = append(rows, []string{
			pkg.Rule,
			pkg.PM,
			pkg.Type,
			pkg.Constraint,
			pkg.Version,
			pkg.InstalledVersion,
			pkg.Major,
			pkg.Minor,
			pkg.Patch,
			pkg.Status,
			pkg.Group,
			pkg.Name,
			pkg.Error,
		})
	}
	return f.WriteCSV(headers, rows)
}

// WriteUpdateResult writes update results in the specified format.
//
// It performs the following operations:
//   - Step 1: Creates a formatter for the requested format
//   - Step 2: Writes the update result using format-specific logic
//
// Parameters:
//   - w: Destination writer for the output
//   - format: Output format (FormatJSON, FormatXML, or FormatCSV)
//   - result: Update result data to write
//
// Returns:
//   - error: When format is unsupported, returns an error; when write fails, returns the underlying error; otherwise returns nil
func WriteUpdateResult(w io.Writer, format Format, result *UpdateResult) error {
	formatter := NewFormatter(format, w)

	switch format {
	case FormatJSON:
		return formatter.WriteJSON(result)
	case FormatXML:
		return formatter.WriteXML(result)
	case FormatCSV:
		return writeUpdateCSV(formatter, result)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// writeUpdateCSV writes update results in CSV format using the formatter.
//
// Parameters:
//   - f: The formatter instance to use for CSV writing
//   - result: Update result data containing package update information
//
// Returns:
//   - error: When CSV write fails; returns nil on success
func writeUpdateCSV(f *Formatter, result *UpdateResult) error {
	headers := []string{"RULE", "PM", "TYPE", "CONSTRAINT", "VERSION", "INSTALLED", "TARGET", "STATUS", "GROUP", "NAME", "ERROR"}
	rows := make([][]string, 0, len(result.Packages))
	for _, pkg := range result.Packages {
		rows = append(rows, []string{
			pkg.Rule,
			pkg.PM,
			pkg.Type,
			pkg.Constraint,
			pkg.Version,
			pkg.InstalledVersion,
			pkg.Target,
			pkg.Status,
			pkg.Group,
			pkg.Name,
			pkg.Error,
		})
	}
	return f.WriteCSV(headers, rows)
}
