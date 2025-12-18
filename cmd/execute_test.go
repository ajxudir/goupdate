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

// TestExecuteWithExitCodes tests the behavior of Execute with different exit codes.
//
// It verifies:
//   - Successful commands return exit code 0
//   - Errors call exitFunc with appropriate exit codes
//   - Partial success errors return ExitPartialFailure code
func TestExecuteWithExitCodes(t *testing.T) {
	oldExit := exitFunc
	defer func() { exitFunc = oldExit }()

	t.Run("success returns 0", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }

		rootCmd.SetArgs([]string{"--help"})
		Execute()

		// --help doesn't error, so exitFunc shouldn't be called
		assert.Equal(t, -1, exitCode)
		rootCmd.SetArgs(nil)
	})

	t.Run("error calls exitFunc with exit code", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }

		// Run with invalid command to trigger an error
		rootCmd.SetArgs([]string{"nonexistent-subcommand-xyz"})
		Execute()

		// Should call exitFunc with failure code
		assert.Equal(t, errors.ExitFailure, exitCode)
		rootCmd.SetArgs(nil)
	})

	t.Run("partial success error uses ExitPartialFailure", func(t *testing.T) {
		exitCode := -1
		exitFunc = func(code int) { exitCode = code }

		// We need to trigger a partial success error through an actual command
		// Use update command with a config that will cause partial success
		oldLoadConfig := loadConfigFunc
		oldGetPackages := getPackagesFunc
		oldApplyInstalled := applyInstalledVersionsFunc
		oldListVersions := listNewerVersionsFunc
		oldUpdatePkg := updatePackageFunc
		oldUpdateFlags := struct {
			skipPreflight   bool
			skipSystemTests bool
			dryRun          bool
			continueOnFail  bool
			skipLock        bool
			yes             bool
			typeFlag        string
			pmFlag          string
			ruleFlag        string
			nameFlag        string
			groupFlag       string
			configFlag      string
			dirFlag         string
			outputFlag      string
		}{
			updateSkipPreflight, updateSkipSystemTests, updateDryRunFlag,
			updateContinueOnFail, updateSkipLockRun, updateYesFlag,
			updateTypeFlag, updatePMFlag, updateRuleFlag, updateNameFlag,
			updateGroupFlag, updateConfigFlag, updateDirFlag, updateOutputFlag,
		}
		defer func() {
			loadConfigFunc = oldLoadConfig
			getPackagesFunc = oldGetPackages
			applyInstalledVersionsFunc = oldApplyInstalled
			listNewerVersionsFunc = oldListVersions
			updatePackageFunc = oldUpdatePkg
			updateSkipPreflight = oldUpdateFlags.skipPreflight
			updateSkipSystemTests = oldUpdateFlags.skipSystemTests
			updateDryRunFlag = oldUpdateFlags.dryRun
			updateContinueOnFail = oldUpdateFlags.continueOnFail
			updateSkipLockRun = oldUpdateFlags.skipLock
			updateYesFlag = oldUpdateFlags.yes
			updateTypeFlag = oldUpdateFlags.typeFlag
			updatePMFlag = oldUpdateFlags.pmFlag
			updateRuleFlag = oldUpdateFlags.ruleFlag
			updateNameFlag = oldUpdateFlags.nameFlag
			updateGroupFlag = oldUpdateFlags.groupFlag
			updateConfigFlag = oldUpdateFlags.configFlag
			updateDirFlag = oldUpdateFlags.dirFlag
			updateOutputFlag = oldUpdateFlags.outputFlag
			rootCmd.SetArgs(nil)
		}()

		// Setup mocks
		loadConfigFunc = func(path, workDir string) (*config.Config, error) {
			return &config.Config{
				WorkingDir: ".",
				Rules: map[string]config.PackageManagerCfg{
					"npm": {Manager: "js", Update: &config.UpdateCfg{}, Outdated: &config.OutdatedCfg{}},
				},
			}, nil
		}

		callCount := 0
		getPackagesFunc = func(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
			return []formats.Package{
				{Name: "react", Version: "17.0.0", Rule: "npm", PackageType: "js", Type: "prod"},
				{Name: "lodash", Version: "4.0.0", Rule: "npm", PackageType: "js", Type: "prod"},
			}, nil
		}
		applyInstalledVersionsFunc = func(pkgs []formats.Package, cfg *config.Config, workDir string) ([]formats.Package, error) {
			for i := range pkgs {
				pkgs[i].InstalledVersion = pkgs[i].Version
			}
			return pkgs, nil
		}
		listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, workDir string) ([]string, error) {
			return []string{"18.0.0", "19.0.0"}, nil
		}

		// First package succeeds, second fails - creating partial success
		updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if callCount == 2 {
				return stderrors.New("update failed for second package")
			}
			return nil
		}

		// Set flags for the test
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

		rootCmd.SetArgs([]string{"update", "--skip-preflight", "--skip-system-tests", "--dry-run", "--continue-on-fail", "--skip-lock", "-y"})
		Execute()

		// Partial success should result in ExitPartialFailure
		assert.Equal(t, errors.ExitPartialFailure, exitCode)
	})
}
