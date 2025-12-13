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
// OUTPUT FORMAT INTEGRATION TESTS
// =============================================================================
//
// These tests verify that structured output formats (JSON, CSV, XML) produce
// valid, non-corrupted output. They detect the issue where non-structured
// content (like "Running system tests...") is mixed with structured output.
//
// Each command is tested with all output formats to ensure:
// - JSON output is valid JSON
// - XML output is valid XML
// - CSV output has consistent columns
// - No plain text is mixed into structured output
// =============================================================================

// -----------------------------------------------------------------------------
// UPDATE COMMAND - DRY-RUN VS NON-DRY-RUN TESTS
// -----------------------------------------------------------------------------

// TestIntegration_Update_DryRun_JSON tests update command with --dry-run and JSON output.
//
// It verifies:
//   - JSON output is valid when using --dry-run
//   - No plain text messages corrupt the JSON
func TestIntegration_Update_DryRun_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{
  "name": "test-update-dry-run-json",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0"
  }
}`
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
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updateOutputFlag = oldOutput
	}()

	// Configure for dry-run with JSON output
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true // DRY RUN
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json" // JSON OUTPUT

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	if output != "" {
		var jsonData interface{}
		err = json.Unmarshal([]byte(output), &jsonData)
		assert.NoError(t, err, "JSON output should be valid JSON: %s", output)

		// Verify no plain text messages in output
		assert.False(t, strings.Contains(output, "Running system tests"),
			"JSON output should not contain plain text messages")
		assert.False(t, strings.Contains(output, "packages will be updated"),
			"JSON output should not contain plain text prompts")
	}
}

// TestIntegration_Update_NoDryRun_JSON tests update command without --dry-run and JSON output.
//
// It verifies:
//   - JSON output is valid when NOT using --dry-run
//   - No plain text messages corrupt the JSON during actual updates
func TestIntegration_Update_NoDryRun_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{
  "name": "test-update-no-dry-run-json",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0"
  }
}`
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
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updateOutputFlag = oldOutput
	}()

	// Configure for NON-dry-run with JSON output
	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // NOT DRY RUN
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true // Skip lock to avoid needing npm
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json" // JSON OUTPUT

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	if output != "" {
		var jsonData interface{}
		err = json.Unmarshal([]byte(output), &jsonData)
		assert.NoError(t, err, "JSON output should be valid JSON: %s", output)

		// Verify no plain text messages in output
		assert.False(t, strings.Contains(output, "Running system tests"),
			"JSON output should not contain plain text messages")
		assert.False(t, strings.Contains(output, "packages will be updated"),
			"JSON output should not contain plain text prompts")
		assert.False(t, strings.Contains(output, "Continue?"),
			"JSON output should not contain interactive prompts")
	}
}

// -----------------------------------------------------------------------------
// SCAN COMMAND OUTPUT FORMAT TESTS
// -----------------------------------------------------------------------------

// TestIntegration_Scan_JSON tests scan command with JSON output.
func TestIntegration_Scan_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
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
	scanOutputFlag = "json"

	// Capture output
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	var jsonData interface{}
	err = json.Unmarshal([]byte(output), &jsonData)
	assert.NoError(t, err, "scan JSON output should be valid: %s", output)

	// Verify no plain text
	assert.False(t, strings.Contains(output, "Scanned package files"),
		"JSON output should not contain plain text header")
}

// TestIntegration_Scan_XML tests scan command with XML output.
func TestIntegration_Scan_XML(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
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

	// Capture output
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Verify XML is valid
	output = strings.TrimSpace(output)
	var xmlData interface{}
	err = xml.Unmarshal([]byte(output), &xmlData)
	assert.NoError(t, err, "scan XML output should be valid: %s", output)
}

// TestIntegration_Scan_CSV tests scan command with CSV output.
func TestIntegration_Scan_CSV(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
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

	// Capture output
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Verify CSV format - should have header row
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.GreaterOrEqual(t, len(lines), 1, "CSV should have at least header row")

	// Check header has expected columns
	header := lines[0]
	assert.Contains(t, header, "RULE", "CSV header should contain RULE column")

	// Verify no plain text
	assert.False(t, strings.Contains(output, "Scanned package files"),
		"CSV output should not contain plain text")
}

// -----------------------------------------------------------------------------
// LIST COMMAND OUTPUT FORMAT TESTS
// -----------------------------------------------------------------------------

// TestIntegration_List_JSON tests list command with JSON output.
func TestIntegration_List_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with a dependency
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
	listTypeFlag = "all"
	listPMFlag = "all"

	// Capture output
	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Verify JSON is valid
	output = strings.TrimSpace(output)
	if output != "" && output != "No packages found" {
		var jsonData interface{}
		err = json.Unmarshal([]byte(output), &jsonData)
		assert.NoError(t, err, "list JSON output should be valid: %s", output)
	}
}

// TestIntegration_List_XML tests list command with XML output.
func TestIntegration_List_XML(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with a dependency
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
	listOutputFlag = "xml"
	listTypeFlag = "all"
	listPMFlag = "all"

	// Capture output
	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Verify XML is valid (if not empty or "No packages found")
	output = strings.TrimSpace(output)
	if output != "" && !strings.Contains(output, "No packages found") {
		var xmlData interface{}
		err = xml.Unmarshal([]byte(output), &xmlData)
		assert.NoError(t, err, "list XML output should be valid: %s", output)
	}
}

// -----------------------------------------------------------------------------
// OUTDATED COMMAND OUTPUT FORMAT TESTS
// -----------------------------------------------------------------------------

// TestIntegration_Outdated_JSON tests outdated command with JSON output.
func TestIntegration_Outdated_JSON(t *testing.T) {
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
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"

	// Capture output
	output := captureStdout(t, func() {
		err := runOutdated(nil, nil)
		_ = err // May error if can't fetch versions
	})

	// Verify JSON is valid if there's output
	output = strings.TrimSpace(output)
	if output != "" && !strings.Contains(output, "No packages found") {
		var jsonData interface{}
		err = json.Unmarshal([]byte(output), &jsonData)
		assert.NoError(t, err, "outdated JSON output should be valid: %s", output)
	}
}

// -----------------------------------------------------------------------------
// UPDATE COMMAND WITH SYSTEM TESTS - OUTPUT FORMAT TESTS
// -----------------------------------------------------------------------------

// TestIntegration_Update_WithSystemTests_JSON tests that system test output doesn't corrupt JSON.
//
// This is a critical test that verifies the bug where "Running system tests..."
// messages are printed even when JSON output is requested.
func TestIntegration_Update_WithSystemTests_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{
  "name": "test-system-tests-json",
  "version": "1.0.0",
  "dependencies": {
    "is-odd": "^3.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create config with system tests
	configYAML := `extends: default
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: "echo test"
      command: "echo ok"
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
	oldSkipLock := updateSkipLockRun
	oldRule := updateRuleFlag
	oldType := updateTypeFlag
	oldPM := updatePMFlag
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateYesFlag = oldYes
		updateSkipPreflight = oldSkipPreflight
		updateSkipSystemTests = oldSkipSystemTests
		updateSkipLockRun = oldSkipLock
		updateRuleFlag = oldRule
		updateTypeFlag = oldType
		updatePMFlag = oldPM
		updateOutputFlag = oldOutput
	}()

	// Configure WITH system tests enabled and JSON output
	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = false   // Run preflight
	updateSkipSystemTests = false // Run system tests
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json" // JSON OUTPUT

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Check for output corruption indicators
	output = strings.TrimSpace(output)
	if output != "" {
		// These strings indicate plain text was mixed with JSON
		corruptionIndicators := []string{
			"Running system tests",
			"All system tests passed",
			"System tests failed",
			"packages will be updated",
			"Continue?",
			"Proceeding with updates",
		}

		for _, indicator := range corruptionIndicators {
			if strings.Contains(output, indicator) {
				// If the output contains plain text, it should NOT be valid JSON
				var jsonData interface{}
				jsonErr := json.Unmarshal([]byte(output), &jsonData)
				if jsonErr != nil {
					t.Errorf("JSON output is corrupted with plain text '%s': %s", indicator, output)
				}
			}
		}

		// Try to parse as JSON - this is the real test
		if strings.HasPrefix(output, "{") || strings.HasPrefix(output, "[") {
			var jsonData interface{}
			err = json.Unmarshal([]byte(output), &jsonData)
			assert.NoError(t, err, "JSON output should be valid JSON without plain text: %s", output)
		}
	}
}

// -----------------------------------------------------------------------------
// EMPTY RESULTS OUTPUT FORMAT TESTS
// -----------------------------------------------------------------------------

// TestIntegration_Update_NoPackages_JSON tests JSON output when no packages found.
func TestIntegration_Update_NoPackages_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Empty directory - no package files

	// Save and restore flags
	oldDir := updateDirFlag
	oldConfig := updateConfigFlag
	oldDryRun := updateDryRunFlag
	oldOutput := updateOutputFlag
	defer func() {
		updateDirFlag = oldDir
		updateConfigFlag = oldConfig
		updateDryRunFlag = oldDryRun
		updateOutputFlag = oldOutput
	}()

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateOutputFlag = "json"

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		assert.NoError(t, err)
	})

	// Should still be valid JSON (empty result)
	output = strings.TrimSpace(output)
	if output != "" {
		var jsonData interface{}
		err := json.Unmarshal([]byte(output), &jsonData)
		assert.NoError(t, err, "Empty result JSON should be valid: %s", output)

		// Should not contain plain text message
		assert.False(t, strings.Contains(output, "No packages found"),
			"JSON output should not contain plain text 'No packages found' message")
	}
}

// TestIntegration_Scan_NoFiles_JSON tests JSON output when no files found.
func TestIntegration_Scan_NoFiles_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

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

	// Capture output
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should be valid JSON even with no files
	output = strings.TrimSpace(output)
	var jsonData interface{}
	err := json.Unmarshal([]byte(output), &jsonData)
	assert.NoError(t, err, "Empty scan JSON should be valid: %s", output)

	// Should not contain plain text
	assert.False(t, strings.Contains(output, "No package files found"),
		"JSON output should not contain plain text message")
}

// -----------------------------------------------------------------------------
// OUTPUT TO FILE SIMULATION TESTS
// -----------------------------------------------------------------------------

// TestIntegration_JSONOutputToFile tests that JSON output can be written to a file.
//
// This simulates the real-world use case: goupdate update --output json > result.json
func TestIntegration_JSONOutputToFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
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
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateOutputFlag = "json"

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Simulate writing to file
	outputFile := filepath.Join(tmpDir, "result.json")
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Read back and verify it's valid JSON
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	content = []byte(strings.TrimSpace(string(content)))
	if len(content) > 0 {
		var jsonData interface{}
		err = json.Unmarshal(content, &jsonData)
		assert.NoError(t, err, "JSON file should contain valid JSON: %s", string(content))
	}
}
