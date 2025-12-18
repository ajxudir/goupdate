package lock

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/packages"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/warnings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyInstalledVersionsFromConfig tests the behavior of ApplyInstalledVersions with configuration.
//
// It verifies:
//   - Regular semver constraints are resolved with LockFound status
//   - Floating constraints like "1.x" are marked with Floating status
//   - Installed versions are correctly extracted from lock files
func TestApplyInstalledVersionsFromConfig(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/npm")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["npm"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "package.json"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "npm"
	}

	enriched, err := ApplyInstalledVersions(result.Packages, cfg, testdataDir)
	require.NoError(t, err)
	t.Logf("enriched: %#v", enriched)

	lookup := map[string]formats.Package{}
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg
	}

	// Regular semver constraints should be LockFound
	assert.Equal(t, "4.17.21", lookup["lodash"].InstalledVersion)
	assert.Equal(t, InstallStatusLockFound, lookup["lodash"].InstallStatus)
	assert.Equal(t, "4.18.3", lookup["express"].InstalledVersion)
	assert.Equal(t, InstallStatusLockFound, lookup["express"].InstallStatus)
	assert.Equal(t, "5.3.0", lookup["chalk"].InstalledVersion)
	assert.Equal(t, InstallStatusLockFound, lookup["chalk"].InstallStatus)

	// Floating constraints (1.x, *) should be marked as Floating
	assert.Equal(t, "1.11.19", lookup["dayjs"].InstalledVersion)
	assert.Equal(t, InstallStatusFloating, lookup["dayjs"].InstallStatus)
}

// TestApplyInstalledVersionsSupportsPackagesConfigLock tests the behavior of NuGet packages.config resolution.
//
// It verifies:
//   - NuGet packages in lock file have LockFound status
//   - Packages not in lock file have NotInLock status
//   - InstalledVersion is correctly set from lock file
func TestApplyInstalledVersionsSupportsPackagesConfigLock(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/nuget")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["nuget"]
	require.NotEmpty(t, rule.LockFiles)

	installed, foundLock, err := resolveInstalledVersions(testdataDir, rule.LockFiles)
	require.NoError(t, err)
	require.True(t, foundLock)
	require.Equal(t, "13.0.3", installed["Newtonsoft.Json"])
	result, err := parser.ParseFile(filepath.Join(testdataDir, "packages.config"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "nuget"
	}

	enriched, err := ApplyInstalledVersions(result.Packages, cfg, testdataDir)
	require.NoError(t, err)

	lookup := map[string]formats.Package{}
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg
	}

	// Newtonsoft.Json is in both packages.config and lock file
	assert.Equal(t, "13.0.3", lookup["Newtonsoft.Json"].InstalledVersion)
	assert.Equal(t, InstallStatusLockFound, lookup["Newtonsoft.Json"].InstallStatus)

	// Moq is in packages.config but not in lock file
	require.Contains(t, lookup, "Moq")
	assert.Equal(t, InstallStatusNotInLock, lookup["Moq"].InstallStatus)
	assert.Equal(t, "#N/A", lookup["Moq"].InstalledVersion)
}

// TestApplyInstalledVersionsHandlesMissingAndUnsupported tests the behavior with missing and unsupported lock files.
//
// It verifies:
//   - Packages with missing lock files have LockMissing status
//   - Packages without lock configuration have NotConfigured status
//   - InstalledVersion is set to #N/A in both cases
func TestApplyInstalledVersionsHandlesMissingAndUnsupported(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		WorkingDir: tmpDir,
		Rules: map[string]config.PackageManagerCfg{
			"missing-lock": {
				LockFiles: []config.LockFileCfg{
					{
						Files:      []string{"missing.lock"},
						Extraction: &config.ExtractionCfg{Pattern: `(?P<n>\w+)\s+(?P<version>\S+)`},
					},
				},
			},
			"unsupported": {},
		},
	}

	pkgs := []formats.Package{
		{Name: "aws", Rule: "missing-lock"},
		{Name: "nginx", Rule: "unsupported"},
	}

	enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, InstallStatusLockMissing, enriched[0].InstallStatus)
	assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
	assert.Equal(t, InstallStatusNotConfigured, enriched[1].InstallStatus)
	assert.Equal(t, "#N/A", enriched[1].InstalledVersion)
}

// TestApplyInstalledVersionsTreatsWildcardAsFloating tests the behavior of wildcard version constraints.
//
// It verifies:
//   - Packages with "*" version are marked with Floating status
//   - InstalledVersion is set to #N/A for wildcard constraints
func TestApplyInstalledVersionsTreatsWildcardAsFloating(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "deps.lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("other 1.0.0\n"), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"with-lock": {
			LockFiles: []config.LockFileCfg{{
				Files:      []string{"**/deps.lock"},
				Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)\s+(?P<version>\S+)`},
			}},
		},
	}}

	pkgs := []formats.Package{{Name: "missing", Version: "*", Rule: "with-lock"}}

	enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
	require.NoError(t, err)

	// Wildcard "*" is a floating constraint - cannot be updated automatically
	assert.Equal(t, InstallStatusFloating, enriched[0].InstallStatus)
	assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
}

// TestApplyInstalledVersionsHandlesEmptyInputs tests the behavior with nil and empty inputs.
//
// It verifies:
//   - nil packages and config return empty result without error
//   - Packages with missing rule configuration are returned unchanged
func TestApplyInstalledVersionsHandlesEmptyInputs(t *testing.T) {
	pkgs, err := ApplyInstalledVersions(nil, nil, "")
	require.NoError(t, err)
	assert.Empty(t, pkgs)

	pkgList := []formats.Package{{Name: "pkg", Rule: "missing"}}
	pkgs, err = ApplyInstalledVersions(pkgList, &config.Config{Rules: map[string]config.PackageManagerCfg{}}, "")
	require.NoError(t, err)
	assert.Equal(t, pkgList, pkgs)
}

// TestApplyInstalledVersionsUsesPackageScope tests the behavior of package-scoped lock file resolution.
//
// It verifies:
//   - Lock files are resolved relative to the package's source directory
//   - Packages in different directories use their respective lock files
//   - Packages without lock files in their scope have LockMissing status
func TestApplyInstalledVersionsUsesPackageScope(t *testing.T) {
	tmpDir := t.TempDir()
	lockedDir := filepath.Join(tmpDir, "with-lock")
	nolockDir := filepath.Join(tmpDir, "without-lock")
	require.NoError(t, os.MkdirAll(lockedDir, 0o755))
	require.NoError(t, os.MkdirAll(nolockDir, 0o755))

	lockPath := filepath.Join(lockedDir, "package-lock.json")
	lockContent := `{"chalk":{"version":"4.1.2"}}`
	require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {
			LockFiles: []config.LockFileCfg{{
				Files:      []string{"**/package-lock.json"},
				Extraction: &config.ExtractionCfg{Pattern: `(?s)"(?P<n>[^"}]+)"\s*:\s*\{[^}]*"version"\s*:\s*"(?P<version>[^"]+)"`},
			}},
		},
	}}

	pkgs := []formats.Package{
		{Name: "chalk", Rule: "npm", Source: filepath.Join(lockedDir, "package.json")},
		{Name: "left-pad", Rule: "npm", Source: filepath.Join(nolockDir, "package.json")},
	}

	enriched, err := ApplyInstalledVersions(pkgs, cfg, "")
	require.NoError(t, err)

	assert.Equal(t, InstallStatusLockFound, enriched[0].InstallStatus)
	assert.Equal(t, "4.1.2", enriched[0].InstalledVersion)
	assert.Equal(t, InstallStatusLockMissing, enriched[1].InstallStatus)
	assert.Equal(t, "#N/A", enriched[1].InstalledVersion)
}

// TestApplyInstalledVersionsUsesWorkingDirFallback tests the behavior of working directory fallback.
//
// It verifies:
//   - Config's WorkingDir is used when package Source is empty
//   - Lock files are found in the working directory
//   - Installed versions are correctly resolved
func TestApplyInstalledVersionsUsesWorkingDirFallback(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "package-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"pkg":{"version":"1.0.0"}}`), 0o644))

	cfg := &config.Config{WorkingDir: tmpDir, Rules: map[string]config.PackageManagerCfg{
		"npm": {
			LockFiles: []config.LockFileCfg{{
				Files:      []string{"**/package-lock.json"},
				Extraction: &config.ExtractionCfg{Pattern: `(?s)"(?P<n>[^"}]+)"\s*:\s*\{[^}]*"version"\s*:\s*"(?P<version>[^"]+)"`},
			}},
		},
	}}

	pkgs := []formats.Package{{Name: "pkg", Rule: "npm"}}
	enriched, err := ApplyInstalledVersions(pkgs, cfg, "")
	require.NoError(t, err)

	assert.Equal(t, InstallStatusLockFound, enriched[0].InstallStatus)
	assert.Equal(t, "1.0.0", enriched[0].InstalledVersion)
}

// TestApplyInstalledVersionsUsesDefaultDirectory tests the behavior of default directory resolution.
//
// It verifies:
//   - Current working directory "." is used when all other paths are empty
//   - Lock files are found in the current directory
//   - Installed versions are correctly resolved
func TestApplyInstalledVersionsUsesDefaultDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	currentDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(currentDir) })
	require.NoError(t, os.Chdir(tmpDir))

	lockPath := filepath.Join(tmpDir, "package-lock.json")
	require.NoError(t, os.WriteFile(lockPath, []byte(`{"pkg":{"version":"2.0.0"}}`), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {
			LockFiles: []config.LockFileCfg{{
				Files:      []string{"**/package-lock.json"},
				Extraction: &config.ExtractionCfg{Pattern: `(?s)"(?P<n>[^"}]+)"\s*:\s*\{[^}]*"version"\s*:\s*"(?P<version>[^"]+)"`},
			}},
		},
	}}

	pkgs := []formats.Package{{Name: "pkg", Rule: "npm"}}
	enriched, err := ApplyInstalledVersions(pkgs, cfg, "")
	require.NoError(t, err)

	assert.Equal(t, InstallStatusLockFound, enriched[0].InstallStatus)
	assert.Equal(t, "2.0.0", enriched[0].InstalledVersion)
}

// TestApplyInstalledVersionsMarksWildcardAsFloating tests the behavior of wildcard marking as floating constraint.
//
// It verifies:
//   - Wildcard "*" version is marked with Floating status
//   - No warnings are issued for floating constraints
func TestApplyInstalledVersionsMarksWildcardAsFloating(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"latest": {
			Manager: "js",
		},
	}}

	pkgs := []formats.Package{{Name: "pkg", Version: "*", Rule: "latest"}}

	var buf bytes.Buffer
	restore := warnings.SetWarningWriter(&buf)
	t.Cleanup(restore)

	enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
	require.NoError(t, err)

	// Wildcard "*" is a floating constraint - cannot be updated automatically
	assert.Equal(t, InstallStatusFloating, enriched[0].InstallStatus)
	assert.Empty(t, buf.String())
}

// TestIssueLatestWarningSkipsWhenNotLatestOrFound tests the behavior of warning suppression.
//
// It verifies:
//   - No warnings are issued for non-latest version indicators
//   - No warnings are issued when package is found in lock file
func TestIssueLatestWarningSkipsWhenNotLatestOrFound(t *testing.T) {
	var buf bytes.Buffer
	restore := warnings.SetWarningWriter(&buf)
	t.Cleanup(restore)

	dedup := make(map[string]struct{})
	rule := config.PackageManagerCfg{Manager: "js"}

	issueLatestWarning(formats.Package{Name: "pkg", Version: "1.0.0", Rule: "rule", InstallStatus: InstallStatusLockFound}, rule, dedup)
	issueLatestWarning(formats.Package{Name: "pkg", Version: "1.0.0", Rule: "rule", InstallStatus: InstallStatusNotInLock}, rule, dedup)

	assert.Empty(t, buf.String())
}

// TestIssueLatestWarningDedupesPerRuleAndName tests the behavior of warning deduplication.
//
// It verifies:
//   - Warnings are deduplicated per rule and package name combination
//   - Second warning for same rule:name pair is suppressed
func TestIssueLatestWarningDedupesPerRuleAndName(t *testing.T) {
	var buf bytes.Buffer
	restore := warnings.SetWarningWriter(&buf)
	t.Cleanup(restore)

	dedup := make(map[string]struct{})
	rule := config.PackageManagerCfg{Manager: "js"}

	pkg := formats.Package{Name: "pkg", Version: "*", Rule: "rule", InstallStatus: InstallStatusNotInLock}

	issueLatestWarning(pkg, rule, dedup)
	issueLatestWarning(pkg, rule, dedup)

	assert.Empty(t, buf.String())
}

// TestApplyInstalledVersionsBubblesUpResolveErrors tests the behavior of error propagation.
//
// It verifies:
//   - Errors from lock file resolution are propagated to caller
//   - Invalid regex patterns in extraction config cause errors
func TestApplyInstalledVersionsBubblesUpResolveErrors(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"broken": {
			LockFiles: []config.LockFileCfg{{
				Files:      []string{"**/lock"},
				Extraction: &config.ExtractionCfg{Pattern: "("},
			}},
		},
	}}

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lock"), []byte("pkg v1.0.0"), 0o644))

	_, err := ApplyInstalledVersions([]formats.Package{{Name: "pkg", Rule: "broken"}}, cfg, dir)
	assert.Error(t, err)
}

// TestResolveInstalledVersionsFromCustomLockFile tests the behavior of custom lock file parsing.
//
// It verifies:
//   - Custom lock files can be parsed with custom extraction patterns
//   - Package names and versions are correctly extracted
//   - Lock file is marked as found
func TestResolveInstalledVersionsFromCustomLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "custom.lock")
	err := os.WriteFile(lockPath, []byte("tool 1.2.3\nlib 9.9.9"), 0644)
	require.NoError(t, err)

	cfg := config.LockFileCfg{
		Files:      []string{"**/custom.lock"},
		Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)\s+(?P<version>[\d\.]+)`},
	}

	resolved, found, err := resolveInstalledVersions(tmpDir, []config.LockFileCfg{cfg})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1.2.3", resolved["tool"])
	assert.Equal(t, "9.9.9", resolved["lib"])
}

// TestResolveInstalledVersionsHandlesEmptyAndMissingFiles tests the behavior with empty and missing files.
//
// It verifies:
//   - Empty lock config returns no results and foundAny=false
//   - Missing lock files return no results and foundAny=false
//   - No errors are returned for missing files
func TestResolveInstalledVersionsHandlesEmptyAndMissingFiles(t *testing.T) {
	resolved, found, err := resolveInstalledVersions("", []config.LockFileCfg{{}})
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, resolved)

	cfg := config.LockFileCfg{Files: []string{"**/does-not-exist.lock"}, Extraction: &config.ExtractionCfg{Pattern: `(?P<n>\w+) (?P<version>\d+)`}}
	resolved, found, err = resolveInstalledVersions(t.TempDir(), []config.LockFileCfg{cfg})
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, resolved)
}

// TestResolveInstalledVersionsReturnsExtractionErrors tests the behavior of extraction errors.
//
// It verifies:
//   - Invalid regex patterns in extraction config cause errors
//   - Errors are properly propagated from extraction function
func TestResolveInstalledVersionsReturnsExtractionErrors(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("pkg v1.0.0"), 0o644))

	cfg := config.LockFileCfg{Files: []string{"lock"}, Extraction: &config.ExtractionCfg{Pattern: "("}}
	_, _, err := resolveInstalledVersions(tmpDir, []config.LockFileCfg{cfg})
	assert.Error(t, err)
}

// TestResolveInstalledVersionsFinderError tests the behavior of file finder errors.
//
// It verifies:
//   - Errors from file finder are propagated to caller
//   - Mocked finder failures are properly handled
func TestResolveInstalledVersionsFinderError(t *testing.T) {
	original := findFilesByPatterns
	findFilesByPatterns = func(baseDir string, patterns []string) ([]string, error) {
		return nil, fmt.Errorf("finder failure")
	}
	defer func() { findFilesByPatterns = original }()

	_, _, err := resolveInstalledVersions("", []config.LockFileCfg{{Files: []string{"lock"}}})
	assert.Error(t, err)
}

// TestResolveInstalledVersionsSkipsEmptyVersionFromExtractor tests the behavior of empty version filtering.
//
// It verifies:
//   - Entries with empty versions are filtered out
//   - Lock file is still marked as found even if no valid versions extracted
func TestResolveInstalledVersionsSkipsEmptyVersionFromExtractor(t *testing.T) {
	baseDir := t.TempDir()
	lockPath := filepath.Join(baseDir, "lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("ignored"), 0o644))

	original := extractVersionsFromFn
	extractVersionsFromFn = func(path string, cfg *config.LockFileCfg) (map[string]string, error) {
		return map[string]string{"pkg": ""}, nil
	}
	defer func() { extractVersionsFromFn = original }()

	resolved, found, err := resolveInstalledVersions(baseDir, []config.LockFileCfg{{Files: []string{"lock"}}})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Empty(t, resolved)
}

// TestResolveInstalledVersionsSkipsEmptyVersions tests the behavior of empty version entries.
//
// It verifies:
//   - Package entries with empty versions are skipped
//   - Lock file is marked as found even with no valid versions
func TestResolveInstalledVersionsSkipsEmptyVersions(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "custom.lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("pkg \n"), 0o644))

	cfg := config.LockFileCfg{
		Files:      []string{"custom.lock"},
		Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)\s*(?P<version>.*)$`},
	}

	resolved, found, err := resolveInstalledVersions(tmpDir, []config.LockFileCfg{cfg})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Empty(t, resolved)
}

// TestExtractVersionsTrimsGoModSuffix tests the behavior of Go module suffix trimming.
//
// It verifies:
//   - "/go.mod" suffix is removed from version strings
//   - Go package versions from go.sum are correctly parsed
func TestExtractVersionsTrimsGoModSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "go.sum")
	err := os.WriteFile(lockPath, []byte("github.com/spf13/cobra v1.8.0/go.mod h1:abc\n"), 0644)
	require.NoError(t, err)

	cfg := config.LockFileCfg{
		Files:      []string{"**/go.sum"},
		Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\S+)\s+(?P<version>v[^\s]+)`},
	}

	resolved, found, err := resolveInstalledVersions(tmpDir, []config.LockFileCfg{cfg})
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "v1.8.0", resolved["github.com/spf13/cobra"])
}

// TestExtractVersionsValidationErrors tests the behavior of extraction validation errors.
//
// It verifies:
//   - nil config causes an error
//   - Invalid regex patterns cause errors
//   - Missing files cause errors
func TestExtractVersionsValidationErrors(t *testing.T) {
	_, err := extractVersionsFromLock("missing", nil)
	assert.Error(t, err)

	cfg := &config.LockFileCfg{Extraction: &config.ExtractionCfg{Pattern: "("}}
	tmpFile := filepath.Join(t.TempDir(), "broken.lock")
	require.NoError(t, os.WriteFile(tmpFile, []byte("pkg v1.0.0"), 0o644))

	_, err = extractVersionsFromLock(tmpFile, cfg)
	assert.Error(t, err)

	cfg = &config.LockFileCfg{Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+) (?P<version>.+)`}}
	_, err = extractVersionsFromLock(filepath.Join(t.TempDir(), "does-not-exist.lock"), cfg)
	assert.Error(t, err)
}

// TestExtractVersionsSkipsInvalidMatches tests the behavior of invalid match filtering.
//
// It verifies:
//   - Matches with empty names or versions are skipped
//   - Results map is empty when all matches are invalid
func TestExtractVersionsSkipsInvalidMatches(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "custom.lock")
	require.NoError(t, os.WriteFile(tmpFile, []byte("pkg  \n"), 0o644))

	cfg := &config.LockFileCfg{Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)\s*(?P<version>.*)$`}}

	results, err := extractVersionsFromLock(tmpFile, cfg)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestResolveInstalledVersionsMergesMultipleLockPatterns tests the behavior of multiple lock file merging.
//
// It verifies:
//   - Multiple lock files are successfully parsed and merged
//   - Packages from different lock files are all present in result
//   - Different lock file formats can coexist
func TestResolveInstalledVersionsMergesMultipleLockPatterns(t *testing.T) {
	scenarioDir, err := filepath.Abs("../testdata_errors/_lock-scenarios")
	require.NoError(t, err, "failed to get absolute path to testdata")
	cfg, err := config.LoadConfig(filepath.Join(scenarioDir, "multi-lock-config.yml"), scenarioDir)
	require.NoError(t, err)

	resolved, found, err := resolveInstalledVersions(scenarioDir, cfg.Rules["multi-lock"].LockFiles)
	require.NoError(t, err)

	assert.True(t, found)
	assert.Equal(t, "1.0.0", resolved["alpha"])
	assert.Equal(t, "2.0.0", resolved["beta"])
	assert.Equal(t, "v0.9.1", resolved["github.com/pkg/errors"])
	assert.Equal(t, "5.4.3", resolved["omega"])
}

// TestResolveInstalledVersionsErrorsOnUnparsableLockFile tests the behavior with unparsable lock files.
//
// It verifies:
//   - Errors from unparsable lock files are returned
//   - Error message contains "failed to parse"
func TestResolveInstalledVersionsErrorsOnUnparsableLockFile(t *testing.T) {
	scenarioDir, err := filepath.Abs("../testdata_errors/_lock-scenarios")
	require.NoError(t, err, "failed to get absolute path to testdata")
	cfg, err := config.LoadConfig(filepath.Join(scenarioDir, "unparsable-lock-config.yml"), scenarioDir)
	require.NoError(t, err)

	_, _, err = resolveInstalledVersions(scenarioDir, cfg.Rules["broken-lock"].LockFiles)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

// TestApplyInstalledVersionsHandlesRulesWithoutLockFiles tests the behavior of rules without lock files.
//
// It verifies:
//   - Packages with rules that have no lock files get NotConfigured status
//   - InstalledVersion is set to #N/A
func TestApplyInstalledVersionsHandlesRulesWithoutLockFiles(t *testing.T) {
	scenarioDir, err := filepath.Abs("../testdata_errors/_lock-scenarios")
	require.NoError(t, err, "failed to get absolute path to testdata")
	cfg, err := config.LoadConfig(filepath.Join(scenarioDir, "no-lock-config.yml"), scenarioDir)
	require.NoError(t, err)

	pkgs := []formats.Package{{Name: "alpha", Rule: "nolock-rule"}}
	enriched, err := ApplyInstalledVersions(pkgs, cfg, scenarioDir)
	require.NoError(t, err)

	assert.Equal(t, InstallStatusNotConfigured, enriched[0].InstallStatus)
	assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
}

// TestExtractVersionsFromLockMissingExtraction tests the behavior with missing extraction config.
//
// It verifies:
//   - Missing extraction pattern causes an error
//   - Error message indicates missing extraction pattern
func TestExtractVersionsFromLockMissingExtraction(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "custom.lock")
	require.NoError(t, os.WriteFile(lockPath, []byte("pkg 1.0.0"), 0o644))

	_, err := extractVersionsFromLock(lockPath, &config.LockFileCfg{})
	assert.Error(t, err)
}

// TestNormalizeLockPackageName tests the behavior of package name normalization.
//
// It verifies:
//   - "node_modules/" prefix is removed from package names
//   - "/go.mod" suffix is removed from package names
//   - Empty names are returned as empty strings
func TestNormalizeLockPackageName(t *testing.T) {
	assert.Equal(t, "lodash", normalizeLockPackageName("node_modules/lodash", ""))
	assert.Equal(t, "github.com/pkg/errors", normalizeLockPackageName("github.com/pkg/errors/go.mod", ""))
	assert.Equal(t, "", normalizeLockPackageName("", ""))
}

// TestApplyInstalledVersionsSelfPinning tests the behavior of self-pinning rules.
//
// It verifies:
//   - Packages with versions use self-pinning when enabled
//   - Wildcard versions cannot be self-pinned
//   - Empty versions cannot be self-pinned
//   - Whitespace-only versions cannot be self-pinned
func TestApplyInstalledVersionsSelfPinning(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"selfpin": {
			SelfPinning: true,
		},
	}}

	t.Run("package with version uses self-pinning", func(t *testing.T) {
		pkgs := []formats.Package{{Name: "pkg", Version: "1.2.3", Rule: "selfpin"}}
		enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, InstallStatusSelfPinned, enriched[0].InstallStatus)
		assert.Equal(t, "1.2.3", enriched[0].InstalledVersion)
	})

	t.Run("package with wildcard version cannot be self-pinned", func(t *testing.T) {
		pkgs := []formats.Package{{Name: "pkg", Version: "*", Rule: "selfpin"}}
		enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
		require.NoError(t, err)
		// Wildcard is floating constraint, so final status is Floating
		assert.Equal(t, InstallStatusFloating, enriched[0].InstallStatus)
		assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
	})

	t.Run("package with empty version cannot be self-pinned", func(t *testing.T) {
		pkgs := []formats.Package{{Name: "pkg", Version: "", Rule: "selfpin"}}
		enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, InstallStatusVersionMissing, enriched[0].InstallStatus)
		assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
	})

	t.Run("package with whitespace-only version cannot be self-pinned", func(t *testing.T) {
		pkgs := []formats.Package{{Name: "pkg", Version: "  ", Rule: "selfpin"}}
		enriched, err := ApplyInstalledVersions(pkgs, cfg, tmpDir)
		require.NoError(t, err)
		assert.Equal(t, InstallStatusVersionMissing, enriched[0].InstallStatus)
		assert.Equal(t, "#N/A", enriched[0].InstalledVersion)
	})
}

// TestNormalizeLockPackageNameUsesAlt tests the behavior of alternative name usage.
//
// It verifies:
//   - Alternative name is used when primary name is empty
//   - Primary name takes precedence when both are provided
func TestNormalizeLockPackageNameUsesAlt(t *testing.T) {
	assert.Equal(t, "alt-name", normalizeLockPackageName("", "alt-name"))
	assert.Equal(t, "primary", normalizeLockPackageName("primary", "alt-name"))
}

// TestParseLockCommandOutput tests the behavior of lock command output parsing.
//
// It verifies:
//   - Empty output returns empty map without error
//   - Default format is JSON
//   - Raw format requires extraction pattern
//   - Unsupported formats cause errors
func TestParseLockCommandOutput(t *testing.T) {
	t.Run("empty output returns empty map", func(t *testing.T) {
		result, err := parseLockCommandOutput([]byte(""), nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("whitespace only output returns empty map", func(t *testing.T) {
		result, err := parseLockCommandOutput([]byte("   \n  "), nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("defaults to json format", func(t *testing.T) {
		result, err := parseLockCommandOutput([]byte(`{"pkg": "1.0.0"}`), nil)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result["pkg"])
	})

	t.Run("uses configured json format", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{Format: "json"}
		result, err := parseLockCommandOutput([]byte(`{"pkg": "1.0.0"}`), extraction)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result["pkg"])
	})

	t.Run("raw format requires pattern", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{Format: "raw"}
		_, err := parseLockCommandOutput([]byte("pkg 1.0.0"), extraction)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "raw format requires command_extraction.pattern")
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{Format: "xml"}
		_, err := parseLockCommandOutput([]byte("<pkg>1.0.0</pkg>"), extraction)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported lock command output format")
	})
}

// TestParseLockCommandJSON tests the behavior of JSON command output parsing.
//
// It verifies:
//   - Object format {"pkg": "version"} is parsed correctly
//   - Array format [{"name": "pkg", "version": "ver"}] is parsed correctly
//   - Nested npm ls format is supported
//   - package-lock.json v3 packages format is supported
//   - Custom JSON keys can be configured
//   - Invalid entries are skipped
func TestParseLockCommandJSON(t *testing.T) {
	t.Run("parses object format", func(t *testing.T) {
		input := []byte(`{"lodash": "4.17.21", "express": "4.18.2"}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "4.18.2", result["express"])
	})

	t.Run("skips empty string values in object format", func(t *testing.T) {
		input := []byte(`{"pkg": "", "other": "1.0.0"}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "pkg")
		assert.Equal(t, "1.0.0", result["other"])
	})

	t.Run("skips non-string values in object format", func(t *testing.T) {
		input := []byte(`{"pkg": 123, "other": "1.0.0"}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.NotContains(t, result, "pkg")
		assert.Equal(t, "1.0.0", result["other"])
	})

	t.Run("parses array format with default keys", func(t *testing.T) {
		input := []byte(`[{"name": "lodash", "version": "4.17.21"}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("parses array format with custom keys", func(t *testing.T) {
		input := []byte(`[{"pkg": "lodash", "ver": "4.17.21"}]`)
		extraction := &config.LockCommandExtractionCfg{
			JSONNameKey:    "pkg",
			JSONVersionKey: "ver",
		}
		result, err := parseLockCommandJSON(input, extraction)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("skips array items with missing name", func(t *testing.T) {
		input := []byte(`[{"version": "4.17.21"}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("skips array items with empty name", func(t *testing.T) {
		input := []byte(`[{"name": "", "version": "4.17.21"}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("parses npm ls nested format", func(t *testing.T) {
		input := []byte(`{"dependencies": {"lodash": {"version": "4.17.21"}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("parses package-lock.json v3 packages format", func(t *testing.T) {
		input := []byte(`{"packages": {"node_modules/lodash": {"version": "4.17.21"}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		input := []byte(`{invalid json`)
		_, err := parseLockCommandJSON(input, nil)
		assert.Error(t, err)
	})

	t.Run("parses packages without node_modules prefix", func(t *testing.T) {
		input := []byte(`{"packages": {"my-local-pkg": {"version": "1.0.0"}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", result["my-local-pkg"])
	})

	t.Run("handles root package entry in packages format", func(t *testing.T) {
		// This tests the root package entry in package-lock.json which has "" as key
		// The implementation includes it (maps to empty string key)
		input := []byte(`{"packages": {"": {"version": "1.0.0"}, "node_modules/lodash": {"version": "4.17.21"}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("handles packages with both node_modules and non-prefixed entries", func(t *testing.T) {
		input := []byte(`{"packages": {"node_modules/lodash": {"version": "4.17.21"}, "local-pkg": {"version": "2.0.0"}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "2.0.0", result["local-pkg"])
	})

	t.Run("skips array items with empty version", func(t *testing.T) {
		input := []byte(`[{"name": "pkg", "version": ""}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handles npm ls output with nested dependencies", func(t *testing.T) {
		// npm ls --json output format
		input := []byte(`{"dependencies": {"lodash": {"version": "4.17.21", "dependencies": {"nested-pkg": {"version": "1.0.0"}}}}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "1.0.0", result["nested-pkg"])
	})

	t.Run("parses pnpm ls array format with dependencies", func(t *testing.T) {
		// pnpm ls --json --depth=0 output format: array with dependencies inside first element
		input := []byte(`[{"name": "my-project", "version": "1.0.0", "dependencies": {"lodash": {"version": "4.17.21"}, "express": {"version": "4.18.2"}}}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "4.18.2", result["express"])
	})

	t.Run("parses pnpm ls array format with devDependencies", func(t *testing.T) {
		// pnpm ls --json --depth=0 output format: array with both deps and devDeps
		input := []byte(`[{"name": "my-project", "version": "1.0.0", "dependencies": {"lodash": {"version": "4.17.21"}}, "devDependencies": {"typescript": {"version": "5.0.0"}}}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "5.0.0", result["typescript"])
	})

	t.Run("parses pnpm ls with scoped packages", func(t *testing.T) {
		// pnpm ls --json output with scoped package names
		input := []byte(`[{"name": "my-project", "dependencies": {"@vue/reactivity": {"version": "3.5.13"}, "@babel/core": {"version": "7.26.0"}}}]`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "3.5.13", result["@vue/reactivity"])
		assert.Equal(t, "7.26.0", result["@babel/core"])
	})

	t.Run("parses yarn list format with data.trees", func(t *testing.T) {
		// yarn list --json --depth=0 output format
		input := []byte(`{"type":"tree","data":{"type":"list","trees":[{"name":"lodash@4.17.21","children":[],"hint":null,"color":null,"depth":0}]}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("parses yarn list with multiple packages", func(t *testing.T) {
		input := []byte(`{"type":"tree","data":{"type":"list","trees":[{"name":"lodash@4.17.21","children":[]},{"name":"express@4.18.2","children":[]}]}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "4.18.2", result["express"])
	})

	t.Run("parses yarn list with scoped packages", func(t *testing.T) {
		input := []byte(`{"type":"tree","data":{"type":"list","trees":[{"name":"@babel/core@7.26.0","children":[]},{"name":"@vue/reactivity@3.5.13","children":[]}]}}`)
		result, err := parseLockCommandJSON(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "7.26.0", result["@babel/core"])
		assert.Equal(t, "3.5.13", result["@vue/reactivity"])
	})
}

// TestParseYarnNameVersion tests the yarn name@version parsing.
//
// It verifies:
//   - Regular packages are parsed correctly
//   - Scoped packages are parsed correctly
//   - Edge cases like empty strings and missing versions are handled
func TestParseYarnNameVersion(t *testing.T) {
	t.Run("parses regular package", func(t *testing.T) {
		name, version := parseYarnNameVersion("lodash@4.17.21")
		assert.Equal(t, "lodash", name)
		assert.Equal(t, "4.17.21", version)
	})

	t.Run("parses scoped package", func(t *testing.T) {
		name, version := parseYarnNameVersion("@babel/core@7.26.0")
		assert.Equal(t, "@babel/core", name)
		assert.Equal(t, "7.26.0", version)
	})

	t.Run("parses deeply scoped package", func(t *testing.T) {
		name, version := parseYarnNameVersion("@vue/reactivity@3.5.13")
		assert.Equal(t, "@vue/reactivity", name)
		assert.Equal(t, "3.5.13", version)
	})

	t.Run("handles empty string", func(t *testing.T) {
		name, version := parseYarnNameVersion("")
		assert.Equal(t, "", name)
		assert.Equal(t, "", version)
	})

	t.Run("handles package without version", func(t *testing.T) {
		name, version := parseYarnNameVersion("lodash")
		assert.Equal(t, "lodash", name)
		assert.Equal(t, "", version)
	})

	t.Run("handles scoped package without version", func(t *testing.T) {
		name, version := parseYarnNameVersion("@babel/core")
		assert.Equal(t, "@babel/core", name)
		assert.Equal(t, "", version)
	})
}

// TestExtractNestedDependencies tests the behavior of nested dependency extraction.
//
// It verifies:
//   - Flat dependencies are extracted correctly
//   - Nested dependencies are extracted recursively
//   - Entries without versions are skipped
//   - Empty versions are skipped
func TestExtractNestedDependencies(t *testing.T) {
	t.Run("extracts flat dependencies", func(t *testing.T) {
		deps := map[string]interface{}{
			"lodash": map[string]interface{}{"version": "4.17.21"},
			"chalk":  map[string]interface{}{"version": "5.0.0"},
		}
		results := make(map[string]string)
		extractNestedDependencies(deps, results)
		assert.Equal(t, "4.17.21", results["lodash"])
		assert.Equal(t, "5.0.0", results["chalk"])
	})

	t.Run("extracts nested dependencies recursively", func(t *testing.T) {
		deps := map[string]interface{}{
			"lodash": map[string]interface{}{
				"version": "4.17.21",
				"dependencies": map[string]interface{}{
					"sub-pkg": map[string]interface{}{"version": "1.0.0"},
				},
			},
		}
		results := make(map[string]string)
		extractNestedDependencies(deps, results)
		assert.Equal(t, "4.17.21", results["lodash"])
		assert.Equal(t, "1.0.0", results["sub-pkg"])
	})

	t.Run("skips entries without version", func(t *testing.T) {
		deps := map[string]interface{}{
			"lodash": map[string]interface{}{"resolved": "https://..."},
		}
		results := make(map[string]string)
		extractNestedDependencies(deps, results)
		assert.Empty(t, results)
	})

	t.Run("skips entries with empty version", func(t *testing.T) {
		deps := map[string]interface{}{
			"lodash": map[string]interface{}{"version": ""},
		}
		results := make(map[string]string)
		extractNestedDependencies(deps, results)
		assert.Empty(t, results)
	})
}

// TestParseLockCommandRaw tests the behavior of raw command output parsing.
//
// It verifies:
//   - Regex patterns with named groups extract correctly
//   - Alternative name group "n" is supported
//   - Matches without names or versions are skipped
//   - Invalid patterns cause errors
func TestParseLockCommandRaw(t *testing.T) {
	t.Run("extracts using pattern with named groups", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{
			Pattern: `(?P<name>[\w-]+)\s+(?P<version>[\d.]+)`,
		}
		input := "lodash 4.17.21\nexpress 4.18.2\n"
		result, err := parseLockCommandRaw(input, extraction)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
		assert.Equal(t, "4.18.2", result["express"])
	})

	t.Run("extracts using alternate name group n", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{
			Pattern: `(?P<n>[\w-]+)@(?P<version>[\d.]+)`,
		}
		input := "lodash@4.17.21"
		result, err := parseLockCommandRaw(input, extraction)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("skips matches without name", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{
			Pattern: `(?P<version>[\d.]+)`,
		}
		input := "4.17.21"
		result, err := parseLockCommandRaw(input, extraction)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("skips matches without version", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{
			Pattern: `(?P<name>[\w-]+)`,
		}
		input := "lodash"
		result, err := parseLockCommandRaw(input, extraction)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error for invalid pattern", func(t *testing.T) {
		extraction := &config.LockCommandExtractionCfg{
			Pattern: `(`,
		}
		_, err := parseLockCommandRaw("input", extraction)
		assert.Error(t, err)
	})
}

// TestExtractVersionsFromCommand tests the behavior of command-based version extraction.
//
// It verifies:
//   - Commands are executed and output is parsed
//   - lock_file and base_dir placeholders are replaced
//   - JSON and raw format extraction both work
//   - Command failures with no output return errors
//   - Timeout configuration is respected
func TestExtractVersionsFromCommand(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")
	require.NoError(t, os.WriteFile(lockPath, []byte(""), 0o644))

	t.Run("executes command and parses JSON output", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			Commands: `echo '{"lodash": "4.17.21"}'`,
		}
		result, err := extractVersionsFromLock(lockPath, cfg)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("supports lock_file placeholder", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			Commands: `echo '{"file": "{{lock_file}}"}'`,
		}
		result, err := extractVersionsFromLock(lockPath, cfg)
		require.NoError(t, err)
		assert.Equal(t, lockPath, result["file"])
	})

	t.Run("supports base_dir placeholder", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			Commands: `echo '{"dir": "{{base_dir}}"}'`,
		}
		result, err := extractVersionsFromLock(lockPath, cfg)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, result["dir"])
	})

	t.Run("returns error when command fails with no output", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			Commands:       `exit 1`,
			TimeoutSeconds: 5,
		}
		_, err := extractVersionsFromLock(lockPath, cfg)
		assert.Error(t, err)
	})

	t.Run("supports raw format extraction", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			Commands: `echo "lodash 4.17.21"`,
			CommandExtraction: &config.LockCommandExtractionCfg{
				Format:  "raw",
				Pattern: `(?P<name>[\w-]+)\s+(?P<version>[\d.]+)`,
			},
		}
		result, err := extractVersionsFromLock(lockPath, cfg)
		require.NoError(t, err)
		assert.Equal(t, "4.17.21", result["lodash"])
	})

	t.Run("respects timeout", func(t *testing.T) {
		cfg := &config.LockFileCfg{
			TimeoutSeconds: 1,
		}
		assert.Equal(t, 1, cfg.GetTimeoutSeconds())
	})
}

// Silence unused import warnings
var _ = utils.FindFilesByPatterns
