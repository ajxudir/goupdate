package formats

import (
	"fmt"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/utils"
)

// RawParser parses raw text files using regex patterns (e.g., requirements.txt).
//
// It supports plain text package managers such as pip (requirements.txt) and Go modules,
// using configurable regex patterns to extract package names, versions, and constraints.
type RawParser struct{}

// Parse parses raw text content and extracts package dependencies using configured patterns.
//
// It performs the following operations:
//   - Converts the raw bytes to text
//   - Extracts sections from INI-style files if multiple fields are configured
//   - Applies regex patterns to match package declarations
//   - Extracts package name, version, and constraint from regex named groups
//   - Applies constraint mapping and package overrides
//   - Filters ignored packages based on configuration
//
// The regex pattern should use named groups: "name", "version", and optionally "constraint".
// Alternative group names "n" and "version_alt" are also supported.
//
// Parameters:
//   - content: The raw bytes of the text package manifest file
//   - cfg: The package manager configuration with extraction patterns and field mappings
//
// Returns:
//   - []Package: A list of parsed packages with names, versions, and constraints
//   - error: Returns an error if the regex pattern is invalid; returns nil on successful parse
func (p *RawParser) Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error) {
	text := string(content)
	var packages []Package

	for fieldName, pkgType := range cfg.Fields {
		var pattern string
		if cfg.Extraction != nil && cfg.Extraction.Pattern != "" {
			pattern = cfg.Extraction.Pattern
		} else {
			// Return empty if no pattern configured
			return packages, nil
		}

		sectionText := text
		if len(cfg.Fields) > 1 {
			sectionText = extractSection(text, fieldName)
			if sectionText == "" {
				continue
			}
		}

		matches, err := utils.ExtractAllMatches(pattern, sectionText)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}

		for _, match := range matches {
			name := match["name"]
			if name == "" {
				name = match["n"]
			}
			constraint := match["constraint"]
			version := match["version"]
			if version == "" {
				version = match["version_alt"]
			}

			if name == "" || shouldIgnorePackage(name, cfg) {
				continue
			}

			vInfo := utils.VersionInfo{
				Constraint: constraint,
				Version:    version,
			}

			if cfg.ConstraintMapping != nil {
				vInfo.Constraint = utils.MapConstraint(vInfo.Constraint, cfg.ConstraintMapping)
			}

			// Apply package-specific overrides
			vInfo = utils.ApplyPackageOverride(name, vInfo, cfg)

			vInfo = utils.NormalizeDeclaredVersion(name, vInfo, cfg)

			packages = append(packages, Package{
				Name:        name,
				Version:     vInfo.Version,
				Constraint:  vInfo.Constraint,
				Type:        pkgType,
				PackageType: cfg.Manager,
			})
		}
	}

	return packages, nil
}
