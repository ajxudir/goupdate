package update

import (
	"errors"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	pkgerrors "github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProcessGroupedPlansWithProgress(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("handles empty plans", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		ProcessGroupedPlansWithProgress(ctx, []*PlannedUpdate{}, &results, progress, callbacks)

		assert.Empty(t, results)
		assert.Equal(t, 0, progress.count)
	})

	t.Run("processes single plan", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
				GroupKey: "npm:js",
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		ProcessGroupedPlansWithProgress(ctx, plans, &results, progress, callbacks)

		assert.Len(t, results, 1)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("processes multiple groups", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
				GroupKey: "npm:js:frontend",
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("vue", "2.0.0", "2.0.0"),
					Target: "3.0.0",
					Status: constants.StatusPlanned,
				},
				GroupKey: "npm:js:backend",
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		ProcessGroupedPlansWithProgress(ctx, plans, &results, progress, callbacks)

		assert.Len(t, results, 2)
		assert.Equal(t, 2, progress.count)
	})
}

func TestProcessGroupPlansWithProgress(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("processes empty plans", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansWithProgress(ctx, []*PlannedUpdate{}, &results, progress, callbacks)

		assert.Empty(t, results)
		assert.Equal(t, 0, progress.count)
	})

	t.Run("processes single package", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansWithProgress(ctx, plans, &results, progress, callbacks)

		assert.Len(t, results, 1)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("uses group lock for multiple packages in dry run", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // dry run
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
				Cfg: &config.UpdateCfg{Commands: "npm install"},
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("vue", "2.0.0", "2.0.0"),
					Target: "3.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansWithProgress(ctx, plans, &results, progress, callbacks)

		assert.Len(t, results, 2)
		assert.Equal(t, 2, progress.count)
	})

	t.Run("handles rollback on group error with multiple packages", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if callCount == 2 {
				return errors.New("update failed")
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false) // dry run to avoid actual lock
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
				Cfg:      &config.UpdateCfg{Commands: "npm install"},
				Original: "17.0.0",
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("vue", "2.0.0", "2.0.0"),
					Target: "3.0.0",
					Status: constants.StatusPlanned,
				},
				Cfg:      &config.UpdateCfg{Commands: "npm install"},
				Original: "2.0.0",
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansWithProgress(ctx, plans, &results, progress, callbacks)

		// Results should still be collected
		assert.Greater(t, len(results), 0)
	})

	t.Run("triggers rollback on group error in non-dry-run", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			// First call is update
			if callCount == 1 {
				// Simulate successful update
				return nil
			}
			// Second package fails, triggering rollback
			return errors.New("update failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, true) // non-dry run, skipLockRun=true to avoid running lock command
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
				Cfg:      &config.UpdateCfg{Commands: "npm install"},
				Original: "17.0.0",
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("vue", "2.0.0", "2.0.0"),
					Target: "3.0.0",
					Status: constants.StatusPlanned,
				},
				Cfg:      &config.UpdateCfg{Commands: "npm install"},
				Original: "2.0.0",
			},
		}
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansWithProgress(ctx, plans, &results, progress, callbacks)

		// At least one result should be collected
		assert.Greater(t, len(results), 0)
	})
}

func TestProcessGroupWithGroupLockProgress(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("returns error with nil config", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, nil, &applied, &results, progress, callbacks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no update configuration found")
	})

	t.Run("processes dry run successfully", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("skips non-updatable packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusNotConfigured,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("handles update error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("update failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.Error(t, err)
	})

	t.Run("handles unsupported error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return &pkgerrors.UnsupportedError{Reason: "not configured"}
		}

		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		// Unsupported errors don't count as group errors
		assert.NoError(t, err)
	})

	t.Run("validates and tracks in results", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "18.0.0", "18.0.0"),
			}, nil
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("handles validation error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("handles group error path with multiple packages", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if callCount == 2 {
				return errors.New("second update failed")
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("vue", "2.0.0", "2.0.0"),
					Target: "3.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		assert.Error(t, err)
		// First package was applied, second had error
		assert.Len(t, applied, 1)
		assert.Equal(t, 1, progress.count) // First package increments progress in else branch
	})

	t.Run("tracks floating status on group error path", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("update failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusFloating,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		_ = processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, progress, callbacks)

		// Should be tracked as unsupported
		assert.Len(t, tracker.packages, 1)
	})
}
