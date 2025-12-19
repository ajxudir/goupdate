package update

import (
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// versionsMatch compares two version strings, normalizing the 'v' prefix.
// This handles cases where one version has 'v' prefix (e.g., "v3.16.1") and the other doesn't ("3.16.1").
func versionsMatch(v1, v2 string) bool {
	normalize := func(v string) string {
		return strings.TrimPrefix(strings.TrimSpace(v), "v")
	}
	return normalize(v1) == normalize(v2)
}

// PackageUpdater is a function type for updating a package to a target version.
type PackageUpdater func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error

// ExecutionCallbacks provides callback functions for execution events.
type ExecutionCallbacks struct {
	// OnResultReady is called when a result is ready to be displayed
	OnResultReady func(res UpdateResult, dryRun bool)
	// DeriveReason derives unsupported reason for a package
	DeriveReason UnsupportedReasonDeriver
	// OnSystemTestFailure is called when a system test fails
	OnSystemTestFailure func(pkgName string, isCritical bool)
}

// ValidateUpdatedPackage validates that a package was updated successfully using drift detection.
// This is the post-update drift check that verifies the manifest and lock file were correctly updated.
func ValidateUpdatedPackage(plan *PlannedUpdate, reloadList func() ([]formats.Package, error), baseline map[string]VersionSnapshot) error {
	if reloadList == nil {
		return nil
	}

	// Suppress verbose output during reload to reduce noise
	verbose.Suppress()
	packages, err := reloadList()
	verbose.Unsuppress()

	if err != nil {
		verbose.Printf("Drift check FAILED: %s - reload error: %v\n", plan.Res.Pkg.Name, err)
		return err
	}

	key := PackageKey(plan.Res.Pkg)
	var found *formats.Package
	for idx := range packages {
		p := packages[idx]
		if PackageKey(p) == key {
			found = &p
			break
		}
	}

	if found == nil {
		verbose.Printf("Drift check FAILED: %s not found after reload\n", plan.Res.Pkg.Name)
		return fmt.Errorf("package %s (%s/%s) missing after update validation", plan.Res.Pkg.Name, plan.Res.Pkg.PackageType, plan.Res.Pkg.Rule)
	}

	if !versionsMatch(found.Version, plan.Res.Target) {
		verbose.Printf("Drift check MISMATCH: %s expected %s, got %s\n",
			plan.Res.Pkg.Name, plan.Res.Target, found.Version)
		return fmt.Errorf("version mismatch after update: expected %s, found %s", plan.Res.Target, found.Version)
	}

	if found.InstalledVersion != "" && found.InstalledVersion != constants.PlaceholderNA && !versionsMatch(found.InstalledVersion, plan.Res.Target) {
		verbose.Printf("Drift check MISMATCH: %s installed=%s, expected %s (lock file not updated)\n",
			plan.Res.Pkg.Name, found.InstalledVersion, plan.Res.Target)
		return fmt.Errorf("installed version mismatch after update: expected %s, got %s (lock file may not have been updated)", plan.Res.Target, found.InstalledVersion)
	}

	// Update the plan's package with the reloaded values for accurate display
	plan.Res.Pkg.Version = found.Version
	plan.Res.Pkg.InstalledVersion = found.InstalledVersion

	verbose.Debugf("Drift check: %s → %s ✓", plan.Res.Pkg.Name, plan.Res.Target)
	return nil
}

// ValidatePreUpdateState performs a pre-update drift check to verify the package is at its expected original version.
// This detects if another process modified the package between planning and execution.
func ValidatePreUpdateState(plan *PlannedUpdate, reloadList func() ([]formats.Package, error)) error {
	if reloadList == nil {
		return nil
	}

	// Suppress verbose output during reload to reduce noise
	verbose.Suppress()
	packages, err := reloadList()
	verbose.Unsuppress()

	if err != nil {
		return nil // Non-fatal - continue with update
	}

	key := PackageKey(plan.Res.Pkg)
	var found *formats.Package
	for idx := range packages {
		p := packages[idx]
		if PackageKey(p) == key {
			found = &p
			break
		}
	}

	if found == nil {
		return nil // Non-fatal
	}

	if !versionsMatch(found.Version, plan.Original) {
		verbose.Printf("Pre-update drift DETECTED: %s changed %s → %s (external modification?)\n",
			plan.Res.Pkg.Name, plan.Original, found.Version)
		// Update the Original to match current state so rollback works correctly
		plan.Original = found.Version
	}

	return nil
}

// RollbackPlans rolls back all applied plans to their original versions.
// Returns a combined error if any rollbacks failed, allowing callers to know if rollback was successful.
func RollbackPlans(plans []*PlannedUpdate, cfg *config.Config, workDir string, ctx *UpdateContext, groupErr error, updater PackageUpdater, dryRun, skipLock bool) error {
	verbose.Printf("Rolling back %d packages due to error: %v\n", len(plans), groupErr)
	var rollbackErrors []error

	for _, plan := range plans {
		verbose.Debugf("Rolling back %s: %s → %s", plan.Res.Pkg.Name, plan.Res.Target, plan.Original)
		rollbackErr := updater(plan.Res.Pkg, plan.Original, cfg, workDir, dryRun, skipLock)
		if rollbackErr != nil {
			wrappedErr := fmt.Errorf("%s (%s/%s) rollback failed: %w", plan.Res.Pkg.Name, plan.Res.Pkg.PackageType, plan.Res.Pkg.Rule, rollbackErr)
			ctx.AppendFailure(wrappedErr)
			rollbackErrors = append(rollbackErrors, wrappedErr)
			verbose.Printf("Rollback FAILED for %s: %v\n", plan.Res.Pkg.Name, rollbackErr)
		} else {
			verbose.Debugf("Rolled back %s to %s successfully", plan.Res.Pkg.Name, plan.Original)
			// Verify rollback with drift check
			if ctx.ReloadList != nil && !dryRun {
				driftErr := verifyRollbackDrift(plan, ctx.ReloadList)
				if driftErr != nil {
					verbose.Printf("DRIFT CHECK FAILED for %s: %v\n", plan.Res.Pkg.Name, driftErr)
					rollbackErrors = append(rollbackErrors, driftErr)
				}
			}
		}
		if plan.Res.Status == constants.StatusUpdated {
			plan.Res.Status = constants.StatusFailed
			if plan.Res.Err == nil {
				plan.Res.Err = groupErr
			}
		}
	}

	if len(rollbackErrors) > 0 {
		verbose.Printf("Rollback completed with %d errors\n", len(rollbackErrors))
		return stderrors.Join(rollbackErrors...)
	}
	verbose.Debugf("Rollback completed successfully for all %d packages", len(plans))
	return nil
}

// verifyRollbackDrift verifies that a rollback actually restored the package to its original version.
// This drift check helps detect cases where the rollback command succeeded but the manifest wasn't updated.
func verifyRollbackDrift(plan *PlannedUpdate, reloadList func() ([]formats.Package, error)) error {
	if reloadList == nil {
		return nil
	}

	// Suppress verbose output during reload to reduce noise
	verbose.Suppress()
	packages, err := reloadList()
	verbose.Unsuppress()

	if err != nil {
		verbose.Printf("Rollback drift check FAILED: %s - reload error: %v\n", plan.Res.Pkg.Name, err)
		return fmt.Errorf("drift check failed: could not reload packages: %w", err)
	}

	key := PackageKey(plan.Res.Pkg)
	var found *formats.Package
	for idx := range packages {
		p := packages[idx]
		if PackageKey(p) == key {
			found = &p
			break
		}
	}

	if found == nil {
		verbose.Printf("Rollback drift check FAILED: %s not found after reload\n", plan.Res.Pkg.Name)
		return fmt.Errorf("drift check failed: package %s missing after rollback", plan.Res.Pkg.Name)
	}

	if !versionsMatch(found.Version, plan.Original) {
		verbose.Printf("Rollback drift check MISMATCH: %s expected %s, got %s\n",
			plan.Res.Pkg.Name, plan.Original, found.Version)
		return fmt.Errorf("drift check failed: %s version mismatch after rollback (expected %s, found %s)",
			plan.Res.Pkg.Name, plan.Original, found.Version)
	}

	verbose.Debugf("Rollback drift check: %s → %s ✓", plan.Res.Pkg.Name, plan.Original)
	return nil
}

// SummarizeGroupFailure marks all packages in a group as failed.
func SummarizeGroupFailure(plans []*PlannedUpdate, groupErr error) {
	for _, plan := range plans {
		res := &plan.Res
		if res.Status == lock.InstallStatusNotConfigured || res.Status == constants.StatusConfigError || res.Status == constants.StatusSummarizeError {
			continue
		}

		res.Status = constants.StatusFailed
		if res.Err == nil {
			res.Err = groupErr
		}
	}
}

// HandleUpdateError handles errors from update operations.
func HandleUpdateError(updateErr error, res *UpdateResult, ctx *UpdateContext, deriveReason UnsupportedReasonDeriver) {
	res.Err = updateErr
	if errors.IsUnsupported(updateErr) {
		res.Status = lock.InstallStatusNotConfigured
		if ctx.Unsupported != nil {
			ctx.Unsupported.Add(res.Pkg, deriveReason(res.Pkg, ctx.Cfg, updateErr, false))
		}
		return
	}

	res.Status = constants.StatusFailed
	ctx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", res.Pkg.Name, res.Pkg.PackageType, res.Pkg.Rule, updateErr))
}

// ApplyPlannedUpdate applies a single planned update.
func ApplyPlannedUpdate(plan *PlannedUpdate, cfg *config.Config, workDir string, updater PackageUpdater, dryRun, skipLock bool) error {
	return updater(plan.Res.Pkg, plan.Res.Target, cfg, workDir, dryRun, skipLock)
}

// ShouldTrackUnsupported returns true if the status indicates the package should be tracked.
func ShouldTrackUnsupported(status string) bool {
	return strings.EqualFold(status, lock.InstallStatusNotConfigured) ||
		strings.EqualFold(status, lock.InstallStatusFloating) ||
		strings.EqualFold(status, lock.InstallStatusVersionMissing)
}

// CollectUpdateErrors collects errors from update results.
func CollectUpdateErrors(results []UpdateResult) []error {
	var errs []error
	for _, res := range results {
		if res.Err != nil && !errors.IsUnsupported(res.Err) {
			errs = append(errs, res.Err)
		}
	}
	return errs
}

// SystemTestFailure records a system test failure for later display.
type SystemTestFailure struct {
	PkgName    string
	Result     interface{ FormatResultsQuiet() string }
	IsCritical bool
}

// ExecutionOptions holds options for execution functions.
type ExecutionOptions struct {
	DryRun      bool
	SkipLockRun bool
}

// ProcessGroupedPlansLive processes all grouped plans with live output.
func ProcessGroupedPlansLive(ctx *UpdateContext, plans []*PlannedUpdate, results *[]UpdateResult, callbacks ExecutionCallbacks) {
	if len(plans) == 0 {
		return
	}

	verbose.Debugf("Processing %d packages for update", len(plans))

	start := 0
	for start < len(plans) {
		end := start + 1
		for end < len(plans) && plans[end].GroupKey == plans[start].GroupKey {
			end++
		}

		processGroupPlansLive(ctx, plans[start:end], results, callbacks)
		start = end
	}
}

// processGroupPlansLive processes a single group of plans with live output and rollback support.
//
// It performs the following operations:
//   - Step 1: Determine if group-level locking should be used (when multiple packages in group)
//   - Step 2: Process packages either with group lock or individually
//   - Step 3: Rollback all applied updates if group-level error occurs
//   - Step 4: Display system test failures if any occurred
//
// Parameters:
//   - ctx: Update context containing configuration and tracking state
//   - plans: Slice of planned updates for a single group
//   - results: Pointer to results slice to append update results
//   - callbacks: Callbacks for result display and reason derivation
//
// Returns:
//   - This function does not return a value; it modifies results in place and handles errors via context
func processGroupPlansLive(ctx *UpdateContext, plans []*PlannedUpdate, results *[]UpdateResult, callbacks ExecutionCallbacks) {
	if len(plans) == 0 {
		return
	}

	useGroupLock := len(plans) > 1
	var groupUpdateCfg *config.UpdateCfg
	if useGroupLock {
		for _, plan := range plans {
			if plan.Cfg != nil {
				groupUpdateCfg = plan.Cfg
				break
			}
		}
	}

	var groupErr error
	applied := make([]*PlannedUpdate, 0, len(plans))
	var systemTestFailures []SystemTestFailure

	if useGroupLock && !ctx.DryRun && !ctx.SkipLockRun {
		groupErr = processGroupWithGroupLock(ctx, plans, groupUpdateCfg, &applied, results, &systemTestFailures, callbacks)
	} else {
		groupErr = processGroupPerPackage(ctx, plans, &applied, results, &systemTestFailures, callbacks)
	}

	if groupErr != nil && !ctx.DryRun && useGroupLock {
		rollbackErr := RollbackPlans(applied, ctx.Cfg, ctx.WorkDir, ctx, groupErr, ctx.UpdaterFunc, ctx.DryRun, ctx.SkipLockRun)
		if rollbackErr != nil {
			groupErr = stderrors.Join(groupErr, fmt.Errorf("rollback failed: %w", rollbackErr))
		}
		SummarizeGroupFailure(plans, groupErr)
	}

	DisplaySystemTestFailures(systemTestFailures)
}

// processGroupWithGroupLock processes a group using a single lock command for all packages.
//
// It performs the following operations:
//   - Step 1: Update declared versions for all packages (skip lock commands)
//   - Step 2: Run a single group-level lock command after all updates
//   - Step 3: Validate all packages were updated correctly
//   - Step 4: Run system tests if configured
//   - Step 5: Append results and invoke display callbacks
//
// Parameters:
//   - ctx: Update context with configuration and state
//   - plans: Planned updates for packages in this group
//   - groupUpdateCfg: Update configuration for the group
//   - applied: Pointer to slice tracking successfully applied updates (for rollback)
//   - results: Pointer to results slice to append update results
//   - systemTestFailures: Pointer to slice collecting system test failures
//   - callbacks: Callbacks for result display and unsupported reason derivation
//
// Returns:
//   - error: Returns error if group lock fails or any package update fails; returns nil if all succeed
func processGroupWithGroupLock(ctx *UpdateContext, plans []*PlannedUpdate, groupUpdateCfg *config.UpdateCfg, applied *[]*PlannedUpdate, results *[]UpdateResult, systemTestFailures *[]SystemTestFailure, callbacks ExecutionCallbacks) error {
	if groupUpdateCfg == nil {
		return fmt.Errorf("no update configuration found for grouped packages; ensure at least one package has a valid update config")
	}

	var groupErr error

	for _, plan := range plans {
		res := &plan.Res
		if ShouldSkipUpdate(res) {
			handleSkippedUpdate(ctx, res, results, callbacks)
			continue
		}

		// Pre-update drift check: verify package is at expected original version
		if !ctx.DryRun {
			_ = ValidatePreUpdateState(plan, ctx.ReloadList)
		}

		updateErr := ctx.UpdaterFunc(plan.Res.Pkg, plan.Res.Target, ctx.Cfg, ctx.WorkDir, ctx.DryRun, true)
		if updateErr != nil {
			HandleUpdateError(updateErr, res, ctx, callbacks.DeriveReason)
			if !errors.IsUnsupported(updateErr) {
				groupErr = stderrors.Join(groupErr, updateErr)
			}
			continue
		}

		*applied = append(*applied, plan)
	}

	if len(*applied) > 0 && groupErr == nil && !ctx.DryRun {
		// Check if any package in the group needs -W flag (with all dependencies)
		withAllDeps := false
		for _, plan := range *applied {
			if ruleCfg, ok := ctx.Cfg.Rules[plan.Res.Pkg.Rule]; ok {
				if ruleCfg.ShouldUpdateWithAllDependencies(plan.Res.Pkg.Name) {
					withAllDeps = true
					break
				}
			}
		}
		verbose.Debugf("Post-manifest drift check: running group lock command to sync lock file")
		lockErr := RunGroupLockCommand(groupUpdateCfg, ctx.WorkDir, withAllDeps)
		if lockErr != nil {
			groupErr = lockErr
			ctx.AppendFailure(fmt.Errorf("group lock failed: %w", lockErr))
			for _, plan := range *applied {
				plan.Res.Status = constants.StatusFailed
				plan.Res.Err = lockErr
			}
		}
	}

	if groupErr == nil {
		for _, plan := range *applied {
			validateErr := ValidateUpdatedPackage(plan, ctx.ReloadList, ctx.Baseline)
			if validateErr != nil {
				plan.Res.Status = constants.StatusFailed
				plan.Res.Err = validateErr
				ctx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", plan.Res.Pkg.Name, plan.Res.Pkg.PackageType, plan.Res.Pkg.Rule, validateErr))
				groupErr = stderrors.Join(groupErr, validateErr)
			} else {
				plan.Res.Status = constants.StatusUpdated
				plan.Res.Err = nil
				RefreshAvailableVersions(plan)
			}
		}
	}

	if ctx.ShouldRunSystemTestsAfterEach() && groupErr == nil && len(*applied) > 0 {
		groupErr = runGroupSystemTests(ctx, *applied, systemTestFailures)
	}

	for _, plan := range *applied {
		if ShouldTrackUnsupported(plan.Res.Status) {
			ctx.Unsupported.Add(plan.Res.Pkg, callbacks.DeriveReason(plan.Res.Pkg, ctx.Cfg, plan.Res.Err, false))
		}
		*results = append(*results, plan.Res)
		if callbacks.OnResultReady != nil {
			callbacks.OnResultReady(plan.Res, ctx.DryRun)
		}
	}

	return groupErr
}

// processGroupPerPackage processes each package in a group individually with separate lock commands.
//
// It performs the following operations:
//   - Step 1: For each package, update declared version and run individual lock command
//   - Step 2: Validate each package after update
//   - Step 3: Run system tests after each package if configured
//   - Step 4: Append results and invoke display callbacks for each package
//
// Parameters:
//   - ctx: Update context with configuration and state
//   - plans: Planned updates for packages in this group
//   - applied: Pointer to slice tracking successfully applied updates
//   - results: Pointer to results slice to append update results
//   - systemTestFailures: Pointer to slice collecting system test failures
//   - callbacks: Callbacks for result display and unsupported reason derivation
//
// Returns:
//   - error: Returns combined error if any package updates fail; returns nil if all succeed
func processGroupPerPackage(ctx *UpdateContext, plans []*PlannedUpdate, applied *[]*PlannedUpdate, results *[]UpdateResult, systemTestFailures *[]SystemTestFailure, callbacks ExecutionCallbacks) error {
	var groupErr error

	for _, plan := range plans {
		res := &plan.Res
		if ShouldSkipUpdate(res) {
			handleSkippedUpdate(ctx, res, results, callbacks)
			continue
		}

		// Pre-update drift check: verify package is at expected original version
		if !ctx.DryRun {
			_ = ValidatePreUpdateState(plan, ctx.ReloadList)
		}

		updateErr := ApplyPlannedUpdate(plan, ctx.Cfg, ctx.WorkDir, ctx.UpdaterFunc, ctx.DryRun, ctx.SkipLockRun)
		if updateErr != nil {
			HandleUpdateError(updateErr, res, ctx, callbacks.DeriveReason)
			if !errors.IsUnsupported(updateErr) {
				groupErr = stderrors.Join(groupErr, updateErr)
			}
			appendResultAndPrint(ctx, res, results, callbacks)
			// Stop on first error unless ContinueOnError is set
			if !ctx.ContinueOnError && !errors.IsUnsupported(updateErr) {
				return groupErr
			}
			continue
		}

		*applied = append(*applied, plan)
		if !ctx.DryRun {
			validateErr := ValidateUpdatedPackage(plan, ctx.ReloadList, ctx.Baseline)
			if validateErr != nil {
				res.Status = constants.StatusFailed
				res.Err = validateErr
				ctx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", res.Pkg.Name, res.Pkg.PackageType, res.Pkg.Rule, validateErr))
				groupErr = stderrors.Join(groupErr, validateErr)
				appendResultAndPrint(ctx, res, results, callbacks)
				// Stop on first validation error unless ContinueOnError is set
				if !ctx.ContinueOnError {
					return groupErr
				}
				continue
			}
		}

		res.Status = constants.StatusUpdated
		res.Err = nil
		RefreshAvailableVersions(plan)

		if ctx.ShouldRunSystemTestsAfterEach() {
			_ = runPackageSystemTests(ctx, plan, &groupErr, systemTestFailures)
		}

		appendResultAndPrint(ctx, res, results, callbacks)
	}

	return groupErr
}

// handleSkippedUpdate handles updates that should be skipped due to status conditions.
//
// It performs the following operations:
//   - Step 1: Track unsupported packages if applicable
//   - Step 2: Append result to results slice
//   - Step 3: Invoke display callback if configured
//
// Parameters:
//   - ctx: Update context for tracking unsupported packages
//   - res: The update result to handle
//   - results: Pointer to results slice to append the result
//   - callbacks: Callbacks for result display and reason derivation
//
// Returns:
//   - This function does not return a value; it modifies results in place
func handleSkippedUpdate(ctx *UpdateContext, res *UpdateResult, results *[]UpdateResult, callbacks ExecutionCallbacks) {
	if ShouldTrackUnsupported(res.Status) {
		ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
	}
	*results = append(*results, *res)
	if callbacks.OnResultReady != nil {
		callbacks.OnResultReady(*res, ctx.DryRun)
	}
}

// appendResultAndPrint appends a result to the results slice and triggers the display callback.
//
// It performs the following operations:
//   - Step 1: Track unsupported packages if applicable
//   - Step 2: Append result to results slice
//   - Step 3: Invoke display callback to print the result
//
// Parameters:
//   - ctx: Update context for tracking unsupported packages
//   - res: The update result to append
//   - results: Pointer to results slice to append the result
//   - callbacks: Callbacks for result display and reason derivation
//
// Returns:
//   - This function does not return a value; it modifies results in place
func appendResultAndPrint(ctx *UpdateContext, res *UpdateResult, results *[]UpdateResult, callbacks ExecutionCallbacks) {
	if ShouldTrackUnsupported(res.Status) {
		ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
	}
	*results = append(*results, *res)
	if callbacks.OnResultReady != nil {
		callbacks.OnResultReady(*res, ctx.DryRun)
	}
}

// runGroupSystemTests runs system tests for a group of updated packages and handles failures.
//
// It performs the following operations:
//   - Step 1: Execute system tests using the configured runner
//   - Step 2: Attach test results to all applied package updates
//   - Step 3: Mark packages as failed if tests are critical and stop-on-fail is enabled
//   - Step 4: Track non-critical test failures for later display
//
// Parameters:
//   - ctx: Update context with system test runner configuration
//   - applied: Slice of successfully applied updates to test
//   - systemTestFailures: Pointer to slice collecting system test failures
//
// Returns:
//   - error: Returns error if critical tests fail and stop-on-fail is enabled; returns nil otherwise
func runGroupSystemTests(ctx *UpdateContext, applied []*PlannedUpdate, systemTestFailures *[]SystemTestFailure) error {
	testResult := ctx.SystemTestRunner.RunAfterUpdate()
	for _, plan := range applied {
		plan.Res.SystemTestResult = testResult
	}
	isCritical := testResult.HasCriticalFailure() && ctx.SystemTestRunner.StopOnFail()
	if isCritical {
		verbose.Printf("System tests FAILED for group (%d/%d, %v)\n",
			testResult.PassedCount(), len(testResult.Tests), testResult.TotalDuration)
		for _, plan := range applied {
			plan.Res.Status = constants.StatusFailed
			plan.Res.Err = fmt.Errorf("system tests failed: %s", testResult.Summary())
		}
		err := fmt.Errorf("system tests failed: %s", testResult.Summary())
		ctx.AppendFailure(err)
		return err
	}
	if !testResult.Passed() {
		verbose.Debugf("System tests: %d/%d passed for group (%v)",
			testResult.PassedCount(), len(testResult.Tests), testResult.TotalDuration)
		*systemTestFailures = append(*systemTestFailures, SystemTestFailure{
			PkgName:    "group",
			Result:     testResult,
			IsCritical: isCritical,
		})
	} else {
		verbose.Debugf("System tests passed for group (%d/%d, %v)",
			testResult.PassedCount(), len(testResult.Tests), testResult.TotalDuration)
	}
	return nil
}

// runPackageSystemTests runs system tests for a single package update and handles failures with rollback.
//
// It performs the following operations:
//   - Step 1: Execute system tests using the configured runner
//   - Step 2: Attach test results to the package update
//   - Step 3: Rollback package if tests are critical and stop-on-fail is enabled
//   - Step 4: Mark package as failed and track error if critical test fails
//   - Step 5: Track non-critical test failures for later display
//
// Parameters:
//   - ctx: Update context with system test runner configuration
//   - plan: The planned update that was applied
//   - groupErr: Pointer to group error to accumulate errors
//   - systemTestFailures: Pointer to slice collecting system test failures
//
// Returns:
//   - error: Returns nil; errors are tracked via context and groupErr pointer
func runPackageSystemTests(ctx *UpdateContext, plan *PlannedUpdate, groupErr *error, systemTestFailures *[]SystemTestFailure) error {
	testResult := ctx.SystemTestRunner.RunAfterUpdate()
	plan.Res.SystemTestResult = testResult
	isCritical := testResult.HasCriticalFailure() && ctx.SystemTestRunner.StopOnFail()
	if isCritical {
		verbose.Printf("System tests FAILED for %s (%d/%d, %v) - rolling back\n",
			plan.Res.Pkg.Name, testResult.PassedCount(), len(testResult.Tests), testResult.TotalDuration)
		rollbackErr := ctx.UpdaterFunc(plan.Res.Pkg, plan.Original, ctx.Cfg, ctx.WorkDir, ctx.DryRun, ctx.SkipLockRun)
		if rollbackErr != nil {
			verbose.Printf("Rollback failed for %s: %v\n", plan.Res.Pkg.Name, rollbackErr)
			ctx.AppendFailure(fmt.Errorf("%s: rollback failed: %w", plan.Res.Pkg.Name, rollbackErr))
		} else {
			// Verify rollback with drift check
			if ctx.ReloadList != nil && !ctx.DryRun {
				driftErr := verifyRollbackDrift(plan, ctx.ReloadList)
				if driftErr != nil {
					verbose.Printf("DRIFT CHECK FAILED for %s: %v\n", plan.Res.Pkg.Name, driftErr)
					ctx.AppendFailure(driftErr)
				}
			}
		}
		plan.Res.Status = constants.StatusFailed
		plan.Res.Err = fmt.Errorf("system tests failed: %s", testResult.Summary())
		ctx.AppendFailure(fmt.Errorf("%s: %w", plan.Res.Pkg.Name, plan.Res.Err))
		*groupErr = stderrors.Join(*groupErr, plan.Res.Err)
	}
	if !testResult.Passed() {
		verbose.Debugf("System tests: %d/%d passed for %s (%v)",
			testResult.PassedCount(), len(testResult.Tests), plan.Res.Pkg.Name, testResult.TotalDuration)
		*systemTestFailures = append(*systemTestFailures, SystemTestFailure{
			PkgName:    plan.Res.Pkg.Name,
			Result:     testResult,
			IsCritical: isCritical,
		})
	} else {
		verbose.Debugf("System tests passed for %s (%d/%d, %v)",
			plan.Res.Pkg.Name, testResult.PassedCount(), len(testResult.Tests), testResult.TotalDuration)
	}
	return nil
}

// DisplaySystemTestFailures displays system test failures.
func DisplaySystemTestFailures(failures []SystemTestFailure) {
	if len(failures) == 0 {
		return
	}

	fmt.Println()
	for _, failure := range failures {
		if failure.IsCritical {
			fmt.Printf("System tests failed after %s update:\n", failure.PkgName)
		} else {
			fmt.Printf("System test warning after %s update:\n", failure.PkgName)
		}
		fmt.Print(failure.Result.FormatResultsQuiet())
	}
}

// ProcessGroupedPlansWithProgress processes all grouped plans with progress indicator.
func ProcessGroupedPlansWithProgress(ctx *UpdateContext, plans []*PlannedUpdate, results *[]UpdateResult, progress ProgressReporter, callbacks ExecutionCallbacks) {
	if len(plans) == 0 {
		return
	}

	verbose.Debugf("Processing %d packages for update", len(plans))

	start := 0
	for start < len(plans) {
		end := start + 1
		for end < len(plans) && plans[end].GroupKey == plans[start].GroupKey {
			end++
		}

		processGroupPlansWithProgress(ctx, plans[start:end], results, progress, callbacks)
		start = end
	}
}

// ProgressReporter is an interface for progress reporting.
type ProgressReporter interface {
	Increment()
}

// processGroupPlansWithProgress processes a single group with progress indicator and rollback support.
//
// It performs the following operations:
//   - Step 1: Determine if group-level locking should be used
//   - Step 2: Process packages with progress reporting
//   - Step 3: Rollback all applied updates if group-level error occurs
//
// Parameters:
//   - ctx: Update context containing configuration and tracking state
//   - plans: Slice of planned updates for a single group
//   - results: Pointer to results slice to append update results
//   - progress: Progress reporter to increment after each package
//   - callbacks: Callbacks for result display and reason derivation
//
// Returns:
//   - This function does not return a value; it modifies results in place and handles errors via context
func processGroupPlansWithProgress(ctx *UpdateContext, plans []*PlannedUpdate, results *[]UpdateResult, progress ProgressReporter, callbacks ExecutionCallbacks) {
	if len(plans) == 0 {
		return
	}

	useGroupLock := len(plans) > 1
	var groupUpdateCfg *config.UpdateCfg
	if useGroupLock {
		for _, plan := range plans {
			if plan.Cfg != nil {
				groupUpdateCfg = plan.Cfg
				break
			}
		}
	}

	var groupErr error
	applied := make([]*PlannedUpdate, 0, len(plans))

	if useGroupLock && !ctx.DryRun && !ctx.SkipLockRun {
		groupErr = processGroupWithGroupLockProgress(ctx, plans, groupUpdateCfg, &applied, results, progress, callbacks)
	} else {
		groupErr = processGroupPerPackageProgress(ctx, plans, &applied, results, progress, callbacks)
	}

	if groupErr != nil && !ctx.DryRun && useGroupLock {
		rollbackErr := RollbackPlans(applied, ctx.Cfg, ctx.WorkDir, ctx, groupErr, ctx.UpdaterFunc, ctx.DryRun, ctx.SkipLockRun)
		if rollbackErr != nil {
			groupErr = stderrors.Join(groupErr, fmt.Errorf("rollback failed: %w", rollbackErr))
		}
		SummarizeGroupFailure(plans, groupErr)
	}
}

// processGroupWithGroupLockProgress processes a group using a single lock command with progress reporting.
//
// It performs the following operations:
//   - Step 1: Update declared versions for all packages
//   - Step 2: Run a single group-level lock command after all updates
//   - Step 3: Validate all packages were updated correctly
//   - Step 4: Append results and increment progress for each package
//
// Parameters:
//   - ctx: Update context with configuration and state
//   - plans: Planned updates for packages in this group
//   - groupUpdateCfg: Update configuration for the group
//   - applied: Pointer to slice tracking successfully applied updates
//   - results: Pointer to results slice to append update results
//   - progress: Progress reporter to increment after each package
//   - callbacks: Callbacks for unsupported reason derivation
//
// Returns:
//   - error: Returns error if group lock fails or any package update fails; returns nil if all succeed
func processGroupWithGroupLockProgress(ctx *UpdateContext, plans []*PlannedUpdate, groupUpdateCfg *config.UpdateCfg, applied *[]*PlannedUpdate, results *[]UpdateResult, progress ProgressReporter, callbacks ExecutionCallbacks) error {
	if groupUpdateCfg == nil {
		return fmt.Errorf("no update configuration found for grouped packages; ensure at least one package has a valid update config")
	}

	var groupErr error

	for _, plan := range plans {
		res := &plan.Res
		if ShouldSkipUpdate(res) {
			if ShouldTrackUnsupported(res.Status) {
				ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
			}
			*results = append(*results, *res)
			if progress != nil {
				progress.Increment()
			}
			continue
		}

		// Pre-update drift check: verify package is at expected original version
		if !ctx.DryRun {
			_ = ValidatePreUpdateState(plan, ctx.ReloadList)
		}

		updateErr := ctx.UpdaterFunc(plan.Res.Pkg, plan.Res.Target, ctx.Cfg, ctx.WorkDir, ctx.DryRun, true)
		if updateErr != nil {
			HandleUpdateError(updateErr, res, ctx, callbacks.DeriveReason)
			if !errors.IsUnsupported(updateErr) {
				groupErr = stderrors.Join(groupErr, updateErr)
			}
			continue
		}

		*applied = append(*applied, plan)
	}

	if len(*applied) > 0 && groupErr == nil && !ctx.DryRun {
		// Check if any package in the group needs -W flag (with all dependencies)
		withAllDeps := false
		for _, plan := range *applied {
			if ruleCfg, ok := ctx.Cfg.Rules[plan.Res.Pkg.Rule]; ok {
				if ruleCfg.ShouldUpdateWithAllDependencies(plan.Res.Pkg.Name) {
					withAllDeps = true
					break
				}
			}
		}
		verbose.Debugf("Post-manifest drift check: running group lock command to sync lock file")
		lockErr := RunGroupLockCommand(groupUpdateCfg, ctx.WorkDir, withAllDeps)
		if lockErr != nil {
			groupErr = lockErr
			ctx.AppendFailure(fmt.Errorf("group lock failed: %w", lockErr))
			for _, plan := range *applied {
				plan.Res.Status = constants.StatusFailed
				plan.Res.Err = lockErr
			}
		}
	}

	if groupErr == nil {
		for _, plan := range *applied {
			validateErr := ValidateUpdatedPackage(plan, ctx.ReloadList, ctx.Baseline)
			if validateErr != nil {
				plan.Res.Status = constants.StatusFailed
				plan.Res.Err = validateErr
				ctx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", plan.Res.Pkg.Name, plan.Res.Pkg.PackageType, plan.Res.Pkg.Rule, validateErr))
				groupErr = stderrors.Join(groupErr, validateErr)
			} else {
				plan.Res.Status = constants.StatusUpdated
				plan.Res.Err = nil
				RefreshAvailableVersions(plan)
			}
			if ShouldTrackUnsupported(plan.Res.Status) {
				ctx.Unsupported.Add(plan.Res.Pkg, callbacks.DeriveReason(plan.Res.Pkg, ctx.Cfg, plan.Res.Err, false))
			}
			*results = append(*results, plan.Res)
			if progress != nil {
				progress.Increment()
			}
		}
	} else {
		for _, plan := range *applied {
			if ShouldTrackUnsupported(plan.Res.Status) {
				ctx.Unsupported.Add(plan.Res.Pkg, callbacks.DeriveReason(plan.Res.Pkg, ctx.Cfg, plan.Res.Err, false))
			}
			*results = append(*results, plan.Res)
			if progress != nil {
				progress.Increment()
			}
		}
	}

	return groupErr
}

// processGroupPerPackageProgress processes each package individually with separate lock commands and progress reporting.
//
// It performs the following operations:
//   - Step 1: For each package, update declared version and run individual lock command
//   - Step 2: Validate each package after update
//   - Step 3: Append results and increment progress for each package
//
// Parameters:
//   - ctx: Update context with configuration and state
//   - plans: Planned updates for packages in this group
//   - applied: Pointer to slice tracking successfully applied updates
//   - results: Pointer to results slice to append update results
//   - progress: Progress reporter to increment after each package
//   - callbacks: Callbacks for unsupported reason derivation
//
// Returns:
//   - error: Returns combined error if any package updates fail; returns nil if all succeed
func processGroupPerPackageProgress(ctx *UpdateContext, plans []*PlannedUpdate, applied *[]*PlannedUpdate, results *[]UpdateResult, progress ProgressReporter, callbacks ExecutionCallbacks) error {
	var groupErr error

	for _, plan := range plans {
		res := &plan.Res
		if ShouldSkipUpdate(res) {
			if ShouldTrackUnsupported(res.Status) {
				ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
			}
			*results = append(*results, *res)
			if progress != nil {
				progress.Increment()
			}
			continue
		}

		// Pre-update drift check: verify package is at expected original version
		if !ctx.DryRun {
			_ = ValidatePreUpdateState(plan, ctx.ReloadList)
		}

		updateErr := ApplyPlannedUpdate(plan, ctx.Cfg, ctx.WorkDir, ctx.UpdaterFunc, ctx.DryRun, ctx.SkipLockRun)
		if updateErr != nil {
			HandleUpdateError(updateErr, res, ctx, callbacks.DeriveReason)
			if !errors.IsUnsupported(updateErr) {
				groupErr = stderrors.Join(groupErr, updateErr)
			}
			if ShouldTrackUnsupported(res.Status) {
				ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
			}
			*results = append(*results, *res)
			if progress != nil {
				progress.Increment()
			}
			// Stop on first error unless ContinueOnError is set
			if !ctx.ContinueOnError && !errors.IsUnsupported(updateErr) {
				return groupErr
			}
			continue
		}

		*applied = append(*applied, plan)
		if !ctx.DryRun {
			validateErr := ValidateUpdatedPackage(plan, ctx.ReloadList, ctx.Baseline)
			if validateErr != nil {
				res.Status = constants.StatusFailed
				res.Err = validateErr
				ctx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", res.Pkg.Name, res.Pkg.PackageType, res.Pkg.Rule, validateErr))
				groupErr = stderrors.Join(groupErr, validateErr)
				if ShouldTrackUnsupported(res.Status) {
					ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
				}
				*results = append(*results, *res)
				if progress != nil {
					progress.Increment()
				}
				// Stop on first validation error unless ContinueOnError is set
				if !ctx.ContinueOnError {
					return groupErr
				}
				continue
			}
		}

		res.Status = constants.StatusUpdated
		res.Err = nil
		RefreshAvailableVersions(plan)

		if ShouldTrackUnsupported(res.Status) {
			ctx.Unsupported.Add(res.Pkg, callbacks.DeriveReason(res.Pkg, ctx.Cfg, res.Err, false))
		}
		*results = append(*results, *res)
		if progress != nil {
			progress.Increment()
		}
	}

	return groupErr
}
