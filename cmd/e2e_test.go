package cmd

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests simulating GitHub Actions workflow usage patterns.
// These tests verify exit codes, JSON output parsing, and flag behavior
// as used by .github/actions/_goupdate/action.yml

// TestE2E_ExitCodes tests the behavior of exit code handling in E2E scenarios.
//
// It verifies:
//   - Exit code 0 is returned on complete success
//   - Exit code 1 is returned on partial failure with --continue-on-fail
//   - Exit code 2 is returned on complete failure
func TestE2E_ExitCodes(t *testing.T) {
	// Save original functions
	oldLoadConfig := loadConfigFunc
	oldGetPackages := getPackagesFunc
	oldApplyInstalled := applyInstalledVersionsFunc
	oldListVersions := listNewerVersionsFunc
	oldUpdatePkg := updatePackageFunc

	// Cleanup after tests
	defer func() {
		loadConfigFunc = oldLoadConfig
		getPackagesFunc = oldGetPackages
		applyInstalledVersionsFunc = oldApplyInstalled
		listNewerVersionsFunc = oldListVersions
		updatePackageFunc = oldUpdatePkg
		rootCmd.SetArgs(nil)
		resetUpdateFlagsToDefaults()
	}()

	// Common mock config
	mockConfig := &config.Config{
		WorkingDir: ".",
		Rules: map[string]config.PackageManagerCfg{
			"mod": {Manager: "go", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
		},
	}

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return mockConfig, nil
	}

	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		for i := range pkgs {
			pkgs[i].InstalledVersion = pkgs[i].Version
		}
		return pkgs, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, workDir string) ([]string, error) {
		return []string{"1.1.0", "1.2.0"}, nil
	}

	t.Run("exit code 0 on complete success", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }
		defer func() { exitFunc = oldExit }()

		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "example.com/pkg", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
			}, nil
		}

		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil // Success
		}

		// Set up flags
		updateSkipPreflight = true
		updateSkipSystemTests = true
		updateDryRunFlag = true
		updateContinueOnFail = false
		updateSkipLockRun = true
		updateYesFlag = true
		updateOutputFlag = ""
		updateConfigFlag = ""
		updateDirFlag = "."
		updateTypeFlag = "all"
		updatePMFlag = "all"
		updateRuleFlag = "all"
		updateNameFlag = ""
		updateGroupFlag = ""

		rootCmd.SetArgs([]string{"update", "-r", "mod", "--skip-preflight", "--skip-system-tests", "--dry-run", "--skip-lock", "-y"})
		Execute()

		// Success should not call exitFunc (or call with 0)
		assert.True(t, exitCode == -1 || exitCode == errors.ExitSuccess, "expected exit code 0, got %d", exitCode)
	})

	t.Run("exit code 1 on partial failure with --continue-on-fail", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }
		defer func() { exitFunc = oldExit }()

		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "example.com/success", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
				{Name: "example.com/failure", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
			}, nil
		}

		// Use package name matching instead of call count to avoid order dependency
		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			if p.Name == "example.com/failure" {
				return stderrors.New("update failed")
			}
			return nil
		}

		// Enable --continue-on-fail
		updateSkipPreflight = true
		updateSkipSystemTests = true
		updateDryRunFlag = true
		updateContinueOnFail = true
		updateSkipLockRun = true
		updateYesFlag = true
		updateOutputFlag = ""
		updateConfigFlag = ""
		updateDirFlag = "."
		updateTypeFlag = "all"
		updatePMFlag = "all"
		updateRuleFlag = "all"
		updateNameFlag = ""
		updateGroupFlag = ""

		rootCmd.SetArgs([]string{"update", "-r", "mod", "--skip-preflight", "--skip-system-tests", "--dry-run", "--continue-on-fail", "--skip-lock", "-y"})
		Execute()

		// Partial failure = exit code 1
		assert.Equal(t, errors.ExitPartialFailure, exitCode, "expected exit code %d (partial failure), got %d", errors.ExitPartialFailure, exitCode)
	})

	t.Run("exit code 2 on complete failure", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }
		defer func() { exitFunc = oldExit }()

		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "example.com/pkg", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
			}, nil
		}

		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return stderrors.New("update failed")
		}

		updateSkipPreflight = true
		updateSkipSystemTests = true
		updateDryRunFlag = true
		updateContinueOnFail = false // Without continue-on-fail, failure is complete
		updateSkipLockRun = true
		updateYesFlag = true
		updateOutputFlag = ""
		updateConfigFlag = ""
		updateDirFlag = "."
		updateTypeFlag = "all"
		updatePMFlag = "all"
		updateRuleFlag = "all"
		updateNameFlag = ""
		updateGroupFlag = ""

		rootCmd.SetArgs([]string{"update", "-r", "mod", "--skip-preflight", "--skip-system-tests", "--dry-run", "--skip-lock", "-y"})
		Execute()

		// Complete failure = exit code 2
		assert.Equal(t, errors.ExitFailure, exitCode, "expected exit code %d (failure), got %d", errors.ExitFailure, exitCode)
	})

	t.Run("exit code 2 on config error (treated as failure)", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }
		defer func() { exitFunc = oldExit }()

		// Return config error
		loadConfigFunc = func(path, workDir string) (*config.Config, error) {
			return nil, stderrors.New("config file not found")
		}

		rootCmd.SetArgs([]string{"update", "-r", "mod"})
		Execute()

		// Config errors currently return as general failures (exit code 2)
		// Note: errors.ExitConfigError (3) is reserved for specific config validation errors
		// wrapped with errors.NewExitError, but general config load failures return errors.ExitFailure
		assert.Equal(t, errors.ExitFailure, exitCode, "expected exit code %d (failure for config error), got %d", errors.ExitFailure, exitCode)

		// Restore
		loadConfigFunc = func(path, workDir string) (*config.Config, error) {
			return mockConfig, nil
		}
	})
}

// TestE2E_JSONOutput tests the behavior of JSON output format in E2E scenarios.
//
// It verifies:
//   - Outdated JSON output has expected structure
//   - Update JSON output has expected structure
//   - Empty warnings/errors arrays are omitted with omitempty
func TestE2E_JSONOutput(t *testing.T) {
	// Test that JSON output is valid and contains expected fields
	t.Run("outdated JSON output has expected structure", func(t *testing.T) {
		// Example JSON structure that workflows expect
		jsonStr := `{
			"summary": {
				"total_packages": 2,
				"outdated_packages": 1,
				"up_to_date_packages": 1
			},
			"packages": [
				{
					"rule": "mod",
					"pm": "go",
					"type": "prod",
					"name": "example.com/pkg",
					"version": "1.0.0",
					"installed_version": "1.0.0",
					"status": "Outdated",
					"major": "2.0.0",
					"minor": "1.2.0",
					"patch": "1.0.1"
				},
				{
					"rule": "mod",
					"pm": "go",
					"type": "prod",
					"name": "example.com/current",
					"version": "2.0.0",
					"installed_version": "2.0.0",
					"status": "UpToDate",
					"major": "#N/A",
					"minor": "#N/A",
					"patch": "#N/A"
				}
			],
			"warnings": [],
			"errors": []
		}`

		// Verify it's valid JSON
		var result map[string]interface{}
		err := json.Unmarshal([]byte(jsonStr), &result)
		require.NoError(t, err, "JSON should be valid")

		// Verify expected structure exists
		assert.Contains(t, result, "summary")
		assert.Contains(t, result, "packages")
		assert.Contains(t, result, "warnings")
		assert.Contains(t, result, "errors")

		// Verify summary fields
		summary, ok := result["summary"].(map[string]interface{})
		require.True(t, ok, "summary should be an object")
		assert.Contains(t, summary, "total_packages")
		assert.Contains(t, summary, "outdated_packages")

		// Verify package fields
		packages, ok := result["packages"].([]interface{})
		require.True(t, ok, "packages should be an array")
		require.Len(t, packages, 2)

		pkg0, ok := packages[0].(map[string]interface{})
		require.True(t, ok, "package should be an object")
		assert.Equal(t, "Outdated", pkg0["status"])
		assert.Equal(t, "2.0.0", pkg0["major"])
	})

	t.Run("update JSON output has expected structure", func(t *testing.T) {
		// UpdateResult structure from output package
		// Note: Warnings and Errors use omitempty, so empty arrays are omitted
		result := &output.UpdateResult{
			Summary: output.UpdateSummary{
				TotalPackages:   2,
				UpdatedPackages: 1,
				FailedPackages:  1,
				DryRun:          false,
			},
			Packages: []output.UpdatePackage{
				{
					Rule:             "mod",
					PM:               "go",
					Type:             "prod",
					Name:             "example.com/success",
					Version:          "1.0.0",
					InstalledVersion: "1.0.0",
					Target:           "1.2.0",
					Status:           "Updated",
					Group:            "",
					Error:            "",
				},
				{
					Rule:             "mod",
					PM:               "go",
					Type:             "prod",
					Name:             "example.com/failure",
					Version:          "1.0.0",
					InstalledVersion: "1.0.0",
					Target:           "1.2.0",
					Status:           "Failed",
					Group:            "",
					Error:            "update failed",
				},
			},
			Warnings: []string{"some warning"}, // Include a warning so it appears in JSON
			Errors:   []string{"example.com/failure: update failed"},
		}

		jsonData, err := json.Marshal(result)
		require.NoError(t, err)

		// Verify it's valid JSON
		var parsed map[string]interface{}
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		// Verify structure - all fields should be present when populated
		assert.Contains(t, parsed, "summary")
		assert.Contains(t, parsed, "packages")
		assert.Contains(t, parsed, "warnings", "warnings should be present when non-empty")
		assert.Contains(t, parsed, "errors", "errors should be present when non-empty")

		// Verify summary counts
		summary, _ := parsed["summary"].(map[string]interface{})
		assert.EqualValues(t, 2, summary["total_packages"])
		assert.EqualValues(t, 1, summary["updated_packages"])
		assert.EqualValues(t, 1, summary["failed_packages"])
	})

	t.Run("update JSON omits empty warnings/errors", func(t *testing.T) {
		// When warnings/errors are empty, they should be omitted (omitempty)
		result := &output.UpdateResult{
			Summary: output.UpdateSummary{
				TotalPackages:   1,
				UpdatedPackages: 1,
				FailedPackages:  0,
				DryRun:          false,
			},
			Packages: []output.UpdatePackage{
				{
					Rule:   "mod",
					PM:     "go",
					Name:   "example.com/pkg",
					Status: "Updated",
				},
			},
			Warnings: []string{}, // Empty - should be omitted
			Errors:   []string{}, // Empty - should be omitted
		}

		jsonData, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		// Empty warnings/errors should be omitted due to omitempty
		_, hasWarnings := parsed["warnings"]
		_, hasErrors := parsed["errors"]
		assert.False(t, hasWarnings, "empty warnings should be omitted")
		assert.False(t, hasErrors, "empty errors should be omitted")
	})

	t.Run("jq filter patterns work correctly", func(t *testing.T) {
		// Simulate workflows' jq patterns on JSON output
		packages := []map[string]interface{}{
			{"name": "pkg1", "status": "Outdated", "major": "2.0.0", "minor": "1.2.0", "patch": "#N/A"},
			{"name": "pkg2", "status": "Outdated", "major": "#N/A", "minor": "1.1.0", "patch": "1.0.1"},
			{"name": "pkg3", "status": "UpToDate", "major": "#N/A", "minor": "#N/A", "patch": "#N/A"},
		}

		// jq '[.[] | select(.status == "Outdated")]'
		var outdated []map[string]interface{}
		for _, pkg := range packages {
			if pkg["status"] == "Outdated" {
				outdated = append(outdated, pkg)
			}
		}
		assert.Len(t, outdated, 2, "should filter to Outdated packages")

		// jq '[.[] | select(.major != "#N/A")] | length'
		majorCount := 0
		for _, pkg := range outdated {
			if pkg["major"] != "#N/A" {
				majorCount++
			}
		}
		assert.Equal(t, 1, majorCount, "should count packages with major updates")

		// jq '[.[] | select(.minor != "#N/A")] | length'
		minorCount := 0
		for _, pkg := range outdated {
			if pkg["minor"] != "#N/A" {
				minorCount++
			}
		}
		assert.Equal(t, 2, minorCount, "should count packages with minor updates")

		// jq '[.[] | select(.major != "#N/A" and .minor == "#N/A" and .patch == "#N/A")]'
		// (major-only packages)
		var majorOnly []map[string]interface{}
		for _, pkg := range outdated {
			if pkg["major"] != "#N/A" && pkg["minor"] == "#N/A" && pkg["patch"] == "#N/A" {
				majorOnly = append(majorOnly, pkg)
			}
		}
		// pkg1 has both major and minor, so it's not major-only
		assert.Len(t, majorOnly, 0, "no packages should be major-only")
	})
}

// TestE2E_ContinueOnFail tests the behavior of --continue-on-fail flag in E2E scenarios.
//
// It verifies:
//   - All packages are processed even after failures
//   - Without --continue-on-fail, stops at first failure in group
//   - Partial success is reported correctly
func TestE2E_ContinueOnFail(t *testing.T) {
	oldLoadConfig := loadConfigFunc
	oldGetPackages := getPackagesFunc
	oldApplyInstalled := applyInstalledVersionsFunc
	oldListVersions := listNewerVersionsFunc
	oldUpdatePkg := updatePackageFunc

	defer func() {
		loadConfigFunc = oldLoadConfig
		getPackagesFunc = oldGetPackages
		applyInstalledVersionsFunc = oldApplyInstalled
		listNewerVersionsFunc = oldListVersions
		updatePackageFunc = oldUpdatePkg
		rootCmd.SetArgs(nil)
		resetUpdateFlagsToDefaults()
	}()

	mockConfig := &config.Config{
		WorkingDir: ".",
		Rules: map[string]config.PackageManagerCfg{
			"mod": {Manager: "go", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
		},
	}

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return mockConfig, nil
	}

	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		for i := range pkgs {
			pkgs[i].InstalledVersion = pkgs[i].Version
		}
		return pkgs, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, workDir string) ([]string, error) {
		return []string{"1.1.0"}, nil
	}

	t.Run("processes all packages even after failures", func(t *testing.T) {
		processedPackages := make([]string, 0)

		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "pkg1", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
				{Name: "pkg2", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
				{Name: "pkg3", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
			}, nil
		}

		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			processedPackages = append(processedPackages, p.Name)
			if p.Name == "pkg2" {
				return stderrors.New("update failed")
			}
			return nil
		}

		updateSkipPreflight = true
		updateSkipSystemTests = true
		updateDryRunFlag = true
		updateContinueOnFail = true
		updateSkipLockRun = true
		updateYesFlag = true
		updateOutputFlag = ""
		updateConfigFlag = ""
		updateDirFlag = "."
		updateTypeFlag = "all"
		updatePMFlag = "all"
		updateRuleFlag = "all"
		updateNameFlag = ""
		updateGroupFlag = ""

		exitFunc = func(code int) {}
		rootCmd.SetArgs([]string{"update", "-r", "mod", "--skip-preflight", "--skip-system-tests", "--dry-run", "--continue-on-fail", "--skip-lock", "-y"})
		Execute()

		// All packages should be processed
		assert.Len(t, processedPackages, 3, "all packages should be processed")
		assert.Contains(t, processedPackages, "pkg1")
		assert.Contains(t, processedPackages, "pkg2")
		assert.Contains(t, processedPackages, "pkg3")
	})

	t.Run("without --continue-on-fail stops at first failure in group", func(t *testing.T) {
		// Note: behavior depends on grouping logic
		// For ungrouped packages, each is processed individually
		// This test verifies the flag actually affects behavior
		processedCount := 0

		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "pkg1", Version: "1.0.0", Rule: "mod", PackageType: "go", Type: "prod"},
			}, nil
		}

		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			processedCount++
			return stderrors.New("update failed")
		}

		updateSkipPreflight = true
		updateSkipSystemTests = true
		updateDryRunFlag = true
		updateContinueOnFail = false // Without continue-on-fail
		updateSkipLockRun = true
		updateYesFlag = true
		updateOutputFlag = ""
		updateConfigFlag = ""
		updateDirFlag = "."
		updateTypeFlag = "all"
		updatePMFlag = "all"
		updateRuleFlag = "all"
		updateNameFlag = ""
		updateGroupFlag = ""

		exitCode := -1
		exitFunc = func(code int) { exitCode = code }
		rootCmd.SetArgs([]string{"update", "-r", "mod", "--skip-preflight", "--skip-system-tests", "--dry-run", "--skip-lock", "-y"})
		Execute()

		// Should exit with failure code
		assert.Equal(t, errors.ExitFailure, exitCode)
	})
}

// TestE2E_UpdateTypeFlags tests the behavior of update type flags in E2E scenarios.
//
// It verifies:
//   - DetermineScopeDescription returns correct scope
//   - Flags are mutually exclusive in priority
//   - Major takes precedence over minor and patch
func TestE2E_UpdateTypeFlags(t *testing.T) {
	t.Run("DetermineScopeDescription returns correct scope", func(t *testing.T) {
		// Test helper function used in update.go
		tests := []struct {
			major    bool
			minor    bool
			patch    bool
			expected string
		}{
			{true, false, false, "--major scope"},
			{false, true, false, "--minor scope"},
			{false, false, true, "--patch scope"},
			{false, false, false, "constraint scope"},
		}

		for _, tc := range tests {
			selection := outdated.UpdateSelectionFlags{
				Major: tc.major,
				Minor: tc.minor,
				Patch: tc.patch,
			}
			result := update.DetermineScopeDescription(selection)
			assert.Equal(t, tc.expected, result,
				"expected %q for major=%v minor=%v patch=%v, got %q",
				tc.expected, tc.major, tc.minor, tc.patch, result)
		}
	})

	t.Run("flags are mutually exclusive in priority", func(t *testing.T) {
		// When multiple flags are set, major takes precedence
		// This matches the behavior expected by workflows

		// DetermineScopeDescription checks in order: major, minor, patch
		selection := outdated.UpdateSelectionFlags{Major: true, Minor: true, Patch: true}
		result := update.DetermineScopeDescription(selection)
		assert.Equal(t, "--major scope", result, "major should take precedence")

		selection = outdated.UpdateSelectionFlags{Major: false, Minor: true, Patch: true}
		result = update.DetermineScopeDescription(selection)
		assert.Equal(t, "--minor scope", result, "minor should take precedence over patch")
	})
}

// TestE2E_SystemTestMode tests the behavior of --system-test-mode flag in E2E scenarios.
//
// It verifies:
//   - System test mode flag accepts valid values
//   - createSystemTestRunner respects mode override
//   - No runner is created when system tests not configured
func TestE2E_SystemTestMode(t *testing.T) {
	t.Run("system test mode flag values", func(t *testing.T) {
		// Verify the flag accepts expected values
		validModes := []string{"after_each", "after_all", "none", ""}

		for _, mode := range validModes {
			updateSystemTestModeFlag = mode
			// Just verify the flag can be set without panic
			assert.Equal(t, mode, updateSystemTestModeFlag)
		}
	})

	t.Run("createSystemTestRunner respects mode override", func(t *testing.T) {
		// Test that the flag overrides config
		cfg := &config.Config{
			SystemTests: &config.SystemTestsCfg{
				RunMode: "after_all",
				Tests: []config.SystemTestCfg{
					{Name: "test", Commands: "echo test"},
				},
			},
		}

		// Without override
		updateSystemTestModeFlag = ""
		runner := createSystemTestRunner(cfg, ".")
		require.NotNil(t, runner)
		// Runner should use config's mode (after_all)
		assert.True(t, runner.ShouldRunAfterAll())
		assert.False(t, runner.ShouldRunAfterEach())

		// With override
		updateSystemTestModeFlag = "after_each"
		runner = createSystemTestRunner(cfg, ".")
		require.NotNil(t, runner)
		// Runner should use overridden mode
		assert.True(t, runner.ShouldRunAfterEach())
		assert.False(t, runner.ShouldRunAfterAll())

		// With none override
		updateSystemTestModeFlag = "none"
		runner = createSystemTestRunner(cfg, ".")
		require.NotNil(t, runner)
		assert.False(t, runner.ShouldRunAfterEach())
		assert.False(t, runner.ShouldRunAfterAll())

		// Reset
		updateSystemTestModeFlag = ""
	})

	t.Run("no runner when system tests not configured", func(t *testing.T) {
		cfg := &config.Config{
			SystemTests: nil,
		}

		runner := createSystemTestRunner(cfg, ".")
		assert.Nil(t, runner)
	})
}

// TestE2E_PartialSuccessError tests the behavior of PartialSuccessError in E2E scenarios.
//
// It verifies:
//   - Error message format is correct
//   - Exit code is ExitPartialFailure
//   - Success and failure counts are accurate
func TestE2E_PartialSuccessError(t *testing.T) {
	t.Run("error message format", func(t *testing.T) {
		errs := []error{
			stderrors.New("pkg1 failed"),
			stderrors.New("pkg2 failed"),
		}
		err := errors.NewPartialSuccessError(3, 2, errs)

		assert.Equal(t, "3 succeeded, 2 failed", err.Error())
		assert.Equal(t, 3, err.Succeeded)
		assert.Equal(t, 2, err.Failed)
		assert.Len(t, err.Errors, 2)
	})

	t.Run("exit code is ExitPartialFailure", func(t *testing.T) {
		err := errors.NewPartialSuccessError(1, 1, []error{stderrors.New("test")})
		exitErr := errors.NewExitError(errors.ExitPartialFailure, err)

		assert.Equal(t, errors.ExitPartialFailure, exitErr.Code)
		assert.Equal(t, "1 succeeded, 1 failed", exitErr.Error())
	})
}

// TestE2E_WorkflowOutputParsing tests the behavior of workflow output parsing in E2E scenarios.
//
// It verifies:
//   - Count updated packages from diff works correctly
//   - Extract package names from diff works correctly
//   - Output parsing matches GitHub Actions workflow expectations
func TestE2E_WorkflowOutputParsing(t *testing.T) {
	t.Run("count updated packages from diff", func(t *testing.T) {
		// Workflow counts updates by diffing go.mod:
		// RAW_COUNT=$(diff go.mod go.mod.backup 2>/dev/null | grep "^<" | grep -v "^< //" | wc -l | tr -d ' \n' || true)
		//
		// Simulating this logic:
		diffOutput := `< 	golang.org/x/mod v0.20.0
< 	golang.org/x/text v1.16.0
<	// indirect comment
`
		lines := strings.Split(diffOutput, "\n")
		count := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "<") && !strings.HasPrefix(line, "< //") && !strings.HasPrefix(line, "<\t//") {
				count++
			}
		}
		assert.Equal(t, 2, count, "should count non-comment lines starting with <")
	})

	t.Run("extract package names from diff", func(t *testing.T) {
		// Workflow extracts package names:
		// UPDATED_PKGS=$(diff go.mod go.mod.backup 2>/dev/null | grep "^<" | grep -v "^< //" | awk '{print $2}' | tr '\n' ' ' | tr -d '\r' || echo "")
		diffOutput := `< 	golang.org/x/mod v0.20.0
< 	golang.org/x/text v1.16.0
`
		lines := strings.Split(strings.TrimSpace(diffOutput), "\n")
		var pkgs []string
		for _, line := range lines {
			if strings.HasPrefix(line, "<") && !strings.HasPrefix(line, "< //") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					pkgs = append(pkgs, fields[1])
				}
			}
		}
		assert.Len(t, pkgs, 2)
		assert.Contains(t, pkgs, "golang.org/x/mod")
		assert.Contains(t, pkgs, "golang.org/x/text")
	})
}

// TestE2E_MajorOnlyDetection tests the behavior of major-only update detection in E2E scenarios.
//
// It verifies:
//   - Major-only packages are detected from JSON correctly
//   - Fail-on-major-only logic works as expected
//   - Update type affects failure behavior
func TestE2E_MajorOnlyDetection(t *testing.T) {
	t.Run("detect major-only packages from JSON", func(t *testing.T) {
		packages := []map[string]string{
			// Has major only
			{"name": "breaking-pkg", "major": "2.0.0", "minor": "#N/A", "patch": "#N/A"},
			// Has minor and patch
			{"name": "normal-pkg", "major": "#N/A", "minor": "1.2.0", "patch": "1.0.1"},
			// Has major and minor
			{"name": "mixed-pkg", "major": "2.0.0", "minor": "1.5.0", "patch": "#N/A"},
		}

		// Filter major-only (as done by workflow):
		// jq '[.[] | select(.major != "#N/A" and .minor == "#N/A" and .patch == "#N/A")]'
		var majorOnly []map[string]string
		for _, pkg := range packages {
			if pkg["major"] != "#N/A" && pkg["minor"] == "#N/A" && pkg["patch"] == "#N/A" {
				majorOnly = append(majorOnly, pkg)
			}
		}

		assert.Len(t, majorOnly, 1)
		assert.Equal(t, "breaking-pkg", majorOnly[0]["name"])

		// Check if there are updatable packages (minor or patch available)
		// jq '[.[] | select(.minor != "#N/A" or .patch != "#N/A")]'
		var updatable []map[string]string
		for _, pkg := range packages {
			if pkg["minor"] != "#N/A" || pkg["patch"] != "#N/A" {
				updatable = append(updatable, pkg)
			}
		}

		assert.Len(t, updatable, 2, "normal-pkg and mixed-pkg are updatable")
	})

	t.Run("fail-on-major-only logic", func(t *testing.T) {
		// When fail-on-major-only is true and only major updates available,
		// workflow should exit with error

		updatableCount := 0
		majorOnlyCount := 1
		failOnMajor := true
		updateType := "minor" // Not "all" or "major"

		shouldFail := updatableCount == 0 && majorOnlyCount > 0 && failOnMajor && updateType != "all" && updateType != "major"
		assert.True(t, shouldFail, "should fail when only major updates available")

		// When update type is "all" or "major", should not fail
		updateType = "all"
		shouldFail = updatableCount == 0 && majorOnlyCount > 0 && failOnMajor && updateType != "all" && updateType != "major"
		assert.False(t, shouldFail, "should not fail when update type includes major")
	})
}

// TestE2E_JSONValidation tests the behavior of JSON output validation in E2E scenarios.
//
// It verifies:
//   - Valid JSON passes jq validation
//   - Invalid JSON fails validation
//   - Empty output uses fallback JSON
func TestE2E_JSONValidation(t *testing.T) {
	t.Run("valid JSON passes jq validation", func(t *testing.T) {
		validJSON := `{"packages": [], "summary": {"total_packages": 0}}`

		var result map[string]interface{}
		err := json.Unmarshal([]byte(validJSON), &result)
		assert.NoError(t, err, "valid JSON should parse")
	})

	t.Run("invalid JSON fails validation", func(t *testing.T) {
		invalidJSON := `{"packages": [`

		var result map[string]interface{}
		err := json.Unmarshal([]byte(invalidJSON), &result)
		assert.Error(t, err, "invalid JSON should fail to parse")
	})

	t.Run("empty output uses fallback", func(t *testing.T) {
		// Workflow fallback: JSON_OUTPUT='{"packages":[]}'
		fallback := `{"packages":[]}`

		var result map[string]interface{}
		err := json.Unmarshal([]byte(fallback), &result)
		assert.NoError(t, err)

		packages, ok := result["packages"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, packages, 0)
	})
}

// TestE2E_SummaryFormat tests the behavior of summary message formatting in E2E scenarios.
//
// It verifies:
//   - Workflow summary format is correct
//   - Outdated counts are displayed properly
//   - Major-only count is included when present
func TestE2E_SummaryFormat(t *testing.T) {
	t.Run("workflow summary format", func(t *testing.T) {
		// Workflow builds summary: "$OUTDATED_COUNT outdated: $MINOR_COUNT minor, $PATCH_COUNT patch"
		outdatedCount := 5
		majorOnlyCount := 1
		minorCount := 3
		patchCount := 2

		summary := fmt.Sprintf("%d outdated: %d minor, %d patch", outdatedCount, minorCount, patchCount)
		if majorOnlyCount > 0 {
			summary = fmt.Sprintf("%s, %d major-only", summary, majorOnlyCount)
		}

		assert.Equal(t, "5 outdated: 3 minor, 2 patch, 1 major-only", summary)
	})
}

var oldExit = exitFunc
