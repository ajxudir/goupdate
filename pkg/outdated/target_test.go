package outdated

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// TestParseVersionAndCurrentVersionHelpers tests the behavior of version parsing and current version helpers.
//
// It verifies:
//   - parseVersion returns valid parsed version with numbers
//   - parseVersion returns invalid result for empty string
//   - CurrentVersionForOutdated prefers installed version
//   - CurrentVersionForOutdated falls back to version when installed is #N/A
func TestParseVersionAndCurrentVersionHelpers(t *testing.T) {
	strategy, err := newVersioningStrategy(nil)
	require.NoError(t, err)

	parsed, ok := strategy.parseVersion("1.2.3")
	require.True(t, ok)
	assert.True(t, parsed.hasNumbers)

	parsed, ok = strategy.parseVersion("")
	assert.False(t, ok)
	assert.Empty(t, parsed.normalized)

	assert.Equal(t, "1.0.0", CurrentVersionForOutdated(formats.Package{InstalledVersion: "1.0.0", Version: "0.1.0"}))
	assert.Equal(t, "0.1.0", CurrentVersionForOutdated(formats.Package{InstalledVersion: "#N/A", Version: "0.1.0"}))
}

// TestSelectTargetVersion tests the behavior of SelectTargetVersion in non-incremental mode.
//
// It verifies:
//   - Major flag selects major version first
//   - Falls back to minor when major is #N/A
//   - Falls back to patch when major and minor are #N/A
//   - Minor flag with patch available selects minor
//   - Caret constraint selects minor version
//   - Returns error when no suitable version is found
func TestSelectTargetVersion(t *testing.T) {
	// Non-incremental mode tests (existing behavior)
	target, err := SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{Major: true}, "", false)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", target)

	target, err = SelectTargetVersion("#N/A", "1.1.0", "1.0.1", UpdateSelectionFlags{Major: true}, "", false)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target)

	target, err = SelectTargetVersion("#N/A", "#N/A", "1.0.1", UpdateSelectionFlags{Minor: true}, "", false)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target)

	// Test minor flag with minor available but no patch - hits line 736
	target, err = SelectTargetVersion("#N/A", "1.1.0", "#N/A", UpdateSelectionFlags{Minor: true}, "", false)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target, "Minor flag should return minor when patch is N/A")

	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "^", false)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target)

	_, err = SelectTargetVersion("#N/A", "#N/A", "#N/A", UpdateSelectionFlags{Patch: true}, "", false)
	assert.Error(t, err)
}

// TestSelectTargetVersion_Incremental tests the behavior of SelectTargetVersion in incremental mode.
//
// It verifies:
//   - Incremental mode prioritizes patch → minor → major
//   - Major flag with incremental picks smallest available step
//   - Minor flag with incremental only considers patch and minor
//   - Patch flag with incremental only considers patch
//   - Constraint-based scoping works with incremental mode
func TestSelectTargetVersion_Incremental(t *testing.T) {
	// Incremental mode prioritizes patch → minor → major

	// With --major flag, still pick patch first (smallest step)
	target, err := SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{Major: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with --major should pick patch first")

	// With --major flag, pick minor if no patch
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{Major: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target, "Incremental with --major should pick minor if no patch")

	// With --major flag, pick major if no patch or minor
	target, err = SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{Major: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", target, "Incremental with --major should pick major if no patch or minor")

	// With --minor flag, pick patch first
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{Minor: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with --minor should pick patch first")

	// With --minor flag, pick minor if no patch (but NOT major)
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{Minor: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target, "Incremental with --minor should pick minor if no patch")

	// With --minor flag, error if only major available
	_, err = SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{Minor: true}, "", true)
	assert.Error(t, err, "Incremental with --minor should error if only major available")

	// With --patch flag, only patch allowed
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{Patch: true}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with --patch should only pick patch")

	// With --patch flag, error if no patch
	_, err = SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{Patch: true}, "", true)
	assert.Error(t, err, "Incremental with --patch should error if no patch")

	// No flags with ^ constraint: patch first, then minor
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "^", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with ^ constraint should pick patch first")

	// No flags with ^ constraint: pick minor if no patch
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{}, "^", true)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target, "Incremental with ^ constraint should pick minor if no patch")

	// No flags with ~ constraint: only patch
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "~", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with ~ constraint should pick patch")

	// No flags with no constraint: patch → minor → major
	target, err = SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.0.1", target, "Incremental with no constraint should pick patch first")

	target, err = SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "1.1.0", target, "Incremental with no constraint should pick minor if no patch")

	target, err = SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{}, "", true)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", target, "Incremental with no constraint should pick major if no patch or minor")
}

// TestSelectTargetVersionIncrementalTildeConstraint tests incremental mode with tilde constraint.
//
// It verifies:
//   - Incremental with tilde constraint returns error when no patch available
func TestSelectTargetVersionIncrementalTildeConstraint(t *testing.T) {
	// Test incremental with tilde constraint and no patch available
	_, err := SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{}, "~", true)
	assert.Error(t, err, "Incremental with ~ constraint should error if no patch")
}

// TestSelectTargetVersionNonIncrementalEdgeCases tests edge cases for non-incremental target selection.
//
// It verifies:
//   - Minor flag with no minor or patch returns error
//   - Patch flag with only patch available selects patch
//   - Tilde constraint selects patch
//   - Empty constraint falls back to major
//   - Various flag and constraint combinations
func TestSelectTargetVersionNonIncrementalEdgeCases(t *testing.T) {
	t.Run("minor flag with no minor or patch", func(t *testing.T) {
		_, err := SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{Minor: true}, "", false)
		assert.Error(t, err)
	})

	t.Run("patch flag only patch available", func(t *testing.T) {
		target, err := SelectTargetVersion("#N/A", "#N/A", "1.0.1", UpdateSelectionFlags{Patch: true}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("no flags with tilde constraint", func(t *testing.T) {
		target, err := SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "~", false)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("no flags with no constraint falls back to major", func(t *testing.T) {
		target, err := SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", target)
	})

	t.Run("major flag fallback to patch", func(t *testing.T) {
		target, err := SelectTargetVersion("#N/A", "#N/A", "1.0.1", UpdateSelectionFlags{Major: true}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("no flags with empty constraint minor fallback", func(t *testing.T) {
		target, err := SelectTargetVersion("#N/A", "1.1.0", "#N/A", UpdateSelectionFlags{}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", target)
	})

	t.Run("no flags with empty constraint patch fallback", func(t *testing.T) {
		target, err := SelectTargetVersion("#N/A", "#N/A", "1.0.1", UpdateSelectionFlags{}, "", false)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("no flags with caret constraint patch fallback", func(t *testing.T) {
		target, err := SelectTargetVersion("#N/A", "#N/A", "1.0.1", UpdateSelectionFlags{}, "^", false)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("no flags with tilde constraint no patch error", func(t *testing.T) {
		_, err := SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{}, "~", false)
		assert.Error(t, err)
	})

	t.Run("incremental no flags with empty constraint", func(t *testing.T) {
		target, err := SelectTargetVersion("2.0.0", "1.1.0", "1.0.1", UpdateSelectionFlags{}, "", true)
		require.NoError(t, err)
		assert.Equal(t, "1.0.1", target)
	})

	t.Run("incremental no flags with empty constraint minor fallback", func(t *testing.T) {
		target, err := SelectTargetVersion("2.0.0", "1.1.0", "#N/A", UpdateSelectionFlags{}, "", true)
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", target)
	})

	t.Run("incremental no flags with empty constraint major fallback", func(t *testing.T) {
		target, err := SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{}, "", true)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", target)
	})

	t.Run("incremental no flags with caret no minor or patch error", func(t *testing.T) {
		_, err := SelectTargetVersion("2.0.0", "#N/A", "#N/A", UpdateSelectionFlags{}, "^", true)
		assert.Error(t, err)
	})
}

// TestSummarizeAvailableVersions tests the behavior of SummarizeAvailableVersions.
//
// It verifies:
//   - Correctly identifies major, minor, and patch updates
func TestSummarizeAvailableVersions(t *testing.T) {
	major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"2.0.0", "1.1.0", "1.0.1"}, nil, false)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", major)
	assert.Equal(t, "1.1.0", minor)
	assert.Equal(t, "1.0.1", patch)
}

// TestSummarizeAvailableVersionsIncremental tests incremental mode version summarization.
//
// It verifies:
//   - Incremental mode selects nearest (not newest) versions for each category
func TestSummarizeAvailableVersionsIncremental(t *testing.T) {
	major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"2.0.0", "3.0.0", "1.1.0", "1.2.0", "1.0.1", "1.0.2"}, nil, true)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", major)
	assert.Equal(t, "1.1.0", minor)
	assert.Equal(t, "1.0.1", patch)
}

// TestSummarizeAvailableVersionsInvalidBase tests version summarization with invalid base version.
//
// It verifies:
//   - Returns #N/A for all categories when base version is invalid
func TestSummarizeAvailableVersionsInvalidBase(t *testing.T) {
	major, minor, patch, err := SummarizeAvailableVersions("invalid", []string{"1.0.0"}, nil, false)
	require.NoError(t, err)
	assert.Equal(t, "#N/A", major)
	assert.Equal(t, "#N/A", minor)
	assert.Equal(t, "#N/A", patch)
}

// TestGetVersionCandidates tests the behavior of getVersionCandidates.
//
// It verifies:
//   - Major scope returns correct candidate order for incremental and non-incremental
//   - Minor scope returns correct candidate order for incremental and non-incremental
//   - Patch scope returns only patch candidate
//   - Unknown scope returns nil
func TestGetVersionCandidates(t *testing.T) {
	t.Run("major scope non-incremental", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "major", false)
		assert.Equal(t, []string{"2.0.0", "1.1.0", "1.0.1"}, result)
	})

	t.Run("major scope incremental", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "major", true)
		assert.Equal(t, []string{"1.0.1", "1.1.0", "2.0.0"}, result)
	})

	t.Run("minor scope non-incremental", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "minor", false)
		assert.Equal(t, []string{"1.1.0", "1.0.1"}, result)
	})

	t.Run("minor scope incremental", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "minor", true)
		assert.Equal(t, []string{"1.0.1", "1.1.0"}, result)
	})

	t.Run("patch scope", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "patch", false)
		assert.Equal(t, []string{"1.0.1"}, result)
	})

	t.Run("patch scope incremental same as non-incremental", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "patch", true)
		assert.Equal(t, []string{"1.0.1"}, result)
	})

	t.Run("unknown scope returns nil", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "unknown", false)
		assert.Nil(t, result)
	})

	t.Run("empty scope returns nil", func(t *testing.T) {
		result := getVersionCandidates("2.0.0", "1.1.0", "1.0.1", "", false)
		assert.Nil(t, result)
	})
}

// TestDetermineScope tests the behavior of determineScope.
//
// It verifies:
//   - Flags take precedence over constraints
//   - Empty and star constraints return major scope
//   - Caret constraint returns minor scope
//   - Tilde constraint returns patch scope
//   - Unknown constraints fall back to major scope
func TestDetermineScope(t *testing.T) {
	t.Run("major flag takes precedence", func(t *testing.T) {
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{Major: true}, "~"))
	})

	t.Run("minor flag takes precedence over constraint", func(t *testing.T) {
		assert.Equal(t, "minor", determineScope(UpdateSelectionFlags{Minor: true}, ""))
	})

	t.Run("patch flag takes precedence", func(t *testing.T) {
		assert.Equal(t, "patch", determineScope(UpdateSelectionFlags{Patch: true}, "^"))
	})

	t.Run("empty constraint returns major", func(t *testing.T) {
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{}, ""))
	})

	t.Run("star constraint returns major", func(t *testing.T) {
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{}, "*"))
	})

	t.Run("caret constraint returns minor", func(t *testing.T) {
		assert.Equal(t, "minor", determineScope(UpdateSelectionFlags{}, "^"))
	})

	t.Run("tilde constraint returns patch", func(t *testing.T) {
		assert.Equal(t, "patch", determineScope(UpdateSelectionFlags{}, "~"))
	})

	t.Run("unknown constraint normalizes and falls back to major", func(t *testing.T) {
		// "unknown" normalizes to "=" which doesn't match "", "*", "^", or "~"
		// So it should fall through to return "major"
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{}, "unknown"))
	})

	t.Run("exact constraint falls back to major", func(t *testing.T) {
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{}, "="))
	})

	t.Run("greater than constraint falls back to major", func(t *testing.T) {
		assert.Equal(t, "major", determineScope(UpdateSelectionFlags{}, ">"))
	})
}

// TestSummarizeAvailableVersionsError tests error cases and edge cases for SummarizeAvailableVersions.
//
// It verifies:
//   - Invalid versioning config returns error
//   - Non-incremental mode selects highest versions
//   - Invalid versions in list are skipped
//   - Incremental mode selects nearest versions
//   - Prerelease to stable transitions
//   - Same version not considered update
//   - No updates available returns #N/A
func TestSummarizeAvailableVersionsError(t *testing.T) {
	t.Run("invalid versioning config returns error", func(t *testing.T) {
		cfg := &config.VersioningCfg{Regex: "(invalid regex"}
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"1.1.0"}, cfg, false)
		assert.Error(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "#N/A", patch)
	})

	t.Run("incremental=false uses compare > 0", func(t *testing.T) {
		// With incremental=false, the function should use compare > 0 for isBetterCandidate
		cfg := &config.VersioningCfg{}
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"2.0.0", "3.0.0", "1.1.0", "1.2.0", "1.0.1", "1.0.2"}, cfg, false)
		require.NoError(t, err)
		// Should select the highest versions
		assert.Equal(t, "3.0.0", major)
		assert.Equal(t, "1.2.0", minor)
		assert.Equal(t, "1.0.2", patch)
	})

	t.Run("invalid versions in list are skipped", func(t *testing.T) {
		// Include invalid versions that cannot be parsed
		cfg := &config.VersioningCfg{}
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"not-semver", "2.0.0", "invalid", "1.1.0", "also-not-valid", "1.0.1"}, cfg, false)
		require.NoError(t, err)
		// Only valid versions should be considered
		assert.Equal(t, "2.0.0", major)
		assert.Equal(t, "1.1.0", minor)
		assert.Equal(t, "1.0.1", patch)
	})

	t.Run("incremental mode selects nearest when multiple candidates exist", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// In incremental mode, when multiple major candidates exist (2.0.0, 3.0.0, 4.0.0),
		// it should select the nearest (2.0.0), not the furthest (4.0.0)
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"4.0.0", "2.0.0", "3.0.0", "1.3.0", "1.1.0", "1.2.0", "1.0.3", "1.0.1", "1.0.2"}, cfg, true)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", major)
		assert.Equal(t, "1.1.0", minor)
		assert.Equal(t, "1.0.1", patch)
	})

	t.Run("prerelease to stable transition detected as patch", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// When current is 1.0.0-rc03 and 1.0.0 is available, it should be detected as patch
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0-rc03", []string{"1.0.0"}, cfg, false)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "1.0.0", patch)
	})

	t.Run("prerelease to newer prerelease detected as patch", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// When current is 1.0.0-alpha and 1.0.0-beta is available
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0-alpha", []string{"1.0.0-beta"}, cfg, false)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "1.0.0-beta", patch)
	})

	t.Run("same version not considered update", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// Same version should not be considered an update
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{"1.0.0"}, cfg, false)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "#N/A", patch)
	})

	t.Run("incremental prerelease to stable selects nearest", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// In incremental mode, multiple patch candidates should select the nearest
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0-alpha", []string{"1.0.0", "1.0.0-beta", "1.0.0-rc01"}, cfg, true)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		// Should select nearest: 1.0.0-beta (closer than 1.0.0 or 1.0.0-rc01)
		assert.Equal(t, "1.0.0-beta", patch)
	})

	t.Run("no updates available returns N/A", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		// Only older versions available
		major, minor, patch, err := SummarizeAvailableVersions("2.0.0", []string{"1.0.0", "0.5.0"}, cfg, false)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "#N/A", patch)
	})

	t.Run("empty versions list returns N/A", func(t *testing.T) {
		cfg := &config.VersioningCfg{}
		major, minor, patch, err := SummarizeAvailableVersions("1.0.0", []string{}, cfg, false)
		require.NoError(t, err)
		assert.Equal(t, "#N/A", major)
		assert.Equal(t, "#N/A", minor)
		assert.Equal(t, "#N/A", patch)
	})
}
