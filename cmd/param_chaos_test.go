package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// PARAMETER CHAOS TESTS
// =============================================================================
//
// These tests verify behavior with edge cases, unusual inputs, and stress
// scenarios for command parameters.
// =============================================================================

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
	listTypeFlag = "" // Empty
	listPMFlag = ""   // Empty
	listRuleFlag = "" // Empty
	listNameFlag = "" // Empty

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
			var cmdErr error
			output := captureStdout(t, func() {
				cmdErr = runList(nil, nil)
			})

			// Log error for debugging (may error but should not panic)
			if cmdErr != nil {
				t.Logf("runList (special char '%s') returned error: %v", tc.value, cmdErr)
			}

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
	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runList(nil, nil)
	})

	// Log error for debugging
	if cmdErr != nil {
		t.Logf("runList (very long param) returned error: %v", cmdErr)
	}

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
		name    string
		typeVal string
		pmVal   string
		nameVal string
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

			var cmdErr error
			output := captureStdout(t, func() {
				cmdErr = runList(nil, nil)
			})

			// Log error for debugging
			if cmdErr != nil {
				t.Logf("runList (comma-separated '%s') returned error: %v", tc.name, cmdErr)
			}

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

	var cmdErr error
	output := captureStdout(t, func() {
		cmdErr = runScan(nil, nil)
	})

	// Log error for debugging (may error or fall back to default)
	if cmdErr != nil {
		t.Logf("runScan (invalid format) returned error: %v", cmdErr)
	}

	// Should produce some output (default table format)
	assert.NotEmpty(t, output, "Invalid format should fall back to default")
}

// -----------------------------------------------------------------------------
// PARTIAL ERROR HANDLING TESTS
// -----------------------------------------------------------------------------

// TestPartialError_Update_ContinueOnFail tests that partial errors are properly
// communicated in structured output.
