package output

import "encoding/xml"

// ScanResult represents the output data for the scan command.
//
// Fields:
//   - XMLName: XML root element name (used only for XML marshaling)
//   - Summary: Aggregate statistics about the scan operation
//   - Files: List of individual file entries discovered during scanning
type ScanResult struct {
	XMLName xml.Name    `json:"-" xml:"scanResult"`
	Summary ScanSummary `json:"summary" xml:"summary"`
	Files   []ScanEntry `json:"files" xml:"files>file"`
}

// ScanSummary holds summary statistics for scan results.
//
// Fields:
//   - Directory: The directory path that was scanned
//   - TotalEntries: Total number of entries found during the scan
//   - UniqueFiles: Number of unique files discovered
//   - RulesMatched: Number of pattern matching rules that found matches
//   - ValidFiles: Count of files that passed validation
//   - InvalidFiles: Count of files that failed validation
type ScanSummary struct {
	Directory    string `json:"directory" xml:"directory"`
	TotalEntries int    `json:"total_entries" xml:"totalEntries"`
	UniqueFiles  int    `json:"unique_files" xml:"uniqueFiles"`
	RulesMatched int    `json:"rules_matched" xml:"rulesMatched"`
	ValidFiles   int    `json:"valid_files" xml:"validFiles"`
	InvalidFiles int    `json:"invalid_files" xml:"invalidFiles"`
}

// ScanEntry represents a single scanned file entry.
//
// Fields:
//   - Rule: The pattern matching rule that identified this file
//   - PM: Package manager identifier (e.g., "npm", "pip", "go")
//   - Format: File format or type (e.g., "package.json", "requirements.txt")
//   - File: Absolute or relative path to the file
//   - Status: Current status of the entry (e.g., "valid", "invalid")
//   - Error: Error message if the entry failed validation (omitted if empty)
type ScanEntry struct {
	Rule   string `json:"rule" xml:"rule"`
	PM     string `json:"pm" xml:"pm"`
	Format string `json:"format" xml:"format"`
	File   string `json:"file" xml:"file"`
	Status string `json:"status" xml:"status"`
	Error  string `json:"error,omitempty" xml:"error,omitempty"`
}

// ListResult represents the output data for the list command.
//
// Fields:
//   - XMLName: XML root element name (used only for XML marshaling)
//   - Summary: Aggregate statistics about the list operation
//   - Packages: List of package entries
//   - Warnings: Warning messages generated during the list operation (omitted if empty)
type ListResult struct {
	XMLName  xml.Name      `json:"-" xml:"listResult"`
	Summary  ListSummary   `json:"summary" xml:"summary"`
	Packages []ListPackage `json:"packages" xml:"packages>package"`
	Warnings []string      `json:"warnings,omitempty" xml:"warnings>warning,omitempty"`
}

// ListSummary holds summary statistics for list results.
//
// Fields:
//   - TotalPackages: Total number of packages in the list
type ListSummary struct {
	TotalPackages int `json:"total_packages" xml:"totalPackages"`
}

// ListPackage represents a package entry in the list output.
//
// Fields:
//   - Rule: The pattern matching rule that identified this package
//   - PM: Package manager identifier (e.g., "npm", "pip", "go")
//   - Type: Package type (e.g., "direct", "dev", "peer")
//   - Constraint: Version constraint specified in the dependency file
//   - Version: Latest available version
//   - InstalledVersion: Currently installed version
//   - Status: Current status of the package (e.g., "ok", "missing")
//   - Group: Optional grouping identifier (omitted if empty)
//   - Name: Package name
type ListPackage struct {
	Rule             string `json:"rule" xml:"rule"`
	PM               string `json:"pm" xml:"pm"`
	Type             string `json:"type" xml:"type"`
	Constraint       string `json:"constraint" xml:"constraint"`
	Version          string `json:"version" xml:"version"`
	InstalledVersion string `json:"installed_version" xml:"installedVersion"`
	Status           string `json:"status" xml:"status"`
	Group            string `json:"group,omitempty" xml:"group,omitempty"`
	Name             string `json:"name" xml:"name"`
}

// OutdatedResult represents the output data for the outdated command.
//
// Fields:
//   - XMLName: XML root element name (used only for XML marshaling)
//   - Summary: Aggregate statistics about the outdated operation
//   - Packages: List of package entries with version information
//   - Warnings: Warning messages generated during the outdated check (omitted if empty)
//   - Errors: Error messages generated during the outdated check (omitted if empty)
type OutdatedResult struct {
	XMLName  xml.Name          `json:"-" xml:"outdatedResult"`
	Summary  OutdatedSummary   `json:"summary" xml:"summary"`
	Packages []OutdatedPackage `json:"packages" xml:"packages>package"`
	Warnings []string          `json:"warnings,omitempty" xml:"warnings>warning,omitempty"`
	Errors   []string          `json:"errors,omitempty" xml:"errors>error,omitempty"`
}

// OutdatedSummary holds summary statistics for outdated results.
//
// Fields:
//   - TotalPackages: Total number of packages checked
//   - OutdatedPackages: Number of packages with available updates
//   - UpToDatePackages: Number of packages already at the latest version
//   - FailedPackages: Number of packages that failed the version check
//   - HasMajor: Number of packages with major updates available
//   - HasMinor: Number of packages with minor updates available
//   - HasPatch: Number of packages with patch updates available
type OutdatedSummary struct {
	TotalPackages    int `json:"total_packages" xml:"totalPackages"`
	OutdatedPackages int `json:"outdated_packages" xml:"outdatedPackages"`
	UpToDatePackages int `json:"uptodate_packages" xml:"uptodatePackages"`
	FailedPackages   int `json:"failed_packages" xml:"failedPackages"`
	HasMajor         int `json:"has_major" xml:"hasMajor"`
	HasMinor         int `json:"has_minor" xml:"hasMinor"`
	HasPatch         int `json:"has_patch" xml:"hasPatch"`
}

// OutdatedPackage represents a package entry in the outdated output.
//
// Fields:
//   - Rule: The pattern matching rule that identified this package
//   - PM: Package manager identifier (e.g., "npm", "pip", "go")
//   - Type: Package type (e.g., "direct", "dev", "peer")
//   - Constraint: Version constraint specified in the dependency file
//   - Version: Latest available version
//   - InstalledVersion: Currently installed version
//   - Major: Latest available major version
//   - Minor: Latest available minor version
//   - Patch: Latest available patch version
//   - Status: Current status (e.g., "outdated", "up-to-date", "failed")
//   - Group: Optional grouping identifier (omitted if empty)
//   - Name: Package name
//   - Error: Error message if the version check failed (omitted if empty)
type OutdatedPackage struct {
	Rule             string `json:"rule" xml:"rule"`
	PM               string `json:"pm" xml:"pm"`
	Type             string `json:"type" xml:"type"`
	Constraint       string `json:"constraint" xml:"constraint"`
	Version          string `json:"version" xml:"version"`
	InstalledVersion string `json:"installed_version" xml:"installedVersion"`
	Major            string `json:"major" xml:"major"`
	Minor            string `json:"minor" xml:"minor"`
	Patch            string `json:"patch" xml:"patch"`
	Status           string `json:"status" xml:"status"`
	Group            string `json:"group,omitempty" xml:"group,omitempty"`
	Name             string `json:"name" xml:"name"`
	Error            string `json:"error,omitempty" xml:"error,omitempty"`
}

// UpdateResult represents the output data for the update command.
//
// Fields:
//   - XMLName: XML root element name (used only for XML marshaling)
//   - Summary: Aggregate statistics about the update operation
//   - Packages: List of package entries with update information
//   - Warnings: Warning messages generated during the update operation (omitted if empty)
//   - Errors: Error messages generated during the update operation (omitted if empty)
type UpdateResult struct {
	XMLName  xml.Name        `json:"-" xml:"updateResult"`
	Summary  UpdateSummary   `json:"summary" xml:"summary"`
	Packages []UpdatePackage `json:"packages" xml:"packages>package"`
	Warnings []string        `json:"warnings,omitempty" xml:"warnings>warning,omitempty"`
	Errors   []string        `json:"errors,omitempty" xml:"errors>error,omitempty"`
}

// UpdateSummary holds summary statistics for update results.
//
// Fields:
//   - TotalPackages: Total number of packages processed
//   - UpdatedPackages: Number of packages successfully updated
//   - FailedPackages: Number of packages that failed to update
//   - DryRun: Whether this was a dry-run (no actual updates performed)
type UpdateSummary struct {
	TotalPackages   int  `json:"total_packages" xml:"totalPackages"`
	UpdatedPackages int  `json:"updated_packages" xml:"updatedPackages"`
	FailedPackages  int  `json:"failed_packages" xml:"failedPackages"`
	DryRun          bool `json:"dry_run" xml:"dryRun"`
}

// UpdatePackage represents a package entry in the update output.
//
// Fields:
//   - Rule: The pattern matching rule that identified this package
//   - PM: Package manager identifier (e.g., "npm", "pip", "go")
//   - Type: Package type (e.g., "direct", "dev", "peer")
//   - Constraint: Version constraint specified in the dependency file
//   - Version: Latest available version
//   - InstalledVersion: Currently installed version before update
//   - Target: Target version for the update
//   - Status: Current status (e.g., "updated", "failed", "skipped")
//   - Group: Optional grouping identifier (omitted if empty)
//   - Name: Package name
//   - Error: Error message if the update failed (omitted if empty)
type UpdatePackage struct {
	Rule             string `json:"rule" xml:"rule"`
	PM               string `json:"pm" xml:"pm"`
	Type             string `json:"type" xml:"type"`
	Constraint       string `json:"constraint" xml:"constraint"`
	Version          string `json:"version" xml:"version"`
	InstalledVersion string `json:"installed_version" xml:"installedVersion"`
	Target           string `json:"target" xml:"target"`
	Status           string `json:"status" xml:"status"`
	Group            string `json:"group,omitempty" xml:"group,omitempty"`
	Name             string `json:"name" xml:"name"`
	Error            string `json:"error,omitempty" xml:"error,omitempty"`
}
