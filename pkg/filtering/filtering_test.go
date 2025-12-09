package filtering

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ajxudir/goupdate/pkg/formats"
)

// TestFilterPackagesWithFilters tests the FilterPackagesWithFilters function.
//
// Parameters:
//   - pkgs: Slice of packages to filter
//   - typeFilter: Package type filter ("all" or specific type)
//   - pmFilter: Package manager filter ("all" or specific PM)
//   - ruleFilter: Rule name filter ("all" or specific rule)
//   - nameFilter: Package name filter (empty string for all)
//   - groupFilter: Group name filter (empty string for all)
//
// It verifies that:
//   - No filters returns all packages
//   - Filtering by type, package manager, rule, name, and group works
//   - Multiple filters can be combined
func TestFilterPackagesWithFilters(t *testing.T) {
	pkgs := []formats.Package{
		{Name: "pkg1", Type: "prod", PackageType: "npm", Rule: "rule1", Group: "core"},
		{Name: "pkg2", Type: "dev", PackageType: "npm", Rule: "rule1", Group: ""},
		{Name: "pkg3", Type: "prod", PackageType: "go", Rule: "rule2", Group: "utils"},
	}

	t.Run("no filters returns all", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "all", "all", "all", "", "")
		assert.Len(t, result, 3)
	})

	t.Run("filter by type", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "prod", "all", "all", "", "")
		assert.Len(t, result, 2)
	})

	t.Run("filter by package manager", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "all", "npm", "all", "", "")
		assert.Len(t, result, 2)
	})

	t.Run("filter by rule", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "all", "all", "rule2", "", "")
		assert.Len(t, result, 1)
	})

	t.Run("filter by name", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "all", "all", "all", "pkg1", "")
		assert.Len(t, result, 1)
	})

	t.Run("filter by group", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "all", "all", "all", "", "core")
		assert.Len(t, result, 1)
	})

	t.Run("multiple filters", func(t *testing.T) {
		result := FilterPackagesWithFilters(pkgs, "prod", "npm", "all", "", "")
		assert.Len(t, result, 1)
		assert.Equal(t, "pkg1", result[0].Name)
	})
}

// TestMatchesType tests the MatchesType function.
//
// Parameters:
//   - pkg: Package to check
//   - filter: Type filter string ("all" or specific type)
//   - allowedTypes: Slice of allowed type values
//
// It verifies that:
//   - "all" filter matches any type
//   - Specific filter matches only that type
//   - Non-matching filter returns false
func TestMatchesType(t *testing.T) {
	pkg := formats.Package{Type: "prod"}

	assert.True(t, MatchesType(pkg, "all", []string{}))
	assert.True(t, MatchesType(pkg, "prod", []string{"prod"}))
	assert.False(t, MatchesType(pkg, "dev", []string{"dev"}))
}

// TestMatchesPM tests the MatchesPM function.
//
// Parameters:
//   - pkg: Package to check
//   - filter: Package manager filter string ("all" or specific PM)
//   - allowedPMs: Slice of allowed PM values
//
// It verifies that:
//   - "all" filter matches any PM
//   - Specific filter matches only that PM
//   - Non-matching filter returns false
func TestMatchesPM(t *testing.T) {
	pkg := formats.Package{PackageType: "npm"}

	assert.True(t, MatchesPM(pkg, "all", []string{}))
	assert.True(t, MatchesPM(pkg, "npm", []string{"npm"}))
	assert.False(t, MatchesPM(pkg, "go", []string{"go"}))
}

// TestMatchesRule tests the MatchesRule function.
//
// Parameters:
//   - pkg: Package to check
//   - filter: Rule name filter string ("all" or specific rule)
//   - allowedRules: Slice of allowed rule values
//
// It verifies that:
//   - "all" filter matches any rule
//   - Specific filter matches only that rule
//   - Non-matching filter returns false
func TestMatchesRule(t *testing.T) {
	pkg := formats.Package{Rule: "rule1"}

	assert.True(t, MatchesRule(pkg, "all", []string{}))
	assert.True(t, MatchesRule(pkg, "rule1", []string{"rule1"}))
	assert.False(t, MatchesRule(pkg, "rule2", []string{"rule2"}))
}

// TestMatchesName tests the MatchesName function.
//
// Parameters:
//   - pkg: Package to check
//   - filter: Name filter string (empty for all, case-insensitive match)
//   - allowedNames: Slice of allowed name values
//
// It verifies that:
//   - Empty filter matches any name
//   - Case-insensitive matching works
//   - Non-matching filter returns false
func TestMatchesName(t *testing.T) {
	pkg := formats.Package{Name: "MyPackage"}

	assert.True(t, MatchesName(pkg, "", []string{}))
	assert.True(t, MatchesName(pkg, "mypackage", []string{"mypackage"}))
	assert.False(t, MatchesName(pkg, "other", []string{"other"}))
}

// TestMatchesGroup tests the MatchesGroup function.
//
// Parameters:
//   - pkg: Package to check
//   - filter: Group name filter string (empty for all)
//   - allowedGroups: Slice of allowed group values
//
// It verifies that:
//   - Empty filter matches any group
//   - Specific filter matches only that group
//   - Non-matching filter returns false
func TestMatchesGroup(t *testing.T) {
	pkg := formats.Package{Group: "core"}

	assert.True(t, MatchesGroup(pkg, "", []string{}))
	assert.True(t, MatchesGroup(pkg, "core", []string{"core"}))
	assert.False(t, MatchesGroup(pkg, "utils", []string{"utils"}))
}

// TestFilterByGroup tests the FilterByGroup function.
//
// Parameters:
//   - pkgs: Slice of packages to filter
//   - group: Group name to filter by (empty for all)
//
// It verifies that:
//   - Empty filter returns all packages
//   - Specific group filter returns only matching packages
func TestFilterByGroup(t *testing.T) {
	pkgs := []formats.Package{
		{Name: "pkg1", Group: "core"},
		{Name: "pkg2", Group: "utils"},
		{Name: "pkg3", Group: ""},
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		result := FilterByGroup(pkgs, "")
		assert.Len(t, result, 3)
	})

	t.Run("filter by group", func(t *testing.T) {
		result := FilterByGroup(pkgs, "core")
		assert.Len(t, result, 1)
		assert.Equal(t, "pkg1", result[0].Name)
	})

	t.Run("whitespace only filter returns all", func(t *testing.T) {
		// After TrimAndSplit, "   " becomes empty slice, triggering len(groupFilters) == 0
		result := FilterByGroup(pkgs, "   ")
		assert.Len(t, result, 3)
	})
}

// Tests for interfaces.go

// TestOptionsFilter tests the OptionsFilter struct and its Filter method.
//
// It verifies that:
//   - Filter method applies all filter options
//   - Empty options returns all packages
func TestOptionsFilter(t *testing.T) {
	pkgs := []formats.Package{
		{Name: "pkg1", Type: "prod", PackageType: "npm", Rule: "rule1"},
		{Name: "pkg2", Type: "dev", PackageType: "npm", Rule: "rule1"},
		{Name: "pkg3", Type: "prod", PackageType: "go", Rule: "rule2"},
	}

	t.Run("Filter method", func(t *testing.T) {
		filter := &OptionsFilter{
			Options: FilterOptions{
				Type: "prod",
				PM:   "npm",
			},
		}
		result := filter.Filter(pkgs)
		assert.Len(t, result, 1)
		assert.Equal(t, "pkg1", result[0].Name)
	})

	t.Run("empty options returns all", func(t *testing.T) {
		filter := &OptionsFilter{Options: FilterOptions{}}
		result := filter.Filter(pkgs)
		assert.Len(t, result, 3)
	})
}

// Tests for options.go

// TestFilterOptionsIsEmpty tests the IsEmpty method of FilterOptions.
//
// It verifies that:
//   - Empty struct returns true
//   - All "all" values returns true
//   - Any specific filter returns false
func TestFilterOptionsIsEmpty(t *testing.T) {
	t.Run("empty options", func(t *testing.T) {
		opts := FilterOptions{}
		assert.True(t, opts.IsEmpty())
	})

	t.Run("all filters set to 'all'", func(t *testing.T) {
		opts := FilterOptions{Type: "all", PM: "all", Rule: "all"}
		assert.True(t, opts.IsEmpty())
	})

	t.Run("with type filter", func(t *testing.T) {
		opts := FilterOptions{Type: "prod"}
		assert.False(t, opts.IsEmpty())
	})

	t.Run("with pm filter", func(t *testing.T) {
		opts := FilterOptions{PM: "npm"}
		assert.False(t, opts.IsEmpty())
	})

	t.Run("with rule filter", func(t *testing.T) {
		opts := FilterOptions{Rule: "frontend"}
		assert.False(t, opts.IsEmpty())
	})

	t.Run("with name filter", func(t *testing.T) {
		opts := FilterOptions{Name: "lodash"}
		assert.False(t, opts.IsEmpty())
	})

	t.Run("with group filter", func(t *testing.T) {
		opts := FilterOptions{Group: "core"}
		assert.False(t, opts.IsEmpty())
	})

	t.Run("with file filter", func(t *testing.T) {
		opts := FilterOptions{File: "*.json"}
		assert.False(t, opts.IsEmpty())
	})
}

// TestFilterOptionsHasTypeFilter tests the HasTypeFilter method.
//
// It verifies that:
//   - Empty or "all" Type returns false
//   - Specific Type value returns true
func TestFilterOptionsHasTypeFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasTypeFilter())
	assert.False(t, FilterOptions{Type: ""}.HasTypeFilter())
	assert.False(t, FilterOptions{Type: "all"}.HasTypeFilter())
	assert.True(t, FilterOptions{Type: "prod"}.HasTypeFilter())
	assert.True(t, FilterOptions{Type: "dev"}.HasTypeFilter())
}

// TestFilterOptionsHasPMFilter tests the HasPMFilter method.
//
// It verifies that:
//   - Empty or "all" PM returns false
//   - Specific PM value returns true
func TestFilterOptionsHasPMFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasPMFilter())
	assert.False(t, FilterOptions{PM: ""}.HasPMFilter())
	assert.False(t, FilterOptions{PM: "all"}.HasPMFilter())
	assert.True(t, FilterOptions{PM: "npm"}.HasPMFilter())
	assert.True(t, FilterOptions{PM: "go"}.HasPMFilter())
}

// TestFilterOptionsHasRuleFilter tests the HasRuleFilter method.
//
// It verifies that:
//   - Empty or "all" Rule returns false
//   - Specific Rule value returns true
func TestFilterOptionsHasRuleFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasRuleFilter())
	assert.False(t, FilterOptions{Rule: ""}.HasRuleFilter())
	assert.False(t, FilterOptions{Rule: "all"}.HasRuleFilter())
	assert.True(t, FilterOptions{Rule: "frontend"}.HasRuleFilter())
}

// TestFilterOptionsHasNameFilter tests the HasNameFilter method.
//
// It verifies that:
//   - Empty Name returns false
//   - Specific Name value returns true
func TestFilterOptionsHasNameFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasNameFilter())
	assert.False(t, FilterOptions{Name: ""}.HasNameFilter())
	assert.True(t, FilterOptions{Name: "lodash"}.HasNameFilter())
}

// TestFilterOptionsHasGroupFilter tests the HasGroupFilter method.
//
// It verifies that:
//   - Empty Group returns false
//   - Specific Group value returns true
func TestFilterOptionsHasGroupFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasGroupFilter())
	assert.False(t, FilterOptions{Group: ""}.HasGroupFilter())
	assert.True(t, FilterOptions{Group: "core"}.HasGroupFilter())
}

// TestFilterOptionsHasFileFilter tests the HasFileFilter method.
//
// It verifies that:
//   - Empty File returns false
//   - Specific File pattern returns true
func TestFilterOptionsHasFileFilter(t *testing.T) {
	assert.False(t, FilterOptions{}.HasFileFilter())
	assert.False(t, FilterOptions{File: ""}.HasFileFilter())
	assert.True(t, FilterOptions{File: "*.json"}.HasFileFilter())
}

// TestFromFlagsWithFile tests the FromFlagsWithFile constructor.
//
// Parameters:
//   - typeFilter: Package type filter
//   - pmFilter: Package manager filter
//   - ruleFilter: Rule name filter
//   - nameFilter: Package name filter
//   - groupFilter: Group name filter
//   - fileFilter: File pattern filter
//
// It verifies that:
//   - All parameters are correctly assigned to FilterOptions fields
func TestFromFlagsWithFile(t *testing.T) {
	opts := FromFlagsWithFile("prod", "npm", "rule1", "pkg", "group", "*.json")
	assert.Equal(t, "prod", opts.Type)
	assert.Equal(t, "npm", opts.PM)
	assert.Equal(t, "rule1", opts.Rule)
	assert.Equal(t, "pkg", opts.Name)
	assert.Equal(t, "group", opts.Group)
	assert.Equal(t, "*.json", opts.File)
}

// TestFilterOptionsWithType tests the WithType builder method.
//
// Parameters:
//   - t: Type value to set
//
// It verifies that:
//   - Type field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithType(t *testing.T) {
	opts := FilterOptions{PM: "npm"}
	newOpts := opts.WithType("prod")
	assert.Equal(t, "prod", newOpts.Type)
	assert.Equal(t, "npm", newOpts.PM) // Original field preserved
}

// TestFilterOptionsWithPM tests the WithPM builder method.
//
// Parameters:
//   - pm: Package manager value to set
//
// It verifies that:
//   - PM field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithPM(t *testing.T) {
	opts := FilterOptions{Type: "prod"}
	newOpts := opts.WithPM("npm")
	assert.Equal(t, "npm", newOpts.PM)
	assert.Equal(t, "prod", newOpts.Type)
}

// TestFilterOptionsWithRule tests the WithRule builder method.
//
// Parameters:
//   - rule: Rule name value to set
//
// It verifies that:
//   - Rule field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithRule(t *testing.T) {
	opts := FilterOptions{Type: "prod"}
	newOpts := opts.WithRule("frontend")
	assert.Equal(t, "frontend", newOpts.Rule)
	assert.Equal(t, "prod", newOpts.Type)
}

// TestFilterOptionsWithName tests the WithName builder method.
//
// Parameters:
//   - name: Package name value to set
//
// It verifies that:
//   - Name field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithName(t *testing.T) {
	opts := FilterOptions{Type: "prod"}
	newOpts := opts.WithName("lodash")
	assert.Equal(t, "lodash", newOpts.Name)
	assert.Equal(t, "prod", newOpts.Type)
}

// TestFilterOptionsWithGroup tests the WithGroup builder method.
//
// Parameters:
//   - group: Group name value to set
//
// It verifies that:
//   - Group field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithGroup(t *testing.T) {
	opts := FilterOptions{Type: "prod"}
	newOpts := opts.WithGroup("core")
	assert.Equal(t, "core", newOpts.Group)
	assert.Equal(t, "prod", newOpts.Type)
}

// TestFilterOptionsWithFile tests the WithFile builder method.
//
// Parameters:
//   - file: File pattern value to set
//
// It verifies that:
//   - File field is set correctly
//   - Other fields are preserved
func TestFilterOptionsWithFile(t *testing.T) {
	opts := FilterOptions{Type: "prod"}
	newOpts := opts.WithFile("*.json")
	assert.Equal(t, "*.json", newOpts.File)
	assert.Equal(t, "prod", newOpts.Type)
}
