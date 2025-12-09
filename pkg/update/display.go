package update

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/display"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/lock"
	"github.com/ajxudir/goupdate/pkg/outdated"
	"github.com/ajxudir/goupdate/pkg/output"
	"github.com/ajxudir/goupdate/pkg/systemtest"
)

// Note: ShouldTrackUnsupported and CollectUpdateErrors are defined in execution.go

// UpdateSummaryCounts holds all counts for the update summary display.
type UpdateSummaryCounts struct {
	ToUpdate int // Packages that will be / were updated
	UpToDate int // Packages already at target version
	Failed   int // Packages that failed to update
	HasMajor int // Packages with major updates still available
	HasMinor int // Packages with minor updates still available
	HasPatch int // Packages with patch updates still available
}

// UpdateSummaryMode indicates whether the summary is for preview or post-update.
type UpdateSummaryMode int

const (
	SummaryModePreview UpdateSummaryMode = iota // Before updates (preview)
	SummaryModeResult                           // After updates (actual results)
	SummaryModeDryRun                           // Dry-run mode (planned, not executed)
)

// FormatConstraintDisplay formats the constraint for display, showing scope override if applicable.
// When a flag overrides the package's constraint, the display shows the effective constraint
// with the flag that caused the override (e.g., "Major (--major)").
func FormatConstraintDisplay(p formats.Package, selection outdated.UpdateSelectionFlags) string {
	return display.FormatConstraintDisplayWithFlags(p, selection.Major, selection.Minor, selection.Patch)
}

// SafeFromVersion returns the original version (for "from" display in update summaries).
// Checks in order: OriginalInstalled (lock file), OriginalVersion (declared), then current Pkg.Version.
func SafeFromVersion(res UpdateResult) string {
	if res.OriginalInstalled != "" && res.OriginalInstalled != constants.PlaceholderNA {
		return res.OriginalInstalled
	}
	if res.OriginalVersion != "" && res.OriginalVersion != constants.PlaceholderNA {
		return res.OriginalVersion
	}
	return display.SafeDeclaredValue(res.Pkg.Version)
}

// DetermineScopeDescription returns a description of the update scope based on selection flags.
func DetermineScopeDescription(selection outdated.UpdateSelectionFlags) string {
	if selection.Major {
		return "--major scope"
	}
	if selection.Minor {
		return "--minor scope"
	}
	if selection.Patch {
		return "--patch scope"
	}
	return "constraint scope"
}

// ComputeSummaryFromPlans computes summary counts from planned updates (for preview).
func ComputeSummaryFromPlans(plans []*PlannedUpdate) UpdateSummaryCounts {
	var counts UpdateSummaryCounts

	for _, plan := range plans {
		res := plan.Res

		if res.Status == lock.InstallStatusNotConfigured || res.Status == constants.StatusConfigError ||
			res.Status == constants.StatusFailed || res.Status == constants.StatusSummarizeError || res.Status == lock.InstallStatusFloating {
			if res.Err != nil || strings.HasPrefix(res.Status, constants.StatusFailed) {
				counts.Failed++
			}
			continue
		}

		target := strings.TrimSpace(res.Target)
		if target != "" && target != constants.PlaceholderNA {
			counts.ToUpdate++
		} else {
			counts.UpToDate++
		}

		if res.Major != "" && res.Major != constants.PlaceholderNA && res.Major != target {
			counts.HasMajor++
		}
		if res.Minor != "" && res.Minor != constants.PlaceholderNA && res.Minor != target {
			counts.HasMinor++
		}
		if res.Patch != "" && res.Patch != constants.PlaceholderNA && res.Patch != target {
			counts.HasPatch++
		}
	}

	return counts
}

// ComputeSummaryFromResults computes summary counts from update results (for post-update).
func ComputeSummaryFromResults(results []UpdateResult) UpdateSummaryCounts {
	var counts UpdateSummaryCounts

	for _, res := range results {
		switch res.Status {
		case constants.StatusUpdated, constants.StatusPlanned:
			counts.ToUpdate++
		case constants.StatusUpToDate:
			counts.UpToDate++
		default:
			if res.Err != nil || strings.HasPrefix(res.Status, constants.StatusFailed) {
				counts.Failed++
			} else if res.Status != lock.InstallStatusNotConfigured && res.Status != lock.InstallStatusFloating {
				counts.UpToDate++
			}
		}

		target := strings.TrimSpace(res.Target)
		if res.Major != "" && res.Major != constants.PlaceholderNA && res.Major != target {
			counts.HasMajor++
		}
		if res.Minor != "" && res.Minor != constants.PlaceholderNA && res.Minor != target {
			counts.HasMinor++
		}
		if res.Patch != "" && res.Patch != constants.PlaceholderNA && res.Patch != target {
			counts.HasPatch++
		}
	}

	return counts
}

// FormatUpdateSummary formats the summary counts into display strings.
func FormatUpdateSummary(counts UpdateSummaryCounts, mode UpdateSummaryMode) (updated, upToDate, moreMajor, moreMinor, morePatch string) {
	switch mode {
	case SummaryModePreview:
		if counts.ToUpdate > 0 {
			updated = fmt.Sprintf("%d to update", counts.ToUpdate)
		}
	case SummaryModeResult:
		if counts.ToUpdate > 0 {
			updated = fmt.Sprintf("%d updated", counts.ToUpdate)
		}
	case SummaryModeDryRun:
		if counts.ToUpdate > 0 {
			updated = fmt.Sprintf("%d planned", counts.ToUpdate)
		}
	}

	if counts.UpToDate > 0 {
		upToDate = fmt.Sprintf("%d up to date", counts.UpToDate)
	}
	if counts.HasMajor > 0 {
		moreMajor = fmt.Sprintf("%d have major updates", counts.HasMajor)
	}
	if counts.HasMinor > 0 {
		moreMinor = fmt.Sprintf("%d have minor updates", counts.HasMinor)
	}
	if counts.HasPatch > 0 {
		morePatch = fmt.Sprintf("%d have patch updates", counts.HasPatch)
	}

	return
}

// PrintUpdateRow prints a single update result row using the shared table formatter.
func PrintUpdateRow(res UpdateResult, table *output.Table, dryRun bool, selection outdated.UpdateSelectionFlags) {
	status := res.Status
	if res.Status == constants.StatusUpdated && dryRun {
		status = constants.StatusPlanned
	}

	statusDisplay := display.FormatStatus(status)
	target := res.Target
	if target == "" {
		target = constants.PlaceholderNA
	}

	constraintDisplay := FormatConstraintDisplay(res.Pkg, selection)

	row := table.FormatRow(
		res.Pkg.Rule,
		res.Pkg.PackageType,
		res.Pkg.Type,
		constraintDisplay,
		display.SafeDeclaredValue(res.Pkg.Version),
		display.SafeInstalledValue(res.Pkg.InstalledVersion),
		target,
		statusDisplay,
		res.Group,
		res.Pkg.Name,
	)
	fmt.Println(row)
	// Force flush to ensure realtime output in CI environments (GitHub Actions, etc.)
	_ = os.Stdout.Sync()
}

// BuildUpdateTableFromPackages creates a table with column widths calculated from package data.
func BuildUpdateTableFromPackages(packages []formats.Package, selection outdated.UpdateSelectionFlags) *output.Table {
	groups := make([]string, len(packages))
	for i, p := range packages {
		groups[i] = p.Group
	}
	showGroup := output.ShouldShowGroupColumn(groups)

	table := output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumnWithMinWidth("TARGET", 12).
		AddColumnWithMinWidth("STATUS", 14).
		AddConditionalColumn("GROUP", showGroup).
		AddColumn("NAME")

	for _, p := range packages {
		constraintDisplay := FormatConstraintDisplay(p, selection)
		table.UpdateWidths(
			p.Rule,
			p.PackageType,
			p.Type,
			constraintDisplay,
			display.SafeDeclaredValue(p.Version),
			display.SafeInstalledValue(p.InstalledVersion),
			"", // TARGET - use minimum width
			"", // STATUS - use minimum width
			p.Group,
			p.Name,
		)
	}

	return table
}

// PrintUpdatePreview prints a detailed preview showing packages that will be updated.
func PrintUpdatePreview(plans []*PlannedUpdate, table *output.Table, selection outdated.UpdateSelectionFlags) {
	var willUpdate, hasMoreUpdates, hasMajorOnly []*PlannedUpdate

	for _, plan := range plans {
		res := plan.Res
		if IsNonUpdatableStatus(res.Status) {
			continue
		}

		currentVersion := strings.TrimSpace(res.Pkg.InstalledVersion)
		if currentVersion == "" || currentVersion == constants.PlaceholderNA {
			currentVersion = strings.TrimSpace(res.Pkg.Version)
		}
		targetVersion := strings.TrimSpace(res.Target)

		hasMajor := res.Major != "" && res.Major != constants.PlaceholderNA

		if targetVersion != "" && targetVersion != currentVersion {
			willUpdate = append(willUpdate, plan)
			// Track if this package also has major updates available
			if hasMajor && res.Major != targetVersion {
				hasMajorOnly = append(hasMajorOnly, plan)
			}
		} else if display.HasAvailableUpdates(res.Major, res.Minor, res.Patch) {
			hasMoreUpdates = append(hasMoreUpdates, plan)
			// Track major-only packages
			if hasMajor && res.Minor == constants.PlaceholderNA && res.Patch == constants.PlaceholderNA {
				hasMajorOnly = append(hasMajorOnly, plan)
			}
		}
	}

	scope := DetermineScopeDescription(selection)

	fmt.Println()
	fmt.Println("Update Plan")
	fmt.Println(strings.Repeat("═", 70))
	fmt.Println()

	if len(willUpdate) > 0 {
		fmt.Printf("Will update (%s):\n", scope)
		for _, plan := range willUpdate {
			res := plan.Res
			availableInfo := display.FormatAvailableVersions(res.Target, res.Major, res.Minor, res.Patch)
			fmt.Printf("  %-20s %s → %s  %s\n",
				res.Pkg.Name,
				SafeFromVersion(res),
				res.Target,
				availableInfo)
		}
		fmt.Println()
	}

	if len(hasMoreUpdates) > 0 {
		fmt.Println("Up to date (other updates available):")
		for _, plan := range hasMoreUpdates {
			res := plan.Res
			availableInfo := display.FormatAvailableVersionsUpToDate(res.Major, res.Minor, res.Patch)
			fmt.Printf("  %-20s %s  %s\n",
				res.Pkg.Name,
				display.SafeDeclaredValue(res.Pkg.Version),
				availableInfo)
		}
		fmt.Println()
	}

	counts := ComputeSummaryFromPlans(plans)
	PrintUpdateSummaryLines(counts, SummaryModePreview)

	// Show major updates warning at the end for visibility
	if counts.HasMajor > 0 {
		fmt.Println()
		fmt.Printf("⚠️  %d package(s) have MAJOR updates available (not auto-applied)\n", counts.HasMajor)
	}
}

// FormatSummaryStrings formats the summary counts into display strings for cmd layer.
func FormatSummaryStrings(counts UpdateSummaryCounts, mode UpdateSummaryMode) (summaryLine, availableLine string) {
	var actionVerb string
	switch mode {
	case SummaryModePreview:
		actionVerb = "to update"
	case SummaryModeResult:
		actionVerb = "updated"
	case SummaryModeDryRun:
		actionVerb = "planned"
	}

	var parts []string
	if counts.ToUpdate > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", counts.ToUpdate, actionVerb))
	}
	if counts.UpToDate > 0 {
		parts = append(parts, fmt.Sprintf("%d up-to-date", counts.UpToDate))
	}
	if counts.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", counts.Failed))
	}

	if len(parts) > 0 {
		summaryLine = fmt.Sprintf("Summary: %s", strings.Join(parts, ", "))
	}

	if counts.HasMajor > 0 || counts.HasMinor > 0 || counts.HasPatch > 0 {
		var remaining []string
		if counts.HasMajor > 0 {
			remaining = append(remaining, fmt.Sprintf("%d have major", counts.HasMajor))
		}
		if counts.HasMinor > 0 {
			remaining = append(remaining, fmt.Sprintf("%d have minor", counts.HasMinor))
		}
		if counts.HasPatch > 0 {
			remaining = append(remaining, fmt.Sprintf("%d have patch", counts.HasPatch))
		}

		suffix := "available"
		if mode != SummaryModePreview {
			suffix = "updates still available"
		}
		availableLine = fmt.Sprintf("         (%s %s)", strings.Join(remaining, ", "), suffix)
	}

	return summaryLine, availableLine
}

// PrintUpdateSummaryLines prints the formatted summary to stdout.
func PrintUpdateSummaryLines(counts UpdateSummaryCounts, mode UpdateSummaryMode) {
	summaryLine, availableLine := FormatSummaryStrings(counts, mode)
	if summaryLine != "" {
		fmt.Println(summaryLine)
	}
	if availableLine != "" {
		fmt.Println(availableLine)
	}
}

// PrintUpdateSummary prints a final summary of update results including remaining available updates.
func PrintUpdateSummary(results []UpdateResult, dryRun bool, afterAllTestResult SystemTestResultFormatter) {
	if len(results) == 0 {
		return
	}

	var updated, failed, hasMoreUpdates []UpdateResult

	for _, res := range results {
		if res.Status == constants.StatusFailed {
			failed = append(failed, res)
			continue
		}
		if res.Status == constants.StatusConfigError || res.Status == constants.StatusSummarizeError {
			continue
		}
		if res.Status == lock.InstallStatusNotConfigured || res.Status == lock.InstallStatusFloating {
			continue
		}

		if res.Status == constants.StatusUpdated || res.Status == constants.StatusPlanned {
			updated = append(updated, res)
		} else if res.Status == constants.StatusUpToDate && display.HasAvailableUpdates(res.Major, res.Minor, res.Patch) {
			hasMoreUpdates = append(hasMoreUpdates, res)
		}
	}

	if len(updated) > 0 || len(failed) > 0 {
		fmt.Println()
		fmt.Println("Update Summary")
		fmt.Println(strings.Repeat("═", 70))

		if len(updated) > 0 {
			fmt.Println()
			actionVerb := "Successfully updated"
			if dryRun {
				actionVerb = "Planned updates"
			}

			fmt.Printf("%s:\n", actionVerb)
			for _, res := range updated {
				availableInfo := display.FormatAvailableVersions(res.Target, res.Major, res.Minor, res.Patch)
				fmt.Printf("  %-20s %s → %s  %s\n",
					res.Pkg.Name,
					SafeFromVersion(res),
					res.Target,
					availableInfo)

				if res.SystemTestResult != nil && len(res.SystemTestResult.Tests) > 0 {
					printSystemTestResultDirect(res.SystemTestResult, "    ")
				}
			}

			if afterAllTestResult != nil && afterAllTestResult.TestCount() > 0 && !dryRun {
				fmt.Println()
				fmt.Println("  System tests (after all updates):")
				PrintSystemTestSummary(afterAllTestResult, "    ")
			}
		}

		if len(failed) > 0 && !dryRun {
			fmt.Println()
			fmt.Println("Failed updates:")
			for _, res := range failed {
				fmt.Printf("  %s %-20s %s → %s\n",
					constants.IconError,
					res.Pkg.Name,
					SafeFromVersion(res),
					res.Target)

				if res.Err != nil {
					fmt.Printf("     └─ %s\n", res.Err.Error())
				}

				if res.SystemTestResult != nil && len(res.SystemTestResult.Tests) > 0 {
					printSystemTestResultDirect(res.SystemTestResult, "     ")
				}
			}
		}

		fmt.Println()

		if len(hasMoreUpdates) > 0 {
			fmt.Println("Up to date (other updates available):")
			for _, res := range hasMoreUpdates {
				availableInfo := display.FormatAvailableVersionsUpToDate(res.Major, res.Minor, res.Patch)
				fmt.Printf("  %-20s %s  %s\n",
					res.Pkg.Name,
					display.SafeDeclaredValue(res.Pkg.Version),
					availableInfo)
			}
			fmt.Println()
		}
	}

	counts := ComputeSummaryFromResults(results)
	mode := SummaryModeResult
	if dryRun {
		mode = SummaryModeDryRun
	}
	PrintUpdateSummaryLines(counts, mode)
}

// SystemTestResultFormatter is an interface for system test result formatting.
type SystemTestResultFormatter interface {
	TestCount() int
	PassedCount() int
	Passed() bool
	TotalDuration() time.Duration
	Tests() []SystemTestInfo
}

// SystemTestInfo holds information about a single system test.
type SystemTestInfo interface {
	GetName() string
	GetPassed() bool
	GetDuration() time.Duration
	GetOutput() string
}

// PrintSystemTestSummary prints a compact summary of system test results.
func PrintSystemTestSummary(result SystemTestResultFormatter, indent string) {
	if result == nil || result.TestCount() == 0 {
		return
	}

	passed := result.PassedCount()
	total := result.TestCount()
	allPassed := result.Passed()

	icon := constants.IconCheckmark
	if !allPassed {
		icon = constants.IconCross
	}

	fmt.Printf("%s%s System tests: %d/%d passed [%s]\n",
		indent, icon, passed, total, FormatTestDuration(result.TotalDuration()))

	for _, t := range result.Tests() {
		testIcon := constants.IconCheckmark
		if !t.GetPassed() {
			testIcon = constants.IconCross
		}
		fmt.Printf("%s  %s %s [%s]\n", indent, testIcon, t.GetName(), FormatTestDuration(t.GetDuration()))

		if !t.GetPassed() && t.GetOutput() != "" {
			for _, line := range strings.Split(strings.TrimSpace(t.GetOutput()), "\n") {
				fmt.Printf("%s    %s\n", indent, line)
			}
		}
	}
}

// FormatTestDuration formats a duration for display in test summaries.
func FormatTestDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// PrintUpdateErrorsWithHints prints errors with actionable resolution hints.
func PrintUpdateErrorsWithHints(errs []error, enhanceFunc func(error) string) {
	if len(errs) == 0 {
		return
	}

	fmt.Println()
	for _, err := range errs {
		fmt.Printf("%s %s\n", constants.IconError, enhanceFunc(err))
	}
}

// PrintUpdateStructured outputs update results in a structured format (CSV, JSON, XML).
func PrintUpdateStructured(results []UpdateResult, warnings []string, errs []string, format output.Format, dryRun bool, selection outdated.UpdateSelectionFlags, writeFunc func(w io.Writer, format output.Format, result *output.UpdateResult) error) error {
	packages := make([]output.UpdatePackage, 0, len(results))

	var updatedCount, failedCount int

	for _, res := range results {
		status := res.Status
		if res.Status == constants.StatusUpdated && dryRun {
			status = constants.StatusPlanned
		}

		constraintDisplay := FormatConstraintDisplay(res.Pkg, selection)

		var errStr string
		if res.Err != nil {
			errStr = res.Err.Error()
		}

		target := res.Target
		if target == "" {
			target = constants.PlaceholderNA
		}

		packages = append(packages, output.UpdatePackage{
			Rule:             res.Pkg.Rule,
			PM:               res.Pkg.PackageType,
			Type:             res.Pkg.Type,
			Constraint:       constraintDisplay,
			Version:          display.SafeDeclaredValue(res.Pkg.Version),
			InstalledVersion: display.SafeInstalledValue(res.Pkg.InstalledVersion),
			Target:           target,
			Status:           status,
			Group:            res.Group,
			Name:             res.Pkg.Name,
			Error:            errStr,
		})

		switch status {
		case constants.StatusUpdated, constants.StatusPlanned:
			updatedCount++
		default:
			if res.Err != nil || strings.HasPrefix(status, constants.StatusFailed) {
				failedCount++
			}
		}
	}

	result := &output.UpdateResult{
		Summary: output.UpdateSummary{
			TotalPackages:   len(packages),
			UpdatedPackages: updatedCount,
			FailedPackages:  failedCount,
			DryRun:          dryRun,
		},
		Packages: packages,
		Warnings: warnings,
		Errors:   errs,
	}

	return writeFunc(os.Stdout, format, result)
}

// printSystemTestResultDirect prints system test results using the actual systemtest.Result type.
// This is used for inline results within UpdateResult that use the direct type.
func printSystemTestResultDirect(result *systemtest.Result, indent string) {
	if result == nil || len(result.Tests) == 0 {
		return
	}

	passed := result.PassedCount()
	total := len(result.Tests)
	allPassed := result.Passed()

	icon := constants.IconCheckmark
	if !allPassed {
		icon = constants.IconCross
	}

	fmt.Printf("%s%s System tests: %d/%d passed [%s]\n",
		indent, icon, passed, total, FormatTestDuration(result.TotalDuration))

	for _, t := range result.Tests {
		testIcon := constants.IconCheckmark
		if !t.Passed {
			testIcon = constants.IconCross
		}
		fmt.Printf("%s  %s %s [%s]\n", indent, testIcon, t.Name, FormatTestDuration(t.Duration))

		if !t.Passed && t.Output != "" {
			for _, line := range strings.Split(strings.TrimSpace(t.Output), "\n") {
				fmt.Printf("%s    %s\n", indent, line)
			}
		}
	}
}

