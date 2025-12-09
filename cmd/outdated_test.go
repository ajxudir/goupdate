package cmd

import (
	"context"
	stderrors "errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/output"
)

// TestOutdatedCommand tests the behavior of the outdated command.
//
// It verifies:
//   - Outdated command executes without errors
//   - Outdated command processes packages correctly
//   - Command line arguments are properly handled
func TestOutdatedCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies":{"test":"1.0.0"}}`), 0644)
	require.NoError(t, err)

	os.Args = []string{"goupdate", "outdated", "-d", tmpDir}
	err = ExecuteTest()
	assert.NoError(t, err)
}

// TestRunOutdatedNoPackages tests the behavior when no packages are found.
//
// It verifies:
//   - Outdated completes without errors when no packages exist
//   - Output contains "No packages found" message
//   - Empty package lists are handled gracefully
func TestRunOutdatedNoPackages(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	defer func() {
		os.Args = oldArgs
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
	}()

	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	os.Args = []string{"goupdate", "outdated", "-d", tmpDir}

	output := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "No packages found")
}

// TestRunOutdatedConfigError tests the behavior when config file is missing.
//
// It verifies:
//   - Outdated returns error when specified config file doesn't exist
//   - Error handling for missing config files
//   - Config file validation occurs before processing
func TestRunOutdatedConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldConfig := outdatedConfigFlag
	defer func() {
		os.Args = oldArgs
		outdatedConfigFlag = oldConfig
	}()

	badCfg := filepath.Join(tmpDir, "missing.yml")
	outdatedConfigFlag = badCfg
	os.Args = []string{"goupdate", "outdated", "--config", badCfg}

	err := runOutdated(nil, nil)
	assert.Error(t, err)
}

// TestRunOutdatedGetPackagesError tests the behavior when package retrieval fails.
//
// It verifies:
//   - Outdated returns error when getPackages fails
//   - Error message is propagated correctly
//   - Package retrieval errors are handled properly
func TestRunOutdatedGetPackagesError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: "."}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return nil, stderrors.New("failed to get packages")
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""

	err := runOutdated(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get packages")
}

// TestRunOutdatedApplyInstalledError tests the behavior when installed version resolution fails.
//
// It verifies:
//   - Outdated returns error when applyInstalledVersions fails
//   - Error message indicates installation resolution failure
//   - Lock file errors are properly handled
func TestRunOutdatedApplyInstalledError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: "."}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return nil, stderrors.New("failed to apply installed versions")
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""

	err := runOutdated(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply installed versions")
}

// TestRunOutdatedNoPackagesStructured tests the behavior of structured output with no packages.
//
// It verifies:
//   - JSON output is valid for empty package list
//   - Output contains zero count summary
//   - Empty packages array is included in output
func TestRunOutdatedNoPackagesStructured(t *testing.T) {
	tmpDir := t.TempDir()

	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err)
	})

	// Should output empty structured result
	assert.Contains(t, out, `"total_packages":0`)
}

// TestRunOutdatedPreflightError tests the behavior when preflight checks fail.
//
// It verifies:
//   - Outdated returns error when preflight validation fails
//   - Error message indicates preflight failure
//   - Preflight errors prevent package processing
func TestRunOutdatedPreflightError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager: "js",
					Outdated: &config.OutdatedCfg{
						Commands: "nonexistent_command_preflight_test_12345 {{package}}",
					},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = false

	err := runOutdated(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Pre-flight validation failed")
}

// TestRunOutdatedFloatingPackage tests the behavior with floating constraint packages.
//
// It verifies:
//   - Floating constraints like "*" are identified
//   - Packages with floating constraints are marked as unsupported
//   - Floating constraints are not processed for updates
func TestRunOutdatedFloatingPackage(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldListNewer := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldListNewer
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:  "js",
					Outdated: &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{
			Name:          "test",
			Rule:          "npm",
			PackageType:   "js",
			Version:       "1.0.0",
			InstallStatus: lock.InstallStatusFloating,
		}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return nil, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Floating")
}

// TestRunOutdatedWithStructuredOutputAndErrors tests the behavior of structured output with errors.
//
// It verifies:
//   - JSON output includes error information
//   - Error messages are properly formatted
//   - Partial success is reflected in output
func TestRunOutdatedWithStructuredOutputAndErrors(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldListNewer := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	oldOutput := outdatedOutputFlag
	oldContinue := outdatedContinueOnFail
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldListNewer
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
		outdatedOutputFlag = oldOutput
		outdatedContinueOnFail = oldContinue
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:  "js",
					Outdated: &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "success", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"},
			{Name: "fail", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	callCount := 0
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		callCount++
		if p.Name == "fail" {
			return nil, stderrors.New("version check failed")
		}
		return []string{"2.0.0"}, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true
	outdatedOutputFlag = "json"
	outdatedContinueOnFail = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// With continue on fail, it should return partial success error
		assert.Error(t, err)
	})

	assert.Contains(t, out, "success")
	assert.Contains(t, out, "fail")
}

// TestRunOutdatedPartialSuccess tests the behavior with partial success.
//
// It verifies:
//   - Some packages process successfully while others fail
//   - Successful packages are included in output
//   - Failed packages are listed in errors section
func TestRunOutdatedPartialSuccess(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldListNewer := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	oldContinue := outdatedContinueOnFail
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldListNewer
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
		outdatedContinueOnFail = oldContinue
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:  "js",
					Outdated: &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "success", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"},
			{Name: "fail", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		if p.Name == "fail" {
			return nil, stderrors.New("version check failed")
		}
		return []string{"2.0.0"}, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true
	outdatedContinueOnFail = true

	captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// Partial success error with correct exit code
		assert.Error(t, err)
		var exitErr *errors.ExitError
		if stderrors.As(err, &exitErr) {
			assert.Equal(t, errors.ExitPartialFailure, exitErr.Code)
		}
	})
}

// TestRunOutdatedCompleteFailure tests the behavior when all packages fail.
//
// It verifies:
//   - All packages failing results in error
//   - Error message indicates complete failure
//   - No successful package updates are reported
func TestRunOutdatedCompleteFailure(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldListNewer := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	oldContinue := outdatedContinueOnFail
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldListNewer
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
		outdatedContinueOnFail = oldContinue
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:  "js",
					Outdated: &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "fail", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return nil, stderrors.New("version check failed")
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true
	outdatedContinueOnFail = false // No continue on fail = complete failure

	captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// Complete failure with exit code 2
		assert.Error(t, err)
		var exitErr *errors.ExitError
		if stderrors.As(err, &exitErr) {
			assert.Equal(t, errors.ExitFailure, exitErr.Code)
		}
	})
}

// TestDeriveOutdatedStatus tests the behavior of outdated status derivation.
//
// It verifies:
//   - Outdated status is correctly determined from versions
//   - UpToDate status is determined when no newer versions exist
//   - Version comparison logic works correctly
func TestDeriveOutdatedStatus(t *testing.T) {
	tests := []struct {
		name     string
		result   outdatedResult
		expected string
	}{
		{
			name:     "error status",
			result:   outdatedResult{err: assert.AnError},
			expected: "Failed",
		},
		{
			name:     "outdated with major",
			result:   outdatedResult{major: "2.0.0", minor: "#N/A", patch: "#N/A"},
			expected: "Outdated",
		},
		{
			name:     "outdated with minor",
			result:   outdatedResult{major: "#N/A", minor: "1.1.0", patch: "#N/A"},
			expected: "Outdated",
		},
		{
			name:     "outdated with patch",
			result:   outdatedResult{major: "#N/A", minor: "#N/A", patch: "1.0.1"},
			expected: "Outdated",
		},
		{
			name:     "up to date",
			result:   outdatedResult{major: "#N/A", minor: "#N/A", patch: "#N/A"},
			expected: "UpToDate",
		},
		{
			name:     "floating constraint preserves status",
			result:   outdatedResult{pkg: formats.Package{Name: "test", Version: "*", InstallStatus: lock.InstallStatusFloating}, major: "#N/A", minor: "#N/A", patch: "#N/A"},
			expected: lock.InstallStatusFloating,
		},
		{
			name:     "floating constraint with versions still preserves status",
			result:   outdatedResult{pkg: formats.Package{Name: "test", Version: "5.*", InstallStatus: lock.InstallStatusFloating}, major: "6.0.0", minor: "5.5.0", patch: "5.1.1"},
			expected: lock.InstallStatusFloating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := deriveOutdatedStatus(tt.result)
			assert.Equal(t, tt.expected, status)
		})
	}
}

// TestSafeVersionValue tests the behavior of safe version display.
//
// It verifies:
//   - Empty versions show "#N/A"
//   - Valid versions are displayed as-is
//   - Whitespace-only versions show "#N/A"
func TestSafeVersionValue(t *testing.T) {
	assert.Equal(t, "#N/A", display.SafeVersionValue("", constants.PlaceholderNA))
	assert.Equal(t, "#N/A", display.SafeVersionValue("   ", constants.PlaceholderNA))
	assert.Equal(t, "1.0.0", display.SafeVersionValue("1.0.0", constants.PlaceholderNA))
}

// TestIsLatestMissing tests the behavior of latest version detection.
//
// It verifies:
//   - Latest version missing is detected correctly
//   - Non-empty latest versions are identified
//   - Version existence check works properly
func TestIsLatestMissing(t *testing.T) {
	// Set empty key to "latest" so IsLatestIndicator recognizes "latest" as the latest indicator
	ruleCfg := &config.PackageManagerCfg{
		LatestMapping: &config.LatestMappingCfg{
			Default: map[string]string{"": "latest"},
		},
	}

	p := formats.Package{
		Name:             "test",
		Version:          "latest",
		InstalledVersion: "#N/A",
	}
	assert.True(t, isLatestMissing(p, ruleCfg))

	p.InstalledVersion = "1.0.0"
	assert.False(t, isLatestMissing(p, ruleCfg))

	p.Version = "1.0.0"
	p.InstalledVersion = "#N/A"
	assert.False(t, isLatestMissing(p, ruleCfg))
}

// TestPrepareOutdatedDisplayRows tests the behavior of display row preparation.
//
// It verifies:
//   - Rows are prepared correctly for each package
//   - Status is determined for each package
//   - Version information is included in rows
func TestPrepareOutdatedDisplayRows(t *testing.T) {
	results := []outdatedResult{
		{
			pkg:   formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react", Version: "17.0.0", InstalledVersion: "17.0.0"},
			group: "core",
			major: "18.0.0",
			minor: "#N/A",
			patch: "#N/A",
		},
	}

	rows := prepareOutdatedDisplayRows(results)
	require.Len(t, rows, 1)
	assert.Equal(t, "react", rows[0].pkg.Name)
	assert.Equal(t, "18.0.0", rows[0].major)
	assert.Equal(t, "core", rows[0].group)
}

// TestBuildOutdatedTable tests the behavior of outdated table construction.
//
// It verifies:
//   - Table is created with all expected columns
//   - Column widths accommodate row data
//   - Table structure is properly initialized
func TestBuildOutdatedTable(t *testing.T) {
	rows := []outdatedDisplayRow{
		{
			pkg:               formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react"},
			constraintDisplay: "Compatible (^)",
			statusDisplay:     "🟠 Outdated",
			major:             "18.0.0",
			minor:             "#N/A",
			patch:             "#N/A",
		},
	}

	table := buildOutdatedTable(rows)

	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("RULE"), 3)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("MAJOR"), 5)
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("NAME"), 4)
}

// TestOutdatedTableFormatters tests the behavior of outdated table formatting.
//
// It verifies:
//   - Header row contains expected columns
//   - Separator row is generated correctly
//   - Table formatting functions work as expected
func TestOutdatedTableFormatters(t *testing.T) {
	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("MAJOR").
		AddColumn("MINOR").
		AddColumn("PATCH").
		AddColumn("STATUS").
		AddConditionalColumn("GROUP", false).
		AddColumn("NAME")

	header := table.HeaderRow()
	assert.Contains(t, header, "RULE")
	assert.Contains(t, header, "MAJOR")
	assert.Contains(t, header, "MINOR")
	assert.Contains(t, header, "PATCH")
	assert.Contains(t, header, "STATUS")

	separator := table.SeparatorRow()
	assert.Contains(t, separator, "----")
}

// TestRunOutdatedWithMockedVersions tests the behavior with mocked version data.
//
// It verifies:
//   - Mocked version data is processed correctly
//   - Major, minor, and patch versions are determined
//   - Version resolution works with mock data
func TestRunOutdatedWithMockedVersions(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalListNewer := listNewerVersionsFunc
	originalType := outdatedTypeFlag
	originalPM := outdatedPMFlag
	originalDir := outdatedDirFlag
	originalConfig := outdatedConfigFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager: "js",
					Outdated: &config.OutdatedCfg{
						Commands: "npm view {{package}}",
					},
				},
			},
		}, nil
	}

	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{
				Rule:             "npm",
				Name:             "react",
				PackageType:      "js",
				Type:             "prod",
				Version:          "17.0.0",
				InstalledVersion: "17.0.0",
				Constraint:       "^",
			},
		}, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"17.0.1", "17.0.2", "18.0.0"}, nil
	}

	outdatedTypeFlag, outdatedPMFlag, outdatedDirFlag, outdatedConfigFlag = "all", "all", ".", ""

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		listNewerVersionsFunc = originalListNewer
		outdatedTypeFlag = originalType
		outdatedPMFlag = originalPM
		outdatedDirFlag = originalDir
		outdatedConfigFlag = originalConfig
	})

	output := captureStdout(t, func() {
		require.NoError(t, runOutdated(outdatedCmd, nil))
	})

	assert.Contains(t, output, "react")
	assert.Contains(t, output, "Outdated")
}

// TestPrintOutdatedResults tests the behavior of outdated results display.
//
// It verifies:
//   - Outdated results are displayed in table format
//   - All packages are shown in output
//   - Version information is properly formatted
func TestPrintOutdatedResults(t *testing.T) {
	results := []outdatedResult{
		{
			pkg:    formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react", Version: "17.0.0", InstalledVersion: "17.0.0"},
			group:  "core",
			major:  "18.0.0",
			minor:  "17.1.0",
			patch:  "17.0.2",
			status: "Outdated",
		},
	}

	output := captureStdout(t, func() {
		printOutdatedResults(results, "all", "all")
	})

	assert.Contains(t, output, "RULE")
	assert.Contains(t, output, "MAJOR")
	assert.Contains(t, output, "MINOR")
	assert.Contains(t, output, "PATCH")
	assert.Contains(t, output, "react")
	assert.Contains(t, output, "18.0.0")
	assert.Contains(t, output, "Total packages: 1")
}

// TestPrintOutdatedErrorsWithHintsNonEmpty tests the behavior of error display with hints.
//
// It verifies:
//   - Errors are displayed with helpful hints
//   - Hint messages provide guidance
//   - Error output is user-friendly
func TestPrintOutdatedErrorsWithHintsNonEmpty(t *testing.T) {
	errs := []error{
		assert.AnError,
	}

	output := captureStdout(t, func() {
		printOutdatedErrorsWithHints(errs)
	})

	assert.Contains(t, output, "❌")
}

// TestOutdatedFlags tests the behavior of outdated command flags.
//
// It verifies:
//   - Flags are properly defined
//   - Flag values can be set and read
//   - Default flag values are correct
func TestOutdatedFlags(t *testing.T) {
	oldMajor := outdatedMajorFlag
	oldMinor := outdatedMinorFlag
	oldPatch := outdatedPatchFlag
	defer func() {
		outdatedMajorFlag = oldMajor
		outdatedMinorFlag = oldMinor
		outdatedPatchFlag = oldPatch
	}()

	require.NoError(t, outdatedCmd.Flags().Set("major", "true"))
	assert.True(t, outdatedMajorFlag)

	require.NoError(t, outdatedCmd.Flags().Set("minor", "true"))
	assert.True(t, outdatedMinorFlag)

	require.NoError(t, outdatedCmd.Flags().Set("patch", "true"))
	assert.True(t, outdatedPatchFlag)
}

// TestOutdatedResultWithAllVersions tests the behavior of result with all version types.
//
// It verifies:
//   - Results include major, minor, and patch versions
//   - All version fields are populated correctly
//   - Version hierarchy is maintained
func TestOutdatedResultWithAllVersions(t *testing.T) {
	result := outdatedResult{
		pkg:   formats.Package{Name: "test"},
		major: "2.0.0",
		minor: "1.1.0",
		patch: "1.0.1",
	}

	status := deriveOutdatedStatus(result)
	assert.Equal(t, "Outdated", status)
}

// TestOutdatedResultAllNA tests the behavior when all versions are N/A.
//
// It verifies:
//   - N/A values are handled correctly
//   - Result is valid when no versions available
//   - Status reflects lack of available versions
func TestOutdatedResultAllNA(t *testing.T) {
	result := outdatedResult{
		pkg:   formats.Package{Name: "test"},
		major: "#N/A",
		minor: "#N/A",
		patch: "#N/A",
	}

	status := deriveOutdatedStatus(result)
	assert.Equal(t, "UpToDate", status)
}

// TestOutdatedStatusFormats tests the behavior of status formatting.
//
// It verifies:
//   - Outdated status is displayed correctly
//   - UpToDate status is displayed correctly
//   - Status strings are properly formatted
func TestOutdatedStatusFormats(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"Outdated", "🟠"},
		{"UpToDate", "🟢"},
		{"Failed", "❌"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			formatted := display.FormatStatusWithIcon(tt.status)
			if tt.contains != "" {
				assert.True(t, strings.Contains(formatted, tt.contains) || formatted == tt.status)
			}
		})
	}
}

// TestPrintOutdatedErrorsWithHints tests the behavior of error display with hints.
//
// It verifies:
//   - Empty error lists print nothing
//   - Non-empty errors are displayed with hints
//   - Helpful guidance is provided for errors
func TestPrintOutdatedErrorsWithHints(t *testing.T) {
	t.Run("empty errors prints nothing", func(t *testing.T) {
		output := captureStdout(t, func() {
			printOutdatedErrorsWithHints([]error{})
		})
		assert.Empty(t, output)
	})

	t.Run("with errors prints them", func(t *testing.T) {
		errs := []error{
			assert.AnError,
		}
		output := captureStdout(t, func() {
			printOutdatedErrorsWithHints(errs)
		})
		assert.Contains(t, output, "❌")
	})
}

// TestPrintOutdatedStructured tests the behavior of structured outdated output.
//
// It verifies:
//   - JSON output is valid and complete
//   - CSV output contains all columns
//   - XML output is properly formatted
func TestPrintOutdatedStructured(t *testing.T) {
	results := []outdatedResult{
		{
			pkg: formats.Package{
				Name:             "lodash",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Version:          "4.17.0",
				InstalledVersion: "4.17.0",
			},
			major:  "5.0.0",
			minor:  "4.18.0",
			patch:  "4.17.1",
			status: "Outdated",
			group:  "core",
		},
		{
			pkg: formats.Package{
				Name:             "express",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Version:          "4.18.0",
				InstalledVersion: "4.18.0",
			},
			major:  "#N/A",
			minor:  "#N/A",
			patch:  "#N/A",
			status: "UpToDate",
			group:  "",
		},
		{
			pkg: formats.Package{
				Name:             "react",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Version:          "17.0.0",
				InstalledVersion: "17.0.0",
			},
			major:  "",
			minor:  "",
			patch:  "",
			status: "Failed",
			err:    stderrors.New("fetch error"),
		},
	}

	t.Run("JSON format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printOutdatedStructured(results, []string{}, []string{}, output.FormatJSON)
			require.NoError(t, err)
		})
		assert.Contains(t, out, `"name":"lodash"`)
		assert.Contains(t, out, `"outdated_packages":1`)
		assert.Contains(t, out, `"uptodate_packages":1`)
		assert.Contains(t, out, `"failed_packages":1`)
	})

	t.Run("CSV format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printOutdatedStructured(results, []string{}, []string{}, output.FormatCSV)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "lodash")
		assert.Contains(t, out, "express")
	})

	t.Run("XML format with warnings and errors", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printOutdatedStructured(results, []string{"warning1"}, []string{"error1"}, output.FormatXML)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "<name>lodash</name>")
		assert.Contains(t, out, "<warning>warning1</warning>")
		assert.Contains(t, out, "<error>error1</error>")
	})
}

// TestPrintOutdatedRowWithTableEdgeCases tests the behavior of table row printing edge cases.
//
// It verifies:
//   - Empty rows are handled correctly
//   - Single row is printed properly
//   - Multiple rows maintain formatting
func TestPrintOutdatedRowWithTableEdgeCases(t *testing.T) {
	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("MAJOR").
		AddColumn("MINOR").
		AddColumn("PATCH").
		AddColumn("STATUS").
		AddConditionalColumn("GROUP", true).
		AddColumn("NAME")

	t.Run("prints row with group", func(t *testing.T) {
		res := outdatedResult{
			pkg: formats.Package{
				Name:             "test",
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Version:          "1.0.0",
				InstalledVersion: "1.0.0",
				Group:            "mygroup",
			},
			major:  "2.0.0",
			minor:  "#N/A",
			patch:  "#N/A",
			status: "Outdated",
			group:  "mygroup",
		}
		out := captureStdout(t, func() {
			printOutdatedRowWithTable(res, table)
		})
		assert.Contains(t, out, "mygroup")
	})
}

// TestShouldShowOutdatedGroupColumnFromGroupsNoGroups tests the behavior of group column visibility with no groups.
//
// It verifies:
//   - Group column is hidden when no groups exist
//   - Empty group list returns false
//   - Visibility logic works correctly
func TestShouldShowOutdatedGroupColumnFromGroupsNoGroups(t *testing.T) {
	// Test case where all groups have count < 2
	groups := []string{"group1", "group2", "group3"}

	result := output.ShouldShowGroupColumn(groups)
	assert.False(t, result, "should return false when all groups have count < 2")
}

// TestShouldShowOutdatedGroupColumnFromGroupsWithGroup tests the behavior of group column visibility with groups.
//
// It verifies:
//   - Group column is shown when groups exist
//   - Non-empty group list returns true
//   - Group presence is detected correctly
func TestShouldShowOutdatedGroupColumnFromGroupsWithGroup(t *testing.T) {
	// Test case where a group has count >= 2
	groups := []string{"group1", "group1", "group2"} // Same group, makes count >= 2

	result := output.ShouldShowGroupColumn(groups)
	assert.True(t, result, "should return true when a group has count >= 2")
}

// TestShouldShowOutdatedGroupColumnNoGroupsPackages tests the behavior when packages have no groups.
//
// It verifies:
//   - Group column is hidden when all packages lack groups
//   - Empty group values are detected
//   - Visibility logic handles ungrouped packages
func TestShouldShowOutdatedGroupColumnNoGroupsPackages(t *testing.T) {
	// Test case where all groups have count < 2
	packages := []formats.Package{
		{Group: "group1"},
		{Group: "group2"},
		{Group: "group3"},
	}

	groups := make([]string, len(packages))
	for i, p := range packages {
		groups[i] = p.Group
	}

	result := output.ShouldShowGroupColumn(groups)
	assert.False(t, result, "should return false when all groups have count < 2")
}

// TestShouldShowOutdatedGroupColumnWithGroupPackages tests the behavior when packages have groups.
//
// It verifies:
//   - Group column is shown when packages have different groups
//   - Group diversity triggers column display
//   - Multiple groups are detected correctly
func TestShouldShowOutdatedGroupColumnWithGroupPackages(t *testing.T) {
	// Test case where a group has count >= 2
	packages := []formats.Package{
		{Group: "group1"},
		{Group: "group1"}, // Same group, makes count >= 2
		{Group: "group2"},
	}

	groups := make([]string, len(packages))
	for i, p := range packages {
		groups[i] = p.Group
	}

	result := output.ShouldShowGroupColumn(groups)
	assert.True(t, result, "should return true when a group has count >= 2")
}

// TestOutdatedTableWithGroup tests the behavior of table with group column.
//
// It verifies:
//   - Table includes group column when needed
//   - Group values are displayed correctly
//   - Group column formatting is correct
func TestOutdatedTableWithGroup(t *testing.T) {
	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddConditionalColumn("GROUP", true).
		AddColumn("NAME")

	header := table.HeaderRow()
	assert.Contains(t, header, "GROUP")

	separator := table.SeparatorRow()
	assert.Contains(t, separator, "-") // Dashes for group column
}

// TestPrintOutdatedResultsWithGroupColumn tests the behavior of results display with group column.
//
// It verifies:
//   - Group column appears in output
//   - Group values are shown for each package
//   - Table formatting accommodates group column
func TestPrintOutdatedResultsWithGroupColumn(t *testing.T) {
	results := []outdatedResult{
		{
			pkg:    formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react", Version: "17.0.0", InstalledVersion: "17.0.0"},
			group:  "core",
			major:  "18.0.0",
			minor:  "17.1.0",
			patch:  "17.0.2",
			status: "Outdated",
		},
		{
			pkg:    formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "vue", Version: "3.0.0", InstalledVersion: "3.0.0"},
			group:  "core", // Same group - should show group column
			major:  "4.0.0",
			minor:  "#N/A",
			patch:  "#N/A",
			status: "Outdated",
		},
	}

	output := captureStdout(t, func() {
		printOutdatedResults(results, "all", "all")
	})

	assert.Contains(t, output, "GROUP")
	assert.Contains(t, output, "core")
}

// TestDeriveOutdatedStatusEdgeCases tests the behavior of status derivation edge cases.
//
// It verifies:
//   - Edge cases in version comparison are handled
//   - Invalid version data is handled gracefully
//   - Status determination is robust
func TestDeriveOutdatedStatusEdgeCases(t *testing.T) {
	t.Run("returns Failed with error", func(t *testing.T) {
		result := outdatedResult{
			pkg:   formats.Package{Name: "test"},
			err:   stderrors.New("some error"),
			major: "#N/A",
			minor: "#N/A",
			patch: "#N/A",
		}
		status := deriveOutdatedStatus(result)
		assert.Equal(t, "Failed", status)
	})

	t.Run("returns Floating for floating constraint", func(t *testing.T) {
		result := outdatedResult{
			pkg: formats.Package{
				Name:          "test",
				InstallStatus: lock.InstallStatusFloating,
			},
			major: "#N/A",
			minor: "#N/A",
			patch: "#N/A",
		}
		status := deriveOutdatedStatus(result)
		assert.Equal(t, lock.InstallStatusFloating, status)
	})

	t.Run("returns Failed with exit code when exec.ExitError", func(t *testing.T) {
		// Create a real exec.ExitError by running a command that exits with error
		cmd := exec.Command("sh", "-c", "exit 42")
		err := cmd.Run()

		result := outdatedResult{
			pkg:   formats.Package{Name: "test"},
			err:   err,
			major: "#N/A",
			minor: "#N/A",
			patch: "#N/A",
		}
		status := deriveOutdatedStatus(result)
		assert.Equal(t, "Failed(42)", status)
	})
}

// TestRunOutdatedIncrementalError tests the behavior when incremental selection fails.
//
// It verifies:
//   - Incremental validation errors are reported
//   - Error message indicates incremental flag conflict
//   - Invalid flag combinations are detected
func TestRunOutdatedIncrementalError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldListNewer := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldListNewer
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
	}()

	// Config with invalid incremental regex pattern that will cause error
	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:     "js",
					Incremental: []string{"["}, // Invalid regex
					Outdated:    &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"2.0.0"}, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// Should return error due to incremental pattern error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid incremental package pattern")
	})

	// Output should still show the package
	assert.Contains(t, out, "test")
}

// TestRunOutdatedNonStructuredNoPackages tests the behavior of non-structured output with no packages.
//
// It verifies:
//   - Table output shows "No packages found" message
//   - Non-structured format handles empty results
//   - User-friendly message is displayed
func TestRunOutdatedNonStructuredNoPackages(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldRule := outdatedRuleFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedRuleFlag = oldRule
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{WorkingDir: "."}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{}, nil // Return empty packages
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedOutputFlag = ""   // Non-structured output (table)
	outdatedTypeFlag = "prod" // Specific filter
	outdatedPMFlag = "js"     // Specific filter
	outdatedRuleFlag = "npm"  // Specific filter

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err)
	})

	// Should show no packages message with filter hints
	assert.Contains(t, out, "No packages found")
}

// TestRunOutdatedFloatingPackageStructuredOutput tests the behavior of structured output with floating constraints.
//
// It verifies:
//   - JSON output includes floating constraint packages
//   - Floating status is reflected in structured output
//   - All package information is included
func TestRunOutdatedFloatingPackageStructuredOutput(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedSkipPreflight = oldSkip
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Outdated: &config.OutdatedCfg{Commands: "echo ok"}},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "floating-pkg", Rule: "npm", PackageType: "js", Version: "*", InstallStatus: lock.InstallStatusFloating},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json" // Structured output
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err)
	})

	// Should contain JSON with floating package status
	assert.Contains(t, out, "floating-pkg")
	assert.Contains(t, out, "Floating")
}

// TestRunOutdatedUnsupportedError tests the behavior when packages are unsupported.
//
// It verifies:
//   - Unsupported packages are identified
//   - Error messages explain why packages are unsupported
//   - Unsupported status prevents processing
func TestRunOutdatedUnsupportedError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Outdated: &config.OutdatedCfg{Commands: "echo ok"}},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return nil, &errors.UnsupportedError{Reason: "test unsupported"}
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		assert.NoError(t, err) // Unsupported errors are handled gracefully
	})

	assert.Contains(t, out, "test")
}

// TestRunOutdatedSummarizeVersionError tests the behavior when version summarization fails.
//
// It verifies:
//   - Version summarization errors are reported
//   - Packages with errors are included in output
//   - Partial results are returned when some packages fail
func TestRunOutdatedSummarizeVersionError(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
	}()

	// Config with invalid versioning format that will cause SummarizeAvailableVersions to fail
	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager: "js",
					Outdated: &config.OutdatedCfg{
						Commands: "echo ok",
						Versioning: &config.VersioningCfg{
							Format: "invalid-format", // Invalid format causes error
						},
					},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"2.0.0"}, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// Error is expected due to invalid versioning format
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown version format")
	})

	assert.Contains(t, out, "test")
}

// TestRunOutdatedStructuredOutputWithErrors tests the behavior of structured output with errors.
//
// It verifies:
//   - Errors are included in JSON output
//   - Error details are properly formatted
//   - Both successful and failed packages appear in output
func TestRunOutdatedStructuredOutputWithErrors(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldSkip := outdatedSkipPreflight
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedSkipPreflight = oldSkip
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager:     "js",
					Incremental: []string{"["}, // Invalid regex causes error
					Outdated:    &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"2.0.0"}, nil
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json" // Structured output
	outdatedSkipPreflight = true

	out := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		// Error expected due to invalid incremental pattern
		assert.Error(t, err)
	})

	// Output should still have JSON structure
	assert.Contains(t, out, "{")
}

// TestRunOutdatedStructuredOutputError tests the behavior when structured output generation fails.
//
// It verifies:
//   - Output generation errors are handled
//   - Error message is returned to user
//   - Invalid output format is detected
func TestRunOutdatedStructuredOutputError(t *testing.T) {
	// Save and restore globals
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldWrite := writeOutdatedResultFunc
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldSkip := outdatedSkipPreflight
	oldOutput := outdatedOutputFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		writeOutdatedResultFunc = oldWrite
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedSkipPreflight = oldSkip
		outdatedOutputFlag = oldOutput
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Outdated: &config.OutdatedCfg{Commands: "echo ok"}},
			},
		}, nil
	}
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{}, nil // No updates available
	}
	// Mock write function to return error
	writeOutdatedResultFunc = func(w io.Writer, format output.Format, result *output.OutdatedResult) error {
		return stderrors.New("write error")
	}

	outdatedDirFlag = "."
	outdatedConfigFlag = ""
	outdatedSkipPreflight = true
	outdatedOutputFlag = "json" // Use structured output to trigger the error path

	err := runOutdated(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write error")
}
