package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/output"
	"github.com/user/goupdate/pkg/packages"
)

// TestScanCommand tests the behavior of the scan command.
//
// It verifies:
//   - Scan command executes without errors
//   - Scan command can process a directory with package files
//   - Command line arguments are properly handled
func TestScanCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0644)
	require.NoError(t, err)

	os.Args = []string{"goupdate", "scan", "-d", tmpDir}
	err = ExecuteTest()
	assert.NoError(t, err)
}

// TestRunScanNoMatches tests the behavior of scan when no package files are found.
//
// It verifies:
//   - Scan completes without errors when no files are found
//   - Output contains "No package files found" message
//   - Empty directories are handled gracefully
func TestRunScanNoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	defer func() {
		os.Args = oldArgs
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	os.Args = []string{"goupdate", "scan", "-d", tmpDir}

	out := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, out, "No package files found")
}

// TestRunScanNoMatchesStructuredOutput tests the behavior of scan with JSON output when no files are found.
//
// It verifies:
//   - JSON output is generated for empty scan results
//   - Output contains valid JSON structure with zero counts
//   - Structured output format works with empty results
func TestRunScanNoMatchesStructuredOutput(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldOutput := scanOutputFlag
	defer func() {
		os.Args = oldArgs
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "json"
	os.Args = []string{"goupdate", "scan", "-d", tmpDir, "--output", "json"}

	out := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should output empty JSON structure
	assert.Contains(t, out, `"total_entries":0`)
	assert.Contains(t, out, `"unique_files":0`)
	assert.Contains(t, out, `"rules_matched":0`)
}

// TestRunScanConfigError tests the behavior of scan when config file is missing.
//
// It verifies:
//   - Scan returns error when specified config file doesn't exist
//   - Error handling for missing config files
//   - Config file validation occurs before scanning
func TestRunScanConfigError(t *testing.T) {
	tmpDir := t.TempDir()

	oldArgs := os.Args
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	defer func() {
		os.Args = oldArgs
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
	}()

	badCfg := filepath.Join(tmpDir, "missing.yml")
	scanDirFlag = tmpDir
	scanConfigFlag = badCfg
	os.Args = []string{"goupdate", "scan", "--config", badCfg}

	err := runScan(nil, nil)
	assert.Error(t, err)
}

// TestRunScanDetectError tests the behavior of scan when file detection fails.
//
// It verifies:
//   - Scan returns error when file detection fails
//   - Error message contains "failed to detect files"
//   - Detection errors are properly propagated
func TestRunScanDetectError(t *testing.T) {
	oldDetect := detectFilesFunc
	defer func() { detectFilesFunc = oldDetect }()

	detectFilesFunc = func(cfg *config.Config, baseDir string) (map[string][]string, error) {
		return nil, fmt.Errorf("detect failure")
	}

	oldArgs := os.Args
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	defer func() {
		os.Args = oldArgs
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
	}()

	scanDirFlag = t.TempDir()
	scanConfigFlag = ""
	os.Args = []string{"goupdate", "scan"}

	err := runScan(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect files")
}

// TestPrintScannedFilesAlignment tests the behavior of table column alignment in scan output.
//
// It verifies:
//   - Table columns are properly aligned
//   - Header and data rows have matching column counts
//   - Column spacing is consistent across rows
func TestPrintScannedFilesAlignment(t *testing.T) {
	baseDir := t.TempDir()
	detected := map[string][]string{
		"composer": {filepath.Join(baseDir, "composer", "composer.json")},
		"mod":      {filepath.Join(baseDir, "go", "go.mod")},
	}

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"composer": {Manager: "php", Format: "json"},
		"mod":      {Manager: "golang", Format: "raw"},
	}}

	output := captureStdout(t, func() {
		printScannedFiles(detected, baseDir, cfg)
	})

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	columnSplit := regexp.MustCompile(`\s{2,}`)
	headIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "RULE") {
			headIdx = i
			break
		}
	}
	require.NotEqual(t, -1, headIdx)
	require.GreaterOrEqual(t, len(lines), headIdx+3)
	row := lines[headIdx+2]
	headColumns := columnSplit.Split(strings.TrimSpace(lines[headIdx]), -1)
	rowColumns := columnSplit.Split(strings.TrimSpace(row), -1)
	assert.Equal(t, len(headColumns), len(rowColumns))
}

// TestPrintScannedFiles tests the behavior of printing scanned files in table format.
//
// It verifies:
//   - Scanned files are displayed with rule, manager, and format information
//   - Total entries, unique files, and rules matched counts are shown
//   - Table output contains all expected fields
func TestPrintScannedFiles(t *testing.T) {
	detected := map[string][]string{
		"npm": {"/repo/package.json"},
	}
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {Manager: "js", Format: "json"},
	}}

	output := captureStdout(t, func() {
		printScannedFiles(detected, "/repo", cfg)
	})

	assert.Contains(t, output, "npm")
	assert.Contains(t, output, "js")
	assert.Contains(t, output, "json")
	assert.Contains(t, output, "Total entries: 1")
	assert.Contains(t, output, "Unique files: 1")
	assert.Contains(t, output, "Rules matched: 1")
}

// TestPrintScannedFilesSorted tests the behavior of file sorting in scan output.
//
// It verifies:
//   - Files are sorted alphabetically within each rule
//   - Sort order is consistent across multiple files
//   - File paths are displayed in correct sorted order
func TestPrintScannedFilesSorted(t *testing.T) {
	detected := map[string][]string{
		"npm": {"/repo/b.json", "/repo/a.json"},
	}
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {Manager: "js", Format: "json"},
	}}

	output := captureStdout(t, func() {
		printScannedFiles(detected, "/repo", cfg)
	})

	lines := strings.Split(output, "\n")
	var fileLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "Scanned") || strings.HasPrefix(trimmed, "RULE") || strings.HasPrefix(trimmed, "----") || strings.HasPrefix(trimmed, "Total") || strings.HasPrefix(trimmed, "Unique") || strings.HasPrefix(trimmed, "Rules") {
			continue
		}
		fileLines = append(fileLines, trimmed)
	}

	require.GreaterOrEqual(t, len(fileLines), 2)
	assert.True(t, strings.Contains(fileLines[0], "a.json"))
	assert.True(t, strings.Contains(fileLines[1], "b.json"))
}

// TestPrintScannedFilesEmptyRelPath tests the behavior when relative path calculation results in empty string.
//
// It verifies:
//   - Files at the base directory level are handled correctly
//   - Empty relative paths fall back to base name
//   - Path display works when file is at base directory
func TestPrintScannedFilesEmptyRelPath(t *testing.T) {
	detected := map[string][]string{
		"npm": {"/tmp"},
	}
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {Manager: "js", Format: "json"},
	}}

	output := captureStdout(t, func() {
		printScannedFiles(detected, "/tmp", cfg)
	})

	assert.Contains(t, output, "tmp")
}

// TestPrintScannedFilesEmptyBaseDirFallback tests the behavior when base directory is empty.
//
// It verifies:
//   - Empty base directory triggers fallback to filepath.Base
//   - File name is still displayed when base directory is missing
//   - Fallback mechanism handles edge cases gracefully
func TestPrintScannedFilesEmptyBaseDirFallback(t *testing.T) {
	// Test case where filepath.Rel returns empty string (when baseDir is empty)
	detected := map[string][]string{
		"npm": {"/some/path/package.json"},
	}
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {Manager: "js", Format: "json"},
	}}

	output := captureStdout(t, func() {
		printScannedFiles(detected, "", cfg)
	})

	// When relPath is empty, filepath.Base should be used as fallback
	assert.Contains(t, output, "package.json")
}

// TestCompareScannedEntries tests the behavior of scanned entry comparison for sorting.
//
// It verifies:
//   - Entries are compared by rule name first
//   - Package manager is compared second
//   - Format is compared third, then file name
func TestCompareScannedEntries(t *testing.T) {
	// Different rules
	a := scannedEntry{rule: "a", pm: "js", format: "json", file: "x"}
	b := scannedEntry{rule: "b", pm: "js", format: "json", file: "x"}
	assert.True(t, compareScannedEntries(a, b))
	assert.False(t, compareScannedEntries(b, a))

	// Same rule, different pm
	c := scannedEntry{rule: "a", pm: "generic", format: "json", file: "x"}
	d := scannedEntry{rule: "a", pm: "js", format: "json", file: "x"}
	assert.True(t, compareScannedEntries(c, d))
	assert.False(t, compareScannedEntries(d, c))

	// Same rule and pm, different format
	e := scannedEntry{rule: "a", pm: "js", format: "json", file: "x"}
	f := scannedEntry{rule: "a", pm: "js", format: "yaml", file: "x"}
	assert.True(t, compareScannedEntries(e, f))
	assert.False(t, compareScannedEntries(f, e))

	// Same rule, pm, format, different file
	g := scannedEntry{rule: "a", pm: "js", format: "json", file: "a.json"}
	h := scannedEntry{rule: "a", pm: "js", format: "json", file: "b.json"}
	assert.True(t, compareScannedEntries(g, h))
	assert.False(t, compareScannedEntries(h, g))
}

// TestBuildScanTable tests the behavior of scan table construction.
//
// It verifies:
//   - Table is built with correct column widths
//   - Column headers are properly sized
//   - Table accommodates all entry data
func TestBuildScanTable(t *testing.T) {
	entries := []scannedEntry{
		{rule: "npm", pm: "js", format: "json", file: "package.json", status: "🟢 valid"},
		{rule: "composer", pm: "php", format: "json", file: "composer.json", status: "❌ invalid"},
	}

	table := buildScanTable(entries)

	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("RULE"), 4)   // "RULE" is 4 chars
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("PM"), 2)     // "PM" is 2 chars
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("FORMAT"), 4) // "json" is 4 chars
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("FILE"), 12)  // "package.json" is 12 chars
	assert.GreaterOrEqual(t, table.GetColumnWidthByHeader("STATUS"), 6) // "STATUS" is 6 chars
}

// TestScanTableFormatters tests the behavior of table formatting for scan results.
//
// It verifies:
//   - Table header row contains all expected columns
//   - Separator row is generated correctly
//   - Table formatting functions work as expected
func TestScanTableFormatters(t *testing.T) {
	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("FORMAT").
		AddColumn("FILE").
		AddColumn("STATUS")

	header := table.HeaderRow()
	assert.Contains(t, header, "RULE")
	assert.Contains(t, header, "PM")
	assert.Contains(t, header, "FORMAT")
	assert.Contains(t, header, "FILE")
	assert.Contains(t, header, "STATUS")

	separator := table.SeparatorRow()
	assert.Contains(t, separator, "----")
}

// TestGetScanOutputFormat tests the behavior of output format detection for scan command.
//
// It verifies:
//   - Default output format is "table"
//   - CSV, JSON, and XML formats are properly recognized
//   - Output format flag is correctly parsed
func TestGetScanOutputFormat(t *testing.T) {
	// Save original value
	origOutput := scanOutputFlag
	defer func() {
		scanOutputFlag = origOutput
	}()

	// Test default (table)
	scanOutputFlag = ""
	assert.Equal(t, "table", string(getScanOutputFormat()))

	// Test CSV
	scanOutputFlag = "csv"
	assert.Equal(t, "csv", string(getScanOutputFormat()))

	// Test JSON
	scanOutputFlag = "json"
	assert.Equal(t, "json", string(getScanOutputFormat()))

	// Test XML
	scanOutputFlag = "xml"
	assert.Equal(t, "xml", string(getScanOutputFormat()))
}

// TestScanCommandJSONOutput tests the behavior of scan command with JSON output.
//
// It verifies:
//   - Scan command produces valid JSON output
//   - JSON contains summary and files sections
//   - JSON output format flag is respected
func TestScanCommandJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0644)
	require.NoError(t, err)

	oldArgs := os.Args
	oldOutput := scanOutputFlag
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	defer func() {
		os.Args = oldArgs
		scanOutputFlag = oldOutput
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "json"
	os.Args = []string{"goupdate", "scan", "-d", tmpDir, "--output", "json"}

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should contain JSON structure
	assert.Contains(t, output, "{")
	assert.Contains(t, output, "\"summary\"")
	assert.Contains(t, output, "\"files\"")
}

// TestScanCommandCSVOutput tests the behavior of scan command with CSV output.
//
// It verifies:
//   - Scan command produces valid CSV output
//   - CSV header contains expected columns including STATUS and ERROR
//   - CSV output format flag is respected
func TestScanCommandCSVOutput(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0644)
	require.NoError(t, err)

	oldArgs := os.Args
	oldOutput := scanOutputFlag
	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	defer func() {
		os.Args = oldArgs
		scanOutputFlag = oldOutput
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
	}()

	scanDirFlag = tmpDir
	scanConfigFlag = ""
	scanOutputFlag = "csv"
	os.Args = []string{"goupdate", "scan", "-d", tmpDir, "--output", "csv"}

	output := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should contain CSV header with STATUS and ERROR columns
	assert.Contains(t, output, "RULE,PM,FORMAT,FILE,STATUS,ERROR")
}

// TestPrintScannedFilesStructured tests the behavior of structured output for scanned files.
//
// It verifies:
//   - JSON, CSV, and XML formats produce valid output
//   - Structured output contains all file information
//   - Sorting and fallback logic works in structured formats
func TestPrintScannedFilesStructured(t *testing.T) {
	detected := map[string][]string{
		"npm": {"/tmp/project/package.json"},
		"pip": {"/tmp/project/requirements.txt"},
	}
	baseDir := "/tmp/project"
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {Manager: "js", Format: "json"},
			"pip": {Manager: "python", Format: "raw"},
		},
	}

	t.Run("JSON format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printScannedFilesStructured(detected, baseDir, cfg, output.FormatJSON)
			require.NoError(t, err)
		})
		assert.Contains(t, out, `"rule":"npm"`)
		assert.Contains(t, out, `"rule":"pip"`)
		assert.Contains(t, out, `"total_entries":2`)
	})

	t.Run("CSV format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printScannedFilesStructured(detected, baseDir, cfg, output.FormatCSV)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "npm")
		assert.Contains(t, out, "pip")
	})

	t.Run("XML format", func(t *testing.T) {
		out := captureStdout(t, func() {
			err := printScannedFilesStructured(detected, baseDir, cfg, output.FormatXML)
			require.NoError(t, err)
		})
		assert.Contains(t, out, "<rule>npm</rule>")
		assert.Contains(t, out, "<rule>pip</rule>")
	})

	t.Run("empty baseDir fallback", func(t *testing.T) {
		// Test case where filepath.Rel returns empty string
		detected := map[string][]string{
			"npm": {"/some/path/package.json"},
		}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Format: "json"},
			},
		}

		out := captureStdout(t, func() {
			err := printScannedFilesStructured(detected, "", cfg, output.FormatJSON)
			require.NoError(t, err)
		})

		// When relPath is empty, filepath.Base should be used as fallback
		assert.Contains(t, out, "package.json")
	})

	t.Run("sorting by rule and file", func(t *testing.T) {
		// Test sorting: multiple rules with multiple files each
		detected := map[string][]string{
			"rule_z": {"/tmp/project/z.json", "/tmp/project/a.json"},
			"rule_a": {"/tmp/project/b.json"},
		}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"rule_z": {Manager: "js", Format: "json"},
				"rule_a": {Manager: "py", Format: "raw"},
			},
		}

		out := captureStdout(t, func() {
			err := printScannedFilesStructured(detected, "/tmp/project", cfg, output.FormatJSON)
			require.NoError(t, err)
		})

		// Should contain all entries sorted properly
		assert.Contains(t, out, "rule_a")
		assert.Contains(t, out, "rule_z")
		assert.Contains(t, out, `"total_entries":3`)
	})
}

// Note: Unit tests for ParseFileFilterPatterns, MatchesFileFilter, and FilterDetectedFiles
// have been moved to pkg/filtering/files_test.go

// TestScanWithFileFilter tests the behavior of scan with file pattern filtering.
//
// It verifies:
//   - File filter patterns exclude unwanted files
//   - Filtered results show only matching files
//   - File count reflects filtered results
func TestScanWithFileFilter(t *testing.T) {
	oldDetect := detectFilesFunc
	defer func() { detectFilesFunc = oldDetect }()

	baseDir := t.TempDir()
	detectFilesFunc = func(cfg *config.Config, dir string) (map[string][]string, error) {
		return map[string][]string{
			"mod": {filepath.Join(dir, "go.mod"), filepath.Join(dir, "testdata/go.mod")},
		}, nil
	}

	oldDir := scanDirFlag
	oldConfig := scanConfigFlag
	oldFile := scanFileFlag
	oldOutput := scanOutputFlag
	defer func() {
		scanDirFlag = oldDir
		scanConfigFlag = oldConfig
		scanFileFlag = oldFile
		scanOutputFlag = oldOutput
	}()

	scanDirFlag = baseDir
	scanConfigFlag = ""
	scanFileFlag = "go.mod"
	scanOutputFlag = ""

	out := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should only include root go.mod, not testdata/go.mod
	assert.Contains(t, out, "go.mod")
	assert.Contains(t, out, "Total entries: 1")
}

// TestScanWithMalformedFiles tests the behavior of scan when encountering invalid files.
//
// It verifies:
//   - Scan continues processing when files have parse errors
//   - Both valid and invalid files are reported
//   - Invalid files show error status with counts
func TestScanWithMalformedFiles(t *testing.T) {
	// Test that scan continues even when files have parse errors
	tmpDir := t.TempDir()

	// Create a valid package.json
	validJSON := `{"name": "test", "dependencies": {}}`
	err := os.WriteFile(filepath.Join(tmpDir, "valid.json"), []byte(validJSON), 0644)
	require.NoError(t, err)

	// Create an invalid package.json
	invalidJSON := `{"name": "test" "dependencies": {}}`
	err = os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte(invalidJSON), 0644)
	require.NoError(t, err)

	oldDetect := detectFilesFunc
	defer func() { detectFilesFunc = oldDetect }()

	detectFilesFunc = func(cfg *config.Config, dir string) (map[string][]string, error) {
		return map[string][]string{
			"npm": {filepath.Join(dir, "valid.json"), filepath.Join(dir, "invalid.json")},
		}, nil
	}

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
	scanOutputFlag = ""

	out := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should show both files - one valid, one invalid
	assert.Contains(t, out, "valid.json")
	assert.Contains(t, out, "invalid.json")
	assert.Contains(t, out, "🟢 valid")
	assert.Contains(t, out, "❌ invalid")
	assert.Contains(t, out, "Valid files: 1")
	assert.Contains(t, out, "Invalid files: 1")
}

// TestScanWithMalformedFilesJSON tests the behavior of JSON output with malformed files.
//
// It verifies:
//   - JSON output includes error information for invalid files
//   - Invalid file count is tracked in JSON summary
//   - Error messages are included in JSON output
func TestScanWithMalformedFilesJSON(t *testing.T) {
	// Test JSON output with malformed files
	tmpDir := t.TempDir()

	// Create an invalid package.json
	invalidJSON := `{"name": "test" "dependencies": {}}`
	err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(invalidJSON), 0644)
	require.NoError(t, err)

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

	out := captureStdout(t, func() {
		err := runScan(nil, nil)
		assert.NoError(t, err)
	})

	// Should contain error message in JSON output
	assert.Contains(t, out, `"status":"❌ invalid"`)
	assert.Contains(t, out, `"error"`)
	assert.Contains(t, out, `"invalid_files":1`)
}

// TestValidateFile tests the behavior of file validation for scan results.
//
// It verifies:
//   - Valid JSON files return "valid" status
//   - Invalid JSON files return "invalid" status with error message
//   - File validation correctly identifies parsing errors
func TestValidateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid JSON file
	validJSON := `{"name": "test"}`
	validPath := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validPath, []byte(validJSON), 0644)
	require.NoError(t, err)

	// Create invalid JSON file
	invalidJSON := `{"name": "test"`
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte(invalidJSON), 0644)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	cfg := &config.PackageManagerCfg{
		Manager: "js",
		Format:  "json",
		Fields:  map[string]string{"dependencies": "runtime"},
	}

	t.Run("valid file", func(t *testing.T) {
		status, errMsg := validateFile(parser, validPath, cfg)
		assert.Equal(t, "🟢 valid", status)
		assert.Empty(t, errMsg)
	})

	t.Run("invalid file", func(t *testing.T) {
		status, errMsg := validateFile(parser, invalidPath, cfg)
		assert.Equal(t, "❌ invalid", status)
		assert.NotEmpty(t, errMsg)
	})
}
