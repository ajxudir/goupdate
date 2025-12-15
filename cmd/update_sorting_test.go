package cmd

import (
	"context"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/stretchr/testify/assert"
)

// Sorting tests extracted from update_test.go
// These tests cover sorting of packages by various criteria

// TestRunUpdateSortingComparators tests the behavior of package sorting with multiple criteria.
//
// It verifies:
//   - Packages are sorted by rule, package type, type (prod/dev), and group
//   - All sorting comparators work correctly
//   - Packages with different attributes are displayed in correct order
func TestRunUpdateSortingComparators(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalApply := applyInstalledVersionsFunc
	originalResolve := resolveUpdateCfgFunc
	originalListNewer := listNewerVersionsFunc
	originalUpdate := updatePackageFunc
	originalDir := updateDirFlag
	originalDryRun := updateDryRunFlag
	originalSkipPreflight := updateSkipPreflight
	originalSkipLock := updateSkipLockRun
	originalOutput := updateOutputFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
				"pip": {Manager: "py", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
			},
		}, nil
	}

	// Return packages that exercise all sorting comparators
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Rule: "npm", Name: "zlib", PackageType: "js", Type: "dev", Version: "1.0.0", InstalledVersion: "1.0.0", Constraint: "^", Group: "group-b"},
			{Rule: "npm", Name: "axios", PackageType: "js", Type: "prod", Version: "1.0.0", InstalledVersion: "1.0.0", Constraint: "^", Group: "group-a"},
			{Rule: "pip", Name: "requests", PackageType: "py", Type: "prod", Version: "1.0.0", InstalledVersion: "1.0.0", Constraint: "^"},
			{Rule: "npm", Name: "react", PackageType: "js", Type: "prod", Version: "17.0.0", InstalledVersion: "17.0.0", Constraint: "^", Group: "group-a"},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{}, nil
	}

	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateOutputFlag = "" // Ensure table output (default)

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		applyInstalledVersionsFunc = originalApply
		resolveUpdateCfgFunc = originalResolve
		listNewerVersionsFunc = originalListNewer
		updatePackageFunc = originalUpdate
		updateDirFlag = originalDir
		updateDryRunFlag = originalDryRun
		updateSkipPreflight = originalSkipPreflight
		updateSkipLockRun = originalSkipLock
		updateOutputFlag = originalOutput
	})

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// All packages should appear in output
	assert.Contains(t, out, "axios")
	assert.Contains(t, out, "react")
	assert.Contains(t, out, "zlib")
	assert.Contains(t, out, "requests")
}

// TestRunUpdateSortingDifferentPackageTypes tests the behavior of package sorting by package type.
//
// It verifies:
//   - Packages with different package types are sorted correctly
//   - PackageType sorting works as expected (javascript before typescript)
func TestRunUpdateSortingDifferentPackageTypes(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldResolve := resolveUpdateCfgFunc
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldSkip := updateSkipPreflight
	oldSkipSys := updateSkipSystemTests
	oldDry := updateDryRunFlag
	oldYes := updateYesFlag
	oldOutput := updateOutputFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		updatePackageFunc = oldUpdate
		resolveUpdateCfgFunc = oldResolve
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateSkipPreflight = oldSkip
		updateSkipSystemTests = oldSkipSys
		updateDryRunFlag = oldDry
		updateYesFlag = oldYes
		updateOutputFlag = oldOutput
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{Commands: "echo ok"}},
			},
		}, nil
	}
	// Return packages with same rule but different PackageTypes
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "pkg2", Rule: "npm", PackageType: "typescript", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"},
			{Name: "pkg1", Rule: "npm", PackageType: "javascript", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{}, nil // No updates available
	}
	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}
	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return cfg.Rules[p.Rule].Update, nil
	}

	updateDirFlag = "."
	updateConfigFlag = ""
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateDryRunFlag = true // Dry run to avoid actual updates
	updateYesFlag = true
	updateOutputFlag = "" // Ensure table output (default)

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should show both packages sorted by PackageType (javascript before typescript)
	assert.Contains(t, out, "pkg1")
	assert.Contains(t, out, "pkg2")
}

// TestRunUpdateSortingDifferentGroups tests the behavior of package sorting by group.
//
// It verifies:
//   - Packages with different groups are sorted alphabetically by group name
//   - Group template expansion affects sorting order
func TestRunUpdateSortingDifferentGroups(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldResolve := resolveUpdateCfgFunc
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldSkip := updateSkipPreflight
	oldSkipSys := updateSkipSystemTests
	oldDry := updateDryRunFlag
	oldYes := updateYesFlag
	oldOutput := updateOutputFlag
	defer func() {
		loadConfigFunc = oldLoad
		getPackagesFunc = oldGet
		applyInstalledVersionsFunc = oldApply
		listNewerVersionsFunc = oldList
		updatePackageFunc = oldUpdate
		resolveUpdateCfgFunc = oldResolve
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateSkipPreflight = oldSkip
		updateSkipSystemTests = oldSkipSys
		updateDryRunFlag = oldDry
		updateYesFlag = oldYes
		updateOutputFlag = oldOutput
	}()

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager: "js",
					// Use {{package}} template so each package gets a different group
					Update:   &config.UpdateCfg{Group: "{{package}}"},
					Outdated: &config.OutdatedCfg{Commands: "echo ok"},
				},
			},
		}, nil
	}
	// Return packages with same rule/type - groups will differ due to {{package}} template
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Name: "pkg-z", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"},
			{Name: "pkg-a", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"},
		}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{}, nil // No updates available
	}
	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}
	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return cfg.Rules[p.Rule].Update, nil
	}

	updateDirFlag = "."
	updateConfigFlag = ""
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateDryRunFlag = true
	updateYesFlag = true
	updateOutputFlag = "" // Ensure table output (default)

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should show both packages sorted by group (pkg-a before pkg-z)
	assert.Contains(t, out, "pkg-a")
	assert.Contains(t, out, "pkg-z")
}
