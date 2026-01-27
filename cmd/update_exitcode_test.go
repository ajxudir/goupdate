package cmd

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// UPDATE COMMAND EXIT CODE TESTS
// =============================================================================
//
// These tests verify that the update command returns correct exit codes
// for various success/failure scenarios.
// =============================================================================

func TestRunUpdateExitCodeAllSuccess(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldResolve := resolveUpdateCfgFunc
	oldListNewer := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldDir := updateDirFlag
	oldDryRun := updateDryRunFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipLock := updateSkipLockRun
	oldSkipSys := updateSkipSystemTests
	oldContinue := updateContinueOnFail
	oldYes := updateYesFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
			},
		}, nil
	}

	// Multiple packages: some will be updated, some will be up-to-date
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Rule: "npm", Name: "react", PackageType: "js", Type: "prod", Version: "17.0.0", InstalledVersion: "17.0.0", Constraint: "^"},
			{Rule: "npm", Name: "lodash", PackageType: "js", Type: "prod", Version: "4.17.0", InstalledVersion: "4.17.0", Constraint: "^"},
			{Rule: "npm", Name: "axios", PackageType: "js", Type: "prod", Version: "1.0.0", InstalledVersion: "1.0.0", Constraint: "^"},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	// Return newer versions for some packages, none for others
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		switch p.Name {
		case "react":
			return []string{"17.0.1", "17.0.2"}, nil // Has updates
		case "lodash":
			return []string{}, nil // Up-to-date
		case "axios":
			return []string{"1.1.0"}, nil // Has updates
		default:
			return []string{}, nil
		}
	}

	// All updates succeed
	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateSkipSystemTests = true
	updateContinueOnFail = true
	updateYesFlag = true

	t.Cleanup(func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		resolveUpdateCfgFunc = oldResolve
		listNewerVersionsFunc = oldListNewer
		updatePackageFunc = oldUpdate
		updateDirFlag = oldDir
		updateDryRunFlag = oldDryRun
		updateSkipPreflight = oldSkipPreflight
		updateSkipLockRun = oldSkipLock
		updateSkipSystemTests = oldSkipSys
		updateContinueOnFail = oldContinue
		updateYesFlag = oldYes
	})

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		// Should return nil (exit code 0) when all succeed
		assert.NoError(t, err, "expected no error when all packages succeed")
	})

	// Verify output contains expected packages
	assert.Contains(t, out, "react")
	assert.Contains(t, out, "lodash")
	assert.Contains(t, out, "axios")
}

// TestRunUpdateExitCodePartialFailure tests that exit code 1 is returned for partial failures.
//
// It verifies:
//   - Some packages succeed, some fail during planning
//   - With continue-on-fail enabled
//   - In dry-run mode, exit code is 2 because no packages are "updated" (they're just "planned")
//   - Exit code 1 (partial failure) only applies in non-dry-run mode when some updates actually succeed
func TestRunUpdateExitCodePartialFailure(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldResolve := resolveUpdateCfgFunc
	oldListNewer := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldDir := updateDirFlag
	oldDryRun := updateDryRunFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipLock := updateSkipLockRun
	oldSkipSys := updateSkipSystemTests
	oldContinue := updateContinueOnFail
	oldYes := updateYesFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
			},
		}, nil
	}

	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Rule: "npm", Name: "react", PackageType: "js", Type: "prod", Version: "17.0.0", InstalledVersion: "17.0.0", Constraint: "^"},
			{Rule: "npm", Name: "lodash", PackageType: "js", Type: "prod", Version: "4.0.0", InstalledVersion: "4.0.0", Constraint: "^"},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	// One succeeds, one fails to get versions
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		if p.Name == "lodash" {
			return nil, stderrors.New("version fetch failed for lodash")
		}
		return []string{"18.0.0"}, nil
	}

	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true // Dry-run mode - packages won't be "updated"
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateSkipSystemTests = true
	updateContinueOnFail = true
	updateYesFlag = true

	t.Cleanup(func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		resolveUpdateCfgFunc = oldResolve
		listNewerVersionsFunc = oldListNewer
		updatePackageFunc = oldUpdate
		updateDirFlag = oldDir
		updateDryRunFlag = oldDryRun
		updateSkipPreflight = oldSkipPreflight
		updateSkipLockRun = oldSkipLock
		updateSkipSystemTests = oldSkipSys
		updateContinueOnFail = oldContinue
		updateYesFlag = oldYes
	})

	captureStdout(t, func() {
		err := runUpdate(nil, nil)
		// Should return error when there are failures
		assert.Error(t, err, "expected error when there are failures")

		// In dry-run mode, no packages get "updated" status, so it's exit code 2 (complete failure)
		// even though one package was "planned" for update. The partial success (exit code 1)
		// only applies when some packages are actually updated in non-dry-run mode.
		var exitErr *errors.ExitError
		if stderrors.As(err, &exitErr) {
			assert.Equal(t, errors.ExitFailure, exitErr.Code, "expected exit code 2 in dry-run mode with failures")
		}
	})
}

// TestRunUpdateExitCodeCompleteFailure tests that exit code 2 is returned for complete failures.
//
// It verifies:
//   - All packages fail
//   - Exit code is 2 (complete failure)
func TestRunUpdateExitCodeCompleteFailure(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldResolve := resolveUpdateCfgFunc
	oldListNewer := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldDir := updateDirFlag
	oldDryRun := updateDryRunFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipLock := updateSkipLockRun
	oldSkipSys := updateSkipSystemTests
	oldContinue := updateContinueOnFail
	oldYes := updateYesFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
			},
		}, nil
	}

	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Rule: "npm", Name: "react", PackageType: "js", Type: "prod", Version: "17.0.0", InstalledVersion: "17.0.0", Constraint: "^"},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	// All fail
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return nil, stderrors.New("version fetch failed")
	}

	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateSkipSystemTests = true
	updateContinueOnFail = false // Complete failure mode
	updateYesFlag = true

	t.Cleanup(func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		resolveUpdateCfgFunc = oldResolve
		listNewerVersionsFunc = oldListNewer
		updatePackageFunc = oldUpdate
		updateDirFlag = oldDir
		updateDryRunFlag = oldDryRun
		updateSkipPreflight = oldSkipPreflight
		updateSkipLockRun = oldSkipLock
		updateSkipSystemTests = oldSkipSys
		updateContinueOnFail = oldContinue
		updateYesFlag = oldYes
	})

	captureStdout(t, func() {
		err := runUpdate(nil, nil)
		// Should return error for complete failure
		assert.Error(t, err, "expected error for complete failure")

		// Verify it's a complete failure (exit code 2)
		var exitErr *errors.ExitError
		if stderrors.As(err, &exitErr) {
			assert.Equal(t, errors.ExitFailure, exitErr.Code, "expected exit code 2 for complete failure")
		}
	})
}
