// Package filtering provides unified package filtering for goupdate.
//
// This package consolidates all filtering logic that was previously scattered
// across cmd/shared/filter.go and cmd/shared/groups.go.
//
// Basic Filtering:
//
// Use FilterOptions to specify filter criteria:
//
//	opts := filtering.FilterOptions{
//	    Type:   "prod",
//	    PM:     "npm",
//	    Rule:   "frontend",
//	    Name:   "react,lodash",
//	    Group:  "core",
//	}
//	filtered := filtering.FilterPackages(packages, opts)
//
// Or use FromFlags for CLI integration:
//
//	opts := filtering.FromFlags(typeFlag, pmFlag, ruleFlag, nameFlag, groupFlag)
//	filtered := filtering.FilterPackages(packages, opts)
//
// Group Assignment:
//
// Assign groups to packages based on configuration:
//
//	packages = filtering.ApplyPackageGroups(packages, cfg)
//
// Sorting:
//
// Sort packages for consistent display:
//
//	sorted := filtering.SortPackagesForDisplay(packages)
package filtering
