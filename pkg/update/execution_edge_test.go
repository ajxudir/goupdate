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

func TestProcessGroupWithGroupLockEdgeCases(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("returns error when groupUpdateCfg is nil", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)

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
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, nil, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no update configuration found")
	})

	t.Run("skips packages with non-updatable status", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", &mockUnsupportedTracker{}).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)

		// Package with floating status should be skipped (it's a non-updatable status)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusFloating,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, applied, 0) // Nothing was applied
		assert.Len(t, results, 1) // Result was still recorded
	})

	t.Run("handles unsupported update error", func(t *testing.T) {
		unsupportedErr := pkgerrors.NewUnsupportedError("update", "test reason", "unsupported-pkg")
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return unsupportedErr
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("unsupported-pkg", "1.0.0", "1.0.0"),
					Target: "2.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		// Unsupported errors don't propagate as group error
		assert.NoError(t, err)
		assert.Len(t, applied, 0)
	})

	t.Run("handles regular update error and propagates it", func(t *testing.T) {
		updateErr := errors.New("update failed")
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return updateErr
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
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		// Regular errors propagate as group error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("invokes OnResultReady callback", func(t *testing.T) {
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
		var failures []SystemTestFailure

		callbackCalled := false
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				callbackCalled = true
			},
		}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.True(t, callbackCalled)
	})
}

// TestProcessGroupWithGroupLockProgressEdgeCases tests edge cases for processGroupWithGroupLockProgress.
func TestProcessGroupWithGroupLockProgressEdgeCases(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("returns error when groupUpdateCfg is nil", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)

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
		mockProgress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, nil, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no update configuration found")
	})

	t.Run("skips packages and increments progress", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", &mockUnsupportedTracker{}).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)

		// Package with NotConfigured status should be skipped but tracked
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusNotConfigured,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		mockProgress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, applied, 0)
		assert.Len(t, results, 1)
		assert.Equal(t, 1, mockProgress.count) // Progress was incremented even for skipped
	})

	t.Run("handles group error in else branch with progress", func(t *testing.T) {
		updateErr := errors.New("update failed")
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			if target == "2.0.0" {
				return updateErr
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("pkg1", "18.0.0", "18.0.0"),
			}, nil
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false)

		// Multiple packages: first succeeds, second fails
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("pkg1", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("pkg2", "1.0.0", "1.0.0"),
					Target: "2.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		mockProgress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		// The else branch with groupErr should be executed, incrementing progress for applied plans
		assert.Equal(t, 1, mockProgress.count) // Only the successful one increments
	})
}

// TestProcessGroupWithGroupLockSystemTests tests system test execution in group lock mode.
func TestProcessGroupWithGroupLockSystemTests(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("runs system tests after successful group update", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		// Create a system test runner
		runner := testutil.CreateSystemTestRunner(nil, false, false) // No tests configured = passes

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "18.0.0", "18.0.0"),
			}, nil
		}

		// Create system tests config for after_each mode
		sysTestsCfg := &config.SystemTestsCfg{
			RunMode: "after_each",
		}
		cfg.SystemTests = sysTestsCfg

		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithSystemTestRunner(runner).
			WithFlags(true, false, false) // dry run to skip actual lock command

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
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
	})
}

// TestProcessGroupWithGroupLockProgressSystemTests tests system test handling with progress.
func TestProcessGroupWithGroupLockProgressSystemTests(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("processes with validation error in progress mode", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("validation error")
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false) // dry run to skip actual lock command
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
		mockProgress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 1, mockProgress.count)
	})

	t.Run("handles group error with progress", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Wrong version - validation error
			}, nil
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false) // dry run to skip lock command
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
		mockProgress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})
}

// TestProcessGroupPerPackageContinueOnError tests ContinueOnError behavior.
func TestProcessGroupPerPackageContinueOnError(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("continues after update error when ContinueOnError is true", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if p.Name == "react" {
				return errors.New("update failed")
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, true, false) // dry run + continue on error
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Pkg: testutil.NPMPackage("lodash", "4.0.0", "4.0.0"), Target: "4.17.21", Status: constants.StatusPlanned}},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackage(ctx, plans, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 2, callCount, "both packages should be processed")
		assert.Len(t, results, 2)
	})

	t.Run("continues after validation error when ContinueOnError is true", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, true, false). // not dry run + continue on error
			WithReloadList(func() ([]formats.Package, error) {
				// Return empty to cause validation failure
				return []formats.Package{}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Pkg: testutil.NPMPackage("lodash", "4.0.0", "4.0.0"), Target: "4.17.21", Status: constants.StatusPlanned}},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackage(ctx, plans, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 2, callCount, "both packages should be processed")
		assert.Len(t, results, 2)
	})
}

// TestProcessGroupWithGroupLockContinueOnError tests ContinueOnError behavior for group lock.
func TestProcessGroupWithGroupLockContinueOnError(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("continues after update error when ContinueOnError is true", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if p.Name == "react" {
				return errors.New("update failed")
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, true, false) // dry run + continue on error
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Pkg: testutil.NPMPackage("lodash", "4.0.0", "4.0.0"), Target: "4.17.21", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 2, callCount, "both packages should be processed")
	})

	t.Run("tracks unsupported packages with OnResultReady callback", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // dry run
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbackInvoked := false
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				callbackInvoked = true
			},
		}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.True(t, callbackInvoked)
	})
}

// TestProcessGroupWithGroupLockProgressContinueOnError tests ContinueOnError behavior for progress variant.
func TestProcessGroupWithGroupLockProgressContinueOnError(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("continues after update error when ContinueOnError is true", func(t *testing.T) {
		callCount := 0
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			callCount++
			if p.Name == "react" {
				return errors.New("update failed")
			}
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, true, false) // dry run + continue on error
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Pkg: testutil.NPMPackage("lodash", "4.0.0", "4.0.0"), Target: "4.17.21", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		// Use nil progress reporter
		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, nil, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 2, callCount, "both packages should be processed")
	})

	t.Run("increments progress for skipped packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: lock.InstallStatusFloating}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.NoError(t, err)
		assert.Equal(t, 1, progressCount, "progress should be incremented for skipped package")
	})
}

// TestProcessGroupWithGroupLockValidation tests validation paths.
func TestProcessGroupWithGroupLockValidation(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("validates packages with reload list in dry run", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false). // dry run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
	})

	t.Run("handles validation failure in dry run", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false). // dry run
			WithReloadList(func() ([]formats.Package, error) {
				// Return wrong version to cause validation failure
				return []formats.Package{
					testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("tracks unsupported status packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // dry run
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: lock.InstallStatusNotConfigured}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, tracker.packages, 1, "unsupported package should be tracked")
	})
}

// TestProcessGroupWithGroupLockProgressValidation tests validation paths for progress variant.
func TestProcessGroupWithGroupLockProgressValidation(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("validates packages with reload list in dry run", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false). // dry run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
		assert.Equal(t, 1, progressCount)
	})

	t.Run("handles validation failure in dry run", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false). // dry run
			WithReloadList(func() ([]formats.Package, error) {
				// Return wrong version to cause validation failure
				return []formats.Package{
					testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("tracks unsupported status packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // dry run
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: lock.InstallStatusNotConfigured}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.NoError(t, err)
		assert.Len(t, tracker.packages, 1, "unsupported package should be tracked")
		assert.Equal(t, 1, progressCount)
	})
}

// TestProcessGroupWithGroupLockNonDryRun tests the lock command execution path in non-dry-run mode.
func TestProcessGroupWithGroupLockNonDryRun(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("executes lock command on successful update", func(t *testing.T) {
		// Mock the exec command function
		originalExec := execCommandFunc
		lockCommandCalled := false
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			lockCommandCalled = true
			return []byte("success"), nil
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.True(t, lockCommandCalled, "lock command should be called in non-dry-run mode")
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
	})

	t.Run("handles lock command failure", func(t *testing.T) {
		originalExec := execCommandFunc
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			return nil, errors.New("lock command failed")
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lock command failed")
		// After lock failure, results should still be appended (with failed status)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("checks with_all_dependencies flag", func(t *testing.T) {
		originalExec := execCommandFunc
		withAllDepsUsed := false
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			withAllDepsUsed = withAllDeps
			return []byte("success"), nil
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		// Create config with with_all_dependencies for react package
		cfg := testutil.NewConfig().WithRule("npm", config.PackageManagerCfg{
			Update: &config.UpdateCfg{
				Commands: "npm install",
			},
			Packages: map[string]config.PackageSettings{
				"react": {WithAllDependencies: true},
			},
		}).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.True(t, withAllDepsUsed, "with_all_dependencies flag should be set")
	})
}

// TestProcessGroupWithGroupLockProgressNonDryRun tests the lock command execution path with progress reporting.
func TestProcessGroupWithGroupLockProgressNonDryRun(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("executes lock command and reports progress", func(t *testing.T) {
		originalExec := execCommandFunc
		lockCommandCalled := false
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			lockCommandCalled = true
			return []byte("success"), nil
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.NoError(t, err)
		assert.True(t, lockCommandCalled, "lock command should be called")
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
		assert.Equal(t, 1, progressCount, "progress should be incremented")
	})

	t.Run("handles lock command failure with progress", func(t *testing.T) {
		originalExec := execCommandFunc
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			return nil, errors.New("lock failed")
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				return []formats.Package{
					testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		assert.Error(t, err)
		// Progress should still be incremented even on failure (in the error path)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
		assert.Equal(t, 1, progressCount, "progress should be incremented even on failure")
	})

	t.Run("tracks unsupported status after validation failure", func(t *testing.T) {
		originalExec := execCommandFunc
		execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
			return []byte("success"), nil
		}
		t.Cleanup(func() { execCommandFunc = originalExec })

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false). // non-dry-run
			WithReloadList(func() ([]formats.Package, error) {
				// Return wrong version to cause validation failure
				return []formats.Package{
					testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				}, nil
			})
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0"), Target: "18.0.0", Status: constants.StatusPlanned}},
		}
		groupCfg := &config.UpdateCfg{Commands: "npm install"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		progressCount := 0
		mockProgress := &mockProgressReporter{incrementFn: func() { progressCount++ }}

		err := processGroupWithGroupLockProgress(ctx, plans, groupCfg, &applied, &results, mockProgress, callbacks)

		// Validation failure returns error
		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})
}
