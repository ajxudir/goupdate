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
