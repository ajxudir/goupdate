package lock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/packages"
)

// TestIntegration_NPM tests the behavior of npm package resolution with real testdata.
//
// It verifies:
//   - npm packages are correctly parsed from package.json
//   - Installed versions are resolved from package-lock.json
//   - Package status is correctly set to LockFound
func TestIntegration_NPM(t *testing.T) {
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

	// Build lookup for easier assertions
	lookup := make(map[string]string)
	statusLookup := make(map[string]string)
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg.InstalledVersion
		statusLookup[pkg.Name] = pkg.InstallStatus
	}

	// Verify core packages have installed versions
	assert.NotEmpty(t, lookup["lodash"], "lodash should have an installed version")
	assert.NotEmpty(t, lookup["express"], "express should have an installed version")
	assert.NotEmpty(t, lookup["axios"], "axios should have an installed version")
	assert.NotEmpty(t, lookup["typescript"], "typescript should have an installed version")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}

// TestIntegration_GoMod tests the behavior of Go module resolution with real testdata.
//
// It verifies:
//   - Go modules are correctly parsed from go.mod
//   - Installed versions are resolved from go.sum
//   - Core packages like cobra and zap are detected
func TestIntegration_GoMod(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/mod")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["mod"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "go.mod"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "mod"
	}

	enriched, err := ApplyInstalledVersions(result.Packages, cfg, testdataDir)
	require.NoError(t, err)

	// Build lookup for easier assertions
	lookup := make(map[string]string)
	statusLookup := make(map[string]string)
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg.InstalledVersion
		statusLookup[pkg.Name] = pkg.InstallStatus
	}

	// Verify core packages are detected
	assert.Contains(t, lookup, "github.com/spf13/cobra")
	assert.Contains(t, lookup, "go.uber.org/zap")
}

// TestIntegration_Composer tests the behavior of Composer package resolution with real testdata.
//
// It verifies:
//   - Composer packages are correctly parsed from composer.json
//   - Installed versions are resolved from composer.lock
//   - Package status is correctly set to LockFound for symfony and guzzle packages
func TestIntegration_Composer(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/composer")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["composer"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "composer.json"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "composer"
	}

	enriched, err := ApplyInstalledVersions(result.Packages, cfg, testdataDir)
	require.NoError(t, err)

	// Build lookup for easier assertions
	lookup := make(map[string]string)
	statusLookup := make(map[string]string)
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg.InstalledVersion
		statusLookup[pkg.Name] = pkg.InstallStatus
	}

	// Verify core packages have installed versions
	assert.NotEmpty(t, lookup["symfony/console"], "symfony/console should have an installed version")
	assert.NotEmpty(t, lookup["guzzlehttp/guzzle"], "guzzlehttp/guzzle should have an installed version")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["symfony/console"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["guzzlehttp/guzzle"])
}

// TestIntegration_LockNotFound tests the behavior when lock file doesn't exist.
//
// It verifies:
//   - All packages have LockMissing status when lock file is not found
//   - InstalledVersion is set to #N/A for all packages
func TestIntegration_LockNotFound(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata_errors/_lock-not-found/npm")
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

	// All packages should have LockMissing status
	for _, pkg := range enriched {
		assert.Equal(t, InstallStatusLockMissing, pkg.InstallStatus,
			"package %s should have LockMissing status", pkg.Name)
		assert.Equal(t, "#N/A", pkg.InstalledVersion)
	}
}

// TestIntegration_LockMissing tests behavior when lock file itself is missing.
//
// When the lock file does not exist, all packages should have LockMissing status
// regardless of their name, because we cannot determine installed versions.
func TestIntegration_LockMissing(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata_errors/_lock-missing/npm")
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

	// Build lookup
	lookup := make(map[string]string)
	statusLookup := make(map[string]string)
	for _, pkg := range enriched {
		lookup[pkg.Name] = pkg.InstalledVersion
		statusLookup[pkg.Name] = pkg.InstallStatus
	}

	// When lock file is missing, all packages should have LockMissing status
	assert.Equal(t, InstallStatusLockMissing, statusLookup["lodash"])
	assert.Equal(t, "#N/A", lookup["lodash"])

	assert.Equal(t, InstallStatusLockMissing, statusLookup["missing-package"])
	assert.Equal(t, "#N/A", lookup["missing-package"])
}
