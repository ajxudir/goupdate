package update

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	pkgerrors "github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/testutil"
)

func TestCompareGroups(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"both empty", "", "", 0},
		{"a has group", "frontend", "", -1},
		{"b has group", "", "backend", 1},
		{"same group", "frontend", "frontend", 0},
		{"a < b", "alpha", "beta", -1},
		{"a > b", "beta", "alpha", 1},
		{"whitespace trimmed", "  frontend  ", "frontend", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareGroups(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPackagesFromPlans(t *testing.T) {
	t.Run("extracts packages from resolved plans", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: testutil.NPMPackage("react", "17.0.0", "17.0.0")},
			{Pkg: testutil.NPMPackage("vue", "3.0.0", "3.0.0")},
		}

		result := ExtractPackagesFromPlans(plans)

		assert.Len(t, result, 2)
		assert.Equal(t, "react", result[0].Name)
		assert.Equal(t, "vue", result[1].Name)
	})

	t.Run("handles empty plans", func(t *testing.T) {
		result := ExtractPackagesFromPlans([]ResolvedUpdatePlan{})
		assert.Empty(t, result)
	})
}

func TestSortResolvedPlans(t *testing.T) {
	t.Run("sorts by rule first", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: formats.Package{Rule: "npm", Name: "a"}},
			{Pkg: formats.Package{Rule: "mod", Name: "b"}},
		}

		SortResolvedPlans(plans)

		assert.Equal(t, "mod", plans[0].Pkg.Rule)
		assert.Equal(t, "npm", plans[1].Pkg.Rule)
	})

	t.Run("sorts by package type within same rule", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Name: "a"}},
			{Pkg: formats.Package{Rule: "npm", PackageType: "golang", Name: "b"}},
		}

		SortResolvedPlans(plans)

		assert.Equal(t, "golang", plans[0].Pkg.PackageType)
		assert.Equal(t, "js", plans[1].Pkg.PackageType)
	})

	t.Run("sorts by group within same package type", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Group: "", Name: "a"}},
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Group: "frontend", Name: "b"}},
		}

		SortResolvedPlans(plans)

		assert.Equal(t, "frontend", plans[0].Pkg.Group)
		assert.Equal(t, "", plans[1].Pkg.Group)
	})

	t.Run("sorts by type within same group", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Group: "frontend", Type: "prod", Name: "a"}},
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Group: "frontend", Type: "dev", Name: "b"}},
		}

		SortResolvedPlans(plans)

		assert.Equal(t, "dev", plans[0].Pkg.Type)
		assert.Equal(t, "prod", plans[1].Pkg.Type)
	})

	t.Run("sorts by name within same type", func(t *testing.T) {
		plans := []ResolvedUpdatePlan{
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react"}},
			{Pkg: formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "angular"}},
		}

		SortResolvedPlans(plans)

		assert.Equal(t, "angular", plans[0].Pkg.Name)
		assert.Equal(t, "react", plans[1].Pkg.Name)
	})
}

func TestCountPendingUpdates(t *testing.T) {
	t.Run("counts plans with target set", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Target: "1.1.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Target: "", Status: constants.StatusUpToDate}},
		}

		result := CountPendingUpdates(plans)

		assert.Equal(t, 2, result)
	})

	t.Run("excludes non-updatable statuses", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Target: "2.0.0", Status: lock.InstallStatusNotConfigured}},
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusFailed}},
		}

		result := CountPendingUpdates(plans)

		assert.Equal(t, 1, result)
	})

	t.Run("handles empty plans", func(t *testing.T) {
		result := CountPendingUpdates([]*PlannedUpdate{})
		assert.Equal(t, 0, result)
	})
}

func TestIsNonUpdatableStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"not configured", lock.InstallStatusNotConfigured, true},
		{"floating", lock.InstallStatusFloating, true},
		{"config error", constants.StatusConfigError, true},
		{"failed", constants.StatusFailed, true},
		{"summarize error", constants.StatusSummarizeError, true},
		{"updated", constants.StatusUpdated, false},
		{"up to date", constants.StatusUpToDate, false},
		{"planned", constants.StatusPlanned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNonUpdatableStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipUpdate(t *testing.T) {
	t.Run("skips when non-updatable status", func(t *testing.T) {
		res := &UpdateResult{Status: lock.InstallStatusNotConfigured, Target: "2.0.0"}
		assert.True(t, ShouldSkipUpdate(res))
	})

	t.Run("skips when no target", func(t *testing.T) {
		res := &UpdateResult{Status: constants.StatusUpToDate, Target: ""}
		assert.True(t, ShouldSkipUpdate(res))
	})

	t.Run("does not skip when has target and updatable status", func(t *testing.T) {
		res := &UpdateResult{Status: constants.StatusPlanned, Target: "2.0.0"}
		assert.False(t, ShouldSkipUpdate(res))
	})
}

func TestIsFloatingConstraint(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"star", "*", true},
		{"semver", "1.0.0", false},
		{"caret", "^1.0.0", false},
		{"tilde", "~1.0.0", false},
		{"latest not floating", "latest", false},
		{"empty not floating", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := formats.Package{Version: tt.version}
			result := IsFloatingConstraint(pkg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRefreshAvailableVersions(t *testing.T) {
	t.Run("refreshes available versions based on target", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Target: "1.5.0",
				Major:  "2.0.0",
				Minor:  "1.1.0",
				Patch:  "1.0.1",
			},
			VersionsInConstraint: []string{"1.0.0", "1.0.1", "1.1.0", "1.5.0", "2.0.0"},
			Versioning:           nil,
			Incremental:          false,
		}

		RefreshAvailableVersions(plan)

		// After refresh, versions should be relative to target 1.5.0
		assert.NotEmpty(t, plan.Res.Major) // 2.0.0 still available
	})

	t.Run("no-op when no versions in constraint", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Target: "2.0.0",
				Major:  "original",
			},
		}

		RefreshAvailableVersions(plan)

		assert.Equal(t, "original", plan.Res.Major)
	})

	t.Run("no-op when no target", func(t *testing.T) {
		plan := &PlannedUpdate{
			Res: UpdateResult{
				Target: "",
				Major:  "original",
			},
			VersionsInConstraint: []string{"1.0.0", "2.0.0"},
		}

		RefreshAvailableVersions(plan)

		assert.Equal(t, "original", plan.Res.Major)
	})
}

func TestResolvePackagePlans(t *testing.T) {
	t.Run("resolves plans for packages", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		cfg := testutil.NewConfig().Build()

		resolver := func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
			return &config.UpdateCfg{Commands: "npm install"}, nil
		}

		result := ResolvePackagePlans(packages, cfg, resolver)

		assert.Len(t, result, 1)
		assert.Equal(t, "react", result[0].Pkg.Name)
		assert.NotNil(t, result[0].Cfg)
		assert.Nil(t, result[0].Err)
	})

	t.Run("captures config errors", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		cfg := testutil.NewConfig().Build()

		resolver := func(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error) {
			return nil, assert.AnError
		}

		result := ResolvePackagePlans(packages, cfg, resolver)

		assert.Len(t, result, 1)
		assert.Nil(t, result[0].Cfg)
		assert.Equal(t, assert.AnError, result[0].Err)
	})
}

func TestBuildGroupedPlans(t *testing.T) {
	mockVersionLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
		return []string{"1.0.0", "1.1.0", "2.0.0"}, nil
	}

	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "derived reason"
	}

	t.Run("builds plans for resolved packages", func(t *testing.T) {
		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		resolved := []ResolvedUpdatePlan{
			{Pkg: testutil.NPMPackage("react", "1.0.0", "1.0.0"), Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, "react", plans[0].Res.Pkg.Name)
	})

	t.Run("handles config errors as unsupported", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(cfg, "/test", tracker)
		resolved := []ResolvedUpdatePlan{
			{Pkg: testutil.NPMPackage("react", "1.0.0", "1.0.0"), Err: &pkgerrors.UnsupportedError{Reason: "no config"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, lock.InstallStatusNotConfigured, plans[0].Res.Status)
		assert.Len(t, tracker.packages, 1)
	})

	t.Run("handles config errors as failed", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		resolved := []ResolvedUpdatePlan{
			{Pkg: testutil.NPMPackage("react", "1.0.0", "1.0.0"), Err: errors.New("config error")},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, constants.StatusConfigError, plans[0].Res.Status)
		assert.Len(t, updateCtx.Failures, 1)
	})

	t.Run("handles floating constraints", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(cfg, "/test", tracker)
		pkg := formats.Package{Name: "react", Rule: "npm", Version: "*"}
		resolved := []ResolvedUpdatePlan{
			{Pkg: pkg, Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, lock.InstallStatusFloating, plans[0].Res.Status)
		assert.Len(t, tracker.packages, 1)
	})

	t.Run("handles exact constraints via = constraint", func(t *testing.T) {
		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("=").Build()
		resolved := []ResolvedUpdatePlan{
			{Pkg: pkg, Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, constants.StatusUpToDate, plans[0].Res.Status)
		assert.Equal(t, "1.0.0", plans[0].Res.Target) // Exact constraint preserves version
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		resolved := []ResolvedUpdatePlan{
			{Pkg: testutil.NPMPackage("react", "1.0.0", "1.0.0"), Cfg: &config.UpdateCfg{Commands: "npm install"}},
			{Pkg: testutil.NPMPackage("vue", "2.0.0", "2.0.0"), Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		plans := BuildGroupedPlans(ctx, resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		// Should return empty or partial results due to cancellation
		assert.Empty(t, plans)
	})

	t.Run("handles version listing errors", func(t *testing.T) {
		errorLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return nil, errors.New("version list failed")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		resolved := []ResolvedUpdatePlan{
			{Pkg: pkg, Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, errorLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, constants.StatusFailed, plans[0].Res.Status)
		assert.Len(t, updateCtx.Failures, 1)
	})

	t.Run("handles unsupported version listing errors", func(t *testing.T) {
		unsupportedLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return nil, &pkgerrors.UnsupportedError{Reason: "not supported"}
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(cfg, "/test", tracker)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		resolved := []ResolvedUpdatePlan{
			{Pkg: pkg, Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, unsupportedLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		assert.Equal(t, lock.InstallStatusNotConfigured, plans[0].Res.Status)
		assert.Len(t, tracker.packages, 1)
	})

	t.Run("determines target version from available versions", func(t *testing.T) {
		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		resolved := []ResolvedUpdatePlan{
			{Pkg: pkg, Cfg: &config.UpdateCfg{Commands: "npm install"}},
		}

		plans := BuildGroupedPlans(context.Background(), resolved, updateCtx, PlanningOptions{}, mockVersionLister, mockDeriveReason)

		assert.Len(t, plans, 1)
		// Should have available versions and a target
		assert.NotEmpty(t, plans[0].Res.Available)
	})
}

func TestHandleConfigErrorInternal(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("handles unsupported error", func(t *testing.T) {
		pkg := testutil.NPMPackage("react", "1.0.0", "1.0.0")
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(testutil.NewConfig().Build(), "/test", tracker)
		unsupportedErr := &pkgerrors.UnsupportedError{Reason: "no config"}

		result := handleConfigError(pkg, unsupportedErr, updateCtx, "1.0.0", mockDeriveReason)

		assert.Equal(t, lock.InstallStatusNotConfigured, result.Res.Status)
		assert.Len(t, tracker.packages, 1)
		assert.Empty(t, updateCtx.Failures)
	})

	t.Run("handles regular error", func(t *testing.T) {
		pkg := testutil.NPMPackage("react", "1.0.0", "1.0.0")
		updateCtx := NewUpdateContext(testutil.NewConfig().Build(), "/test", nil)
		regularErr := errors.New("config error")

		result := handleConfigError(pkg, regularErr, updateCtx, "1.0.0", mockDeriveReason)

		assert.Equal(t, constants.StatusConfigError, result.Res.Status)
		assert.Equal(t, regularErr, result.Res.Err)
		assert.Len(t, updateCtx.Failures, 1)
	})
}

func TestHandleFloatingConstraintInternal(t *testing.T) {
	t.Run("marks package as floating", func(t *testing.T) {
		pkg := formats.Package{Name: "react", Rule: "npm", Version: "*"}
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(testutil.NewConfig().Build(), "/test", tracker)
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := handleFloatingConstraint(pkg, updateCfg, updateCtx, "*")

		assert.Equal(t, lock.InstallStatusFloating, result.Res.Status)
		assert.Equal(t, "*", result.Original)
		assert.Len(t, tracker.packages, 1)
	})

	t.Run("handles nil tracker", func(t *testing.T) {
		pkg := formats.Package{Name: "react", Rule: "npm", Version: "*"}
		updateCtx := NewUpdateContext(testutil.NewConfig().Build(), "/test", nil)
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := handleFloatingConstraint(pkg, updateCfg, updateCtx, "*")

		assert.Equal(t, lock.InstallStatusFloating, result.Res.Status)
	})
}

func TestHandleExactConstraintInternal(t *testing.T) {
	t.Run("returns up to date with target as current version", func(t *testing.T) {
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("").Build()
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := handleExactConstraint(pkg, updateCfg, "1.0.0")

		assert.Equal(t, constants.StatusUpToDate, result.Res.Status)
		assert.Equal(t, "1.0.0", result.Res.Target)
		assert.Equal(t, "1.0.0", result.Original)
	})
}

func TestPlanVersionUpdateInternal(t *testing.T) {
	mockDeriveReason := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
		return "test reason"
	}

	t.Run("creates plan with available versions", func(t *testing.T) {
		versionLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return []string{"1.0.0", "1.1.0", "2.0.0"}, nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		res := UpdateResult{Pkg: pkg, Status: constants.StatusUpToDate}
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := planVersionUpdate(context.Background(), pkg, res, updateCfg, updateCtx, "1.0.0", PlanningOptions{}, versionLister, mockDeriveReason)

		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Res.Available)
		assert.NotEmpty(t, result.VersionsInConstraint)
	})

	t.Run("handles version listing error", func(t *testing.T) {
		errorLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return nil, errors.New("failed to list")
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		res := UpdateResult{Pkg: pkg, Status: constants.StatusUpToDate}
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := planVersionUpdate(context.Background(), pkg, res, updateCfg, updateCtx, "1.0.0", PlanningOptions{}, errorLister, mockDeriveReason)

		assert.Equal(t, constants.StatusFailed, result.Res.Status)
		assert.Len(t, updateCtx.Failures, 1)
	})

	t.Run("handles unsupported version listing error", func(t *testing.T) {
		unsupportedLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return nil, &pkgerrors.UnsupportedError{Reason: "not supported"}
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		tracker := &mockUnsupportedTracker{}
		updateCtx := NewUpdateContext(cfg, "/test", tracker)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		res := UpdateResult{Pkg: pkg, Status: constants.StatusUpToDate}
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := planVersionUpdate(context.Background(), pkg, res, updateCfg, updateCtx, "1.0.0", PlanningOptions{}, unsupportedLister, mockDeriveReason)

		assert.Equal(t, lock.InstallStatusNotConfigured, result.Res.Status)
		assert.Len(t, tracker.packages, 1)
	})

	t.Run("uses incremental mode from options", func(t *testing.T) {
		versionLister := func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
			return []string{"1.0.0", "1.0.1", "1.1.0"}, nil
		}

		cfg := testutil.NewConfig().WithRule("npm", testutil.NPMRule()).Build()
		updateCtx := NewUpdateContext(cfg, "/test", nil)
		pkg := testutil.NewPackage("react").WithRule("npm").WithVersion("1.0.0").WithConstraint("^").Build()
		res := UpdateResult{Pkg: pkg, Status: constants.StatusUpToDate}
		updateCfg := &config.UpdateCfg{Commands: "npm install"}

		result := planVersionUpdate(context.Background(), pkg, res, updateCfg, updateCtx, "1.0.0", PlanningOptions{IncrementalMode: true}, versionLister, mockDeriveReason)

		assert.True(t, result.Incremental)
	})
}
