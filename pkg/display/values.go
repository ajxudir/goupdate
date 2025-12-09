package display

import (
	"strings"

	"github.com/user/goupdate/pkg/constants"
)

// SafeInstalledValue returns a display-safe installed version.
//
// If the value is empty or whitespace-only, returns "#N/A" for consistent display.
// Otherwise returns the trimmed value.
//
// Parameters:
//   - val: The installed version string, may be empty
//
// Returns:
//   - string: The value or "#N/A" if empty
//
// Example:
//
//	display.SafeInstalledValue("")      // Returns "#N/A"
//	display.SafeInstalledValue("1.2.3") // Returns "1.2.3"
func SafeInstalledValue(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return constants.PlaceholderNA
	}
	return val
}

// SafeDeclaredValue returns a display-safe declared version.
//
// If the value is empty, whitespace-only, or "#N/A", returns "*" for consistent display.
// Otherwise returns the trimmed value.
//
// Parameters:
//   - val: The declared version string, may be empty
//
// Returns:
//   - string: The value or "*" if empty/placeholder
//
// Example:
//
//	display.SafeDeclaredValue("")      // Returns "*"
//	display.SafeDeclaredValue("#N/A")  // Returns "*"
//	display.SafeDeclaredValue("1.2.3") // Returns "1.2.3"
func SafeDeclaredValue(val string) string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" || strings.EqualFold(trimmed, constants.PlaceholderNA) {
		return constants.PlaceholderWildcard
	}
	return trimmed
}

// SafeVersionValue returns a display-safe version string.
//
// If the value is empty or whitespace-only, returns the provided placeholder.
//
// Parameters:
//   - val: The version string, may be empty
//   - placeholder: The placeholder to use if val is empty
//
// Returns:
//   - string: The trimmed value or placeholder if empty
//
// Example:
//
//	display.SafeVersionValue("", "#N/A")   // Returns "#N/A"
//	display.SafeVersionValue("1.2.3", "-") // Returns "1.2.3"
func SafeVersionValue(val, placeholder string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return placeholder
	}
	return val
}

// HasAvailableUpdates returns true if any version update is available.
//
// Checks if major, minor, or patch versions are non-empty and not placeholders.
//
// Parameters:
//   - major: Major version string
//   - minor: Minor version string
//   - patch: Patch version string
//
// Returns:
//   - bool: true if at least one version is available
//
// Example:
//
//	display.HasAvailableUpdates("2.0.0", "", "")      // Returns true
//	display.HasAvailableUpdates("#N/A", "#N/A", "")   // Returns false
func HasAvailableUpdates(major, minor, patch string) bool {
	isValid := func(v string) bool {
		v = strings.TrimSpace(v)
		return v != "" && v != constants.PlaceholderNA
	}
	return isValid(major) || isValid(minor) || isValid(patch)
}

// FormatAvailableVersions formats available versions for display after an update.
//
// Returns a formatted string showing which versions are still available
// after updating to the target version.
//
// Parameters:
//   - target: The version that was updated to
//   - major: Available major version
//   - minor: Available minor version
//   - patch: Available patch version
//
// Returns:
//   - string: Formatted string like "(major: 2.0.0, minor: 1.5.0 available)" or empty if none
//
// Example:
//
//	display.FormatAvailableVersions("1.2.3", "2.0.0", "", "")
//	// Returns "(major: 2.0.0 available)"
func FormatAvailableVersions(target, major, minor, patch string) string {
	target = strings.TrimSpace(target)
	major = strings.TrimSpace(major)
	minor = strings.TrimSpace(minor)
	patch = strings.TrimSpace(patch)

	hasMajor := major != "" && major != constants.PlaceholderNA && major != target
	hasMinor := minor != "" && minor != constants.PlaceholderNA && minor != target
	hasPatch := patch != "" && patch != constants.PlaceholderNA && patch != target

	if !hasMajor && !hasMinor && !hasPatch {
		return ""
	}

	var parts []string
	if hasMajor {
		parts = append(parts, "major: "+major)
	}
	if hasMinor {
		parts = append(parts, "minor: "+minor)
	}
	if hasPatch {
		parts = append(parts, "patch: "+patch)
	}

	return "(" + strings.Join(parts, ", ") + " available)"
}

// FormatAvailableVersionsUpToDate formats available versions for packages that are up to date.
//
// Similar to FormatAvailableVersions but doesn't exclude versions matching a target,
// since up-to-date packages have no target version.
//
// Parameters:
//   - major: Available major version
//   - minor: Available minor version
//   - patch: Available patch version
//
// Returns:
//   - string: Formatted string like "(major: 2.0.0, minor: 1.5.0 available)" or empty if none
//
// Example:
//
//	display.FormatAvailableVersionsUpToDate("2.0.0", "1.5.0", "")
//	// Returns "(major: 2.0.0, minor: 1.5.0 available)"
func FormatAvailableVersionsUpToDate(major, minor, patch string) string {
	major = strings.TrimSpace(major)
	minor = strings.TrimSpace(minor)
	patch = strings.TrimSpace(patch)

	hasMajor := major != "" && major != constants.PlaceholderNA
	hasMinor := minor != "" && minor != constants.PlaceholderNA
	hasPatch := patch != "" && patch != constants.PlaceholderNA

	if !hasMajor && !hasMinor && !hasPatch {
		return ""
	}

	var parts []string
	if hasMajor {
		parts = append(parts, "major: "+major)
	}
	if hasMinor {
		parts = append(parts, "minor: "+minor)
	}
	if hasPatch {
		parts = append(parts, "patch: "+patch)
	}

	return "(" + strings.Join(parts, ", ") + " available)"
}

// TruncateWithEllipsis truncates a string and adds "..." if too long.
//
// If the string is shorter than or equal to maxLen, returns unchanged.
// Otherwise truncates and appends "..." (total length = maxLen).
//
// Parameters:
//   - s: The string to truncate
//   - maxLen: Maximum length including ellipsis (minimum 4)
//
// Returns:
//   - string: Original string if shorter than maxLen, or truncated with "..."
//
// Example:
//
//	display.TruncateWithEllipsis("example.com/very/long/package", 20)
//	// Returns "example.com/very/..."
func TruncateWithEllipsis(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatVersion formats a version string for display.
//
// Currently returns the trimmed version as-is to maintain consistency with existing behavior.
// Future versions may remove leading "v" prefix.
//
// Parameters:
//   - version: The version string
//
// Returns:
//   - string: Formatted version (currently just trimmed)
//
// Example:
//
//	display.FormatVersion("1.2.3")  // Returns "1.2.3"
//	display.FormatVersion(" v1.2.3 ") // Returns "v1.2.3"
func FormatVersion(version string) string {
	version = strings.TrimSpace(version)
	// Optionally strip leading 'v' for consistency
	// This is a common pattern but may not be universally desired
	// For now, return as-is to maintain consistency with existing behavior
	return version
}

// IsValidVersion returns true if the version string is a valid, non-placeholder value.
//
// Parameters:
//   - version: The version string to check
//
// Returns:
//   - bool: true if version is non-empty and not a placeholder
//
// Example:
//
//	display.IsValidVersion("1.2.3")  // Returns true
//	display.IsValidVersion("#N/A")   // Returns false
//	display.IsValidVersion("")       // Returns false
//	display.IsValidVersion("*")      // Returns false
func IsValidVersion(version string) bool {
	v := strings.TrimSpace(version)
	return v != "" && v != constants.PlaceholderNA && v != constants.PlaceholderWildcard
}
