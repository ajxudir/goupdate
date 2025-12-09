package formats

import (
	"encoding/json"
	"fmt"

	"github.com/ajxudir/goupdate/pkg/config"
)

// JSONParser parses JSON package files (e.g., package.json).
//
// It supports JSON-based package managers such as npm, Yarn, and Composer,
// extracting dependencies from configured field names.
type JSONParser struct{}

// Parse parses JSON content and extracts package dependencies.
//
// It performs the following operations:
//   - Unmarshals the JSON content into a map structure
//   - Iterates through configured fields (e.g., "dependencies", "devDependencies")
//   - Extracts package names and version strings from each field
//   - Applies version parsing, constraint mapping, and package overrides
//   - Filters ignored packages based on configuration
//
// Parameters:
//   - content: The raw bytes of the JSON package manifest file
//   - cfg: The package manager configuration with field mappings and rules
//
// Returns:
//   - []Package: A list of parsed packages with names, versions, and dependency types
//   - error: Returns an error if the JSON is invalid; returns nil on successful parse
func (p *JSONParser) Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	var packages []Package

	for field, pkgType := range cfg.Fields {
		deps, ok := data[field].(map[string]interface{})
		if !ok {
			continue
		}

		for name, version := range deps {
			versionStr, ok := version.(string)
			if !ok {
				continue
			}

			if shouldIgnorePackage(name, cfg) {
				continue
			}

			vInfo := processVersion(versionStr, name, cfg)
			packages = append(packages, newPackage(name, vInfo, pkgType, cfg))
		}
	}

	return packages, nil
}
