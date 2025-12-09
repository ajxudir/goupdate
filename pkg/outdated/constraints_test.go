package outdated

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ajxudir/goupdate/pkg/formats"
)

// TestFilterVersionsByConstraint tests the behavior of FilterVersionsByConstraint.
//
// It verifies:
//   - Compatible (caret) constraint filtering
//   - Patch (tilde) constraint filtering
//   - Flag overrides for major/minor/patch updates
func TestFilterVersionsByConstraint(t *testing.T) {
	tests := []struct {
		name     string
		pkg      formats.Package
		versions []string
		flags    UpdateSelectionFlags
		expected []string
	}{
		{
			name:     "compatible constraint",
			pkg:      formats.Package{Version: "1.0.0", Constraint: "^"},
			versions: []string{"0.9.0", "1.1.0", "2.0.0"},
			flags:    UpdateSelectionFlags{},
			expected: []string{"1.1.0"},
		},
		{
			name:     "patch constraint",
			pkg:      formats.Package{Version: "1.0.0", Constraint: "~"},
			versions: []string{"1.0.1", "1.1.0", "2.0.0"},
			flags:    UpdateSelectionFlags{},
			expected: []string{"1.0.1"},
		},
		{
			name:     "major flag override",
			pkg:      formats.Package{Version: "1.0.0", Constraint: "~", InstalledVersion: "1.0.0"},
			versions: []string{"1.0.1", "1.1.0", "2.0.0"},
			flags:    UpdateSelectionFlags{Major: true},
			expected: []string{"1.0.1", "1.1.0", "2.0.0"},
		},
		{
			name:     "minor flag override",
			pkg:      formats.Package{Version: "1.0.0", Constraint: "~", InstalledVersion: "1.0.0"},
			versions: []string{"1.0.1", "1.1.0", "2.0.0"},
			flags:    UpdateSelectionFlags{Minor: true},
			expected: []string{"1.0.1", "1.1.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterVersionsByConstraint(tt.pkg, tt.versions, tt.flags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeConstraint tests the behavior of NormalizeConstraint.
//
// It verifies:
//   - Normalizes == to =
//   - Normalizes ~= to ~
//   - Normalizes "exact" to =
//   - Returns supported constraints unchanged
//   - Returns = for unknown constraints
func TestNormalizeConstraint(t *testing.T) {
	assert.Equal(t, "=", NormalizeConstraint("=="))
	assert.Equal(t, "~", NormalizeConstraint("~="))
	assert.Equal(t, "=", NormalizeConstraint("exact"))
	assert.Equal(t, "^", NormalizeConstraint("^"))
	assert.Equal(t, "=", NormalizeConstraint("unknown"))
}

// TestIsExactConstraint tests the behavior of IsExactConstraint.
//
// It verifies:
//   - Returns true for = constraint
//   - Returns true for == constraint
//   - Returns true for "exact" constraint
//   - Returns false for other constraints
func TestIsExactConstraint(t *testing.T) {
	assert.True(t, IsExactConstraint("="))
	assert.True(t, IsExactConstraint("=="))
	assert.True(t, IsExactConstraint("exact"))
	assert.False(t, IsExactConstraint("^"))
	assert.False(t, IsExactConstraint("~"))
}

// TestMatchesExactConstraint tests the behavior of matchesExactConstraint.
//
// It verifies:
//   - Returns false for empty reference or candidate
//   - Compares major only for 1 segment
//   - Compares major.minor for 2 segments
//   - Compares full version for 3 segments
func TestMatchesExactConstraint(t *testing.T) {
	t.Run("empty reference returns false", func(t *testing.T) {
		assert.False(t, matchesExactConstraint("", "v1.0.0", 3))
	})

	t.Run("empty candidate returns false", func(t *testing.T) {
		assert.False(t, matchesExactConstraint("v1.0.0", "", 3))
	})

	t.Run("segments 1 compares major only", func(t *testing.T) {
		assert.True(t, matchesExactConstraint("v1.0.0", "v1.5.3", 1))
		assert.False(t, matchesExactConstraint("v1.0.0", "v2.0.0", 1))
	})

	t.Run("segments 2 compares major.minor", func(t *testing.T) {
		assert.True(t, matchesExactConstraint("v1.2.0", "v1.2.5", 2))
		assert.False(t, matchesExactConstraint("v1.2.0", "v1.3.0", 2))
	})

	t.Run("segments 3 compares full version", func(t *testing.T) {
		assert.True(t, matchesExactConstraint("v1.2.3", "v1.2.3", 3))
		assert.False(t, matchesExactConstraint("v1.2.3", "v1.2.4", 3))
	})
}

// TestCountConstraintSegments tests the behavior of countConstraintSegments.
//
// It verifies:
//   - Counts single, two, and three segment versions
//   - Handles prerelease versions
//   - Handles v prefix
//   - Returns 0 for empty or #N/A versions
//   - Caps at 3 for versions with more than 3 segments
//   - Returns 0 for whitespace-only input
func TestCountConstraintSegments(t *testing.T) {
	t.Run("single segment", func(t *testing.T) {
		assert.Equal(t, 1, countConstraintSegments("1"))
	})

	t.Run("two segments", func(t *testing.T) {
		assert.Equal(t, 2, countConstraintSegments("1.0"))
	})

	t.Run("three segments", func(t *testing.T) {
		assert.Equal(t, 3, countConstraintSegments("1.0.0"))
	})

	t.Run("with prerelease", func(t *testing.T) {
		assert.Equal(t, 3, countConstraintSegments("1.0.0-beta"))
	})

	t.Run("with v prefix", func(t *testing.T) {
		assert.Equal(t, 3, countConstraintSegments("v1.0.0"))
	})

	t.Run("empty version", func(t *testing.T) {
		assert.Equal(t, 0, countConstraintSegments(""))
	})

	t.Run("N/A version returns 0", func(t *testing.T) {
		assert.Equal(t, 0, countConstraintSegments("#N/A"))
	})

	t.Run("more than 3 segments caps at 3", func(t *testing.T) {
		assert.Equal(t, 3, countConstraintSegments("1.2.3.4"))
		assert.Equal(t, 3, countConstraintSegments("1.2.3.4.5.6"))
	})

	t.Run("whitespace only version", func(t *testing.T) {
		assert.Equal(t, 0, countConstraintSegments("   "))
	})
}

// TestIsFullyPinnedVersion tests the behavior of IsFullyPinnedVersion.
//
// It verifies:
//   - Three segment versions are pinned
//   - Versions with more than three segments are pinned
//   - One or two segment versions are not pinned
//   - Empty or #N/A versions are not pinned
//   - Prerelease versions with three segments are pinned
func TestIsFullyPinnedVersion(t *testing.T) {
	t.Run("three segments is pinned", func(t *testing.T) {
		assert.True(t, IsFullyPinnedVersion("1.0.0"))
		assert.True(t, IsFullyPinnedVersion("v1.0.0"))
		assert.True(t, IsFullyPinnedVersion("1.2.3"))
	})

	t.Run("more than three segments is pinned", func(t *testing.T) {
		assert.True(t, IsFullyPinnedVersion("1.0.0.0"))
		assert.True(t, IsFullyPinnedVersion("1.2.3.4.5"))
	})

	t.Run("two segments is not pinned", func(t *testing.T) {
		assert.False(t, IsFullyPinnedVersion("1.0"))
		assert.False(t, IsFullyPinnedVersion("v1.0"))
	})

	t.Run("one segment is not pinned", func(t *testing.T) {
		assert.False(t, IsFullyPinnedVersion("1"))
		assert.False(t, IsFullyPinnedVersion("v1"))
	})

	t.Run("empty version is not pinned", func(t *testing.T) {
		assert.False(t, IsFullyPinnedVersion(""))
		assert.False(t, IsFullyPinnedVersion("#N/A"))
	})

	t.Run("prerelease versions are pinned if three segments", func(t *testing.T) {
		assert.True(t, IsFullyPinnedVersion("1.0.0-alpha"))
		assert.True(t, IsFullyPinnedVersion("1.0.0-rc.1"))
		assert.True(t, IsFullyPinnedVersion("1.0.0-beta.2+build.123"))
	})
}

// TestFilterVersionsByConstraintEdgeCases tests edge cases for FilterVersionsByConstraint.
//
// It verifies:
//   - Exact constraint filtering
//   - Patch flag with tilde constraint
//   - Empty versions list returns empty
//   - Greater/less than constraint filtering
//   - Constraint behavior with empty references
//   - Unknown and star constraint normalization
//   - Non-semver version handling
//   - Flag overrides with installed versions
func TestFilterVersionsByConstraintEdgeCases(t *testing.T) {
	t.Run("exact constraint filters to matching", func(t *testing.T) {
		pkg := formats.Package{Version: "1.2.0", Constraint: "="}
		versions := []string{"1.2.0", "1.2.1", "1.3.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.2.0"}, result)
	})

	t.Run("patch flag with tilde constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "~", InstalledVersion: "1.0.0"}
		versions := []string{"1.0.1", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{Patch: true})
		assert.Equal(t, []string{"1.0.1"}, result)
	})

	t.Run("empty versions returns empty", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "^"}
		result := FilterVersionsByConstraint(pkg, []string{}, UpdateSelectionFlags{})
		assert.Empty(t, result)
	})

	t.Run("greater or equal constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: ">="}
		versions := []string{"0.9.0", "1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, result)
	})

	t.Run("greater constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: ">"}
		versions := []string{"0.9.0", "1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.1.0", "2.0.0"}, result)
	})

	t.Run("less or equal constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.5.0", Constraint: "<="}
		versions := []string{"1.0.0", "1.5.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "1.5.0"}, result)
	})

	t.Run("less than constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "2.0.0", Constraint: "<"}
		versions := []string{"1.0.0", "1.5.0", "2.0.0", "3.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "1.5.0"}, result)
	})

	t.Run("tilde constraint with empty reference", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "~"}
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// All versions allowed when reference is empty
		assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, result)
	})

	t.Run("exact constraint with empty reference", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "="}
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// All versions allowed when reference is empty
		assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, result)
	})

	t.Run("unknown constraint normalizes to exact", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "unknown"}
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// Unknown constraint normalizes to "=" (exact), so only matching version allowed
		assert.Equal(t, []string{"1.0.0"}, result)
	})

	t.Run("star constraint normalizes to empty", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "*"}
		versions := []string{"1.0.0", "2.0.0", "3.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// All versions allowed with * constraint
		assert.Equal(t, []string{"1.0.0", "2.0.0", "3.0.0"}, result)
	})

	t.Run("invalid semver versions are skipped", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "^"}
		versions := []string{"1.0.0", "invalid", "not-a-version", "1.1.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "1.1.0"}, result)
	})

	t.Run("major flag overrides constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "~", InstalledVersion: "1.0.0"}
		versions := []string{"1.0.1", "1.1.0", "2.0.0", "3.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{Major: true})
		// All versions allowed with major flag
		assert.Contains(t, result, "2.0.0")
		assert.Contains(t, result, "3.0.0")
	})

	t.Run("minor flag uses caret constraint", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "=", InstalledVersion: "1.0.0"}
		versions := []string{"1.0.1", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{Minor: true})
		assert.Equal(t, []string{"1.0.1", "1.1.0"}, result)
	})

	t.Run("empty version uses installed version for reference", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "^", InstalledVersion: "1.0.0"}
		versions := []string{"0.9.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.1.0"}, result)
	})

	t.Run("flag overrides use installed version as reference", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "^", InstalledVersion: "1.5.0"}
		versions := []string{"1.1.0", "1.4.0", "1.6.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{Minor: true})
		// With minor flag and InstalledVersion=1.5.0, should allow 1.x.x versions
		assert.Contains(t, result, "1.1.0")
		assert.Contains(t, result, "1.4.0")
		assert.Contains(t, result, "1.6.0")
		assert.NotContains(t, result, "2.0.0")
	})

	t.Run("non-semver versions excluded when constraint requires filtering", func(t *testing.T) {
		pkg := formats.Package{Version: "1.0.0", Constraint: "^"}
		versions := []string{"1.1.0", "1.0.0.0.1", "next-release"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// 1.1.0 is valid semver within ^1.0.0 constraint
		// 1.0.0.0.1 and next-release are non-semver and should be excluded when constraint is set
		assert.Equal(t, []string{"1.1.0"}, result)
	})

	t.Run("caret constraint with empty reference allows all semver", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "^", InstalledVersion: ""}
		versions := []string{"1.0.0", "2.0.0", "3.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "2.0.0", "3.0.0"}, result)
	})

	t.Run("greater than or equal with empty reference allows all semver", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: ">=", InstalledVersion: ""}
		versions := []string{"1.0.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, result)
	})

	t.Run("greater than with empty reference allows all semver", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: ">", InstalledVersion: ""}
		versions := []string{"1.0.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, result)
	})

	t.Run("less than or equal with empty reference allows all semver", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "<=", InstalledVersion: ""}
		versions := []string{"1.0.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, result)
	})

	t.Run("less than with empty reference allows all semver", func(t *testing.T) {
		pkg := formats.Package{Version: "", Constraint: "<", InstalledVersion: ""}
		versions := []string{"1.0.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, result)
	})

	t.Run("empty version uses installed version for segments counting", func(t *testing.T) {
		// Tests the constraintSegments == 0 branch inside reference == "" block
		// p.Version is empty, so reference comes from InstalledVersion
		// and constraintSegments should also come from InstalledVersion
		pkg := formats.Package{Version: "", Constraint: "=", InstalledVersion: "1.0"}
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{})
		// With InstalledVersion="1.0" (2 segments), exact match should allow 1.0.x
		assert.Contains(t, result, "1.0.0")
		assert.NotContains(t, result, "1.1.0")
		assert.NotContains(t, result, "2.0.0")
	})

	t.Run("patch flag with currentVersion for exact match", func(t *testing.T) {
		// Tests the flags.Patch branch that uses currentVersion
		pkg := formats.Package{Version: "1.0.0", Constraint: "=", InstalledVersion: "1.0.5"}
		versions := []string{"1.0.1", "1.0.6", "1.1.0", "2.0.0"}
		result := FilterVersionsByConstraint(pkg, versions, UpdateSelectionFlags{Patch: true})
		// Patch flag + InstalledVersion=1.0.5 should allow 1.0.x only
		assert.Contains(t, result, "1.0.1")
		assert.Contains(t, result, "1.0.6")
		assert.NotContains(t, result, "1.1.0")
		assert.NotContains(t, result, "2.0.0")
	})
}
