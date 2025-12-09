package outdated

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/goupdate/pkg/config"
)

// TestFilterNewerVersionsSemver tests the behavior of FilterNewerVersions with semver format.
//
// It verifies:
//   - Filters versions newer than current using semver comparison
//   - Handles v-prefix correctly
//   - Removes duplicates
func TestFilterNewerVersionsSemver(t *testing.T) {
	versions, err := FilterNewerVersions("1.0.0", []string{"v1.0.0", "1.1.0", "v2.0.0", "v2.0.0"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"v2.0.0", "1.1.0"}, versions)
}

// TestFilterNewerVersionsDockerStyleTags tests the behavior of FilterNewerVersions with Docker-style tags.
//
// It verifies:
//   - Filters versions with prefixes like "alpine-"
func TestFilterNewerVersionsDockerStyleTags(t *testing.T) {
	versions, err := FilterNewerVersions("alpine-v1", []string{"alpine-v1", "alpine-1", "alpine-1.1", "alpine-2"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpine-2", "alpine-1.1"}, versions)
}

// TestFilterNewerVersionsNumeric tests the behavior of FilterNewerVersions with numeric format.
//
// It verifies:
//   - Numeric version comparison for date-based versions
func TestFilterNewerVersionsNumeric(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "numeric"}
	versions, err := FilterNewerVersions("20240101", []string{"20231212", "20240202", "20240101"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"20240202"}, versions)
}

// TestFilterNewerVersionsRegex tests the behavior of FilterNewerVersions with custom regex format.
//
// It verifies:
//   - Custom regex pattern extracts version components correctly
func TestFilterNewerVersionsRegex(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "regex", Regex: `alpine[-v]?(?P<major>\d+)(?:[\.-](?P<minor>\d+))?`}
	versions, err := FilterNewerVersions("alpine-15.4", []string{"alpine-15-4", "alpine-15.5", "alpine-16"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpine-16", "alpine-15.5"}, versions)
}

// TestFilterNewerVersionsOrdered tests the behavior of FilterNewerVersions with ordered format.
//
// It verifies:
//   - Ordered list filtering with default descending order
func TestFilterNewerVersionsOrdered(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "ordered"}
	versions, err := FilterNewerVersions("git-hash-b", []string{"git-hash-c", "git-hash-b", "git-hash-a"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"git-hash-c"}, versions)
}

// TestFilterNewerVersionsOrderedAscending tests the behavior of FilterNewerVersions with ordered ascending format.
//
// It verifies:
//   - Ordered list filtering with explicit ascending sort order
func TestFilterNewerVersionsOrderedAscending(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "ordered", Sort: "asc"}
	versions, err := FilterNewerVersions("build-2", []string{"build-1", "build-2", "build-3"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"build-3"}, versions)
}

// TestFilterNewerVersionsPassthroughWhenBaseUnknown tests the behavior when base version cannot be parsed.
//
// It verifies:
//   - Returns all versions sorted when base version is invalid
func TestFilterNewerVersionsPassthroughWhenBaseUnknown(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "numeric"}
	versions, err := FilterNewerVersions("not-a-number", []string{"zeta", "alpha", "beta"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "zeta"}, versions)
}

// TestFilterNewerVersionsNumericAscending tests the behavior of FilterNewerVersions with numeric ascending sort.
//
// It verifies:
//   - Numeric version filtering with ascending sort order
func TestFilterNewerVersionsNumericAscending(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "numeric", Sort: "asc"}
	versions, err := FilterNewerVersions("2", []string{"1", "3", "4"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"3", "4"}, versions)
}

// TestFilterNewerVersionsOrderedWithoutBase tests the behavior of FilterNewerVersions with ordered format and empty base.
//
// It verifies:
//   - Returns all versions when base is empty with ordered format
//   - Deduplicates versions
func TestFilterNewerVersionsOrderedWithoutBase(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "ordered"}
	versions, err := FilterNewerVersions("", []string{"c", "b", "a", "b"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, versions)
}

// TestNewVersioningStrategyErrors tests error cases for newVersioningStrategy.
//
// It verifies:
//   - Returns error for unsupported format
//   - Returns error for invalid regex pattern
func TestNewVersioningStrategyErrors(t *testing.T) {
	_, err := newVersioningStrategy(&config.VersioningCfg{Format: "custom"})
	assert.Error(t, err)

	_, err = newVersioningStrategy(&config.VersioningCfg{Format: "regex", Regex: "[invalid"})
	assert.Error(t, err)
}

// TestNewVersioningStrategyAliases tests the behavior of format aliases.
//
// It verifies:
//   - "sorted" alias maps to ordered format
func TestNewVersioningStrategyAliases(t *testing.T) {
	strategy, err := newVersioningStrategy(&config.VersioningCfg{Format: "sorted"})
	require.NoError(t, err)
	assert.Equal(t, versionFormatOrdered, strategy.format)
}

// TestParseNumericGroupInvalidValues tests the behavior of parseNumericGroup with invalid values.
//
// It verifies:
//   - Returns false for empty numeric string
//   - Returns false for non-numeric values
func TestParseNumericGroupInvalidValues(t *testing.T) {
	re := regexp.MustCompile(`(?P<major>\d+)(?:-(?P<minor>\w*))?`)
	match := re.FindStringSubmatch("10-")
	_, ok := parseNumericGroup(match, re, "minor", 2)
	assert.False(t, ok)

	match = re.FindStringSubmatch("10-abc")
	_, ok = parseNumericGroup(match, re, "minor", 2)
	assert.False(t, ok)
}

// TestParseNumericGroupIndexFallback tests the behavior of parseNumericGroup when named groups don't exist.
//
// It verifies:
//   - Falls back to numeric index when named group is missing
func TestParseNumericGroupIndexFallback(t *testing.T) {
	// Regex without named groups - should fallback to index
	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	match := re.FindStringSubmatch("1.2.3")

	// Named group "major" doesn't exist, should fall back to index 1
	major, ok := parseNumericGroup(match, re, "major", 1)
	assert.True(t, ok)
	assert.Equal(t, 1, major)

	// Named group "minor" doesn't exist, should fall back to index 2
	minor, ok := parseNumericGroup(match, re, "minor", 2)
	assert.True(t, ok)
	assert.Equal(t, 2, minor)
}

// TestNormalizeLoose tests the behavior of normalizeLoose.
//
// It verifies:
//   - Strips v prefix from versions
//   - Returns non-version strings unchanged
func TestNormalizeLoose(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", strategy.normalizeLoose("v1.2.3"))
	assert.Equal(t, "vnext", strategy.normalizeLoose("vnext"))
}

// TestSemverParts tests the behavior of semverParts.
//
// It verifies:
//   - Extracts major, minor, patch from 2-segment version
//   - Handles non-numeric patch segments
func TestSemverParts(t *testing.T) {
	major, minor, patch := semverParts("v1.2")
	assert.Equal(t, 1, major)
	assert.Equal(t, 2, minor)
	assert.Equal(t, 0, patch)

	_, _, patch = semverParts("v1.2.x")
	assert.Equal(t, 0, patch)
}

// TestCompareFallsBackToNormalized tests the behavior of compare when falling back to string comparison.
//
// It verifies:
//   - Falls back to normalized string comparison for numeric format
//   - compareInts returns 0 for equal values
func TestCompareFallsBackToNormalized(t *testing.T) {
	strategy, err := newVersioningStrategy(&config.VersioningCfg{Format: "numeric"})
	require.NoError(t, err)

	left := parsedVersion{normalized: "a"}
	right := parsedVersion{normalized: "b"}
	assert.Less(t, strategy.compare(left, right), 0)

	assert.Equal(t, 0, compareInts(5, 5))
}

// TestCompareWithCanonicalSemver tests the behavior of compare with canonical semver.
//
// It verifies:
//   - Compares versions using canonical semver format
func TestCompareWithCanonicalSemver(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	// Both have canonical semver
	a := parsedVersion{canonical: "v1.0.0"}
	b := parsedVersion{canonical: "v2.0.0"}
	assert.Less(t, strategy.compare(a, b), 0)
	assert.Greater(t, strategy.compare(b, a), 0)
}

// TestCompareWithNumericParts tests the behavior of compare with numeric version components.
//
// It verifies:
//   - Compares major versions first
//   - Compares minor when major is equal
//   - Compares patch when major and minor are equal
//   - Falls back to normalized comparison when all numeric parts are equal
func TestCompareWithNumericParts(t *testing.T) {
	strategy, err := newVersioningStrategy(&config.VersioningCfg{Format: "numeric"})
	require.NoError(t, err)

	// Different major
	a := parsedVersion{hasNumbers: true, major: 1, minor: 0, patch: 0, normalized: "1.0.0"}
	b := parsedVersion{hasNumbers: true, major: 2, minor: 0, patch: 0, normalized: "2.0.0"}
	assert.Less(t, strategy.compare(a, b), 0)

	// Same major, different minor
	a = parsedVersion{hasNumbers: true, major: 1, minor: 1, patch: 0, normalized: "1.1.0"}
	b = parsedVersion{hasNumbers: true, major: 1, minor: 2, patch: 0, normalized: "1.2.0"}
	assert.Less(t, strategy.compare(a, b), 0)

	// Same major and minor, different patch
	a = parsedVersion{hasNumbers: true, major: 1, minor: 1, patch: 1, normalized: "1.1.1"}
	b = parsedVersion{hasNumbers: true, major: 1, minor: 1, patch: 2, normalized: "1.1.2"}
	assert.Less(t, strategy.compare(a, b), 0)

	// All same - falls back to normalized comparison
	a = parsedVersion{hasNumbers: true, major: 1, minor: 1, patch: 1, normalized: "1.1.1-alpha"}
	b = parsedVersion{hasNumbers: true, major: 1, minor: 1, patch: 1, normalized: "1.1.1-beta"}
	assert.Less(t, strategy.compare(a, b), 0)
}

// TestCanonicalSemverInvalid tests the behavior of canonicalSemver with invalid inputs.
//
// It verifies:
//   - Returns empty string for non-semver input
//   - Returns valid semver unchanged
//   - Returns empty for whitespace-only input
//   - Completes partial versions
func TestCanonicalSemverInvalid(t *testing.T) {
	assert.Equal(t, "", canonicalSemver("not-semver"))
	assert.Equal(t, "v1.2.3", canonicalSemver("v1.2.3"))
	assert.Equal(t, "", canonicalSemver("   "))
	assert.Equal(t, "v1.0.0", canonicalSemver("1"))
	assert.Equal(t, "v1.2.0", canonicalSemver("1.2"))
}

// TestExtractPartsWithoutMatch tests the behavior of extractParts when regex doesn't match.
//
// It verifies:
//   - Returns false when input doesn't match regex pattern
func TestExtractPartsWithoutMatch(t *testing.T) {
	strategy, err := newVersioningStrategy(&config.VersioningCfg{Format: "regex", Regex: "not(major)"})
	require.NoError(t, err)
	_, _, _, ok := strategy.extractParts("abc")
	assert.False(t, ok)
}

// TestNewVersioningStrategyDefaultFormat tests the behavior of newVersioningStrategy with default format.
//
// It verifies:
//   - Nil config uses semver format
//   - Empty format string uses semver format
func TestNewVersioningStrategyDefaultFormat(t *testing.T) {
	t.Run("nil config uses semver format", func(t *testing.T) {
		strategy, err := newVersioningStrategy(nil)
		require.NoError(t, err)
		assert.Equal(t, versionFormatSemver, strategy.format)
	})

	t.Run("empty format uses semver", func(t *testing.T) {
		cfg := &config.VersioningCfg{Format: ""}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)
		assert.Equal(t, versionFormatSemver, strategy.format)
	})
}

// TestFilterNewerVersionsError tests error cases for FilterNewerVersions.
//
// It verifies:
//   - Returns error for invalid versioning config format
func TestFilterNewerVersionsError(t *testing.T) {
	// Invalid versioning config should return error
	cfg := &config.VersioningCfg{Format: "invalid_format"}
	_, err := FilterNewerVersions("1.0.0", []string{"2.0.0"}, cfg)
	assert.Error(t, err)
}

// TestFilterNewerVersionsWithStrategyCompare tests the behavior of FilterNewerVersions with comparison logic.
//
// It verifies:
//   - Uses compare for non-ordered format
//   - Passthrough when base is invalid
//   - Skips empty strings in semver filtering
func TestFilterNewerVersionsWithStrategyCompare(t *testing.T) {
	t.Run("uses compare for non-ordered format", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		result, err := FilterNewerVersions("1.0.0", []string{"0.9.0", "1.0.0", "1.1.0", "2.0.0"}, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "1.1.0")
		assert.Contains(t, result, "2.0.0")
		assert.NotContains(t, result, "0.9.0")
	})

	t.Run("passthrough when base invalid and version invalid", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// When base cannot be parsed and version cannot be parsed, version goes to passthrough
		result, err := FilterNewerVersions("not-a-version", []string{"also-not-valid", "neither-is-this"}, cfg)
		require.NoError(t, err)
		// Both invalid versions should be in passthrough (sorted alphabetically)
		assert.Contains(t, result, "also-not-valid")
		assert.Contains(t, result, "neither-is-this")
	})

	t.Run("skips empty strings in semver filtering", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// Empty strings should be skipped in non-ordered format
		result, err := FilterNewerVersions("1.0.0", []string{"", "2.0.0", "3.0.0"}, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "2.0.0")
		assert.Contains(t, result, "3.0.0")
		assert.Len(t, result, 2)
	})
}

// TestFilterOrderedContinueBranches tests the behavior of filterOrdered with edge cases.
//
// It verifies:
//   - Filters ordered versions with duplicates in ascending order
//   - Filters ordered versions with empty strings
func TestFilterOrderedContinueBranches(t *testing.T) {
	t.Run("filters ordered versions with duplicates ascending", func(t *testing.T) {
		// With Sort: "asc", versions after current should be included
		cfg := &config.VersioningCfg{Format: "ordered", Sort: "asc"}
		result, err := FilterNewerVersions("1.0.0", []string{"1.0.0", "1.0.0", "2.0.0", "2.0.0", "3.0.0"}, cfg)
		require.NoError(t, err)
		// Duplicates should be skipped, and only versions after 1.0.0 included
		assert.Equal(t, 2, len(result))
		assert.Contains(t, result, "2.0.0")
		assert.Contains(t, result, "3.0.0")
	})

	t.Run("filters ordered versions with empty strings", func(t *testing.T) {
		cfg := &config.VersioningCfg{Format: "ordered", Sort: "asc"}
		result, err := FilterNewerVersions("1.0.0", []string{"1.0.0", "", "  ", "2.0.0"}, cfg)
		require.NoError(t, err)
		// Empty strings should be skipped
		assert.Contains(t, result, "2.0.0")
	})
}

// TestExtractPartsReturnsFalse tests the behavior of extractParts when parsing fails.
//
// It verifies:
//   - Returns false when version doesn't match regex pattern
func TestExtractPartsReturnsFalse(t *testing.T) {
	t.Run("invalid version format returns false", func(t *testing.T) {
		cfg := &config.VersioningCfg{Regex: `^v(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)$`}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// Version that doesn't match the pattern
		major, minor, patch, ok := strategy.extractParts("not-a-version")
		assert.False(t, ok)
		assert.Equal(t, 0, major)
		assert.Equal(t, 0, minor)
		assert.Equal(t, 0, patch)
	})
}

// TestFilterNewerVersionsWithStrategyContinueBranches tests the behavior of FilterNewerVersions with edge cases.
//
// It verifies:
//   - Skips invalid parsed versions
//   - Handles base version that cannot be parsed
func TestFilterNewerVersionsWithStrategyContinueBranches(t *testing.T) {
	t.Run("skips invalid parsed versions", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// "not-semver" cannot be parsed, should be skipped
		result, err := FilterNewerVersions("1.0.0", []string{"not-semver", "2.0.0", "also-invalid"}, cfg)
		require.NoError(t, err)
		assert.Contains(t, result, "2.0.0")
		assert.Len(t, result, 1) // Only 2.0.0 should be included
	})

	t.Run("skips when base version cannot be parsed", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// Base version is invalid
		result, err := FilterNewerVersions("not-a-version", []string{"1.0.0", "2.0.0"}, cfg)
		require.NoError(t, err)
		// When base cannot be parsed, all valid versions are returned
		assert.Contains(t, result, "1.0.0")
		assert.Contains(t, result, "2.0.0")
	})
}

// TestExtractPartsMajorParsingFails tests the behavior of extractParts when major version parsing fails.
//
// It verifies:
//   - Returns false when matched major group is not numeric
func TestExtractPartsMajorParsingFails(t *testing.T) {
	t.Run("version matches but major group is not numeric", func(t *testing.T) {
		// Regex that matches but major group is not a number
		cfg := &config.VersioningCfg{Regex: `^(?P<major>[a-z]+)\.(?P<minor>\d+)\.(?P<patch>\d+)$`}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// This matches but "abc" cannot be parsed as int
		major, minor, patch, ok := strategy.extractParts("abc.1.2")
		assert.False(t, ok)
		assert.Equal(t, 0, major)
		assert.Equal(t, 0, minor)
		assert.Equal(t, 0, patch)
	})
}

// TestExtractPartsTieBreaking tests the behavior of extractParts with tie-breaking logic.
//
// It verifies:
//   - Prefers longer match when scores are equal
//   - Handles version with multiple potential matches
func TestExtractPartsTieBreaking(t *testing.T) {
	t.Run("prefers longer match when scores are equal", func(t *testing.T) {
		// This tests the tie-breaking logic: when two matches have the same score,
		// the longer match is preferred
		cfg := &config.VersioningCfg{Format: "semver"}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// Testing with a 4-segment version where multiple matches may occur
		major, minor, patch, ok := strategy.extractParts("1.2.3.4")
		assert.True(t, ok)
		assert.Equal(t, 1, major)
		assert.Equal(t, 2, minor)
		assert.Equal(t, 3, patch)
	})

	t.Run("handles version with multiple potential matches", func(t *testing.T) {
		// Testing with a complex version that may have multiple regex matches
		cfg := &config.VersioningCfg{Format: "semver"}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// "v1.0.0.0" should extract 1.0.0
		major, minor, patch, ok := strategy.extractParts("v1.0.0.0")
		assert.True(t, ok)
		assert.Equal(t, 1, major)
		assert.Equal(t, 0, minor)
		assert.Equal(t, 0, patch)
	})
}

// TestExtractPartsNoMatchReturnsNil tests the behavior of extractParts with no capture groups.
//
// It verifies:
//   - Returns false when regex has no named groups
func TestExtractPartsNoMatchReturnsNil(t *testing.T) {
	t.Run("regex with no capture groups", func(t *testing.T) {
		// Create a regex that matches but has no named groups
		cfg := &config.VersioningCfg{Regex: `\d+`}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// Should fail because major group doesn't exist
		_, _, _, ok := strategy.extractParts("123")
		assert.False(t, ok)
	})
}

// TestVersionFormatsFromPackageManagers tests version formats used by various package managers.
//
// This ensures the versioning system handles real-world version strings correctly.
//
// It verifies:
//   - NPM/Yarn/PNPM semver versions
//   - Python PyPI PEP 440 versions
//   - Go module versions
//   - Ruby Gem versions
//   - PHP Composer versions
//   - NuGet .NET versions
//   - Maven/Gradle Java versions
//   - Docker image tags
//   - 4+ segment versions
//   - CalVer versions
//   - Prefixed versions
func TestVersionFormatsFromPackageManagers(t *testing.T) {
	// Test data structure for version format tests
	type versionTest struct {
		version       string
		wantMajor     int
		wantMinor     int
		wantPatch     int
		shouldParse   bool
		description   string
		compareHigher string // optional: version that should compare higher
	}

	t.Run("NPM/Yarn/PNPM semver versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic semver", "1.2.4"},
			{"0.0.1", 0, 0, 1, true, "initial development", "0.0.2"},
			{"1.0.0-alpha", 1, 0, 0, true, "alpha prerelease", "1.0.0"},
			{"1.0.0-alpha.1", 1, 0, 0, true, "numbered alpha", "1.0.0-alpha.2"},
			{"1.0.0-beta.2", 1, 0, 0, true, "beta prerelease", "1.0.0-rc.1"},
			{"1.0.0-rc.1", 1, 0, 0, true, "release candidate", "1.0.0"},
			{"1.0.0+build.123", 1, 0, 0, true, "build metadata", ""},
			{"1.0.0-beta+exp.sha.5114f85", 1, 0, 0, true, "prerelease with build", ""},
			{"16.8.0", 16, 8, 0, true, "React-style version", "17.0.0"},
			{"14.0.0-canary.37", 14, 0, 0, true, "Next.js canary", "14.0.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, patch, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
					assert.Equal(t, v.wantPatch, patch, "patch for %s", v.version)
				}
				if v.compareHigher != "" {
					p1, _ := strategy.parseVersion(v.version)
					p2, _ := strategy.parseVersion(v.compareHigher)
					assert.Equal(t, -1, strategy.compare(p1, p2), "%s should be < %s", v.version, v.compareHigher)
				}
			})
		}
	})

	t.Run("Python PyPI PEP 440 versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic version", ""},
			{"1.0a1", 1, 0, 0, true, "alpha shorthand", ""},
			{"1.0b2", 1, 0, 0, true, "beta shorthand", ""},
			{"1.0rc3", 1, 0, 0, true, "rc shorthand", ""},
			{"1.0.post1", 1, 0, 0, true, "post release", ""},
			{"1.0.dev1", 1, 0, 0, true, "dev release", ""},
			{"2.0.0a1", 2, 0, 0, true, "alpha with minor", ""},
			{"3.11.0", 3, 11, 0, true, "Python version style", "3.12.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
				}
			})
		}
	})

	t.Run("Go module versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"v1.2.3", 1, 2, 3, true, "standard go version", "v1.2.4"},
			{"v0.0.0-20210101120000-abcdef123456", 0, 0, 0, true, "pseudo version", ""},
			{"v1.0.0-rc1", 1, 0, 0, true, "release candidate", "v1.0.0"},
			{"v2.0.0", 2, 0, 0, true, "major v2 module", "v3.0.0"},
			{"v1.21.0", 1, 21, 0, true, "Go runtime version style", "v1.22.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, patch, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
					assert.Equal(t, v.wantPatch, patch, "patch for %s", v.version)
				}
			})
		}
	})

	t.Run("Ruby Gem versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic gem version", ""},
			{"1.2.3.pre", 1, 2, 3, true, "prerelease", ""},
			{"1.2.3.beta.1", 1, 2, 3, true, "beta with number", ""},
			{"7.0.8", 7, 0, 8, true, "Rails version style", "7.1.0"},
			{"3.2.2", 3, 2, 2, true, "Ruby version style", "3.3.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, _, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
				}
			})
		}
	})

	t.Run("PHP Composer versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic composer version", ""},
			{"v1.2.3", 1, 2, 3, true, "v-prefixed", ""},
			{"1.2.3-alpha.1", 1, 2, 3, true, "alpha", ""},
			{"8.2.0", 8, 2, 0, true, "Laravel version style", "9.0.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, _, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
				}
			})
		}
	})

	t.Run("NuGet .NET versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic nuget version", ""},
			{"1.2.3-preview.1", 1, 2, 3, true, "preview release", "1.2.3"},
			{"1.2.3-beta.1", 1, 2, 3, true, "beta release", ""},
			{"8.0.0", 8, 0, 0, true, ".NET version style", "9.0.0"},
			{"6.0.0-rc.1", 6, 0, 0, true, "release candidate", "6.0.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, _, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
				}
			})
		}
	})

	t.Run("Maven/Gradle Java versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic maven version", ""},
			{"1.2.3-SNAPSHOT", 1, 2, 3, true, "snapshot", "1.2.3"},
			{"1.2.3.RELEASE", 1, 2, 3, true, "Spring release", ""},
			{"5.3.30", 5, 3, 30, true, "Spring version style", "6.0.0"},
			{"21.0.1", 21, 0, 1, true, "Java JDK version", "22.0.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, _, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
				}
			})
		}
	})

	t.Run("Docker image tags", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.2.3", 1, 2, 3, true, "basic docker tag", ""},
			{"v1.2.3", 1, 2, 3, true, "v-prefixed", ""},
			{"3.18.4", 3, 18, 4, true, "Alpine version", "3.19.0"},
			{"22.04", 22, 4, 0, true, "Ubuntu version", "24.04"},
			{"bookworm", 0, 0, 0, false, "Debian codename", ""},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, _, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
				}
			})
		}
	})

	t.Run("4+ segment versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"1.0.0.0", 1, 0, 0, true, "4-segment version", "1.0.0.1"},
			{"1.0.0.1", 1, 0, 0, true, "4-segment patch", "1.0.0.2"},
			{"10.0.0.0.1", 10, 0, 0, true, "5-segment version", ""},
			{"1.2.3.4.5.6", 1, 2, 3, true, "6-segment version", ""},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, patch, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
					assert.Equal(t, v.wantPatch, patch, "patch for %s", v.version)
				}
			})
		}
	})

	t.Run("CalVer versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"2024.01.15", 2024, 1, 15, true, "full date CalVer", "2024.01.16"},
			{"2024.1.15", 2024, 1, 15, true, "short month CalVer", ""},
			{"24.1.0", 24, 1, 0, true, "YY.MM CalVer", "24.2.0"},
			{"2024.1", 2024, 1, 0, true, "year.month CalVer", "2024.2"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, _, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
				}
			})
		}
	})

	t.Run("prefixed versions", func(t *testing.T) {
		strategy, _ := newVersioningStrategy(nil)
		versions := []versionTest{
			{"next-14.0.3", 14, 0, 3, true, "next.js style prefix", "next-14.0.4"},
			{"release-2024.01.15", 2024, 1, 15, true, "release prefix CalVer", ""},
			{"alpine-3.18.4", 3, 18, 4, true, "alpine prefix", "alpine-3.19.0"},
			{"node-18.19.0", 18, 19, 0, true, "node prefix", "node-20.0.0"},
		}

		for _, v := range versions {
			t.Run(v.description, func(t *testing.T) {
				major, minor, patch, ok := strategy.extractParts(v.version)
				assert.Equal(t, v.shouldParse, ok, "parse result for %s", v.version)
				if v.shouldParse {
					assert.Equal(t, v.wantMajor, major, "major for %s", v.version)
					assert.Equal(t, v.wantMinor, minor, "minor for %s", v.version)
					assert.Equal(t, v.wantPatch, patch, "patch for %s", v.version)
				}
			})
		}
	})
}

// TestNumericVersionFormat tests the numeric versioning strategy for date-based versions.
//
// It verifies:
//   - Moodle-style datetime versions
//   - Build number versions
//   - Numeric ordering is correct
func TestNumericVersionFormat(t *testing.T) {
	cfg := &config.VersioningCfg{Format: "numeric"}
	strategy, err := newVersioningStrategy(cfg)
	require.NoError(t, err)

	t.Run("Moodle-style datetime versions", func(t *testing.T) {
		versions := []string{"2024060100", "2024060101", "2024070100", "2023010100"}

		// Should parse as major-only
		for _, v := range versions {
			major, minor, patch, ok := strategy.extractParts(v)
			assert.True(t, ok, "should parse %s", v)
			assert.Greater(t, major, 0, "major should be set for %s", v)
			assert.Equal(t, 0, minor, "minor should be 0 for %s", v)
			assert.Equal(t, 0, patch, "patch should be 0 for %s", v)
		}
	})

	t.Run("build number versions", func(t *testing.T) {
		versions := []string{"150", "200", "250", "1000"}

		for _, v := range versions {
			major, _, _, ok := strategy.extractParts(v)
			assert.True(t, ok, "should parse %s", v)
			assert.Greater(t, major, 0, "major should be set for %s", v)
		}
	})

	t.Run("numeric ordering is correct", func(t *testing.T) {
		p1, _ := strategy.parseVersion("2024060100")
		p2, _ := strategy.parseVersion("2024070100")
		assert.Equal(t, -1, strategy.compare(p1, p2), "2024060100 should be < 2024070100")
	})
}

// TestOrderedVersionFormat tests the ordered versioning strategy for list-based versions.
//
// It verifies:
//   - Descending order (newest first)
//   - Ascending order (oldest first)
func TestOrderedVersionFormat(t *testing.T) {
	t.Run("descending order (newest first)", func(t *testing.T) {
		cfg := &config.VersioningCfg{Format: "ordered", Sort: "desc"}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// In desc order, first item is newest
		versions := []string{"bookworm", "bullseye", "buster", "stretch"}
		result := strategy.filterOrdered("bullseye", versions)

		// Should return versions before bullseye (i.e., newer)
		assert.Contains(t, result, "bookworm")
		assert.NotContains(t, result, "buster")
		assert.NotContains(t, result, "stretch")
	})

	t.Run("ascending order (oldest first)", func(t *testing.T) {
		cfg := &config.VersioningCfg{Format: "ordered", Sort: "asc"}
		strategy, err := newVersioningStrategy(cfg)
		require.NoError(t, err)

		// In asc order, last item is newest
		versions := []string{"stretch", "buster", "bullseye", "bookworm"}
		result := strategy.filterOrdered("bullseye", versions)

		// Should return versions after bullseye (i.e., newer)
		assert.Contains(t, result, "bookworm")
		assert.NotContains(t, result, "stretch")
		assert.NotContains(t, result, "buster")
	})
}

// TestVersionConstraintMapping tests constraint handling for various formats.
//
// It verifies:
//   - Constraint determines update scope
//   - Flags override constraints
func TestVersionConstraintMapping(t *testing.T) {
	t.Run("constraint determines update scope", func(t *testing.T) {
		testCases := []struct {
			constraint    string
			expectedScope string
		}{
			{"", "major"},     // No constraint = all updates
			{"*", "major"},    // Wildcard = all updates
			{"^", "minor"},    // Caret = minor + patch
			{"~", "patch"},    // Tilde = patch only
			{"=", "major"},    // Exact = falls back to major for scope determination
			{">=", "major"},   // Greater/equal = falls back to major
			{">", "major"},    // Greater = falls back to major
			{"<=", "major"},   // Less/equal = falls back to major
			{"<", "major"},    // Less = falls back to major
		}

		for _, tc := range testCases {
			t.Run(tc.constraint, func(t *testing.T) {
				scope := determineScope(UpdateSelectionFlags{}, tc.constraint)
				assert.Equal(t, tc.expectedScope, scope, "constraint %q", tc.constraint)
			})
		}
	})

	t.Run("flags override constraints", func(t *testing.T) {
		// Major flag always returns major scope
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{Major: true}, "~"))
		// Minor flag returns minor scope
		assert.Equal(t, "minor", determineScope(UpdateSelectionFlags{Minor: true}, "~"))
		// Patch flag returns patch scope
		assert.Equal(t, "patch", determineScope(UpdateSelectionFlags{Patch: true}, "^"))
	})
}

// TestVersionOrdering tests that version sorting works correctly.
//
// It verifies:
//   - Semver ordering
//   - Mixed format ordering
func TestVersionOrdering(t *testing.T) {
	strategy, _ := newVersioningStrategy(nil)

	t.Run("semver ordering", func(t *testing.T) {
		versions := []string{
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-beta",
			"1.0.0-beta.2",
			"1.0.0-rc.1",
			"1.0.0",
			"1.0.1",
			"1.1.0",
			"2.0.0",
		}

		// Each version should be less than the next
		for i := 0; i < len(versions)-1; i++ {
			p1, _ := strategy.parseVersion(versions[i])
			p2, _ := strategy.parseVersion(versions[i+1])
			result := strategy.compare(p1, p2)
			assert.Equal(t, -1, result, "%s should be < %s", versions[i], versions[i+1])
		}
	})

	t.Run("mixed format ordering", func(t *testing.T) {
		// v-prefixed and non-prefixed should compare equally
		p1, _ := strategy.parseVersion("1.0.0")
		p2, _ := strategy.parseVersion("v1.0.0")
		assert.Equal(t, 0, strategy.compare(p1, p2), "1.0.0 should equal v1.0.0")
	})
}
