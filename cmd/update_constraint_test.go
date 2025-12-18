package cmd

import (
	"context"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/stretchr/testify/assert"
)

// Constraint tests extracted from update_test.go
// These tests cover floating constraint validation and exact constraint handling

// TestFloatingConstraintInGroupShowsFloating tests the behavior of floating constraints in groups.
//
// It verifies:
//   - Floating constraints in grouped packages are marked as "Floating" status
//   - Packages with wildcard versions like "8.*" are identified as floating
//   - Floating constraints are shown as unsupported in dry-run mode
func TestFloatingConstraintInGroupShowsFloating(t *testing.T) {
	// Test that floating constraints in groups are marked as Floating (unsupported)
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalListNewer := listNewerVersionsFunc
	originalResolve := resolveUpdateCfgFunc
	originalType := updateTypeFlag
	originalPM := updatePMFlag
	originalDir := updateDirFlag
	originalConfig := updateConfigFlag
	originalDryRun := updateDryRunFlag
	originalSkipLock := updateSkipLockRun
	originalSkipPreflight := updateSkipPreflight
	originalOutput := updateOutputFlag
	originalRule := updateRuleFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"nuget": {
					Manager: "dotnet",
					Update: &config.UpdateCfg{
						Commands: "dotnet restore",
						Group:    "dotnet-deps",
					},
				},
			},
		}, nil
	}

	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{
				Rule:             "nuget",
				Name:             "Newtonsoft.Json",
				PackageType:      "dotnet",
				Type:             "prod",
				Version:          "8.*", // Floating constraint
				InstalledVersion: "8.0.4",
				Constraint:       "",
			},
		}, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"8.0.5", "9.0.0"}, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		ruleCfg := cfg.Rules[p.Rule]
		return ruleCfg.Update, nil
	}

	updateTypeFlag, updatePMFlag, updateDirFlag, updateConfigFlag = "all", "all", ".", ""
	updateDryRunFlag = true
	updateSkipLockRun = true
	updateSkipPreflight = true
	updateOutputFlag = "" // Ensure table output (default)
	updateRuleFlag = "all"

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		listNewerVersionsFunc = originalListNewer
		resolveUpdateCfgFunc = originalResolve
		updateTypeFlag = originalType
		updatePMFlag = originalPM
		updateDirFlag = originalDir
		updateConfigFlag = originalConfig
		updateDryRunFlag = originalDryRun
		updateSkipLockRun = originalSkipLock
		updateSkipPreflight = originalSkipPreflight
		updateOutputFlag = originalOutput
		updateRuleFlag = originalRule
	})

	output := captureStdout(t, func() {
		err := runUpdate(updateCmd, nil)
		// No error - floating constraints are just marked as unsupported
		assert.NoError(t, err)
	})

	// Should show as Floating (unsupported)
	assert.Contains(t, output, "Floating")
	assert.Contains(t, output, "Newtonsoft.Json")
}

// TestFloatingConstraintShowsUnsupported tests the behavior of floating constraint display.
//
// It verifies:
//   - Floating constraints are marked as unsupported
//   - Packages with wildcard versions are not automatically updateable
//   - Update command displays floating constraint status correctly
func TestFloatingConstraintShowsUnsupported(t *testing.T) {
	// Test that floating constraints are marked as unsupported (not updateable automatically)
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalListNewer := listNewerVersionsFunc
	originalResolve := resolveUpdateCfgFunc
	originalType := updateTypeFlag
	originalPM := updatePMFlag
	originalDir := updateDirFlag
	originalConfig := updateConfigFlag
	originalDryRun := updateDryRunFlag
	originalSkipLock := updateSkipLockRun
	originalSkipPreflight := updateSkipPreflight
	originalOutput := updateOutputFlag
	originalRule := updateRuleFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"nuget": {
					Manager: "dotnet",
					Update: &config.UpdateCfg{
						Commands: "dotnet restore",
					},
				},
			},
		}, nil
	}

	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{
				Rule:             "nuget",
				Name:             "Newtonsoft.Json",
				PackageType:      "dotnet",
				Type:             "prod",
				Version:          "8.*", // Floating constraint
				InstalledVersion: "8.0.4",
				Constraint:       "",
			},
		}, nil
	}

	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"8.0.5", "9.0.0"}, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		ruleCfg := cfg.Rules[p.Rule]
		return ruleCfg.Update, nil
	}

	updateTypeFlag, updatePMFlag, updateDirFlag, updateConfigFlag = "all", "all", ".", ""
	updateDryRunFlag = true
	updateSkipLockRun = true
	updateSkipPreflight = true
	updateOutputFlag = "" // Ensure table output (default)
	updateRuleFlag = "all"

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		listNewerVersionsFunc = originalListNewer
		resolveUpdateCfgFunc = originalResolve
		updateTypeFlag = originalType
		updatePMFlag = originalPM
		updateDirFlag = originalDir
		updateConfigFlag = originalConfig
		updateDryRunFlag = originalDryRun
		updateSkipLockRun = originalSkipLock
		updateSkipPreflight = originalSkipPreflight
		updateOutputFlag = originalOutput
		updateRuleFlag = originalRule
	})

	output := captureStdout(t, func() {
		_ = runUpdate(updateCmd, nil)
	})

	// Should show as Floating (unsupported status)
	assert.Contains(t, output, "Floating")
	// The package should be listed
	assert.Contains(t, output, "Newtonsoft.Json")
	assert.Contains(t, output, "8.*")
}

// TestRunUpdateFloatingConstraint tests the behavior of runUpdate with floating constraints.
//
// It verifies:
//   - Packages with floating constraints like "17.*" are identified
//   - Floating constraints are marked as such in update output
//   - Update process handles floating constraints without errors
func TestRunUpdateFloatingConstraint(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalApply := applyInstalledVersionsFunc
	originalResolve := resolveUpdateCfgFunc
	originalUpdate := updatePackageFunc
	originalDir := updateDirFlag
	originalDryRun := updateDryRunFlag
	originalSkipPreflight := updateSkipPreflight
	originalSkipLock := updateSkipLockRun
	originalOutput := updateOutputFlag
	originalRule := updateRuleFlag

	loadConfigFunc = func(path, workDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: ".",
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
			},
		}, nil
	}

	// Use a wildcard version like "17.*" which is a true floating constraint
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{
			{Rule: "npm", Name: "react", PackageType: "js", Version: "17.*", InstalledVersion: "17.0.0"},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateOutputFlag = "" // Ensure table output (default)
	updateRuleFlag = "all"

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		applyInstalledVersionsFunc = originalApply
		resolveUpdateCfgFunc = originalResolve
		updatePackageFunc = originalUpdate
		updateDirFlag = originalDir
		updateDryRunFlag = originalDryRun
		updateSkipPreflight = originalSkipPreflight
		updateSkipLockRun = originalSkipLock
		updateOutputFlag = originalOutput
		updateRuleFlag = originalRule
	})

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "Floating")
}

// TestRunUpdateExactConstraint tests the behavior of runUpdate with exact constraints.
//
// It verifies:
//   - Packages with exact constraints (=) are identified correctly
//   - Exact constraint packages show "Exact (=)" status in output
//   - Update process handles exact constraints properly
func TestRunUpdateExactConstraint(t *testing.T) {
	originalLoad := loadConfigFunc
	originalGet := getPackagesFunc
	originalApply := applyInstalledVersionsFunc
	originalResolve := resolveUpdateCfgFunc
	originalUpdate := updatePackageFunc
	originalDir := updateDirFlag
	originalDryRun := updateDryRunFlag
	originalSkipPreflight := updateSkipPreflight
	originalSkipLock := updateSkipLockRun
	originalOutput := updateOutputFlag
	originalRule := updateRuleFlag

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
			{Rule: "npm", Name: "react", PackageType: "js", Version: "17.0.0", InstalledVersion: "17.0.0", Constraint: "="},
		}, nil
	}

	applyInstalledVersionsFunc = func(packages []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return packages, nil
	}

	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return &config.UpdateCfg{}, nil
	}

	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	updateDirFlag = "."
	updateDryRunFlag = true
	updateSkipPreflight = true
	updateSkipLockRun = true
	updateOutputFlag = "" // Ensure table output (default)
	updateRuleFlag = "all"

	t.Cleanup(func() {
		loadConfigFunc = originalLoad
		getPackagesFunc = originalGet
		applyInstalledVersionsFunc = originalApply
		resolveUpdateCfgFunc = originalResolve
		updatePackageFunc = originalUpdate
		updateDirFlag = originalDir
		updateDryRunFlag = originalDryRun
		updateSkipPreflight = originalSkipPreflight
		updateSkipLockRun = originalSkipLock
		updateOutputFlag = originalOutput
		updateRuleFlag = originalRule
	})

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// With exact constraint (=), the package shows as "Planned" in dry-run mode
	assert.Contains(t, out, "Exact (=)")
	assert.Contains(t, out, "react")
}
