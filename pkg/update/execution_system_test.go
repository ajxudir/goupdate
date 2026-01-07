package update

import (
	"errors"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

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

		assert.NoError(t, err)                   // Errors are non-fatal
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

		assert.NoError(t, err)                   // Not found is non-fatal
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
