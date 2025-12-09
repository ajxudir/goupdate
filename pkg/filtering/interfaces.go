package filtering

import (
	"github.com/ajxudir/goupdate/pkg/formats"
)

// PackageFilter defines the interface for filtering packages.
//
// This interface enables testing code that depends on package filtering
// by allowing mock implementations to be substituted.
//
// Example:
//
//	type mockFilter struct {
//	    result []formats.Package
//	}
//	func (m *mockFilter) Filter(pkgs []formats.Package) []formats.Package {
//	    return m.result
//	}
type PackageFilter interface {
	// Filter applies filtering logic to a slice of packages.
	//
	// Parameters:
	//   - pkgs: Packages to filter
	//
	// Returns:
	//   - []formats.Package: Filtered packages
	Filter(pkgs []formats.Package) []formats.Package
}

// OptionsFilter is an adapter that implements PackageFilter using FilterOptions.
//
// Example:
//
//	opts := filtering.FromFlags("prod", "npm", "", "", "")
//	filter := &filtering.OptionsFilter{Options: opts}
//	result := filter.Filter(packages)
type OptionsFilter struct {
	Options FilterOptions
}

// Filter applies the options-based filtering to packages.
//
// Parameters:
//   - pkgs: Packages to filter
//
// Returns:
//   - []formats.Package: Filtered packages
func (f *OptionsFilter) Filter(pkgs []formats.Package) []formats.Package {
	return FilterPackages(pkgs, f.Options)
}

// Verify that OptionsFilter implements the PackageFilter interface.
var _ PackageFilter = (*OptionsFilter)(nil)
