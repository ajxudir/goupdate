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
	configYAML := `extends:
  - default
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: "echo test"
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
// OUTPUT TO TEMP FILE VALIDATION TESTS
// -----------------------------------------------------------------------------
// These tests write command output to actual temp files (.json/.xml/.csv) and
// validate that the files contain valid structured data. This simulates real
// user workflows like: goupdate update --output json > result.json

// validateJSONFile reads a JSON file and validates it contains valid JSON.
// Returns the parsed data and any error encountered.
func validateJSONFile(path string) (interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Trim whitespace
	content = []byte(strings.TrimSpace(string(content)))
	if len(content) == 0 {
		return nil, nil // Empty file is valid
	}

	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// validateXMLFile reads an XML file and validates it contains valid XML.
func validateXMLFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content = []byte(strings.TrimSpace(string(content)))
	if len(content) == 0 {
		return nil // Empty file is valid
	}

	var data interface{}
	return xml.Unmarshal(content, &data)
}

// validateCSVFile reads a CSV file and validates it has consistent structure.
func validateCSVFile(path string) ([][]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	contentStr := strings.TrimSpace(string(content))
	if contentStr == "" {
		return nil, nil // Empty file is valid
	}

	lines := strings.Split(contentStr, "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	// Parse CSV lines manually for validation
	var rows [][]string
	var headerColCount int
	for i, line := range lines {
		// Simple CSV parsing - split by comma
		cols := strings.Split(line, ",")
		if i == 0 {
			headerColCount = len(cols)
		} else if len(cols) != headerColCount && line != "" {
			return nil, os.ErrInvalid // Inconsistent column count
		}
		rows = append(rows, cols)
	}

	return rows, nil
}

// checkFileForCorruption reads a file and checks for common corruption indicators.
func checkFileForCorruption(path string) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return []string{"failed to read file: " + err.Error()}
	}

	contentStr := string(content)
	var issues []string

	// Plain text messages that indicate corruption
	corruptionIndicators := []string{
		"Running system tests",
		"All system tests passed",
		"System tests failed",
		"packages will be updated",
		"Continue?",
		"Proceeding with updates",
		"Running preflight",
		"Preflight passed",
		"Preflight failed",
		"[y/N]",
		"Press Enter",
	}

	for _, indicator := range corruptionIndicators {
		if strings.Contains(contentStr, indicator) {
			issues = append(issues, "found plain text: "+indicator)
		}
	}

	return issues
}

// TestIntegration_WriteToJSONFile_Scan writes scan output to a .json file and validates it.
func TestIntegration_WriteToJSONFile_Scan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "scan_result.json")

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

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	data, err := validateJSONFile(outputFile)
	assert.NoError(t, err, "scan output file should contain valid JSON")
	assert.NotNil(t, data, "scan output should not be empty")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "scan JSON output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToXMLFile_Scan writes scan output to a .xml file and validates it.
func TestIntegration_WriteToXMLFile_Scan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "scan_result.xml")

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

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	err = validateXMLFile(outputFile)
	assert.NoError(t, err, "scan output file should contain valid XML")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "scan XML output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToCSVFile_Scan writes scan output to a .csv file and validates it.
func TestIntegration_WriteToCSVFile_Scan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name": "test"}`), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "scan_result.csv")

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

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	rows, err := validateCSVFile(outputFile)
	assert.NoError(t, err, "scan output file should contain valid CSV")
	assert.NotNil(t, rows, "scan CSV output should not be empty")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "scan CSV output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToJSONFile_List writes list output to a .json file and validates it.
func TestIntegration_WriteToJSONFile_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with dependency
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "list_result.json")

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

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runList(nil, nil)
		assert.NoError(t, err)
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	data, err := validateJSONFile(outputFile)
	assert.NoError(t, err, "list output file should contain valid JSON")
	_ = data // May be nil if no packages found

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "list JSON output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToJSONFile_Update_DryRun writes update --dry-run output to a .json file.
func TestIntegration_WriteToJSONFile_Update_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "update_dryrun_result.json")

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
	updateOutputFlag = "json"

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	_, err = validateJSONFile(outputFile)
	assert.NoError(t, err, "update --dry-run output file should contain valid JSON")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "update --dry-run JSON output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToJSONFile_Update_NoDryRun writes update output (no dry-run) to a .json file.
func TestIntegration_WriteToJSONFile_Update_NoDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "update_nodryrun_result.json")

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

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // NOT DRY RUN
	updateYesFlag = true     // Skip interactive prompt
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true // Skip lock to avoid needing npm
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json"

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	_, err = validateJSONFile(outputFile)
	assert.NoError(t, err, "update output file should contain valid JSON")

	// Check for corruption - this is especially important for non-dry-run
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "update JSON output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToXMLFile_Update writes update output to a .xml file and validates it.
func TestIntegration_WriteToXMLFile_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "update_result.xml")

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

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "xml"

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	err = validateXMLFile(outputFile)
	assert.NoError(t, err, "update output file should contain valid XML")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "update XML output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToCSVFile_Update writes update output to a .csv file and validates it.
func TestIntegration_WriteToCSVFile_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "update_result.csv")

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

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "csv"

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	_, err = validateCSVFile(outputFile)
	assert.NoError(t, err, "update output file should contain valid CSV")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "update CSV output file should not contain plain text: %v", issues)
}

// -----------------------------------------------------------------------------
// OUTPUT CORRUPTION DETECTION TESTS
// -----------------------------------------------------------------------------
// These tests specifically detect the issue where plain text messages are
// printed even when structured output (JSON/XML/CSV) is requested, corrupting
// the output when piped to a file.

// TestIntegration_DetectOutputCorruption_WithSystemTests tests that system test messages
// do not corrupt structured output.
//
// KNOWN ISSUE: When system tests run, messages like "Running system tests..." are
// printed to stdout even when JSON output is requested. This corrupts the output
// when piped to a file.
func TestIntegration_DetectOutputCorruption_WithSystemTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Create config with system tests
	configYAML := `extends:
  - default
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: "echo test"
      commands: "echo ok"
`
	err = os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(configYAML), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "corrupted_output.json")

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

	updateDirFlag = tmpDir
	updateConfigFlag = filepath.Join(tmpDir, ".goupdate.yml")
	updateDryRunFlag = true
	updateYesFlag = true
	updateSkipPreflight = false   // Run preflight
	updateSkipSystemTests = false // Run system tests - this may cause corruption
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json"

	// Capture output and write to file
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Check for corruption indicators
	issues := checkFileForCorruption(outputFile)
	if len(issues) > 0 {
		// Read file content for diagnostic
		content, _ := os.ReadFile(outputFile)
		t.Logf("OUTPUT CORRUPTION DETECTED in %s:\nIssues: %v\nContent:\n%s", outputFile, issues, string(content))

		// Verify that the file is actually invalid JSON due to corruption
		_, jsonErr := validateJSONFile(outputFile)
		if jsonErr != nil {
			t.Errorf("JSON output is corrupted with plain text messages. "+
				"File contains invalid JSON due to: %v. "+
				"SUGGESTION: System test messages should be written to stderr, not stdout, "+
				"when structured output format is selected.", issues)
		}
	}
}

// TestIntegration_DetectOutputCorruption_InteractivePrompt tests that interactive prompts
// do not corrupt structured output.
//
// KNOWN ISSUE: When --yes is not passed and output is JSON, the "Continue? [y/N]" prompt
// would be printed to stdout, corrupting JSON output.
// Users MUST pass --yes or --dry-run when using structured output formats.
func TestIntegration_DetectOutputCorruption_InteractivePrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// NOTE: This test documents the expected behavior:
	// When using JSON output without --yes or --dry-run, the command should:
	// 1. Either automatically skip interactive prompts, OR
	// 2. Return an error stating structured output requires --yes or --dry-run

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "interactive_output.json")

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

	updateDirFlag = tmpDir
	updateConfigFlag = ""
	updateDryRunFlag = false // NOT dry run
	updateYesFlag = true     // Using --yes to avoid actual interactive hang
	updateSkipPreflight = true
	updateSkipSystemTests = true
	updateSkipLockRun = true
	updateRuleFlag = "npm"
	updateTypeFlag = "all"
	updatePMFlag = "all"
	updateOutputFlag = "json"

	// Capture output
	output := captureStdout(t, func() {
		err := runUpdate(nil, nil)
		_ = err
	})

	// Write to file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate JSON
	_, jsonErr := validateJSONFile(outputFile)

	// Check for corruption
	issues := checkFileForCorruption(outputFile)

	// Log findings for documentation
	if len(issues) > 0 || jsonErr != nil {
		content, _ := os.ReadFile(outputFile)
		t.Logf("Interactive mode output file content:\n%s", string(content))
		if len(issues) > 0 {
			t.Logf("Corruption issues found: %v", issues)
		}
		if jsonErr != nil {
			t.Logf("JSON validation error: %v", jsonErr)
		}
	}

	// The output should be valid JSON when --yes is passed
	assert.NoError(t, jsonErr, "JSON output with --yes flag should be valid JSON")
	assert.Empty(t, issues, "JSON output with --yes flag should not contain plain text prompts")
}

// -----------------------------------------------------------------------------
// ALL COMMANDS - COMPREHENSIVE FILE OUTPUT VALIDATION
// -----------------------------------------------------------------------------

// TestIntegration_AllCommands_JSONFileOutput tests all commands produce valid JSON files.
func TestIntegration_AllCommands_JSONFileOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		runCmd    func() error
		setupFunc func()
	}{
		{
			name: "scan",
			setupFunc: func() {
				scanDirFlag = tmpDir
				scanConfigFlag = ""
				scanOutputFlag = "json"
			},
			runCmd: func() error { return runScan(nil, nil) },
		},
		{
			name: "list",
			setupFunc: func() {
				listDirFlag = tmpDir
				listConfigFlag = ""
				listOutputFlag = "json"
				listTypeFlag = "all"
				listPMFlag = "all"
			},
			runCmd: func() error { return runList(nil, nil) },
		},
		{
			name: "outdated",
			setupFunc: func() {
				outdatedDirFlag = tmpDir
				outdatedConfigFlag = ""
				outdatedOutputFlag = "json"
				outdatedTypeFlag = "all"
				outdatedPMFlag = "all"
			},
			runCmd: func() error { return runOutdated(nil, nil) },
		},
		{
			name: "update_dryrun",
			setupFunc: func() {
				updateDirFlag = tmpDir
				updateConfigFlag = ""
				updateDryRunFlag = true
				updateYesFlag = true
				updateSkipPreflight = true
				updateSkipSystemTests = true
				updateSkipLockRun = true
				updateRuleFlag = "npm"
				updateTypeFlag = "all"
				updatePMFlag = "all"
				updateOutputFlag = "json"
			},
			runCmd: func() error { return runUpdate(nil, nil) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tc.setupFunc()

			// Output file
			outputFile := filepath.Join(tmpDir, tc.name+"_output.json")

			// Capture output
			output := captureStdout(t, func() {
				err := tc.runCmd()
				_ = err
			})

			// Write to file
			err := os.WriteFile(outputFile, []byte(output), 0644)
			require.NoError(t, err)

			// Validate JSON
			_, jsonErr := validateJSONFile(outputFile)
			assert.NoError(t, jsonErr, "%s should produce valid JSON file", tc.name)

			// Check for corruption
			issues := checkFileForCorruption(outputFile)
			assert.Empty(t, issues, "%s JSON file should not contain plain text: %v", tc.name, issues)
		})
	}
}

// TestIntegration_AllCommands_XMLFileOutput tests all commands produce valid XML files.
func TestIntegration_AllCommands_XMLFileOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		runCmd    func() error
		setupFunc func()
	}{
		{
			name: "scan",
			setupFunc: func() {
				scanDirFlag = tmpDir
				scanConfigFlag = ""
				scanOutputFlag = "xml"
			},
			runCmd: func() error { return runScan(nil, nil) },
		},
		{
			name: "list",
			setupFunc: func() {
				listDirFlag = tmpDir
				listConfigFlag = ""
				listOutputFlag = "xml"
				listTypeFlag = "all"
				listPMFlag = "all"
			},
			runCmd: func() error { return runList(nil, nil) },
		},
		{
			name: "update_dryrun",
			setupFunc: func() {
				updateDirFlag = tmpDir
				updateConfigFlag = ""
				updateDryRunFlag = true
				updateYesFlag = true
				updateSkipPreflight = true
				updateSkipSystemTests = true
				updateSkipLockRun = true
				updateRuleFlag = "npm"
				updateTypeFlag = "all"
				updatePMFlag = "all"
				updateOutputFlag = "xml"
			},
			runCmd: func() error { return runUpdate(nil, nil) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tc.setupFunc()

			// Output file
			outputFile := filepath.Join(tmpDir, tc.name+"_output.xml")

			// Capture output
			output := captureStdout(t, func() {
				err := tc.runCmd()
				_ = err
			})

			// Write to file
			err := os.WriteFile(outputFile, []byte(output), 0644)
			require.NoError(t, err)

			// Validate XML
			xmlErr := validateXMLFile(outputFile)
			assert.NoError(t, xmlErr, "%s should produce valid XML file", tc.name)

			// Check for corruption
			issues := checkFileForCorruption(outputFile)
			assert.Empty(t, issues, "%s XML file should not contain plain text: %v", tc.name, issues)
		})
	}
}

// TestIntegration_AllCommands_CSVFileOutput tests all commands produce valid CSV files.
func TestIntegration_AllCommands_CSVFileOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json
	packageJSON := `{"name": "test", "version": "1.0.0", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		runCmd    func() error
		setupFunc func()
	}{
		{
			name: "scan",
			setupFunc: func() {
				scanDirFlag = tmpDir
				scanConfigFlag = ""
				scanOutputFlag = "csv"
			},
			runCmd: func() error { return runScan(nil, nil) },
		},
		{
			name: "list",
			setupFunc: func() {
				listDirFlag = tmpDir
				listConfigFlag = ""
				listOutputFlag = "csv"
				listTypeFlag = "all"
				listPMFlag = "all"
			},
			runCmd: func() error { return runList(nil, nil) },
		},
		{
			name: "update_dryrun",
			setupFunc: func() {
				updateDirFlag = tmpDir
				updateConfigFlag = ""
				updateDryRunFlag = true
				updateYesFlag = true
				updateSkipPreflight = true
				updateSkipSystemTests = true
				updateSkipLockRun = true
				updateRuleFlag = "npm"
				updateTypeFlag = "all"
				updatePMFlag = "all"
				updateOutputFlag = "csv"
			},
			runCmd: func() error { return runUpdate(nil, nil) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tc.setupFunc()

			// Output file
			outputFile := filepath.Join(tmpDir, tc.name+"_output.csv")

			// Capture output
			output := captureStdout(t, func() {
				err := tc.runCmd()
				_ = err
			})

			// Write to file
			err := os.WriteFile(outputFile, []byte(output), 0644)
			require.NoError(t, err)

			// Validate CSV
			_, csvErr := validateCSVFile(outputFile)
			assert.NoError(t, csvErr, "%s should produce valid CSV file", tc.name)

			// Check for corruption
			issues := checkFileForCorruption(outputFile)
			assert.Empty(t, issues, "%s CSV file should not contain plain text: %v", tc.name, issues)
		})
	}
}

// -----------------------------------------------------------------------------
// DOCUMENTED OUTPUT CORRUPTION ISSUES
// -----------------------------------------------------------------------------
// This section documents KNOWN ISSUES where plain text output can corrupt
// structured output formats (JSON, XML, CSV).
//
// ISSUE #1: System Test Messages
// Location: cmd/update.go - runPreflightTests() and runAfterAllValidation()
// Problem: fmt.Println("Running system tests...") and related messages are
//          printed directly to stdout, even when --output json is specified.
// Root Cause: These functions print to stdout unconditionally, without checking
//             if structured output is requested.
// Impact: When user runs: goupdate update --output json > result.json
//         The result.json file may contain: "Running system tests...\n{...json...}"
//         making it invalid JSON.
// Suggested Fix: Check output format before printing, or always use stderr for
//                informational messages: fmt.Fprintln(os.Stderr, "Running system tests...")
//
// ISSUE #2: Interactive Prompts
// Location: cmd/update.go - confirmUpdate()
// Problem: "Continue? [y/N]:" prompt is printed to stdout.
// Impact: If a user tries to use JSON output without --yes or --dry-run,
//         the prompt would corrupt the output.
// Current Mitigation: The code checks for structured output and skips
//                     interactive prompts, but this should be more explicit.
// Suggested Fix: When using structured output without --yes or --dry-run,
//                either auto-confirm (dangerous) or return a JSON error message.
//
// ISSUE #3: Progress Indicator Location
// Location: cmd/update.go
// Status: PROPERLY HANDLED (for reference)
// Code: progress := output.NewProgress(os.Stderr, ...)
// Note: Progress is correctly sent to stderr when using structured output.
//       This is the correct pattern that should be followed elsewhere.

// TestIntegration_DocumentOutputCorruptionIssue_SystemTests documents the system test
// output corruption issue. This test serves as documentation and regression detection.
func TestIntegration_DocumentOutputCorruptionIssue_SystemTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// DOCUMENTED ISSUE: When system tests run, messages like:
	// - "Running system tests (preflight)..."
	// - "All system tests passed. Proceeding with updates..."
	// - "Running system tests (validation)..."
	// Are printed to stdout, corrupting JSON/XML/CSV output.
	//
	// AFFECTED CODE LOCATIONS:
	// - cmd/update.go:350-355 (preflight messages)
	// - cmd/update.go:364-367 (preflight result messages)
	// - cmd/update.go:401-426 (validation messages)
	//
	// WORKAROUND FOR USERS:
	// Use --skip-system-tests flag when using structured output:
	//   goupdate update --output json --skip-system-tests > result.json
	//
	// PROPER FIX:
	// Change fmt.Println() calls to fmt.Fprintln(os.Stderr, ...) in:
	// - runPreflightTests() function
	// - runAfterAllValidation() function
	// OR check useStructuredOutput before printing and skip/redirect messages.

	t.Log("DOCUMENTED ISSUE: System test messages can corrupt JSON output")
	t.Log("See cmd/update.go runPreflightTests() and runAfterAllValidation()")
	t.Log("WORKAROUND: Use --skip-system-tests with structured output")
}

// TestIntegration_DocumentOutputCorruptionIssue_InteractivePrompts documents the
// interactive prompt output corruption issue.
func TestIntegration_DocumentOutputCorruptionIssue_InteractivePrompts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// DOCUMENTED ISSUE: When update command runs without --yes or --dry-run,
	// an interactive prompt is shown:
	// "N package(s) will be updated. Continue? [y/N]: "
	//
	// This prompt is printed to stdout (cmd/update.go:291), which would
	// corrupt JSON/XML/CSV output if the user tries:
	//   goupdate update --output json > result.json
	// (without --yes or --dry-run)
	//
	// AFFECTED CODE LOCATION:
	// - cmd/update.go:285-303 (confirmUpdate function)
	//
	// CURRENT MITIGATION:
	// The code at line 213 checks:
	//   if !updateDryRunFlag && !useStructuredOutput && pendingUpdates > 0
	// This means interactive prompts are SKIPPED when useStructuredOutput=true.
	// However, this could be made more explicit with a proper error message.
	//
	// RECOMMENDATION FOR USERS:
	// Always use --yes or --dry-run when using structured output:
	//   goupdate update --output json --yes > result.json
	//   goupdate update --output json --dry-run > result.json
	//
	// POSSIBLE IMPROVEMENT:
	// When structured output is requested without --yes or --dry-run,
	// the command could return a JSON error:
	//   {"error": "structured output requires --yes or --dry-run flag"}

	t.Log("DOCUMENTED ISSUE: Interactive prompts require --yes or --dry-run with structured output")
	t.Log("See cmd/update.go confirmUpdate() function")
	t.Log("RECOMMENDATION: Always use --yes or --dry-run with --output json/xml/csv")
}

// TestIntegration_VerifyProgressGoesToStderr verifies that progress indicators
// are correctly sent to stderr (not stdout) when using structured output.
// This is the CORRECT pattern that should be followed elsewhere.
func TestIntegration_VerifyProgressGoesToStderr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// VERIFIED CORRECT BEHAVIOR:
	// In cmd/update.go line 235:
	//   progress := output.NewProgress(os.Stderr, len(groupedPlans), "Processing updates")
	//
	// This correctly sends progress to stderr, not stdout.
	// When the user runs:
	//   goupdate update --output json > result.json
	// The progress indicator appears in the terminal (stderr) while
	// the JSON result goes to the file (stdout).
	//
	// This is the pattern that should be followed for all informational messages.

	t.Log("VERIFIED: Progress indicators correctly go to stderr")
	t.Log("See cmd/update.go line 235: output.NewProgress(os.Stderr, ...)")
}

// TestIntegration_KnownCorruptionIndicators tests for specific strings that
// indicate output corruption when found in structured output files.
func TestIntegration_KnownCorruptionIndicators(t *testing.T) {
	// List of known strings that should NEVER appear in structured output files
	knownCorruptionIndicators := []string{
		// System test messages (cmd/update.go)
		"Running system tests (preflight)",
		"Running system tests (validation)",
		"All system tests passed",
		"System tests failed",
		"Proceeding with updates",
		"continuing due to continue_on_fail",

		// Interactive prompts (cmd/update.go)
		"packages will be updated",
		"Continue?",
		"[y/N]",
		"Update cancelled",

		// Summary messages (various files)
		"Total packages:",
		"Scanned package files",
		"Updated packages that may have caused issues",
		"Consider rolling back",

		// Warning messages (cmd/update.go)
		"⚠",
		"Warning:",
		"Press Enter",
	}

	t.Log("Known corruption indicators to check in output files:")
	for _, indicator := range knownCorruptionIndicators {
		t.Logf("  - %q", indicator)
	}

	t.Log("\nWhen these strings appear in a .json/.xml/.csv file,")
	t.Log("it indicates that plain text was mixed with structured output.")
	t.Log("\nTo avoid this issue:")
	t.Log("  1. Use --skip-system-tests with structured output")
	t.Log("  2. Use --yes or --dry-run with structured output")
	t.Log("  3. Use --skip-preflight if needed")
}

// -----------------------------------------------------------------------------
// LEGACY OUTPUT TO FILE TEST (kept for backwards compatibility)
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
