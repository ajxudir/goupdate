package filtering

import (
	"path/filepath"
	"strings"

	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/utils"
)

// FileFilterPatterns holds include and exclude glob patterns for file filtering.
type FileFilterPatterns struct {
	Include []string
	Exclude []string
}

// ParseFileFilterPatterns parses a comma-separated filter string into include/exclude patterns.
// Patterns starting with ! are treated as exclusion patterns.
//
// Parameters:
//   - filter: Comma-separated patterns (e.g., "*.json,!node_modules/*")
//
// Returns:
//   - FileFilterPatterns: Parsed include and exclude patterns
//
// Example:
//
//	patterns := filtering.ParseFileFilterPatterns("package.json,go.mod,!vendor/*")
//	// patterns.Include = ["package.json", "go.mod"]
//	// patterns.Exclude = ["vendor/*"]
func ParseFileFilterPatterns(filter string) FileFilterPatterns {
	var patterns FileFilterPatterns
	for _, p := range strings.Split(filter, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "!") {
			patterns.Exclude = append(patterns.Exclude, strings.TrimPrefix(p, "!"))
		} else {
			patterns.Include = append(patterns.Include, p)
		}
	}
	return patterns
}

// MatchesFileFilter checks if a file path matches the filter patterns.
// If include patterns exist, the file must match at least one.
// If the file matches any exclude pattern, it is rejected.
//
// Parameters:
//   - path: The file path to check
//   - patterns: The filter patterns to match against
//
// Returns:
//   - bool: true if the path matches the filter criteria
//
// Example:
//
//	patterns := ParseFileFilterPatterns("*.json,!test/*")
//	filtering.MatchesFileFilter("package.json", patterns)  // true
//	filtering.MatchesFileFilter("test/data.json", patterns) // false
func MatchesFileFilter(path string, patterns FileFilterPatterns) bool {
	// Check excludes first - if any exclude pattern matches, reject
	for _, pattern := range patterns.Exclude {
		if utils.MatchGlob(path, pattern) {
			return false
		}
	}

	// If no include patterns, accept all (that weren't excluded)
	if len(patterns.Include) == 0 {
		return true
	}

	// Check includes - accept if any pattern matches
	for _, pattern := range patterns.Include {
		if utils.MatchGlob(path, pattern) {
			return true
		}
	}

	return false
}

// FilterPackagesByFile filters packages by their source file path patterns.
// Patterns support glob syntax and comma-separated values.
// Patterns starting with ! are exclusion patterns.
//
// Parameters:
//   - pkgs: The packages to filter
//   - filterPattern: Comma-separated patterns (e.g., "*.json,!vendor/*")
//   - baseDir: Base directory for relative path calculation
//
// Returns:
//   - []formats.Package: Packages whose source files match the filter
//
// Example:
//
//	filtered := filtering.FilterPackagesByFile(pkgs, "package.json,go.mod", "/project")
func FilterPackagesByFile(pkgs []formats.Package, filterPattern, baseDir string) []formats.Package {
	patterns := ParseFileFilterPatterns(filterPattern)
	if len(patterns.Include) == 0 && len(patterns.Exclude) == 0 {
		return pkgs
	}

	var result []formats.Package
	for _, p := range pkgs {
		relPath, _ := filepath.Rel(baseDir, p.Source)
		if relPath == "" {
			relPath = filepath.Base(p.Source)
		}

		if MatchesFileFilter(relPath, patterns) {
			result = append(result, p)
		}
	}
	return result
}

// FilterDetectedFiles filters a map of detected files by source file path patterns.
// This is similar to FilterPackagesByFile but works with the scan command's output.
//
// Parameters:
//   - detected: Map of rule -> file paths
//   - filterPattern: Comma-separated patterns (e.g., "*.json,!vendor/*")
//   - baseDir: Base directory for relative path calculation
//
// Returns:
//   - map[string][]string: Filtered map with only matching files
//
// Example:
//
//	detected := map[string][]string{
//	    "npm": {"/project/package.json", "/project/vendor/package.json"},
//	}
//	filtered := filtering.FilterDetectedFiles(detected, "*.json,!vendor/*", "/project")
func FilterDetectedFiles(detected map[string][]string, filterPattern, baseDir string) map[string][]string {
	patterns := ParseFileFilterPatterns(filterPattern)
	if len(patterns.Include) == 0 && len(patterns.Exclude) == 0 {
		return detected
	}

	result := make(map[string][]string)
	for rule, files := range detected {
		var filteredFiles []string
		for _, file := range files {
			relPath, _ := filepath.Rel(baseDir, file)
			if relPath == "" {
				relPath = filepath.Base(file)
			}

			if MatchesFileFilter(relPath, patterns) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		if len(filteredFiles) > 0 {
			result[rule] = filteredFiles
		}
	}
	return result
}
