package update

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/systemtest"
	"github.com/ajxudir/goupdate/pkg/utils"
)

// UpdateResult holds the result of an update operation for a single package.
type UpdateResult struct {
	Pkg               formats.Package
	Target            string
	Status            string
	Err               error
	Available         []string
	Group             string
	Major             string             // Latest major version available
	Minor             string             // Latest minor version available
	Patch             string             // Latest patch version available
	OriginalInstalled string             // Original installed version before update (for summary display)
	SystemTestResult  *systemtest.Result // System test results for this package (if run)
}

// PlannedUpdate holds the plan for updating a single package.
type PlannedUpdate struct {
	Cfg                  *config.UpdateCfg
	Res                  UpdateResult
	Original             string
	GroupKey             string
	VersionsInConstraint []string              // All versions within constraint (for post-update refresh)
	Versioning           *config.VersioningCfg // Versioning config for re-summarizing
	Incremental          bool                  // Whether incremental mode is used
}

// ResolvedUpdatePlan holds the resolved configuration for a package update.
type ResolvedUpdatePlan struct {
	Pkg formats.Package
	Cfg *config.UpdateCfg
	Err error
}

// PlanningOptions holds configuration options for the planning phase.
type PlanningOptions struct {
	// IncrementalMode forces incremental updates for all packages
	IncrementalMode bool
}

// VersionLister is a function type for listing newer versions of a package.
type VersionLister func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error)

// ConfigResolver is a function type for resolving update configuration for a package.
type ConfigResolver func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error)

// UnsupportedReasonDeriver is a function type for deriving unsupported reasons.
type UnsupportedReasonDeriver func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string

// ResolvePackagePlans resolves update configurations for all packages.
func ResolvePackagePlans(packages []formats.Package, cfg *config.Config, resolver ConfigResolver) []ResolvedUpdatePlan {
	resolved := make([]ResolvedUpdatePlan, 0, len(packages))
	for _, p := range packages {
		cfgForPkg, cfgErr := resolver(p, cfg)
		resolvedPkg := p
		if cfgErr == nil {
			resolvedPkg.Group = NormalizeUpdateGroup(cfgForPkg, p)
		} else {
			resolvedPkg.Group = NormalizeUpdateGroup(nil, p)
		}
		resolved = append(resolved, ResolvedUpdatePlan{Pkg: resolvedPkg, Cfg: cfgForPkg, Err: cfgErr})
	}
	return resolved
}

// SortResolvedPlans sorts the resolved plans by rule, package type, group, type, and name.
func SortResolvedPlans(resolved []ResolvedUpdatePlan) {
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].Pkg.Rule != resolved[j].Pkg.Rule {
			return resolved[i].Pkg.Rule < resolved[j].Pkg.Rule
		}
		if resolved[i].Pkg.PackageType != resolved[j].Pkg.PackageType {
			return resolved[i].Pkg.PackageType < resolved[j].Pkg.PackageType
		}
		if cmp := CompareGroups(resolved[i].Pkg.Group, resolved[j].Pkg.Group); cmp != 0 {
			return cmp < 0
		}
		if resolved[i].Pkg.Type != resolved[j].Pkg.Type {
			return resolved[i].Pkg.Type < resolved[j].Pkg.Type
		}
		return resolved[i].Pkg.Name < resolved[j].Pkg.Name
	})
}

// CompareGroups compares two group names for sorting.
// Groups with names sort before empty groups.
func CompareGroups(a, b string) int {
	aVal := strings.TrimSpace(a)
	bVal := strings.TrimSpace(b)

	aHas := aVal != ""
	bHas := bVal != ""

	if aHas && !bHas {
		return -1
	}
	if bHas && !aHas {
		return 1
	}

	if aVal == bVal {
		return 0
	}

	if aVal < bVal {
		return -1
	}

	return 1
}

// ExtractPackagesFromPlans extracts the packages from resolved plans.
func ExtractPackagesFromPlans(resolved []ResolvedUpdatePlan) []formats.Package {
	pkgs := make([]formats.Package, 0, len(resolved))
	for _, plan := range resolved {
		pkgs = append(pkgs, plan.Pkg)
	}
	return pkgs
}

// BuildGroupedPlans builds the grouped update plans from resolved plans.
// The ctx parameter allows cancellation of long-running version fetches.
func BuildGroupedPlans(
	ctx context.Context,
	resolved []ResolvedUpdatePlan,
	updateCtx *UpdateContext,
	opts PlanningOptions,
	listVersions VersionLister,
	deriveReason UnsupportedReasonDeriver,
) []*PlannedUpdate {
	var groupedPlans []*PlannedUpdate

	for _, plan := range resolved {
		// Check for context cancellation to allow early termination
		if ctx.Err() != nil {
			break
		}

		p := plan.Pkg
		res := UpdateResult{
			Pkg:               p,
			Status:            constants.StatusUpToDate,
			Group:             p.Group,
			OriginalInstalled: p.InstalledVersion,
		}
		originalVersion := p.Version

		updateCfg, cfgErr := plan.Cfg, plan.Err
		if cfgErr != nil {
			planned := handleConfigError(p, cfgErr, updateCtx, originalVersion, deriveReason)
			groupedPlans = append(groupedPlans, planned)
			continue
		}

		// Handle floating constraints
		if IsFloatingConstraint(p) {
			planned := handleFloatingConstraint(p, updateCfg, updateCtx, originalVersion)
			groupedPlans = append(groupedPlans, planned)
			continue
		}

		// Handle exact constraints - but only skip version lookup if truly fully pinned (3+ segments)
		// For versions with fewer segments (e.g., "5.4"), patch updates are still allowed
		if outdated.IsExactConstraint(p.Constraint) && outdated.IsFullyPinnedVersion(p.Version) {
			planned := handleExactConstraint(p, updateCfg, originalVersion)
			groupedPlans = append(groupedPlans, planned)
			continue
		}

		// Get available versions and plan update
		planned := planVersionUpdate(ctx, p, res, updateCfg, updateCtx, originalVersion, opts, listVersions, deriveReason)
		groupedPlans = append(groupedPlans, planned)
	}

	return groupedPlans
}

// handleConfigError handles packages with configuration errors during planning.
//
// It performs the following operations:
//   - Step 1: Create an UpdateResult with appropriate status
//   - Step 2: Track unsupported packages if error is UnsupportedError
//   - Step 3: Track configuration errors and append to failures
//   - Step 4: Return a PlannedUpdate with error status
//
// Parameters:
//   - p: The package with configuration error
//   - cfgErr: The configuration error encountered
//   - updateCtx: Update context for tracking unsupported packages and failures
//   - originalVersion: Original version of the package for rollback
//   - deriveReason: Function to derive unsupported reason message
//
// Returns:
//   - *PlannedUpdate: Planned update with error status and no target version
func handleConfigError(p formats.Package, cfgErr error, updateCtx *UpdateContext, originalVersion string, deriveReason UnsupportedReasonDeriver) *PlannedUpdate {
	res := UpdateResult{
		Pkg:               p,
		Status:            constants.StatusUpToDate,
		Group:             p.Group,
		OriginalInstalled: p.InstalledVersion,
	}

	if errors.IsUnsupported(cfgErr) {
		res.Status = lock.InstallStatusNotConfigured
		if updateCtx.Unsupported != nil {
			updateCtx.Unsupported.Add(p, deriveReason(p, updateCtx.Cfg, cfgErr, false))
		}
	} else {
		res.Status = constants.StatusConfigError
		res.Err = cfgErr
		updateCtx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", p.Name, p.PackageType, p.Rule, cfgErr))
	}
	res.Group = NormalizeUpdateGroup(nil, p)
	groupKey := UpdateGroupKey(nil, p)
	return &PlannedUpdate{Cfg: nil, Res: res, Original: originalVersion, GroupKey: groupKey}
}

// handleFloatingConstraint handles packages with floating version constraints during planning.
//
// It performs the following operations:
//   - Step 1: Normalize update group for display
//   - Step 2: Create an UpdateResult with floating status
//   - Step 3: Track as unsupported since floating constraints cannot be updated automatically
//   - Step 4: Return a PlannedUpdate with floating status
//
// Parameters:
//   - p: The package with floating constraint (e.g., "latest", "*")
//   - updateCfg: Update configuration for the package
//   - updateCtx: Update context for tracking unsupported packages
//   - originalVersion: Original version of the package for rollback
//
// Returns:
//   - *PlannedUpdate: Planned update with floating status and explanation message
func handleFloatingConstraint(p formats.Package, updateCfg *config.UpdateCfg, updateCtx *UpdateContext, originalVersion string) *PlannedUpdate {
	groupDisplay := NormalizeUpdateGroup(updateCfg, p)
	groupKey := UpdateGroupKey(updateCfg, p)
	res := UpdateResult{
		Pkg:               p,
		Status:            lock.InstallStatusFloating,
		Group:             groupDisplay,
		OriginalInstalled: p.InstalledVersion,
	}
	if updateCtx.Unsupported != nil {
		updateCtx.Unsupported.Add(p, fmt.Sprintf("floating constraint '%s' cannot be updated automatically; remove the constraint or update manually", p.Version))
	}
	return &PlannedUpdate{Cfg: updateCfg, Res: res, Original: originalVersion, GroupKey: groupKey}
}

// handleExactConstraint handles packages with exact version constraints during planning.
//
// It performs the following operations:
//   - Step 1: Create an UpdateResult with up-to-date status
//   - Step 2: Set target to current version (no update needed)
//   - Step 3: Normalize update group for display
//   - Step 4: Return a PlannedUpdate indicating no update is required
//
// Parameters:
//   - p: The package with exact constraint (e.g., "1.2.3" without range operators)
//   - updateCfg: Update configuration for the package
//   - originalVersion: Original version of the package for rollback
//
// Returns:
//   - *PlannedUpdate: Planned update with up-to-date status and target set to current version
func handleExactConstraint(p formats.Package, updateCfg *config.UpdateCfg, originalVersion string) *PlannedUpdate {
	res := UpdateResult{
		Pkg:               p,
		Status:            constants.StatusUpToDate,
		Target:            p.Version,
		Group:             NormalizeUpdateGroup(updateCfg, p),
		OriginalInstalled: p.InstalledVersion,
	}
	groupKey := UpdateGroupKey(updateCfg, p)
	return &PlannedUpdate{Cfg: updateCfg, Res: res, Original: originalVersion, GroupKey: groupKey}
}

// planVersionUpdate plans the version update for a package.
// The ctx parameter allows cancellation of long-running version fetches.
func planVersionUpdate(
	ctx context.Context,
	p formats.Package,
	res UpdateResult,
	updateCfg *config.UpdateCfg,
	updateCtx *UpdateContext,
	originalVersion string,
	opts PlanningOptions,
	listVersions VersionLister,
	deriveReason UnsupportedReasonDeriver,
) *PlannedUpdate {
	cfg := updateCtx.Cfg
	selection := updateCtx.Selection

	versions, err := listVersions(ctx, p, cfg, updateCtx.WorkDir)
	filtered := outdated.FilterVersionsByConstraint(p, versions, selection)
	res.Available = filtered

	groupDisplay := NormalizeUpdateGroup(updateCfg, p)
	res.Group = groupDisplay
	groupKey := UpdateGroupKey(updateCfg, p)

	configIncremental, incrementalErr := config.ShouldUpdateIncrementally(p, cfg)
	if incrementalErr != nil {
		res.Status = constants.StatusConfigError
		res.Err = incrementalErr
		updateCtx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", p.Name, p.PackageType, p.Rule, incrementalErr))
		return &PlannedUpdate{Cfg: updateCfg, Res: res, Original: originalVersion, GroupKey: groupKey}
	}

	// opts.IncrementalMode flag forces incremental mode for all packages
	incremental := opts.IncrementalMode || configIncremental

	if err != nil {
		if errors.IsUnsupported(err) {
			res.Status = lock.InstallStatusNotConfigured
			if updateCtx.Unsupported != nil {
				updateCtx.Unsupported.Add(p, deriveReason(p, cfg, err, false))
			}
		} else {
			res.Status = constants.StatusFailed
			res.Err = err
			updateCtx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", p.Name, p.PackageType, p.Rule, err))
		}
		return &PlannedUpdate{Cfg: updateCfg, Res: res, Original: originalVersion, GroupKey: groupKey}
	}

	ruleCfg := cfg.Rules[p.Rule]
	var versioning *config.VersioningCfg
	if ruleCfg.Outdated != nil {
		versioning = ruleCfg.Outdated.Versioning
	}

	// Filter versions by package's original constraint (not by scope) to get all available within constraint
	allWithinConstraint := outdated.FilterVersionsByConstraint(p, versions, outdated.UpdateSelectionFlags{})

	// Summarize all versions within constraint (for remaining updates summary)
	major, minor, patch, summarizeErr := outdated.SummarizeAvailableVersions(outdated.CurrentVersionForOutdated(p), allWithinConstraint, versioning, incremental)
	if summarizeErr != nil {
		res.Status = constants.StatusSummarizeError
		res.Err = summarizeErr
		updateCtx.AppendFailure(fmt.Errorf("%s (%s/%s): %w", p.Name, p.PackageType, p.Rule, summarizeErr))
		return &PlannedUpdate{Cfg: updateCfg, Res: res, Original: originalVersion, GroupKey: groupKey}
	}

	// Store available versions for preview and summary
	res.Major = major
	res.Minor = minor
	res.Patch = patch

	// Summarize FILTERED versions to get target based on selection scope.
	// Error is intentionally ignored - if version selection fails, target will be empty
	// and the package will be shown as up-to-date (no update available for the filtered scope).
	filteredMajor, filteredMinor, filteredPatch, _ := outdated.SummarizeAvailableVersions(outdated.CurrentVersionForOutdated(p), filtered, versioning, incremental)
	target, _ := outdated.SelectTargetVersion(filteredMajor, filteredMinor, filteredPatch, selection, p.Constraint, incremental)
	res.Target = target

	return &PlannedUpdate{
		Cfg:                  updateCfg,
		Res:                  res,
		Original:             originalVersion,
		GroupKey:             groupKey,
		VersionsInConstraint: allWithinConstraint,
		Versioning:           versioning,
		Incremental:          incremental,
	}
}

// IsFloatingConstraint checks if the package has a floating constraint.
func IsFloatingConstraint(p formats.Package) bool {
	return utils.IsFloatingConstraint(p.Version)
}

// CountPendingUpdates counts the number of packages that have a target version set for update.
func CountPendingUpdates(plans []*PlannedUpdate) int {
	count := 0
	for _, plan := range plans {
		if plan.Res.Target != "" && !IsNonUpdatableStatus(plan.Res.Status) {
			count++
		}
	}
	return count
}

// IsNonUpdatableStatus returns true if the status indicates the package cannot be updated.
func IsNonUpdatableStatus(status string) bool {
	return status == lock.InstallStatusNotConfigured ||
		status == lock.InstallStatusFloating ||
		status == constants.StatusConfigError ||
		status == constants.StatusFailed ||
		status == constants.StatusSummarizeError
}

// ShouldSkipUpdate returns true if the update result status indicates the update should be skipped.
func ShouldSkipUpdate(res *UpdateResult) bool {
	return IsNonUpdatableStatus(res.Status) || res.Target == ""
}

// RefreshAvailableVersions recalculates major/minor/patch available versions
// using the updated target version as the new baseline.
func RefreshAvailableVersions(plan *PlannedUpdate) {
	if plan.VersionsInConstraint == nil || plan.Res.Target == "" {
		return
	}

	major, minor, patch, err := outdated.SummarizeAvailableVersions(
		plan.Res.Target,
		plan.VersionsInConstraint,
		plan.Versioning,
		plan.Incremental,
	)
	if err != nil {
		return
	}

	plan.Res.Major = major
	plan.Res.Minor = minor
	plan.Res.Patch = patch
}
