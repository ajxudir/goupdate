package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/packages"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/spf13/cobra"
)

var (
	scanDirFlag    string
	scanConfigFlag string
	scanOutputFlag string
	scanFileFlag   string
)

var detectFilesFunc = packages.DetectFiles

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Discover package manifest files",
	Long:  `Scan for all package files based on configuration rules.`,
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVarP(&scanDirFlag, "directory", "d", ".", "Directory to scan")
	scanCmd.Flags().StringVarP(&scanConfigFlag, "config", "c", "", "Config file path")
	scanCmd.Flags().StringVarP(&scanOutputFlag, "output", "o", "", "Output format: json, csv, xml (default: table)")
	scanCmd.Flags().StringVarP(&scanFileFlag, "file", "f", "", "Filter by file path patterns (comma-separated, supports globs)")
}

// runScan executes the scan command to discover package manifest files.
//
// Scans the working directory for files matching configured rules and
// validates each file by attempting to parse it.
//
// Parameters:
//   - cmd: Cobra command instance
//   - args: Command line arguments (unused)
//
// Returns:
//   - error: Returns error on config loading or detection failure
func runScan(cmd *cobra.Command, args []string) error {
	// Scan uses non-validating config load to avoid errors from malformed test fixtures
	cfg, err := loadConfigWithoutValidation(scanConfigFlag, scanDirFlag)
	if err != nil {
		return err
	}

	workDir := resolveWorkingDir(scanDirFlag, cfg)
	cfg.WorkingDir = workDir

	detected, err := detectFilesFunc(cfg, workDir)
	if err != nil {
		return fmt.Errorf("failed to detect files: %w", err)
	}

	// Apply file filter if specified
	if scanFileFlag != "" {
		detected = filtering.FilterDetectedFiles(detected, scanFileFlag, workDir)
	}

	if len(detected) == 0 {
		outputFormat := getScanOutputFormat()
		if output.IsStructuredFormat(outputFormat) {
			// Output empty result in structured format
			result := &output.ScanResult{
				Summary: output.ScanSummary{
					Directory:    workDir,
					TotalEntries: 0,
					UniqueFiles:  0,
					RulesMatched: 0,
				},
				Files: []output.ScanEntry{},
			}
			return output.WriteScanResult(os.Stdout, outputFormat, result)
		}
		fmt.Printf("No package files found in %s\n", workDir)
		return nil
	}

	outputFormat := getScanOutputFormat()
	if output.IsStructuredFormat(outputFormat) {
		return printScannedFilesStructured(detected, workDir, cfg, outputFormat)
	}

	printScannedFiles(detected, workDir, cfg)
	return nil
}

// getScanOutputFormat determines the output format for scan results.
//
// Parses the --output flag value and returns the corresponding format.
// If no flag is specified, defaults to table format.
//
// Returns:
//   - output.Format: Parsed format (JSON, CSV, XML, or Table)
func getScanOutputFormat() output.Format {
	return output.ParseFormat(scanOutputFlag)
}

// printScannedFilesStructured outputs scan results in a structured format.
//
// Converts detected files to structured output entries, validates each file
// by parsing, and outputs in the requested format (JSON, CSV, or XML).
//
// Parameters:
//   - detected: Map of rule names to detected file paths
//   - baseDir: Base directory for relative path calculation
//   - cfg: Configuration containing rule definitions
//   - format: Output format to use
//
// Returns:
//   - error: Returns error on output failure
func printScannedFilesStructured(detected map[string][]string, baseDir string, cfg *config.Config, format output.Format) error {
	var entries []output.ScanEntry
	uniqueFiles := make(map[string]struct{})
	validFiles := 0
	invalidFiles := 0
	parser := packages.NewDynamicParser()

	for rule, files := range detected {
		ruleCfg := cfg.Rules[rule]
		for _, file := range files {
			relPath, _ := filepath.Rel(baseDir, file)
			if relPath == "" {
				relPath = filepath.Base(file)
			}

			// Validate the file by trying to parse it
			status, errMsg := validateFile(parser, file, &ruleCfg)

			entries = append(entries, output.ScanEntry{
				Rule:   rule,
				PM:     ruleCfg.Manager,
				Format: ruleCfg.Format,
				File:   relPath,
				Status: status,
				Error:  errMsg,
			})
			uniqueFiles[relPath] = struct{}{}
			if status == constants.ValidationValid {
				validFiles++
			} else {
				invalidFiles++
			}
		}
	}

	// Sort entries for consistent output
	// Note: Since PM and Format are derived from cfg.Rules[rule], entries with the
	// same rule always have the same PM and Format. We sort by Rule first, then by File.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Rule != entries[j].Rule {
			return entries[i].Rule < entries[j].Rule
		}
		return entries[i].File < entries[j].File
	})

	result := &output.ScanResult{
		Summary: output.ScanSummary{
			Directory:    baseDir,
			TotalEntries: len(entries),
			UniqueFiles:  len(uniqueFiles),
			RulesMatched: len(detected),
			ValidFiles:   validFiles,
			InvalidFiles: invalidFiles,
		},
		Files: entries,
	}

	return output.WriteScanResult(os.Stdout, format, result)
}

// scannedEntry represents a single scanned file entry for display.
type scannedEntry struct {
	rule   string
	pm     string
	format string
	file   string
	status string
	errMsg string
}

// compareScannedEntries compares two scanned entries for consistent sorting.
//
// Implements multi-key sorting by rule, package manager, format, and file name.
// Used to ensure deterministic output order in scan results.
//
// Parameters:
//   - a: First entry to compare
//   - b: Second entry to compare
//
// Returns:
//   - bool: True if entry a should come before entry b
func compareScannedEntries(a, b scannedEntry) bool {
	if a.rule != b.rule {
		return a.rule < b.rule
	}
	if a.pm != b.pm {
		return a.pm < b.pm
	}
	if a.format != b.format {
		return a.format < b.format
	}
	return a.file < b.file
}

// printScannedFiles outputs scan results in table format to stdout.
//
// Displays each detected file with its rule, package manager, format,
// and validation status. Includes summary statistics.
//
// Parameters:
//   - detected: Map of rule names to detected file paths
//   - baseDir: Base directory for relative path display
//   - cfg: Configuration containing rule definitions
func printScannedFiles(detected map[string][]string, baseDir string, cfg *config.Config) {
	fmt.Printf("Scanned package files in %s\n\n", baseDir)

	totalFiles := 0
	validFiles := 0
	invalidFiles := 0
	uniqueFiles := make(map[string]struct{})
	var entries []scannedEntry
	parser := packages.NewDynamicParser()

	for rule, files := range detected {
		ruleCfg := cfg.Rules[rule]
		for _, file := range files {
			relPath, _ := filepath.Rel(baseDir, file)
			if relPath == "" {
				relPath = filepath.Base(file)
			}

			// Validate the file by trying to parse it
			status, errMsg := validateFile(parser, file, &ruleCfg)

			entries = append(entries, scannedEntry{
				rule:   rule,
				pm:     ruleCfg.Manager,
				format: ruleCfg.Format,
				file:   relPath,
				status: status,
				errMsg: errMsg,
			})
			totalFiles++
			uniqueFiles[relPath] = struct{}{}
			if status == constants.ValidationValid {
				validFiles++
			} else {
				invalidFiles++
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return compareScannedEntries(entries[i], entries[j])
	})

	table := buildScanTable(entries)
	fmt.Println(table.HeaderRow())
	fmt.Println(table.SeparatorRow())

	for _, entry := range entries {
		fmt.Println(table.FormatRow(entry.rule, entry.pm, entry.format, entry.file, entry.status))
	}
	fmt.Printf("\nTotal entries: %d\n", totalFiles)
	fmt.Printf("Unique files: %d\n", len(uniqueFiles))
	fmt.Printf("Rules matched: %d\n", len(detected))
	fmt.Printf("Valid files: %d\n", validFiles)
	fmt.Printf("Invalid files: %d\n", invalidFiles)
}

// buildScanTable creates a table formatter with calculated column widths.
//
// Initializes a table with RULE, PM, FORMAT, FILE, and STATUS columns,
// then calculates optimal column widths based on entry content.
//
// Parameters:
//   - entries: Scanned entries to calculate widths from
//
// Returns:
//   - *output.Table: Configured table formatter ready for output
func buildScanTable(entries []scannedEntry) *output.Table {
	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("FORMAT").
		AddColumn("FILE").
		AddColumn("STATUS")

	for _, entry := range entries {
		table.UpdateWidths(entry.rule, entry.pm, entry.format, entry.file, entry.status)
	}

	return table
}

// validateFile attempts to parse a file and returns its validation status.
//
// Attempts to parse the file using the dynamic parser and reports whether
// the file is valid or contains syntax/format errors. Suppresses verbose
// output during validation since scan only needs to report file validity.
//
// Parameters:
//   - parser: Dynamic parser instance for file parsing
//   - filePath: Path to the file to validate
//   - cfg: Package manager configuration for this file type
//
// Returns:
//   - status: ValidationValid if file parses successfully, ValidationInvalid if it fails
//   - errMsg: Empty string on success, error message on failure
func validateFile(parser *packages.DynamicParser, filePath string, cfg *config.PackageManagerCfg) (status string, errMsg string) {
	// Suppress verbose output during validation - scan only needs to validate, not log parsing details
	verbose.Suppress()
	_, err := parser.ParseFile(filePath, cfg)
	verbose.Unsuppress()
	if err != nil {
		return constants.ValidationInvalid, err.Error()
	}
	return constants.ValidationValid, ""
}
