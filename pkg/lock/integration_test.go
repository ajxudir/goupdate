package lock

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/packages"
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

// TestIntegration_PNPM tests the behavior of pnpm package resolution with real testdata.
//
// It verifies:
//   - pnpm packages are correctly parsed from package.json
//   - Installed versions are resolved from pnpm-lock.yaml
//   - Package status is correctly set to LockFound
func TestIntegration_PNPM(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/pnpm")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["pnpm"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "package.json"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "pnpm"
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

	// Verify specific versions from pnpm-lock.yaml
	assert.Equal(t, "4.17.21", lookup["lodash"], "lodash should be version 4.17.21")
	assert.Equal(t, "4.18.3", lookup["express"], "express should be version 4.18.3")
	assert.Equal(t, "1.6.8", lookup["axios"], "axios should be version 1.6.8")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}

// TestIntegration_Yarn tests the behavior of yarn package resolution with real testdata.
//
// It verifies:
//   - yarn packages are correctly parsed from package.json
//   - Installed versions are resolved from yarn.lock
//   - Package status is correctly set to LockFound
func TestIntegration_Yarn(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/yarn")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["yarn"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "package.json"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "yarn"
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

	// Verify specific versions from yarn.lock
	assert.Equal(t, "4.17.21", lookup["lodash"], "lodash should be version 4.17.21")
	assert.Equal(t, "4.18.3", lookup["express"], "express should be version 4.18.3")
	assert.Equal(t, "1.6.8", lookup["axios"], "axios should be version 1.6.8")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}

// TestIntegration_Requirements tests the behavior of pip requirements.txt resolution with real testdata.
//
// requirements.txt uses self-pinning mode - the declared version IS the installed version.
// This test verifies packages with pinned versions (==) use declared version.
func TestIntegration_Requirements(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/requirements")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["requirements"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "requirements.txt"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "requirements"
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

	// Self-pinning mode: pinned packages (==) get declared version as installed
	assert.Equal(t, "2.31.0", lookup["requests"], "requests should have declared version as installed (self-pinning)")
	assert.Equal(t, "23.12.0", lookup["black"], "black should have declared version as installed (self-pinning)")

	// Floating constraints (*) should have Floating status
	assert.Equal(t, InstallStatusFloating, statusLookup["celery"], "celery with * should be Floating")
	assert.Equal(t, InstallStatusFloating, statusLookup["uvicorn"], "uvicorn with * should be Floating")
}

// TestIntegration_Pipfile tests the behavior of Pipfile resolution with real testdata.
//
// It verifies:
//   - Pipfile packages are correctly parsed
//   - Installed versions are resolved from Pipfile.lock
//   - Package status is correctly set to LockFound
func TestIntegration_Pipfile(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/pipfile")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["pipfile"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "Pipfile"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "pipfile"
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

	// Verify core packages have installed versions from Pipfile.lock
	assert.NotEmpty(t, lookup["django"], "django should have an installed version")
	assert.NotEmpty(t, lookup["flask"], "flask should have an installed version")
	assert.NotEmpty(t, lookup["requests"], "requests should have an installed version")
	assert.NotEmpty(t, lookup["pytest"], "pytest should have an installed version")

	// Verify specific versions from Pipfile.lock
	assert.Equal(t, "4.2.8", lookup["django"], "django should be version 4.2.8")
	assert.Equal(t, "3.0.1", lookup["flask"], "flask should be version 3.0.1")
	assert.Equal(t, "2.31.0", lookup["requests"], "requests should be version 2.31.0")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["django"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["flask"])
}

// TestIntegration_MSBuild tests the behavior of MSBuild/csproj package resolution with real testdata.
//
// It verifies:
//   - MSBuild packages are correctly parsed from .csproj files
//   - Installed versions are resolved from packages.lock.json
//   - Package status is correctly set to LockFound
func TestIntegration_MSBuild(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/msbuild")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["msbuild"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "TestProject.csproj"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "msbuild"
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

	// Verify core packages have installed versions from packages.lock.json
	assert.NotEmpty(t, lookup["Microsoft.Extensions.Hosting"], "Microsoft.Extensions.Hosting should have an installed version")
	assert.NotEmpty(t, lookup["Microsoft.EntityFrameworkCore"], "Microsoft.EntityFrameworkCore should have an installed version")
	assert.NotEmpty(t, lookup["Swashbuckle.AspNetCore"], "Swashbuckle.AspNetCore should have an installed version")

	// Verify specific versions from packages.lock.json
	assert.Equal(t, "8.0.0", lookup["Microsoft.Extensions.Hosting"], "Microsoft.Extensions.Hosting should be version 8.0.0")
	assert.Equal(t, "8.0.2", lookup["Microsoft.EntityFrameworkCore"], "Microsoft.EntityFrameworkCore should be version 8.0.2")
	assert.Equal(t, "6.5.0", lookup["Swashbuckle.AspNetCore"], "Swashbuckle.AspNetCore should be version 6.5.0")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["Microsoft.Extensions.Hosting"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["Microsoft.EntityFrameworkCore"])
}

// TestIntegration_NuGet tests the behavior of NuGet packages.config resolution with real testdata.
//
// It verifies:
//   - NuGet packages are correctly parsed from packages.config
//   - Installed versions are resolved from packages.lock.json
//   - Package status is correctly set to LockFound
func TestIntegration_NuGet(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/nuget")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["nuget"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "packages.config"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "nuget"
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

	// Verify core packages have installed versions from packages.lock.json
	assert.NotEmpty(t, lookup["Newtonsoft.Json"], "Newtonsoft.Json should have an installed version")
	assert.NotEmpty(t, lookup["Serilog"], "Serilog should have an installed version")
	assert.NotEmpty(t, lookup["Dapper"], "Dapper should have an installed version")

	// Verify specific versions from packages.lock.json
	assert.Equal(t, "13.0.3", lookup["Newtonsoft.Json"], "Newtonsoft.Json should be version 13.0.3")
	assert.Equal(t, "3.1.1", lookup["Serilog"], "Serilog should be version 3.1.1")
	assert.Equal(t, "2.1.28", lookup["Dapper"], "Dapper should be version 2.1.28")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["Newtonsoft.Json"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["Serilog"])
}

// TestIntegration_NPM_LockfileV1 tests npm lockfileVersion 1 (npm 5-6 format).
//
// This ensures backwards compatibility with older npm lock files that use the
// flat "dependencies" object format without the "packages" section.
func TestIntegration_NPM_LockfileV1(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/npm_v1")
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

	// Verify all packages have installed versions from v1 lock file
	assert.Equal(t, "4.17.21", lookup["lodash"], "lodash should be version 4.17.21")
	assert.Equal(t, "4.18.3", lookup["express"], "express should be version 4.18.3")
	assert.Equal(t, "1.6.8", lookup["axios"], "axios should be version 1.6.8")
	assert.Equal(t, "5.4.5", lookup["typescript"], "typescript should be version 5.4.5")
	assert.Equal(t, "3.2.5", lookup["prettier"], "prettier should be version 3.2.5")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}

// TestIntegration_NPM_LockfileV2 tests npm lockfileVersion 2 (npm 7-8 format).
//
// This format includes both "packages" and "dependencies" sections for backwards
// compatibility. npm ls --package-lock-only should handle both formats.
func TestIntegration_NPM_LockfileV2(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/npm_v2")
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

	// Verify all packages have installed versions from v2 lock file
	assert.Equal(t, "4.17.21", lookup["lodash"], "lodash should be version 4.17.21")
	assert.Equal(t, "4.18.3", lookup["express"], "express should be version 4.18.3")
	assert.Equal(t, "1.6.8", lookup["axios"], "axios should be version 1.6.8")
	assert.Equal(t, "5.4.5", lookup["typescript"], "typescript should be version 5.4.5")
	assert.Equal(t, "3.2.5", lookup["prettier"], "prettier should be version 3.2.5")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}

// TestIntegration_PNPM_LockfileV6 tests pnpm lockfileVersion 6.0 (pnpm 8.x format).
//
// This ensures backwards compatibility with pnpm 8.x lock files that use the
// importers section with simpler version strings (no peer dep suffixes).
func TestIntegration_PNPM_LockfileV6(t *testing.T) {
	testdataDir, err := filepath.Abs("../testdata/pnpm_v6")
	require.NoError(t, err, "failed to get absolute path to testdata")

	cfg, err := config.LoadConfig("", testdataDir)
	require.NoError(t, err)

	parser := packages.NewDynamicParser()
	rule := cfg.Rules["pnpm"]
	result, err := parser.ParseFile(filepath.Join(testdataDir, "package.json"), &rule)
	require.NoError(t, err)

	for i := range result.Packages {
		result.Packages[i].Rule = "pnpm"
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

	// Verify all packages have installed versions from v6 lock file
	assert.Equal(t, "4.17.21", lookup["lodash"], "lodash should be version 4.17.21")
	assert.Equal(t, "4.18.3", lookup["express"], "express should be version 4.18.3")
	assert.Equal(t, "1.6.8", lookup["axios"], "axios should be version 1.6.8")
	assert.Equal(t, "5.4.5", lookup["typescript"], "typescript should be version 5.4.5")
	assert.Equal(t, "3.2.5", lookup["prettier"], "prettier should be version 3.2.5")

	// Verify status is correct
	assert.Equal(t, InstallStatusLockFound, statusLookup["lodash"])
	assert.Equal(t, InstallStatusLockFound, statusLookup["express"])
}
