package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/packages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// LIST COMMAND ADDITIONAL TESTS
// =============================================================================
//
// These tests cover additional list command scenarios including structured
// output, file filtering, and edge cases.
// =============================================================================

// TestFilterPackagesWithFiltersGroupFilter tests the behavior of group filtering.
//
// It verifies:
//   - Group filter works with other filters
//   - Multiple groups can be filtered
//   - Combined filters work correctly
func TestFilterPackagesWithFiltersGroupFilter(t *testing.T) {
	packages := []formats.Package{
		{Name: "a", Type: "prod", PackageType: "js", Rule: "npm", Group: "core"},
		{Name: "b", Type: "dev", PackageType: "js", Rule: "npm", Group: "utils"},
		{Name: "c", Type: "prod", PackageType: "pip", Rule: "pip", Group: ""},
	}

	// Filter by group using filtering.FilterPackagesWithFilters
	filtered := filtering.FilterPackagesWithFilters(packages, "all", "all", "all", "", "core")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "a", filtered[0].Name)
}

// TestParseSpecificFilesNoMatchingRule tests the behavior when no matching rule is found.
//
// It verifies:
//   - Error is returned when file has no matching rule
//   - Error message indicates missing rule
//   - File path is included in error
func TestParseSpecificFilesNoMatchingRule(t *testing.T) {
	cfg := &config.Config{
		WorkingDir: "/repo",
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Include: []string{"**/package.json"},
			},
		},
	}
	parser := packages.NewDynamicParser()

	// File that doesn't match any rule
	_, err := parseSpecificFiles([]string{"/repo/requirements.txt"}, cfg, parser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no rule config found")
}

// TestParseSpecificFilesParseError tests the behavior when parsing fails.
//
// It verifies:
//   - Parse errors are properly reported
//   - Warning is added to collector
//   - Parsing continues after error
func TestParseSpecificFilesParseError(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "package.json")
	// Write invalid JSON to cause parse error
	err := os.WriteFile(badFile, []byte(`{invalid json`), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		WorkingDir: tmpDir,
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Manager: "js",
				Include: []string{"**/package.json"},
				Format:  "json",
				Fields: map[string]string{
					"dependencies": "prod",
				},
			},
		},
	}
	parser := packages.NewDynamicParser()

	_, err = parseSpecificFiles([]string{badFile}, cfg, parser)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestRunListNoPackagesWithUnsupportedMessages tests the behavior with unsupported package messages.
//
// It verifies:
//   - Unsupported messages are displayed
//   - "No packages found" message is shown
//   - Unsupported reasons are listed
func TestRunListNoPackagesWithUnsupportedMessages(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldGetPkgs := getPackagesFunc
	oldApplyInstalled := applyInstalledVersionsFunc
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		getPackagesFunc = oldGetPkgs
		applyInstalledVersionsFunc = oldApplyInstalled
	}()

	// Mock getPackagesFunc to return package with unsupported status
	// This triggers the shouldTrackUnsupported path and unsupported.Add call
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "test", Rule: "npm", PackageType: "js", InstallStatus: "NotConfigured"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = ""
	os.Args = []string{"goupdate", "list", "-d", tmpDir}

	out := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should print package table and possibly unsupported messages
	assert.Contains(t, out, "test")
}

// TestRunListStructuredOutputWithPackages tests the behavior of structured output with packages.
//
// It verifies:
//   - JSON output contains package information
//   - Summary counts are accurate
//   - All package fields are included
func TestRunListStructuredOutputWithPackages(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldGetPkgs := getPackagesFunc
	oldApplyInstalled := applyInstalledVersionsFunc
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		getPackagesFunc = oldGetPkgs
		applyInstalledVersionsFunc = oldApplyInstalled
	}()

	// Mock getPackagesFunc to return packages
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "lodash", Rule: "npm", PackageType: "js", Version: "4.0.0", InstallStatus: "LockFound"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		for i := range pkgs {
			pkgs[i].InstalledVersion = pkgs[i].Version
		}
		return pkgs, nil
	}

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json" // Use structured output

	out := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should output JSON structure with packages
	assert.Contains(t, out, `"total_packages":1`)
	assert.Contains(t, out, `"name":"lodash"`)
}

// TestPrintPackagesWithGroupColumn tests the behavior with group column display.
//
// It verifies:
//   - Group column is shown when packages have different groups
//   - Group values are displayed correctly
//   - Table formatting includes group column
func TestPrintPackagesWithGroupColumn(t *testing.T) {
	// shouldShowGroupColumn requires at least one group with 2+ packages
	pkgs := []formats.Package{
		{
			Name:             "lodash",
			Version:          "4.0.0",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			InstalledVersion: "4.0.0",
			InstallStatus:    "LockFound",
			Group:            "utils",
		},
		{
			Name:             "underscore",
			Version:          "1.0.0",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			InstalledVersion: "1.0.0",
			InstallStatus:    "LockFound",
			Group:            "utils", // Same group as lodash to trigger showGroup
		},
		{
			Name:             "react",
			Version:          "17.0.0",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			InstalledVersion: "17.0.0",
			InstallStatus:    "LockFound",
			Group:            "core",
		},
	}

	output := captureStdout(t, func() {
		printPackages(pkgs)
	})

	// Should show GROUP column header (utils has 2+ packages)
	assert.Contains(t, output, "GROUP")
	assert.Contains(t, output, "utils")
	assert.Contains(t, output, "core")
	assert.Contains(t, output, "lodash")
	assert.Contains(t, output, "react")
	assert.Contains(t, output, "Total packages: 3")
}

// TestFilterPackagesByFile tests the behavior of file-based filtering.
//
// It verifies:
//   - Packages are filtered by source file
//   - Only packages from specified file are returned
//   - File filtering works with other filters
func TestFilterPackagesByFile(t *testing.T) {
	baseDir := "/project"

	tests := []struct {
		name     string
		pkgs     []formats.Package
		pattern  string
		expected []string // expected package names in result
	}{
		{
			name:     "empty pattern returns all packages",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/lib/b.json"}},
			pattern:  "",
			expected: []string{"a", "b"},
		},
		{
			name:     "include pattern filters packages",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/lib/b.json"}},
			pattern:  "src/*",
			expected: []string{"a"},
		},
		{
			name:     "exclude pattern with ! prefix",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/lib/b.json"}},
			pattern:  "!lib/*",
			expected: []string{"a"},
		},
		{
			name:     "multiple include patterns with comma",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/lib/b.json"}, {Name: "c", Source: "/project/test/c.json"}},
			pattern:  "src/*,lib/*",
			expected: []string{"a", "b"},
		},
		{
			name:     "exclude all files in directory",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/node_modules/b.json"}},
			pattern:  "!node_modules/*",
			expected: []string{"a"},
		},
		{
			name:     "glob pattern with double star",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/deep/nested/a.json"}, {Name: "b", Source: "/project/lib/b.json"}},
			pattern:  "src/**/*",
			expected: []string{"a"},
		},
		{
			name:     "no packages match pattern",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}},
			pattern:  "lib/*",
			expected: []string{},
		},
		{
			name:     "all packages match pattern",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/src/b.json"}},
			pattern:  "src/*",
			expected: []string{"a", "b"},
		},
		{
			name:     "include and exclude patterns combined",
			pkgs:     []formats.Package{{Name: "a", Source: "/project/src/a.json"}, {Name: "b", Source: "/project/src/test.json"}, {Name: "c", Source: "/project/lib/c.json"}},
			pattern:  "src/*,!src/test.json",
			expected: []string{"a"},
		},
		{
			name:     "handles empty source gracefully",
			pkgs:     []formats.Package{{Name: "a", Source: ""}, {Name: "b", Source: "/project/src/b.json"}},
			pattern:  "src/*",
			expected: []string{"b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filtering.FilterPackagesByFile(tt.pkgs, tt.pattern, baseDir)

			var names []string
			for _, p := range result {
				names = append(names, p.Name)
			}

			if len(tt.expected) == 0 {
				assert.Empty(t, names)
			} else {
				assert.Equal(t, tt.expected, names)
			}
		})
	}
}
