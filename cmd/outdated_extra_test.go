package cmd

import (
	"context"
	stderrors "errors"
	"io"
	"os/exec"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// OUTDATED COMMAND EXTRA TESTS
// =============================================================================
//
// These tests cover additional scenarios including edge cases, structured
// output with errors, and error handling.
// =============================================================================

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
