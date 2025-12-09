package formats

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/utils"
)

// shouldIgnorePackage checks if a package should be ignored based on configuration.
//
// It performs the following checks:
//   - Matches the package name against ignore patterns using regex
//   - Checks if the package has an ignore flag in package-specific overrides
//
// Parameters:
//   - name: The package name to check
//   - cfg: The package manager configuration containing ignore patterns and overrides
//
// Returns:
//   - bool: Returns true if the package should be ignored; false otherwise
func shouldIgnorePackage(name string, cfg *config.PackageManagerCfg) bool {
	if cfg == nil {
		return false
	}

	for _, ignored := range cfg.Ignore {
		if matched, _ := regexp.MatchString(ignored, name); matched {
			return true
		}
	}

	if override, exists := cfg.PackageOverrides[name]; exists && override.Ignore {
		return true
	}

	return false
}

// GetNestedField retrieves a value from a nested map using dot notation (e.g., "foo.bar.baz").
// Returns nil if the path doesn't exist or if an intermediate value is not a map.
func GetNestedField(data map[string]interface{}, field string) interface{} {
	parts := strings.Split(field, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case map[interface{}]interface{}:
			current = v[part]
		default:
			return nil
		}

		if current == nil {
			return nil
		}
	}

	return current
}

// extractSection extracts lines from a specific section in an INI-style text.
//
// It scans the text for section headers in brackets (e.g., [section-name]) and
// returns all lines that belong to the specified section until the next section
// header is encountered.
//
// Parameters:
//   - text: The full text content to scan
//   - section: The section name to extract (without brackets)
//
// Returns:
//   - string: The lines belonging to the section, joined by newlines
func extractSection(text, section string) string {
	scanner := bufio.NewScanner(strings.NewReader(text))
	var lines []string
	currentSection := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = strings.Trim(trimmed, "[]")
			continue
		}

		if currentSection == section {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// processVersion parses a version string and applies all standard transformations.
//
// It performs the following operations:
//   - Parses the raw version string to extract version and constraint
//   - Maps the constraint to standardized format using constraint mapping
//   - Applies package-specific overrides from configuration
//   - Normalizes the version format according to the package manager rules
//
// Parameters:
//   - versionStr: The raw version string from the manifest file (e.g., "^1.2.3", ">=2.0.0")
//   - pkgName: The package name for applying package-specific overrides
//   - cfg: The package manager configuration with mappings and rules
//
// Returns:
//   - utils.VersionInfo: A struct containing the parsed and normalized version and constraint
func processVersion(versionStr, pkgName string, cfg *config.PackageManagerCfg) utils.VersionInfo {
	vInfo := utils.ParseVersion(versionStr)

	if cfg.ConstraintMapping != nil {
		vInfo.Constraint = utils.MapConstraint(vInfo.Constraint, cfg.ConstraintMapping)
	}

	vInfo = utils.ApplyPackageOverride(pkgName, vInfo, cfg)
	vInfo = utils.NormalizeDeclaredVersion(pkgName, vInfo, cfg)

	return vInfo
}

// newPackage creates a Package struct from version info and configuration.
//
// Parameters:
//   - name: The package name
//   - vInfo: The parsed version information containing version and constraint
//   - pkgType: The dependency type ("prod" or "dev")
//   - cfg: The package manager configuration for setting the PackageType field
//
// Returns:
//   - Package: A fully populated Package struct ready for use
func newPackage(name string, vInfo utils.VersionInfo, pkgType string, cfg *config.PackageManagerCfg) Package {
	return Package{
		Name:        name,
		Version:     vInfo.Version,
		Constraint:  vInfo.Constraint,
		Type:        pkgType,
		PackageType: cfg.Manager,
	}
}

// parseImageFromMap extracts image name and version from a map structure.
//
// This function is used to parse container image specifications from YAML/JSON
// dependencies, particularly for Docker Compose and similar formats. It supports
// extraction using regex patterns or by splitting image strings.
//
// It performs the following operations:
//   - Looks for an "image" key in the dependency map
//   - Applies regex pattern extraction if configured
//   - Falls back to splitting the image string by colon (image:tag format)
//
// Parameters:
//   - depMap: The dependency map containing the image specification
//   - resolvedName: The initial package name (may be updated if extracted from image)
//   - versionStr: The initial version string (may be updated)
//   - cfg: The package manager configuration with extraction patterns
//
// Returns:
//   - string: The extracted or parsed version/tag
//   - string: The resolved image/package name
func parseImageFromMap(depMap map[string]interface{}, resolvedName, versionStr string, cfg *config.PackageManagerCfg) (string, string) {
	imageVal, ok := depMap["image"]
	if !ok {
		return versionStr, resolvedName
	}

	image, ok := imageVal.(string)
	if !ok {
		return "", resolvedName
	}

	text := fmt.Sprintf("image: %s", image)
	if cfg.Extraction != nil && cfg.Extraction.Pattern != "" {
		if matches, _ := utils.ExtractAllMatches(cfg.Extraction.Pattern, text); len(matches) > 0 {
			if matches[0]["name"] != "" {
				resolvedName = matches[0]["name"]
			}
			versionStr = matches[0]["version"]
		}
	}

	if versionStr == "" {
		parts := strings.Split(image, ":")
		resolvedName = parts[0]
		if len(parts) > 1 {
			versionStr = parts[1]
		}
	}

	return versionStr, resolvedName
}
