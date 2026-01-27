package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// UPDATE INTEGRATION TESTS - GROUP PACKAGE UPDATES
// =============================================================================
//
// These tests verify that group updates work correctly - when packages are
// configured to update together as a group, all packages in the group are
// updated while others remain unchanged.
// =============================================================================

func TestIntegration_ComposerGroupUpdate_OnlyGroupedPackagesUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if composer is available
	if _, err := exec.LookPath("composer"); err != nil {
		t.Skip("composer not available")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goupdate-composer-group-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create composer.json with multiple packages - some grouped, some not
	// Using older versions that can be updated
	composerJSON := `{
	"name": "test/group-update",
	"require": {
		"psr/log": "^1.1",
		"psr/container": "^1.0",
		"psr/http-message": "^1.0"
	}
}`
	err = os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(composerJSON), 0644)
	require.NoError(t, err, "failed to create composer.json")

	// Create .goupdate.yml with a group containing only psr/log and psr/container
	// psr/http-message is NOT in the group
	goupdateYML := `
extends: [default]
rules:
  composer:
    groups:
      psr-core:
        - psr/log
        - psr/container
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(goupdateYML), 0644)
	require.NoError(t, err, "failed to create .goupdate.yml")

	// Run composer install to create lock file
	cmd := exec.Command("composer", "install", "--no-interaction", "--prefer-dist")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run composer install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "composer.lock")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original composer.lock")

	// Parse original versions
	originalVersions := parseComposerLockVersions(t, originalLock)
	t.Logf("Original versions: %v", originalVersions)

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
	oldGroup := updateGroupFlag
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
		updateGroupFlag = oldGroup
	})

	// Configure for real execution - target the psr-core group
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = ""
	updateRuleFlag = "composer"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true
	updateGroupFlag = "psr-core"

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified composer.lock")

	// Parse modified versions
	modifiedVersions := parseComposerLockVersions(t, modifiedLock)
	t.Logf("Modified versions: %v", modifiedVersions)

	// Count changed packages
	changedPackages := []string{}
	for pkg, origVer := range originalVersions {
		if modVer, exists := modifiedVersions[pkg]; exists && origVer != modVer {
			changedPackages = append(changedPackages, pkg)
			t.Logf("Package %s changed: %s -> %s", pkg, origVer, modVer)
		}
	}

	// Verify that ONLY grouped packages (psr/log, psr/container) were updated
	// psr/http-message should NOT be updated
	for _, pkg := range changedPackages {
		isGrouped := pkg == "psr/log" || pkg == "psr/container"
		assert.True(t, isGrouped, "Package %s was updated but is NOT in the group psr-core", pkg)
	}

	// Verify psr/http-message was NOT changed
	if origVer, exists := originalVersions["psr/http-message"]; exists {
		modVer := modifiedVersions["psr/http-message"]
		assert.Equal(t, origVer, modVer, "psr/http-message should NOT be updated (not in group)")
	}

	t.Logf("Group update verified: %d packages changed, all in group", len(changedPackages))
}

// TestIntegration_NPMGroupUpdate_OnlyGroupedPackagesUpdated verifies that
// npm group updates only affect packages in the specified group.
func TestIntegration_NPMGroupUpdate_OnlyGroupedPackagesUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if npm is available
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not available")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goupdate-npm-group-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create package.json with multiple packages - some grouped, some not
	packageJSON := `{
	"name": "test-npm-group",
	"version": "1.0.0",
	"dependencies": {
		"is-odd": "^3.0.0",
		"is-even": "^1.0.0",
		"is-number": "^7.0.0"
	}
}`
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Create .goupdate.yml with a group containing only is-odd and is-even
	// is-number is NOT in the group
	goupdateYML := `
extends: [default]
rules:
  npm:
    groups:
      is-checks:
        - is-odd
        - is-even
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(goupdateYML), 0644)
	require.NoError(t, err, "failed to create .goupdate.yml")

	// Run npm install to create lock file
	cmd := exec.Command("npm", "install", "--package-lock-only")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run npm install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "package-lock.json")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original package-lock.json")

	// Parse original versions
	originalVersions := parseNPMLockVersions(t, originalLock)
	t.Logf("Original versions: %v", originalVersions)

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
	oldGroup := updateGroupFlag
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
		updateGroupFlag = oldGroup
	})

	// Configure for real execution - target the is-checks group
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = ""
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true
	updateGroupFlag = "is-checks"

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified package-lock.json")

	// Parse modified versions
	modifiedVersions := parseNPMLockVersions(t, modifiedLock)
	t.Logf("Modified versions: %v", modifiedVersions)

	// Count changed packages (only top-level, not transitive dependencies)
	topLevelPackages := []string{"is-odd", "is-even", "is-number"}
	changedPackages := []string{}
	for _, pkg := range topLevelPackages {
		origVer := originalVersions[pkg]
		modVer := modifiedVersions[pkg]
		if origVer != "" && modVer != "" && origVer != modVer {
			changedPackages = append(changedPackages, pkg)
			t.Logf("Package %s changed: %s -> %s", pkg, origVer, modVer)
		}
	}

	// Verify that ONLY grouped packages were potentially updated
	// is-number should NOT be updated
	for _, pkg := range changedPackages {
		isGrouped := pkg == "is-odd" || pkg == "is-even"
		assert.True(t, isGrouped, "Package %s was updated but is NOT in the group is-checks", pkg)
	}

	// Verify is-number was NOT changed
	if origVer := originalVersions["is-number"]; origVer != "" {
		modVer := modifiedVersions["is-number"]
		assert.Equal(t, origVer, modVer, "is-number should NOT be updated (not in group)")
	}

	t.Logf("Group update verified: %d top-level packages changed, all in group", len(changedPackages))
}

// TestIntegration_GoModGroupUpdate_OnlyGroupedPackagesUpdated verifies that
// Go mod group updates only affect packages in the specified group.
func TestIntegration_GoModGroupUpdate_OnlyGroupedPackagesUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if go is available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goupdate-gomod-group-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create go.mod with multiple packages - some grouped, some not
	goMod := `module testmod

go 1.21

require (
	github.com/stretchr/testify v1.8.0
	github.com/pkg/errors v0.9.0
	golang.org/x/text v0.3.0
)
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err, "failed to create go.mod")

	// Create minimal main.go to make it a valid module
	mainGo := `package main

import (
	_ "github.com/stretchr/testify/assert"
	_ "github.com/pkg/errors"
	_ "golang.org/x/text/language"
)

func main() {}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err, "failed to create main.go")

	// Create .goupdate.yml with a group containing only stretchr/testify and pkg/errors
	// golang.org/x/text is NOT in the group
	goupdateYML := `
extends: [default]
rules:
  mod:
    groups:
      github-pkgs:
        - github.com/stretchr/testify
        - github.com/pkg/errors
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(goupdateYML), 0644)
	require.NoError(t, err, "failed to create .goupdate.yml")

	// Run go mod tidy to create go.sum
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run go mod tidy: %s", string(output))

	// Read original go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	originalGoMod, err := os.ReadFile(goModPath)
	require.NoError(t, err, "failed to read original go.mod")

	// Parse original versions from go.mod
	originalVersions := parseGoModVersions(string(originalGoMod))
	t.Logf("Original versions: %v", originalVersions)

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
	oldGroup := updateGroupFlag
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
		updateGroupFlag = oldGroup
	})

	// Configure for real execution - target the github-pkgs group
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = ""
	updateRuleFlag = "mod"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true
	updateGroupFlag = "github-pkgs"

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified go.mod
	modifiedGoMod, err := os.ReadFile(goModPath)
	require.NoError(t, err, "failed to read modified go.mod")

	// Parse modified versions
	modifiedVersions := parseGoModVersions(string(modifiedGoMod))
	t.Logf("Modified versions: %v", modifiedVersions)

	// Count changed packages
	changedPackages := []string{}
	for pkg, origVer := range originalVersions {
		if modVer, exists := modifiedVersions[pkg]; exists && origVer != modVer {
			changedPackages = append(changedPackages, pkg)
			t.Logf("Package %s changed: %s -> %s", pkg, origVer, modVer)
		}
	}

	// Verify that ONLY grouped packages were potentially updated
	// golang.org/x/text should NOT be updated
	for _, pkg := range changedPackages {
		isGrouped := pkg == "github.com/stretchr/testify" || pkg == "github.com/pkg/errors"
		assert.True(t, isGrouped, "Package %s was updated but is NOT in the group github-pkgs", pkg)
	}

	// Verify golang.org/x/text was NOT changed
	if origVer, exists := originalVersions["golang.org/x/text"]; exists {
		modVer := modifiedVersions["golang.org/x/text"]
		assert.Equal(t, origVer, modVer, "golang.org/x/text should NOT be updated (not in group)")
	}

	t.Logf("Group update verified: %d packages changed, all in group", len(changedPackages))
}

// parseGoModVersions extracts package versions from go.mod content
func parseGoModVersions(content string) map[string]string {
	versions := make(map[string]string)
	// Pattern matches: github.com/pkg/errors v0.9.1
	re := regexp.MustCompile(`(?m)^\s*(\S+)\s+(v[\d.]+)`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			versions[match[1]] = match[2]
		}
	}
	return versions
}

// TestIntegration_PNPMGroupUpdate_OnlyGroupedPackagesUpdated verifies that
// pnpm group updates only affect packages in the specified group.
func TestIntegration_PNPMGroupUpdate_OnlyGroupedPackagesUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if pnpm is available
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not available")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goupdate-pnpm-group-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create package.json with multiple packages - some grouped, some not
	packageJSON := `{
	"name": "test-pnpm-group",
	"version": "1.0.0",
	"dependencies": {
		"is-odd": "^3.0.0",
		"is-even": "^1.0.0",
		"is-number": "^7.0.0"
	}
}`
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Create .goupdate.yml with a group containing only is-odd and is-even
	// is-number is NOT in the group
	goupdateYML := `
extends: [default]
rules:
  pnpm:
    groups:
      is-checks:
        - is-odd
        - is-even
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(goupdateYML), 0644)
	require.NoError(t, err, "failed to create .goupdate.yml")

	// Run pnpm install to create lock file
	cmd := exec.Command("pnpm", "install", "--lockfile-only")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to run pnpm install: %s", string(output))

	// Read original lock file
	lockPath := filepath.Join(tmpDir, "pnpm-lock.yaml")
	originalLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read original pnpm-lock.yaml")
	t.Logf("Original pnpm-lock.yaml size: %d bytes", len(originalLock))

	// Parse original versions
	originalVersions := parsePNPMLockVersions(string(originalLock))
	t.Logf("Original versions: %v", originalVersions)

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
	oldGroup := updateGroupFlag
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
		updateGroupFlag = oldGroup
	})

	// Configure for real execution - target the is-checks group
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = ""
	updateRuleFlag = "pnpm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true
	updateGroupFlag = "is-checks"

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified pnpm-lock.yaml")

	// Parse modified versions
	modifiedVersions := parsePNPMLockVersions(string(modifiedLock))
	t.Logf("Modified versions: %v", modifiedVersions)

	// Count changed packages (only top-level, not transitive dependencies)
	topLevelPackages := []string{"is-odd", "is-even", "is-number"}
	changedPackages := []string{}
	for _, pkg := range topLevelPackages {
		origVer := originalVersions[pkg]
		modVer := modifiedVersions[pkg]
		if origVer != "" && modVer != "" && origVer != modVer {
			changedPackages = append(changedPackages, pkg)
			t.Logf("Package %s changed: %s -> %s", pkg, origVer, modVer)
		}
	}

	// Verify that ONLY grouped packages were potentially updated
	// is-number should NOT be updated
	for _, pkg := range changedPackages {
		isGrouped := pkg == "is-odd" || pkg == "is-even"
		assert.True(t, isGrouped, "Package %s was updated but is NOT in the group is-checks", pkg)
	}

	// Verify is-number was NOT changed
	if origVer := originalVersions["is-number"]; origVer != "" {
		modVer := modifiedVersions["is-number"]
		assert.Equal(t, origVer, modVer, "is-number should NOT be updated (not in group)")
	}

	t.Logf("Group update verified: %d top-level packages changed, all in group", len(changedPackages))
}

// parsePNPMLockVersions extracts package versions from pnpm-lock.yaml content
func parsePNPMLockVersions(content string) map[string]string {
	versions := make(map[string]string)
	// Try v9 format first: package@version in dependencies section
	// Pattern: 'is-odd': version: 3.0.1
	reV9 := regexp.MustCompile(`(?m)^\s+'?([^@'\s:]+)'?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(\d+\.\d+\.\d+)`)
	matchesV9 := reV9.FindAllStringSubmatch(content, -1)
	for _, match := range matchesV9 {
		if len(match) >= 3 {
			versions[match[1]] = match[2]
		}
	}

	// Try v6/v7/v8 format: /package@version:
	reV678 := regexp.MustCompile(`(?m)^/([^@]+)@(\d+\.\d+\.\d+):`)
	matchesV678 := reV678.FindAllStringSubmatch(content, -1)
	for _, match := range matchesV678 {
		if len(match) >= 3 {
			versions[match[1]] = match[2]
		}
	}
	return versions
}

// TestIntegration_YarnGroupUpdate_OnlyGroupedPackagesUpdated verifies that
// yarn group updates only affect packages in the specified group.
func TestIntegration_YarnGroupUpdate_OnlyGroupedPackagesUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if yarn is available
	if _, err := exec.LookPath("yarn"); err != nil {
		t.Skip("yarn not available")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goupdate-yarn-group-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	// Create package.json with multiple packages - some grouped, some not
	packageJSON := `{
	"name": "test-yarn-group",
	"version": "1.0.0",
	"dependencies": {
		"is-odd": "^3.0.0",
		"is-even": "^1.0.0",
		"is-number": "^7.0.0"
	}
}`
	err = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err, "failed to create package.json")

	// Create .goupdate.yml with a group containing only is-odd and is-even
	// is-number is NOT in the group
	goupdateYML := `
extends: [default]
rules:
  yarn:
    groups:
      is-checks:
        - is-odd
        - is-even
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(goupdateYML), 0644)
	require.NoError(t, err, "failed to create .goupdate.yml")

	// Run yarn install to create lock file
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

	// Parse original versions
	originalVersions := parseYarnLockVersions(string(originalLock))
	t.Logf("Original versions: %v", originalVersions)

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
	oldGroup := updateGroupFlag
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
		updateGroupFlag = oldGroup
	})

	// Configure for real execution - target the is-checks group
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = false // REAL EXECUTION
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = false
	updateNameFlag = ""
	updateRuleFlag = "yarn"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updatePatchFlag = true
	updateGroupFlag = "is-checks"

	// Run update
	captureStdout(t, func() {
		_ = runUpdate(nil, nil)
	})

	// Read modified lock file
	modifiedLock, err := os.ReadFile(lockPath)
	require.NoError(t, err, "failed to read modified yarn.lock")

	// Parse modified versions
	modifiedVersions := parseYarnLockVersions(string(modifiedLock))
	t.Logf("Modified versions: %v", modifiedVersions)

	// Count changed packages (only top-level, not transitive dependencies)
	topLevelPackages := []string{"is-odd", "is-even", "is-number"}
	changedPackages := []string{}
	for _, pkg := range topLevelPackages {
		origVer := originalVersions[pkg]
		modVer := modifiedVersions[pkg]
		if origVer != "" && modVer != "" && origVer != modVer {
			changedPackages = append(changedPackages, pkg)
			t.Logf("Package %s changed: %s -> %s", pkg, origVer, modVer)
		}
	}

	// Verify that ONLY grouped packages were potentially updated
	// is-number should NOT be updated
	for _, pkg := range changedPackages {
		isGrouped := pkg == "is-odd" || pkg == "is-even"
		assert.True(t, isGrouped, "Package %s was updated but is NOT in the group is-checks", pkg)
	}

	// Verify is-number was NOT changed
	if origVer := originalVersions["is-number"]; origVer != "" {
		modVer := modifiedVersions["is-number"]
		assert.Equal(t, origVer, modVer, "is-number should NOT be updated (not in group)")
	}

	t.Logf("Group update verified: %d top-level packages changed, all in group", len(changedPackages))
}

// parseYarnLockVersions extracts package versions from yarn.lock content
func parseYarnLockVersions(content string) map[string]string {
	versions := make(map[string]string)
	// Classic yarn format: "package@^version":\n  version "x.y.z"
	reClassic := regexp.MustCompile(`(?m)^"?([^@"]+)@[^:]+:\s*\n\s+version\s+"([^"]+)"`)
	matchesClassic := reClassic.FindAllStringSubmatch(content, -1)
	for _, match := range matchesClassic {
		if len(match) >= 3 {
			versions[match[1]] = match[2]
		}
	}

	// Berry format: "package@npm:^version":\n  version: x.y.z
	reBerry := regexp.MustCompile(`(?m)^"([^@"]+)@(?:npm:)?[^"]+":\s*\n\s+version:\s*([^\s\n]+)`)
	matchesBerry := reBerry.FindAllStringSubmatch(content, -1)
	for _, match := range matchesBerry {
		if len(match) >= 3 {
			versions[match[1]] = match[2]
		}
	}
	return versions
}

// =============================================================================
// SYSTEM TEST AND ROLLBACK INTEGRATION TESTS
// These tests verify that system tests run correctly after updates and that
// rollback works properly when system tests fail.
// =============================================================================

// TestIntegration_SystemTests_AfterEach_RollbackOnFailure verifies that when
// system tests are configured with run_mode: after_each and a test fails,
// the update is rolled back to the original version.
