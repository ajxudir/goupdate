package cmd

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/preflight"
	"github.com/ajxudir/goupdate/pkg/supervision"
	"github.com/ajxudir/goupdate/pkg/update"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/ajxudir/goupdate/pkg/warnings"
	"github.com/spf13/cobra"
)

var (
	outdatedTypeFlag       string
	outdatedPMFlag         string
	outdatedRuleFlag       string
	outdatedNameFlag       string
	outdatedGroupFlag      string
	outdatedConfigFlag     string
	outdatedDirFlag        string
	outdatedFileFlag       string
	outdatedMajorFlag      bool
	outdatedMinorFlag      bool
	outdatedPatchFlag      bool
	outdatedNoTimeoutFlag  bool
	outdatedSkipPreflight  bool
	outdatedContinueOnFail bool
	outdatedOutputFlag     string
)

var listNewerVersionsFunc = outdated.ListNewerVersions

// writeOutdatedResultFunc allows mocking structured output in tests
var writeOutdatedResultFunc = output.WriteOutdatedResult

var outdatedCmd = &cobra.Command{
	Use:   "outdated [file...]",
	Short: "Find packages with available updates",
	Long:  `Compare installed versions against available versions to find outdated packages.`,
	RunE:  runOutdated,
}

func init() {
	outdatedCmd.Flags().StringVarP(&outdatedTypeFlag, "type", "t", "all", "Filter by type (comma-separated): all,prod,dev")
	outdatedCmd.Flags().StringVarP(&outdatedPMFlag, "package-manager", "p", "all", "Filter by package manager (comma-separated)")
	outdatedCmd.Flags().StringVarP(&outdatedRuleFlag, "rule", "r", "all", "Filter by rule (comma-separated)")
	outdatedCmd.Flags().StringVarP(&outdatedNameFlag, "name", "n", "", "Filter by package name (comma-separated)")
	outdatedCmd.Flags().StringVarP(&outdatedGroupFlag, "group", "g", "", "Filter by group (comma-separated)")
	outdatedCmd.Flags().StringVarP(&outdatedConfigFlag, "config", "c", "", "Config file path")
	outdatedCmd.Flags().StringVarP(&outdatedDirFlag, "directory", "d", ".", "Directory to scan")
	outdatedCmd.Flags().StringVarP(&outdatedFileFlag, "file", "f", "", "Filter by file path patterns (comma-separated, supports globs)")
	outdatedCmd.Flags().BoolVar(&outdatedMajorFlag, "major", false, "Allow major, minor, and patch comparisons")
	outdatedCmd.Flags().BoolVar(&outdatedMinorFlag, "minor", false, "Allow minor and patch comparisons")
	outdatedCmd.Flags().BoolVar(&outdatedPatchFlag, "patch", false, "Restrict comparisons to patch scope")
	outdatedCmd.Flags().BoolVar(&outdatedNoTimeoutFlag, "no-timeout", false, "Disable command timeouts")
	outdatedCmd.Flags().BoolVar(&outdatedSkipPreflight, "skip-preflight", false, "Skip pre-flight command validation")
	outdatedCmd.Flags().BoolVar(&outdatedContinueOnFail, "continue-on-fail", false, "Continue processing remaining packages after failures (exit code 1 for partial success)")
	outdatedCmd.Flags().StringVarP(&outdatedOutputFlag, "output", "o", "", "Output format: json, csv, xml (default: table)")
}

// outdatedResult holds the result of checking a package for available updates.
type outdatedResult struct {
	pkg           formats.Package
	group         string
	major         string
	minor         string
	patch         string
	target        string
	status        string
	available     []string
	err           error
	latestMissing bool
}

const (
	outdatedStatusOutdated = constants.StatusOutdated
	outdatedStatusUpToDate = constants.StatusUpToDate
	outdatedStatusFailed   = constants.StatusFailed
)

// runOutdated executes the outdated command to find packages with available updates.
//
// Checks each package against its registry for newer versions, categorizing
// available updates by major, minor, and patch versions.
//
// Parameters:
//   - cmd: Cobra command instance
//   - args: Optional file paths to check (empty to auto-detect)
//
// Returns:
//   - error: Returns ExitError with appropriate code on failure
func runOutdated(cmd *cobra.Command, args []string) error {
	// Validate flag compatibility before proceeding
	outputFormat := getOutdatedOutputFormat()
	if err := output.ValidateStructuredOutputFlags(outputFormat, verboseFlag); err != nil {
		return err
	}

	collector := &display.WarningCollector{}
	restoreWarnings := warnings.SetWarningWriter(collector)
	defer restoreWarnings()
	unsupported := supervision.NewUnsupportedTracker()

	workDir := outdatedDirFlag

	cfg, err := loadAndValidateConfig(outdatedConfigFlag, workDir)
	if err != nil {
		return err // Error already formatted with hints
	}

	workDir = resolveWorkingDir(workDir, cfg)
	cfg.WorkingDir = workDir
	cfg.NoTimeout = outdatedNoTimeoutFlag

	packages, err := getPackagesFunc(cfg, args, workDir)
	if err != nil {
		return err
	}

	// Apply file filter if specified
	if outdatedFileFlag != "" {
		packages = filtering.FilterPackagesByFile(packages, outdatedFileFlag, workDir)
	}

	packages = filtering.FilterPackagesWithFilters(packages, outdatedTypeFlag, outdatedPMFlag, outdatedRuleFlag, outdatedNameFlag, "")
	packages, err = applyInstalledVersionsFunc(packages, cfg, workDir)
	if err != nil {
		return err
	}
	packages = filtering.ApplyPackageGroups(packages, cfg)
	packages = filtering.FilterByGroup(packages, outdatedGroupFlag)
	for _, p := range packages {
		if supervision.ShouldTrackUnsupported(p.InstallStatus) {
			unsupported.Add(p, supervision.DeriveUnsupportedReason(p, cfg, nil, false))
		}
	}

	if len(packages) == 0 {
		if output.IsStructuredFormat(outputFormat) {
			return printOutdatedStructured(nil, collector.Messages(), nil, outputFormat)
		}
		display.PrintNoPackagesMessageWithFilters(os.Stdout, outdatedTypeFlag, outdatedPMFlag, outdatedRuleFlag)
		return nil
	}

	// Run pre-flight validation unless skipped
	if !outdatedSkipPreflight {
		validation := preflight.ValidatePackages(packages, cfg)
		if validation.HasErrors() {
			verbose.Infof("Exit code %d (config error): preflight validation failed - %s", errors.ExitConfigError, validation.ErrorMessage())
			return errors.NewExitError(errors.ExitConfigError, fmt.Errorf("%s\n  💡 Options:\n     --skip-preflight     Bypass validation if commands are available through other means\n     --rule <name>        Filter to specific rules (e.g., --rule npm)\n     enabled: false       Disable unused rules in your config file", validation.ErrorMessage()))
		}
	}

	ordered := filtering.SortPackagesForDisplay(packages)

	// For structured output, suppress progress entirely (no stderr output)
	// Progress messages are only shown in table (interactive) mode
	useStructuredOutput := output.IsStructuredFormat(outputFormat)
	var progress *output.Progress // nil for structured output - Progress methods are nil-safe

	var table *output.Table
	if !useStructuredOutput {
		// Calculate column widths from package data (before fetching versions)
		table = buildOutdatedTableFromPackages(ordered)

		// Print header
		fmt.Println(table.HeaderRow())
		fmt.Println(table.SeparatorRow())
	}

	results := make([]outdatedResult, 0, len(ordered))
	var errs []error
	selection := outdated.UpdateSelectionFlags{Major: outdatedMajorFlag, Minor: outdatedMinorFlag, Patch: outdatedPatchFlag}

	for _, p := range ordered {
		ruleCfg := cfg.Rules[p.Rule]

		// Skip outdated command for Ignored packages - they are excluded by config
		if p.InstallStatus == lock.InstallStatusIgnored {
			result := outdatedResult{
				pkg:    p,
				group:  p.Group,
				major:  constants.PlaceholderNA,
				minor:  constants.PlaceholderNA,
				patch:  constants.PlaceholderNA,
				status: lock.InstallStatusIgnored,
			}
			results = append(results, result)
			if useStructuredOutput {
				progress.Increment()
			} else {
				printOutdatedRowWithTable(result, table)
			}
			continue
		}

		// Skip outdated command for Floating packages - they cannot be processed automatically
		// because their constraints (*, x, ranges) make version comparison meaningless
		if p.InstallStatus == lock.InstallStatusFloating {
			result := outdatedResult{
				pkg:    p,
				group:  p.Group,
				major:  constants.PlaceholderNA,
				minor:  constants.PlaceholderNA,
				patch:  constants.PlaceholderNA,
				status: lock.InstallStatusFloating,
			}
			results = append(results, result)
			if useStructuredOutput {
				progress.Increment()
			} else {
				printOutdatedRowWithTable(result, table)
			}
			continue
		}

		versions, err := listNewerVersionsFunc(context.Background(), p, cfg, workDir)

		result := outdatedResult{pkg: p, group: p.Group, err: err, major: constants.PlaceholderNA, minor: constants.PlaceholderNA, patch: constants.PlaceholderNA, latestMissing: isLatestMissing(p, &ruleCfg)}
		if err == nil {
			// For display, show ALL available versions (including major) without constraint filtering
			// This ensures users see major updates even when their package uses ^ or ~ constraints
			displayFiltered := outdated.FilterVersionsByConstraint(p, versions, outdated.UpdateSelectionFlags{Major: true})
			targetFiltered := outdated.FilterVersionsByConstraint(p, versions, selection)
			result.available = targetFiltered

			incremental, incrementalErr := config.ShouldUpdateIncrementally(p, cfg)
			if incrementalErr != nil {
				result.err = stderrors.Join(result.err, incrementalErr)
			} else {
				displayMajor, displayMinor, displayPatch, summarizeErr := outdated.SummarizeAvailableVersions(outdated.CurrentVersionForOutdated(p), displayFiltered, ruleCfg.Outdated.Versioning, incremental)
				if summarizeErr != nil {
					result.err = stderrors.Join(result.err, summarizeErr)
				} else {
					result.major = displayMajor
					result.minor = displayMinor
					result.patch = displayPatch
				}

				targetMajor, targetMinor, targetPatch, targetSummarizeErr := outdated.SummarizeAvailableVersions(outdated.CurrentVersionForOutdated(p), targetFiltered, ruleCfg.Outdated.Versioning, incremental)
				if targetSummarizeErr != nil {
					result.err = stderrors.Join(result.err, targetSummarizeErr)
				}

				if target, targetErr := outdated.SelectTargetVersion(targetMajor, targetMinor, targetPatch, selection, p.Constraint, incremental); targetErr == nil {
					result.target = target
				}
			}
		}

		unsupportedErr := errors.IsUnsupported(err)
		if unsupportedErr {
			result.err = nil
			result.status = lock.InstallStatusNotConfigured
			unsupported.Add(p, supervision.DeriveUnsupportedReason(p, cfg, err, result.latestMissing))
		} else {
			result.status = deriveOutdatedStatus(result)
			// Note: shouldTrackUnsupported is not checked here because deriveOutdatedStatus
			// only returns Floating (handled earlier), Failed, Outdated, or UpToDate.
			// NotConfigured status is only set in the if branch above.
			if result.err != nil {
				errs = append(errs, fmt.Errorf("%s (%s/%s): %w", p.Name, p.PackageType, p.Rule, result.err))
			}
		}

		results = append(results, result)

		if useStructuredOutput {
			progress.Increment()
		} else {
			// Print row immediately (live output)
			printOutdatedRowWithTable(result, table)
		}
	}

	if useStructuredOutput {
		progress.Done()
		// Convert errors to strings for output
		var errStrings []string
		for _, e := range errs {
			errStrings = append(errStrings, e.Error())
		}
		if err := printOutdatedStructured(results, collector.Messages(), errStrings, outputFormat); err != nil {
			return err
		}
	} else {
		// Convert results to summary format
		summaryData := make([]update.OutdatedResultData, len(results))
		for i, res := range results {
			summaryData[i] = update.OutdatedResultData{
				Status: res.status,
				Major:  res.major,
				Minor:  res.minor,
				Patch:  res.patch,
				Err:    res.err,
			}
		}

		fmt.Printf("\nTotal packages: %d\n", len(results))
		counts := update.ComputeSummaryFromOutdatedResults(summaryData)
		update.PrintUpdateSummaryLines(counts, update.SummaryModeOutdated)
		display.PrintUnsupportedMessages(os.Stdout, unsupported.Messages())
		display.PrintWarnings(os.Stdout, collector.Messages())
		printOutdatedErrorsWithHints(errs)
	}

	if len(errs) > 0 {
		// Count successful checks for partial success detection
		successCount := 0
		for _, res := range results {
			if res.err == nil && res.status != lock.InstallStatusNotConfigured {
				successCount++
			}
		}

		if successCount > 0 && outdatedContinueOnFail {
			// Partial success: some checks succeeded, some failed
			verbose.Infof("Exit code %d (partial failure): %d succeeded, %d failed with --continue-on-fail flag", errors.ExitPartialFailure, successCount, len(errs))
			return errors.NewExitError(errors.ExitPartialFailure, errors.NewPartialSuccessError(successCount, len(errs), errs))
		}

		// Complete failure (or no --continue-on-fail flag)
		verbose.Infof("Exit code %d (failure): %d packages failed, successCount=%d, continueOnFail=%v", errors.ExitFailure, len(errs), successCount, outdatedContinueOnFail)
		return errors.NewExitError(errors.ExitFailure, stderrors.Join(errs...))
	}

	verbose.Infof("Exit code %d (success): all %d packages checked successfully", errors.ExitSuccess, len(results))
	return nil
}

// getOutdatedOutputFormat determines the output format for outdated results.
//
// Parses the --output flag value and returns the corresponding format.
// If no flag is specified, defaults to table format.
//
// Returns:
//   - output.Format: Parsed format (JSON, CSV, XML, or Table)
func getOutdatedOutputFormat() output.Format {
	return output.ParseFormat(outdatedOutputFlag)
}

// printOutdatedStructured outputs outdated results in a structured format.
//
// Converts results to structured output format with package information,
// version availability, and status. Includes summary counts for outdated,
// up-to-date, and failed packages.
//
// Parameters:
//   - results: Outdated check results to output
//   - warnings: Warning messages to include
//   - errs: Error messages to include
//   - format: Output format (JSON, CSV, or XML)
//
// Returns:
//   - error: Returns error on output failure
func printOutdatedStructured(results []outdatedResult, warnings []string, errs []string, format output.Format) error {
	packages := make([]output.OutdatedPackage, 0, len(results))

	var outdatedCount, uptodateCount, failedCount int
	var hasMajor, hasMinor, hasPatch int

	for _, res := range results {
		constraintDisplay := display.FormatConstraintDisplayWithFlags(res.pkg, outdatedMajorFlag, outdatedMinorFlag, outdatedPatchFlag)

		var errStr string
		if res.err != nil {
			errStr = res.err.Error()
		}

		packages = append(packages, output.OutdatedPackage{
			Rule:             res.pkg.Rule,
			PM:               res.pkg.PackageType,
			Type:             res.pkg.Type,
			Constraint:       constraintDisplay,
			Version:          display.SafeDeclaredValue(res.pkg.Version),
			InstalledVersion: display.SafeInstalledValue(res.pkg.InstalledVersion),
			Major:            res.major,
			Minor:            res.minor,
			Patch:            res.patch,
			Status:           res.status,
			Group:            res.group,
			Name:             res.pkg.Name,
			Error:            errStr,
		})

		// Count packages with available updates by type
		if res.major != constants.PlaceholderNA {
			hasMajor++
		}
		if res.minor != constants.PlaceholderNA {
			hasMinor++
		}
		if res.patch != constants.PlaceholderNA {
			hasPatch++
		}

		switch res.status {
		case outdatedStatusOutdated:
			outdatedCount++
		case outdatedStatusUpToDate:
			uptodateCount++
		default:
			if res.err != nil || strings.HasPrefix(res.status, outdatedStatusFailed) {
				failedCount++
			}
		}
	}

	result := &output.OutdatedResult{
		Summary: output.OutdatedSummary{
			TotalPackages:    len(packages),
			OutdatedPackages: outdatedCount,
			UpToDatePackages: uptodateCount,
			FailedPackages:   failedCount,
			HasMajor:         hasMajor,
			HasMinor:         hasMinor,
			HasPatch:         hasPatch,
		},
		Packages: packages,
		Warnings: warnings,
		Errors:   errs,
	}

	return writeOutdatedResultFunc(os.Stdout, format, result)
}

// isLatestMissing checks if a package declared as "latest" has no resolved version.
//
// Parameters:
//   - p: Package to check
//   - ruleCfg: Rule configuration for latest indicators
//
// Returns:
//   - bool: True if package uses latest indicator but has no installed version
func isLatestMissing(p formats.Package, ruleCfg *config.PackageManagerCfg) bool {
	return utils.IsLatestIndicator(p.Version, p.Name, ruleCfg) && strings.EqualFold(strings.TrimSpace(p.InstalledVersion), constants.PlaceholderNA)
}

// deriveOutdatedStatus determines the display status for an outdated check result.
//
// Returns status based on available updates (Outdated), errors (Failed),
// floating constraints (Floating), or no updates available (UpToDate).
//
// Parameters:
//   - res: Outdated check result
//
// Returns:
//   - string: Status constant (StatusOutdated, StatusUpToDate, StatusFailed, etc.)
func deriveOutdatedStatus(res outdatedResult) string {
	// Preserve Floating status - these packages cannot be processed automatically
	// because their constraints (*, x, ranges) make version comparison meaningless
	if res.pkg.InstallStatus == lock.InstallStatusFloating {
		return lock.InstallStatusFloating
	}

	if res.err != nil {
		if code := outdated.ExtractExitCode(res.err); code != "" {
			return fmt.Sprintf("%s(%s)", outdatedStatusFailed, code)
		}
		return outdatedStatusFailed
	}

	if res.major != constants.PlaceholderNA || res.minor != constants.PlaceholderNA || res.patch != constants.PlaceholderNA {
		return outdatedStatusOutdated
	}

	return outdatedStatusUpToDate
}

// outdatedDisplayRow holds pre-formatted display values for a single outdated result row.
type outdatedDisplayRow struct {
	pkg               formats.Package
	constraintDisplay string
	statusDisplay     string
	major             string
	minor             string
	patch             string
	target            string
	group             string
}

// prepareOutdatedDisplayRows converts outdated results to display rows.
//
// Formats constraint and status values for each result for table output.
//
// Parameters:
//   - results: Outdated check results to format
//
// Returns:
//   - []outdatedDisplayRow: Formatted rows ready for table output
func prepareOutdatedDisplayRows(results []outdatedResult) []outdatedDisplayRow {
	rows := make([]outdatedDisplayRow, 0, len(results))

	for _, res := range results {
		constraintDisplay := display.FormatConstraintDisplayWithFlags(res.pkg, outdatedMajorFlag, outdatedMinorFlag, outdatedPatchFlag)

		rows = append(rows, outdatedDisplayRow{
			pkg:               res.pkg,
			constraintDisplay: constraintDisplay,
			statusDisplay:     display.FormatStatusWithIcon(res.status),
			major:             res.major,
			minor:             res.minor,
			patch:             res.patch,
			target:            display.SafeVersionValue(res.target, constants.PlaceholderNA),
			group:             res.group,
		})
	}

	return rows
}

// printOutdatedResults outputs outdated results in table format to stdout.
//
// Parameters:
//   - results: Outdated check results to display
//   - typeFlag: Type filter value for output context
//   - pmFlag: Package manager filter value for output context
func printOutdatedResults(results []outdatedResult, typeFlag, pmFlag string) {
	rows := prepareOutdatedDisplayRows(results)
	table := buildOutdatedTable(rows)

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
			row.major,
			row.minor,
			row.patch,
			row.statusDisplay,
			row.group,
			row.pkg.Name,
		))
	}

	fmt.Printf("\nTotal packages: %d\n", len(results))
}

// buildOutdatedTable creates a table formatter with calculated column widths.
//
// Initializes a table with package and version columns including MAJOR, MINOR,
// and PATCH columns for available updates. Conditionally includes GROUP column.
//
// Parameters:
//   - rows: Display rows to calculate widths from
//
// Returns:
//   - *output.Table: Configured table formatter ready for output
func buildOutdatedTable(rows []outdatedDisplayRow) *output.Table {
	// Extract groups to determine if GROUP column should be shown
	groups := make([]string, len(rows))
	for i, row := range rows {
		groups[i] = row.group
	}
	showGroup := output.ShouldShowGroupColumn(groups)

	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("MAJOR").
		AddColumn("MINOR").
		AddColumn("PATCH").
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
			row.major,
			row.minor,
			row.patch,
			row.statusDisplay,
			row.group,
			row.pkg.Name,
		)
	}

	return table
}

// buildOutdatedTableFromPackages creates a table formatter from package data.
//
// Creates the table before version fetching with reserved minimum widths
// for version-related columns. This allows streaming output during the
// version checking process.
//
// Parameters:
//   - packages: Packages to calculate base widths from
//
// Returns:
//   - *output.Table: Configured table formatter with reserved column widths
func buildOutdatedTableFromPackages(packages []formats.Package) *output.Table {
	// Extract groups to determine if GROUP column should be shown
	groups := make([]string, len(packages))
	for i, p := range packages {
		groups[i] = p.Group
	}
	showGroup := output.ShouldShowGroupColumn(groups)

	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumnWithMinWidth("MAJOR", 12).  // Reserve space for version numbers
		AddColumnWithMinWidth("MINOR", 12).  // Reserve space for version numbers
		AddColumnWithMinWidth("PATCH", 12).  // Reserve space for version numbers
		AddColumnWithMinWidth("STATUS", 14). // Reserve space for "🔴 Unsupported"
		AddConditionalColumn("GROUP", showGroup).
		AddColumn("NAME")

	for _, p := range packages {
		table.UpdateWidths(
			p.Rule,
			p.PackageType,
			p.Type,
			display.FormatConstraintDisplayWithFlags(p, outdatedMajorFlag, outdatedMinorFlag, outdatedPatchFlag),
			display.SafeDeclaredValue(p.Version),
			display.SafeInstalledValue(p.InstalledVersion),
			"", "", "", "", // Placeholders for MAJOR, MINOR, PATCH, STATUS (will use min widths)
			p.Group,
			p.Name,
		)
	}

	return table
}

// printOutdatedRowWithTable prints a single outdated result row.
//
// Formats and outputs one row of outdated results using the provided
// table formatter. Used for streaming output during version checking.
//
// Parameters:
//   - res: Outdated result to display
//   - table: Table formatter with column widths
func printOutdatedRowWithTable(res outdatedResult, table *output.Table) {
	fmt.Println(table.FormatRow(
		res.pkg.Rule,
		res.pkg.PackageType,
		res.pkg.Type,
		display.FormatConstraintDisplayWithFlags(res.pkg, outdatedMajorFlag, outdatedMinorFlag, outdatedPatchFlag),
		display.SafeDeclaredValue(res.pkg.Version),
		display.SafeInstalledValue(res.pkg.InstalledVersion),
		res.major,
		res.minor,
		res.patch,
		display.FormatStatusWithIcon(res.status),
		res.group,
		res.pkg.Name,
	))
}

// printOutdatedErrorsWithHints prints errors with actionable resolution hints.
//
// Formats error messages with context-aware hints to help users resolve
// common issues like missing commands or network problems.
//
// Parameters:
//   - errs: Errors to display with hints
func printOutdatedErrorsWithHints(errs []error) {
	if len(errs) == 0 {
		return
	}

	fmt.Println()
	fmt.Print(errors.FormatErrorsWithHints(errs))
}
