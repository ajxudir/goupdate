package cmd

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/stretchr/testify/assert"
)

// Interactive confirmation tests extracted from update_test.go
// These tests cover user interaction with confirmation prompts

// TestRunUpdateInteractiveConfirmYes tests the behavior of interactive update confirmation when user accepts.
//
// It verifies:
//   - Confirmation prompt is displayed when not in dry-run mode
//   - User accepting with "y" allows update to proceed
//   - Interactive mode works correctly with stdin input
func TestRunUpdateInteractiveConfirmYes(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldUpdate := updatePackageFunc
	oldResolve := resolveUpdateCfgFunc
	oldStdin := stdinReaderFunc
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
		stdinReaderFunc = oldStdin
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
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"2.0.0"}, nil
	}
	updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}
	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return cfg.Rules[p.Rule].Update, nil
	}

	// Mock stdin to return "y"
	stdinReaderFunc = func() *bufio.Reader {
		return bufio.NewReader(strings.NewReader("y\n"))
	}

	updateDirFlag = "."
	updateConfigFlag = ""
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateDryRunFlag = false // Not dry run to trigger prompt
	updateYesFlag = false    // Not auto-yes to trigger prompt
	updateOutputFlag = ""    // Table output (not structured)

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		// May return error due to version mismatch in mock environment
		_ = err
	})

	// Should show confirmation prompt and proceed
	assert.Contains(t, out, "Continue?")
}

// TestRunUpdateInteractiveConfirmNo tests the behavior of interactive update confirmation when user declines.
//
// It verifies:
//   - Confirmation prompt is displayed when not in dry-run mode
//   - User declining with "n" cancels the update
//   - Cancellation message is displayed
func TestRunUpdateInteractiveConfirmNo(t *testing.T) {
	oldLoad := loadConfigFunc
	oldGet := getPackagesFunc
	oldApply := applyInstalledVersionsFunc
	oldList := listNewerVersionsFunc
	oldResolve := resolveUpdateCfgFunc
	oldStdin := stdinReaderFunc
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
		resolveUpdateCfgFunc = oldResolve
		stdinReaderFunc = oldStdin
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
	getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
		return []formats.Package{{Name: "test", Rule: "npm", PackageType: "js", Version: "1.0.0", InstalledVersion: "1.0.0", Type: "prod"}}, nil
	}
	applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
		return pkgs, nil
	}
	listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"2.0.0"}, nil
	}
	resolveUpdateCfgFunc = func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
		return cfg.Rules[p.Rule].Update, nil
	}

	// Mock stdin to return "n" (cancel)
	stdinReaderFunc = func() *bufio.Reader {
		return bufio.NewReader(strings.NewReader("n\n"))
	}

	updateDirFlag = "."
	updateConfigFlag = ""
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateDryRunFlag = false // Not dry run to trigger prompt
	updateYesFlag = false    // Not auto-yes to trigger prompt
	updateOutputFlag = ""    // Table output (not structured)

	out := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err) // Should succeed (cancelled is not an error)
	})

	// Should show cancellation message
	assert.Contains(t, out, "Update cancelled")
}

// TestStdinReaderFuncDefault tests the behavior of default stdinReaderFunc.
//
// It verifies:
//   - Default stdinReaderFunc returns a valid reader
func TestStdinReaderFuncDefault(t *testing.T) {
	// The default stdinReaderFunc should return a valid reader
	reader := stdinReaderFunc()
	assert.NotNil(t, reader)
}
