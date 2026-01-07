package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// OUTPUT FORMAT INTEGRATION TESTS - ALL COMMANDS
// =============================================================================
//
// These tests verify that all commands (scan, list, outdated, update) produce
// valid structured output in all formats (JSON, XML, CSV).
// =============================================================================

func TestIntegration_AllCommands_JSONFileOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Save all flags that will be modified
	oldScanDir, oldScanConfig, oldScanOutput := scanDirFlag, scanConfigFlag, scanOutputFlag
	oldListDir, oldListConfig, oldListOutput := listDirFlag, listConfigFlag, listOutputFlag
	oldListType, oldListPM := listTypeFlag, listPMFlag
	oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput := outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag
	oldOutdatedType, oldOutdatedPM := outdatedTypeFlag, outdatedPMFlag
	oldUpdateDir, oldUpdateConfig, oldUpdateOutput := updateDirFlag, updateConfigFlag, updateOutputFlag
	oldUpdateDryRun, oldUpdateYes := updateDryRunFlag, updateYesFlag
	oldUpdateSkipPreflight, oldUpdateSkipSystemTests := updateSkipPreflight, updateSkipSystemTests
	oldUpdateSkipLock, oldUpdateRule := updateSkipLockRun, updateRuleFlag
	oldUpdateType, oldUpdatePM := updateTypeFlag, updatePMFlag
	defer func() {
		scanDirFlag, scanConfigFlag, scanOutputFlag = oldScanDir, oldScanConfig, oldScanOutput
		listDirFlag, listConfigFlag, listOutputFlag = oldListDir, oldListConfig, oldListOutput
		listTypeFlag, listPMFlag = oldListType, oldListPM
		outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag = oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput
		outdatedTypeFlag, outdatedPMFlag = oldOutdatedType, oldOutdatedPM
		updateDirFlag, updateConfigFlag, updateOutputFlag = oldUpdateDir, oldUpdateConfig, oldUpdateOutput
		updateDryRunFlag, updateYesFlag = oldUpdateDryRun, oldUpdateYes
		updateSkipPreflight, updateSkipSystemTests = oldUpdateSkipPreflight, oldUpdateSkipSystemTests
		updateSkipLockRun, updateRuleFlag = oldUpdateSkipLock, oldUpdateRule
		updateTypeFlag, updatePMFlag = oldUpdateType, oldUpdatePM
	}()

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

			// Capture output and command error
			var cmdErr error
			output := captureStdout(t, func() {
				cmdErr = tc.runCmd()
			})

			// Log any command error for debugging
			if cmdErr != nil {
				t.Logf("%s command returned error: %v", tc.name, cmdErr)
			}

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

	// Save all flags that will be modified
	oldScanDir, oldScanConfig, oldScanOutput := scanDirFlag, scanConfigFlag, scanOutputFlag
	oldListDir, oldListConfig, oldListOutput := listDirFlag, listConfigFlag, listOutputFlag
	oldListType, oldListPM := listTypeFlag, listPMFlag
	oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput := outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag
	oldOutdatedType, oldOutdatedPM := outdatedTypeFlag, outdatedPMFlag
	oldOutdatedSkipPreflight := outdatedSkipPreflight
	oldUpdateDir, oldUpdateConfig, oldUpdateOutput := updateDirFlag, updateConfigFlag, updateOutputFlag
	oldUpdateDryRun, oldUpdateYes := updateDryRunFlag, updateYesFlag
	oldUpdateSkipPreflight, oldUpdateSkipSystemTests := updateSkipPreflight, updateSkipSystemTests
	oldUpdateSkipLock, oldUpdateRule := updateSkipLockRun, updateRuleFlag
	oldUpdateType, oldUpdatePM := updateTypeFlag, updatePMFlag
	defer func() {
		scanDirFlag, scanConfigFlag, scanOutputFlag = oldScanDir, oldScanConfig, oldScanOutput
		listDirFlag, listConfigFlag, listOutputFlag = oldListDir, oldListConfig, oldListOutput
		listTypeFlag, listPMFlag = oldListType, oldListPM
		outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag = oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput
		outdatedTypeFlag, outdatedPMFlag = oldOutdatedType, oldOutdatedPM
		outdatedSkipPreflight = oldOutdatedSkipPreflight
		updateDirFlag, updateConfigFlag, updateOutputFlag = oldUpdateDir, oldUpdateConfig, oldUpdateOutput
		updateDryRunFlag, updateYesFlag = oldUpdateDryRun, oldUpdateYes
		updateSkipPreflight, updateSkipSystemTests = oldUpdateSkipPreflight, oldUpdateSkipSystemTests
		updateSkipLockRun, updateRuleFlag = oldUpdateSkipLock, oldUpdateRule
		updateTypeFlag, updatePMFlag = oldUpdateType, oldUpdatePM
	}()

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
			name: "outdated",
			setupFunc: func() {
				outdatedDirFlag = tmpDir
				outdatedConfigFlag = ""
				outdatedOutputFlag = "xml"
				outdatedTypeFlag = "all"
				outdatedPMFlag = "all"
				outdatedSkipPreflight = true
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

			// Capture output and command error
			var cmdErr error
			output := captureStdout(t, func() {
				cmdErr = tc.runCmd()
			})

			// Log any command error for debugging
			if cmdErr != nil {
				t.Logf("%s command returned error: %v", tc.name, cmdErr)
			}

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

	// Save all flags that will be modified
	oldScanDir, oldScanConfig, oldScanOutput := scanDirFlag, scanConfigFlag, scanOutputFlag
	oldListDir, oldListConfig, oldListOutput := listDirFlag, listConfigFlag, listOutputFlag
	oldListType, oldListPM := listTypeFlag, listPMFlag
	oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput := outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag
	oldOutdatedType, oldOutdatedPM := outdatedTypeFlag, outdatedPMFlag
	oldOutdatedSkipPreflight := outdatedSkipPreflight
	oldUpdateDir, oldUpdateConfig, oldUpdateOutput := updateDirFlag, updateConfigFlag, updateOutputFlag
	oldUpdateDryRun, oldUpdateYes := updateDryRunFlag, updateYesFlag
	oldUpdateSkipPreflight, oldUpdateSkipSystemTests := updateSkipPreflight, updateSkipSystemTests
	oldUpdateSkipLock, oldUpdateRule := updateSkipLockRun, updateRuleFlag
	oldUpdateType, oldUpdatePM := updateTypeFlag, updatePMFlag
	defer func() {
		scanDirFlag, scanConfigFlag, scanOutputFlag = oldScanDir, oldScanConfig, oldScanOutput
		listDirFlag, listConfigFlag, listOutputFlag = oldListDir, oldListConfig, oldListOutput
		listTypeFlag, listPMFlag = oldListType, oldListPM
		outdatedDirFlag, outdatedConfigFlag, outdatedOutputFlag = oldOutdatedDir, oldOutdatedConfig, oldOutdatedOutput
		outdatedTypeFlag, outdatedPMFlag = oldOutdatedType, oldOutdatedPM
		outdatedSkipPreflight = oldOutdatedSkipPreflight
		updateDirFlag, updateConfigFlag, updateOutputFlag = oldUpdateDir, oldUpdateConfig, oldUpdateOutput
		updateDryRunFlag, updateYesFlag = oldUpdateDryRun, oldUpdateYes
		updateSkipPreflight, updateSkipSystemTests = oldUpdateSkipPreflight, oldUpdateSkipSystemTests
		updateSkipLockRun, updateRuleFlag = oldUpdateSkipLock, oldUpdateRule
		updateTypeFlag, updatePMFlag = oldUpdateType, oldUpdatePM
	}()

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
			name: "outdated",
			setupFunc: func() {
				outdatedDirFlag = tmpDir
				outdatedConfigFlag = ""
				outdatedOutputFlag = "csv"
				outdatedTypeFlag = "all"
				outdatedPMFlag = "all"
				outdatedSkipPreflight = true
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

			// Capture output and command error
			var cmdErr error
			output := captureStdout(t, func() {
				cmdErr = tc.runCmd()
			})

			// Log any command error for debugging
			if cmdErr != nil {
				t.Logf("%s command returned error: %v", tc.name, cmdErr)
			}

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
