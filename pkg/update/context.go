package update

import (
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/systemtest"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// VersionSnapshot stores the version state of a package for validation.
type VersionSnapshot struct {
	Version   string
	Installed string
}

// UnsupportedTracker is an interface for tracking packages that cannot be updated.
// This allows the cmd layer to provide its own implementation while pkg/update
// can work with any compatible tracker.
type UnsupportedTracker interface {
	Add(p formats.Package, reason string)
	Messages() []string
}

// UpdateContext encapsulates the common parameters and state needed during update operations.
// This reduces function parameter counts and improves code maintainability.
type UpdateContext struct {
	// Configuration
	Cfg     *config.Config
	WorkDir string

	// Flags
	DryRun          bool
	ContinueOnError bool
	SkipLockRun     bool
	IncrementalMode bool // Force incremental updates (one version step at a time)

	// Version selection flags (also used for display formatting)
	Selection outdated.UpdateSelectionFlags

	// Tracking
	Unsupported UnsupportedTracker
	Failures    []error
	Baseline    map[string]VersionSnapshot

	// Display
	Table *output.Table

	// System tests
	SystemTestRunner *systemtest.Runner

	// Functions
	ReloadList func() ([]formats.Package, error)

	// Derive unsupported reason function (injected by cmd layer)
	DeriveUnsupportedReason func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string

	// UpdaterFunc is the function used to update packages
	UpdaterFunc func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error

	// SkipSystemTests flag (set by CLI)
	SkipSystemTests bool
}

// NewUpdateContext creates a new UpdateContext with the given parameters.
func NewUpdateContext(cfg *config.Config, workDir string, unsupported UnsupportedTracker) *UpdateContext {
	return &UpdateContext{
		Cfg:         cfg,
		WorkDir:     workDir,
		Unsupported: unsupported,
		Failures:    make([]error, 0),
	}
}

// WithFlags sets the execution flags on the context and returns the context for chaining.
func (ctx *UpdateContext) WithFlags(dryRun, continueOnError, skipLockRun bool) *UpdateContext {
	ctx.DryRun = dryRun
	ctx.ContinueOnError = continueOnError
	ctx.SkipLockRun = skipLockRun
	return ctx
}

// WithBaseline sets the version baseline for validation and returns the context for chaining.
func (ctx *UpdateContext) WithBaseline(baseline map[string]VersionSnapshot) *UpdateContext {
	ctx.Baseline = baseline
	return ctx
}

// WithTable sets the output table and returns the context for chaining.
func (ctx *UpdateContext) WithTable(table *output.Table) *UpdateContext {
	ctx.Table = table
	return ctx
}

// WithSystemTestRunner sets the system test runner and returns the context for chaining.
func (ctx *UpdateContext) WithSystemTestRunner(runner *systemtest.Runner) *UpdateContext {
	ctx.SystemTestRunner = runner
	return ctx
}

// WithReloadList sets the reload function and returns the context for chaining.
func (ctx *UpdateContext) WithReloadList(reloadList func() ([]formats.Package, error)) *UpdateContext {
	ctx.ReloadList = reloadList
	return ctx
}

// WithSelection sets the version selection flags and returns the context for chaining.
func (ctx *UpdateContext) WithSelection(selection outdated.UpdateSelectionFlags) *UpdateContext {
	ctx.Selection = selection
	return ctx
}

// WithSkipSystemTests sets the skip system tests flag and returns the context for chaining.
func (ctx *UpdateContext) WithSkipSystemTests(skip bool) *UpdateContext {
	ctx.SkipSystemTests = skip
	return ctx
}

// WithIncrementalMode sets the incremental mode flag and returns the context for chaining.
func (ctx *UpdateContext) WithIncrementalMode(incremental bool) *UpdateContext {
	ctx.IncrementalMode = incremental
	return ctx
}

// WithDeriveUnsupportedReason sets the function to derive unsupported reasons.
func (ctx *UpdateContext) WithDeriveUnsupportedReason(fn func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string) *UpdateContext {
	ctx.DeriveUnsupportedReason = fn
	return ctx
}

// WithUpdaterFunc sets the package updater function.
func (ctx *UpdateContext) WithUpdaterFunc(fn func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error) *UpdateContext {
	ctx.UpdaterFunc = fn
	return ctx
}

// ShouldRunSystemTestsAfterEach returns true if system tests should run after each update.
func (ctx *UpdateContext) ShouldRunSystemTestsAfterEach() bool {
	return ctx.SystemTestRunner != nil && ctx.SystemTestRunner.ShouldRunAfterEach() && !ctx.SkipSystemTests
}

// AppendFailure adds an error to the failures slice.
func (ctx *UpdateContext) AppendFailure(err error) {
	if err != nil {
		ctx.Failures = append(ctx.Failures, err)
	}
}

// SnapshotVersions creates a map of package keys to their version snapshots.
// This captures the baseline state before updates for drift detection.
func SnapshotVersions(packages []formats.Package) map[string]VersionSnapshot {
	snapshots := make(map[string]VersionSnapshot)
	for _, p := range packages {
		key := PackageKey(p)
		snapshots[key] = VersionSnapshot{Version: p.Version, Installed: p.InstalledVersion}
	}
	verbose.Debugf("Baseline snapshot captured: %d packages", len(snapshots))
	return snapshots
}

// PackageKey generates a unique key for a package.
func PackageKey(p formats.Package) string {
	return p.Rule + "|" + p.PackageType + "|" + p.Type + "|" + p.Name
}
