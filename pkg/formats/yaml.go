package formats

import (
	"fmt"

	"github.com/ajxudir/goupdate/pkg/config"
	"gopkg.in/yaml.v3"
)

// YAMLParser parses YAML package files.
//
// It supports YAML-based package managers such as Composer, Conda, and Docker Compose,
// handling both map-based and array-based dependency structures with nested field access.
type YAMLParser struct{}

// Parse parses YAML content and extracts package dependencies.
//
// It performs the following operations:
//   - Unmarshals the YAML content into a nested map structure
//   - Retrieves dependency fields using dot notation (e.g., "dependencies.production")
//   - Handles both map-based dependencies (name: version) and array-based dependencies
//   - Extracts container image specifications for Docker Compose files
//   - Applies version parsing, constraint mapping, and package overrides
//   - Filters ignored packages based on configuration
//
// Parameters:
//   - content: The raw bytes of the YAML package manifest file
//   - cfg: The package manager configuration with field mappings and extraction rules
//
// Returns:
//   - []Package: A list of parsed packages with names, versions, and dependency types
//   - error: Returns an error if the YAML is invalid; returns nil on successful parse
func (p *YAMLParser) Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	var packages []Package

	for field, pkgType := range cfg.Fields {
		fieldValue := GetNestedField(data, field)
		if fieldValue == nil {
			continue
		}

		switch deps := fieldValue.(type) {
		case map[string]interface{}:
			for name, version := range deps {
				versionStr := ""
				switch v := version.(type) {
				case string:
					versionStr = v
				case map[string]interface{}, map[interface{}]interface{}:
					// handled below
				default:
					versionStr = fmt.Sprintf("%v", v)
				}
				resolvedName := name

				switch depMap := version.(type) {
				case map[string]interface{}:
					versionStr, resolvedName = parseImageFromMap(depMap, resolvedName, versionStr, cfg)
				case map[interface{}]interface{}:
					normalized := make(map[string]interface{})
					for k, v := range depMap {
						if key, ok := k.(string); ok {
							normalized[key] = v
						}
					}
					versionStr, resolvedName = parseImageFromMap(normalized, resolvedName, versionStr, cfg)
				}

				vInfo := processVersion(versionStr, resolvedName, cfg)
				pkg := newPackage(resolvedName, vInfo, pkgType, cfg)

				// Check if package should be ignored and set reason
				if reason := getIgnoreReason(resolvedName, cfg); reason != "" {
					pkg.IgnoreReason = reason
				}

				packages = append(packages, pkg)
			}

		case []interface{}:
			for _, dep := range deps {
				depMap, ok := dep.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := depMap["name"].(string)
				version, _ := depMap["version"].(string)

				if name == "" || version == "" {
					continue
				}

				vInfo := processVersion(version, name, cfg)
				pkg := newPackage(name, vInfo, pkgType, cfg)

				// Check if package should be ignored and set reason
				if reason := getIgnoreReason(name, cfg); reason != "" {
					pkg.IgnoreReason = reason
				}

				packages = append(packages, pkg)
			}
		}
	}

	return packages, nil
}
