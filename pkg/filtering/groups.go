package filtering

import (
	"sort"
	"strings"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// ApplyPackageGroups assigns group names to packages based on config rules.
//
// Groups are assigned in order of priority:
// 1. Rule-level groups (rules.<rule>.groups)
// 2. Top-level groups (groups)
// 3. Update config groups (rules.<rule>.update.group)
//
// Parameters:
//   - pkgs: Slice of packages to assign groups to
//   - cfg: Configuration containing group definitions
//
// Returns:
//   - []formats.Package: Packages with groups assigned (modified in place)
//
// Example:
//
//	packages = filtering.ApplyPackageGroups(packages, cfg)
func ApplyPackageGroups(pkgs []formats.Package, cfg *config.Config) []formats.Package {
	// Build rule-level group keys map
	groupKeysByRule := make(map[string][]string, len(cfg.Rules))
	for ruleKey, ruleCfg := range cfg.Rules {
		if len(ruleCfg.Groups) == 0 {
			continue
		}
		groupKeysByRule[ruleKey] = SortedGroupKeys(ruleCfg.Groups)
	}

	topLevelGroups := SortedGroupKeys(cfg.Groups)

	// First pass: assign rule-level groups
	for i := range pkgs {
		ruleCfg, ok := cfg.Rules[pkgs[i].Rule]
		if !ok {
			continue
		}

		for _, groupID := range groupKeysByRule[pkgs[i].Rule] {
			if PackageMatchesGroup(pkgs[i], ruleCfg.Groups[groupID]) {
				pkgs[i].Group = groupID
				break
			}
		}
	}

	// Second pass: assign top-level groups for packages without a group
	for i := range pkgs {
		if strings.TrimSpace(pkgs[i].Group) != "" {
			continue
		}

		for _, groupID := range topLevelGroups {
			if PackageMatchesGroup(pkgs[i], cfg.Groups[groupID]) {
				pkgs[i].Group = groupID
				break
			}
		}
	}

	// Third pass: assign update config groups for remaining packages
	for i := range pkgs {
		ruleCfg, ok := cfg.Rules[pkgs[i].Rule]
		if !ok {
			continue
		}

		if strings.TrimSpace(pkgs[i].Group) != "" {
			continue
		}

		if group, ok := ResolveUpdateGroup(ruleCfg.Update, pkgs[i]); ok {
			pkgs[i].Group = group
		}
	}

	return pkgs
}

// PackageMatchesGroup checks if a package matches a group configuration.
//
// Matching is case-insensitive.
//
// Parameters:
//   - p: Package to check
//   - cfg: Group configuration with package list
//
// Returns:
//   - bool: true if package name matches any name in the group
func PackageMatchesGroup(p formats.Package, cfg config.GroupCfg) bool {
	for _, name := range cfg.Packages {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}

		if strings.EqualFold(trimmed, p.Name) {
			return true
		}
	}

	return false
}

// SortedGroupKeys returns sorted keys from a groups map.
//
// Returns keys in alphabetical order for deterministic group assignment.
//
// Parameters:
//   - groups: Map of group configurations
//
// Returns:
//   - []string: Sorted group keys in alphabetical order; nil if groups is empty
//
// Example:
//
//	groups := map[string]config.GroupCfg{
//	    "backend": {},
//	    "frontend": {},
//	}
//	keys := filtering.SortedGroupKeys(groups)  // ["backend", "frontend"]
func SortedGroupKeys(groups map[string]config.GroupCfg) []string {
	if len(groups) == 0 {
		return nil
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

// ResolveUpdateGroup returns the group from update config if set.
//
// Parameters:
//   - updateCfg: Update configuration, may be nil
//   - p: Package (unused, for future extensions)
//
// Returns:
//   - string: Group name
//   - bool: true if a group was found
func ResolveUpdateGroup(updateCfg *config.UpdateCfg, _ formats.Package) (string, bool) {
	if updateCfg == nil {
		return "", false
	}

	if updateCfg.Group != "" {
		return updateCfg.Group, true
	}

	return "", false
}

// SortPackagesForDisplay returns packages sorted for display output.
//
// Sort order:
// 1. Rule (alphabetical)
// 2. Package type (alphabetical)
// 3. Group (grouped packages before ungrouped)
// 4. Dependency type (alphabetical)
// 5. Name (alphabetical)
//
// Parameters:
//   - pkgs: Packages to sort
//
// Returns:
//   - []formats.Package: Sorted copy of packages
func SortPackagesForDisplay(pkgs []formats.Package) []formats.Package {
	sorted := append([]formats.Package(nil), pkgs...)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Rule != sorted[j].Rule {
			return sorted[i].Rule < sorted[j].Rule
		}
		if sorted[i].PackageType != sorted[j].PackageType {
			return sorted[i].PackageType < sorted[j].PackageType
		}
		if cmp := CompareGroups(sorted[i].Group, sorted[j].Group); cmp != 0 {
			return cmp < 0
		}
		if sorted[i].Type != sorted[j].Type {
			return sorted[i].Type < sorted[j].Type
		}
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// CompareGroups compares two group names for sorting.
//
// Packages with groups sort before packages without groups.
//
// Parameters:
//   - a: First group name
//   - b: Second group name
//
// Returns:
//   - int: -1 if a < b, 0 if equal, 1 if a > b
func CompareGroups(a, b string) int {
	aVal := strings.TrimSpace(a)
	bVal := strings.TrimSpace(b)

	aHas := aVal != ""
	bHas := bVal != ""

	// Groups with names sort before empty groups
	if aHas && !bHas {
		return -1
	}
	if bHas && !aHas {
		return 1
	}

	if aVal == bVal {
		return 0
	}

	if aVal < bVal {
		return -1
	}

	return 1
}

// HasGroup returns true if the package has a non-empty group.
//
// Parameters:
//   - p: Package to check
//
// Returns:
//   - bool: true if package has a group assigned
func HasGroup(p formats.Package) bool {
	return strings.TrimSpace(p.Group) != ""
}

// GroupPackages groups packages by their group name.
//
// Parameters:
//   - pkgs: Packages to group
//
// Returns:
//   - map[string][]formats.Package: Packages grouped by group name (empty string for ungrouped)
func GroupPackages(pkgs []formats.Package) map[string][]formats.Package {
	grouped := make(map[string][]formats.Package)
	for _, p := range pkgs {
		grouped[p.Group] = append(grouped[p.Group], p)
	}
	return grouped
}
