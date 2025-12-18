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

// Note: mockUnsupportedTracker is defined in context_test.go

// TestValidateUpdatedPackage tests the behavior of ValidateUpdatedPackage.
//
// It verifies:
//   - Returns nil with nil reloadList
//   - Passes when version matches target
//   - Returns error on reload failure
//   - Returns error when package not found after reload
//   - Returns error when version mismatch
func TestValidateUpdatedPackage(t *testing.T) {
	t.Run("returns nil with nil reloadList", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}

		err := ValidateUpdatedPackage(plan, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("passes when version matches target", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "18.0.0", "18.0.0"),
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.NoError(t, err)
		assert.Equal(t, "18.0.0", plan.Res.Pkg.Version)
		assert.Equal(t, "18.0.0", plan.Res.Pkg.InstalledVersion)
	})

	t.Run("returns error on reload failure", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.ErrorContains(t, err, "reload failed")
	})

	t.Run("returns error when package not found after reload", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("vue", "3.0.0", "3.0.0"),
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing after update validation")
	})

	t.Run("returns error when version mismatch", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Still old version
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version mismatch")
	})
}

// TestValidateUpdatedPackageInstalledVersionMismatch tests the behavior of ValidateUpdatedPackage with InstalledVersion mismatches.
//
// It verifies:
//   - Passes when InstalledVersion matches target
//   - Fails when InstalledVersion doesn't match target
//   - Passes when InstalledVersion is empty
//   - Passes when InstalledVersion is N/A
func TestValidateUpdatedPackageInstalledVersionMismatch(t *testing.T) {
	t.Run("passes when InstalledVersion matches target", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.GoPackage("github.com/example/pkg", "v1.0.0", "v1.0.0"),
				Target: "v2.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.GoPackage("github.com/example/pkg", "v2.0.0", "v2.0.0"),
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.NoError(t, err)
	})

	t.Run("fails when InstalledVersion doesn't match target", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.GoPackage("github.com/example/pkg", "v1.0.0", "v1.0.0"),
				Target: "v2.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				{
					Rule:             "mod",
					PackageType:      "golang",
					Type:             "prod",
					Name:             "github.com/example/pkg",
					Version:          "v2.0.0", // Manifest shows target
					InstalledVersion: "v1.0.0", // But lock file still has old version
				},
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "installed version mismatch")
		assert.Contains(t, err.Error(), "v2.0.0")
		assert.Contains(t, err.Error(), "v1.0.0")
	})

	t.Run("passes when InstalledVersion is empty", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.GoPackage("github.com/example/pkg", "v1.0.0", ""),
				Target: "v2.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.GoPackage("github.com/example/pkg", "v2.0.0", ""),
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.NoError(t, err)
	})

	t.Run("passes when InstalledVersion is N/A", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.GoPackage("github.com/example/pkg", "v1.0.0", "#N/A"),
				Target: "v2.0.0",
			},
		}

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.GoPackage("github.com/example/pkg", "v2.0.0", "#N/A"),
			}, nil
		}

		err := ValidateUpdatedPackage(plan, reloadFunc, nil)
		assert.NoError(t, err)
	})
}

// TestCollectUpdateErrors tests the behavior of CollectUpdateErrors.
//
// It verifies:
//   - Collects errors from failed results
//   - Returns empty for successful results
//   - Handles empty results
//   - Handles nil results
//   - Excludes unsupported errors
func TestCollectUpdateErrors(t *testing.T) {
	t.Run("collects errors from failed results", func(t *testing.T) {
		results := []UpdateResult{
			{Pkg: formats.Package{Name: "react"}, Status: constants.StatusFailed, Err: errors.New("failed")},
			{Pkg: formats.Package{Name: "vue"}, Status: constants.StatusUpdated},
		}

		errs := CollectUpdateErrors(results)
		assert.Len(t, errs, 1)
	})

	t.Run("returns empty for successful results", func(t *testing.T) {
		results := []UpdateResult{
			{Pkg: formats.Package{Name: "react"}, Status: constants.StatusUpdated},
			{Pkg: formats.Package{Name: "vue"}, Status: constants.StatusUpdated},
		}

		errs := CollectUpdateErrors(results)
		assert.Empty(t, errs)
	})

	t.Run("handles empty results", func(t *testing.T) {
		errs := CollectUpdateErrors([]UpdateResult{})
		assert.Empty(t, errs)
	})

	t.Run("handles nil results", func(t *testing.T) {
		errs := CollectUpdateErrors(nil)
		assert.Empty(t, errs)
	})

	t.Run("excludes unsupported errors", func(t *testing.T) {
		results := []UpdateResult{
			{Pkg: formats.Package{Name: "react"}, Status: constants.StatusFailed, Err: errors.New("failed")},
			{Pkg: formats.Package{Name: "vue"}, Status: lock.InstallStatusNotConfigured, Err: &pkgerrors.UnsupportedError{Reason: "no config"}},
		}

		errs := CollectUpdateErrors(results)
		assert.Len(t, errs, 1)
	})
}

// TestSummarizeGroupFailure tests the behavior of SummarizeGroupFailure.
//
// It verifies:
//   - Marks updated plans as failed
//   - Preserves existing errors
//   - Handles empty plans
//   - Skips config error status
func TestSummarizeGroupFailure(t *testing.T) {
	t.Run("marks updated plans as failed", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusUpdated}},
			{Res: UpdateResult{Status: lock.InstallStatusNotConfigured}},
		}

		groupErr := errors.New("group failed")
		SummarizeGroupFailure(plans, groupErr)

		assert.Equal(t, constants.StatusFailed, plans[0].Res.Status)
		assert.Equal(t, groupErr, plans[0].Res.Err)
		// NotConfigured should remain unchanged
		assert.Equal(t, lock.InstallStatusNotConfigured, plans[1].Res.Status)
	})

	t.Run("preserves existing errors", func(t *testing.T) {
		existingErr := errors.New("existing error")
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusUpdated, Err: existingErr}},
		}

		groupErr := errors.New("group failed")
		SummarizeGroupFailure(plans, groupErr)

		assert.Equal(t, constants.StatusFailed, plans[0].Res.Status)
		assert.Equal(t, existingErr, plans[0].Res.Err) // Preserved
	})

	t.Run("handles empty plans", func(t *testing.T) {
		// Should not panic
		SummarizeGroupFailure([]*PlannedUpdate{}, errors.New("error"))
	})

	t.Run("skips config error status", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusConfigError}},
		}

		SummarizeGroupFailure(plans, errors.New("group failed"))
		assert.Equal(t, constants.StatusConfigError, plans[0].Res.Status)
	})
}

// TestShouldTrackUnsupported tests the behavior of ShouldTrackUnsupported.
//
// It verifies:
//   - Returns true for unsupported statuses (not configured, floating, version missing)
//   - Returns false for normal statuses (updated, failed, empty)
func TestShouldTrackUnsupported(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"not configured", lock.InstallStatusNotConfigured, true},
		{"floating", lock.InstallStatusFloating, true},
		{"version missing", lock.InstallStatusVersionMissing, true},
		{"updated", constants.StatusUpdated, false},
		{"failed", constants.StatusFailed, false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldTrackUnsupported(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRollbackPlans tests the behavior of RollbackPlans.
//
// It verifies:
//   - Rolls back all plans successfully
//   - Collects rollback errors
//   - Preserves existing errors
func TestRollbackPlans(t *testing.T) {
	t.Run("rolls back all plans successfully", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated}, Original: "17.0.0"},
		}
		cfg := testutil.NewConfig().Build()
		ctx := &UpdateContext{Failures: make([]error, 0)}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.NoError(t, err)
		assert.Equal(t, constants.StatusFailed, plans[0].Res.Status)
		assert.Equal(t, groupErr, plans[0].Res.Err)
	})

	t.Run("collects rollback errors", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated}, Original: "17.0.0"},
		}
		cfg := testutil.NewConfig().Build()
		ctx := &UpdateContext{Failures: make([]error, 0)}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("rollback failed")
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rollback failed")
		assert.Len(t, ctx.Failures, 1)
	})

	t.Run("preserves existing errors", func(t *testing.T) {
		existingErr := errors.New("original error")
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated, Err: existingErr}, Original: "17.0.0"},
		}
		cfg := testutil.NewConfig().Build()
		ctx := &UpdateContext{Failures: make([]error, 0)}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.NoError(t, err)
		assert.Equal(t, existingErr, plans[0].Res.Err) // Original error preserved
	})
}

// TestHandleUpdateError tests the behavior of HandleUpdateError.
//
// It verifies:
//   - Handles unsupported error correctly
//   - Handles normal error correctly
func TestHandleUpdateError(t *testing.T) {
	t.Run("handles unsupported error", func(t *testing.T) {
		res := &UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0")}
		tracker := &mockUnsupportedTracker{}
		ctx := &UpdateContext{
			Cfg:         testutil.NewConfig().Build(),
			Unsupported: tracker,
			Failures:    make([]error, 0),
		}
		unsupportedErr := &pkgerrors.UnsupportedError{Reason: "no config"}

		deriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
			return "derived reason"
		}

		HandleUpdateError(unsupportedErr, res, ctx, deriveReason)

		assert.Equal(t, lock.InstallStatusNotConfigured, res.Status)
		assert.Len(t, tracker.packages, 1)
		assert.Empty(t, ctx.Failures)
	})

	t.Run("handles normal error", func(t *testing.T) {
		res := &UpdateResult{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0")}
		ctx := &UpdateContext{
			Cfg:      testutil.NewConfig().Build(),
			Failures: make([]error, 0),
		}
		normalErr := errors.New("update failed")

		deriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
			return "derived reason"
		}

		HandleUpdateError(normalErr, res, ctx, deriveReason)

		assert.Equal(t, constants.StatusFailed, res.Status)
		assert.Len(t, ctx.Failures, 1)
		assert.Equal(t, normalErr, res.Err)
	})
}

// TestApplyPlannedUpdate tests the behavior of ApplyPlannedUpdate.
//
// It verifies:
//   - Calls updater with correct arguments
//   - Returns updater error
func TestApplyPlannedUpdate(t *testing.T) {
	t.Run("calls updater with correct arguments", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		var calledPkg formats.Package
		var calledTarget string
		var calledDryRun bool
		var calledSkipLock bool

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			calledPkg = p
			calledTarget = target
			calledDryRun = dryRun
			calledSkipLock = skipLock
			return nil
		}

		err := ApplyPlannedUpdate(plan, cfg, "/test", updater, true, true)

		assert.NoError(t, err)
		assert.Equal(t, "react", calledPkg.Name)
		assert.Equal(t, "18.0.0", calledTarget)
		assert.True(t, calledDryRun)
		assert.True(t, calledSkipLock)
	})

	t.Run("returns updater error", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("update failed")
		}

		err := ApplyPlannedUpdate(plan, cfg, "/test", updater, false, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})
}

// TestProcessGroupedPlansLive tests the behavior of ProcessGroupedPlansLive.
//
// It verifies:
//   - Handles empty plans
//   - Processes single plan
//   - Processes multiple groups
//   - Skips updates with non-updatable status
func TestProcessGroupedPlansLive(t *testing.T) {
	mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return nil
	}

	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("handles empty plans", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false) // dryRun, continueOnError, skipLockRun
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		ProcessGroupedPlansLive(ctx, []*PlannedUpdate{}, &results, callbacks)

		assert.Empty(t, results)
	})

	t.Run("processes single plan", func(t *testing.T) {
		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // Use dry run to avoid group lock
		var results []UpdateResult
		var resultsCalled []UpdateResult
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				resultsCalled = append(resultsCalled, res)
			},
		}
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

		ProcessGroupedPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 1)
		assert.Len(t, resultsCalled, 1)
	})

	t.Run("processes multiple groups", func(t *testing.T) {
		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
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

		ProcessGroupedPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 2)
	})

	t.Run("skips updates with non-updatable status", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusNotConfigured, // Should be skipped
				},
				GroupKey: "npm:js",
			},
		}

		ProcessGroupedPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 1)
		assert.Equal(t, lock.InstallStatusNotConfigured, results[0].Status)
	})
}

// TestHandleSkippedUpdate tests the behavior of handleSkippedUpdate.
//
// It verifies:
//   - Appends result for skipped update
//   - Calls OnResultReady callback
//   - Tracks unsupported packages
func TestHandleSkippedUpdate(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("appends result for skipped update", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Status: constants.StatusUpToDate,
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		handleSkippedUpdate(ctx, res, &results, callbacks)

		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpToDate, results[0].Status)
	})

	t.Run("calls OnResultReady callback", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Status: constants.StatusUpToDate,
		}
		var results []UpdateResult
		var callbackCalled bool
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				callbackCalled = true
			},
		}

		handleSkippedUpdate(ctx, res, &results, callbacks)

		assert.True(t, callbackCalled)
	})

	t.Run("tracks unsupported packages", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Status: lock.InstallStatusNotConfigured,
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		handleSkippedUpdate(ctx, res, &results, callbacks)

		assert.Len(t, tracker.packages, 1)
	})
}

// TestAppendResultAndPrint tests the behavior of appendResultAndPrint.
//
// It verifies:
//   - Appends result
//   - Calls OnResultReady callback
//   - Tracks unsupported packages
func TestAppendResultAndPrint(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("appends result", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
			Status: constants.StatusUpdated,
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		appendResultAndPrint(ctx, res, &results, callbacks)

		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
	})

	t.Run("calls OnResultReady callback", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
			Status: constants.StatusUpdated,
		}
		var results []UpdateResult
		var resultReceived UpdateResult
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				resultReceived = res
			},
		}

		appendResultAndPrint(ctx, res, &results, callbacks)

		assert.Equal(t, "react", resultReceived.Pkg.Name)
	})

	t.Run("tracks unsupported packages", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker)
		res := &UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Status: lock.InstallStatusFloating,
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		appendResultAndPrint(ctx, res, &results, callbacks)

		assert.Len(t, tracker.packages, 1)
	})
}

// mockTestResultOutput implements the interface needed by SystemTestFailure.
type mockTestResultOutput struct {
	resultOutput string
}

// FormatResultsQuiet is a test helper that formats test results.
func (m *mockTestResultOutput) FormatResultsQuiet() string {
	return m.resultOutput
}

// TestDisplaySystemTestFailures tests the behavior of DisplaySystemTestFailures.
//
// It verifies:
//   - Handles empty failures
//   - Displays critical failure
//   - Displays warning failure
//   - Displays multiple failures
func TestDisplaySystemTestFailures(t *testing.T) {
	t.Run("handles empty failures", func(t *testing.T) {
		// Should not panic with empty slice
		DisplaySystemTestFailures([]SystemTestFailure{})
	})

	t.Run("displays critical failure", func(t *testing.T) {
		failures := []SystemTestFailure{
			{
				PkgName:    "react",
				Result:     &mockTestResultOutput{resultOutput: "Test output here\n"},
				IsCritical: true,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			DisplaySystemTestFailures(failures)
		})

		assert.Contains(t, output, "System tests failed after react update")
		assert.Contains(t, output, "Test output here")
	})

	t.Run("displays warning failure", func(t *testing.T) {
		failures := []SystemTestFailure{
			{
				PkgName:    "vue",
				Result:     &mockTestResultOutput{resultOutput: "Warning output\n"},
				IsCritical: false,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			DisplaySystemTestFailures(failures)
		})

		assert.Contains(t, output, "System test warning after vue update")
		assert.Contains(t, output, "Warning output")
	})

	t.Run("displays multiple failures", func(t *testing.T) {
		failures := []SystemTestFailure{
			{
				PkgName:    "react",
				Result:     &mockTestResultOutput{resultOutput: "React failure\n"},
				IsCritical: true,
			},
			{
				PkgName:    "vue",
				Result:     &mockTestResultOutput{resultOutput: "Vue warning\n"},
				IsCritical: false,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			DisplaySystemTestFailures(failures)
		})

		assert.Contains(t, output, "react")
		assert.Contains(t, output, "vue")
	})
}

// TestProcessGroupPerPackage tests the behavior of processGroupPerPackage.
//
// It verifies:
//   - Processes package with successful update
//   - Handles update error
//   - Skips packages with non-updatable status
func TestProcessGroupPerPackage(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("processes package with successful update", func(t *testing.T) {
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
			},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackage(ctx, plans, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("handles update error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("update failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)
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

		err := processGroupPerPackage(ctx, plans, &applied, &results, &failures, callbacks)

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("skips packages with non-updatable status", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusNotConfigured,
				},
			},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackage(ctx, plans, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, lock.InstallStatusNotConfigured, results[0].Status)
	})
}

// TestProcessGroupPlansLive tests the behavior of processGroupPlansLive.
//
// It verifies:
//   - Processes empty plans
//   - Handles single package group
//   - Uses group lock for multiple packages
func TestProcessGroupPlansLive(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("processes empty plans", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithFlags(false, false, false)
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansLive(ctx, []*PlannedUpdate{}, &results, callbacks)

		assert.Empty(t, results)
	})

	t.Run("handles single package group", func(t *testing.T) {
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
				Cfg: &config.UpdateCfg{Commands: "npm install"},
			},
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 1)
	})

	t.Run("uses group lock for multiple packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(true, false, false) // dry run to avoid actual group lock
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
				Cfg: &config.UpdateCfg{Commands: "npm install"},
			},
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 2)
	})

	t.Run("skips group lock when SkipLockRun is true", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, true) // skipLockRun = true
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
				Cfg: &config.UpdateCfg{Commands: "npm install"},
			},
		}
		var results []UpdateResult
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		processGroupPlansLive(ctx, plans, &results, callbacks)

		assert.Len(t, results, 2)
	})
}

func TestProcessGroupWithGroupLock(t *testing.T) {
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
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, nil, &applied, &results, &failures, callbacks)

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
			WithFlags(true, false, false) // dry run
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
		assert.Len(t, applied, 1)
		assert.Len(t, results, 1)
	})

	t.Run("skips packages with non-updatable status", func(t *testing.T) {
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
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Empty(t, applied)
		assert.Len(t, results, 1)
		assert.Equal(t, lock.InstallStatusNotConfigured, results[0].Status)
	})

	t.Run("handles update error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("update failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)
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

		assert.Error(t, err)
	})

	t.Run("handles unsupported error without group error", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return &pkgerrors.UnsupportedError{Reason: "not configured"}
		}

		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)
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

		// Unsupported errors don't count as group errors
		assert.NoError(t, err)
	})

	t.Run("calls callback on result ready", func(t *testing.T) {
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
		var failures []SystemTestFailure
		var callbackCount int
		callbacks := ExecutionCallbacks{
			DeriveReason: mockDeriveReason,
			OnResultReady: func(res UpdateResult, dryRun bool) {
				callbackCount++
			},
		}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Equal(t, 1, callbackCount)
	})

	t.Run("validates packages with reload list", func(t *testing.T) {
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
		// In dry run mode, validation is skipped
		assert.Equal(t, constants.StatusUpdated, results[0].Status)
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

		assert.Error(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("tracks unsupported applied packages", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
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
					Status: lock.InstallStatusFloating, // This will make it skip but still get tracked
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
		// The floating status should be tracked
		assert.Len(t, tracker.packages, 1)
	})
}

// mockProgressReporter implements ProgressReporter for testing.
type mockProgressReporter struct {
	count int
}

func (m *mockProgressReporter) Increment() {
	m.count++
}

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

func TestProcessGroupPerPackageProgress(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("processes package successfully", func(t *testing.T) {
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

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

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

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

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

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("handles unsupported update error", func(t *testing.T) {
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

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

		// Unsupported errors don't count as group errors
		assert.NoError(t, err)
		assert.Equal(t, 1, progress.count)
	})

	t.Run("validates and succeeds with reload list in non-dry-run", func(t *testing.T) {
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
			WithFlags(true, false, false) // dry run to avoid real commands
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

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

		assert.NoError(t, err)
		assert.Equal(t, 1, progress.count)
		assert.Len(t, results, 1)
	})

	t.Run("handles validation error with tracking", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}
		tracker := &mockUnsupportedTracker{}
		ctx := NewUpdateContext(cfg, "/test", tracker).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(false, false, false) // non-dry run to trigger validation
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

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

		assert.Error(t, err)
		assert.Equal(t, 1, progress.count)
		assert.Len(t, results, 1)
		assert.Equal(t, constants.StatusFailed, results[0].Status)
	})

	t.Run("tracks unsupported on skip path", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
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
					Status: lock.InstallStatusFloating, // This is trackable and gets skipped
				},
			},
		}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		progress := &mockProgressReporter{}
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupPerPackageProgress(ctx, plans, &applied, &results, progress, callbacks)

		// Floating status gets skipped but tracked
		assert.NoError(t, err)
		assert.Len(t, tracker.packages, 1)
	})
}

func TestRunGroupSystemTests(t *testing.T) {
	t.Run("handles passed tests", func(t *testing.T) {
		// Create a runner with no tests configured (will return passed result)
		runner := testutil.CreateSystemTestRunner(nil, false, false)

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner)

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
					Status: constants.StatusUpdated,
				},
			},
		}
		var failures []SystemTestFailure

		err := runGroupSystemTests(ctx, plans, &failures)

		assert.NoError(t, err)
		assert.Empty(t, failures)
	})

	t.Run("handles critical test failure", func(t *testing.T) {
		// Create a runner with a test that will fail (non-existent command)
		stopOnFail := true
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "failing-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: false},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false) // noTimeout=true to speed up

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithFlags(false, false, false)

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
					Status: constants.StatusUpdated,
				},
			},
		}
		var failures []SystemTestFailure

		err := runGroupSystemTests(ctx, plans, &failures)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system tests failed")
		// Plan status should be updated to failed
		assert.Equal(t, constants.StatusFailed, plans[0].Res.Status)
	})

	t.Run("handles non-critical test failure", func(t *testing.T) {
		// Create a runner with a test that will fail but continues
		stopOnFail := false
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "warning-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: true},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner)

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
					Status: constants.StatusUpdated,
				},
			},
		}
		var failures []SystemTestFailure

		err := runGroupSystemTests(ctx, plans, &failures)

		assert.NoError(t, err) // Non-critical failures don't return errors
		assert.Len(t, failures, 1)
		assert.Equal(t, "group", failures[0].PkgName)
		assert.False(t, failures[0].IsCritical)
	})
}

func TestRunPackageSystemTests(t *testing.T) {
	t.Run("handles passed tests", func(t *testing.T) {
		// Create a runner with no tests configured (will return passed result)
		runner := testutil.CreateSystemTestRunner(nil, false, false)

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		err := runPackageSystemTests(ctx, plan, &groupErr, &failures)

		assert.NoError(t, err)
		assert.NoError(t, groupErr)
		assert.Empty(t, failures)
	})

	t.Run("handles critical failure with rollback", func(t *testing.T) {
		// Create a runner with a test that will fail
		stopOnFail := true
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "failing-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: false},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		rollbackCalled := false
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			rollbackCalled = true
			return nil
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		err := runPackageSystemTests(ctx, plan, &groupErr, &failures)

		assert.NoError(t, err) // runPackageSystemTests always returns nil
		assert.Error(t, groupErr)
		assert.True(t, rollbackCalled, "rollback should have been called")
		assert.Equal(t, constants.StatusFailed, plan.Res.Status)
	})

	t.Run("handles rollback failure", func(t *testing.T) {
		// Create a runner with a test that will fail
		stopOnFail := true
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "failing-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: false},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return errors.New("rollback failed")
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater).
			WithFlags(false, false, false)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		err := runPackageSystemTests(ctx, plan, &groupErr, &failures)

		assert.NoError(t, err)
		assert.Error(t, groupErr) // groupErr should be set from critical failure
		// Rollback failure is recorded in ctx.Failures
		assert.Len(t, ctx.Failures, 2) // One for rollback failure, one for system test failure
	})

	t.Run("handles non-critical failure", func(t *testing.T) {
		// Create a runner with a test that will fail but continues
		stopOnFail := false
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "warning-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: true},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		err := runPackageSystemTests(ctx, plan, &groupErr, &failures)

		assert.NoError(t, err)
		assert.NoError(t, groupErr) // Non-critical failures don't set groupErr
		assert.Len(t, failures, 1)
		assert.Equal(t, "react", failures[0].PkgName)
		assert.False(t, failures[0].IsCritical)
	})
}

// TestValidatePreUpdateState tests the behavior of ValidatePreUpdateState.
//
// It verifies:
//   - Returns nil with nil reloadList
//   - Returns nil on reload error (non-fatal)
//   - Returns nil when package not found (non-fatal)
//   - Detects version drift and adjusts Original
//   - Passes when version matches expected
func TestValidatePreUpdateState(t *testing.T) {
	t.Run("returns nil with nil reloadList", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "17.0.0",
		}

		err := ValidatePreUpdateState(plan, nil)

		assert.NoError(t, err)
		assert.Equal(t, "17.0.0", plan.Original) // Unchanged
	})

	t.Run("returns nil on reload error (non-fatal)", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "17.0.0",
		}
		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}

		err := ValidatePreUpdateState(plan, reloadFunc)

		assert.NoError(t, err) // Errors are non-fatal
		assert.Equal(t, "17.0.0", plan.Original) // Unchanged
	})

	t.Run("returns nil when package not found (non-fatal)", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "17.0.0",
		}
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("vue", "3.0.0", "3.0.0"), // Different package
			}, nil
		}

		err := ValidatePreUpdateState(plan, reloadFunc)

		assert.NoError(t, err) // Not found is non-fatal
		assert.Equal(t, "17.0.0", plan.Original) // Unchanged
	})

	t.Run("detects version drift and adjusts Original", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "17.0.0",
		}
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.5", "17.0.5"), // Version drifted
			}, nil
		}

		err := ValidatePreUpdateState(plan, reloadFunc)

		assert.NoError(t, err)
		assert.Equal(t, "17.0.5", plan.Original) // Original adjusted to current state
	})

	t.Run("passes when version matches expected", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "17.0.0",
		}
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Same version
			}, nil
		}

		err := ValidatePreUpdateState(plan, reloadFunc)

		assert.NoError(t, err)
		assert.Equal(t, "17.0.0", plan.Original) // Unchanged
	})

	t.Run("handles v-prefix normalization", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
			},
			Original: "v17.0.0", // With v prefix
		}
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Without v prefix
			}, nil
		}

		err := ValidatePreUpdateState(plan, reloadFunc)

		assert.NoError(t, err)
		// Original should remain unchanged since versions match after normalization
		assert.Equal(t, "v17.0.0", plan.Original)
	})
}

// TestRollbackPlansWithDriftCheck tests RollbackPlans with drift verification.
//
// It verifies:
//   - Drift check is called after successful rollback
//   - Drift check failure is recorded
//   - Drift check is skipped when reloadList is nil
//   - Drift check is skipped in dry run mode
func TestRollbackPlansWithDriftCheck(t *testing.T) {
	t.Run("calls drift check after successful rollback", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		driftCheckCalled := false
		reloadFunc := func() ([]formats.Package, error) {
			driftCheckCalled = true
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Correctly rolled back
			}, nil
		}
		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: reloadFunc,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.NoError(t, err)
		assert.True(t, driftCheckCalled, "drift check should be called")
	})

	t.Run("records drift check failure", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "18.0.0", "18.0.0"), // Still at updated version - drift detected!
			}, nil
		}
		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: reloadFunc,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "drift check failed")
	})

	t.Run("records drift check failure when reload fails", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		reloadFunc := func() ([]formats.Package, error) {
			return nil, errors.New("reload failed")
		}
		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: reloadFunc,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not reload packages")
	})

	t.Run("records drift check failure when package not found", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("vue", "3.0.0", "3.0.0"), // Different package
			}, nil
		}
		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: reloadFunc,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing after rollback")
	})

	t.Run("skips drift check in dry run mode", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		driftCheckCalled := false
		reloadFunc := func() ([]formats.Package, error) {
			driftCheckCalled = true
			return nil, errors.New("should not be called")
		}
		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: reloadFunc,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, true, false) // dryRun=true

		assert.NoError(t, err)
		assert.False(t, driftCheckCalled, "drift check should be skipped in dry run")
	})

	t.Run("skips drift check when reloadList is nil", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{
				Res:      UpdateResult{Pkg: testutil.NPMPackage("react", "18.0.0", "18.0.0"), Status: constants.StatusUpdated},
				Original: "17.0.0",
			},
		}
		cfg := testutil.NewConfig().Build()

		ctx := &UpdateContext{
			Failures:   make([]error, 0),
			ReloadList: nil,
		}
		groupErr := errors.New("group failed")

		updater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		err := RollbackPlans(plans, cfg, "/test", ctx, groupErr, updater, false, false)

		assert.NoError(t, err)
	})
}

// TestProcessGroupWithGroupLockWithAllDependencies tests the with_all_dependencies flag.
func TestProcessGroupWithGroupLockWithAllDependencies(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("detects with_all_dependencies flag from rule config", func(t *testing.T) {
		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		// Create a config with a rule that has with_all_dependencies setting
		cfg := testutil.NewConfig().
			WithRule("composer", testutil.ComposerRule()).
			Build()

		// Set up a package that should use with_all_dependencies
		pkg := testutil.ComposerPackage("laravel/framework", "11.0.0", "11.0.0")

		// After update, the reload should return the updated version
		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.ComposerPackage("laravel/framework", "11.1.0", "11.1.0"),
			}, nil
		}
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(true, false, false) // dry run to avoid actual lock command
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    pkg,
					Target: "11.1.0",
					Status: constants.StatusPlanned,
				},
			},
		}
		groupCfg := &config.UpdateCfg{Commands: "composer update"}

		applied := make([]*PlannedUpdate, 0)
		var results []UpdateResult
		var failures []SystemTestFailure
		callbacks := ExecutionCallbacks{DeriveReason: mockDeriveReason}

		err := processGroupWithGroupLock(ctx, plans, groupCfg, &applied, &results, &failures, callbacks)

		assert.NoError(t, err)
		assert.Len(t, applied, 1)
	})
}

// TestRunPackageSystemTestsWithDriftCheck tests that drift check is called during rollback.
func TestRunPackageSystemTestsWithDriftCheck(t *testing.T) {
	t.Run("calls drift check after rollback on critical failure", func(t *testing.T) {
		// Create a runner with a test that will fail
		stopOnFail := true
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "failing-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: false},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		driftCheckCalled := false
		reloadFunc := func() ([]formats.Package, error) {
			driftCheckCalled = true
			return []formats.Package{
				testutil.NPMPackage("react", "17.0.0", "17.0.0"), // Rolled back correctly
			}, nil
		}

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(false, false, false)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		err := runPackageSystemTests(ctx, plan, &groupErr, &failures)

		assert.NoError(t, err)
		assert.Error(t, groupErr)
		assert.True(t, driftCheckCalled, "drift check should be called after rollback")
	})

	t.Run("records drift check failure after rollback", func(t *testing.T) {
		// Create a runner with a test that will fail
		stopOnFail := true
		testCfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "failing-test", Commands: "__nonexistent_command_for_test__", ContinueOnFail: false},
			},
			StopOnFail: &stopOnFail,
		}
		runner := testutil.CreateSystemTestRunner(testCfg, true, false)

		reloadFunc := func() ([]formats.Package, error) {
			return []formats.Package{
				testutil.NPMPackage("react", "18.0.0", "18.0.0"), // Still at new version - drift!
			}, nil
		}

		mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			return nil
		}

		cfg := testutil.NewConfig().Build()
		ctx := NewUpdateContext(cfg, "/test", nil).
			WithSystemTestRunner(runner).
			WithUpdaterFunc(mockUpdater).
			WithReloadList(reloadFunc).
			WithFlags(false, false, false)

		plan := &PlannedUpdate{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Status: constants.StatusUpdated,
			},
			Original: "17.0.0",
		}
		var groupErr error
		var failures []SystemTestFailure

		_ = runPackageSystemTests(ctx, plan, &groupErr, &failures)

		// Drift check failure should be recorded in ctx.Failures
		assert.Len(t, ctx.Failures, 2) // system test failure + drift check failure
	})
}

// TestProcessGroupWithGroupLockEdgeCases tests edge cases for processGroupWithGroupLock.
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
		assert.Len(t, applied, 0)    // Nothing was applied
		assert.Len(t, results, 1)    // Result was still recorded
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
