package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// TestApplyPackageGroups tests the behavior of ApplyPackageGroups.
//
// It verifies:
//   - Rule-level groups take priority over top-level groups
//   - Top-level groups are assigned when no rule-level group matches
//   - Packages are modified in place with assigned groups
func TestApplyPackageGroups(t *testing.T) {
	cfg := &config.Config{
		Groups: map[string]config.GroupCfg{
			"core": {Packages: []string{"pkg1"}},
		},
		Rules: map[string]config.PackageManagerCfg{
			"rule1": {
				Groups: map[string]config.GroupCfg{
					"utils": {Packages: []string{"pkg2"}},
				},
			},
		},
	}

	pkgs := []formats.Package{
		{Name: "pkg1", Rule: "rule1"},
		{Name: "pkg2", Rule: "rule1"},
		{Name: "pkg3", Rule: "rule1"},
	}

	result := ApplyPackageGroups(pkgs, cfg)

	assert.Equal(t, "utils", result[1].Group) // pkg2 matches rule-level group
	assert.Equal(t, "core", result[0].Group)  // pkg1 matches top-level group
}

// TestApplyPackageGroupsThirdPass tests the third pass of ApplyPackageGroups.
//
// It verifies:
//   - Update config groups are assigned when no other groups match
//   - Packages without rule-level or top-level groups fall back to update config groups
func TestApplyPackageGroupsThirdPass(t *testing.T) {
	// Test the third pass: assign update config groups for remaining packages
	cfg := &config.Config{
		Groups: map[string]config.GroupCfg{},
		Rules: map[string]config.PackageManagerCfg{
			"rule1": {
				Groups: map[string]config.GroupCfg{},
				Update: &config.UpdateCfg{
					Group: "update-group",
				},
			},
		},
	}

	pkgs := []formats.Package{
		{Name: "pkg1", Rule: "rule1"},
	}

	result := ApplyPackageGroups(pkgs, cfg)

	// pkg1 should get the update config group since no other groups matched
	assert.Equal(t, "update-group", result[0].Group)
}

// TestApplyPackageGroupsUnknownRule tests ApplyPackageGroups with unknown rules.
//
// It verifies:
//   - Packages with unknown rules can still match top-level groups
//   - Unknown rules don't cause errors or skip group assignment
func TestApplyPackageGroupsUnknownRule(t *testing.T) {
	// Test when package has unknown rule
	cfg := &config.Config{
		Groups: map[string]config.GroupCfg{
			"core": {Packages: []string{"pkg1"}},
		},
		Rules: map[string]config.PackageManagerCfg{},
	}

	pkgs := []formats.Package{
		{Name: "pkg1", Rule: "unknown-rule"},
	}

	result := ApplyPackageGroups(pkgs, cfg)

	// pkg1 matches top-level group since rule doesn't exist
	assert.Equal(t, "core", result[0].Group)
}

// TestPackageMatchesGroup tests the behavior of PackageMatchesGroup.
//
// It verifies:
//   - Case-insensitive package name matching
//   - Empty package names in config are skipped
//   - Non-matching packages return false
func TestPackageMatchesGroup(t *testing.T) {
	cfg := config.GroupCfg{Packages: []string{"pkg1", "Pkg2", ""}}

	assert.True(t, PackageMatchesGroup(formats.Package{Name: "pkg1"}, cfg))
	assert.True(t, PackageMatchesGroup(formats.Package{Name: "PKG2"}, cfg)) // case insensitive
	assert.False(t, PackageMatchesGroup(formats.Package{Name: "pkg3"}, cfg))
}

// TestSortedGroupKeys tests the behavior of SortedGroupKeys.
//
// It verifies:
//   - Group keys are sorted alphabetically
//   - Empty maps return nil
//   - Nil maps return nil
func TestSortedGroupKeys(t *testing.T) {
	groups := map[string]config.GroupCfg{
		"zebra": {},
		"alpha": {},
		"beta":  {},
	}

	result := SortedGroupKeys(groups)
	assert.Equal(t, []string{"alpha", "beta", "zebra"}, result)

	// Empty map returns nil
	assert.Nil(t, SortedGroupKeys(nil))
	assert.Nil(t, SortedGroupKeys(map[string]config.GroupCfg{}))
}

// TestResolveUpdateGroup tests the behavior of ResolveUpdateGroup.
//
// It verifies:
//   - Nil config returns false with empty group
//   - Empty group in config returns false
//   - Non-empty group returns true with the group name
func TestResolveUpdateGroup(t *testing.T) {
	pkg := formats.Package{Name: "pkg1"}

	t.Run("nil config", func(t *testing.T) {
		group, ok := ResolveUpdateGroup(nil, pkg)
		assert.False(t, ok)
		assert.Empty(t, group)
	})

	t.Run("empty group", func(t *testing.T) {
		cfg := &config.UpdateCfg{}
		group, ok := ResolveUpdateGroup(cfg, pkg)
		assert.False(t, ok)
		assert.Empty(t, group)
	})

	t.Run("with group", func(t *testing.T) {
		cfg := &config.UpdateCfg{Group: "mygroup"}
		group, ok := ResolveUpdateGroup(cfg, pkg)
		assert.True(t, ok)
		assert.Equal(t, "mygroup", group)
	})
}

// TestSortPackagesForDisplay tests the behavior of SortPackagesForDisplay.
//
// It verifies:
//   - Packages are sorted by rule first
//   - Within same rule, sorted by package type
//   - Within same package type, sorted by group (non-empty first)
//   - Within same group, sorted by dependency type
//   - Finally sorted by name
//   - Complex multi-field sorting works correctly
func TestSortPackagesForDisplay(t *testing.T) {
	t.Run("sorts by rule first", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "pkg1", Rule: "rule2"},
			{Name: "pkg2", Rule: "rule1"},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "rule1", result[0].Rule)
		assert.Equal(t, "rule2", result[1].Rule)
	})

	t.Run("sorts by package type when rule is same", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "pkg1", Rule: "rule1", PackageType: "npm"},
			{Name: "pkg2", Rule: "rule1", PackageType: "go"},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "go", result[0].PackageType)
		assert.Equal(t, "npm", result[1].PackageType)
	})

	t.Run("sorts by group when rule and pm are same", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "pkg1", Rule: "rule1", PackageType: "npm", Group: ""},
			{Name: "pkg2", Rule: "rule1", PackageType: "npm", Group: "core"},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "core", result[0].Group) // non-empty group first
		assert.Equal(t, "", result[1].Group)
	})

	t.Run("sorts by type when rule, pm, and group are same", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "pkg1", Rule: "rule1", PackageType: "npm", Group: "core", Type: "prod"},
			{Name: "pkg2", Rule: "rule1", PackageType: "npm", Group: "core", Type: "dev"},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "dev", result[0].Type)
		assert.Equal(t, "prod", result[1].Type)
	})

	t.Run("sorts by name when all else is same", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "zebra", Rule: "rule1", PackageType: "npm", Group: "core", Type: "prod"},
			{Name: "alpha", Rule: "rule1", PackageType: "npm", Group: "core", Type: "prod"},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "alpha", result[0].Name)
		assert.Equal(t, "zebra", result[1].Name)
	})

	t.Run("complex sort with multiple fields", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "pkg2", Rule: "rule1", PackageType: "npm", Type: "dev", Group: ""},
			{Name: "pkg1", Rule: "rule1", PackageType: "npm", Type: "prod", Group: "core"},
			{Name: "pkg3", Rule: "rule2", PackageType: "go", Type: "prod", Group: ""},
		}
		result := SortPackagesForDisplay(pkgs)
		assert.Equal(t, "pkg1", result[0].Name) // rule1, npm, core (has group)
		assert.Equal(t, "pkg2", result[1].Name) // rule1, npm, "" (no group)
		assert.Equal(t, "pkg3", result[2].Name) // rule2
	})
}

// TestCompareGroups tests the behavior of CompareGroups.
//
// It verifies:
//   - Non-empty groups sort before empty groups
//   - Identical groups compare as equal
//   - Alphabetical comparison for different non-empty groups
//   - Whitespace-only groups are treated as empty
func TestCompareGroups(t *testing.T) {
	// Non-empty groups come before empty
	assert.Equal(t, -1, CompareGroups("a", ""))
	assert.Equal(t, 1, CompareGroups("", "a"))

	// Same groups
	assert.Equal(t, 0, CompareGroups("a", "a"))
	assert.Equal(t, 0, CompareGroups("", ""))

	// Alphabetical comparison
	assert.Equal(t, -1, CompareGroups("a", "b"))
	assert.Equal(t, 1, CompareGroups("b", "a"))

	// Whitespace handling
	assert.Equal(t, 0, CompareGroups("  ", ""))
}

// TestHasGroup tests the behavior of HasGroup.
//
// It verifies:
//   - Packages with non-empty groups return true
//   - Packages with empty groups return false
//   - Packages with whitespace-only groups return false
func TestHasGroup(t *testing.T) {
	assert.True(t, HasGroup(formats.Package{Group: "core"}))
	assert.False(t, HasGroup(formats.Package{Group: ""}))
	assert.False(t, HasGroup(formats.Package{Group: "   "}))
}

// TestGroupPackages tests the behavior of GroupPackages.
//
// It verifies:
//   - Packages are grouped by their group name
//   - Multiple packages with same group are grouped together
//   - Empty group names are treated as a separate group
func TestGroupPackages(t *testing.T) {
	pkgs := []formats.Package{
		{Name: "pkg1", Group: "core"},
		{Name: "pkg2", Group: "core"},
		{Name: "pkg3", Group: "utils"},
		{Name: "pkg4", Group: ""},
	}

	result := GroupPackages(pkgs)
	assert.Len(t, result["core"], 2)
	assert.Len(t, result["utils"], 1)
	assert.Len(t, result[""], 1)
}
