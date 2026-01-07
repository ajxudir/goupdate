package testutil

import (
	"fmt"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/stretchr/testify/assert"
)

// These tests ensure the test utility functions are covered.
// Since these are helper functions for other tests, we just verify they work correctly.

func TestPackageBuilder(t *testing.T) {
	t.Run("builds package with all fields", func(t *testing.T) {
		pkg := NewPackage("test-pkg").
			WithRule("npm").
			WithType("prod").
			WithPackageType("js").
			WithVersion("1.0.0").
			WithInstalledVersion("1.0.0").
			WithConstraint("^").
			WithSource("package.json").
			WithGroup("frontend").
			Build()

		assert.Equal(t, "test-pkg", pkg.Name)
		assert.Equal(t, "npm", pkg.Rule)
		assert.Equal(t, "prod", pkg.Type)
		assert.Equal(t, "js", pkg.PackageType)
		assert.Equal(t, "1.0.0", pkg.Version)
		assert.Equal(t, "1.0.0", pkg.InstalledVersion)
		assert.Equal(t, "^", pkg.Constraint)
		assert.Equal(t, "package.json", pkg.Source)
		assert.Equal(t, "frontend", pkg.Group)
	})
}

func TestNPMPackage(t *testing.T) {
	pkg := NPMPackage("react", "17.0.0", "17.0.0")

	assert.Equal(t, "react", pkg.Name)
	assert.Equal(t, "npm", pkg.Rule)
	assert.Equal(t, "js", pkg.PackageType)
	assert.Equal(t, "prod", pkg.Type)
	assert.Equal(t, "17.0.0", pkg.Version)
	assert.Equal(t, "17.0.0", pkg.InstalledVersion)
	assert.Equal(t, "^", pkg.Constraint)
}

func TestGoPackage(t *testing.T) {
	pkg := GoPackage("github.com/example/pkg", "v1.0.0", "v1.0.0")

	assert.Equal(t, "github.com/example/pkg", pkg.Name)
	assert.Equal(t, "mod", pkg.Rule)
	assert.Equal(t, "golang", pkg.PackageType)
	assert.Equal(t, "prod", pkg.Type)
	assert.Equal(t, "v1.0.0", pkg.Version)
	assert.Equal(t, "v1.0.0", pkg.InstalledVersion)
}

func TestDotNetPackage(t *testing.T) {
	pkg := DotNetPackage("Newtonsoft.Json", "13.0.0", "13.0.0")

	assert.Equal(t, "Newtonsoft.Json", pkg.Name)
	assert.Equal(t, "nuget", pkg.Rule)
	assert.Equal(t, "dotnet", pkg.PackageType)
	assert.Equal(t, "prod", pkg.Type)
}

func TestPythonPackage(t *testing.T) {
	pkg := PythonPackage("requests", "2.28.0", "2.28.0")

	assert.Equal(t, "requests", pkg.Name)
	assert.Equal(t, "pip", pkg.Rule)
	assert.Equal(t, "python", pkg.PackageType)
	assert.Equal(t, "prod", pkg.Type)
}

func TestConfigBuilder(t *testing.T) {
	t.Run("builds config with all fields", func(t *testing.T) {
		cfg := NewConfig().
			WithWorkingDir("/test/dir").
			WithRule("npm", NPMRule()).
			Build()

		assert.Equal(t, "/test/dir", cfg.WorkingDir)
		assert.Contains(t, cfg.Rules, "npm")
	})
}

func TestNPMRule(t *testing.T) {
	rule := NPMRule()

	assert.Equal(t, "js", rule.Manager)
	assert.Equal(t, "json", rule.Format)
	assert.NotNil(t, rule.Update)
	assert.NotNil(t, rule.Outdated)
}

func TestGoModRule(t *testing.T) {
	rule := GoModRule()

	assert.Equal(t, "go", rule.Manager)
	assert.Equal(t, "raw", rule.Format)
	assert.NotNil(t, rule.Update)
	assert.NotNil(t, rule.Outdated)
}

func TestNuGetRule(t *testing.T) {
	rule := NuGetRule()

	assert.Equal(t, "dotnet", rule.Manager)
	assert.Equal(t, "xml", rule.Format)
	assert.NotNil(t, rule.Update)
	assert.NotNil(t, rule.Outdated)
}

func TestSimpleRule(t *testing.T) {
	rule := SimpleRule("npm install")

	assert.Equal(t, "json", rule.Format)
	assert.NotNil(t, rule.Update)
	assert.Equal(t, "npm install", rule.Update.Commands)
}

func TestRuleWithGroup(t *testing.T) {
	rule := RuleWithGroup("npm install", "frontend")

	assert.Equal(t, "json", rule.Format)
	assert.NotNil(t, rule.Update)
	assert.Equal(t, "npm install", rule.Update.Commands)
	assert.Equal(t, "frontend", rule.Update.Group)
}

func TestCreateUpdateTable(t *testing.T) {
	table := CreateUpdateTable()

	assert.NotNil(t, table)
	// Table should have columns
	assert.Greater(t, table.ColumnCount(), 0)
}

func TestCreateUpdateTableWithGroup(t *testing.T) {
	table := CreateUpdateTableWithGroup()

	assert.NotNil(t, table)
	assert.Greater(t, table.ColumnCount(), 0)
}

func TestCreateOutdatedTable(t *testing.T) {
	table := CreateOutdatedTable()

	assert.NotNil(t, table)
	assert.Greater(t, table.ColumnCount(), 0)
}

func TestCaptureStdout(t *testing.T) {
	output := CaptureStdout(t, func() {
		fmt.Print("hello")
	})

	assert.Equal(t, "hello", output)
}

func TestCaptureStderr(t *testing.T) {
	output := CaptureStderr(t, func() {
		// Write to stderr is tricky in tests, so just verify it doesn't panic
	})

	assert.Empty(t, output)
}

func TestCaptureOutput(t *testing.T) {
	stdout, stderr := CaptureOutput(t, func() {
		fmt.Print("stdout content")
	})

	assert.Equal(t, "stdout content", stdout)
	assert.Empty(t, stderr)
}

func TestCreateSystemTestRunner(t *testing.T) {
	t.Run("creates runner with nil config", func(t *testing.T) {
		runner := CreateSystemTestRunner(nil, false, false)
		assert.NotNil(t, runner)
		// With nil config, runner has no tests
		assert.False(t, runner.HasTests())
	})

	t.Run("creates runner with config", func(t *testing.T) {
		cfg := &config.SystemTestsCfg{
			Tests: []config.SystemTestCfg{
				{Name: "test1", Commands: "echo hello"},
			},
		}
		runner := CreateSystemTestRunner(cfg, true, true)
		assert.NotNil(t, runner)
		assert.True(t, runner.HasTests())
	})
}

func TestComposerRule(t *testing.T) {
	rule := ComposerRule()

	assert.Equal(t, "php", rule.Manager)
	assert.Equal(t, "json", rule.Format)
	assert.NotNil(t, rule.Fields)
	assert.Equal(t, "prod", rule.Fields["require"])
	assert.Equal(t, "dev", rule.Fields["require-dev"])
	assert.NotNil(t, rule.Update)
	assert.Contains(t, rule.Update.Commands, "composer require")
	assert.NotNil(t, rule.Outdated)
	assert.Contains(t, rule.Outdated.Commands, "composer show")
}

func TestComposerPackage(t *testing.T) {
	pkg := ComposerPackage("psr/log", "1.1.0", "1.1.4")

	assert.Equal(t, "psr/log", pkg.Name)
	assert.Equal(t, "composer", pkg.Rule)
	assert.Equal(t, "php", pkg.PackageType)
	assert.Equal(t, "prod", pkg.Type)
	assert.Equal(t, "1.1.0", pkg.Version)
	assert.Equal(t, "1.1.4", pkg.InstalledVersion)
	assert.Equal(t, "^", pkg.Constraint)
}
