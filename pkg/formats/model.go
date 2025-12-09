// Package formats defines the data structures and interfaces for parsing package
// manifest files. It provides parsers for JSON, YAML, XML, and custom formats,
// with support for XPath/JSONPath-like field extraction.
package formats

import "github.com/user/goupdate/pkg/config"

// Package represents a declared dependency captured by a parser.
//
// Fields:
//   - Name: The package name as declared in the manifest file
//   - Version: The parsed version number (e.g., "1.2.3")
//   - Constraint: The version constraint operator (e.g., "==", ">=", "~>")
//   - Type: The dependency type ("prod" for production, "dev" for development)
//   - PackageType: The package manager name (e.g., "npm", "pip", "nuget")
//   - Rule: The update rule name from configuration
//   - Source: The source file path where this package was declared
//   - InstalledVersion: The currently installed version (if known)
//   - InstallStatus: The installation status (e.g., "installed", "missing")
//   - Group: Optional dependency group or category
type Package struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Constraint       string `json:"constraint"`
	Type             string `json:"type"`
	PackageType      string `json:"package_type"`
	Rule             string `json:"rule"`
	Source           string `json:"source"`
	InstalledVersion string `json:"installed_version"`
	InstallStatus    string `json:"install_status"`
	Group            string `json:"group,omitempty"`
}

// GetName returns the package name and implements the config.PackageRef interface.
//
// Returns:
//   - string: The package name from the Name field
func (p Package) GetName() string {
	return p.Name
}

// GetRule returns the update rule name and implements the config.PackageRef interface.
//
// Returns:
//   - string: The update rule name from the Rule field
func (p Package) GetRule() string {
	return p.Rule
}

// PackageList is a collection of packages found in a single source file.
//
// Fields:
//   - Packages: The list of Package objects parsed from the source file
//   - Source: The path to the source file containing these packages
type PackageList struct {
	Packages []Package `json:"packages"`
	Source   string    `json:"source"`
}

// FormatParser defines the interface for parsing package file contents.
//
// Implementations should parse the raw file content according to their format
// (JSON, YAML, XML, or raw text) and extract package dependencies based on
// the provided configuration.
type FormatParser interface {
	// Parse parses raw file content and extracts package dependencies.
	//
	// Parameters:
	//   - content: The raw bytes of the package manifest file
	//   - cfg: The package manager configuration containing fields, patterns, and rules
	//
	// Returns:
	//   - []Package: A list of parsed packages with their versions and metadata
	//   - error: Returns an error if the content is invalid or cannot be parsed; returns nil on success
	Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error)
}
