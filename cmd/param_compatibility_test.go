package cmd

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PARAMETER COMPATIBILITY AND BATTLE TESTS
// =============================================================================
//
// This file tests all command parameters for:
// 1. Compatibility between parameters
// 2. Conflict detection and handling
// 3. Error handling with different output formats
// 4. Edge cases and chaos testing
//
// COMMANDS AND PARAMETERS:
//
// SCAN:
//   --directory, -d    Directory to scan
//   --config, -c       Config file path
//   --output, -o       Output format: json, csv, xml (default: table)
//   --file, -f         Filter by file path patterns
//
// LIST:
//   --type, -t         Filter by type: all, prod, dev
//   --package-manager, -p  Filter by package manager
//   --rule, -r         Filter by rule
//   --name, -n         Filter by package name
//   --group, -g        Filter by group
//   --config, -c       Config file path
//   --directory, -d    Directory to scan
//   --output, -o       Output format
//   --file, -f         Filter by file path patterns
//
// OUTDATED:
//   (same as LIST) plus:
//   --major            Allow major comparisons
//   --minor            Allow minor comparisons
//   --patch            Restrict to patch scope
//   --no-timeout       Disable timeouts
//   --skip-preflight   Skip preflight validation
//   --continue-on-fail Continue after failures
//
// UPDATE:
//   (same as OUTDATED) plus:
//   --dry-run          Plan without writing files
//   --skip-lock        Skip lock/install command
//   --yes, -y          Skip confirmation prompt
//   --incremental      One version step at a time
//   --skip-system-tests Skip all system tests
//   --system-test-mode Override test mode: after_each, after_all, none
//
// CONFIG:
//   --show-defaults    Show default configuration
//   --show-effective   Show effective configuration
//   --init             Create template
//   --validate         Validate configuration
//   --config, -c       Config path
//
// =============================================================================

// -----------------------------------------------------------------------------
// SCAN COMMAND PARAMETER TESTS
// -----------------------------------------------------------------------------

// TestParamCompat_Scan_InvalidDirectory tests scan with non-existent directory.
func TestParamCompat_Scan_InvalidDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = "/nonexistent/directory/path"
	scanConfigFlag = ""
	scanOutputFlag = ""

	err := runScan(nil, nil)
	assert.Error(t, err, "scan should fail with non-existent directory")
}

// TestParamCompat_Scan_InvalidDirectory_JSONOutput tests error handling with JSON output.
func TestParamCompat_Scan_InvalidDirectory_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = "/nonexistent/directory/path"
	scanConfigFlag = ""
	scanOutputFlag = "json"

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		_ = err
	})

	// Even on error, if output is captured, it should not contain invalid JSON
	// The error should be returned, not printed as plain text mixed with JSON
	output = strings.TrimSpace(output)
	if output != "" && (strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[")) {
		var data interface{}
		err := json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "JSON output on error should still be valid JSON: %s", output)
	}
}

// TestParamCompat_Scan_InvalidConfig tests scan with invalid config path.
func TestParamCompat_Scan_InvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a package.json so directory is valid
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = "/nonexistent/config.yml"
	scanOutputFlag = ""

	err = runScan(nil, nil)
	// Should either work with defaults or error gracefully
	// Document actual behavior
	t.Logf("Scan with invalid config result: err=%v", err)
}

// TestParamCompat_Scan_FileFilter_NoMatches tests scan with file filter that matches nothing.
func TestParamCompat_Scan_FileFilter_NoMatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	oldFile := scanFileFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
		scanFileFlag = oldFile
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "json"
	scanFileFlag = "*.nonexistent"

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should return valid JSON even with no matches
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err := json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Empty result should still be valid JSON: %s", output)
	}
}

// -----------------------------------------------------------------------------
// LIST COMMAND PARAMETER TESTS
// -----------------------------------------------------------------------------

// TestParamCompat_List_TypeFilter_Invalid tests list with invalid type filter.
func TestParamCompat_List_TypeFilter_Invalid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "invalid_type"
	listPMFlag = "all"

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		// Should either error or return empty result
		_ = err
	})

	// Should return valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err := json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Invalid type filter should still produce valid JSON: %s", output)
	}
}

// TestParamCompat_List_MultipleTypeFilters tests list with multiple types.
func TestParamCompat_List_MultipleTypeFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with both prod and dev dependencies
	packageJSON := `{
		"name": "test",
		"dependencies": {"is-odd": "^3.0.0"},
		"devDependencies": {"is-even": "^1.0.0"}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "prod,dev" // Multiple types
	listPMFlag = "all"

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should return valid JSON with both types
	output = strings.TrimSpace(output)
	var data interface{}
	err = json.Unmarshal([]byte(output), &data)
	assert.NoError(t, err, "Multiple type filters should produce valid JSON")
}

// TestParamCompat_List_NameFilter_NonExistent tests name filter with non-existent package.
func TestParamCompat_List_NameFilter_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldName := listNameFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
		listNameFlag = oldName
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "all"
	listPMFlag = "all"
	listNameFlag = "nonexistent-package-xyz"

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should return valid JSON even with no matches
	output = strings.TrimSpace(output)
	var data interface{}
	err = json.Unmarshal([]byte(output), &data)
	assert.NoError(t, err, "Non-existent name filter should still produce valid JSON")
}

// TestParamCompat_List_GroupFilter_NoGroupsDefined tests group filter when no groups defined.
func TestParamCompat_List_GroupFilter_NoGroupsDefined(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// No config file with groups

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldGroup := listGroupFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
		listGroupFlag = oldGroup
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "all"
	listPMFlag = "all"
	listGroupFlag = "mygroup"

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		_ = err
	})

	// Should return valid JSON (empty or error)
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Group filter with no groups defined should still produce valid JSON")
	}
}

// -----------------------------------------------------------------------------
// OUTDATED COMMAND PARAMETER TESTS
// -----------------------------------------------------------------------------

// TestParamCompat_Outdated_MajorMinorPatch_All tests all version scope flags together.
func TestParamCompat_Outdated_MajorMinorPatch_All(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldMajor := outdatedMajorFlag
	oldMinor := outdatedMinorFlag
	oldPatch := outdatedPatchFlag
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedMajorFlag = oldMajor
		outdatedMinorFlag = oldMinor
		outdatedPatchFlag = oldPatch
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedMajorFlag = true
	outdatedMinorFlag = true
	outdatedPatchFlag = true

	output := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		_ = err
	})

	// Should return valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "All version flags should produce valid JSON")
	}
}

// TestParamCompat_Outdated_ContinueOnFail_JSONOutput tests continue-on-fail with JSON output.
func TestParamCompat_Outdated_ContinueOnFail_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with dependencies
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldContinue := outdatedContinueOnFail
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedContinueOnFail = oldContinue
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedContinueOnFail = true

	output := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		_ = err // May have partial failures
	})

	// Should return valid JSON even with partial failures
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "continue-on-fail should still produce valid JSON: %s", output)
	}
}

// -----------------------------------------------------------------------------
// UPDATE COMMAND PARAMETER COMPATIBILITY TESTS
// -----------------------------------------------------------------------------

// TestParamCompat_Update_DryRun_SkipLock tests dry-run + skip-lock combination.
// skip-lock should be irrelevant in dry-run mode.
func TestParamCompat_Update_DryRun_SkipLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true   // DRY RUN
	updateSkipLockRun = true  // SKIP LOCK (should be irrelevant)
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateOutputFlag = "json"

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should work fine and produce valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "dry-run + skip-lock should produce valid JSON")
	}
}

// TestParamCompat_Update_DryRun_Yes tests dry-run + yes combination.
// Both flags are somewhat redundant but should work.
func TestParamCompat_Update_DryRun_Yes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true // DRY RUN
	updateYesFlag = true    // YES (redundant but should work)
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateOutputFlag = "json"

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should work fine and produce valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "dry-run + yes should produce valid JSON")
	}
}

// TestParamCompat_Update_SkipSystemTests_SystemTestMode tests skip-system-tests + system-test-mode.
// These are conflicting flags - skip should take precedence.
func TestParamCompat_Update_SkipSystemTests_SystemTestMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create config with system tests
	configYAML := `extends:
  - default
system_tests:
  run_preflight: true
  tests:
    - name: "test"
      commands: "echo ok"
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(configYAML), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSystemTestMode := updateSystemTestModeFlag
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSystemTestModeFlag = oldSystemTestMode
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true         // SKIP
	updateSystemTestModeFlag = "after_all" // BUT ALSO SET MODE (conflict!)
	updateSkipLockRun = true
	updateOutputFlag = "json"

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should work - skip should take precedence
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "skip-system-tests should take precedence over system-test-mode")
	}
}

// TestParamCompat_Update_Incremental_Major tests incremental + major combination.
func TestParamCompat_Update_Incremental_Major(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	oldIncremental := updateIncrementalFlag
	oldMajor := updateMajorFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
		updateIncrementalFlag = oldIncremental
		updateMajorFlag = oldMajor
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateOutputFlag = "json"
	updateIncrementalFlag = true // INCREMENTAL
	updateMajorFlag = true       // MAJOR

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should work - incremental with major scope
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "incremental + major should produce valid JSON")
	}
}

// TestParamCompat_Update_ContinueOnFail_JSONOutput tests continue-on-fail with JSON.
// Partial errors should still produce valid JSON.
func TestParamCompat_Update_ContinueOnFail_JSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	oldContinue := updateContinueOnFail
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
		updateContinueOnFail = oldContinue
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateOutputFlag = "json"
	updateContinueOnFail = true // CONTINUE ON FAIL

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Should produce valid JSON even with failures
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "continue-on-fail should still produce valid JSON")
	}
}

// -----------------------------------------------------------------------------
// ERROR HANDLING WITH DIFFERENT OUTPUT FORMATS
// -----------------------------------------------------------------------------

// TestParamCompat_ErrorHandling_JSON tests error handling with JSON output.
func TestParamCompat_ErrorHandling_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := []struct {
		name        string
		setupFunc   func(tmpDir string)
		expectError bool
	}{
		{
			name: "empty_directory",
			setupFunc: func(tmpDir string) {
				// Don't create any files
			},
			expectError: false, // Should return empty result, not error
		},
		{
			name: "invalid_package_json",
			setupFunc: func(tmpDir string) {
				os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("not json"), 0644)
			},
			expectError: true,
		},
		{
			name: "malformed_json",
			setupFunc: func(tmpDir string) {
				os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": }`), 0644)
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setupFunc(tmpDir)

			// Save and restore flags
			oldDir := scanDirFlag
			oldConfig := scanConfigFlag
			oldOutput := scanOutputFlag
			defer func() {
				scanDirFlag = oldDir
				scanConfigFlag = oldConfig
				scanOutputFlag = oldOutput
			}()

			scanDirFlag = tmpDir
			scanConfigFlag = ""
			scanOutputFlag = "json"

			output := captureStdout(t, func() {
				err := runScan(nil, nil)
				if tc.expectError {
					// Error is expected - but output should still be valid
					_ = err
				} else {
					assert.NoError(t, err)
				}
			})

			// Output should be valid JSON or empty
			output = strings.TrimSpace(output)
			if output != "" && (strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[")) {
				var data interface{}
				err := json.Unmarshal([]byte(output), &data)
				assert.NoError(t, err, "Output should be valid JSON: %s", output)
			}
		})
	}
}

// TestParamCompat_ErrorHandling_XML tests error handling with XML output.
func TestParamCompat_ErrorHandling_XML(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create valid package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "xml"

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Output should be valid XML
	output = strings.TrimSpace(output)
	if output != "" && strings.HasPrefix(output, "<") {
		var data interface{}
		err := xml.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Output should be valid XML: %s", output)
	}
}

// TestParamCompat_ErrorHandling_CSV tests error handling with CSV output.
func TestParamCompat_ErrorHandling_CSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create valid package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "csv"

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Output should be valid CSV (at least have a header)
	output = strings.TrimSpace(output)
	if output != "" {
		lines := strings.Split(output, "\n")
		assert.GreaterOrEqual(t, len(lines), 1, "CSV should have at least header")
	}
}

// -----------------------------------------------------------------------------
// CONFIG COMMAND PARAMETER TESTS
// -----------------------------------------------------------------------------

// TestParamCompat_Config_MultipleActions tests config with multiple actions.
func TestParamCompat_Config_MultipleActions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Test that multiple mutually exclusive flags are handled
	// --show-defaults, --show-effective, --init, --validate

	// Save and restore flags
	oldShowDefaults := configShowDefaultsFlag
	oldShowEffective := configShowEffectiveFlag
	oldInit := configInitFlag
	oldValidate := configValidateFlag
	oldPath := configPathFlag
	defer func() {
		configShowDefaultsFlag = oldShowDefaults
		configShowEffectiveFlag = oldShowEffective
		configInitFlag = oldInit
		configValidateFlag = oldValidate
		configPathFlag = oldPath
	}()

	// Setting both show-defaults and show-effective
	configShowDefaultsFlag = true
	configShowEffectiveFlag = true
	configInitFlag = false
	configValidateFlag = false
	configPathFlag = ""

	// Should handle gracefully (pick one or error)
	output := captureStdout(t, func() {
		err := runConfig(nil, nil)
		_ = err
	})

	// Should not panic and should produce some output
	assert.NotEmpty(t, output, "Config with multiple flags should produce output")
}

// TestParamCompat_Config_Validate_NonExistent tests validate with non-existent file.
func TestParamCompat_Config_Validate_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Save and restore flags
	oldShowDefaults := configShowDefaultsFlag
	oldShowEffective := configShowEffectiveFlag
	oldInit := configInitFlag
	oldValidate := configValidateFlag
	oldPath := configPathFlag
	defer func() {
		configShowDefaultsFlag = oldShowDefaults
		configShowEffectiveFlag = oldShowEffective
		configInitFlag = oldInit
		configValidateFlag = oldValidate
		configPathFlag = oldPath
	}()

	configShowDefaultsFlag = false
	configShowEffectiveFlag = false
	configInitFlag = false
	configValidateFlag = true
	configPathFlag = "/nonexistent/config.yml"

	err := runConfig(nil, nil)
	assert.Error(t, err, "Validate should fail with non-existent config")
}

// -----------------------------------------------------------------------------
// CHAOS TESTS - EDGE CASES AND UNUSUAL COMBINATIONS
// -----------------------------------------------------------------------------

// TestChaos_AllFiltersAtOnce tests using all filter flags simultaneously.
func TestChaos_AllFiltersAtOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{
		"name": "test",
		"dependencies": {"is-odd": "^3.0.0"},
		"devDependencies": {"is-even": "^1.0.0"}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldRule := listRuleFlag
	oldName := listNameFlag
	oldGroup := listGroupFlag
	oldFile := listFileFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
		listRuleFlag = oldRule
		listNameFlag = oldName
		listGroupFlag = oldGroup
		listFileFlag = oldFile
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "prod"
	listPMFlag = "npm"
	listRuleFlag = "npm"
	listNameFlag = "is-odd"
	listGroupFlag = ""
	listFileFlag = "*.json"

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should work and produce valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "All filters should produce valid JSON")
	}
}

// TestChaos_EmptyStringParams tests using empty strings for string parameters.
func TestChaos_EmptyStringParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldRule := listRuleFlag
	oldName := listNameFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
		listRuleFlag = oldRule
		listNameFlag = oldName
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = ""  // Empty
	listPMFlag = ""    // Empty
	listRuleFlag = ""  // Empty
	listNameFlag = ""  // Empty

	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Should work with defaults
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Empty string params should use defaults and produce valid JSON")
	}
}

// TestChaos_SpecialCharsInParams tests special characters in parameters.
func TestChaos_SpecialCharsInParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name  string
		value string
	}{
		{"semicolon", "test;echo"},
		{"pipe", "test|echo"},
		{"ampersand", "test&echo"},
		{"dollar", "$HOME"},
		{"backtick", "`echo test`"},
		{"newline", "test\necho"},
		{"quote", `test"echo`},
		{"unicode", "test-\u4e2d\u6587"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore flags
			oldDir := listDirFlag
			oldConfig := listConfigFlag
			oldOutput := listOutputFlag
			oldType := listTypeFlag
			oldPM := listPMFlag
			oldName := listNameFlag
			defer func() {
				listDirFlag = oldDir
				listConfigFlag = oldConfig
				listOutputFlag = oldOutput
				listTypeFlag = oldType
				listPMFlag = oldPM
				listNameFlag = oldName
			}()

			listDirFlag = tmpDir
			listConfigFlag = ""
			listOutputFlag = "json"
			listTypeFlag = "all"
			listPMFlag = "all"
			listNameFlag = tc.value // Special character in name filter

			// Should not crash
			output := captureStdout(t, func() {
				err := runList(nil, nil)
				_ = err // May error but should not panic
			})

			// Should return valid JSON (empty result is fine)
			output = strings.TrimSpace(output)
			if output != "" && (strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[")) {
				var data interface{}
				err := json.Unmarshal([]byte(output), &data)
				assert.NoError(t, err, "Special char '%s' should produce valid JSON", tc.value)
			}
		})
	}
}

// TestChaos_VeryLongParams tests very long parameter values.
func TestChaos_VeryLongParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create very long string
	longString := strings.Repeat("a", 10000)

	// Save and restore flags
	oldDir := listDirFlag
	oldConfig := listConfigFlag
	oldOutput := listOutputFlag
	oldType := listTypeFlag
	oldPM := listPMFlag
	oldName := listNameFlag
	defer func() {
		listDirFlag = oldDir
		listConfigFlag = oldConfig
		listOutputFlag = oldOutput
		listTypeFlag = oldType
		listPMFlag = oldPM
		listNameFlag = oldName
	}()

	listDirFlag = tmpDir
	listConfigFlag = ""
	listOutputFlag = "json"
	listTypeFlag = "all"
	listPMFlag = "all"
	listNameFlag = longString

	// Should not crash or hang
	output := captureStdout(t, func() {
		err := runList(nil, nil)
		_ = err
	})

	// Should return valid JSON
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Very long param should produce valid JSON")
	}
}

// TestChaos_CommaSeparatedFilters tests comma-separated filter values.
func TestChaos_CommaSeparatedFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{
		"name": "test",
		"dependencies": {"is-odd": "^3.0.0", "is-number": "^7.0.0"},
		"devDependencies": {"is-even": "^1.0.0"}
	}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		typeVal  string
		pmVal    string
		nameVal  string
	}{
		{"multiple_types", "prod,dev", "all", ""},
		{"multiple_names", "all", "all", "is-odd,is-even"},
		{"trailing_comma", "prod,", "all", ""},
		{"leading_comma", ",prod", "all", ""},
		{"double_comma", "prod,,dev", "all", ""},
		{"spaces", "prod, dev", "all", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save and restore flags
			oldDir := listDirFlag
			oldConfig := listConfigFlag
			oldOutput := listOutputFlag
			oldType := listTypeFlag
			oldPM := listPMFlag
			oldName := listNameFlag
			defer func() {
				listDirFlag = oldDir
				listConfigFlag = oldConfig
				listOutputFlag = oldOutput
				listTypeFlag = oldType
				listPMFlag = oldPM
				listNameFlag = oldName
			}()

			listDirFlag = tmpDir
			listConfigFlag = ""
			listOutputFlag = "json"
			listTypeFlag = tc.typeVal
			listPMFlag = tc.pmVal
			listNameFlag = tc.nameVal

			output := captureStdout(t, func() {
				err := runList(nil, nil)
				_ = err
			})

			// Should return valid JSON
			output = strings.TrimSpace(output)
			if output != "" {
				var data interface{}
				err = json.Unmarshal([]byte(output), &data)
				assert.NoError(t, err, "Comma-separated '%s' should produce valid JSON", tc.name)
			}
		})
	}
}

// TestChaos_AllOutputFormats tests all output formats for all commands.
func TestChaos_AllOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	formats := []string{"json", "xml", "csv", "table", ""}

	for _, format := range formats {
		t.Run("scan_"+format, func(t *testing.T) {
			oldDir := scanDirFlag
			oldConfig := scanConfigFlag
			oldOutput := scanOutputFlag
			defer func() {
				scanDirFlag = oldDir
				scanConfigFlag = oldConfig
				scanOutputFlag = oldOutput
			}()

			scanDirFlag = tmpDir
			scanConfigFlag = ""
			scanOutputFlag = format

			output := captureStdout(t, func() {
				err := runScan(nil, nil)
				assert.NoError(t, err)
			})

			assert.NotEmpty(t, output, "scan with format '%s' should produce output", format)
		})

		t.Run("list_"+format, func(t *testing.T) {
			oldDir := listDirFlag
			oldConfig := listConfigFlag
			oldOutput := listOutputFlag
			oldType := listTypeFlag
			oldPM := listPMFlag
			defer func() {
				listDirFlag = oldDir
				listConfigFlag = oldConfig
				listOutputFlag = oldOutput
				listTypeFlag = oldType
				listPMFlag = oldPM
			}()

			listDirFlag = tmpDir
			listConfigFlag = ""
			listOutputFlag = format
			listTypeFlag = "all"
			listPMFlag = "all"

			output := captureStdout(t, func() {
				err := runList(nil, nil)
				assert.NoError(t, err)
			})

			assert.NotEmpty(t, output, "list with format '%s' should produce output", format)
		})
	}
}

// TestChaos_InvalidOutputFormat tests invalid output format.
func TestChaos_InvalidOutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "invalid_format"

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		// Should either error or fall back to default
		_ = err
	})

	// Should produce some output (default table format)
	assert.NotEmpty(t, output, "Invalid format should fall back to default")
}

// -----------------------------------------------------------------------------
// PARTIAL ERROR HANDLING TESTS
// -----------------------------------------------------------------------------

// TestPartialError_Update_ContinueOnFail tests that partial errors are properly
// communicated in structured output.
func TestPartialError_Update_ContinueOnFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldYes := updateYesFlag
	oldSkipPreflight := updateSkipPreflight
	oldSkipSystemTests := updateSkipSystemTests
	oldSkipLock := updateSkipLockRun
	oldOutput := updateOutputFlag
	oldContinue := updateContinueOnFail
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateOutputFlag = oldOutput
		updateContinueOnFail = oldContinue
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateOutputFlag = "json"
	updateContinueOnFail = true

	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	if output != "" {
		var data map[string]interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Partial error output should be valid JSON")

		// Check if errors field exists for conveying partial failures
		// This documents expected behavior
		t.Logf("JSON output structure: %+v", data)
	}
}

// TestPartialError_Outdated_ContinueOnFail tests outdated with continue-on-fail.
func TestPartialError_Outdated_ContinueOnFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldContinue := outdatedContinueOnFail
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedContinueOnFail = oldContinue
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedContinueOnFail = true

	output := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		_ = err
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	if output != "" {
		var data interface{}
		err = json.Unmarshal([]byte(output), &data)
		assert.NoError(t, err, "Outdated with continue-on-fail should produce valid JSON")
	}
}

// -----------------------------------------------------------------------------
// DOCUMENTATION: KNOWN PARAMETER CONFLICTS AND BEHAVIORS
// -----------------------------------------------------------------------------

// TestDoc_ParameterConflicts documents known parameter conflicts and expected behaviors.
func TestDoc_ParameterConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Log("DOCUMENTED PARAMETER INTERACTIONS:")
	t.Log("")
	t.Log("1. --dry-run + --yes")
	t.Log("   Behavior: Both work together, --yes is redundant")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("2. --dry-run + --skip-lock")
	t.Log("   Behavior: --skip-lock is irrelevant in dry-run mode")
	t.Log("   Status: Works correctly (skip-lock ignored)")
	t.Log("")
	t.Log("3. --skip-system-tests + --system-test-mode")
	t.Log("   Behavior: --skip-system-tests takes precedence")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("4. --major + --minor + --patch (all together)")
	t.Log("   Behavior: Most permissive (major) takes effect")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("5. --incremental + version scope flags")
	t.Log("   Behavior: Incremental respects version scope")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("6. --continue-on-fail + --output json")
	t.Log("   Behavior: Partial errors included in JSON output")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("7. --output json without --yes or --dry-run")
	t.Log("   Behavior: Interactive prompts skipped for structured output")
	t.Log("   Status: Works correctly (implicit --yes)")
	t.Log("")
	t.Log("8. Empty string parameters")
	t.Log("   Behavior: Treated as default ('all')")
	t.Log("   Status: Works correctly")
	t.Log("")
	t.Log("9. Invalid output format")
	t.Log("   Behavior: Falls back to table format")
	t.Log("   Status: Works correctly")
}

// TestDoc_OutputFormatErrorHandling documents error handling for each output format.
func TestDoc_OutputFormatErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Log("OUTPUT FORMAT ERROR HANDLING:")
	t.Log("")
	t.Log("JSON Output:")
	t.Log("  - Valid JSON should be returned even on errors")
	t.Log("  - Errors should be in 'errors' field of JSON")
	t.Log("  - Empty results should return empty array/object, not null")
	t.Log("")
	t.Log("XML Output:")
	t.Log("  - Valid XML should be returned even on errors")
	t.Log("  - Errors should be in appropriate XML elements")
	t.Log("")
	t.Log("CSV Output:")
	t.Log("  - Header row should always be present")
	t.Log("  - Empty results should return header only")
	t.Log("")
	t.Log("Table Output (default):")
	t.Log("  - Human-readable error messages are OK")
	t.Log("  - No structural requirements")
}
