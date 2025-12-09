package formats

import (
	"fmt"
	"strings"
)

// GetFormatParser returns the appropriate parser for a given format.
//
// It performs the following operations:
//   - Trims whitespace from the format string
//   - Validates that the format is not empty
//   - Returns the corresponding parser implementation
//
// Parameters:
//   - format: The format name (e.g., "json", "yaml", "xml", "raw")
//
// Returns:
//   - FormatParser: The parser implementation for the specified format
//   - error: Returns an error if format is empty or unsupported; returns nil on success
func GetFormatParser(format string) (FormatParser, error) {
	format = strings.TrimSpace(format)
	if format == "" {
		return nil, fmt.Errorf("format cannot be empty")
	}

	switch format {
	case "json":
		return &JSONParser{}, nil
	case "yaml":
		return &YAMLParser{}, nil
	case "xml":
		return &XMLParser{}, nil
	case "raw":
		return &RawParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
