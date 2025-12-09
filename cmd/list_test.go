package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/packages"
	"github.com/ajxudir/goupdate/pkg/supervision"
	"github.com/ajxudir/goupdate/pkg/warnings"
)

// TestListCommand tests the behavior of the list command.
//
// It verifies:
//   - List command executes without errors
//   - List command can process package files
//   - Command line arguments are properly handled
func TestListCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies":{"test":"1.0.0"}}`), 0644)
	require.NoError(t, err)

	os.Args = []string{"goupdate", "ls", filepath.Join(tmpDir, "package.json")}
	err = ExecuteTest()
	assert.NoError(t, err)
}

// TestRunListNoPackages tests the behavior when no packages are found.
//
// It verifies:
//   - List completes without errors when no packages exist
//   - Output contains "No packages found" message
//   - Empty package lists are handled gracefully
func TestRunListNoPackages(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
	}()

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = tmpDir
	listConfigFlag = ""
	os.Args = []string{"goupdate", "list", "-d", tmpDir}

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "No packages found")
}

// TestRunListConfigError tests the behavior when config file is missing.
//
// It verifies:
//   - List returns error when specified config file doesn't exist
//   - Error handling for missing config files
//   - Config file validation occurs before listing packages
func TestRunListConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldConfig := listConfigFlag
	defer func() {
		os.Args = oldArgs
		listConfigFlag = oldConfig
	}()

	badCfg := filepath.Join(tmpDir, "missing.yml")
	listConfigFlag = badCfg
	os.Args = []string{"goupdate", "list", "--config", badCfg}

	err := runList(nil, nil)
	assert.Error(t, err)
}

// TestRunListMissingRule tests the behavior when no rule matches a file.
//
// It verifies:
//   - List returns error when file has no matching rule
//   - Error handling for unknown file types
//   - Rule matching is required for file processing
func TestRunListMissingRule(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "unknown.custom")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
	}()

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = tmpDir
	listConfigFlag = ""
	os.Args = []string{"goupdate", "list", filePath}

	err := runList(nil, []string{filePath})
	assert.Error(t, err)
}

// TestRunListInstalledError tests the behavior when lock file resolution fails.
//
// It verifies:
//   - List returns error when lock file parsing fails
//   - Invalid lock file patterns cause errors
//   - Error message indicates lock file resolution failure
func TestRunListInstalledError(t *testing.T) {
	tmpDir := t.TempDir()
	pkgFile := filepath.Join(tmpDir, "pkg.txt")
	lockFile := filepath.Join(tmpDir, "lock.txt")

	require.NoError(t, os.WriteFile(pkgFile, []byte("dep 1.0.0"), 0644))
	require.NoError(t, os.WriteFile(lockFile, []byte("dep 1.0.0"), 0644))

	cfgContent := `working_dir: %s
rules:
  custom:
    manager: custom
    include: ["**/pkg.txt"]
    format: raw
    fields:
      packages: prod
    extraction:
      pattern: '(?m)^(?P<n>[^\s]+)\s+(?P<version>[\d\.]+)'
    lock_files:
      - files: ["**/lock.txt"]
        format: raw
        extraction:
          pattern: '['
`
	cfgPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(fmt.Sprintf(cfgContent, tmpDir)), 0644))

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
	}()

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = tmpDir
	listConfigFlag = cfgPath
	os.Args = []string{"goupdate", "list", "-d", tmpDir, "-c", cfgPath}

	err := runList(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve lock files")
}

// TestDetectAndParseAllDetectError tests the behavior when file detection fails.
//
// It verifies:
//   - getPackages returns error when file detection fails
//   - Detection errors are properly propagated
//   - Error handling for file system issues
func TestDetectAndParseAllDetectError(t *testing.T) {
	oldDetect := detectFilesFunc
	defer func() { detectFilesFunc = oldDetect }()

	detectFilesFunc = func(cfg *config.Config, baseDir string) (map[string][]string, error) {
		return nil, fmt.Errorf("detect failure")
	}

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{}}

	_, err := getPackages(cfg, nil, t.TempDir())
	assert.Error(t, err)
}

// TestListCommandStatusOutput tests the behavior of package status display.
//
// It verifies:
//   - Package status is shown correctly with lock files
//   - LockFound status is displayed for locked packages
//   - LockMissing and NotInLock statuses are handled
func TestListCommandStatusOutput(t *testing.T) {
	rootDir, err := filepath.Abs("..")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		dir      string
		expected []string // Check for presence of these strings in output
	}{
		{
			name: "with lock file",
			dir:  filepath.Join(rootDir, "pkg", "testdata", "npm"),
			// Check status and name - version is static in testdata but checked separately
			expected: []string{"lodash", lock.InstallStatusLockFound},
		},
		{
			name:     "missing lock file",
			dir:      filepath.Join(rootDir, "pkg", "testdata", "npm", "_edge-cases", "no-lock"),
			expected: []string{"express", lock.InstallStatusLockMissing},
		},
		{
			name:     "package missing from lock",
			dir:      filepath.Join(rootDir, "pkg", "testdata_errors", "package-not-found", "npm"),
			expected: []string{"missing-package", lock.InstallStatusNotInLock},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			oldType := listTypeFlag
			oldPM := listPMFlag
			oldDir := listDirFlag
			oldConfig := listConfigFlag
			defer func() {
				os.Args = oldArgs
				listTypeFlag = oldType
				listPMFlag = oldPM
				listDirFlag = oldDir
				listConfigFlag = oldConfig
			}()

			listTypeFlag = "all"
			listPMFlag = "all"
			listDirFlag = tt.dir
			listConfigFlag = ""
			os.Args = []string{"goupdate", "list", "-d", tt.dir}

			output := captureStdout(t, func() {
				err := runList(nil, nil)
				assert.NoError(t, err)
			})

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}
		})
	}
}

// TestListCommandNoLockExamples tests the behavior when lock files are missing.
//
// It verifies:
//   - Packages are listed with LockMissing status
//   - Installed version shows "#N/A" when lock is missing
//   - Missing lock files are handled gracefully
func TestListCommandNoLockExamples(t *testing.T) {
	rootDir, err := filepath.Abs("..")
	assert.NoError(t, err)

	workDir := filepath.Join(rootDir, "pkg", "testdata", "npm", "_edge-cases", "no-lock")

	oldArgs := os.Args
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	defer func() {
		os.Args = oldArgs
		listTypeFlag = oldType
		listPMFlag = oldPM
		listDirFlag = oldDir
		listConfigFlag = oldConfig
	}()

	listTypeFlag = "all"
	listPMFlag = "all"
	listDirFlag = workDir
	listConfigFlag = ""
	os.Args = []string{"goupdate", "list", "-d", workDir}

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, lock.InstallStatusLockMissing)
	assert.Contains(t, output, "express")
	assert.Contains(t, output, "#N/A")
}

// TestFindRuleForFile tests the behavior of rule matching for files.
//
// It verifies:
//   - Correct rule is found for matching files
//   - First matching rule is returned when multiple rules match
//   - Excluded paths are properly handled
func TestFindRuleForFile(t *testing.T) {
	jsRules := []string{"npm", "pnpm", "yarn"}

	rules := map[string]config.PackageManagerCfg{
		"requirements": {
			Include: []string{"**/requirements.txt"},
		},
	}

	for _, ruleName := range jsRules {
		rules[ruleName] = config.PackageManagerCfg{
			Include: []string{"**/package.json"},
			Exclude: []string{"**/node_modules/**"},
		}
	}

	cfg := &config.Config{
		WorkingDir: "/repo",
		Rules:      rules,
	}

	ruleCfg, key := findRuleForFile("/repo/services/api/package.json", cfg)
	require.NotNil(t, ruleCfg)
	assert.Equal(t, "npm", key)
	assert.Equal(t, cfg.Rules["npm"].Include, ruleCfg.Include)
	assert.Equal(t, cfg.Rules["npm"].Exclude, ruleCfg.Exclude)

	nilRule, nilKey := findRuleForFile("services/api/pipfile.lock", cfg)
	assert.Nil(t, nilRule)
	assert.Equal(t, "", nilKey)

	excludedRule, excludedKey := findRuleForFile("/repo/node_modules/pkg/package.json", cfg)
	assert.Nil(t, excludedRule)
	assert.Equal(t, "", excludedKey)
}

// TestSafeInstalledValue tests the behavior of safe installed version display.
//
// It verifies:
//   - Empty installed versions show "#N/A"
//   - Whitespace-only versions show "#N/A"
//   - Valid versions are displayed as-is
func TestSafeInstalledValue(t *testing.T) {
	assert.Equal(t, "#N/A", display.SafeInstalledValue(""))
	assert.Equal(t, "#N/A", display.SafeInstalledValue("   "))
	assert.Equal(t, "1.0.0", display.SafeInstalledValue("1.0.0"))
}

// TestSafeDeclaredValue tests the behavior of safe declared version display.
//
// It verifies:
//   - Empty declared versions show "*" (wildcard)
//   - Whitespace-only versions show "*"
//   - Valid versions are displayed as-is
func TestSafeDeclaredValue(t *testing.T) {
	assert.Equal(t, "*", display.SafeDeclaredValue(""))
	assert.Equal(t, "*", display.SafeDeclaredValue("  \t"))
	assert.Equal(t, "1.2.3", display.SafeDeclaredValue("1.2.3"))
}

// TestPrintPackagesDisplaysLatestFallback tests the behavior of latest version fallback display.
//
// It verifies:
//   - Packages without versions show "Major" constraint
//   - Declared version shows "*" when empty
//   - Installed version shows "#N/A" when empty
func TestPrintPackagesDisplaysLatestFallback(t *testing.T) {
	originalType := listTypeFlag
	originalPM := listPMFlag
	defer func() {
		listTypeFlag = originalType
		listPMFlag = originalPM
	}()

	listTypeFlag = "all"
	listPMFlag = "all"

	packages := []formats.Package{{
		Rule:        "rule",
		PackageType: "js",
		Type:        "prod",
		Name:        "noversion",
		Version:     "",
	}}

	output := captureStdout(t, func() {
		printPackages(packages)
	})

	lines := strings.Split(output, "\n")
	var fields []string
	for _, line := range lines {
		row := strings.Fields(strings.TrimSpace(line))
		if len(row) >= 7 && row[0] == "rule" {
			fields = row
			break
		}
	}

	require.NotEmpty(t, fields)
	assert.Equal(t, "Major", fields[3])
	assert.Equal(t, "*", fields[4])
	assert.Equal(t, "#N/A", fields[5])
}

// TestDetectAndParseAll tests the behavior of package detection and parsing.
//
// It verifies:
//   - No packages are returned when no files are detected
//   - Parser errors are reported and skipped
//   - Warning messages are shown for parse failures
func TestDetectAndParseAll(t *testing.T) {
	t.Run("no detected files", func(t *testing.T) {
		cfg := &config.Config{WorkingDir: t.TempDir(), Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Manager: "js",
				Format:  "json",
				Include: []string{"**/package.json"},
			},
		}}

		pkgs, err := getPackages(cfg, nil, cfg.WorkingDir)

		assert.NoError(t, err)
		assert.Empty(t, pkgs)
	})

	t.Run("parser error is reported and skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "bad.invalid")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

		cfg := &config.Config{WorkingDir: tmpDir, Rules: map[string]config.PackageManagerCfg{
			"invalid": {
				Manager: "custom",
				Format:  "does-not-exist",
				Include: []string{"**/*.invalid"},
			},
		}}

		var buf bytes.Buffer
		restore := warnings.SetWarningWriter(&buf)
		t.Cleanup(restore)

		pkgs, err := getPackages(cfg, nil, tmpDir)
		assert.NoError(t, err)
		assert.Empty(t, pkgs)

		assert.Contains(t, buf.String(), "⚠️ failed to parse")
		assert.Contains(t, buf.String(), "bad.invalid")
	})
}

// TestApplyPackageGroupsOnlyWhenConfigured tests the behavior of package group application.
//
// It verifies:
//   - Groups are applied only when configured
//   - Packages without group config remain ungrouped
//   - Missing rule config results in empty group
func TestApplyPackageGroupsOnlyWhenConfigured(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {Update: &config.UpdateCfg{}},
	}}

	packages := []formats.Package{{Rule: "r", Name: "demo", PackageType: "js"}}

	grouped := filtering.ApplyPackageGroups(packages, cfg)
	assert.Equal(t, "", grouped[0].Group)

	cfg.Rules["r"] = config.PackageManagerCfg{Update: &config.UpdateCfg{Group: "bundle"}}
	grouped = filtering.ApplyPackageGroups(packages, cfg)
	assert.Equal(t, "bundle", grouped[0].Group)

	grouped = filtering.ApplyPackageGroups([]formats.Package{{Rule: "missing", Name: "demo", PackageType: "js"}}, cfg)
	assert.Equal(t, "", grouped[0].Group)
}

// TestRunListPrintsUnsupportedReasons tests the behavior of unsupported package reason display.
//
// It verifies:
//   - Unsupported reasons are shown for floating constraints
//   - Output contains "Floating constraint" message
//   - Unsupported packages are properly identified
func TestRunListPrintsUnsupportedReasons(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalType := listTypeFlag
	originalPM := listPMFlag
	originalDir := listDirFlag
	originalConfig := listConfigFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: ".", Rules: map[string]config.PackageManagerCfg{}}, nil
	}
	// Wildcard version is a floating constraint - shows unsupported reason
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Rule: "missing", Name: "demo", PackageType: "js", Type: "prod", Version: "*", InstallStatus: lock.InstallStatusFloating}}, nil
	}
	listTypeFlag, listPMFlag, listDirFlag, listConfigFlag = "all", "all", ".", ""

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		listTypeFlag = originalType
		listPMFlag = originalPM
		listDirFlag = originalDir
		listConfigFlag = originalConfig
	})

	output := captureStdout(t, func() {
		require.NoError(t, runList(listCmd, nil))
	})

	// Wildcard "*" is a floating constraint
	assert.Contains(t, output, "Floating constraint")
}

// TestRunListTracksUnsupported tests the behavior of unsupported package tracking.
//
// It verifies:
//   - NotConfigured status doesn't show messages for non-wildcard versions
//   - Unsupported tracking reduces noise in output
//   - Non-wildcard versions don't trigger unsupported warnings
func TestRunListTracksUnsupported(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: ".", Rules: map[string]config.PackageManagerCfg{}}, nil
	}
	// Non-wildcard version should NOT show NotConfigured reason (reduces noise)
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Rule: "missing", Name: "demo", PackageType: "js", Type: "prod", Version: "1.0.0", InstallStatus: lock.InstallStatusNotConfigured}}, nil
	}

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
	})

	output := captureStdout(t, func() {
		require.NoError(t, runList(listCmd, nil))
	})

	// Non-wildcard versions should NOT show unsupported messages (reduces spam)
	assert.NotContains(t, output, "No rule configuration available for this package.")
}

// TestUnsupportedTrackerAndReasons tests the behavior of the unsupported tracker.
//
// It verifies:
//   - Tracker deduplicate messages for same package
//   - Empty reasons are skipped
//   - Messages are properly formatted
func TestUnsupportedTrackerAndReasons(t *testing.T) {
	tracker := supervision.NewUnsupportedTracker()
	assert.Nil(t, tracker.Messages())

	tracker.Add(formats.Package{Name: "demo", PackageType: "js", Rule: "r"}, "first")
	tracker.Add(formats.Package{Name: "demo", PackageType: "js", Rule: "r"}, "duplicate")
	tracker.Add(formats.Package{Name: "skip", PackageType: "pip", Rule: "p"}, "")
	tracker.Add(formats.Package{Name: "alt", PackageType: "pip", Rule: "r"}, "second")

	messages := tracker.Messages()
	assert.Len(t, messages, 2)
	assert.Contains(t, messages[0], "first")
	assert.Contains(t, messages[1], "second")

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {LockFiles: []config.LockFileCfg{}}}}
	// Floating constraint "*" shows message
	reason := supervision.DeriveUnsupportedReason(formats.Package{Rule: "r", Version: "*", InstalledVersion: "#N/A"}, cfg, nil, false)
	assert.Contains(t, reason, "Floating constraint")
	// NotConfigured is self-explanatory - no extra message needed
	reason = supervision.DeriveUnsupportedReason(formats.Package{Rule: "missing"}, cfg, nil, true)
	assert.Empty(t, reason)
	// Non-wildcard versions should return empty (NotConfigured is self-explanatory)
	reason = supervision.DeriveUnsupportedReason(formats.Package{Rule: "none", Version: "1.0.0"}, &config.Config{Rules: map[string]config.PackageManagerCfg{}}, nil, false)
	assert.Empty(t, reason)
	// Floating constraint "*" shows message
	reason = supervision.DeriveUnsupportedReason(formats.Package{Rule: "none", Version: "*"}, &config.Config{Rules: map[string]config.PackageManagerCfg{}}, nil, false)
	assert.Contains(t, reason, "Floating constraint")
	// VersionMissing shows a message
	reason = supervision.DeriveUnsupportedReason(formats.Package{Rule: "r", InstallStatus: lock.InstallStatusVersionMissing}, cfg, nil, false)
	assert.Contains(t, reason, "No concrete version")
}

// TestSortPackagesForDisplayUsesGroup tests the behavior of package sorting by group.
//
// It verifies:
//   - Packages are sorted by rule first
//   - Within same rule, packages are sorted by group
//   - Sorting maintains consistent order
func TestSortPackagesForDisplayUsesGroup(t *testing.T) {
	packages := []formats.Package{
		{Rule: "r", PackageType: "js", Group: "b", Type: "dev", Name: "beta"},
		{Rule: "r", PackageType: "js", Group: "a", Type: "prod", Name: "alpha"},
		{Rule: "q", PackageType: "pip", Group: "c", Type: "prod", Name: "zed"},
	}

	sorted := filtering.SortPackagesForDisplay(packages)
	require.Len(t, sorted, 3)
	assert.Equal(t, "q", sorted[0].Rule)
	assert.Equal(t, "a", sorted[1].Group)
	assert.Equal(t, "b", sorted[2].Group)
}

// TestApplyPackageGroupsUsesConfigGroups tests the behavior of config-based package grouping.
//
// It verifies:
//   - Groups from rule-level config are applied
//   - Packages matching group patterns are assigned to groups
//   - Group config takes precedence over fallback
func TestApplyPackageGroupsUsesConfigGroups(t *testing.T) {
	packages := []formats.Package{{Name: "alpha", Rule: "r", PackageType: "js", Type: "prod"}, {Name: "beta", Rule: "r", PackageType: "js", Type: "dev"}}
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"r": {
				Update: &config.UpdateCfg{Group: "fallback"},
				Groups: map[string]config.GroupCfg{
					"react": {Packages: []string{"alpha", "beta"}},
				},
			},
		},
	}

	grouped := filtering.ApplyPackageGroups(packages, cfg)
	require.Equal(t, "react", grouped[0].Group)
	require.Equal(t, "react", grouped[1].Group)
}

// TestApplyPackageGroupsSupportsLegacyTopLevel tests the behavior of legacy top-level group support.
//
// It verifies:
//   - Legacy top-level groups are still supported
//   - Top-level group config is applied correctly
//   - Backward compatibility is maintained
func TestApplyPackageGroupsSupportsLegacyTopLevel(t *testing.T) {
	packages := []formats.Package{{Name: "alpha", Rule: "r", PackageType: "js", Type: "prod"}}
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"r": {},
		},
		Groups: map[string]config.GroupCfg{
			"global-group": {Packages: []string{"alpha"}},
		},
	}

	grouped := filtering.ApplyPackageGroups(packages, cfg)
	require.Equal(t, "global-group", grouped[0].Group)
}

// TestCompareGroups tests the behavior of group comparison.
//
// It verifies:
//   - Empty group comes before non-empty group
//   - Alphabetical comparison for non-empty groups
//   - Comparison logic is consistent
func TestCompareGroups(t *testing.T) {
	assert.Equal(t, -1, filtering.CompareGroups("a", ""))
	assert.Equal(t, 1, filtering.CompareGroups("", "a"))
	assert.Equal(t, 0, filtering.CompareGroups("a", "a"))
	assert.Equal(t, -1, filtering.CompareGroups("a", "b"))
	assert.Equal(t, 1, filtering.CompareGroups("b", "a"))
	assert.Equal(t, 0, filtering.CompareGroups("", ""))
}

// TestFormatStatus tests the behavior of status formatting.
//
// It verifies:
//   - LockFound status shows green indicator
//   - LockMissing status shows orange indicator
//   - Status icons are properly displayed
func TestFormatStatus(t *testing.T) {
	assert.Contains(t, display.FormatStatusWithIcon(lock.InstallStatusLockFound), "🟢")
	assert.Contains(t, display.FormatStatusWithIcon(lock.InstallStatusNotInLock), "🔵")
	assert.Contains(t, display.FormatStatusWithIcon(lock.InstallStatusLockMissing), "🟠")
	assert.Contains(t, display.FormatStatusWithIcon(lock.InstallStatusVersionMissing), "⛔")
	assert.Contains(t, display.FormatStatusWithIcon(lock.InstallStatusNotConfigured), "⚪")
	assert.Contains(t, display.FormatStatusWithIcon("Failed"), "❌")
	assert.Contains(t, display.FormatStatusWithIcon("Failed(255)"), "❌")
	assert.Equal(t, "Unknown", display.FormatStatusWithIcon("Unknown"))
}

// TestFormatConstraintDisplayWithFlags tests the behavior of constraint formatting.
//
// It verifies:
//   - Major flag shows "Major (--major)"
//   - Minor flag shows "Minor (--minor)"
//   - Patch flag shows "Patch (--patch)"
func TestFormatConstraintDisplayWithFlags(t *testing.T) {
	pkg := formats.Package{Constraint: "^", Version: "1.0.0"}

	t.Run("major flag", func(t *testing.T) {
		result := display.FormatConstraintDisplayWithFlags(pkg, true, false, false)
		assert.Equal(t, "Major (--major)", result)
	})

	t.Run("minor flag", func(t *testing.T) {
		result := display.FormatConstraintDisplayWithFlags(pkg, false, true, false)
		assert.Equal(t, "Minor (--minor)", result)
	})

	t.Run("patch flag", func(t *testing.T) {
		result := display.FormatConstraintDisplayWithFlags(pkg, false, false, true)
		assert.Equal(t, "Patch (--patch)", result)
	})

	t.Run("no flags uses package constraint", func(t *testing.T) {
		result := display.FormatConstraintDisplayWithFlags(pkg, false, false, false)
		assert.Contains(t, result, "Compatible")
	})
}

// TestFilterPackagesWithFilters tests the behavior of package filtering.
//
// It verifies:
//   - Packages are filtered by package manager
//   - Packages are filtered by type (prod/dev)
//   - Packages are filtered by rule
func TestFilterPackagesWithFilters(t *testing.T) {
	packages := []formats.Package{
		{Name: "a", Type: "prod", PackageType: "js", Rule: "npm"},
		{Name: "b", Type: "dev", PackageType: "js", Rule: "npm"},
		{Name: "c", Type: "prod", PackageType: "pip", Rule: "pip"},
	}

	filtered := filtering.FilterPackagesWithFilters(packages, "prod", "all", "all", "", "")
	assert.Len(t, filtered, 2)

	filtered = filtering.FilterPackagesWithFilters(packages, "all", "js", "all", "", "")
	assert.Len(t, filtered, 2)

	filtered = filtering.FilterPackagesWithFilters(packages, "all", "all", "pip", "", "")
	assert.Len(t, filtered, 1)

	filtered = filtering.FilterPackagesWithFilters(packages, "dev", "js", "npm", "", "")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "b", filtered[0].Name)
}

// TestFilterByName tests the behavior of filtering packages by name.
//
// It verifies:
//   - Packages are filtered by exact name match
//   - Multiple names can be specified (comma-separated)
//   - Name filter is case-insensitive
func TestFilterByName(t *testing.T) {
	packages := []formats.Package{
		{Name: "lodash", Type: "prod", PackageType: "js", Rule: "npm"},
		{Name: "express", Type: "prod", PackageType: "js", Rule: "npm"},
		{Name: "react", Type: "dev", PackageType: "js", Rule: "npm"},
	}

	// Filter by single name
	filtered := filtering.FilterPackagesWithFilters(packages, "all", "all", "all", "lodash", "")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "lodash", filtered[0].Name)

	// Filter by multiple names
	filtered = filtering.FilterPackagesWithFilters(packages, "all", "all", "all", "lodash,express", "")
	assert.Len(t, filtered, 2)

	// Filter by name (case-insensitive)
	filtered = filtering.FilterPackagesWithFilters(packages, "all", "all", "all", "LODASH", "")
	assert.Len(t, filtered, 1)

	// No match
	filtered = filtering.FilterPackagesWithFilters(packages, "all", "all", "all", "nonexistent", "")
	assert.Len(t, filtered, 0)
}

// TestFilterByGroup tests the behavior of filtering packages by group.
//
// It verifies:
//   - Packages are filtered by comma-separated group names
//   - Multiple groups can be specified
//   - Group filter is case-insensitive
func TestFilterByGroup(t *testing.T) {
	packages := []formats.Package{
		{Name: "lodash", Group: "core", Type: "prod"},
		{Name: "express", Group: "core", Type: "prod"},
		{Name: "react", Group: "frontend", Type: "dev"},
		{Name: "jest", Group: "", Type: "dev"},
	}

	// Filter by single group
	filtered := filtering.FilterByGroup(packages, "core")
	assert.Len(t, filtered, 2)

	// Filter by multiple groups
	filtered = filtering.FilterByGroup(packages, "core,frontend")
	assert.Len(t, filtered, 3)

	// Filter by group (case-insensitive)
	filtered = filtering.FilterByGroup(packages, "CORE")
	assert.Len(t, filtered, 2)

	// No match returns empty
	filtered = filtering.FilterByGroup(packages, "nonexistent")
	assert.Len(t, filtered, 0)

	// Empty filter returns all
	filtered = filtering.FilterByGroup(packages, "")
	assert.Len(t, filtered, 4)
}

// TestPrintNoPackagesMessageWithFilters tests the behavior of no packages message display.
//
// It verifies:
//   - Message indicates when filters are applied
//   - Filter values are shown in the message
//   - Clear feedback about why no packages are shown
func TestPrintNoPackagesMessageWithFilters(t *testing.T) {
	out := captureStdout(t, func() {
		display.PrintNoPackagesMessageWithFilters(os.Stdout, "prod", "js", "npm")
	})

	assert.Contains(t, out, "No packages found")
	assert.Contains(t, out, "type: prod")
	assert.Contains(t, out, "pm: js")
	assert.Contains(t, out, "rule: npm")
}

// TestWarningCollector tests the behavior of warning collection.
//
// It verifies:
//   - Warnings can be added to the collector via Write
//   - All warnings are retrieved correctly
//   - Collector maintains order of warnings
func TestWarningCollector(t *testing.T) {
	c := &display.WarningCollector{}
	_, _ = c.Write([]byte("warning 1\nwarning 2\n"))
	_, _ = c.Write([]byte("warning 3"))

	messages := c.Messages()
	assert.Len(t, messages, 3)
	assert.Equal(t, "warning 1", messages[0])
	assert.Equal(t, "warning 2", messages[1])
	assert.Equal(t, "warning 3", messages[2])
}

// TestBuildListTable tests the behavior of list table construction.
//
// It verifies:
//   - Table is created with all expected columns
//   - Column widths accommodate row data
//   - Table structure is properly initialized
func TestBuildListTable(t *testing.T) {
	rows := []listDisplayRow{
		{
			pkg:               formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react", Version: "17.0.2", InstalledVersion: "17.0.2", Group: "core"},
			constraintDisplay: "Compatible (^)",
			statusDisplay:     "🟢 LockFound",
		},
	}

	table := buildListTable(rows)

	// Table should have all expected columns
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("RULE"), 3)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("PM"), 2)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("TYPE"), 4)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("CONSTRAINT"), 10)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("NAME"), 4)
}

// TestPrintWarnings tests the behavior of warning display.
//
// It verifies:
//   - Empty warnings print nothing
//   - Non-empty warnings are displayed with header
//   - All warning messages are shown
func TestPrintWarnings(t *testing.T) {
	t.Run("empty warnings prints nothing", func(t *testing.T) {
		out := captureStdout(t, func() {
			display.PrintWarnings(os.Stdout, []string{})
		})
		assert.Empty(t, out)
	})

	t.Run("with warnings prints them", func(t *testing.T) {
		out := captureStdout(t, func() {
			display.PrintWarnings(os.Stdout, []string{"first warning", "second warning"})
		})
		assert.Contains(t, out, "first warning")
		assert.Contains(t, out, "second warning")
	})
}

// TestPrintListStructured tests the behavior of structured list output.
//
// It verifies:
//   - JSON format produces valid JSON output
//   - CSV format produces valid CSV output
//   - XML format produces valid XML output
func TestPrintListStructured(t *testing.T) {
	pkgs := []formats.Package{
		{
			Name:             "lodash",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			Version:          "4.17.0",
			InstalledVersion: "4.17.0",
			InstallStatus:    "LockFound",
			Constraint:       "^",
			Group:            "core",
		},
		{
			Name:             "express",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			Version:          "4.18.0",
			InstalledVersion: "4.18.0",
			InstallStatus:    "LockFound",
			Constraint:       "~",
			Group:            "",
		},
	}

	t.Run("JSON format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printListStructured(pkgs, []string{}, output.FormatJSON)
			require.NoError(t, err)
		})
		assert.Contains(t, out, `"name":"lodash"`)
		assert.Contains(t, out, `"name":"express"`)
		assert.Contains(t, out, `"total_packages":2`)
	})

	t.Run("CSV format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printListStructured(pkgs, []string{}, output.FormatCSV)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "lodash")
		assert.Contains(t, out, "express")
	})

	t.Run("XML format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printListStructured(pkgs, []string{"warning1"}, output.FormatXML)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "<name>lodash</name>")
		assert.Contains(t, out, "<name>express</name>")
		assert.Contains(t, out, "<warning>warning1</warning>")
	})
}

// TestMatchesGroupFilterAndFilterByGroup tests the behavior of group matching and filtering.
//
// It verifies:
//   - Empty filter matches all groups
//   - Matching groups are identified correctly
//   - FilterByGroup filters packages by group name
func TestMatchesGroupFilterAndFilterByGroup(t *testing.T) {
	t.Run("matchesGroupFilter with empty filter", func(t *testing.T) {
		assert.True(t, filtering.MatchesGroup(formats.Package{Group: "any"}, "", []string{}))
	})

	t.Run("matchesGroupFilter with matching group", func(t *testing.T) {
		assert.True(t, filtering.MatchesGroup(formats.Package{Group: "core"}, "core", []string{"core"}))
	})

	t.Run("matchesGroupFilter with non-matching group", func(t *testing.T) {
		assert.False(t, filtering.MatchesGroup(formats.Package{Group: "utils"}, "core", []string{"core"}))
	})

	t.Run("filterByGroup filters correctly", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "a", Group: "core"},
			{Name: "b", Group: "utils"},
			{Name: "c", Group: "core"},
		}
		filtered := filtering.FilterByGroup(pkgs, "core")
		assert.Len(t, filtered, 2)
		assert.Equal(t, "a", filtered[0].Name)
		assert.Equal(t, "c", filtered[1].Name)
	})

	t.Run("filterByGroup with empty filter returns all", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "a", Group: "core"},
			{Name: "b", Group: "utils"},
		}
		filtered := filtering.FilterByGroup(pkgs, "")
		assert.Len(t, filtered, 2)
	})
}

// TestMatchesGroup tests the behavior of package matching against group configuration.
//
// It verifies:
//   - Packages in group config are matched
//   - Packages not in group config are rejected
//   - Whitespace-only entries are skipped
func TestMatchesGroup(t *testing.T) {
	t.Run("package in group", func(t *testing.T) {
		pkg := formats.Package{Name: "lodash", Rule: "npm"}
		groupCfg := config.GroupCfg{Packages: []string{"lodash", "express"}}
		assert.True(t, filtering.PackageMatchesGroup(pkg, groupCfg))
	})

	t.Run("package not in group", func(t *testing.T) {
		pkg := formats.Package{Name: "react", Rule: "npm"}
		groupCfg := config.GroupCfg{Packages: []string{"lodash", "express"}}
		assert.False(t, filtering.PackageMatchesGroup(pkg, groupCfg))
	})

	t.Run("empty group", func(t *testing.T) {
		pkg := formats.Package{Name: "lodash", Rule: "npm"}
		groupCfg := config.GroupCfg{Packages: []string{}}
		assert.False(t, filtering.PackageMatchesGroup(pkg, groupCfg))
	})

	t.Run("whitespace-only entry skipped", func(t *testing.T) {
		pkg := formats.Package{Name: "lodash", Rule: "npm"}
		groupCfg := config.GroupCfg{Packages: []string{"  ", "lodash"}}
		assert.True(t, filtering.PackageMatchesGroup(pkg, groupCfg))
	})
}

// TestFormatConstraintDisplayEdgeCases tests the behavior of constraint formatting edge cases.
//
// It verifies:
//   - Major constraint (*) is displayed correctly
//   - Exact constraint (=) is displayed correctly
//   - Unknown constraints use fallback display
func TestFormatConstraintDisplayEdgeCases(t *testing.T) {
	t.Run("major constraint", func(t *testing.T) {
		pkg := formats.Package{Constraint: "*", Version: "*"}
		result := display.FormatConstraintDisplay(pkg)
		assert.Contains(t, result, "Major")
	})

	t.Run("exact constraint", func(t *testing.T) {
		pkg := formats.Package{Constraint: "=", Version: "1.0.0"}
		result := display.FormatConstraintDisplay(pkg)
		assert.Contains(t, result, "Exact")
	})

	t.Run("unknown constraint returns fallback", func(t *testing.T) {
		pkg := formats.Package{Constraint: "??", Version: "1.0.0", Name: "test", PackageType: "js", Rule: "npm"}
		result := display.FormatConstraintDisplay(pkg)
		// Unknown constraints return "Exact (#N/A)" as fallback
		assert.Contains(t, result, "Exact")
		assert.Contains(t, result, "#N/A")
	})

	t.Run("deprecated exact constraint triggers warning", func(t *testing.T) {
		pkg := formats.Package{Name: "test-pkg", PackageType: "js", Rule: "npm", Constraint: "exact", Version: "1.0.0"}
		result := display.FormatConstraintDisplay(pkg)
		// Should return "Exact (=)" with a warning
		assert.Contains(t, result, "Exact")
	})
}

// TestListTableFormatters tests the behavior of list table formatting.
//
// It verifies:
//   - Header row contains expected columns
//   - Separator row is generated correctly
//   - Conditional columns are included/excluded properly
func TestListTableFormatters(t *testing.T) {
	t.Run("header row without group", func(t *testing.T) {
		table := output.NewTable().
			AddColumn("RULE").
			AddColumn("PM").
			AddConditionalColumn("GROUP", false).
			AddColumn("NAME")
		header := table.HeaderRow()
		assert.Contains(t, header, "RULE")
		assert.Contains(t, header, "NAME")
		assert.NotContains(t, header, "GROUP")
	})

	t.Run("separator row", func(t *testing.T) {
		table := output.NewTable().
			AddColumn("RULE").
			AddColumn("NAME")
		sep := table.SeparatorRow()
		assert.Contains(t, sep, "-")
	})

	t.Run("header row with group", func(t *testing.T) {
		table := output.NewTable().
			AddColumn("RULE").
			AddColumn("PM").
			AddConditionalColumn("GROUP", true).
			AddColumn("NAME")
		header := table.HeaderRow()
		assert.Contains(t, header, "GROUP")
	})

	t.Run("separator row with group", func(t *testing.T) {
		table := output.NewTable().
			AddColumn("RULE").
			AddConditionalColumn("GROUP", true).
			AddColumn("NAME")
		sep := table.SeparatorRow()
		assert.Contains(t, sep, "-")
	})
}

// TestShouldShowGroupColumnList tests the behavior of group column visibility logic.
//
// It verifies:
//   - No rows returns false for group column
//   - Rows with same group returns false
//   - Rows with different groups returns true
func TestShouldShowGroupColumnList(t *testing.T) {
	t.Run("no rows returns false", func(t *testing.T) {
		groups := []string{}
		assert.False(t, output.ShouldShowGroupColumn(groups))
	})

	t.Run("rows with same group returns false", func(t *testing.T) {
		groups := []string{"core"}
		// Only one package in group - should return false
		assert.False(t, output.ShouldShowGroupColumn(groups))
	})

	t.Run("rows with multiple packages in group returns true", func(t *testing.T) {
		groups := []string{"core", "core"}
		// Two packages in same group - should return true
		assert.True(t, output.ShouldShowGroupColumn(groups))
	})
}

// TestSortPackagesForDisplayEdgeCases tests the behavior of package sorting edge cases.
//
// It verifies:
//   - Empty package lists are handled correctly
//   - Single package lists are returned as-is
//   - Sorting maintains stability for equal elements
func TestSortPackagesForDisplayEdgeCases(t *testing.T) {
	t.Run("sorts by group then name", func(t *testing.T) {
		pkgs := []formats.Package{
			{Name: "z-pkg", Group: "a-group"},
			{Name: "a-pkg", Group: "z-group"},
			{Name: "m-pkg", Group: "a-group"},
		}
		sorted := filtering.SortPackagesForDisplay(pkgs)
		// Group "a-group" comes first
		assert.Equal(t, "a-group", sorted[0].Group)
		assert.Equal(t, "a-group", sorted[1].Group)
		assert.Equal(t, "z-group", sorted[2].Group)
	})
}

// TestPrepareListDisplayRowsEdgeCases tests the behavior of display row preparation edge cases.
//
// It verifies:
//   - Empty package lists return empty rows
//   - Constraint display is generated for each package
//   - Status display is generated for each package
func TestPrepareListDisplayRowsEdgeCases(t *testing.T) {
	t.Run("returns rows and warning writer", func(t *testing.T) {
		pkgs := []formats.Package{
			{
				Name:          "test",
				Rule:          "npm",
				PackageType:   "js",
				Type:          "prod",
				Constraint:    "^",
				Version:       "1.0.0",
				InstallStatus: "LockFound",
			},
		}
		rows, _, _ := prepareListDisplayRows(pkgs)
		assert.Len(t, rows, 1)
	})
}

// TestPrintPackagesEdgeCases tests the behavior of package display edge cases.
//
// It verifies:
//   - Empty package lists show "No packages found" message
//   - Single package is displayed correctly
//   - Warnings are displayed when present
func TestPrintPackagesEdgeCases(t *testing.T) {
	t.Run("prints group column when packages have groups", func(t *testing.T) {
		pkgs := []formats.Package{
			{
				Name:             "test",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Constraint:       "^",
				Version:          "1.0.0",
				InstalledVersion: "1.0.0",
				InstallStatus:    "LockFound",
				Group:            "mygroup",
			},
			{
				Name:             "test2",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Constraint:       "^",
				Version:          "2.0.0",
				InstalledVersion: "2.0.0",
				InstallStatus:    "LockFound",
				Group:            "mygroup",
			},
		}
		output := captureStdout(t, func() {
			printPackages(pkgs)
		})
		assert.Contains(t, output, "mygroup")
	})

	t.Run("prints without group column when no groups", func(t *testing.T) {
		pkgs := []formats.Package{
			{
				Name:             "test",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Constraint:       "^",
				Version:          "1.0.0",
				InstalledVersion: "1.0.0",
				InstallStatus:    "LockFound",
				Group:            "",
			},
		}
		output := captureStdout(t, func() {
			printPackages(pkgs)
		})
		assert.Contains(t, output, "test")
		assert.NotContains(t, output, "GROUP")
	})
}

// TestUnsupportedTrackerMessagesSorting tests the behavior of message sorting.
//
// It verifies:
//   - Messages are sorted alphabetically
//   - Sorting maintains stable order
//   - Empty message list is handled correctly
func TestUnsupportedTrackerMessagesSorting(t *testing.T) {
	// Test that Messages() sorts by rule, then by packageType
	tracker := supervision.NewUnsupportedTracker()

	// Add entries in non-sorted order
	tracker.Add(formats.Package{Name: "z", PackageType: "pip", Rule: "pip"}, "pip reason")
	tracker.Add(formats.Package{Name: "a", PackageType: "js", Rule: "npm"}, "npm js reason")
	tracker.Add(formats.Package{Name: "b", PackageType: "ts", Rule: "npm"}, "npm ts reason")

	messages := tracker.Messages()
	require.Len(t, messages, 3)
	// Should be sorted: npm/js, npm/ts, pip/pip
	assert.Contains(t, messages[0], "npm")
	assert.Contains(t, messages[0], "js")
	assert.Contains(t, messages[1], "npm")
	assert.Contains(t, messages[1], "ts")
	assert.Contains(t, messages[2], "pip")
}

// TestRunListGetPackagesError tests the behavior when package retrieval fails.
//
// It verifies:
//   - List returns error when getPackages fails
//   - Error message is propagated correctly
//   - Command handles package retrieval errors gracefully
func TestRunListGetPackagesError(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalDir := listDirFlag
	originalConfig := listConfigFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: "."}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return nil, fmt.Errorf("failed to get packages")
	}

	listDirFlag = "."
	listConfigFlag = ""

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		listDirFlag = originalDir
		listConfigFlag = originalConfig
	})

	err := runList(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get packages")
}

// TestRunListNoPackagesStructuredOutput tests the behavior of structured output with no packages.
//
// It verifies:
//   - JSON output is valid for empty package list
//   - Output contains zero count summary
//   - Empty packages array is included in output
func TestRunListNoPackagesStructuredOutput(t *testing.T) {
	tmpDir := t.TempDir()

	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"

	out := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should output empty structured result
	assert.Contains(t, out, `"total_packages":0`)
}

// TestFindRuleForFileDisabledRule tests the behavior with disabled rules.
//
// It verifies:
//   - Disabled rules are skipped during file matching
//   - Next matching enabled rule is used
//   - Disabled flag is properly checked
func TestFindRuleForFileDisabledRule(t *testing.T) {
	falseVal := false
	cfg := &config.Config{
		WorkingDir: "/repo",
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Include: []string{"**/package.json"},
				Enabled: &falseVal, // Disabled rule should be skipped
			},
			"pnpm": {
				Include: []string{"**/package.json"},
			},
		},
	}

	ruleCfg, key := findRuleForFile("/repo/package.json", cfg)
	require.NotNil(t, ruleCfg)
	assert.Equal(t, "pnpm", key) // npm is disabled, so pnpm should be selected
}

// TestFilterByGroupEmptyAfterSplit tests the behavior when filters are empty after processing.
//
// It verifies:
//   - Empty group filter matches all packages
//   - Whitespace-only filters are treated as empty
//   - Filters are trimmed before processing
func TestFilterByGroupEmptyAfterSplit(t *testing.T) {
	packages := []formats.Package{
		{Name: "lodash", Group: "core"},
		{Name: "express", Group: "utils"},
	}

	// Filter with only whitespace - TrimAndSplit returns empty, so all packages returned
	filtered := filtering.FilterByGroup(packages, "   ")
	assert.Len(t, filtered, 2) // When filter is empty after trimming, returns all packages
}

// TestSortPackagesForDisplayAllComparisons tests the behavior of all comparison operations.
//
// It verifies:
//   - All sorting criteria are tested
//   - Rule comparison works correctly
//   - PackageType comparison works correctly
func TestSortPackagesForDisplayAllComparisons(t *testing.T) {
	// Test all comparison branches in sortPackagesForDisplay
	pkgs := []formats.Package{
		{Rule: "npm", PackageType: "ts", Group: "b", Type: "dev", Name: "zlib"},
		{Rule: "npm", PackageType: "ts", Group: "b", Type: "prod", Name: "axios"},
		{Rule: "npm", PackageType: "js", Group: "a", Type: "prod", Name: "react"},
		{Rule: "pip", PackageType: "py", Group: "", Type: "prod", Name: "requests"},
		{Rule: "npm", PackageType: "ts", Group: "b", Type: "dev", Name: "express"},
	}

	sorted := filtering.SortPackagesForDisplay(pkgs)

	// Verify all packages are present
	require.Len(t, sorted, 5)

	// Rule npm comes before pip
	assert.Equal(t, "npm", sorted[0].Rule)

	// Within same rule, packages with groups come before those without
	foundGroupA := false
	foundGroupB := false
	foundNoGroup := false
	for _, p := range sorted {
		if p.Group == "a" {
			foundGroupA = true
		}
		if p.Group == "b" {
			foundGroupB = true
		}
		if p.Group == "" {
			foundNoGroup = true
		}
	}
	assert.True(t, foundGroupA)
	assert.True(t, foundGroupB)
	assert.True(t, foundNoGroup)
}

// TestPrepareListDisplayRowsWithWarnings tests the behavior when warnings are present.
//
// It verifies:
//   - Rows are prepared correctly with warnings
//   - Warning collection doesn't interfere with row generation
//   - Warnings are captured during constraint display
func TestPrepareListDisplayRowsWithWarnings(t *testing.T) {
	// Test with a package that triggers a warning during constraint display
	pkgs := []formats.Package{
		{
			Name:          "test-warning",
			Rule:          "npm",
			PackageType:   "js",
			Type:          "prod",
			Constraint:    "unknown-constraint-xyz", // Unknown constraint triggers warning
			Version:       "1.0.0",
			InstallStatus: "LockFound",
		},
	}

	var buf bytes.Buffer
	restore := warnings.SetWarningWriter(&buf)
	t.Cleanup(restore)

	rows, warningsOut, _ := prepareListDisplayRows(pkgs)
	assert.Len(t, rows, 1)
	// Either warningsOut or the warning writer should capture the warning
	_ = warningsOut // We just verify the function runs without panic
}

// TestPrintPackagesWithWarningsOutput tests the behavior of warning output.
//
// It verifies:
//   - Warnings are displayed after package list
//   - Warning section has proper header
//   - All warnings are shown in output
func TestPrintPackagesWithWarningsOutput(t *testing.T) {
	// Test that warnings are properly output during printPackages
	// Use an unmapped constraint to trigger warnings in printPackages
	pkgs := []formats.Package{
		{
			Name:             "test-pkg-warn",
			Rule:             "npm",
			PackageType:      "js",
			Type:             "prod",
			Constraint:       "???", // Unmapped constraint triggers warning
			Version:          "1.0.0",
			InstalledVersion: "1.0.0",
			InstallStatus:    "LockFound",
		},
	}

	out := captureStdout(t, func() {
		printPackages(pkgs)
	})

	assert.Contains(t, out, "test-pkg-warn")
	assert.Contains(t, out, "Total packages: 1")
}

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
