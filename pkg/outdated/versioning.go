package outdated

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/user/goupdate/pkg/config"
)

const (
	versionFormatSemver  = "semver"
	versionFormatNumeric = "numeric"
	versionFormatRegex   = "regex"
	versionFormatOrdered = "ordered"
)

var (
	defaultVersionRegex = regexp.MustCompile(`(?i)(?P<major>\d+)(?:[._-]?(?P<minor>\d+))?(?:[._-]?(?P<patch>\d+))?`)
	numericVersionRegex = regexp.MustCompile(`(?P<major>\d+)`)
)

// parsedVersion represents a parsed and normalized version string.
//
// Fields:
//   - raw: The original raw version string as provided
//   - canonical: The canonical semver representation (e.g., "v1.2.3")
//   - normalized: A normalized form for comparison (e.g., "1.2.3")
//   - major: The major version number extracted from the version
//   - minor: The minor version number extracted from the version
//   - patch: The patch version number extracted from the version
//   - hasNumbers: Whether numeric parts were successfully extracted
type parsedVersion struct {
	raw        string
	canonical  string
	normalized string
	major      int
	minor      int
	patch      int
	hasNumbers bool
}

// versioningStrategy represents the strategy for parsing and comparing versions.
//
// Fields:
//   - format: The version format (semver, numeric, regex, or ordered)
//   - regex: The compiled regex pattern for extracting version components
//   - sortDesc: Whether to sort versions in descending order (newest first)
type versioningStrategy struct {
	format   string
	regex    *regexp.Regexp
	sortDesc bool
}

// newVersioningStrategy creates a new versioning strategy from configuration.
//
// It performs the following operations:
//   - Determines the version format from config (defaults to semver)
//   - Configures sort direction (ascending or descending)
//   - Compiles the appropriate regex pattern for version extraction
//
// Parameters:
//   - cfg: Versioning configuration; if nil, uses semver format with descending sort
//
// Returns:
//   - versioningStrategy: Configured strategy for version parsing and comparison
//   - error: When format is unknown or regex compilation fails; returns nil on success
func newVersioningStrategy(cfg *config.VersioningCfg) (versioningStrategy, error) {
	format := versionFormatSemver
	sortDesc := true

	if cfg != nil {
		switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
		case "", versionFormatSemver:
			format = versionFormatSemver
		case versionFormatNumeric:
			format = versionFormatNumeric
		case versionFormatRegex:
			format = versionFormatRegex
		case versionFormatOrdered, "list", "sorted":
			format = versionFormatOrdered
		default:
			return versioningStrategy{}, fmt.Errorf("unknown version format: %s", cfg.Format)
		}

		if strings.ToLower(strings.TrimSpace(cfg.Sort)) == "asc" {
			sortDesc = false
		}
	}

	strategy := versioningStrategy{format: format, sortDesc: sortDesc}

	switch {
	case cfg != nil && strings.TrimSpace(cfg.Regex) != "":
		re, err := regexp.Compile(cfg.Regex)
		if err != nil {
			return versioningStrategy{}, fmt.Errorf("invalid version regex: %w", err)
		}
		strategy.regex = re
	case format == versionFormatNumeric:
		strategy.regex = numericVersionRegex
	default:
		strategy.regex = defaultVersionRegex
	}

	return strategy, nil
}

// parseVersion parses a version string using the strategy's format and returns a parsedVersion.
//
// It performs the following operations:
//   - Cleans and validates the input version string
//   - Attempts semver parsing for semver format
//   - Falls back to regex-based extraction for major/minor/patch
//   - Generates normalized form for comparison
//
// Parameters:
//   - version: The version string to parse (may include prefixes like "v")
//
// Returns:
//   - parsedVersion: The parsed version with extracted components
//   - bool: True if version was successfully parsed, false otherwise
func (s versioningStrategy) parseVersion(version string) (parsedVersion, bool) {
	cleaned := strings.TrimSpace(version)
	if cleaned == "" || cleaned == "#N/A" {
		return parsedVersion{}, false
	}

	pv := parsedVersion{raw: cleaned}

	if s.format == versionFormatOrdered {
		major, minor, patch, ok := s.extractParts(cleaned)
		if ok {
			pv.major = major
			pv.minor = minor
			pv.patch = patch
			pv.hasNumbers = true
			pv.normalized = fmt.Sprintf("%d.%d.%d", major, minor, patch)
			return pv, true
		}

		pv.normalized = s.normalizeLoose(cleaned)
		return pv, pv.normalized != ""
	}

	if s.format != versionFormatNumeric && s.format != versionFormatRegex {
		if canonical := canonicalSemver(cleaned); canonical != "" {
			pv.canonical = canonical
			pv.major, pv.minor, pv.patch = semverParts(canonical)
			pv.hasNumbers = true
			pv.normalized = fmt.Sprintf("%d.%d.%d", pv.major, pv.minor, pv.patch)
			return pv, true
		}
	}

	major, minor, patch, ok := s.extractParts(cleaned)
	if !ok {
		pv.normalized = s.normalizeLoose(cleaned)
		return pv, false
	}

	pv.major = major
	pv.minor = minor
	pv.patch = patch
	pv.hasNumbers = true
	pv.normalized = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	return pv, true
}

// compare compares two parsed versions and returns their ordering.
//
// It performs the following operations:
//   - Prefers semver comparison when both have canonical forms
//   - Falls back to numeric comparison (major, minor, patch) when available
//   - Uses string comparison of normalized forms as final fallback
//
// Parameters:
//   - a: The first version to compare
//   - b: The second version to compare
//
// Returns:
//   - int: Negative if a < b, zero if a == b, positive if a > b
func (s versioningStrategy) compare(a, b parsedVersion) int {
	if a.canonical != "" && b.canonical != "" {
		return semver.Compare(a.canonical, b.canonical)
	}

	if a.hasNumbers && b.hasNumbers {
		if a.major != b.major {
			return compareInts(a.major, b.major)
		}

		if a.minor != b.minor {
			return compareInts(a.minor, b.minor)
		}

		if a.patch != b.patch {
			return compareInts(a.patch, b.patch)
		}
	}

	return strings.Compare(a.normalized, b.normalized)
}

// sortComparable sorts a slice of parsed versions in place using the strategy's sort direction.
//
// Parameters:
//   - entries: Slice of parsed versions to sort (modified in place)
func (s versioningStrategy) sortComparable(entries []parsedVersion) {
	sort.SliceStable(entries, func(i, j int) bool {
		comparison := s.compare(entries[i], entries[j])
		if s.sortDesc {
			return comparison > 0
		}

		return comparison < 0
	})
}

// keyFor returns a unique key for deduplication based on the parsed version.
//
// Parameters:
//   - parsed: The parsed version structure
//   - raw: The original raw version string as fallback
//
// Returns:
//   - string: A normalized key for deduplication (canonical form, normalized form, or loose normalization)
func (s versioningStrategy) keyFor(parsed parsedVersion, raw string) string {
	// Prefer canonical form when available - it preserves prerelease identifiers
	// so that 1.0.0 and 1.0.0-rc03 are treated as different versions
	if parsed.canonical != "" {
		return parsed.canonical
	}

	// For non-semver versions (4+ segments, calver, etc.), use the loose normalized
	// form of the raw string to avoid incorrectly deduplicating versions like
	// 1.0.0.0 and 1.0.0.1 which both extract to normalized "1.0.0"
	return s.normalizeLoose(raw)
}

// extractParts extracts major, minor, and patch version components using regex.
//
// It performs the following operations:
//   - Applies the strategy's regex to find version components
//   - Uses named groups (major, minor, patch) or positional groups
//   - Selects the best match (most complete) from multiple matches
//
// Parameters:
//   - version: The version string to extract components from
//
// Returns:
//   - int: Major version number (0 if not found)
//   - int: Minor version number (0 if not found)
//   - int: Patch version number (0 if not found)
//   - bool: True if at least major version was found, false otherwise
func (s versioningStrategy) extractParts(version string) (int, int, int, bool) {
	// regex is always set by newVersioningStrategy
	matches := s.regex.FindAllStringSubmatch(version, -1)
	if len(matches) == 0 {
		return 0, 0, 0, false
	}

	// Select the best match: prefer matches with more captured groups (more complete version)
	// This handles cases like "1.0.0.0" where the regex might match both "1.0.0" and the trailing "0"
	var bestMatch []string
	bestScore := -1
	for _, match := range matches {
		score := 0
		// Count non-empty captured groups (excluding full match at index 0)
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				score++
			}
		}
		// Also prefer longer full matches to break ties
		if score > bestScore || (score == bestScore && len(match[0]) > len(bestMatch[0])) {
			bestMatch = match
			bestScore = score
		}
	}

	if bestMatch == nil {
		return 0, 0, 0, false
	}

	major, majorOK := parseNumericGroup(bestMatch, s.regex, "major", 1)
	if !majorOK {
		return 0, 0, 0, false
	}

	minor, _ := parseNumericGroup(bestMatch, s.regex, "minor", 2)
	patch, _ := parseNumericGroup(bestMatch, s.regex, "patch", 3)

	return major, minor, patch, true
}

// filterOrdered filters versions for ordered format by maintaining list position.
//
// It performs the following operations:
//   - Deduplicates versions using normalized keys
//   - Finds the position of the current version in the list
//   - Returns versions before (desc) or after (asc) the current position
//
// Parameters:
//   - current: The current version to use as baseline
//   - versions: List of available versions in their original order
//
// Returns:
//   - []string: Filtered versions based on position and sort direction
func (s versioningStrategy) filterOrdered(current string, versions []string) []string {
	// Use keyFor for consistent key generation with the version loop below
	parsed, _ := s.parseVersion(current)
	baseKey := s.keyFor(parsed, current)

	type orderedEntry struct {
		raw string
		key string
	}

	seen := make(map[string]struct{})
	entries := make([]orderedEntry, 0, len(versions))

	for _, version := range versions {
		cleaned := strings.TrimSpace(version)
		if cleaned == "" {
			continue
		}

		parsed, _ := s.parseVersion(cleaned)
		key := s.keyFor(parsed, cleaned)

		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		entries = append(entries, orderedEntry{raw: cleaned, key: key})
	}

	baseIndex := -1
	if baseKey != "" {
		for idx, entry := range entries {
			if entry.key == baseKey {
				baseIndex = idx
				break
			}
		}
	}

	filtered := make([]string, 0, len(entries))
	for idx, entry := range entries {
		if baseIndex == -1 {
			filtered = append(filtered, entry.raw)
			continue
		}

		if s.sortDesc {
			if idx < baseIndex {
				filtered = append(filtered, entry.raw)
			}
		} else if idx > baseIndex {
			filtered = append(filtered, entry.raw)
		}
	}

	return filtered
}

// normalizeLoose performs loose normalization of a version string for comparison.
//
// It performs the following operations:
//   - Trims whitespace
//   - Removes leading "v" prefix if followed by a digit
//   - Converts to lowercase
//
// Parameters:
//   - raw: The raw version string to normalize
//
// Returns:
//   - string: Normalized version string for comparison; empty string if input is empty
func (s versioningStrategy) normalizeLoose(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "v") && len(trimmed) > 1 && isDigit(rune(trimmed[1])) {
		trimmed = trimmed[1:]
	}

	return strings.ToLower(trimmed)
}

// parseNumericGroup extracts and parses a numeric group from a regex match.
//
// It performs the following operations:
//   - Attempts to find the value by named group first
//   - Falls back to positional index if named group not found
//   - Parses the matched string as an integer
//
// Parameters:
//   - match: The regex match result array
//   - re: The compiled regex with potential named groups
//   - name: The name of the group to extract (e.g., "major", "minor", "patch")
//   - index: The fallback positional index if named group not found
//
// Returns:
//   - int: The parsed integer value (0 if parsing fails)
//   - bool: True if the value was found and parsed successfully, false otherwise
func parseNumericGroup(match []string, re *regexp.Regexp, name string, index int) (int, bool) {
	value := ""

	if idx := re.SubexpIndex(name); idx >= 0 && idx < len(match) {
		value = match[idx]
	} else if index < len(match) {
		value = match[index]
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return parsed, true
}

// semverParts extracts major, minor, and patch components from a semver string.
//
// It performs the following operations:
//   - Removes "v" prefix if present
//   - Splits on dots to get version parts
//   - Parses each part as an integer
//
// Parameters:
//   - version: The semver version string (e.g., "v1.2.3" or "1.2.3")
//
// Returns:
//   - int: Major version number (0 if not present or invalid)
//   - int: Minor version number (0 if not present or invalid)
//   - int: Patch version number (0 if not present or invalid)
func semverParts(version string) (int, int, int) {
	trimmed := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(trimmed, ".", 3)

	major := parsePart(parts, 0)
	minor := parsePart(parts, 1)
	patch := parsePart(parts, 2)

	return major, minor, patch
}

// parsePart parses a single version part from a split version string.
//
// Parameters:
//   - parts: Array of version parts split by delimiter
//   - index: The index of the part to parse
//
// Returns:
//   - int: The parsed integer value, or 0 if index out of bounds or parsing fails
func parsePart(parts []string, index int) int {
	if index >= len(parts) {
		return 0
	}

	value, err := strconv.Atoi(parts[index])
	if err != nil {
		return 0
	}

	return value
}

// compareInts compares two integers and returns their ordering.
//
// Parameters:
//   - a: The first integer to compare
//   - b: The second integer to compare
//
// Returns:
//   - int: 1 if a > b, -1 if a < b, 0 if a == b
func compareInts(a, b int) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

// isDigit checks if a rune is a numeric digit (0-9).
//
// Parameters:
//   - r: The rune to check
//
// Returns:
//   - bool: True if the rune is a digit between '0' and '9', false otherwise
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// canonicalSemver converts a version string to canonical semver format.
//
// It performs the following operations:
//   - Cleans and validates the input
//   - Adds "v" prefix if missing
//   - Pads missing minor/patch with zeros until valid semver is found
//   - Returns canonical form using semver.Canonical
//
// Parameters:
//   - version: The version string to canonicalize (e.g., "1.2", "v1.2.3")
//
// Returns:
//   - string: Canonical semver string (e.g., "v1.2.0"); empty string if not valid semver
func canonicalSemver(version string) string {
	cleaned := strings.TrimSpace(version)
	if cleaned == "" || cleaned == "#N/A" {
		return ""
	}

	if !strings.HasPrefix(cleaned, "v") {
		cleaned = "v" + cleaned
	}

	trimmed := strings.TrimPrefix(cleaned, "v")
	parts := strings.Split(trimmed, ".")
	for len(parts) > 0 && len(parts) < 3 {
		candidate := "v" + strings.Join(parts, ".")
		if semver.IsValid(candidate) {
			return semver.Canonical(candidate)
		}
		parts = append(parts, "0")
	}

	if semver.IsValid(cleaned) {
		return semver.Canonical(cleaned)
	}

	return ""
}
