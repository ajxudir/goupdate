package update

import (
	"errors"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/systemtest"
	"github.com/ajxudir/goupdate/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

// mockUnsupportedTracker is a simple mock for testing
type mockUnsupportedTracker struct {
	packages []formats.Package
	reasons  []string
}

func (m *mockUnsupportedTracker) Add(p formats.Package, reason string) {
	m.packages = append(m.packages, p)
	m.reasons = append(m.reasons, reason)
}

func (m *mockUnsupportedTracker) Messages() []string {
	return m.reasons
}

func TestNewUpdateContext(t *testing.T) {
	t.Run("creates context with required fields", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()
		tracker := &mockUnsupportedTracker{}

		ctx := NewUpdateContext(cfg, "/test/dir", tracker)

		assert.Equal(t, cfg, ctx.Cfg)
		assert.Equal(t, "/test/dir", ctx.WorkDir)
		assert.Equal(t, tracker, ctx.Unsupported)
		assert.NotNil(t, ctx.Failures)
		assert.Empty(t, ctx.Failures)
	})

	t.Run("creates context with nil tracker", func(t *testing.T) {
		cfg := testutil.NewConfig().Build()

		ctx := NewUpdateContext(cfg, "/test/dir", nil)

		assert.Nil(t, ctx.Unsupported)
	})
}

func TestUpdateContextWithFlags(t *testing.T) {
	t.Run("sets all flags", func(t *testing.T) {
		ctx := &UpdateContext{}

		result := ctx.WithFlags(true, true, true)

		assert.Same(t, ctx, result) // Returns same instance for chaining
		assert.True(t, ctx.DryRun)
		assert.True(t, ctx.ContinueOnError)
		assert.True(t, ctx.SkipLockRun)
	})

	t.Run("sets flags to false", func(t *testing.T) {
		ctx := &UpdateContext{DryRun: true, ContinueOnError: true, SkipLockRun: true}

		ctx.WithFlags(false, false, false)

		assert.False(t, ctx.DryRun)
		assert.False(t, ctx.ContinueOnError)
		assert.False(t, ctx.SkipLockRun)
	})
}

func TestUpdateContextWithBaseline(t *testing.T) {
	t.Run("sets baseline", func(t *testing.T) {
		ctx := &UpdateContext{}
		baseline := map[string]VersionSnapshot{
			"npm|js|prod|react": {Version: "17.0.0", Installed: "17.0.0"},
		}

		result := ctx.WithBaseline(baseline)

		assert.Same(t, ctx, result)
		assert.Equal(t, baseline, ctx.Baseline)
	})

	t.Run("sets nil baseline", func(t *testing.T) {
		ctx := &UpdateContext{Baseline: map[string]VersionSnapshot{}}

		ctx.WithBaseline(nil)

		assert.Nil(t, ctx.Baseline)
	})
}

func TestUpdateContextWithTable(t *testing.T) {
	t.Run("sets table", func(t *testing.T) {
		ctx := &UpdateContext{}
		table := output.NewTable()

		result := ctx.WithTable(table)

		assert.Same(t, ctx, result)
		assert.Equal(t, table, ctx.Table)
	})
}

func TestUpdateContextWithSystemTestRunner(t *testing.T) {
	t.Run("sets system test runner", func(t *testing.T) {
		ctx := &UpdateContext{}
		runner := systemtest.NewRunner(nil, "/test", false, false)

		result := ctx.WithSystemTestRunner(runner)

		assert.Same(t, ctx, result)
		assert.Equal(t, runner, ctx.SystemTestRunner)
	})
}

func TestUpdateContextWithReloadList(t *testing.T) {
	t.Run("sets reload function", func(t *testing.T) {
		ctx := &UpdateContext{}
		called := false
		reloadFn := func() ([]formats.Package, error) {
			called = true
			return nil, nil
		}

		result := ctx.WithReloadList(reloadFn)

		assert.Same(t, ctx, result)
		assert.NotNil(t, ctx.ReloadList)

		// Call the function to verify it was set
		_, _ = ctx.ReloadList()
		assert.True(t, called)
	})
}

func TestUpdateContextWithSelection(t *testing.T) {
	t.Run("sets selection flags", func(t *testing.T) {
		ctx := &UpdateContext{}
		selection := outdated.UpdateSelectionFlags{
			Major: true,
			Minor: false,
			Patch: false,
		}

		result := ctx.WithSelection(selection)

		assert.Same(t, ctx, result)
		assert.True(t, ctx.Selection.Major)
	})
}

func TestUpdateContextWithSkipSystemTests(t *testing.T) {
	t.Run("sets skip flag to true", func(t *testing.T) {
		ctx := &UpdateContext{}

		result := ctx.WithSkipSystemTests(true)

		assert.Same(t, ctx, result)
		assert.True(t, ctx.SkipSystemTests)
	})

	t.Run("sets skip flag to false", func(t *testing.T) {
		ctx := &UpdateContext{SkipSystemTests: true}

		ctx.WithSkipSystemTests(false)

		assert.False(t, ctx.SkipSystemTests)
	})
}

func TestUpdateContextWithIncrementalMode(t *testing.T) {
	t.Run("sets incremental mode to true", func(t *testing.T) {
		ctx := &UpdateContext{}

		result := ctx.WithIncrementalMode(true)

		assert.Same(t, ctx, result)
		assert.True(t, ctx.IncrementalMode)
	})

	t.Run("sets incremental mode to false", func(t *testing.T) {
		ctx := &UpdateContext{IncrementalMode: true}

		ctx.WithIncrementalMode(false)

		assert.False(t, ctx.IncrementalMode)
	})
}

func TestUpdateContextWithDeriveUnsupportedReason(t *testing.T) {
	t.Run("sets derive function", func(t *testing.T) {
		ctx := &UpdateContext{}
		called := false
		deriveFn := func(p formats.Package, cfg *config.Config, err error, latestMissing bool) string {
			called = true
			return "test reason"
		}

		result := ctx.WithDeriveUnsupportedReason(deriveFn)

		assert.Same(t, ctx, result)
		assert.NotNil(t, ctx.DeriveUnsupportedReason)

		// Call the function to verify it was set
		reason := ctx.DeriveUnsupportedReason(formats.Package{}, nil, nil, false)
		assert.True(t, called)
		assert.Equal(t, "test reason", reason)
	})
}

func TestUpdateContextWithUpdaterFunc(t *testing.T) {
	t.Run("sets updater function", func(t *testing.T) {
		ctx := &UpdateContext{}
		called := false
		updaterFn := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
			called = true
			return nil
		}

		result := ctx.WithUpdaterFunc(updaterFn)

		assert.Same(t, ctx, result)
		assert.NotNil(t, ctx.UpdaterFunc)

		// Call the function to verify it was set
		_ = ctx.UpdaterFunc(formats.Package{}, "1.0.0", nil, "/test", false, false)
		assert.True(t, called)
	})
}

func TestShouldRunSystemTestsAfterEach(t *testing.T) {
	t.Run("returns false when no runner", func(t *testing.T) {
		ctx := &UpdateContext{}

		assert.False(t, ctx.ShouldRunSystemTestsAfterEach())
	})

	t.Run("returns false when skip system tests is true", func(t *testing.T) {
		cfg := &config.SystemTestsCfg{
			RunMode: config.SystemTestRunModeAfterEach,
			Tests:   []config.SystemTestCfg{{Name: "test", Commands: "echo test"}},
		}
		ctx := &UpdateContext{
			SystemTestRunner: systemtest.NewRunner(cfg, "/test", false, false),
			SkipSystemTests:  true,
		}

		assert.False(t, ctx.ShouldRunSystemTestsAfterEach())
	})

	t.Run("returns false when runner says don't run after each", func(t *testing.T) {
		cfg := &config.SystemTestsCfg{
			RunMode: config.SystemTestRunModeAfterAll,
			Tests:   []config.SystemTestCfg{{Name: "test", Commands: "echo test"}},
		}
		ctx := &UpdateContext{
			SystemTestRunner: systemtest.NewRunner(cfg, "/test", false, false),
			SkipSystemTests:  false,
		}

		assert.False(t, ctx.ShouldRunSystemTestsAfterEach())
	})

	t.Run("returns true when all conditions met", func(t *testing.T) {
		cfg := &config.SystemTestsCfg{
			RunMode: config.SystemTestRunModeAfterEach,
			Tests:   []config.SystemTestCfg{{Name: "test", Commands: "echo test"}},
		}
		ctx := &UpdateContext{
			SystemTestRunner: systemtest.NewRunner(cfg, "/test", false, false),
			SkipSystemTests:  false,
		}

		assert.True(t, ctx.ShouldRunSystemTestsAfterEach())
	})
}

func TestAppendFailure(t *testing.T) {
	t.Run("appends error to failures", func(t *testing.T) {
		ctx := &UpdateContext{Failures: make([]error, 0)}
		err := errors.New("test error")

		ctx.AppendFailure(err)

		assert.Len(t, ctx.Failures, 1)
		assert.Equal(t, err, ctx.Failures[0])
	})

	t.Run("appends multiple errors", func(t *testing.T) {
		ctx := &UpdateContext{Failures: make([]error, 0)}

		ctx.AppendFailure(errors.New("error 1"))
		ctx.AppendFailure(errors.New("error 2"))

		assert.Len(t, ctx.Failures, 2)
	})

	t.Run("ignores nil error", func(t *testing.T) {
		ctx := &UpdateContext{Failures: make([]error, 0)}

		ctx.AppendFailure(nil)

		assert.Empty(t, ctx.Failures)
	})
}

func TestPackageKey(t *testing.T) {
	tests := []struct {
		name     string
		pkg      formats.Package
		expected string
	}{
		{
			name:     "npm package",
			pkg:      formats.Package{Rule: "npm", PackageType: "js", Type: "prod", Name: "react"},
			expected: "npm|js|prod|react",
		},
		{
			name:     "go module",
			pkg:      formats.Package{Rule: "mod", PackageType: "golang", Type: "prod", Name: "github.com/example/pkg"},
			expected: "mod|golang|prod|github.com/example/pkg",
		},
		{
			name:     "dev dependency",
			pkg:      formats.Package{Rule: "npm", PackageType: "js", Type: "dev", Name: "jest"},
			expected: "npm|js|dev|jest",
		},
		{
			name:     "empty fields",
			pkg:      formats.Package{Name: "test"},
			expected: "|||test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := PackageKey(tt.pkg)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestSnapshotVersions(t *testing.T) {
	t.Run("creates snapshots for packages", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			testutil.NPMPackage("vue", "3.0.0", "3.0.0"),
		}

		snapshots := SnapshotVersions(packages)
		assert.Len(t, snapshots, 2)

		reactKey := PackageKey(packages[0])
		assert.Contains(t, snapshots, reactKey)
		assert.Equal(t, "17.0.0", snapshots[reactKey].Version)
		assert.Equal(t, "17.0.0", snapshots[reactKey].Installed)

		vueKey := PackageKey(packages[1])
		assert.Contains(t, snapshots, vueKey)
		assert.Equal(t, "3.0.0", snapshots[vueKey].Version)
		assert.Equal(t, "3.0.0", snapshots[vueKey].Installed)
	})

	t.Run("handles empty package list", func(t *testing.T) {
		snapshots := SnapshotVersions([]formats.Package{})
		assert.Empty(t, snapshots)
	})

	t.Run("handles nil package list", func(t *testing.T) {
		snapshots := SnapshotVersions(nil)
		assert.Empty(t, snapshots)
	})

	t.Run("preserves different installed versions", func(t *testing.T) {
		packages := []formats.Package{
			{
				Rule:             "npm",
				PackageType:      "js",
				Type:             "prod",
				Name:             "lodash",
				Version:          "4.17.0",
				InstalledVersion: "4.17.21",
			},
		}

		snapshots := SnapshotVersions(packages)
		key := PackageKey(packages[0])
		assert.Equal(t, "4.17.0", snapshots[key].Version)
		assert.Equal(t, "4.17.21", snapshots[key].Installed)
	})
}

func TestVersionSnapshot(t *testing.T) {
	t.Run("stores version and installed", func(t *testing.T) {
		snapshot := VersionSnapshot{
			Version:   "1.0.0",
			Installed: "1.0.0",
		}
		assert.Equal(t, "1.0.0", snapshot.Version)
		assert.Equal(t, "1.0.0", snapshot.Installed)
	})
}
