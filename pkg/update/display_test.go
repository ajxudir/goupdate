package update

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/systemtest"
	"github.com/ajxudir/goupdate/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

// Note: We use lock.InstallStatusNotConfigured and lock.InstallStatusFloating in tests
// Note: Tests for FormatStatus, SafeDeclaredValue, SafeInstalledValue, HasAvailableUpdates,
// FormatAvailableVersions, and FormatAvailableVersionsUpToDate are in pkg/display

func TestFormatConstraintDisplay(t *testing.T) {
	t.Run("with major flag", func(t *testing.T) {
		selection := outdated.UpdateSelectionFlags{Major: true}
		result := FormatConstraintDisplay(formats.Package{}, selection)
		assert.Equal(t, "Major (--major)", result)
	})

	t.Run("with minor flag", func(t *testing.T) {
		selection := outdated.UpdateSelectionFlags{Minor: true}
		result := FormatConstraintDisplay(formats.Package{}, selection)
		assert.Equal(t, "Minor (--minor)", result)
	})

	t.Run("with patch flag", func(t *testing.T) {
		selection := outdated.UpdateSelectionFlags{Patch: true}
		result := FormatConstraintDisplay(formats.Package{}, selection)
		assert.Equal(t, "Patch (--patch)", result)
	})

	t.Run("without flags uses package constraint", func(t *testing.T) {
		pkg := formats.Package{Constraint: "^"}
		selection := outdated.UpdateSelectionFlags{}
		result := FormatConstraintDisplay(pkg, selection)
		assert.Contains(t, result, "Compatible")
	})
}

func TestSafeFromVersion(t *testing.T) {
	t.Run("uses original installed if present", func(t *testing.T) {
		res := UpdateResult{
			OriginalInstalled: "1.0.0",
			Pkg:               formats.Package{Version: "2.0.0"},
		}
		result := SafeFromVersion(res)
		assert.Equal(t, "1.0.0", result)
	})

	t.Run("uses package version if original installed is empty", func(t *testing.T) {
		res := UpdateResult{
			OriginalInstalled: "",
			Pkg:               formats.Package{Version: "2.0.0"},
		}
		result := SafeFromVersion(res)
		assert.Equal(t, "2.0.0", result)
	})

	t.Run("uses package version if original installed is N/A", func(t *testing.T) {
		res := UpdateResult{
			OriginalInstalled: constants.PlaceholderNA,
			Pkg:               formats.Package{Version: "2.0.0"},
		}
		result := SafeFromVersion(res)
		assert.Equal(t, "2.0.0", result)
	})

	t.Run("uses original version if original installed is N/A but original version set", func(t *testing.T) {
		res := UpdateResult{
			OriginalInstalled: constants.PlaceholderNA,
			OriginalVersion:   "1.5.0",
			Pkg:               formats.Package{Version: "2.0.0"},
		}
		result := SafeFromVersion(res)
		assert.Equal(t, "1.5.0", result)
	})

	t.Run("uses original version if original installed is empty but original version set", func(t *testing.T) {
		res := UpdateResult{
			OriginalInstalled: "",
			OriginalVersion:   "1.5.0",
			Pkg:               formats.Package{Version: "2.0.0"},
		}
		result := SafeFromVersion(res)
		assert.Equal(t, "1.5.0", result)
	})
}

func TestDetermineScopeDescription(t *testing.T) {
	tests := []struct {
		name      string
		selection outdated.UpdateSelectionFlags
		expected  string
	}{
		{"major flag", outdated.UpdateSelectionFlags{Major: true}, "--major scope"},
		{"minor flag", outdated.UpdateSelectionFlags{Minor: true}, "--minor scope"},
		{"patch flag", outdated.UpdateSelectionFlags{Patch: true}, "--patch scope"},
		{"no flags", outdated.UpdateSelectionFlags{}, "constraint scope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineScopeDescription(tt.selection)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeSummaryFromPlans(t *testing.T) {
	t.Run("counts packages to update", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusPlanned}},
			{Res: UpdateResult{Target: "1.1.0", Status: constants.StatusPlanned}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 2, counts.ToUpdate)
	})

	t.Run("counts up to date packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "", Status: constants.StatusUpToDate}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 1, counts.UpToDate)
	})

	t.Run("counts failed packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusFailed, Err: assert.AnError}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 1, counts.Failed)
	})

	t.Run("counts available updates", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "1.0.0", Major: "2.0.0", Minor: "1.1.0", Patch: "1.0.1"}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 1, counts.HasMajor)
		assert.Equal(t, 1, counts.HasMinor)
		assert.Equal(t, 1, counts.HasPatch)
	})

	t.Run("skips not configured packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: lock.InstallStatusNotConfigured}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.ToUpdate)
		assert.Equal(t, 0, counts.UpToDate)
	})

	t.Run("skips config error packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusConfigError}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.ToUpdate)
		assert.Equal(t, 0, counts.UpToDate)
	})

	t.Run("skips floating packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: lock.InstallStatusFloating}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.ToUpdate)
		assert.Equal(t, 0, counts.UpToDate)
	})

	t.Run("skips summarize error packages", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Status: constants.StatusSummarizeError}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.ToUpdate)
		assert.Equal(t, 0, counts.UpToDate)
	})

	t.Run("counts failed status with error", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusFailed, Err: assert.AnError}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 1, counts.Failed)
	})

	t.Run("counts failed status without error but with prefix", func(t *testing.T) {
		// When status is exactly StatusFailed, it counts as failed
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Status: constants.StatusFailed}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 1, counts.Failed)
	})

	t.Run("excludes N/A from available updates", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "1.0.0", Major: constants.PlaceholderNA, Minor: constants.PlaceholderNA, Patch: constants.PlaceholderNA}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.HasMajor)
		assert.Equal(t, 0, counts.HasMinor)
		assert.Equal(t, 0, counts.HasPatch)
	})

	t.Run("excludes versions matching target", func(t *testing.T) {
		plans := []*PlannedUpdate{
			{Res: UpdateResult{Target: "2.0.0", Major: "2.0.0", Minor: "2.0.0", Patch: "2.0.0"}},
		}
		counts := ComputeSummaryFromPlans(plans)
		assert.Equal(t, 0, counts.HasMajor)
		assert.Equal(t, 0, counts.HasMinor)
		assert.Equal(t, 0, counts.HasPatch)
	})
}

func TestComputeSummaryFromResults(t *testing.T) {
	t.Run("counts updated packages", func(t *testing.T) {
		results := []UpdateResult{
			{Status: constants.StatusUpdated},
			{Status: constants.StatusPlanned},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 2, counts.ToUpdate)
	})

	t.Run("counts up to date packages", func(t *testing.T) {
		results := []UpdateResult{
			{Status: constants.StatusUpToDate},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 1, counts.UpToDate)
	})

	t.Run("counts failed packages", func(t *testing.T) {
		results := []UpdateResult{
			{Status: constants.StatusFailed, Err: assert.AnError},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 1, counts.Failed)
	})

	t.Run("counts failed status prefix", func(t *testing.T) {
		results := []UpdateResult{
			{Status: "Failed: some reason"},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 1, counts.Failed)
	})

	t.Run("counts other status as up to date", func(t *testing.T) {
		results := []UpdateResult{
			{Status: "some_other_status"},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 1, counts.UpToDate)
	})

	t.Run("excludes not configured status", func(t *testing.T) {
		results := []UpdateResult{
			{Status: lock.InstallStatusNotConfigured},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 0, counts.UpToDate)
		assert.Equal(t, 0, counts.Failed)
	})

	t.Run("excludes floating status", func(t *testing.T) {
		results := []UpdateResult{
			{Status: lock.InstallStatusFloating},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 0, counts.UpToDate)
		assert.Equal(t, 0, counts.Failed)
	})

	t.Run("counts available updates", func(t *testing.T) {
		results := []UpdateResult{
			{Target: "1.0.0", Major: "2.0.0", Minor: "1.1.0", Patch: "1.0.1"},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 1, counts.HasMajor)
		assert.Equal(t, 1, counts.HasMinor)
		assert.Equal(t, 1, counts.HasPatch)
	})

	t.Run("excludes versions matching target", func(t *testing.T) {
		results := []UpdateResult{
			{Target: "2.0.0", Major: "2.0.0"},
		}
		counts := ComputeSummaryFromResults(results)
		assert.Equal(t, 0, counts.HasMajor)
	})
}

func TestFormatUpdateSummary(t *testing.T) {
	t.Run("preview mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5}
		updated, _, _, _, _ := FormatUpdateSummary(counts, SummaryModePreview)
		assert.Equal(t, "5 to update", updated)
	})

	t.Run("result mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5}
		updated, _, _, _, _ := FormatUpdateSummary(counts, SummaryModeResult)
		assert.Equal(t, "5 updated", updated)
	})

	t.Run("dry run mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5}
		updated, _, _, _, _ := FormatUpdateSummary(counts, SummaryModeDryRun)
		assert.Equal(t, "5 planned", updated)
	})

	t.Run("formats up to date", func(t *testing.T) {
		counts := UpdateSummaryCounts{UpToDate: 3}
		_, upToDate, _, _, _ := FormatUpdateSummary(counts, SummaryModeResult)
		assert.Equal(t, "3 up to date", upToDate)
	})

	t.Run("formats available updates", func(t *testing.T) {
		counts := UpdateSummaryCounts{HasMajor: 1, HasMinor: 2, HasPatch: 3}
		_, _, moreMajor, moreMinor, morePatch := FormatUpdateSummary(counts, SummaryModeResult)
		assert.Equal(t, "1 have major updates", moreMajor)
		assert.Equal(t, "2 have minor updates", moreMinor)
		assert.Equal(t, "3 have patch updates", morePatch)
	})

	t.Run("empty strings for zero counts", func(t *testing.T) {
		counts := UpdateSummaryCounts{}
		updated, upToDate, moreMajor, moreMinor, morePatch := FormatUpdateSummary(counts, SummaryModeResult)
		assert.Empty(t, updated)
		assert.Empty(t, upToDate)
		assert.Empty(t, moreMajor)
		assert.Empty(t, moreMinor)
		assert.Empty(t, morePatch)
	})
}

func TestFormatTestDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"milliseconds", 500 * time.Millisecond, "500ms"},
		{"one second", time.Second, "1.0s"},
		{"seconds", 2500 * time.Millisecond, "2.5s"},
		{"zero", 0, "0ms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTestDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatSummaryStrings(t *testing.T) {
	t.Run("preview mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5, UpToDate: 3}
		summary, _ := FormatSummaryStrings(counts, SummaryModePreview)
		assert.Contains(t, summary, "5 to update")
		assert.Contains(t, summary, "3 up-to-date")
	})

	t.Run("result mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5, Failed: 2}
		summary, _ := FormatSummaryStrings(counts, SummaryModeResult)
		assert.Contains(t, summary, "5 updated")
		assert.Contains(t, summary, "2 failed")
	})

	t.Run("dry run mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 5}
		summary, _ := FormatSummaryStrings(counts, SummaryModeDryRun)
		assert.Contains(t, summary, "5 planned")
	})

	t.Run("available updates line", func(t *testing.T) {
		counts := UpdateSummaryCounts{HasMajor: 1, HasMinor: 2}
		_, available := FormatSummaryStrings(counts, SummaryModeResult)
		// New format: always shows all three counts for regex-friendly parsing
		assert.Contains(t, available, "1 major")
		assert.Contains(t, available, "2 minor")
		assert.Contains(t, available, "0 patch")
		assert.Contains(t, available, "updates still available")
	})

	t.Run("preview mode available suffix", func(t *testing.T) {
		counts := UpdateSummaryCounts{HasMajor: 1}
		_, available := FormatSummaryStrings(counts, SummaryModePreview)
		assert.Contains(t, available, "1 major")
		assert.Contains(t, available, "available")
		assert.NotContains(t, available, "still")
	})

	t.Run("outdated mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 3, UpToDate: 5, HasMajor: 2, HasMinor: 1, HasPatch: 3}
		summary, available := FormatSummaryStrings(counts, SummaryModeOutdated)
		assert.Contains(t, summary, "3 outdated")
		assert.Contains(t, summary, "5 up-to-date")
		assert.Contains(t, available, "2 major")
		assert.Contains(t, available, "1 minor")
		assert.Contains(t, available, "3 patch")
		assert.Contains(t, available, "available")
		assert.NotContains(t, available, "still")
	})

	t.Run("zero counts shown for regex-friendly parsing", func(t *testing.T) {
		counts := UpdateSummaryCounts{ToUpdate: 1, HasMajor: 0, HasMinor: 0, HasPatch: 0}
		_, available := FormatSummaryStrings(counts, SummaryModeResult)
		// All counts should be shown, even zeros
		assert.Contains(t, available, "0 major")
		assert.Contains(t, available, "0 minor")
		assert.Contains(t, available, "0 patch")
	})
}

func TestBuildUpdateTableFromPackages(t *testing.T) {
	t.Run("creates table with correct columns", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		selection := outdated.UpdateSelectionFlags{}

		table := BuildUpdateTableFromPackages(packages, selection)

		assert.NotNil(t, table)
		// Table should have at least 9 columns (RULE, PM, TYPE, CONSTRAINT, VERSION, INSTALLED, TARGET, STATUS, NAME)
		assert.GreaterOrEqual(t, table.ColumnCount(), 9)
	})

	t.Run("shows group column when groups present", func(t *testing.T) {
		packages := []formats.Package{
			{Name: "react", Group: "frontend"},
		}
		selection := outdated.UpdateSelectionFlags{}

		table := BuildUpdateTableFromPackages(packages, selection)

		// Should have 10 columns when group is shown
		assert.Equal(t, 10, table.ColumnCount())
	})
}

func TestPrintUpdateRow(t *testing.T) {
	t.Run("prints update result row", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		table := BuildUpdateTableFromPackages(packages, outdated.UpdateSelectionFlags{})

		res := UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Target: "18.0.0",
			Status: constants.StatusUpdated,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateRow(res, table, false, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, "react")
		assert.Contains(t, output, "18.0.0")
		assert.Contains(t, output, constants.StatusUpdated)
	})

	t.Run("shows planned for dry run", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		table := BuildUpdateTableFromPackages(packages, outdated.UpdateSelectionFlags{})

		res := UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Target: "18.0.0",
			Status: constants.StatusUpdated,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateRow(res, table, true, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, constants.StatusPlanned)
	})

	t.Run("shows NA for empty target", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		table := BuildUpdateTableFromPackages(packages, outdated.UpdateSelectionFlags{})

		res := UpdateResult{
			Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
			Target: "",
			Status: constants.StatusUpToDate,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateRow(res, table, false, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, constants.PlaceholderNA)
	})
}

func TestPrintUpdatePreview(t *testing.T) {
	t.Run("prints preview with updates", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		table := BuildUpdateTableFromPackages(packages, outdated.UpdateSelectionFlags{})

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdatePreview(plans, table, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, "react")
		assert.Contains(t, output, "18.0.0")
	})

	t.Run("skips non-updatable packages", func(t *testing.T) {
		packages := []formats.Package{
			testutil.NPMPackage("react", "17.0.0", "17.0.0"),
		}
		table := BuildUpdateTableFromPackages(packages, outdated.UpdateSelectionFlags{})

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "18.0.0",
					Status: lock.InstallStatusNotConfigured,
				},
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdatePreview(plans, table, outdated.UpdateSelectionFlags{})
		})

		// Should show "No packages to update" or similar
		assert.NotContains(t, output, "Will update")
	})

	t.Run("shows packages with more available updates", func(t *testing.T) {
		table := testutil.CreateUpdateTable()

		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
					Target: "17.0.0", // Same as current, but has other updates available
					Status: constants.StatusUpToDate,
					Major:  "18.0.0", // Has major update available
				},
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdatePreview(plans, table, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, "Up to date")
		assert.Contains(t, output, "react")
	})

	t.Run("uses Version when InstalledVersion is empty", func(t *testing.T) {
		table := testutil.CreateUpdateTable()

		pkg := testutil.NPMPackage("react", "17.0.0", "")
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    pkg,
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdatePreview(plans, table, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, "react")
		assert.Contains(t, output, "17.0.0")
	})

	t.Run("uses Version when InstalledVersion is N/A", func(t *testing.T) {
		table := testutil.CreateUpdateTable()

		pkg := testutil.NPMPackage("react", "17.0.0", constants.PlaceholderNA)
		plans := []*PlannedUpdate{
			{
				Res: UpdateResult{
					Pkg:    pkg,
					Target: "18.0.0",
					Status: constants.StatusPlanned,
				},
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdatePreview(plans, table, outdated.UpdateSelectionFlags{})
		})

		assert.Contains(t, output, "react")
		assert.Contains(t, output, "17.0.0")
	})
}

func TestPrintUpdateSummary(t *testing.T) {
	t.Run("prints summary with updates", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Target: "18.0.0",
				Status: constants.StatusUpdated,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, false, nil)
		})

		assert.Contains(t, output, "Update Summary")
		assert.Contains(t, output, "react")
	})

	t.Run("prints dry run summary", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Target: "18.0.0",
				Status: constants.StatusPlanned,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, true, nil)
		})

		assert.Contains(t, output, "Planned updates")
	})

	t.Run("prints summary with failed updates", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
				Status: constants.StatusFailed,
				Err:    assert.AnError,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, false, nil)
		})

		assert.Contains(t, output, "Failed updates")
		assert.Contains(t, output, "react")
	})

	t.Run("prints remaining updates available", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Status: constants.StatusUpToDate,
				Major:  "18.0.0",
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, false, nil)
		})

		assert.Contains(t, output, "available")
	})

	t.Run("handles empty results", func(t *testing.T) {
		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary([]UpdateResult{}, false, nil)
		})

		assert.Empty(t, output)
	})

	t.Run("skips config error status", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
				Status: constants.StatusConfigError,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, false, nil)
		})

		// Config errors are skipped, so no content about the package
		assert.NotContains(t, output, "Successfully updated")
	})

	t.Run("skips floating status", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "*", "*"),
				Target: "",
				Status: lock.InstallStatusFloating,
			},
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummary(results, false, nil)
		})

		// Floating packages are skipped
		assert.NotContains(t, output, "react")
	})
}

func TestPrintUpdateSummaryLines(t *testing.T) {
	t.Run("prints summary lines for result mode", func(t *testing.T) {
		counts := UpdateSummaryCounts{
			ToUpdate: 2,
			Failed:   1,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummaryLines(counts, SummaryModeResult)
		})

		assert.NotEmpty(t, output)
	})

	t.Run("prints dry run summary lines", func(t *testing.T) {
		counts := UpdateSummaryCounts{
			ToUpdate: 3,
			UpToDate: 2,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummaryLines(counts, SummaryModeDryRun)
		})

		assert.NotEmpty(t, output)
	})

	t.Run("prints preview summary lines", func(t *testing.T) {
		counts := UpdateSummaryCounts{
			ToUpdate: 5,
			HasMajor: 1,
			HasMinor: 2,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateSummaryLines(counts, SummaryModePreview)
		})

		assert.NotEmpty(t, output)
	})
}

// mockTestInfo implements SystemTestInfo for testing
type mockTestInfo struct {
	name     string
	passed   bool
	duration time.Duration
	output   string
}

func (m *mockTestInfo) GetName() string            { return m.name }
func (m *mockTestInfo) GetPassed() bool            { return m.passed }
func (m *mockTestInfo) GetDuration() time.Duration { return m.duration }
func (m *mockTestInfo) GetOutput() string          { return m.output }

// mockSystemTestResult implements SystemTestResultFormatter for testing
type mockSystemTestResult struct {
	tests    []SystemTestInfo
	passed   bool
	duration time.Duration
}

func (m *mockSystemTestResult) TestCount() int { return len(m.tests) }
func (m *mockSystemTestResult) PassedCount() int {
	count := 0
	for _, t := range m.tests {
		if t.GetPassed() {
			count++
		}
	}
	return count
}
func (m *mockSystemTestResult) Passed() bool                 { return m.passed }
func (m *mockSystemTestResult) TotalDuration() time.Duration { return m.duration }
func (m *mockSystemTestResult) Tests() []SystemTestInfo      { return m.tests }

func TestPrintSystemTestSummary(t *testing.T) {
	t.Run("prints summary with all passed", func(t *testing.T) {
		result := &mockSystemTestResult{
			tests: []SystemTestInfo{
				&mockTestInfo{name: "test1", passed: true, duration: 100 * time.Millisecond},
				&mockTestInfo{name: "test2", passed: true, duration: 200 * time.Millisecond},
			},
			passed:   true,
			duration: 300 * time.Millisecond,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintSystemTestSummary(result, "")
		})

		assert.Contains(t, output, "System tests")
		assert.Contains(t, output, "2/2 passed")
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "test2")
	})

	t.Run("prints summary with failures", func(t *testing.T) {
		result := &mockSystemTestResult{
			tests: []SystemTestInfo{
				&mockTestInfo{name: "test1", passed: true, duration: 100 * time.Millisecond},
				&mockTestInfo{name: "test2", passed: false, duration: 200 * time.Millisecond, output: "error message"},
			},
			passed:   false,
			duration: 300 * time.Millisecond,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintSystemTestSummary(result, "  ")
		})

		assert.Contains(t, output, "1/2 passed")
		assert.Contains(t, output, "error message")
	})

	t.Run("handles nil result", func(t *testing.T) {
		output := testutil.CaptureStdout(t, func() {
			PrintSystemTestSummary(nil, "")
		})

		assert.Empty(t, output)
	})

	t.Run("handles empty tests", func(t *testing.T) {
		result := &mockSystemTestResult{
			tests:    []SystemTestInfo{},
			passed:   true,
			duration: 0,
		}

		output := testutil.CaptureStdout(t, func() {
			PrintSystemTestSummary(result, "")
		})

		assert.Empty(t, output)
	})
}

func TestPrintUpdateStructured(t *testing.T) {
	t.Run("outputs structured result", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Target: "18.0.0",
				Status: constants.StatusUpdated,
			},
		}

		var calledFormat output.Format
		var calledResult *output.UpdateResult
		writeFunc := func(w io.Writer, format output.Format, result *output.UpdateResult) error {
			calledFormat = format
			calledResult = result
			return nil
		}

		err := PrintUpdateStructured(results, nil, nil, output.FormatJSON, false, outdated.UpdateSelectionFlags{}, writeFunc)

		assert.NoError(t, err)
		assert.Equal(t, output.FormatJSON, calledFormat)
		assert.NotNil(t, calledResult)
		assert.Len(t, calledResult.Packages, 1)
		assert.Equal(t, 1, calledResult.Summary.UpdatedPackages)
	})

	t.Run("handles dry run mode", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "18.0.0", "18.0.0"),
				Target: "18.0.0",
				Status: constants.StatusUpdated,
			},
		}

		var calledResult *output.UpdateResult
		writeFunc := func(w io.Writer, format output.Format, result *output.UpdateResult) error {
			calledResult = result
			return nil
		}

		err := PrintUpdateStructured(results, nil, nil, output.FormatJSON, true, outdated.UpdateSelectionFlags{}, writeFunc)

		assert.NoError(t, err)
		assert.True(t, calledResult.Summary.DryRun)
		// In dry run mode, status becomes Planned
		assert.Equal(t, constants.StatusPlanned, calledResult.Packages[0].Status)
	})

	t.Run("includes failed packages in count", func(t *testing.T) {
		results := []UpdateResult{
			{
				Pkg:    testutil.NPMPackage("react", "17.0.0", "17.0.0"),
				Target: "18.0.0",
				Status: constants.StatusFailed,
				Err:    assert.AnError,
			},
		}

		var calledResult *output.UpdateResult
		writeFunc := func(w io.Writer, format output.Format, result *output.UpdateResult) error {
			calledResult = result
			return nil
		}

		err := PrintUpdateStructured(results, nil, nil, output.FormatJSON, false, outdated.UpdateSelectionFlags{}, writeFunc)

		assert.NoError(t, err)
		assert.Equal(t, 1, calledResult.Summary.FailedPackages)
	})

	t.Run("includes warnings and errors", func(t *testing.T) {
		results := []UpdateResult{}
		warnings := []string{"warning1"}
		errs := []string{"error1"}

		var calledResult *output.UpdateResult
		writeFunc := func(w io.Writer, format output.Format, result *output.UpdateResult) error {
			calledResult = result
			return nil
		}

		err := PrintUpdateStructured(results, warnings, errs, output.FormatJSON, false, outdated.UpdateSelectionFlags{}, writeFunc)

		assert.NoError(t, err)
		assert.Len(t, calledResult.Warnings, 1)
		assert.Len(t, calledResult.Errors, 1)
	})
}

func TestPrintUpdateErrorsWithHints(t *testing.T) {
	t.Run("prints errors with hints", func(t *testing.T) {
		errs := []error{
			assert.AnError,
		}
		enhanceFunc := func(err error) string {
			return "enhanced: " + err.Error()
		}

		output := testutil.CaptureStdout(t, func() {
			PrintUpdateErrorsWithHints(errs, enhanceFunc)
		})

		// Should contain error information
		assert.Contains(t, output, "error")
	})

	t.Run("handles empty errors", func(t *testing.T) {
		output := testutil.CaptureStdout(t, func() {
			PrintUpdateErrorsWithHints([]error{}, nil)
		})

		// Should be empty for no errors
		assert.Empty(t, output)
	})
}

func TestPrintSystemTestResultDirect(t *testing.T) {
	t.Run("handles nil result", func(t *testing.T) {
		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(nil, "")
		})

		assert.Empty(t, output)
	})

	t.Run("handles empty result", func(t *testing.T) {
		result := &systemtest.Result{}
		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(result, "")
		})

		assert.Empty(t, output)
	})

	t.Run("prints all passed tests", func(t *testing.T) {
		result := &systemtest.Result{
			Tests: []systemtest.TestResult{
				{Name: "test1", Passed: true, Duration: time.Millisecond * 100},
				{Name: "test2", Passed: true, Duration: time.Millisecond * 200},
			},
			TotalDuration: time.Millisecond * 300,
		}

		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(result, "  ")
		})

		assert.Contains(t, output, "System tests: 2/2 passed")
		assert.Contains(t, output, "test1")
		assert.Contains(t, output, "test2")
	})

	t.Run("prints failed tests with output", func(t *testing.T) {
		result := &systemtest.Result{
			Tests: []systemtest.TestResult{
				{Name: "test1", Passed: true, Duration: time.Millisecond * 100},
				{Name: "test2", Passed: false, Duration: time.Millisecond * 200, Output: "Error output\nSecond line"},
			},
			TotalDuration: time.Millisecond * 300,
		}

		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(result, "")
		})

		assert.Contains(t, output, "System tests: 1/2 passed")
		assert.Contains(t, output, "Error output")
		assert.Contains(t, output, "Second line")
	})

	t.Run("uses indent prefix", func(t *testing.T) {
		result := &systemtest.Result{
			Tests: []systemtest.TestResult{
				{Name: "test1", Passed: true, Duration: time.Millisecond * 100},
			},
			TotalDuration: time.Millisecond * 100,
		}

		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(result, ">>")
		})

		assert.Contains(t, output, ">>")
	})

	t.Run("handles failed test without output", func(t *testing.T) {
		result := &systemtest.Result{
			Tests: []systemtest.TestResult{
				{Name: "test1", Passed: false, Duration: time.Millisecond * 100, Error: errors.New("failed")},
			},
			TotalDuration: time.Millisecond * 100,
		}

		output := testutil.CaptureStdout(t, func() {
			printSystemTestResultDirect(result, "")
		})

		assert.Contains(t, output, "System tests: 0/1 passed")
		assert.Contains(t, output, "test1")
	})
}
