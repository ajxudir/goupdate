package cmd

import (
	"bufio"
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/filtering"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/preflight"
	"github.com/ajxudir/goupdate/pkg/supervision"
	"github.com/ajxudir/goupdate/pkg/systemtest"
	"github.com/ajxudir/goupdate/pkg/update"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/ajxudir/goupdate/pkg/warnings"
	"github.com/spf13/cobra"
)

// CLI flags
var (
	updateTypeFlag           string
	updatePMFlag             string
	updateRuleFlag           string
	updateNameFlag           string
	updateGroupFlag          string
	updateConfigFlag         string
	updateDirFlag            string
	updateFileFlag           string
	updateMajorFlag          bool
	updateMinorFlag          bool
	updatePatchFlag          bool
	updateIncrementalFlag    bool
	updateDryRunFlag         bool
	updateSkipLockRun        bool
	updateYesFlag            bool
	updateNoTimeoutFlag      bool
	updateContinueOnFail     bool
	updateSkipPreflight      bool
	updateOutputFlag         string
	updateSkipSystemTests    bool
	updateSystemTestModeFlag string
)

// Testable function variables
var updatePackageFunc = update.UpdatePackage
var resolveUpdateCfgFunc = update.ResolveUpdateCfg
var stdinReaderFunc = func() *bufio.Reader { return bufio.NewReader(os.Stdin) }
var writeUpdateResultFunc = output.WriteUpdateResult

// ValidationRunner is an interface for running validation tests.
// This allows mocking in tests.
type ValidationRunner interface {
	RunValidation() *systemtest.Result
	StopOnFail() bool
}

var updateCmd = &cobra.Command{
	Use:   "update [file...]",
	Short: "Apply package updates",
	Long:  `Plans and applies updates by combining constraint-aware selections with configured install commands.`,
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().StringVarP(&updateTypeFlag, "type", "t", "all", "Filter by type (comma-separated): all,prod,dev")
	updateCmd.Flags().StringVarP(&updatePMFlag, "package-manager", "p", "all", "Filter by package manager (comma-separated)")
	updateCmd.Flags().StringVarP(&updateRuleFlag, "rule", "r", "all", "Filter by rule (comma-separated)")
	updateCmd.Flags().StringVarP(&updateNameFlag, "name", "n", "", "Filter by package name (comma-separated)")
	updateCmd.Flags().StringVarP(&updateGroupFlag, "group", "g", "", "Filter by group (comma-separated)")
	updateCmd.Flags().StringVarP(&updateConfigFlag, "config", "c", "", "Config file path")
	updateCmd.Flags().StringVarP(&updateDirFlag, "directory", "d", ".", "Directory to scan")
	updateCmd.Flags().StringVarP(&updateFileFlag, "file", "f", "", "Filter by file path patterns (comma-separated, supports globs)")
	updateCmd.Flags().BoolVar(&updateMajorFlag, "major", false, "Force major upgrades (cascade to minor/patch)")
	updateCmd.Flags().BoolVar(&updateMinorFlag, "minor", false, "Force minor upgrades (cascade to patch)")
	updateCmd.Flags().BoolVar(&updatePatchFlag, "patch", false, "Force patch upgrades")
	updateCmd.Flags().BoolVar(&updateDryRunFlag, "dry-run", false, "Plan updates without writing files")
	updateCmd.Flags().BoolVar(&updateSkipLockRun, "skip-lock", false, "Skip running lock/install command")
	updateCmd.Flags().BoolVarP(&updateYesFlag, "yes", "y", false, "Skip confirmation prompt")
	updateCmd.Flags().BoolVar(&updateNoTimeoutFlag, "no-timeout", false, "Disable command timeouts")
	updateCmd.Flags().BoolVar(&updateContinueOnFail, "continue-on-fail", false, "Continue processing remaining packages after failures")
	updateCmd.Flags().BoolVar(&updateIncrementalFlag, "incremental", false, "Force incremental updates (one version step at a time)")
	updateCmd.Flags().BoolVar(&updateSkipPreflight, "skip-preflight", false, "Skip pre-flight command validation")
	updateCmd.Flags().StringVarP(&updateOutputFlag, "output", "o", "", "Output format: json, csv, xml (default: table)")
	updateCmd.Flags().BoolVar(&updateSkipSystemTests, "skip-system-tests", false, "Skip all system tests (preflight and validation)")
	updateCmd.Flags().StringVar(&updateSystemTestModeFlag, "system-test-mode", "", "Override system test run mode: after_each, after_all, none")
}

// runUpdate executes the update command to apply package updates.
//
// Plans and applies updates by finding available versions, creating an update
// plan, and executing updates with optional system test validation.
//
// Parameters:
//   - cmd: Cobra command instance
//   - args: Optional file paths to update (empty to auto-detect)
//
// Returns:
//   - error: Returns ExitError with appropriate code on failure
func runUpdate(cmd *cobra.Command, args []string) error {
	// Validate flag compatibility before proceeding
	outputFormat := output.ParseFormat(updateOutputFlag)
	if err := output.ValidateStructuredOutputFlags(outputFormat, verboseFlag); err != nil {
		return err
	}
	if err := output.ValidateUpdateStructuredFlags(outputFormat, updateYesFlag, updateDryRunFlag); err != nil {
		return err
	}

	collector := &display.WarningCollector{}
	restoreWarnings := warnings.SetWarningWriter(collector)
	defer restoreWarnings()
	unsupported := supervision.NewUnsupportedTracker()

	workDir := updateDirFlag

	cfg, err := loadAndValidateConfig(updateConfigFlag, workDir)
	if err != nil {
		return err
	}

	workDir = resolveWorkingDir(workDir, cfg)
	cfg.WorkingDir = workDir
	cfg.NoTimeout = updateNoTimeoutFlag

	packages, err := getPackagesFunc(cfg, args, workDir)
	if err != nil {
		return err
	}

	// Apply filters
	if updateFileFlag != "" {
		packages = filtering.FilterPackagesByFile(packages, updateFileFlag, workDir)
	}
	packages = filtering.FilterPackagesWithFilters(packages, updateTypeFlag, updatePMFlag, updateRuleFlag, updateNameFlag, "")
	packages, err = applyInstalledVersionsFunc(packages, cfg, workDir)
	if err != nil {
		return err
	}
	packages = filtering.ApplyPackageGroups(packages, cfg)
	packages = filtering.FilterByGroup(packages, updateGroupFlag)

	for _, p := range packages {
		if update.ShouldTrackUnsupported(p.InstallStatus) {
			unsupported.Add(p, supervision.DeriveUnsupportedReason(p, cfg, nil, false))
		}
	}

	if len(packages) == 0 {
		if output.IsStructuredFormat(outputFormat) {
			return printUpdateStructuredOutput(nil, collector.Messages(), nil, outputFormat)
		}
		display.PrintNoPackagesMessageWithFilters(os.Stdout, updateTypeFlag, updatePMFlag, updateRuleFlag)
		return nil
	}

	// Run pre-flight validation
	if !updateSkipPreflight {
		validation := preflight.ValidatePackages(packages, cfg)
		if validation.HasErrors() {
			verbose.Infof("Exit code %d (config error): preflight validation failed - %s", errors.ExitConfigError, validation.ErrorMessage())
			return errors.NewExitError(errors.ExitConfigError, fmt.Errorf("%s\n  💡 Options:\n     --skip-preflight     Bypass validation if commands are available through other means\n     --rule <name>        Filter to specific rules (e.g., --rule npm)\n     enabled: false       Disable unused rules in your config file", validation.ErrorMessage()))
		}
	}

	// Create system test runner and run preflight tests
	systemTestRunner := createSystemTestRunner(cfg, workDir)
	if err := runPreflightTests(systemTestRunner); err != nil {
		return err
	}

	// Build selection flags
	selection := outdated.UpdateSelectionFlags{Major: updateMajorFlag, Minor: updateMinorFlag, Patch: updatePatchFlag}

	// Resolve and build plans
	resolved := update.ResolvePackagePlans(packages, cfg, resolveUpdateCfgFunc)
	update.SortResolvedPlans(resolved)
	resolvedPkgs := update.ExtractPackagesFromPlans(resolved)
	baseline := update.SnapshotVersions(packages)

	// Build context for cancellation support
	cmdCtx := context.Background()
	if cmd != nil && cmd.Context() != nil {
		cmdCtx = cmd.Context()
	}

	// Create update context
	updateCtx := update.NewUpdateContext(cfg, workDir, unsupported).
		WithFlags(updateDryRunFlag, updateContinueOnFail, updateSkipLockRun).
		WithBaseline(baseline).
		WithSystemTestRunner(systemTestRunner).
		WithSelection(selection).
		WithSkipSystemTests(updateSkipSystemTests).
		WithIncrementalMode(updateIncrementalFlag).
		WithUpdaterFunc(updatePackageFunc).
		WithReloadList(func() ([]formats.Package, error) {
			return reloadPackages(cfg, args, workDir, unsupported)
		})

	// Build grouped plans
	opts := update.PlanningOptions{IncrementalMode: updateIncrementalFlag}
	groupedPlans := update.BuildGroupedPlans(cmdCtx, resolved, updateCtx, opts, listNewerVersionsFunc, supervision.DeriveUnsupportedReason)

	// Calculate column widths
	table := update.BuildUpdateTableFromPackages(resolvedPkgs, selection)
	pendingUpdates := update.CountPendingUpdates(groupedPlans)

	useStructuredOutput := output.IsStructuredFormat(outputFormat)

	// Show preview and confirm for non-dry-run updates
	if !updateDryRunFlag && !useStructuredOutput && pendingUpdates > 0 {
		update.PrintUpdatePreview(groupedPlans, table, selection)

		if !confirmUpdate(pendingUpdates) {
			return nil
		}
		fmt.Println()
	}

	var results []update.UpdateResult
	updateCtx.WithTable(table)

	// Create callbacks for live output
	callbacks := update.ExecutionCallbacks{
		OnResultReady: func(res update.UpdateResult, dryRun bool) {
			update.PrintUpdateRow(res, table, dryRun, selection)
		},
		DeriveReason: supervision.DeriveUnsupportedReason,
	}

	if useStructuredOutput {
		// Process without progress indicator - structured output suppresses stderr
		// Progress messages are only shown in table (interactive) mode
		update.ProcessGroupedPlansWithProgress(updateCtx, groupedPlans, &results, nil, callbacks)

		var errStrings []string
		for _, e := range updateCtx.Failures {
			errStrings = append(errStrings, e.Error())
		}
		if err := printUpdateStructuredOutput(results, collector.Messages(), errStrings, outputFormat); err != nil {
			return err
		}
	} else {
		// Print header and process with live output
		fmt.Println(table.HeaderRow())
		fmt.Println(table.SeparatorRow())
		_ = os.Stdout.Sync()

		update.ProcessGroupedPlansLive(updateCtx, groupedPlans, &results, callbacks)

		fmt.Printf("\nTotal packages: %d\n", len(results))

		// Run after_all system tests
		var afterAllTestResult *systemtest.Result
		if systemTestRunner != nil && systemTestRunner.ShouldRunAfterAll() && !updateSkipSystemTests && !updateDryRunFlag {
			var afterAllErr error
			afterAllTestResult, afterAllErr = runAfterAllValidation(systemTestRunner, results, updateCtx)
			if afterAllErr != nil {
				updateCtx.AppendFailure(afterAllErr)
			}
		}

		// Print summaries
		update.PrintUpdateSummary(results, updateDryRunFlag, wrapSystemTestResult(afterAllTestResult))
		display.PrintUnsupportedMessages(os.Stdout, unsupported.Messages())
		display.PrintWarnings(os.Stdout, collector.Messages())
		update.PrintUpdateErrorsWithHints(updateCtx.Failures, errors.EnhanceErrorWithHint)
	}

	return handleUpdateResult(results, updateCtx)
}

// confirmUpdate prompts the user to confirm the update.
//
// Skips prompt if --yes flag is set. Reads user input from stdin.
//
// Parameters:
//   - pendingUpdates: Number of packages pending update
//
// Returns:
//   - bool: True if user confirms or --yes flag is set
func confirmUpdate(pendingUpdates int) bool {
	if updateYesFlag {
		fmt.Printf("\n%d package(s) will be updated. Proceeding (--yes)...\n", pendingUpdates)
		return true
	}

	fmt.Printf("\n%d package(s) will be updated. Continue? [y/N]: ", pendingUpdates)
	reader := stdinReaderFunc()
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("\nUpdate cancelled (input not available).")
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Update cancelled.")
		return false
	}
	return true
}

// createSystemTestRunner creates a system test runner based on configuration.
//
// Returns nil if no system tests are configured. Applies --system-test-mode
// flag override if specified.
//
// Parameters:
//   - cfg: Configuration containing system test settings
//   - workDir: Working directory for test execution
//
// Returns:
//   - *systemtest.Runner: Test runner instance (nil if not configured)
func createSystemTestRunner(cfg *config.Config, workDir string) *systemtest.Runner {
	if cfg.SystemTests == nil {
		return nil
	}

	systemTestsCfg := cfg.SystemTests
	if updateSystemTestModeFlag != "" {
		overrideCfg := *systemTestsCfg
		overrideCfg.RunMode = updateSystemTestModeFlag
		systemTestsCfg = &overrideCfg
	}

	return systemtest.NewRunner(systemTestsCfg, workDir, updateNoTimeoutFlag, false)
}

// runPreflightTests runs preflight system tests if configured.
//
// Skips if runner is nil, preflight is not configured, --skip-system-tests
// flag is set, or running in dry-run mode.
//
// Parameters:
//   - runner: System test runner instance
//
// Returns:
//   - error: Returns ExitError if tests fail critically with stop_on_fail
func runPreflightTests(runner *systemtest.Runner) error {
	if runner == nil || !runner.ShouldRunPreflight() || updateSkipSystemTests || updateDryRunFlag {
		return nil
	}

	preflightResult := runner.RunPreflight()

	if !preflightResult.Passed() || verbose.IsEnabled() {
		fmt.Println()
		fmt.Println("Running system tests (preflight)...")
		if verbose.IsEnabled() {
			fmt.Print(preflightResult.FormatResults())
		} else {
			fmt.Print(preflightResult.FormatResultsQuiet())
		}
	}

	if preflightResult.HasCriticalFailure() && runner.StopOnFail() {
		verbose.Infof("Exit code %d (config error): system tests failed before updates - %s", errors.ExitConfigError, preflightResult.Summary())
		return errors.NewExitError(errors.ExitConfigError, fmt.Errorf("system tests failed before updates:\n%s\n  💡 Options:\n     --skip-system-tests  Skip system tests\n     --dry-run           Preview updates without running tests", preflightResult.Summary()))
	}

	if preflightResult.Passed() && verbose.IsEnabled() {
		fmt.Println("All system tests passed. Proceeding with updates...")
	} else if !preflightResult.Passed() {
		fmt.Printf("Warning: %s (continuing due to continue_on_fail settings)\n", preflightResult.Summary())
	}

	return nil
}

// runAfterAllValidation runs system tests after all updates.
//
// Only runs if there were successful updates. Reports failures and lists
// packages that may have caused issues.
//
// Parameters:
//   - runner: Validation runner interface
//   - results: Update results to check for successful updates
//   - ctx: Update context for failure tracking
//
// Returns:
//   - *systemtest.Result: Test results (nil if no tests run)
//   - error: Returns error if tests fail critically with stop_on_fail
func runAfterAllValidation(runner ValidationRunner, results []update.UpdateResult, ctx *update.UpdateContext) (*systemtest.Result, error) {
	updatedCount := 0
	for _, res := range results {
		if res.Status == constants.StatusUpdated {
			updatedCount++
		}
	}

	if updatedCount == 0 {
		return nil, nil
	}

	validationResult := runner.RunValidation()

	if !validationResult.Passed() || verbose.IsEnabled() {
		fmt.Println()
		fmt.Println("Running system tests (validation)...")
		if verbose.IsEnabled() {
			fmt.Print(validationResult.FormatResults())
		} else {
			fmt.Print(validationResult.FormatResultsQuiet())
		}
	}

	if validationResult.HasCriticalFailure() && runner.StopOnFail() {
		fmt.Println()
		fmt.Println("⚠ System tests failed after updates!")
		fmt.Println()
		fmt.Println("Updated packages that may have caused issues:")
		for _, res := range results {
			if res.Status == constants.StatusUpdated {
				fmt.Printf("  • %s %s → %s\n", res.Pkg.Name, update.SafeFromVersion(res), res.Target)
			}
		}
		fmt.Println()
		fmt.Println("Consider rolling back changes or investigating the failures.")
		return validationResult, fmt.Errorf("system tests failed after updates: %s", validationResult.Summary())
	} else if validationResult.Passed() && verbose.IsEnabled() {
		fmt.Println("All system tests passed. Updates validated successfully.")
	} else if !validationResult.Passed() {
		fmt.Printf("Warning: %s (continuing due to continue_on_fail settings)\n", validationResult.Summary())
	}

	return validationResult, nil
}

// reloadPackages reloads and filters packages for validation.
//
// Re-parses package files to get updated versions after updates are applied.
// Applies the same filters used in the original update command.
//
// Parameters:
//   - cfg: Configuration for parsing
//   - args: Original file arguments
//   - workDir: Working directory
//   - unsupported: Tracker for unsupported packages
//
// Returns:
//   - []formats.Package: Refreshed package list
//   - error: Returns error on parsing failure
func reloadPackages(cfg *config.Config, args []string, workDir string, _ *supervision.UnsupportedTracker) ([]formats.Package, error) {
	refreshed, err := getPackagesFunc(cfg, args, workDir)
	if err != nil {
		return nil, err
	}

	refreshed = filtering.FilterPackagesWithFilters(refreshed, updateTypeFlag, updatePMFlag, updateRuleFlag, updateNameFlag, "")
	refreshed, err = applyInstalledVersionsFunc(refreshed, cfg, workDir)
	if err != nil {
		return nil, err
	}
	refreshed = filtering.ApplyPackageGroups(refreshed, cfg)
	refreshed = filtering.FilterByGroup(refreshed, updateGroupFlag)

	// NOTE: Do not add to unsupported tracker here - it's already done during
	// initial package loading. Reloading packages after updates should not
	// re-count unsupported packages (would cause inflated counts).

	return refreshed, nil
}

// printUpdateStructuredOutput outputs results in structured format.
//
// Delegates to the update package's structured output function with
// appropriate flags and selection settings.
//
// Parameters:
//   - results: Update results to output
//   - warnings: Warning messages to include
//   - errs: Error messages to include
//   - format: Output format (JSON, CSV, XML)
//
// Returns:
//   - error: Returns error on output failure
func printUpdateStructuredOutput(results []update.UpdateResult, warnings []string, errs []string, format output.Format) error {
	selection := outdated.UpdateSelectionFlags{Major: updateMajorFlag, Minor: updateMinorFlag, Patch: updatePatchFlag}
	return update.PrintUpdateStructured(results, warnings, errs, format, updateDryRunFlag, selection, writeUpdateResultFunc)
}

// handleUpdateResult handles the final result of the update operation.
//
// Returns appropriate exit error based on success/failure count and
// --continue-on-fail flag setting.
//
// Parameters:
//   - results: Update results for success counting
//   - ctx: Update context containing failure list
//
// Returns:
//   - error: Returns nil on full success, ExitError on any failures
func handleUpdateResult(results []update.UpdateResult, ctx *update.UpdateContext) error {
	if len(ctx.Failures) == 0 {
		verbose.Infof("Exit code %d (success): all %d packages processed successfully", errors.ExitSuccess, len(results))
		return nil
	}

	successCount := 0
	for _, res := range results {
		if res.Status == constants.StatusUpdated || res.Status == constants.StatusPlanned {
			successCount++
		}
	}

	// Log detailed failure info in verbose mode
	if verbose.IsEnabled() {
		fmt.Fprintln(os.Stderr, "\nFailure details:")
		for i, err := range ctx.Failures {
			fmt.Fprintf(os.Stderr, "  [%d] %v\n", i+1, err)
		}
	}

	// Always log exit code reason for diagnostics
	if successCount > 0 && updateContinueOnFail {
		verbose.Infof("Exit code %d (partial failure): %d succeeded, %d failed with --continue-on-fail flag", errors.ExitPartialFailure, successCount, len(ctx.Failures))
		fmt.Fprintf(os.Stderr, "Exit code 1: %d succeeded, %d failed (partial failure with --continue-on-fail)\n", successCount, len(ctx.Failures))
		return errors.NewExitError(errors.ExitPartialFailure, errors.NewPartialSuccessError(successCount, len(ctx.Failures), ctx.Failures))
	}

	verbose.Infof("Exit code %d (failure): %d packages failed, successCount=%d, continueOnFail=%v", errors.ExitFailure, len(ctx.Failures), successCount, updateContinueOnFail)
	fmt.Fprintf(os.Stderr, "Exit code 2: %d failed\n", len(ctx.Failures))
	return errors.NewExitError(errors.ExitFailure, stderrors.Join(ctx.Failures...))
}

// systemTestResultWrapper wraps *systemtest.Result to implement SystemTestResultFormatter.
//
// Provides an adapter between the concrete systemtest.Result type and
// the update package's interface for test result formatting.
type systemTestResultWrapper struct {
	result *systemtest.Result
}

// wrapSystemTestResult wraps a systemtest.Result for use with the update package.
//
// Parameters:
//   - r: Result to wrap (may be nil)
//
// Returns:
//   - update.SystemTestResultFormatter: Wrapped result (nil if input is nil)
func wrapSystemTestResult(r *systemtest.Result) update.SystemTestResultFormatter {
	if r == nil {
		return nil
	}
	return &systemTestResultWrapper{result: r}
}

// TestCount returns the total number of tests in the result.
//
// Returns:
//   - int: Total count of tests (passed and failed)
func (w *systemTestResultWrapper) TestCount() int {
	return len(w.result.Tests)
}

// PassedCount returns the number of tests that passed.
//
// Returns:
//   - int: Count of tests with passing status
func (w *systemTestResultWrapper) PassedCount() int {
	return w.result.PassedCount()
}

// Passed returns true if all tests passed.
//
// Returns:
//   - bool: true if all tests passed; false if any test failed
func (w *systemTestResultWrapper) Passed() bool {
	return w.result.Passed()
}

// TotalDuration returns the total time taken for all tests.
//
// Returns:
//   - time.Duration: Aggregate duration of all test executions
func (w *systemTestResultWrapper) TotalDuration() time.Duration {
	return w.result.TotalDuration
}

// Tests returns information about each individual test.
//
// Creates wrapper objects for each test result to satisfy the SystemTestInfo interface.
//
// Returns:
//   - []update.SystemTestInfo: Slice of wrapped test results
func (w *systemTestResultWrapper) Tests() []update.SystemTestInfo {
	tests := make([]update.SystemTestInfo, len(w.result.Tests))
	for i, t := range w.result.Tests {
		tests[i] = &systemTestInfoWrapper{test: &t}
	}
	return tests
}

// systemTestInfoWrapper wraps *systemtest.TestResult to implement SystemTestInfo.
type systemTestInfoWrapper struct {
	test *systemtest.TestResult
}

// GetName returns the test name.
//
// Returns:
//   - string: Identifier for this test
func (w *systemTestInfoWrapper) GetName() string {
	return w.test.Name
}

// GetPassed returns whether the test passed.
//
// Returns:
//   - bool: true if test passed; false if test failed
func (w *systemTestInfoWrapper) GetPassed() bool {
	return w.test.Passed
}

// GetDuration returns how long the test took.
//
// Returns:
//   - time.Duration: Execution time of this test
func (w *systemTestInfoWrapper) GetDuration() time.Duration {
	return w.test.Duration
}

// GetOutput returns the test output.
//
// Returns:
//   - string: Combined stdout/stderr from test execution
func (w *systemTestInfoWrapper) GetOutput() string {
	return w.test.Output
}
