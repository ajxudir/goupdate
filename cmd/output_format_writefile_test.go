package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// OUTPUT FORMAT INTEGRATION TESTS - WRITE TO FILE
// =============================================================================
//
// These tests verify that --output-file flag correctly writes structured output
// to files without corruption.
// =============================================================================

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

// TestIntegration_WriteToJSONFile_Outdated writes outdated output to a .json file and validates it.
//
// This test simulates: goupdate outdated -o json > outdated.json
// It verifies that:
//   - Only JSON goes to stdout (the file)
//   - Progress messages go to stderr (not captured)
//   - The resulting file contains valid JSON
func TestIntegration_WriteToJSONFile_Outdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with dependency
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "outdated_result.json")

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldSkipPreflight := outdatedSkipPreflight
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedSkipPreflight = oldSkipPreflight
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "json"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedSkipPreflight = true

	// Capture stdout and command error
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runOutdated(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runOutdated (JSON) returned error: %v", cmdErr)
	}

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	data, err := validateJSONFile(outputFile)
	assert.NoError(t, err, "outdated output file should contain valid JSON")
	_ = data // May be nil if no packages found

	// Check for corruption - particularly progress messages
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "outdated JSON output file should not contain plain text: %v", issues)

	// Specifically verify no progress messages in output
	content, _ := os.ReadFile(outputFile)
	assert.False(t, strings.Contains(string(content), "Checking packages"),
		"JSON output file should not contain progress messages - they should go to stderr")
}

// TestIntegration_WriteToXMLFile_Outdated writes outdated output to a .xml file and validates it.
func TestIntegration_WriteToXMLFile_Outdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with dependency
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "outdated_result.xml")

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldSkipPreflight := outdatedSkipPreflight
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedSkipPreflight = oldSkipPreflight
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "xml"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedSkipPreflight = true

	// Capture stdout and command error
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runOutdated(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runOutdated (XML) returned error: %v", cmdErr)
	}

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	err = validateXMLFile(outputFile)
	assert.NoError(t, err, "outdated output file should contain valid XML")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "outdated XML output file should not contain plain text: %v", issues)
}

// TestIntegration_WriteToCSVFile_Outdated writes outdated output to a .csv file and validates it.
func TestIntegration_WriteToCSVFile_Outdated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create package.json with dependency
	packageJSON := `{"name": "test", "dependencies": {"is-odd": "^3.0.0"}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	// Output file
	outputFile := filepath.Join(tmpDir, "outdated_result.csv")

	// Save and restore flags
	oldDir := outdatedDirFlag
	oldConfig := outdatedConfigFlag
	oldOutput := outdatedOutputFlag
	oldType := outdatedTypeFlag
	oldPM := outdatedPMFlag
	oldSkipPreflight := outdatedSkipPreflight
	defer func() {
		outdatedDirFlag = oldDir
		outdatedConfigFlag = oldConfig
		outdatedOutputFlag = oldOutput
		outdatedTypeFlag = oldType
		outdatedPMFlag = oldPM
		outdatedSkipPreflight = oldSkipPreflight
	}()

	outdatedDirFlag = tmpDir
	outdatedConfigFlag = ""
	outdatedOutputFlag = "csv"
	outdatedTypeFlag = "all"
	outdatedPMFlag = "all"
	outdatedSkipPreflight = true

	// Capture stdout and command error
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runOutdated(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runOutdated (CSV) returned error: %v", cmdErr)
	}

	// Write to actual file
	err = os.WriteFile(outputFile, []byte(output), 0644)
	require.NoError(t, err)

	// Validate the file
	rows, err := validateCSVFile(outputFile)
	assert.NoError(t, err, "outdated output file should contain valid CSV")
	assert.NotNil(t, rows, "outdated CSV output should not be empty")

	// Check for corruption
	issues := checkFileForCorruption(outputFile)
	assert.Empty(t, issues, "outdated CSV output file should not contain plain text: %v", issues)
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

	// Capture output and command error, then write to file
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runUpdate(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runUpdate (dry-run) returned error: %v", cmdErr)
	}

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

	// Capture output and command error, then write to file
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runUpdate(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runUpdate (no dry-run) returned error: %v", cmdErr)
	}

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

	// Capture output and command error, then write to file
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runUpdate(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runUpdate (XML output) returned error: %v", cmdErr)
	}

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

	// Capture output and command error, then write to file
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runUpdate(nil, nil)
	})

	// Log any command error for debugging
	if cmdErr != nil {
		t.Logf("runUpdate (CSV output) returned error: %v", cmdErr)
	}

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
