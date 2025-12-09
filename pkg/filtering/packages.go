package filtering

import (
	"github.com/user/goupdate/pkg/formats"
	"github.com/user/goupdate/pkg/utils"
)

// FilterPackages filters packages based on the provided options.
//
// Applies all filters in sequence: type, pm, rule, name, group.
// Packages must match ALL specified filters to be included.
//
// Parameters:
//   - pkgs: Slice of packages to filter
//   - opts: Filter options
//
// Returns:
//   - []formats.Package: Filtered packages
//
// Example:
//
//	opts := filtering.FilterOptions{Type: "prod", PM: "npm"}
//	filtered := filtering.FilterPackages(packages, opts)
func FilterPackages(pkgs []formats.Package, opts FilterOptions) []formats.Package {
	parsed := opts.Parse()
	var filtered []formats.Package

	for _, p := range pkgs {
		if !matchesType(p, opts.Type, parsed.types) {
			continue
		}
		if !matchesPM(p, opts.PM, parsed.pms) {
			continue
		}
		if !matchesRule(p, opts.Rule, parsed.rules) {
			continue
		}
		if !matchesName(p, opts.Name, parsed.names) {
			continue
		}
		if !matchesGroup(p, opts.Group, parsed.groups) {
			continue
		}
		filtered = append(filtered, p)
	}

	return filtered
}

// FilterPackagesWithFilters filters packages based on type, package manager, rule, name, and group flags.
//
// This is a convenience function that creates FilterOptions from individual flag values.
//
// Parameters:
//   - pkgs: Slice of packages to filter
//   - typeFlag: Type filter (prod, dev, all)
//   - pmFlag: Package manager filter
//   - ruleFlag: Rule filter
//   - nameFlag: Name filter
//   - groupFlag: Group filter
//
// Returns:
//   - []formats.Package: Filtered packages
func FilterPackagesWithFilters(pkgs []formats.Package, typeFlag, pmFlag, ruleFlag, nameFlag, groupFlag string) []formats.Package {
	opts := FromFlags(typeFlag, pmFlag, ruleFlag, nameFlag, groupFlag)
	return FilterPackages(pkgs, opts)
}

// MatchesType checks if a package matches the type filter.
//
// Parameters:
//   - p: Package to check
//   - typeFlag: Original type flag value
//   - filters: Parsed type filters
//
// Returns:
//   - bool: true if package matches
func MatchesType(p formats.Package, typeFlag string, filters []string) bool {
	return matchesType(p, typeFlag, filters)
}

// matchesType is the internal implementation.
func matchesType(p formats.Package, typeFlag string, filters []string) bool {
	if typeFlag == FilterAll || len(filters) == 0 {
		return true
	}
	return utils.Contains(filters, p.Type)
}

// MatchesPM checks if a package matches the package manager filter.
//
// Parameters:
//   - p: Package to check
//   - pmFlag: Original pm flag value
//   - filters: Parsed pm filters
//
// Returns:
//   - bool: true if package matches
func MatchesPM(p formats.Package, pmFlag string, filters []string) bool {
	return matchesPM(p, pmFlag, filters)
}

// matchesPM is the internal implementation.
func matchesPM(p formats.Package, pmFlag string, filters []string) bool {
	if pmFlag == FilterAll || len(filters) == 0 {
		return true
	}
	return utils.Contains(filters, p.PackageType)
}

// MatchesRule checks if a package matches the rule filter.
//
// Parameters:
//   - p: Package to check
//   - ruleFlag: Original rule flag value
//   - filters: Parsed rule filters
//
// Returns:
//   - bool: true if package matches
func MatchesRule(p formats.Package, ruleFlag string, filters []string) bool {
	return matchesRule(p, ruleFlag, filters)
}

// matchesRule is the internal implementation.
func matchesRule(p formats.Package, ruleFlag string, filters []string) bool {
	if ruleFlag == FilterAll || len(filters) == 0 {
		return true
	}
	return utils.Contains(filters, p.Rule)
}

// MatchesName checks if a package matches the name filter.
//
// Name matching is case-insensitive.
//
// Parameters:
//   - p: Package to check
//   - nameFlag: Original name flag value
//   - filters: Parsed name filters
//
// Returns:
//   - bool: true if package matches
func MatchesName(p formats.Package, nameFlag string, filters []string) bool {
	return matchesName(p, nameFlag, filters)
}

// matchesName is the internal implementation.
func matchesName(p formats.Package, nameFlag string, filters []string) bool {
	if nameFlag == "" || len(filters) == 0 {
		return true
	}
	return utils.ContainsIgnoreCase(filters, p.Name)
}

// MatchesGroup checks if a package matches the group filter.
//
// Group matching is case-insensitive.
//
// Parameters:
//   - p: Package to check
//   - groupFlag: Original group flag value
//   - filters: Parsed group filters
//
// Returns:
//   - bool: true if package matches
func MatchesGroup(p formats.Package, groupFlag string, filters []string) bool {
	return matchesGroup(p, groupFlag, filters)
}

// matchesGroup is the internal implementation.
func matchesGroup(p formats.Package, groupFlag string, filters []string) bool {
	if groupFlag == "" || len(filters) == 0 {
		return true
	}
	return utils.ContainsIgnoreCase(filters, p.Group)
}

// FilterByGroup filters packages to only include those matching the group filter.
//
// This is a simplified filter that only checks group membership.
//
// Parameters:
//   - pkgs: Slice of packages to filter
//   - groupFlag: Comma-separated group names
//
// Returns:
//   - []formats.Package: Packages matching any of the specified groups
func FilterByGroup(pkgs []formats.Package, groupFlag string) []formats.Package {
	if groupFlag == "" {
		return pkgs
	}
	groupFilters := utils.TrimAndSplit(groupFlag, ",")
	if len(groupFilters) == 0 {
		return pkgs
	}
	var filtered []formats.Package
	for _, p := range pkgs {
		if utils.ContainsIgnoreCase(groupFilters, p.Group) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
