package display

import (
	"strconv"
	"strings"

	"github.com/ajxudir/goupdate/pkg/constants"
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
// after updating to the target version. Only shows versions that are higher
// than the target to avoid noise (e.g., won't show "patch: 7.26.10" when
// target is already "7.28.5").
//
// Parameters:
//   - target: The version that was updated to
//   - major: Available major version
//   - minor: Available minor version
//   - patch: Available patch version
//
// Returns:
//   - string: Formatted string like "(major: 2.0.0 available)" or empty if none
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

	// Only show versions that are valid AND higher than the target
	hasMajor := isValidAndHigher(major, target)
	hasMinor := isValidAndHigher(minor, target)
	hasPatch := isValidAndHigher(patch, target)

	if !hasMajor && !hasMinor && !hasPatch {
		return ""
	}

	// Order: patch → minor → major (minor is most common, so comes first for alignment)
	var parts []string
	if hasPatch {
		parts = append(parts, "patch: "+patch)
	}
	if hasMinor {
		parts = append(parts, "minor: "+minor)
	}
	if hasMajor {
		parts = append(parts, "major: "+major)
	}

	return "(" + strings.Join(parts, ", ") + " available)"
}

// isValidAndHigher checks if a version is valid (non-empty, non-placeholder)
// and higher than the target version.
func isValidAndHigher(version, target string) bool {
	if version == "" || version == constants.PlaceholderNA {
		return false
	}
	if version == target {
		return false
	}
	// Compare versions - only show if available version is higher than target
	return compareVersions(version, target) > 0
}

// compareVersions compares two version strings.
// Returns: positive if v1 > v2, negative if v1 < v2, zero if equal.
// Uses simple numeric comparison of version parts.
func compareVersions(v1, v2 string) int {
	// Strip leading 'v' if present
	v1 = strings.TrimPrefix(strings.TrimSpace(v1), "v")
	v2 = strings.TrimPrefix(strings.TrimSpace(v2), "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(extractNumeric(parts1[i]))
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(extractNumeric(parts2[i]))
		}
		if n1 != n2 {
			return n1 - n2
		}
	}
	return 0
}

// extractNumeric extracts the leading numeric portion of a string.
// E.g., "10-rc1" -> "10", "5" -> "5"
func extractNumeric(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else {
			break
		}
	}
	return result.String()
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

	// Order: patch → minor → major (minor is most common, so comes first for alignment)
	var parts []string
	if hasPatch {
		parts = append(parts, "patch: "+patch)
	}
	if hasMinor {
		parts = append(parts, "minor: "+minor)
	}
	if hasMajor {
		parts = append(parts, "major: "+major)
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
