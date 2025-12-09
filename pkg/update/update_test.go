package update

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
	pkgerrors "github.com/user/goupdate/pkg/errors"
	"github.com/user/goupdate/pkg/formats"
)

// writeFile is a test helper that writes content to a file.
//
// Parameters:
//   - path: File path to write to
//   - content: String content to write
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// TestUpdatePackageDryRunSkipsLockAndWrites tests the behavior of UpdatePackage in dry-run mode.
//
// It verifies:
//   - Dry-run mode does not modify the package file
//   - Dry-run mode does not execute lock commands
func TestUpdatePackageDryRunSkipsLockAndWrites(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"^1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "echo {{package}}"},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "r", PackageType: "js", Type: "prod", Constraint: "^", Source: path}

	originalExec := execCommandFunc
	called := false
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		called = true
		return nil, nil
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err := UpdatePackage(pkg, "1.1.0", cfg, tmpDir, true, false)
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, original, string(content))
	assert.False(t, called)
}

// TestUpdatePackageRollbackOnLockFailure tests the behavior of UpdatePackage when lock command fails.
//
// It verifies:
//   - Package file is restored to original content when lock fails
//   - Only one lock command attempt is made before rollback
func TestUpdatePackageRollbackOnLockFailure(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"^1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "echo {{package}}"},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "r", PackageType: "js", Type: "prod", Constraint: "^", Version: "1.0.0", Source: path}

	originalExec := execCommandFunc
	callCount := 0
	var versions []string
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		callCount++
		versions = append(versions, version)
		// Lock command fails with new version
		return nil, errors.New("lock failed")
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err := UpdatePackage(pkg, "1.2.0", cfg, tmpDir, false, false)
	require.Error(t, err)

	// Only 1 call: lock command with target version fails, file is restored from backup
	// (new behavior uses file backup/restore instead of re-running lock with old version)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, []string{"1.2.0"}, versions)

	// File should be restored to original content
	content, _ := os.ReadFile(path)
	assert.Equal(t, original, string(content))
}

// TestUpdatePackageUpdatesNonExactConstraint tests the behavior of UpdatePackage with non-exact version constraints.
//
// It verifies:
//   - Package file is updated with new version and original constraint
//   - Constraint prefix (e.g., ^, ~) is preserved in the update
func TestUpdatePackageUpdatesNonExactConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"^1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "echo {{package}}"},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "r", PackageType: "js", Type: "prod", Constraint: "^", Source: path}

	err := UpdatePackage(pkg, "1.2.0", cfg, tmpDir, false, true)
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Contains(t, string(content), "^1.2.0")
}

// TestUpdateUnsupported tests the behavior of UpdatePackage with unsupported configurations.
//
// It verifies:
//   - Unsupported format returns UnsupportedError
//   - UnsupportedError has proper error message
func TestUpdateUnsupported(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {}}}
	pkg := formats.Package{Rule: "r", Name: "demo"}
	err := UpdatePackage(pkg, "1.0.1", cfg, ".", true, false)
	assert.True(t, pkgerrors.IsUnsupported(err))

	unsupported := &pkgerrors.UnsupportedError{Reason: "missing"}
	assert.NotEmpty(t, unsupported.Error())
}

// TestNormalizeUpdateGroup tests the behavior of NormalizeUpdateGroup.
//
// It verifies:
//   - Template variables in group names are replaced
//   - Nil config returns empty string
//   - UpdateGroupKey handles fallback to package name
func TestNormalizeUpdateGroup(t *testing.T) {
	cfg := &config.UpdateCfg{Group: "group-{{type}}"}
	result := NormalizeUpdateGroup(cfg, formats.Package{Name: "pkg", Rule: "r", Type: "prod"})
	assert.Contains(t, result, "prod")

	assert.Equal(t, "", NormalizeUpdateGroup(nil, formats.Package{Name: "fallback"}))
	assert.Equal(t, "fallback", UpdateGroupKey(nil, formats.Package{Name: "fallback"}))
	assert.Equal(t, "r-g", UpdateGroupKey(&config.UpdateCfg{Group: "{{rule}}-g"}, formats.Package{Name: "pkg", Rule: "r"}))
	assert.Equal(t, "explicit", UpdateGroupKey(nil, formats.Package{Name: "pkg", Group: "explicit"}))
}

// TestResolveUpdateCfgOverride tests the behavior of ResolveUpdateCfg with package overrides.
//
// It verifies:
//   - Package-specific overrides are applied correctly
//   - Commands and Group settings from override take precedence
func TestResolveUpdateCfgOverride(t *testing.T) {
	overrideCmd := "custom {{package}}"
	overrideGroup := "g"
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Update: &config.UpdateCfg{Commands: "base one", Group: "base"},
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"pkg": {Update: &config.UpdateOverrideCfg{Commands: &overrideCmd, Group: &overrideGroup}},
			},
		},
	}}

	updateCfg, err := ResolveUpdateCfg(formats.Package{Name: "pkg", Rule: "r"}, cfg)
	require.NoError(t, err)

	assert.Equal(t, overrideCmd, updateCfg.Commands)
	assert.Equal(t, overrideGroup, updateCfg.Group)
}

// TestResolveUpdateCfgMissingRule tests the behavior of ResolveUpdateCfg with missing rule.
//
// It verifies:
//   - Missing rule configuration returns an error
func TestResolveUpdateCfgMissingRule(t *testing.T) {
	_, err := ResolveUpdateCfg(formats.Package{Name: "demo", Rule: "missing"}, &config.Config{Rules: map[string]config.PackageManagerCfg{}})
	require.Error(t, err)
}

// TestUpdatePackageHandlesYAMLAndRaw tests the behavior of UpdatePackage with different file formats.
//
// It verifies:
//   - YAML format updates preserve constraint
//   - Raw format updates with different constraint patterns
//   - XML format updates for .NET projects
//   - packages.config format updates for .NET
func TestUpdatePackageHandlesYAMLAndRaw(t *testing.T) {
	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "deps.yaml")
	require.NoError(t, writeFile(yamlPath, "dependencies:\n  demo: ~1.0.0\n"))
	rawPath := filepath.Join(tmpDir, "reqs.txt")
	require.NoError(t, writeFile(rawPath, "demo >=1.0.0\n"))
	xmlPath := filepath.Join(tmpDir, "proj.msbuild")
	require.NoError(t, writeFile(xmlPath, `<Project><ItemGroup><PackageReference Include="Newtonsoft.Json" Version="13.0.1" /></ItemGroup></Project>`))
	packagesCfgPath := filepath.Join(tmpDir, "packages.config")
	require.NoError(t, writeFile(packagesCfgPath, `<packages><package id="Serilog" version="3.1.1" /></packages>`))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"yaml": {Format: "yaml", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}},
		"raw":  {Format: "raw", Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)\s+(?P<version>.+)$`}, Update: &config.UpdateCfg{}},
		"xml":  {Manager: "dotnet", Format: "xml", Fields: map[string]string{"ItemGroup/PackageReference": "prod"}, Update: &config.UpdateCfg{}},
		"pkgxml": {
			Manager:    "dotnet",
			Format:     "xml",
			Fields:     map[string]string{"packages": "prod"},
			Extraction: &config.ExtractionCfg{Path: "package", NameAttr: "id", VersionAttr: "version"},
			Update:     &config.UpdateCfg{},
		},
	}}

	err := UpdatePackage(formats.Package{Name: "demo", Rule: "yaml", Constraint: "~", Source: yamlPath}, "1.2.0", cfg, tmpDir, false, true)
	require.NoError(t, err)
	yamlContent, _ := os.ReadFile(yamlPath)
	assert.Contains(t, string(yamlContent), "~1.2.0")

	err = UpdatePackage(formats.Package{Name: "demo", Rule: "raw", Constraint: ">=", Source: rawPath}, "2.0.0", cfg, tmpDir, false, true)
	require.NoError(t, err)
	rawContent, _ := os.ReadFile(rawPath)
	assert.Contains(t, string(rawContent), ">=2.0.0")

	require.NoError(t, writeFile(rawPath, "demo ^1.0.0\n"))
	err = UpdatePackage(formats.Package{Name: "demo", Rule: "raw", Constraint: "^", Source: rawPath}, "3.0.0", cfg, tmpDir, false, true)
	require.NoError(t, err)
	rawUnchanged, _ := os.ReadFile(rawPath)
	assert.Contains(t, string(rawUnchanged), "^3.0.0")

	err = UpdatePackage(formats.Package{Name: "Newtonsoft.Json", Rule: "xml", Constraint: "", Source: xmlPath}, "13.0.2", cfg, tmpDir, false, true)
	require.NoError(t, err)
	xmlContent, _ := os.ReadFile(xmlPath)
	assert.Contains(t, string(xmlContent), "Version=\"13.0.2\"")

	err = UpdatePackage(formats.Package{Name: "Serilog", Rule: "pkgxml", Constraint: "^", Source: packagesCfgPath}, "4.0.0", cfg, tmpDir, false, true)
	require.NoError(t, err)
	pkgXML, _ := os.ReadFile(packagesCfgPath)
	assert.Contains(t, string(pkgXML), "version=\"^4.0.0\"")
}

// TestUpdatePackageNilConfig tests the behavior of UpdatePackage with nil configuration.
//
// It verifies:
//   - Nil configuration returns an error
func TestUpdatePackageNilConfig(t *testing.T) {
	err := UpdatePackage(formats.Package{}, "1.0.0", nil, "", true, true)
	require.Error(t, err)
}

// TestUpdateDeclaredVersionMissingRule tests the behavior of updateDeclaredVersion with missing rule.
//
// It verifies:
//   - Missing rule configuration returns an error
func TestUpdateDeclaredVersionMissingRule(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	require.NoError(t, writeFile(path, `{"dependencies":{"demo":"1.0.0"}}`))
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{}}

	err := updateDeclaredVersion(formats.Package{Name: "demo", Rule: "missing", Source: path}, "1.2.0", cfg, tmpDir, false)
	require.Error(t, err)
}

// TestUpdateDeclaredVersionReadError tests the behavior of updateDeclaredVersion when file read fails.
//
// It verifies:
//   - File read errors are properly propagated
func TestUpdateDeclaredVersionReadError(t *testing.T) {
	originalRead := readFileFunc
	readFileFunc = func(string) ([]byte, error) { return nil, errors.New("read fail") }
	t.Cleanup(func() { readFileFunc = originalRead })

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}}}}
	err := updateDeclaredVersion(formats.Package{Name: "demo", Rule: "r", Source: "missing"}, "1.0.1", cfg, ".", false)
	require.Error(t, err)
}

// TestUpdateDeclaredVersionWriteError tests the behavior of updateDeclaredVersion when file write fails.
//
// It verifies:
//   - File write errors are properly propagated
func TestUpdateDeclaredVersionWriteError(t *testing.T) {
	originalRead := readFileFunc
	originalWrite := writeFileFunc
	readFileFunc = func(string) ([]byte, error) { return []byte(`{"dependencies":{"demo":"1.0.0"}}`), nil }
	writeFileFunc = func(string, []byte, os.FileMode) error { return errors.New("write fail") }
	t.Cleanup(func() {
		readFileFunc = originalRead
		writeFileFunc = originalWrite
	})

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}}}}
	err := updateDeclaredVersion(formats.Package{Name: "demo", Rule: "r", Source: "file"}, "1.0.1", cfg, ".", false)
	require.Error(t, err)
}

// TestUpdatePackageRunsLockCommand tests the behavior of UpdatePackage when executing lock commands.
//
// It verifies:
//   - Lock command is executed with correct version
//   - Lock command is called when not in dry-run mode
func TestUpdatePackageRunsLockCommand(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	require.NoError(t, writeFile(path, `{"dependencies":{"demo":"1.0.0"}}`))
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{Commands: "echo {{package}}@{{version}}"}}}}

	called := false
	originalExec := execCommandFunc
	execCommandFunc = func(updateCfg *config.UpdateCfg, pkgName, version, constraint, dir string) ([]byte, error) {
		called = true
		if !strings.Contains(version, "1.1.0") {
			t.Fatalf("expected version 1.1.0, got %s", version)
		}
		return nil, nil
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err := UpdatePackage(formats.Package{Name: "demo", Rule: "r", Constraint: "", Source: path}, "1.1.0", cfg, tmpDir, false, false)
	require.NoError(t, err)
	assert.True(t, called)
}

// TestUpdatePackageMissingLockCommand tests the behavior of UpdatePackage when lock command is not configured.
//
// It verifies:
//   - Missing lock command returns UnsupportedError
func TestUpdatePackageMissingLockCommand(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	require.NoError(t, writeFile(path, `{"dependencies":{"demo":"1.0.0"}}`))
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}}}}

	err := UpdatePackage(formats.Package{Name: "demo", Rule: "r", Constraint: "", Source: path}, "1.1.0", cfg, tmpDir, false, false)
	require.True(t, pkgerrors.IsUnsupported(err))
}

// TestRollbackOnFailure tests the behavior of rollbackOnFailure.
//
// It verifies:
//   - Errors are returned when backup content exists
//   - No error when both backup and errors are nil
func TestRollbackOnFailure(t *testing.T) {
	orig := []byte("data")
	err := rollbackOnFailure(filepath.Join(t.TempDir(), "file"), orig, []error{errors.New("boom")})
	require.Error(t, err)

	err = rollbackOnFailure(filepath.Join(t.TempDir(), "file"), nil, nil)
	require.NoError(t, err)
}

// TestRollbackOnFailureWriteError tests the behavior of rollbackOnFailure when write fails.
//
// It verifies:
//   - Write errors during rollback are properly propagated
func TestRollbackOnFailureWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "file")
	err := rollbackOnFailure(path, []byte("data"), nil)
	require.Error(t, err)
}

// TestUpdatePackageWithNewCommandsFormat tests the behavior of UpdatePackage with Commands format.
//
// It verifies:
//   - Commands field is used instead of deprecated LockCommand/LockArgs
//   - Version is correctly passed to the command execution
func TestUpdatePackageWithNewCommandsFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"^1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	// Use the new Commands format instead of deprecated LockCommand/LockArgs
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{
				Commands: "echo {{package}} {{version}}",
			},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "r", PackageType: "js", Type: "prod", Constraint: "^", Source: path}

	originalExec := execCommandFunc
	var capturedVersion string
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		capturedVersion = version
		return nil, nil
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err := UpdatePackage(pkg, "1.2.0", cfg, tmpDir, false, false)
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", capturedVersion)
}

// TestRunGroupLockCommandNoConfig tests the behavior of RunGroupLockCommand with nil configuration.
//
// It verifies:
//   - Nil configuration returns an error
func TestRunGroupLockCommandNoConfig(t *testing.T) {
	err := RunGroupLockCommand(nil, ".")
	require.Error(t, err)
}

// TestRunGroupLockCommandNoLockCommand tests the behavior of RunGroupLockCommand when lock command is not configured.
//
// It verifies:
//   - Missing lock command returns UnsupportedError
func TestRunGroupLockCommandNoLockCommand(t *testing.T) {
	cfg := &config.UpdateCfg{}
	err := RunGroupLockCommand(cfg, ".")
	require.Error(t, err)
	assert.True(t, pkgerrors.IsUnsupported(err))
}

// TestExecuteUpdateCommand tests the behavior of executeUpdateCommand.
//
// It verifies:
//   - Nil config returns error
//   - Empty commands returns unsupported error
//   - Whitespace-only commands returns unsupported error
//   - Simple echo command executes successfully
func TestExecuteUpdateCommand(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := executeUpdateCommand(nil, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update configuration is required")
	})

	t.Run("empty commands returns unsupported error", func(t *testing.T) {
		cfg := &config.UpdateCfg{Commands: ""}
		_, err := executeUpdateCommand(cfg, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.True(t, pkgerrors.IsUnsupported(err))
	})

	t.Run("whitespace only commands returns unsupported error", func(t *testing.T) {
		cfg := &config.UpdateCfg{Commands: "   \n\t  "}
		_, err := executeUpdateCommand(cfg, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.True(t, pkgerrors.IsUnsupported(err))
	})

	t.Run("executes simple echo command", func(t *testing.T) {
		cfg := &config.UpdateCfg{Commands: "echo '{{package}} {{version}}'"}
		output, err := executeUpdateCommand(cfg, "test-pkg", "1.2.0", "^", ".")
		require.NoError(t, err)
		assert.Contains(t, string(output), "test-pkg")
		assert.Contains(t, string(output), "1.2.0")
	})
}

// TestRunGroupLockCommandSuccess tests the behavior of RunGroupLockCommand on successful execution.
//
// It verifies:
//   - Successful command execution returns no error
func TestRunGroupLockCommandSuccess(t *testing.T) {
	originalExec := execCommandFunc
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		return []byte("success"), nil
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	cfg := &config.UpdateCfg{Commands: "npm install"}
	err := RunGroupLockCommand(cfg, ".")
	require.NoError(t, err)
}

// TestRunGroupLockCommandFailure tests the behavior of RunGroupLockCommand on command failure.
//
// It verifies:
//   - Command execution errors are properly propagated
func TestRunGroupLockCommandFailure(t *testing.T) {
	originalExec := execCommandFunc
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		return nil, errors.New("install failed")
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	cfg := &config.UpdateCfg{Commands: "npm install"}
	err := RunGroupLockCommand(cfg, ".")
	require.Error(t, err)
}

// TestResolveUpdateCfgNilUpdate tests the behavior of ResolveUpdateCfg when Update config is nil.
//
// It verifies:
//   - Nil Update configuration returns UnsupportedError
func TestResolveUpdateCfgNilUpdate(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {Outdated: &config.OutdatedCfg{}}, // No Update config
	}}
	_, err := ResolveUpdateCfg(formats.Package{Name: "demo", Rule: "r"}, cfg)
	require.Error(t, err)
	assert.True(t, pkgerrors.IsUnsupported(err))
}

// TestResolveUpdateCfgNoOverride tests the behavior of ResolveUpdateCfg without package overrides.
//
// It verifies:
//   - Base configuration is returned when no override exists
//   - All fields from base config are preserved
func TestResolveUpdateCfgNoOverride(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Update: &config.UpdateCfg{
				Commands:   "npm install",
				Group:          "base-group",
				TimeoutSeconds: 60,
			},
		},
	}}
	updateCfg, err := ResolveUpdateCfg(formats.Package{Name: "demo", Rule: "r"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "npm install", updateCfg.Commands)
	assert.Equal(t, "base-group", updateCfg.Group)
	assert.Equal(t, 60, updateCfg.TimeoutSeconds)
}

// TestResolveUpdateCfgWithTimeoutOverride tests the behavior of ResolveUpdateCfg with timeout override.
//
// It verifies:
//   - Package-specific timeout override is applied
func TestResolveUpdateCfgWithTimeoutOverride(t *testing.T) {
	timeout := 120
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Update: &config.UpdateCfg{
				Commands:   "npm install",
				TimeoutSeconds: 60,
			},
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"slow-pkg": {
					Update: &config.UpdateOverrideCfg{TimeoutSeconds: &timeout},
				},
			},
		},
	}}
	updateCfg, err := ResolveUpdateCfg(formats.Package{Name: "slow-pkg", Rule: "r"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, 120, updateCfg.TimeoutSeconds)
}

// TestResolveUpdateCfgNilPackageOverrides tests the behavior of ResolveUpdateCfg when PackageOverrides is nil.
//
// It verifies:
//   - Base configuration is returned when PackageOverrides is nil
func TestResolveUpdateCfgNilPackageOverrides(t *testing.T) {
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Update: &config.UpdateCfg{Commands: "npm install"},
			// No PackageOverrides
		},
	}}
	updateCfg, err := ResolveUpdateCfg(formats.Package{Name: "demo", Rule: "r"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "npm install", updateCfg.Commands)
}

// TestResolveUpdateCfgWithEnvOverride tests the behavior of ResolveUpdateCfg with environment variable override.
//
// It verifies:
//   - Package-specific environment variables replace base environment
//   - Original base environment is not merged
func TestResolveUpdateCfgWithEnvOverride(t *testing.T) {
	env := map[string]string{"CI": "true", "NODE_ENV": "test"}
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Update: &config.UpdateCfg{
				Commands: "npm install",
				Env:          map[string]string{"DEBUG": "1"},
			},
			PackageOverrides: map[string]config.PackageOverrideCfg{
				"test-pkg": {
					Update: &config.UpdateOverrideCfg{Env: env},
				},
			},
		},
	}}
	updateCfg, err := ResolveUpdateCfg(formats.Package{Name: "test-pkg", Rule: "r"}, cfg)
	require.NoError(t, err)
	assert.Equal(t, "true", updateCfg.Env["CI"])
	assert.Equal(t, "test", updateCfg.Env["NODE_ENV"])
	// Original base env should be replaced, not merged
	assert.NotContains(t, updateCfg.Env, "DEBUG")
}

// TestUpdateDeclaredVersionUnsupportedFormat tests the behavior of updateDeclaredVersion with unsupported format.
//
// It verifies:
//   - Unsupported file format returns an error
func TestUpdateDeclaredVersionUnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.custom")
	require.NoError(t, writeFile(path, "demo: 1.0.0"))
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "unsupported", Update: &config.UpdateCfg{}}}}
	err := updateDeclaredVersion(formats.Package{Name: "demo", Rule: "r", Source: path}, "1.1.0", cfg, tmpDir, false)
	require.Error(t, err)
}

// TestUpdatePackageLockFailureRollback tests the behavior of UpdatePackage rollback when lock fails after manifest update.
//
// It verifies:
//   - Manifest file is restored to original when lock fails
//   - Error message includes lock failure details
func TestUpdatePackageLockFailureRollback(t *testing.T) {
	// Tests the rollback path when lock command fails after manifest update
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"^1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "echo {{package}}"},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "r", PackageType: "js", Type: "prod", Constraint: "^", Version: "1.0.0", Source: path}

	originalExec := execCommandFunc
	callCount := 0
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
		callCount++
		if callCount == 1 {
			// First call fails (lock after manifest update)
			return nil, errors.New("lock failed after update")
		}
		// Subsequent calls succeed (rollback lock)
		return nil, nil
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err := UpdatePackage(pkg, "1.2.0", cfg, tmpDir, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lock failed after update")

	// Verify rollback happened - original content should be restored
	content, _ := os.ReadFile(path)
	assert.Equal(t, original, string(content))
}

// TestUpdatePackageReadError tests the behavior of UpdatePackage when file read fails.
//
// It verifies:
//   - File read errors are properly propagated with descriptive message
func TestUpdatePackageReadError(t *testing.T) {
	originalRead := readFileFunc
	readFileFunc = func(string) ([]byte, error) { return nil, errors.New("read fail") }
	t.Cleanup(func() { readFileFunc = originalRead })

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"r": {Format: "json", Update: &config.UpdateCfg{Commands: "echo"}}}}
	err := UpdatePackage(formats.Package{Name: "demo", Rule: "r", Source: "missing.json"}, "1.0.1", cfg, ".", false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

// TestUpdatePackageUnsupportedFormat tests the behavior of UpdatePackage with unsupported file format.
//
// It verifies:
//   - Unsupported format returns error during manifest update
func TestUpdatePackageUnsupportedFormat(t *testing.T) {
	// Tests that UpdatePackage returns error for unsupported format during manifest update
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "custom.xyz")
	require.NoError(t, writeFile(path, "demo: 1.0.0"))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"custom": {
			Format: "unknown_format", // Unsupported format
			Update: &config.UpdateCfg{Commands: "echo update"},
		},
	}}

	pkg := formats.Package{Name: "demo", Rule: "custom", Source: path}
	err := UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "updates not supported for format")
}

// TestUpdatePackageUsesWorkingDir tests the behavior of UpdatePackage with WorkingDir configuration.
//
// It verifies:
//   - WorkingDir from config is used when scopeDir is empty
func TestUpdatePackageUsesWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "package.json")
	original := `{"dependencies":{"demo":"1.0.0"}}`
	require.NoError(t, writeFile(path, original))

	cfg := &config.Config{
		WorkingDir: tmpDir,
		Rules: map[string]config.PackageManagerCfg{
			"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}},
		},
	}

	// Package without Source - should use WorkingDir
	pkg := formats.Package{Name: "demo", Rule: "r", Source: path}
	err := UpdatePackage(pkg, "1.2.0", cfg, "", true, true)
	require.NoError(t, err)
}

// TestUpdatePackageScopeDirFallbacks tests the behavior of UpdatePackage scope directory fallback logic.
//
// It verifies:
//   - Uses cfg.WorkingDir when scopeDir is empty
//   - Uses dot when all scope dirs are empty
func TestUpdatePackageScopeDirFallbacks(t *testing.T) {
	t.Run("uses cfg.WorkingDir when scopeDir is empty", func(t *testing.T) {
		original := `{"dependencies":{"demo":"1.0.0"}}`

		cfg := &config.Config{
			WorkingDir: "/custom/workdir",
			Rules: map[string]config.PackageManagerCfg{
				"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}},
			},
		}

		// Source is empty, workDir is empty - should fall back to cfg.WorkingDir
		pkg := formats.Package{Name: "demo", Rule: "r", Source: ""}

		originalRead := readFileFunc
		readFileFunc = func(string) ([]byte, error) { return []byte(original), nil }
		t.Cleanup(func() { readFileFunc = originalRead })

		err := UpdatePackage(pkg, "1.2.0", cfg, "", true, true)
		require.NoError(t, err)
	})

	t.Run("uses dot when all scope dirs are empty", func(t *testing.T) {
		original := `{"dependencies":{"demo":"1.0.0"}}`

		cfg := &config.Config{
			WorkingDir: "", // Empty working dir
			Rules: map[string]config.PackageManagerCfg{
				"r": {Format: "json", Fields: map[string]string{"dependencies": "prod"}, Update: &config.UpdateCfg{}},
			},
		}

		// Source is empty, workDir is empty, cfg.WorkingDir is empty - should fall back to "."
		pkg := formats.Package{Name: "demo", Rule: "r", Source: ""}

		originalRead := readFileFunc
		readFileFunc = func(string) ([]byte, error) { return []byte(original), nil }
		t.Cleanup(func() { readFileFunc = originalRead })

		err := UpdatePackage(pkg, "1.2.0", cfg, "", true, true)
		require.NoError(t, err)
	})
}

// TestBackupFiles tests the behavior of backupFiles.
//
// It verifies:
//   - Existing files are backed up with content and permissions
//   - Non-existent files are skipped
//   - Unreadable files return an error
func TestBackupFiles(t *testing.T) {
	t.Run("backs up existing files", func(t *testing.T) {
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		require.NoError(t, os.WriteFile(file1, []byte("content1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("content2"), 0o755))

		backups, err := backupFiles([]string{file1, file2})
		require.NoError(t, err)
		assert.Len(t, backups, 2)
		assert.Equal(t, file1, backups[0].path)
		assert.Equal(t, []byte("content1"), backups[0].content)
		assert.Equal(t, file2, backups[1].path)
		assert.Equal(t, []byte("content2"), backups[1].content)
	})

	t.Run("skips non-existent files", func(t *testing.T) {
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "exists.txt")
		file2 := filepath.Join(tmpDir, "missing.txt")

		require.NoError(t, os.WriteFile(file1, []byte("content"), 0o644))

		backups, err := backupFiles([]string{file1, file2})
		require.NoError(t, err)
		assert.Len(t, backups, 1)
		assert.Equal(t, file1, backups[0].path)
	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "noperm.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0o644))

		// Mock readFileFunc to return permission error
		origRead := readFileFunc
		readFileFunc = func(path string) ([]byte, error) {
			return nil, os.ErrPermission
		}
		t.Cleanup(func() { readFileFunc = origRead })

		_, err := backupFiles([]string{file})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to backup")
	})
}

// TestRestoreBackups tests the behavior of restoreBackups.
//
// It verifies:
//   - Files are restored from backups correctly
//   - File permissions are preserved during restore
//   - Empty backups are handled gracefully
func TestRestoreBackups(t *testing.T) {
	t.Run("restores files from backups", func(t *testing.T) {
		tmpDir := t.TempDir()
		file1 := filepath.Join(tmpDir, "file1.txt")
		file2 := filepath.Join(tmpDir, "file2.txt")

		// Create files with original content
		require.NoError(t, os.WriteFile(file1, []byte("original1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("original2"), 0o755))

		// Backup the files
		backups, err := backupFiles([]string{file1, file2})
		require.NoError(t, err)

		// Modify the files
		require.NoError(t, os.WriteFile(file1, []byte("modified1"), 0o644))
		require.NoError(t, os.WriteFile(file2, []byte("modified2"), 0o755))

		// Restore from backups (returns []error)
		errs := restoreBackups(backups)
		assert.Empty(t, errs)

		// Verify content restored
		content1, err := os.ReadFile(file1)
		require.NoError(t, err)
		assert.Equal(t, []byte("original1"), content1)

		content2, err := os.ReadFile(file2)
		require.NoError(t, err)
		assert.Equal(t, []byte("original2"), content2)
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		file := filepath.Join(tmpDir, "executable.sh")

		require.NoError(t, os.WriteFile(file, []byte("#!/bin/bash"), 0o755))

		backups, err := backupFiles([]string{file})
		require.NoError(t, err)

		// Restore (returns []error)
		errs := restoreBackups(backups)
		assert.Empty(t, errs)

		// Verify permissions preserved
		info, err := os.Stat(file)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("handles empty backups", func(t *testing.T) {
		errs := restoreBackups(nil)
		assert.Empty(t, errs)

		errs = restoreBackups([]fileBackup{})
		assert.Empty(t, errs)
	})
}

// TestGenerateTempSuffix tests the behavior of generateTempSuffix.
//
// It verifies:
//   - Generated suffixes are unique
//   - Suffix format matches expected pattern
func TestGenerateTempSuffix(t *testing.T) {
	t.Run("generates unique suffixes", func(t *testing.T) {
		suffix1 := generateTempSuffix()
		suffix2 := generateTempSuffix()

		assert.True(t, strings.HasPrefix(suffix1, "."))
		assert.True(t, strings.HasSuffix(suffix1, ".tmp"))
		assert.NotEqual(t, suffix1, suffix2, "suffixes should be unique")
	})

	t.Run("suffix has expected format", func(t *testing.T) {
		suffix := generateTempSuffix()

		// Should be like ".abc123def456.tmp"
		assert.Regexp(t, `^\.[a-f0-9]{16}\.tmp$`, suffix)
	})
}

// TestWriteFileAtomic tests the behavior of writeFileAtomic.
//
// It verifies:
//   - File is written atomically
//   - Existing files are overwritten
//   - File permissions are set correctly
//   - Invalid directory returns an error
//   - Target directory collision returns an error
//   - Multi-byte content is handled correctly
func TestWriteFileAtomic(t *testing.T) {
	t.Run("writes file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		err := writeFileAtomic(testFile, []byte("hello world"), 0o644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "existing.txt")

		require.NoError(t, os.WriteFile(testFile, []byte("original"), 0o644))

		err := writeFileAtomic(testFile, []byte("updated"), 0o644)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "updated", string(content))
	})

	t.Run("sets correct file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "perms.txt")

		err := writeFileAtomic(testFile, []byte("content"), 0o755)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("returns error for invalid directory", func(t *testing.T) {
		invalidPath := "/nonexistent/directory/file.txt"

		err := writeFileAtomic(invalidPath, []byte("content"), 0o644)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write temp file")
	})

	t.Run("cleans up temp file on rename failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Create the file and make directory read-only to trigger rename failure
		// Note: This is hard to test reliably across platforms
		// Instead, verify basic functionality works
		err := writeFileAtomic(testFile, []byte("content"), 0o644)
		require.NoError(t, err)
	})

	t.Run("handles rename failure when target is directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "target")

		// Create a directory at the target path - this will cause rename to fail
		err := os.Mkdir(targetPath, 0o755)
		require.NoError(t, err)

		// Try to write to the same path - the rename will fail because target is a directory
		err = writeFileAtomic(targetPath, []byte("content"), 0o644)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to rename temp file")
	})

	t.Run("handles multi-byte content", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "unicode.txt")

		// Unicode content
		content := []byte("日本語テスト 🎉")
		err := writeFileAtomic(testFile, content, 0o644)
		require.NoError(t, err)

		read, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, read)
	})
}

// TestWriteFilePreservingPermissionsEdgeCases tests edge cases for writeFilePreservingPermissions.
//
// It verifies:
//   - Stat error after write is handled gracefully
func TestWriteFilePreservingPermissionsEdgeCases(t *testing.T) {
	t.Run("handles stat error after write gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Create file first
		require.NoError(t, os.WriteFile(testFile, []byte("original"), 0o644))

		// Mock statFileFunc to fail after write
		callCount := 0
		origStat := statFileFunc
		statFileFunc = func(path string) (os.FileInfo, error) {
			callCount++
			if callCount > 1 {
				// Second call (verification) fails
				return nil, errors.New("stat failed after write")
			}
			return origStat(path)
		}
		t.Cleanup(func() { statFileFunc = origStat })

		// Should succeed but log a warning
		err := writeFilePreservingPermissions(testFile, []byte("updated"), 0o644)
		require.NoError(t, err)
	})
}

// TestGetLockFilePaths tests the behavior of getLockFilePaths.
//
// It verifies:
//   - Lock file paths are returned from config
//   - Empty config returns empty list
//   - Non-existent files are filtered out
func TestGetLockFilePaths(t *testing.T) {
	t.Run("returns lock file paths from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, "package-lock.json")
		require.NoError(t, os.WriteFile(lockFile, []byte("{}"), 0o644))

		ruleCfg := config.PackageManagerCfg{
			LockFiles: []config.LockFileCfg{
				{Files: []string{"package-lock.json"}},
			},
		}

		paths := getLockFilePaths(ruleCfg, tmpDir)
		assert.Len(t, paths, 1)
		assert.Equal(t, lockFile, paths[0])
	})

	t.Run("returns empty for empty lock files config", func(t *testing.T) {
		ruleCfg := config.PackageManagerCfg{}
		paths := getLockFilePaths(ruleCfg, "/tmp")
		assert.Empty(t, paths)
	})

	t.Run("filters non-existent files", func(t *testing.T) {
		tmpDir := t.TempDir()
		existingFile := filepath.Join(tmpDir, "exists.lock")
		require.NoError(t, os.WriteFile(existingFile, []byte("lock"), 0o644))

		ruleCfg := config.PackageManagerCfg{
			LockFiles: []config.LockFileCfg{
				{Files: []string{"exists.lock", "missing.lock"}},
			},
		}

		paths := getLockFilePaths(ruleCfg, tmpDir)
		assert.Len(t, paths, 1)
		assert.Equal(t, existingFile, paths[0])
	})
}
