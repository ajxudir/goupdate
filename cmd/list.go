package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/packages"
	"github.com/ajxudir/goupdate/pkg/supervision"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/warnings"
	"github.com/spf13/cobra"
)

var (
	listTypeFlag   string
	listPMFlag     string
	listRuleFlag   string
	listNameFlag   string
	listGroupFlag  string
	listConfigFlag string
	listDirFlag    string
	listOutputFlag string
	listFileFlag   string
)

var (
	getPackagesFunc            = getPackages
	applyInstalledVersionsFunc = lock.ApplyInstalledVersions
)

var listCmd = &cobra.Command{
	Use:     "list [file...]",
	Aliases: []string{"ls"},
	Short:   "Show declared and installed package versions",
	Long:    `Resolve declared constraints and installed versions from lock files.`,
	RunE:    runList,
}

func init() {
	listCmd.Flags().StringVarP(&listTypeFlag, "type", "t", "all", "Filter by type (comma-separated): all,prod,dev")
	listCmd.Flags().StringVarP(&listPMFlag, "package-manager", "p", "all", "Filter by package manager (comma-separated)")
	listCmd.Flags().StringVarP(&listRuleFlag, "rule", "r", "all", "Filter by rule (comma-separated)")
	listCmd.Flags().StringVarP(&listNameFlag, "name", "n", "", "Filter by package name (comma-separated)")
	listCmd.Flags().StringVarP(&listGroupFlag, "group", "g", "", "Filter by group (comma-separated)")
	listCmd.Flags().StringVarP(&listConfigFlag, "config", "c", "", "Config file path")
	listCmd.Flags().StringVarP(&listDirFlag, "directory", "d", ".", "Directory to scan")
	listCmd.Flags().StringVarP(&listOutputFlag, "output", "o", "", "Output format: json, csv, xml (default: table)")
	listCmd.Flags().StringVarP(&listFileFlag, "file", "f", "", "Filter by file path patterns (comma-separated, supports globs)")
}

// runList executes the list command to display package versions.
//
// Lists all declared packages with their constraint, version, installed version,
// and status. Supports filtering by type, package manager, rule, name, and group.
//
// Parameters:
//   - cmd: Cobra command instance
//   - args: Optional file paths to list (empty to auto-detect)
//
// Returns:
//   - error: Returns error on config or parsing failure
func runList(cmd *cobra.Command, args []string) error {
	// Validate flag compatibility before proceeding
	outputFormat := getListOutputFormat()
	if err := output.ValidateStructuredOutputFlags(outputFormat, verboseFlag); err != nil {
		return err
	}

	collector := &display.WarningCollector{}
	restoreWarnings := warnings.SetWarningWriter(collector)
	defer restoreWarnings()
	unsupported := supervision.NewUnsupportedTracker()

	workDir := listDirFlag

	cfg, err := loadAndValidateConfig(listConfigFlag, workDir)
	if err != nil {
		return err // Error already formatted with hints
	}

	workDir = resolveWorkingDir(workDir, cfg)
	cfg.WorkingDir = workDir

	pkgs, err := getPackagesFunc(cfg, args, workDir)
	if err != nil {
		return err
	}

	// Apply file filter if specified
	if listFileFlag != "" {
		pkgs = filtering.FilterPackagesByFile(pkgs, listFileFlag, workDir)
	}

	pkgs = filtering.FilterPackagesWithFilters(pkgs, listTypeFlag, listPMFlag, listRuleFlag, listNameFlag, "")
	pkgs, err = applyInstalledVersionsFunc(pkgs, cfg, workDir)
	if err != nil {
		return err
	}
	pkgs = filtering.ApplyPackageGroups(pkgs, cfg)
	pkgs = filtering.FilterByGroup(pkgs, listGroupFlag)
	for _, p := range pkgs {
		if supervision.ShouldTrackUnsupported(p.InstallStatus) {
			unsupported.Add(p, supervision.DeriveUnsupportedReason(p, cfg, nil, false))
		}
	}

	if len(pkgs) == 0 {
		if output.IsStructuredFormat(outputFormat) {
			return printListStructured(pkgs, collector.Messages(), outputFormat)
		}
		display.PrintNoPackagesMessageWithFilters(os.Stdout, listTypeFlag, listPMFlag, listRuleFlag)
		display.PrintUnsupportedMessages(os.Stdout, unsupported.Messages())
		display.PrintWarnings(os.Stdout, collector.Messages())
		return nil
	}

	if output.IsStructuredFormat(outputFormat) {
		return printListStructured(pkgs, collector.Messages(), outputFormat)
	}

	printPackages(pkgs)
	display.PrintUnsupportedMessages(os.Stdout, unsupported.Messages())
	display.PrintWarnings(os.Stdout, collector.Messages())
	return nil
}

// getListOutputFormat determines the output format for list results.
//
// Parses the --output flag value and returns the corresponding format.
// If no flag is specified, defaults to table format.
//
// Returns:
//   - output.Format: Parsed format (JSON, CSV, XML, or Table)
func getListOutputFormat() output.Format {
	return output.ParseFormat(listOutputFlag)
}

// printListStructured outputs list results in a structured format.
//
// Converts packages to structured output format, sorts for consistent display,
// and outputs in the requested format (JSON, CSV, or XML).
//
// Parameters:
//   - pkgs: Packages to output
//   - warnings: Warning messages to include in output
//   - format: Output format to use
//
// Returns:
//   - error: Returns error on output failure
func printListStructured(pkgs []formats.Package, warnings []string, format output.Format) error {
	sortedPkgs := filtering.SortPackagesForDisplay(pkgs)

	packages := make([]output.ListPackage, 0, len(sortedPkgs))
	for _, p := range sortedPkgs {
		constraintDisplay := display.FormatConstraintDisplay(p)
		packages = append(packages, output.ListPackage{
			Rule:             p.Rule,
			PM:               p.PackageType,
			Type:             p.Type,
			Constraint:       constraintDisplay,
			Version:          display.SafeDeclaredValue(p.Version),
			InstalledVersion: display.SafeInstalledValue(p.InstalledVersion),
			Status:           p.InstallStatus,
			Group:            p.Group,
			Name:             p.Name,
			IgnoreReason:     p.IgnoreReason,
		})
	}

	result := &output.ListResult{
		Summary: output.ListSummary{
			TotalPackages: len(packages),
		},
		Packages: packages,
		Warnings: warnings,
	}

	return output.WriteListResult(os.Stdout, format, result)
}

// getPackages retrieves packages either from specified files or by auto-detection.
//
// If args contains file paths, parses only those files. Otherwise, auto-detects
// and parses all matching files in the working directory.
//
// Parameters:
//   - cfg: Configuration containing rules for parsing
//   - args: File paths to parse (empty for auto-detection)
//   - workDir: Working directory for file detection
//
// Returns:
//   - []formats.Package: Parsed packages
//   - error: Returns error on parsing failure
func getPackages(cfg *config.Config, args []string, workDir string) ([]formats.Package, error) {
	parser := packages.NewDynamicParser()

	if len(args) > 0 {
		return parseSpecificFiles(args, cfg, parser)
	}

	return detectAndParseAll(cfg, parser, workDir)
}

// parseSpecificFiles parses a list of explicitly specified files.
//
// Parameters:
//   - files: File paths to parse
//   - cfg: Configuration containing rules for file matching
//   - parser: Parser instance for file parsing
//
// Returns:
//   - []formats.Package: Parsed packages from all files
//   - error: Returns error if no rule matches or parsing fails
func parseSpecificFiles(files []string, cfg *config.Config, parser *packages.DynamicParser) ([]formats.Package, error) {
	var pkgs []formats.Package

	for _, file := range files {
		ruleCfg, ruleKey := findRuleForFile(file, cfg)
		if ruleCfg == nil {
			return nil, fmt.Errorf("no rule config found for file: %s", file)
		}

		pkgList, err := parser.ParseFile(file, ruleCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}

		for i := range pkgList.Packages {
			pkgList.Packages[i].Rule = ruleKey
			pkgList.Packages[i].Source = file
		}

		pkgs = append(pkgs, pkgList.Packages...)
	}

	return pkgs, nil
}

// findRuleForFile finds the appropriate rule configuration for a given file.
//
// Matches the file path against all enabled rules' include/exclude patterns.
// If multiple rules match, resolves the conflict using rule priority logic.
//
// Parameters:
//   - file: File path to match
//   - cfg: Configuration containing rules
//
// Returns:
//   - *config.PackageManagerCfg: Matching rule configuration (nil if no match)
//   - string: Rule key name (empty if no match)
func findRuleForFile(file string, cfg *config.Config) (*config.PackageManagerCfg, string) {
	normalized := filepath.ToSlash(file)
	if cfg != nil && cfg.WorkingDir != "" {
		if rel, err := filepath.Rel(cfg.WorkingDir, file); err == nil {
			normalized = filepath.ToSlash(rel)
		}
	}

	candidates := make([]string, 0)
	ruleCopies := make(map[string]config.PackageManagerCfg)
	for key, rule := range cfg.Rules {
		// Skip disabled rules
		if !rule.IsEnabled() {
			continue
		}
		ruleCopy := rule
		if utils.MatchPatterns(normalized, ruleCopy.Include, ruleCopy.Exclude) {
			candidates = append(candidates, key)
			ruleCopies[key] = ruleCopy
		}
	}

	if len(candidates) == 0 {
		return nil, ""
	}

	selected := candidates[0]
	if len(candidates) > 1 {
		selected = packages.ResolveRuleForFile(cfg, file, candidates)
	}

	rule := ruleCopies[selected]
	return &rule, selected
}

// detectAndParseAll auto-detects and parses all matching package files.
//
// Uses the configured rules to detect files, then parses each with the
// appropriate parser. Continues on parse errors with warnings.
//
// Parameters:
//   - cfg: Configuration containing detection rules
//   - parser: Parser instance for file parsing
//   - workDir: Working directory for file detection
//
// Returns:
//   - []formats.Package: Parsed packages from all detected files
//   - error: Returns error on detection failure
func detectAndParseAll(cfg *config.Config, parser *packages.DynamicParser, workDir string) ([]formats.Package, error) {
	detected, err := detectFilesFunc(cfg, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect files: %w", err)
	}

	if len(detected) == 0 {
		return []formats.Package{}, nil
	}

	var pkgs []formats.Package

	for ruleKey, files := range detected {
		ruleCfg := cfg.Rules[ruleKey]
		for _, file := range files {
			pkgList, err := parser.ParseFile(file, &ruleCfg)
			if err != nil {
				warnings.Warnf("⚠️ failed to parse %s: %v\n", file, err)
				continue
			}
			for i := range pkgList.Packages {
				pkgList.Packages[i].Rule = ruleKey
				pkgList.Packages[i].Source = file
			}
			pkgs = append(pkgs, pkgList.Packages...)
		}
	}

	return pkgs, nil
}

// listDisplayRow holds pre-formatted display values for a single package row.
type listDisplayRow struct {
	pkg               formats.Package
	constraintDisplay string
	statusDisplay     string
}

// prepareListDisplayRows prepares display data for package listing.
//
// Formats constraint and status values for each package while capturing
// any warnings generated during formatting.
//
// Parameters:
//   - pkgs: Packages to prepare for display
//
// Returns:
//   - []listDisplayRow: Formatted rows ready for table output
//   - string: Captured warning messages
//   - io.Writer: Previous warning writer for restoration
func prepareListDisplayRows(pkgs []formats.Package) ([]listDisplayRow, string, io.Writer) {
	rows := make([]listDisplayRow, 0, len(pkgs))
	var warningsOut strings.Builder
	warningWriter := warnings.WarningWriter()

	for _, p := range pkgs {
		constraintDisplay, warn, writer := captureWarnings(func() string {
			return display.FormatConstraintDisplay(p)
		})
		warningWriter = writer
		if warn != "" {
			warningsOut.WriteString(warn)
		}

		rows = append(rows, listDisplayRow{
			pkg:               p,
			constraintDisplay: constraintDisplay,
			statusDisplay:     display.FormatStatusWithIcon(p.InstallStatus),
		})
	}

	return rows, warningsOut.String(), warningWriter
}

// printPackages outputs packages in table format to stdout.
//
// Sorts packages for display, formats all values, and prints a table
// with headers showing all package information.
//
// Parameters:
//   - pkgs: Packages to display
func printPackages(pkgs []formats.Package) {
	sortedPkgs := filtering.SortPackagesForDisplay(pkgs)
	rows, warningsOut, warningWriter := prepareListDisplayRows(sortedPkgs)

	table := buildListTable(rows)

	if warningsOut != "" {
		_, _ = fmt.Fprint(warningWriter, warningsOut)
	}

	fmt.Println(table.HeaderRow())
	fmt.Println(table.SeparatorRow())

	for _, row := range rows {
		fmt.Println(table.FormatRow(
			row.pkg.Rule,
			row.pkg.PackageType,
			row.pkg.Type,
			row.constraintDisplay,
			display.SafeDeclaredValue(row.pkg.Version),
			display.SafeInstalledValue(row.pkg.InstalledVersion),
			row.statusDisplay,
			row.pkg.Group,
			row.pkg.Name,
		))
	}
	fmt.Printf("\nTotal packages: %d\n", len(pkgs))
}

// buildListTable creates a table formatter with calculated column widths.
//
// Initializes a table with package information columns, conditionally
// including the GROUP column based on whether any packages have groups.
//
// Parameters:
//   - rows: Display rows to calculate widths from
//
// Returns:
//   - *output.Table: Configured table formatter ready for output
func buildListTable(rows []listDisplayRow) *output.Table {
	// Extract groups to determine if GROUP column should be shown
	groups := make([]string, len(rows))
	for i, row := range rows {
		groups[i] = row.pkg.Group
	}
	showGroup := output.ShouldShowGroupColumn(groups)

	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("STATUS").
		AddConditionalColumn("GROUP", showGroup).
		AddColumn("NAME")

	for _, row := range rows {
		table.UpdateWidths(
			row.pkg.Rule,
			row.pkg.PackageType,
			row.pkg.Type,
			row.constraintDisplay,
			display.SafeDeclaredValue(row.pkg.Version),
			display.SafeInstalledValue(row.pkg.InstalledVersion),
			row.statusDisplay,
			row.pkg.Group,
			row.pkg.Name,
		)
	}

	return table
}

// captureWarnings executes a function while capturing warning output.
//
// Temporarily redirects warning output to capture any warnings generated
// during the function execution.
//
// Parameters:
//   - format: Function to execute that may generate warnings
//
// Returns:
//   - string: Result of the function
//   - string: Captured warning messages
//   - io.Writer: Previous warning writer for reference
func captureWarnings(format func() string) (string, string, io.Writer) {
	var captured bytes.Buffer
	previousWriter := warnings.WarningWriter()
	restore := warnings.SetWarningWriter(&captured)
	result := format()
	restore()

	return result, captured.String(), previousWriter
}
