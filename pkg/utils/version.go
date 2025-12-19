package utils

import (
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/ajxudir/goupdate/pkg/warnings"
)

var supportedConstraints = map[string]bool{
	"":   true,
	"^":  true,
	"~":  true,
	">=": true,
	"<=": true,
	">":  true,
	"<":  true,
	"=":  true,
	"*":  true,
}

// ValidateConstraint checks if a constraint is valid and returns it or defaults to exact with warning.
//
// It validates that the provided constraint is one of the supported constraint types.
// If the constraint is invalid, a warning is issued and an empty string (exact match) is returned.
//
// Parameters:
//   - constraint: The constraint operator to validate (e.g., "^", "~", ">=", "<=", ">", "<", "=", "*", or "")
//   - packageName: The name of the package being validated (used in warning messages)
//
// Returns:
//   - string: The validated constraint if valid, or "" (exact match) if invalid
func ValidateConstraint(constraint, packageName string) string {
	verbose.Printf("Constraint validation: checking %q for package %q\n", constraint, packageName)
	if supportedConstraints[constraint] {
		verbose.Printf("Constraint validation: %q is valid\n", constraint)
		return constraint
	}

	verbose.Printf("Constraint validation ERROR: %q is invalid, using exact match\n", constraint)
	warnings.Warnf("Invalid constraint '%s' for package '%s', using exact match\n", constraint, packageName)
	return ""
}

// NormalizeDeclaredVersion standardizes version strings that represent latest or missing versions.
//
// It performs the following operations:
//   - Step 1: Builds latest version mappings from configuration
//   - Step 2: Normalizes version strings like "", "#n/a", "latest" to configured latest value
//   - Step 3: Respects per-package and per-manager latest mappings
//
// Parameters:
//   - name: The package name for looking up package-specific mappings
//   - vInfo: The VersionInfo containing constraint and version to normalize
//   - cfg: The package manager configuration containing latest mappings (can be nil)
//
// Returns:
//   - VersionInfo: A new VersionInfo with normalized version string
func NormalizeDeclaredVersion(name string, vInfo VersionInfo, cfg *config.PackageManagerCfg) VersionInfo {
	mappings, _ := buildLatestMappings(name, cfg)

	versionKey := strings.ToLower(strings.TrimSpace(vInfo.Version))
	if mapped, ok := mappings[versionKey]; ok {
		vInfo.Version = mapped
	}

	return vInfo
}

// buildLatestMappings constructs a map of version strings that should be treated as "latest".
//
// It performs the following operations:
//   - Step 1: Starts with default mappings for "", "#n/a", "latest" -> "*"
//   - Step 2: Applies default latest mappings from configuration if available
//   - Step 3: Applies package-specific latest mappings if available
//   - Step 4: Returns both the mapping table and the resolved latest value
//
// Parameters:
//   - name: The package name for looking up package-specific mappings
//   - cfg: The package manager configuration containing latest mappings (can be nil)
//
// Returns:
//   - map[string]string: A map from version strings to their "latest" equivalents
//   - string: The resolved latest value (default "*")
func buildLatestMappings(name string, cfg *config.PackageManagerCfg) (map[string]string, string) {
	latestValue := "*"
	mappings := map[string]string{
		"":       latestValue,
		"#n/a":   latestValue,
		"latest": latestValue,
	}

	applyMappings := func(source map[string]string) {
		for key, value := range source {
			normalizedKey := strings.ToLower(strings.TrimSpace(key))
			mappings[normalizedKey] = value
			if normalizedKey == "" {
				latestValue = value
			}
		}
	}

	if cfg != nil && cfg.LatestMapping != nil {
		if cfg.LatestMapping.Default != nil {
			applyMappings(cfg.LatestMapping.Default)
		}

		if pkgMappings, ok := cfg.LatestMapping.Packages[name]; ok {
			applyMappings(pkgMappings)
		}
	}

	mappings[""] = latestValue
	mappings["#n/a"] = latestValue
	mappings["latest"] = latestValue

	return mappings, latestValue
}

// IsLatestIndicator checks if a version string indicates "latest".
//
// It compares the provided version string against the configured latest value
// for the package manager or specific package. Case-insensitive comparison.
//
// Parameters:
//   - version: The version string to check (e.g., "*", "latest", custom latest indicators)
//   - name: The package name for looking up package-specific latest indicators
//   - cfg: The package manager configuration containing latest mappings (can be nil)
//
// Returns:
//   - bool: true if the version string represents "latest", false otherwise
func IsLatestIndicator(version, name string, cfg *config.PackageManagerCfg) bool {
	_, latestValue := buildLatestMappings(name, cfg)
	return strings.EqualFold(strings.TrimSpace(version), strings.TrimSpace(latestValue))
}

// IsFloatingConstraint checks if a version is a "floating" constraint that shouldn't
// be modified in the manifest. Floating constraints include:
// - Wildcards: "5.*", "5.4.*", "1.x"
// - Ranges: "[8.0.0,9.0.0)", "(1.0,2.0]"
// - Compound constraints: ">=1.0.0 <2.0.0", ">=3.0,<4.0"
// - Pure wildcards: "*" (already latest)
// These constraints express user intent to float within a range, so updating
// the manifest would destroy that intent.
func IsFloatingConstraint(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}

	// Pure wildcard "*" is floating (means "latest")
	if version == "*" {
		verbose.Tracef("Floating check: %q is pure wildcard (floating)", version)
		return true
	}

	// Wildcards embedded in version: "5.*", "5.4.*", "8.x", "1.x.x"
	if strings.Contains(version, ".*") || strings.Contains(version, ".x") {
		verbose.Tracef("Floating check: %q contains embedded wildcard (floating)", version)
		return true
	}

	// Trailing wildcard without dot: "5*" (rare but possible)
	if strings.HasSuffix(version, "*") && version != "*" {
		verbose.Tracef("Floating check: %q has trailing wildcard (floating)", version)
		return true
	}

	// NuGet/MSBuild ranges: "[8.0.0,9.0.0)", "(1.0,2.0]", etc.
	if strings.HasPrefix(version, "[") || strings.HasPrefix(version, "(") {
		verbose.Tracef("Floating check: %q is version range (floating)", version)
		return true
	}

	// Compound constraints with multiple operators: ">=1.0.0 <2.0.0", ">=3.0,<4.0"
	// These have both lower and upper bounds expressed
	hasMin := strings.Contains(version, ">=") || strings.Contains(version, ">")
	hasMax := strings.Contains(version, "<=") || strings.Contains(version, "<")
	if hasMin && hasMax {
		verbose.Tracef("Floating check: %q is compound constraint (floating)", version)
		return true
	}

	// OR constraints: "^2.0|^3.0", ">=1.0 || <0.5"
	if strings.Contains(version, "|") {
		verbose.Tracef("Floating check: %q contains OR constraint (floating)", version)
		return true
	}

	return false
}

// ApplyPackageOverride applies package-specific overrides to version info.
//
// It performs the following operations:
//   - Step 1: Checks if package overrides are configured
//   - Step 2: Looks up override configuration for the specific package
//   - Step 3: Applies constraint override if explicitly set (validates it)
//   - Step 4: Applies version override if provided
//
// Parameters:
//   - name: The package name to look up overrides for
//   - vInfo: The VersionInfo to potentially override
//   - cfg: The package manager configuration containing package overrides (can be nil)
//
// Returns:
//   - VersionInfo: The VersionInfo with overrides applied, or original if no overrides exist
func ApplyPackageOverride(name string, vInfo VersionInfo, cfg *config.PackageManagerCfg) VersionInfo {
	if cfg == nil || cfg.PackageOverrides == nil {
		return vInfo
	}

	override, exists := cfg.PackageOverrides[name]
	if !exists {
		return vInfo
	}

	// Apply constraint override if explicitly set (even if empty string)
	if override.Constraint != nil {
		vInfo.Constraint = ValidateConstraint(*override.Constraint, name)
	}

	// Apply version override
	if override.Version != "" {
		vInfo.Version = override.Version
	}

	return vInfo
}
