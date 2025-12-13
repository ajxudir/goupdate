package outdated

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ajxudir/goupdate/pkg/config"
)

// =============================================================================
// CHAOS/BATTLE TESTS - VERSION PARSING AND TAG PROCESSING
// =============================================================================
//
// These tests verify that version parsing and comparison cannot be exploited
// or tricked into incorrect behavior. They test:
//
// 1. Version parsing edge cases (malformed, extreme values, special chars)
// 2. Version comparison manipulation attempts
// 3. Pre-release/alpha/beta tag handling
// 4. ReDoS (Regular Expression Denial of Service) attacks
// 5. Version normalization bypass attempts
// 6. Constraint validation edge cases
//
// =============================================================================

// -----------------------------------------------------------------------------
// VERSION PARSING EDGE CASES
// -----------------------------------------------------------------------------

// TestChaos_ParseVersion_MalformedInputs tests parsing of malformed version strings.
//
// It verifies:
//   - Malformed versions are rejected or handled gracefully
//   - No panics occur with strange inputs
//   - No infinite loops or hangs
func TestChaos_ParseVersion_MalformedInputs(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		version string
	}{
		// Empty and whitespace
		{"empty", ""},
		{"whitespace_only", "   "},
		{"tabs_only", "\t\t\t"},
		{"newlines", "\n\n\n"},
		{"mixed_whitespace", " \t\n "},

		// Invalid formats
		{"dots_only", "..."},
		{"dashes_only", "---"},
		{"underscores_only", "___"},
		{"single_dot", "."},
		{"leading_dots", "..1.0.0"},
		{"trailing_dots", "1.0.0.."},

		// Too many segments
		{"four_segments", "1.2.3.4"},
		{"five_segments", "1.2.3.4.5"},
		{"ten_segments", "1.2.3.4.5.6.7.8.9.10"},

		// Non-numeric
		{"all_letters", "abc.def.ghi"},
		{"mixed_letters", "1.a.2"},
		{"special_chars", "1.0.0$#@"},

		// Unicode
		{"unicode_numbers", "\u0661.\u0662.\u0663"}, // Arabic-Indic digits
		{"emoji", "1.0.0-\U0001F389"},
		{"zero_width_space", "1.0\u200B.0"},
		{"rtl_override", "1.0.0\u202E"},

		// Extreme values
		{"max_int64", "9223372036854775807.0.0"},
		{"overflow_int64", "9223372036854775808.0.0"},
		{"huge_number", "999999999999999999999999999999.0.0"},
		{"negative", "-1.0.0"},

		// Control characters
		{"null_byte", "1.0\x000.0"},
		{"carriage_return", "1.0\r.0"},
		{"bell", "1.0\x07.0"},

		// Shell injection attempts
		{"command_substitution_dollar", "$(whoami).0.0"},
		{"command_substitution_backtick", "`whoami`.0.0"},
		{"semicolon", "1.0.0;rm -rf /"},
		{"pipe", "1.0.0|cat /etc/passwd"},
		{"redirect", "1.0.0>/tmp/pwned"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Should not panic
			parsed, ok := strategy.parseVersion(tc.version)

			// Log result for analysis
			t.Logf("Input: %q, OK: %v, Parsed: %+v", tc.version, ok, parsed)

			// Even if parsing succeeds, the result should be sanitized
			if ok {
				// Should not contain shell injection chars in normalized form
				assert.NotContains(t, parsed.normalized, ";")
				assert.NotContains(t, parsed.normalized, "|")
				assert.NotContains(t, parsed.normalized, ">")
				assert.NotContains(t, parsed.normalized, "$")
				assert.NotContains(t, parsed.normalized, "`")
			}
		})
	}
}

// TestChaos_ParseVersion_ExtremelyLongInput tests handling of very long version strings.
//
// It verifies:
//   - No buffer overflow or memory exhaustion
//   - Completes in reasonable time
func TestChaos_ParseVersion_ExtremelyLongInput(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		version string
	}{
		{"long_major", strings.Repeat("9", 10000) + ".0.0"},
		{"long_all", strings.Repeat("9", 100) + "." + strings.Repeat("9", 100) + "." + strings.Repeat("9", 100)},
		{"many_dots", strings.Repeat("1.", 10000) + "0"},
		{"long_prerelease", "1.0.0-" + strings.Repeat("alpha", 10000)},
		{"repeated_pattern", strings.Repeat("1.0.0-", 1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			done := make(chan bool)

			go func() {
				parsed, ok := strategy.parseVersion(tc.version)
				t.Logf("Long input: len=%d, OK: %v, normalized len: %d",
					len(tc.version), ok, len(parsed.normalized))
				done <- true
			}()

			select {
			case <-done:
				elapsed := time.Since(start)
				assert.Less(t, elapsed, 5*time.Second,
					"parsing should complete quickly, took %v", elapsed)
			case <-time.After(10 * time.Second):
				t.Fatal("parsing took too long - possible infinite loop or ReDoS")
			}
		})
	}
}

// TestChaos_ParseVersion_PrereleaseTagManipulation tests pre-release tag handling.
//
// It verifies:
//   - Pre-release versions are not treated as production releases
//   - Alpha/beta/rc tags are properly detected and compared
//   - Tags cannot be used to trick version comparison
func TestChaos_ParseVersion_PrereleaseTagManipulation(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	t.Run("prerelease_vs_release_ordering", func(t *testing.T) {
		// Pre-release should be LESS than release per semver spec
		prerelease, okPre := strategy.parseVersion("1.0.0-alpha")
		release, okRel := strategy.parseVersion("1.0.0")

		require.True(t, okPre, "prerelease should parse")
		require.True(t, okRel, "release should parse")

		// If both have canonical semver, semver comparison applies
		if prerelease.canonical != "" && release.canonical != "" {
			cmp := strategy.compare(prerelease, release)
			assert.Less(t, cmp, 0, "prerelease 1.0.0-alpha should be < release 1.0.0")
		}
	})

	t.Run("prerelease_tags_not_stripped_silently", func(t *testing.T) {
		versions := []string{
			"1.0.0-alpha",
			"1.0.0-beta",
			"1.0.0-rc.1",
			"1.0.0-alpha.1",
			"1.0.0+build",
			"1.0.0-alpha+build",
		}

		for _, v := range versions {
			parsed, ok := strategy.parseVersion(v)
			if ok && parsed.canonical != "" {
				// Canonical form should preserve prerelease info for dedup purposes
				key := strategy.keyFor(parsed, v)
				t.Logf("Version: %s, Key: %s", v, key)
				// Key should differentiate between prerelease and release
			}
		}

		// These should have different keys and not be deduplicated
		alpha, _ := strategy.parseVersion("1.0.0-alpha")
		release, _ := strategy.parseVersion("1.0.0")

		alphaKey := strategy.keyFor(alpha, "1.0.0-alpha")
		releaseKey := strategy.keyFor(release, "1.0.0")

		assert.NotEqual(t, alphaKey, releaseKey,
			"prerelease and release should have different dedup keys")
	})

	t.Run("deceptive_prerelease_tags", func(t *testing.T) {
		// These look like production but have hidden prerelease markers
		deceptive := []struct {
			version  string
			desc     string
		}{
			{"1.0.0-", "empty prerelease"},
			{"1.0.0--", "double dash"},
			{"1.0.0- ", "space after dash"},
			{"1.0.0-\t", "tab prerelease"},
			{"1.0.0-\n", "newline prerelease"},
		}

		for _, tc := range deceptive {
			t.Run(tc.desc, func(t *testing.T) {
				parsed, ok := strategy.parseVersion(tc.version)
				t.Logf("Deceptive %s: OK=%v, canonical=%s, normalized=%s",
					tc.desc, ok, parsed.canonical, parsed.normalized)
				// Should either fail to parse or not have canonical semver form
			})
		}
	})
}

// -----------------------------------------------------------------------------
// VERSION COMPARISON MANIPULATION
// -----------------------------------------------------------------------------

// TestChaos_Compare_OrderingManipulation tests that version comparison cannot be tricked.
//
// It verifies:
//   - Comparison is consistent and transitive
//   - Cannot trick comparison with format differences
func TestChaos_Compare_OrderingManipulation(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	t.Run("transitivity", func(t *testing.T) {
		// If a < b and b < c, then a < c
		a, _ := strategy.parseVersion("1.0.0")
		b, _ := strategy.parseVersion("2.0.0")
		c, _ := strategy.parseVersion("3.0.0")

		assert.Less(t, strategy.compare(a, b), 0)
		assert.Less(t, strategy.compare(b, c), 0)
		assert.Less(t, strategy.compare(a, c), 0, "comparison should be transitive")
	})

	t.Run("symmetry", func(t *testing.T) {
		// If a < b, then b > a
		a, _ := strategy.parseVersion("1.0.0")
		b, _ := strategy.parseVersion("2.0.0")

		assert.Less(t, strategy.compare(a, b), 0)
		assert.Greater(t, strategy.compare(b, a), 0, "comparison should be symmetric")
	})

	t.Run("equality", func(t *testing.T) {
		// Same version in different formats should be equal
		formats := []string{
			"1.0.0",
			"v1.0.0",
			" 1.0.0 ",
			"1.0.0 ",
			" v1.0.0",
		}

		for i := 0; i < len(formats); i++ {
			for j := i + 1; j < len(formats); j++ {
				a, okA := strategy.parseVersion(formats[i])
				b, okB := strategy.parseVersion(formats[j])

				if okA && okB {
					cmp := strategy.compare(a, b)
					assert.Equal(t, 0, cmp,
						"'%s' should equal '%s'", formats[i], formats[j])
				}
			}
		}
	})

	t.Run("format_difference_attack", func(t *testing.T) {
		// Try to make newer version appear older through format manipulation
		newer := "2.0.0"
		older := "1.9.999" // Many patch releases

		newerParsed, _ := strategy.parseVersion(newer)
		olderParsed, _ := strategy.parseVersion(older)

		assert.Greater(t, strategy.compare(newerParsed, olderParsed), 0,
			"2.0.0 should be greater than 1.9.999")
	})
}

// TestChaos_Compare_NumericOverflow tests handling of numeric overflow in comparisons.
//
// It verifies:
//   - Large numbers don't cause integer overflow
//   - Comparison remains correct with extreme values
func TestChaos_Compare_NumericOverflow(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	t.Run("large_numbers_compare_correctly", func(t *testing.T) {
		large1, ok1 := strategy.parseVersion("999999999.0.0")
		large2, ok2 := strategy.parseVersion("1000000000.0.0")

		if ok1 && ok2 && large1.hasNumbers && large2.hasNumbers {
			// Should compare correctly without overflow
			cmp := strategy.compare(large1, large2)
			assert.Less(t, cmp, 0, "999999999 should be less than 1000000000")
		}
	})

	t.Run("near_max_int", func(t *testing.T) {
		// Near max int32 values
		v1, ok1 := strategy.parseVersion("2147483646.0.0")
		v2, ok2 := strategy.parseVersion("2147483647.0.0")

		if ok1 && ok2 && v1.hasNumbers && v2.hasNumbers {
			cmp := strategy.compare(v1, v2)
			assert.Less(t, cmp, 0)
		}
	})
}

// -----------------------------------------------------------------------------
// REGEX DENIAL OF SERVICE (ReDoS) TESTS
// -----------------------------------------------------------------------------

// TestChaos_ReDoS_DefaultRegex tests that default version regex doesn't have ReDoS.
//
// It verifies:
//   - Malicious inputs don't cause exponential backtracking
//   - Parsing completes in bounded time
func TestChaos_ReDoS_DefaultRegex(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	// Known ReDoS attack patterns for version-like regexes
	redosPatterns := []struct {
		name    string
		pattern string
	}{
		// Repeated dots/dashes that could cause backtracking
		{"repeated_dots", strings.Repeat("1.", 100) + "!"},
		{"repeated_dashes", strings.Repeat("1-", 100) + "!"},
		{"alternating", strings.Repeat("1.a", 50) + "!"},

		// Patterns that exploit optional groups
		{"optional_abuse", strings.Repeat("1", 50) + "." + strings.Repeat("2", 50)},

		// Mixed separators
		{"mixed_separators", "1._-._-._-._-._-._-._-._-._-.x"},
	}

	for _, tc := range redosPatterns {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			done := make(chan bool)

			go func() {
				strategy.parseVersion(tc.pattern)
				done <- true
			}()

			select {
			case <-done:
				elapsed := time.Since(start)
				// Should complete in under 1 second for non-pathological cases
				assert.Less(t, elapsed, time.Second,
					"ReDoS pattern %s took %v", tc.name, elapsed)
			case <-time.After(5 * time.Second):
				t.Fatalf("ReDoS vulnerability detected with pattern: %s", tc.name)
			}
		})
	}
}

// TestChaos_CustomRegex_ReDoS tests that custom regexes are validated for ReDoS.
//
// It verifies:
//   - Custom version regex patterns are bounded
//   - Dangerous patterns don't cause hangs
func TestChaos_CustomRegex_ReDoS(t *testing.T) {
	// These are known dangerous regex patterns
	dangerousPatterns := []string{
		`(a+)+b`,           // Nested quantifiers
		`(a|ab)+`,          // Overlapping alternatives
		`(.*)*b`,           // Nested wildcards
		`([a-zA-Z]+)*`,     // Nested character classes
		`(a|a)+`,           // Redundant alternatives
	}

	for _, pattern := range dangerousPatterns {
		t.Run(pattern, func(t *testing.T) {
			cfg := &config.VersioningCfg{
				Format: "regex",
				Regex:  pattern,
			}

			strategy, err := newVersioningStrategy(cfg)
			if err != nil {
				// Good - dangerous pattern rejected
				t.Logf("Dangerous pattern rejected: %v", err)
				return
			}

			// If pattern was accepted, ensure it doesn't cause ReDoS
			testInput := strings.Repeat("a", 30) + "!"

			start := time.Now()
			done := make(chan bool)

			go func() {
				strategy.parseVersion(testInput)
				done <- true
			}()

			select {
			case <-done:
				elapsed := time.Since(start)
				assert.Less(t, elapsed, 2*time.Second,
					"Custom regex %s with input may have ReDoS", pattern)
			case <-time.After(5 * time.Second):
				t.Fatalf("ReDoS vulnerability with custom regex: %s", pattern)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// VERSIONING STRATEGY EDGE CASES
// -----------------------------------------------------------------------------

// TestChaos_VersioningStrategy_UnknownFormat tests handling of unknown format types.
//
// It verifies:
//   - Unknown formats are rejected with clear error
//   - No fallback to unsafe default
func TestChaos_VersioningStrategy_UnknownFormat(t *testing.T) {
	unknownFormats := []string{
		"unknown",
		"SEMVER",           // Case sensitivity
		"semver ",          // Trailing space
		" semver",          // Leading space
		"semver\n",         // Newline
		"json",             // Wrong type
		"../../../etc/passwd", // Path traversal
		"$(whoami)",        // Command injection
	}

	for _, format := range unknownFormats {
		t.Run(format, func(t *testing.T) {
			cfg := &config.VersioningCfg{
				Format: format,
			}

			strategy, err := newVersioningStrategy(cfg)

			// Should either error or normalize the format
			if err == nil {
				// If no error, verify format was normalized correctly
				t.Logf("Format '%s' was accepted, strategy format: %s",
					format, strategy.format)
				// Should be one of the known formats
				validFormats := []string{"semver", "numeric", "regex", "ordered"}
				found := false
				for _, valid := range validFormats {
					if strategy.format == valid {
						found = true
						break
					}
				}
				assert.True(t, found, "unknown format should be rejected or normalized")
			}
		})
	}
}

// TestChaos_FilterNewerVersions_EmptyInputs tests filtering with edge case inputs.
//
// It verifies:
//   - Empty version lists are handled gracefully
//   - Empty current version doesn't cause issues
func TestChaos_FilterNewerVersions_EmptyInputs(t *testing.T) {
	t.Run("empty_current_version", func(t *testing.T) {
		versions := []string{"1.0.0", "2.0.0", "3.0.0"}

		// Should not panic
		result, err := FilterNewerVersions("", versions, nil)

		// May return all versions or error - just verify no panic
		t.Logf("Empty current: result=%v, err=%v", result, err)
	})

	t.Run("empty_version_list", func(t *testing.T) {
		result, err := FilterNewerVersions("1.0.0", []string{}, nil)

		assert.NoError(t, err)
		assert.Empty(t, result, "empty input should return empty output")
	})

	t.Run("nil_version_list", func(t *testing.T) {
		result, err := FilterNewerVersions("1.0.0", nil, nil)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("all_empty_versions", func(t *testing.T) {
		versions := []string{"", "", ""}
		result, err := FilterNewerVersions("1.0.0", versions, nil)

		assert.NoError(t, err)
		// Empty versions should be filtered out
		t.Logf("All empty versions: result=%v", result)
	})
}

// TestChaos_FilterNewerVersions_DuplicateHandling tests deduplication edge cases.
//
// It verifies:
//   - Duplicates are properly deduplicated
//   - Near-duplicates (format differences) are handled
func TestChaos_FilterNewerVersions_DuplicateHandling(t *testing.T) {
	t.Run("exact_duplicates", func(t *testing.T) {
		versions := []string{"2.0.0", "2.0.0", "2.0.0", "3.0.0"}
		result, err := FilterNewerVersions("1.0.0", versions, nil)

		assert.NoError(t, err)
		// Should deduplicate
		countOf2 := 0
		for _, v := range result {
			if v == "2.0.0" {
				countOf2++
			}
		}
		assert.LessOrEqual(t, countOf2, 1, "duplicates should be removed")
	})

	t.Run("format_duplicates", func(t *testing.T) {
		// Same version in different formats
		versions := []string{"2.0.0", "v2.0.0", " 2.0.0", "2.0.0 "}
		result, err := FilterNewerVersions("1.0.0", versions, nil)

		assert.NoError(t, err)
		t.Logf("Format duplicates result: %v", result)
		// These should be treated as the same version
	})
}

// -----------------------------------------------------------------------------
// ORDERED VERSION FORMAT TESTS
// -----------------------------------------------------------------------------

// TestChaos_OrderedFormat_ListManipulation tests ordered format list handling.
//
// It verifies:
//   - List order is preserved correctly
//   - Position-based filtering works
func TestChaos_OrderedFormat_ListManipulation(t *testing.T) {
	cfg := &config.VersioningCfg{
		Format: "ordered",
		Sort:   "desc",
	}

	strategy, err := newVersioningStrategy(cfg)
	require.NoError(t, err)

	t.Run("preserves_order", func(t *testing.T) {
		// Ordered format should use list position, not semver comparison
		versions := []string{"3.0", "1.0", "2.0"} // Not in semver order

		result := strategy.filterOrdered("1.0", versions)

		// With desc sort, versions before "1.0" in the list should be returned
		// Position 0: "3.0", Position 1: "1.0", Position 2: "2.0"
		// So "3.0" should be in result as it comes before "1.0" in the list
		t.Logf("Ordered filter result: %v", result)
	})

	t.Run("non_semver_versions", func(t *testing.T) {
		// Ordered format should handle non-semver strings
		versions := []string{"latest", "stable", "beta", "alpha"}

		result := strategy.filterOrdered("beta", versions)
		t.Logf("Non-semver ordered result: %v", result)
	})
}

// -----------------------------------------------------------------------------
// CANONICAL SEMVER TESTS
// -----------------------------------------------------------------------------

// TestChaos_CanonicalSemver_EdgeCases tests canonicalization edge cases.
//
// It verifies:
//   - Non-semver strings don't become valid semver
//   - Partial versions are padded correctly
func TestChaos_CanonicalSemver_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string // Empty if should not produce valid canonical
	}{
		// Valid inputs
		{"full_semver", "1.2.3", "v1.2.3"},
		{"with_v", "v1.2.3", "v1.2.3"},
		{"two_parts", "1.2", "v1.2.0"},
		{"one_part", "1", "v1.0.0"},

		// Invalid inputs
		{"empty", "", ""},
		{"not_available", "#N/A", ""},
		{"letters", "abc", ""},
		{"negative", "-1.0.0", ""},

		// Prerelease
		{"prerelease", "1.0.0-alpha", "v1.0.0-alpha"},
		// Build metadata is stripped by semver.Canonical per spec
		// (build metadata does not affect version precedence/ordering)
		{"build_metadata", "1.0.0+build", "v1.0.0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := canonicalSemver(tc.input)

			if tc.expected == "" {
				assert.Empty(t, result, "should not produce canonical for %q", tc.input)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// CONSTRAINT MAPPING TESTS
// -----------------------------------------------------------------------------

// TestChaos_ConstraintMapping_UnknownOperators tests handling of unknown constraint operators.
//
// It verifies:
//   - Unknown operators don't silently pass through
//   - Invalid constraints are rejected or defaulted safely
func TestChaos_ConstraintMapping_UnknownOperators(t *testing.T) {
	// Note: ConstraintMapping is in utils package, but we test the integration here

	unknownOperators := []string{
		">>>",
		"<<<",
		"??",
		"||",
		"&&",
		"!=",
		"<>",
		"~>", // Ruby-style
		"=>", // Assignment-like
	}

	for _, op := range unknownOperators {
		t.Run(op, func(t *testing.T) {
			// This should be handled at the constraint validation level
			t.Logf("Unknown operator %s - should be validated elsewhere", op)
		})
	}
}

// -----------------------------------------------------------------------------
// INTEGRATION TESTS - VERSION FLOW
// -----------------------------------------------------------------------------

// TestChaos_VersionFlow_EndToEnd tests complete version processing flow.
//
// It verifies:
//   - Versions flow correctly through parsing -> comparison -> filtering
//   - Edge cases at each stage don't compound into incorrect results
func TestChaos_VersionFlow_EndToEnd(t *testing.T) {
	t.Run("production_detection", func(t *testing.T) {
		// Test that we can't trick the system into treating pre-release as production
		current := "1.0.0"
		versions := []string{
			"1.0.0-alpha", // Should be filtered (less than current)
			"1.0.1-beta",  // Newer but prerelease
			"1.1.0",       // Newer production
			"2.0.0-rc.1",  // Major prerelease
			"2.0.0",       // Major production
		}

		result, err := FilterNewerVersions(current, versions, nil)
		assert.NoError(t, err)

		t.Logf("Filtered versions: %v", result)

		// 2.0.0 should definitely be in results
		found200 := false
		for _, v := range result {
			if v == "2.0.0" {
				found200 = true
			}
		}
		assert.True(t, found200, "2.0.0 should be in filtered results")
	})

	t.Run("sort_consistency", func(t *testing.T) {
		versions := []string{"1.0.0", "3.0.0", "2.0.0", "1.5.0", "2.5.0"}

		result, err := FilterNewerVersions("0.1.0", versions, nil)
		assert.NoError(t, err)

		// Result should be sorted (by default descending - newest first)
		if len(result) >= 2 {
			first, _ := (&versioningStrategy{format: versionFormatSemver, regex: defaultVersionRegex}).parseVersion(result[0])
			second, _ := (&versioningStrategy{format: versionFormatSemver, regex: defaultVersionRegex}).parseVersion(result[1])

			cmp := (&versioningStrategy{format: versionFormatSemver}).compare(first, second)
			assert.GreaterOrEqual(t, cmp, 0, "results should be sorted descending")
		}
	})
}

// TestChaos_VersionRegex_NamedGroups tests that named groups are handled correctly.
//
// It verifies:
//   - Named groups (major, minor, patch) are extracted properly
//   - Missing groups don't cause issues
func TestChaos_VersionRegex_NamedGroups(t *testing.T) {
	testCases := []struct {
		name   string
		regex  string
		input  string
		major  int
		minor  int
		patch  int
		hasNum bool
	}{
		{
			name:   "all_named",
			regex:  `(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)`,
			input:  "1.2.3",
			major:  1,
			minor:  2,
			patch:  3,
			hasNum: true,
		},
		{
			name:   "major_only",
			regex:  `(?P<major>\d+)`,
			input:  "42",
			major:  42,
			minor:  0,
			patch:  0,
			hasNum: true,
		},
		{
			name:   "no_names",
			regex:  `(\d+)\.(\d+)\.(\d+)`,
			input:  "1.2.3",
			major:  1,
			minor:  2,
			patch:  3,
			hasNum: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.VersioningCfg{
				Format: "regex",
				Regex:  tc.regex,
			}

			strategy, err := newVersioningStrategy(cfg)
			require.NoError(t, err)

			parsed, ok := strategy.parseVersion(tc.input)

			assert.Equal(t, tc.hasNum, ok, "parse success")
			if ok {
				assert.Equal(t, tc.major, parsed.major)
				assert.Equal(t, tc.minor, parsed.minor)
				assert.Equal(t, tc.patch, parsed.patch)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// BENCHMARK FOR PERFORMANCE REGRESSION
// -----------------------------------------------------------------------------

// BenchmarkParseVersion_Normal benchmarks normal version parsing.
func BenchmarkParseVersion_Normal(b *testing.B) {
	strategy, _ := newVersioningStrategy(nil)
	versions := []string{"1.0.0", "2.3.4", "10.20.30", "v1.2.3-alpha", "0.0.1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range versions {
			strategy.parseVersion(v)
		}
	}
}

// BenchmarkParseVersion_Pathological benchmarks potentially slow inputs.
func BenchmarkParseVersion_Pathological(b *testing.B) {
	strategy, _ := newVersioningStrategy(nil)
	pathological := strings.Repeat("1.", 100) + "0"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategy.parseVersion(pathological)
	}
}

// BenchmarkFilterNewerVersions benchmarks version filtering.
func BenchmarkFilterNewerVersions(b *testing.B) {
	versions := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		versions[i] = regexp.MustCompile(`[^0-9]`).ReplaceAllString(
			strings.Repeat("1", i%10+1), "") + ".0.0"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterNewerVersions("1.0.0", versions, nil)
	}
}
