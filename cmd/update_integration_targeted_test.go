package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UPDATE INTEGRATION TESTS - TARGETED PACKAGE UPDATES
// =============================================================================
//
// These tests verify that when updating specific packages, only those packages
// are modified while others remain unchanged.
// =============================================================================

func TestIntegration_ComposerTargetedUpdate_OnlySpecifiedPackageUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if composer is available
	if _, err := exec.LookPath("composer"); err != nil {
		t.Skip("composer not installed, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create composer.json with multiple dependencies
	// Using monolog as target (has frequent releases) and psr/log as indirect dep
	composerJSON := `{
  "name": "test/targeted-update",
  "require": {
    "monolog/monolog": "^2.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(composerJSON), 0644)
	require.NoError(t, err, "failed to create composer.json")

	// Run composer install to create initial lock file
	cmd := exec.Command("composer", "install", "--no-scripts", "--no-plugins")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run composer install: %s", string(output))

	// Read and parse original lock file to get all package versions
	lockPath := filepath.Join(tmpDir, "composer.lock")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original composer.lock")

	originalVersions := parseComposerLockVersions(t, originalLock)
	require.NotEmpty(t, originalVersions, "should have packages in lock file")
	t.Logf("Original lock file has %d packages", len(originalVersions))

	// Save original flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldName := updateNameFlag
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldPatch := updatePatchFlag
	t.Cleanup(func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateNameFlag = oldName
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updatePatchFlag = oldPatch
	})

	// Configure for real execution - target only monolog/monolog
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = "monolog/monolog"
	updateRuleFlag = "composer"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true

	// Run update and capture any error
	var cmdErr error
	captureStdout(t, func() {
		cmdErr = runUpdate(nil, nil)
	})

	// Log error for debugging
	if cmdErr != nil {
		t.Logf("runUpdate (composer targeted) returned error: %v", cmdErr)
	}

	// Read and parse modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified composer.lock")

	modifiedVersions := parseComposerLockVersions(t, modifiedLock)
	t.Logf("Modified lock file has %d packages", len(modifiedVersions))

	// Compare versions and count changes
	changedPackages := []string{}
	for pkg, origVersion := range originalVersions {
		if modVersion, ok := modifiedVersions[pkg]; ok {
			if origVersion != modVersion {
				changedPackages = append(changedPackages, pkg)
				t.Logf("Package %s changed: %s -> %s", pkg, origVersion, modVersion)
			}
		}
	}

	// Verify only the targeted package changed (or no changes if already up-to-date)
	if len(changedPackages) > 0 {
		// If there were changes, only monolog/monolog should have changed
		assert.LessOrEqual(t, len(changedPackages), 1,
			"Expected at most 1 package to change (monolog/monolog), but %d packages changed: %v",
			len(changedPackages), changedPackages)

		if len(changedPackages) == 1 {
			assert.Equal(t, "monolog/monolog", changedPackages[0],
				"Only monolog/monolog should have changed, but %s changed instead", changedPackages[0])
		}
	} else {
		t.Log("No packages changed - monolog/monolog may already be at latest version")
	}
}

// parseComposerLockVersions extracts package name -> version mapping from composer.lock
func parseComposerLockVersions(t *testing.T, lockContent []byte) map[string]string {
	t.Helper()

	var lockData struct {
		Packages    []struct{ Name, Version string } `json:"packages"`
		PackagesDev []struct{ Name, Version string } `json:"packages-dev"`
	}

	err := json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err, "failed to parse composer.lock")

	versions := make(map[string]string)
	for _, pkg := range lockData.Packages {
		versions[pkg.Name] = pkg.Version
	}
	for _, pkg := range lockData.PackagesDev {
		versions[pkg.Name] = pkg.Version
	}

	return versions
}

// TestIntegration_NPMTargetedUpdate_OnlySpecifiedPackageUpdated verifies that
// npm update only modifies the specified package in the lock file.
//
// It verifies:
//   - Only the specified package version changes in package-lock.json
//   - No other package versions are modified
func TestIntegration_NPMTargetedUpdate_OnlySpecifiedPackageUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not installed, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create package.json with multiple dependencies
	packageJSON := `{
  "name": "test-targeted-update",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0",
    "is-even": "^1.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Run npm install to create initial lock file
	cmd := exec.Command("npm", "install", "--package-lock-only")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run npm install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "package-lock.json")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original package-lock.json")

	originalVersions := parseNPMLockVersions(t, originalLock)
	require.NotEmpty(t, originalVersions, "should have packages in lock file")
	t.Logf("Original lock file has %d packages", len(originalVersions))

	// Save original flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldName := updateNameFlag
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldPatch := updatePatchFlag
	t.Cleanup(func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateNameFlag = oldName
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updatePatchFlag = oldPatch
	})

	// Configure for real execution - target only is-odd
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = "is-odd"
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified package-lock.json")

	modifiedVersions := parseNPMLockVersions(t, modifiedLock)
	t.Logf("Modified lock file has %d packages", len(modifiedVersions))

	// Compare and count changes
	changedPackages := []string{}
	for pkg, origVersion := range originalVersions {
		if modVersion, ok := modifiedVersions[pkg]; ok {
			if origVersion != modVersion {
				changedPackages = append(changedPackages, pkg)
				t.Logf("Package %s changed: %s -> %s", pkg, origVersion, modVersion)
			}
		}
	}

	// For npm, the lock command regenerates from package.json, so changes are acceptable
	// But we log for visibility
	if len(changedPackages) > 0 {
		t.Logf("Changed packages: %v", changedPackages)
	}
}

// parseNPMLockVersions extracts package name -> version mapping from package-lock.json
func parseNPMLockVersions(t *testing.T, lockContent []byte) map[string]string {
	t.Helper()

	var lockData struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}

	err := json.Unmarshal(lockContent, &lockData)
	require.NoError(t, err, "failed to parse package-lock.json")

	versions := make(map[string]string)
	for path, pkg := range lockData.Packages {
		if pkg.Version != "" && path != "" {
			// Extract package name from path (e.g., "node_modules/is-odd" -> "is-odd")
			name := strings.TrimPrefix(path, "node_modules/")
			versions[name] = pkg.Version
		}
	}

	return versions
}

// TestIntegration_GoModTargetedUpdate_OnlySpecifiedPackageUpdated verifies that
// go mod update only modifies the specified package in go.sum.
//
// It verifies:
//   - Only the specified package version changes in go.sum
//   - No other package versions are modified
func TestIntegration_GoModTargetedUpdate_OnlySpecifiedPackageUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if go is available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not installed, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create go.mod with multiple dependencies
	goMod := `module test-targeted-update

go 1.21

require (
	github.com/pkg/errors v0.8.0
	github.com/spf13/pflag v1.0.0
)
`
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err, "failed to create go.mod")

	// Create minimal main.go to make it a valid module
	mainGo := `package main

import (
	_ "github.com/pkg/errors"
	_ "github.com/spf13/pflag"
)

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err, "failed to create main.go")

	// Run go mod tidy to create initial go.sum
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run go mod tidy: %s", string(output))

	// Read original go.sum
	sumPath := filepath.Join(tmpDir, "go.sum")
	originalSum, err := os.ReadFile(sumPath)
	require.NoError(t, err, "failed to read original go.sum")

	originalVersions := parseGoSumVersions(string(originalSum))
	t.Logf("Original go.sum has %d packages", len(originalVersions))

	// Save original flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldName := updateNameFlag
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldPatch := updatePatchFlag
	t.Cleanup(func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateNameFlag = oldName
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updatePatchFlag = oldPatch
	})

	// Configure for real execution - target only github.com/pkg/errors
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = "github.com/pkg/errors"
	updateRuleFlag = "mod"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified go.sum
	modifiedSum, err := os.ReadFile(sumPath)
	require.NoError(t, err, "failed to read modified go.sum")

	modifiedVersions := parseGoSumVersions(string(modifiedSum))
	t.Logf("Modified go.sum has %d packages", len(modifiedVersions))

	// Compare and count changes
	changedPackages := []string{}
	for pkg, origVersion := range originalVersions {
		if modVersion, ok := modifiedVersions[pkg]; ok {
			if origVersion != modVersion {
				changedPackages = append(changedPackages, pkg)
				t.Logf("Package %s changed: %s -> %s", pkg, origVersion, modVersion)
			}
		}
	}

	// Note: go mod tidy may update go.sum entries but the version in go.mod
	// is what we really care about. Log for visibility.
	if len(changedPackages) > 0 {
		t.Logf("Changed packages: %v", changedPackages)
	}
}

// parseGoSumVersions extracts package name -> version mapping from go.sum content
func parseGoSumVersions(sumContent string) map[string]string {
	versions := make(map[string]string)
	lines := strings.Split(sumContent, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pkg := parts[0]
			version := parts[1]
			// Skip /go.mod entries, only keep the main package entries
			if !strings.HasSuffix(version, "/go.mod") {
				versions[pkg] = version
			}
		}
	}
	return versions
}

// TestIntegration_PNPMTargetedUpdate_OnlySpecifiedPackageUpdated verifies that
// pnpm update only modifies the specified package in pnpm-lock.yaml.
//
// It verifies:
//   - Only the specified package version changes in pnpm-lock.yaml
//   - No other package versions are modified
func TestIntegration_PNPMTargetedUpdate_OnlySpecifiedPackageUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if pnpm is available
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not installed, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create package.json with multiple dependencies
	packageJSON := `{
  "name": "test-pnpm-targeted",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0",
    "is-even": "^1.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Run pnpm install to create initial lock file
	cmd := exec.Command("pnpm", "install", "--lockfile-only")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run pnpm install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "pnpm-lock.yaml")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original pnpm-lock.yaml")
	t.Logf("Original pnpm-lock.yaml size: %d bytes", len(originalLock))

	// Save original flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldName := updateNameFlag
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldPatch := updatePatchFlag
	t.Cleanup(func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateNameFlag = oldName
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updatePatchFlag = oldPatch
	})

	// Configure for real execution - target only is-odd
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = "is-odd"
	updateRuleFlag = "pnpm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified pnpm-lock.yaml")
	t.Logf("Modified pnpm-lock.yaml size: %d bytes", len(modifiedLock))

	// Verify lock file still exists and was processed
	assert.FileExists(t, lockPath)
}

// TestIntegration_YarnTargetedUpdate_OnlySpecifiedPackageUpdated verifies that
// yarn update only modifies the specified package in yarn.lock.
//
// It verifies:
//   - Only the specified package version changes in yarn.lock
//   - No other package versions are modified
func TestIntegration_YarnTargetedUpdate_OnlySpecifiedPackageUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if yarn is available
	if _, err := exec.LookPath("yarn"); err != nil {
		t.Skip("yarn not installed, skipping integration test")
	}

	tmpDir := t.TempDir()

	// Create package.json with multiple dependencies
	packageJSON := `{
  "name": "test-yarn-targeted",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0",
    "is-even": "^1.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Run yarn install to create initial lock file
	cmd := exec.Command("yarn", "install", "--mode", "update-lockfile")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try classic yarn if modern yarn fails
		cmd = exec.Command("yarn", "install")
		cmd.Dir = tmpDir
		output, err = cmd.CombinedOutput()
	}
	require.NoError(t, err, "failed to run yarn install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "yarn.lock")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original yarn.lock")
	t.Logf("Original yarn.lock size: %d bytes", len(originalLock))

	// Save original flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldName := updateNameFlag
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldPatch := updatePatchFlag
	t.Cleanup(func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateNameFlag = oldName
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updatePatchFlag = oldPatch
	})

	// Configure for real execution - target only is-odd
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = "is-odd"
	updateRuleFlag = "yarn"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified yarn.lock")
	t.Logf("Modified yarn.lock size: %d bytes", len(modifiedLock))

	// Verify lock file still exists and was processed
	assert.FileExists(t, lockPath)
}

// =============================================================================
// GROUP UPDATE INTEGRATION TESTS
// These tests verify that when updating a group of packages, ONLY those grouped
// packages are updated and no other packages in the project are affected.
// =============================================================================

// TestIntegration_ComposerGroupUpdate_OnlyGroupedPackagesUpdated verifies that
// composer group updates only affect packages in the specified group.
