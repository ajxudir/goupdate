// Package packages provides dynamic parsing of package manifest files.
// It dispatches to format-specific parsers based on configuration and
// supports discovery of manifest files in a project directory.
package packages

import (
	"fmt"
	"os"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// DynamicParser coordinates parsing of files using the configured formats.
//
// It acts as a dispatcher that reads manifest files and delegates to the
// appropriate format-specific parser based on the package manager configuration.
//
// Fields: This type has no exported fields.
type DynamicParser struct{}

// NewDynamicParser creates a new DynamicParser instance.
//
// Returns:
//   - *DynamicParser: A parser capable of handling any configured format
func NewDynamicParser() *DynamicParser {
	return &DynamicParser{}
}

// ParseFile reads a manifest file and parses its packages using format-specific logic.
//
// It performs the following operations:
//   - Validates the package manager configuration
//   - Reads the file contents from disk
//   - Dispatches to the appropriate format parser (JSON, YAML, TOML, etc.)
//   - Returns a structured list of packages with their metadata
//
// Parameters:
//   - filePath: Absolute or relative path to the manifest file to parse
//   - cfg: Package manager configuration specifying format, fields, and parsing rules
//
// Returns:
//   - *formats.PackageList: Parsed packages with source file information
//   - error: When cfg is nil, returns error; when format is missing, returns error;
//     when fields are missing, returns error; when file read fails, returns error;
//     when format is unsupported, returns error; when parsing fails, returns error;
//     otherwise returns nil
func (dp *DynamicParser) ParseFile(filePath string, cfg *config.PackageManagerCfg) (*formats.PackageList, error) {
	if cfg == nil {
		return nil, fmt.Errorf("package manager configuration is required")
	}

	if strings.TrimSpace(cfg.Format) == "" {
		return nil, fmt.Errorf("format missing for %s", filePath)
	}

	if len(cfg.Fields) == 0 {
		return nil, fmt.Errorf("fields configuration missing for %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	parser, err := formats.GetFormatParser(cfg.Format)
	if err != nil {
		return nil, err
	}

	packages, err := parser.Parse(content, cfg)
	if err != nil {
		return nil, err
	}

	return &formats.PackageList{
		Packages: packages,
		Source:   filePath,
	}, nil
}
