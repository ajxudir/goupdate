package filtering

import (
	"github.com/user/goupdate/pkg/utils"
)

// FilterAll is the default filter value that matches all items.
const FilterAll = "all"

// FilterOptions contains all filter criteria for package filtering.
//
// Each field can contain comma-separated values. Empty strings or "all"
// match everything for that criteria.
//
// Fields:
//   - Type: Package dependency type (prod, dev, all)
//   - PM: Package manager (npm, go, composer, all)
//   - Rule: Configuration rule name
//   - Name: Package name (case-insensitive)
//   - Group: Package group (case-insensitive)
//   - File: File path patterns (supports globs)
type FilterOptions struct {
	// Type filters by dependency type (prod, dev, all).
	Type string

	// PM filters by package manager (npm, go, composer, etc.).
	PM string

	// Rule filters by configuration rule name.
	Rule string

	// Name filters by package name (case-insensitive, comma-separated).
	Name string

	// Group filters by package group (case-insensitive, comma-separated).
	Group string

	// File filters by file path patterns (comma-separated, supports globs).
	File string
}

// parsedFilters holds pre-parsed filter slices for efficient matching.
//
// This internal type stores comma-separated filter strings split into slices,
// avoiding repeated parsing during package filtering operations.
//
// Fields:
//   - types: Parsed dependency type filters
//   - pms: Parsed package manager filters
//   - rules: Parsed rule name filters
//   - names: Parsed package name filters
//   - groups: Parsed group name filters
//   - files: Parsed file path filters
type parsedFilters struct {
	types  []string
	pms    []string
	rules  []string
	names  []string
	groups []string
	files  []string
}

// Parse parses the filter options into string slices for matching.
//
// Splits all comma-separated filter values into slices and trims whitespace.
// This is used internally during filtering to avoid repeated parsing.
//
// Returns:
//   - parsedFilters: Pre-parsed filter slices ready for matching
//
// Example:
//
//	opts := filtering.FilterOptions{Type: "prod,dev", Name: "react, lodash"}
//	parsed := opts.Parse()
//	// parsed.types = ["prod", "dev"]
//	// parsed.names = ["react", "lodash"]
func (o FilterOptions) Parse() parsedFilters {
	return parsedFilters{
		types:  utils.TrimAndSplit(o.Type, ","),
		pms:    utils.TrimAndSplit(o.PM, ","),
		rules:  utils.TrimAndSplit(o.Rule, ","),
		names:  utils.TrimAndSplit(o.Name, ","),
		groups: utils.TrimAndSplit(o.Group, ","),
		files:  utils.TrimAndSplit(o.File, ","),
	}
}

// IsEmpty returns true if all filter options are unset or "all".
//
// Returns:
//   - bool: true if no filters are set (all packages would match)
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts.IsEmpty()  // true
//
//	opts = filtering.FilterOptions{Type: "prod"}
//	opts.IsEmpty()  // false
func (o FilterOptions) IsEmpty() bool {
	return (o.Type == "" || o.Type == FilterAll) &&
		(o.PM == "" || o.PM == FilterAll) &&
		(o.Rule == "" || o.Rule == FilterAll) &&
		o.Name == "" &&
		o.Group == "" &&
		o.File == ""
}

// HasTypeFilter returns true if a type filter is set and not "all".
//
// Returns:
//   - bool: true if Type is set to a value other than empty or "all"
func (o FilterOptions) HasTypeFilter() bool {
	return o.Type != "" && o.Type != FilterAll
}

// HasPMFilter returns true if a package manager filter is set and not "all".
//
// Returns:
//   - bool: true if PM is set to a value other than empty or "all"
func (o FilterOptions) HasPMFilter() bool {
	return o.PM != "" && o.PM != FilterAll
}

// HasRuleFilter returns true if a rule filter is set and not "all".
//
// Returns:
//   - bool: true if Rule is set to a value other than empty or "all"
func (o FilterOptions) HasRuleFilter() bool {
	return o.Rule != "" && o.Rule != FilterAll
}

// HasNameFilter returns true if a name filter is set.
//
// Returns:
//   - bool: true if Name is set to a non-empty value
func (o FilterOptions) HasNameFilter() bool {
	return o.Name != ""
}

// HasGroupFilter returns true if a group filter is set.
//
// Returns:
//   - bool: true if Group is set to a non-empty value
func (o FilterOptions) HasGroupFilter() bool {
	return o.Group != ""
}

// HasFileFilter returns true if a file filter is set.
//
// Returns:
//   - bool: true if File is set to a non-empty value
func (o FilterOptions) HasFileFilter() bool {
	return o.File != ""
}

// FromFlags creates FilterOptions from CLI flag values.
//
// Parameters:
//   - typeFlag: Type filter flag value
//   - pmFlag: Package manager filter flag value
//   - ruleFlag: Rule filter flag value
//   - nameFlag: Name filter flag value
//   - groupFlag: Group filter flag value
//
// Returns:
//   - FilterOptions: Populated filter options
//
// Example:
//
//	opts := filtering.FromFlags("prod", "npm", "all", "", "core")
func FromFlags(typeFlag, pmFlag, ruleFlag, nameFlag, groupFlag string) FilterOptions {
	return FilterOptions{
		Type:  typeFlag,
		PM:    pmFlag,
		Rule:  ruleFlag,
		Name:  nameFlag,
		Group: groupFlag,
	}
}

// FromFlagsWithFile creates FilterOptions from CLI flags including file pattern.
//
// Parameters:
//   - typeFlag: Type filter flag value
//   - pmFlag: Package manager filter flag value
//   - ruleFlag: Rule filter flag value
//   - nameFlag: Name filter flag value
//   - groupFlag: Group filter flag value
//   - fileFlag: File path filter patterns
//
// Returns:
//   - FilterOptions: Populated filter options
func FromFlagsWithFile(typeFlag, pmFlag, ruleFlag, nameFlag, groupFlag, fileFlag string) FilterOptions {
	return FilterOptions{
		Type:  typeFlag,
		PM:    pmFlag,
		Rule:  ruleFlag,
		Name:  nameFlag,
		Group: groupFlag,
		File:  fileFlag,
	}
}

// WithType returns a copy with the type filter set.
//
// Parameters:
//   - t: Type filter value (prod, dev, all)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated Type field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithType("prod")
func (o FilterOptions) WithType(t string) FilterOptions {
	o.Type = t
	return o
}

// WithPM returns a copy with the package manager filter set.
//
// Parameters:
//   - pm: Package manager filter value (npm, go, composer, etc.)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated PM field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithPM("npm")
func (o FilterOptions) WithPM(pm string) FilterOptions {
	o.PM = pm
	return o
}

// WithRule returns a copy with the rule filter set.
//
// Parameters:
//   - rule: Rule filter value (configuration rule name)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated Rule field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithRule("frontend")
func (o FilterOptions) WithRule(rule string) FilterOptions {
	o.Rule = rule
	return o
}

// WithName returns a copy with the name filter set.
//
// Parameters:
//   - name: Package name filter (case-insensitive, comma-separated)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated Name field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithName("react,lodash")
func (o FilterOptions) WithName(name string) FilterOptions {
	o.Name = name
	return o
}

// WithGroup returns a copy with the group filter set.
//
// Parameters:
//   - group: Group filter value (case-insensitive, comma-separated)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated Group field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithGroup("core")
func (o FilterOptions) WithGroup(group string) FilterOptions {
	o.Group = group
	return o
}

// WithFile returns a copy with the file filter set.
//
// Parameters:
//   - file: File path filter patterns (comma-separated, supports globs)
//
// Returns:
//   - FilterOptions: New FilterOptions with updated File field
//
// Example:
//
//	opts := filtering.FilterOptions{}
//	opts = opts.WithFile("*.json,!vendor/*")
func (o FilterOptions) WithFile(file string) FilterOptions {
	o.File = file
	return o
}
