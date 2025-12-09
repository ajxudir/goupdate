package outdated

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/goupdate/pkg/config"
	pkgerrors "github.com/user/goupdate/pkg/errors"
	"github.com/user/goupdate/pkg/formats"
)

// TestResolveOutdatedScope tests the behavior of resolveOutdatedScope.
//
// It verifies:
//   - Uses package source directory when available
//   - Falls back to config WorkingDir
//   - Falls back to scopeDir parameter
//   - Falls back to current directory
func TestResolveOutdatedScope(t *testing.T) {
	dir := resolveOutdatedScope(formats.Package{Source: filepath.Join("a", "b", "file.txt")}, &config.Config{}, "")
	assert.Equal(t, filepath.Join("a", "b"), dir)

	dir = resolveOutdatedScope(formats.Package{}, &config.Config{WorkingDir: "root"}, "")
	assert.Equal(t, "root", dir)

	dir = resolveOutdatedScope(formats.Package{}, &config.Config{}, "base")
	assert.Equal(t, "base", dir)

	dir = resolveOutdatedScope(formats.Package{}, &config.Config{}, "")
	assert.Equal(t, ".", dir)
}

// TestApplyDefaultExclusionsUsesConfig tests the behavior of applyDefaultExclusions with configuration.
//
// It verifies:
//   - Default exclusions are applied from config
//   - Existing exclusions are preserved
//   - Empty exclusions are not modified
func TestApplyDefaultExclusionsUsesConfig(t *testing.T) {
	cfg := &config.OutdatedCfg{}
	applyDefaultExclusions(cfg, []string{"^stable$"})
	assert.Equal(t, []string{"^stable$"}, cfg.ExcludeVersionPatterns)

	cfg = &config.OutdatedCfg{ExcludeVersionPatterns: []string{"^prod$"}}
	applyDefaultExclusions(cfg, []string{"^prod$", "^stable$"})
	assert.ElementsMatch(t, []string{"^prod$", "^stable$"}, cfg.ExcludeVersionPatterns)

	cfg = &config.OutdatedCfg{ExcludeVersionPatterns: []string{}}
	applyDefaultExclusions(cfg, []string{"^stable$"})
	assert.Empty(t, cfg.ExcludeVersionPatterns)
}

// TestResolveDefaultExclusionsPrefersRuleLevel tests the behavior of resolveDefaultExclusions with rule-level settings.
//
// It verifies:
//   - Rule-level exclusions take precedence over global exclusions
func TestResolveDefaultExclusionsPrefersRuleLevel(t *testing.T) {
	result := resolveDefaultExclusions(&config.Config{ExcludeVersions: []string{"global"}}, config.PackageManagerCfg{ExcludeVersions: []string{"rule"}})
	assert.Equal(t, []string{"rule"}, result)
}

// TestResolveDefaultExclusionsFallsBackToTopLevel tests the behavior of resolveDefaultExclusions fallback.
//
// It verifies:
//   - Falls back to global exclusions when rule-level is not set
func TestResolveDefaultExclusionsFallsBackToTopLevel(t *testing.T) {
	result := resolveDefaultExclusions(&config.Config{ExcludeVersions: []string{"global"}}, config.PackageManagerCfg{})
	assert.Equal(t, []string{"global"}, result)
}

// TestApplyVersionExclusions tests the behavior of applyVersionExclusions.
//
// It verifies:
//   - Excludes versions matching exact strings
//   - Excludes versions matching regex patterns
func TestApplyVersionExclusions(t *testing.T) {
	cfg := &config.OutdatedCfg{
		ExcludeVersions:        []string{"1.0.0-beta"},
		ExcludeVersionPatterns: []string{"(?i)alpha"},
	}

	versions := []string{"1.0.0", "1.0.0-beta", "2.0.0-alpha", "2.1.0"}
	filtered, err := applyVersionExclusions(versions, cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "2.1.0"}, filtered)
}

// TestApplyVersionExclusionsInvalidPattern tests the behavior of applyVersionExclusions with invalid regex.
//
// It verifies:
//   - Invalid regex pattern returns an error
func TestApplyVersionExclusionsInvalidPattern(t *testing.T) {
	cfg := &config.OutdatedCfg{
		ExcludeVersionPatterns: []string{"[invalid"},
	}

	_, err := applyVersionExclusions([]string{"1.0.0"}, cfg, nil)
	assert.Error(t, err)
}

// TestApplyVersionExclusionsEdgeCases tests edge cases for applyVersionExclusions.
//
// It verifies:
//   - Nil config returns versions unchanged
//   - Empty exclusions returns versions unchanged
//   - Skips empty and whitespace-only exclusion patterns
//   - Skips empty and whitespace-only versions
func TestApplyVersionExclusionsEdgeCases(t *testing.T) {
	t.Run("nil config returns versions unchanged", func(t *testing.T) {
		versions := []string{"1.0.0", "2.0.0"}
		result, err := applyVersionExclusions(versions, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, versions, result)
	})

	t.Run("empty exclusions returns versions unchanged", func(t *testing.T) {
		cfg := &config.OutdatedCfg{}
		versions := []string{"1.0.0", "2.0.0"}
		result, err := applyVersionExclusions(versions, cfg, nil)
		require.NoError(t, err)
		assert.Equal(t, versions, result)
	})

	t.Run("skips empty and whitespace-only exclusion patterns", func(t *testing.T) {
		cfg := &config.OutdatedCfg{
			ExcludeVersions:        []string{"", "  ", "1.0.0"},
			ExcludeVersionPatterns: []string{"", "  "},
		}
		versions := []string{"1.0.0", "2.0.0"}
		result, err := applyVersionExclusions(versions, cfg, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"2.0.0"}, result)
	})

	t.Run("skips empty and whitespace-only versions", func(t *testing.T) {
		cfg := &config.OutdatedCfg{
			ExcludeVersions: []string{"1.0.0"},
		}
		versions := []string{"", "  ", "1.0.0", "2.0.0"}
		result, err := applyVersionExclusions(versions, cfg, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"2.0.0"}, result)
	})

	t.Run("rejects unsafe regex pattern", func(t *testing.T) {
		cfg := &config.OutdatedCfg{
			ExcludeVersionPatterns: []string{"(a+)+"},
		}
		_, err := applyVersionExclusions([]string{"1.0.0"}, cfg, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsafe exclude_version_patterns entry")
	})

	t.Run("allows unsafe regex when security config permits", func(t *testing.T) {
		cfg := &config.OutdatedCfg{
			ExcludeVersionPatterns: []string{"(a+)+"},
		}
		secCfg := &config.SecurityCfg{
			AllowComplexRegex: true,
		}
		// With AllowComplexRegex=true, the pattern should compile (but won't match versions)
		result, err := applyVersionExclusions([]string{"1.0.0", "2.0.0"}, cfg, secCfg)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, result)
	})

	t.Run("respects custom max regex complexity", func(t *testing.T) {
		// Create a pattern longer than 10 chars but shorter than default 1000
		longPattern := "(?i)alpha-" + "x"
		cfg := &config.OutdatedCfg{
			ExcludeVersionPatterns: []string{longPattern},
		}
		secCfg := &config.SecurityCfg{
			MaxRegexComplexity: 5, // Very short limit
		}
		_, err := applyVersionExclusions([]string{"1.0.0"}, cfg, secCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pattern length")
	})
}

// TestIsUnsupported tests the behavior of pkgerrors.IsUnsupported.
//
// It verifies:
//   - Returns true for UnsupportedError
//   - Returns false for nil error
//   - Returns false for other errors
func TestIsUnsupported(t *testing.T) {
	err := &pkgerrors.UnsupportedError{Reason: "test"}
	assert.True(t, pkgerrors.IsUnsupported(err))
	assert.False(t, pkgerrors.IsUnsupported(nil))
	assert.False(t, pkgerrors.IsUnsupported(assert.AnError))
}

// TestUnsupportedErrorMessage tests the behavior of UnsupportedError.Error.
//
// It verifies:
//   - Error message matches the reason
func TestUnsupportedErrorMessage(t *testing.T) {
	err := &pkgerrors.UnsupportedError{Reason: "no outdated config"}
	assert.Equal(t, "no outdated config", err.Error())
}

// TestEnsureGoModFlag tests the behavior of ensureGoModFlag.
//
// It verifies:
//   - Adds -mod=mod flag for go commands without existing mod flag
//   - Skips adding flag when -mod already present
//   - Skips adding flag for non-go commands
func TestEnsureGoModFlag(t *testing.T) {
	args := ensureGoModFlag("go", []string{"list", "-m", "-versions"})
	assert.Contains(t, args, "-mod=mod")

	args = ensureGoModFlag("go", []string{"list", "-mod=readonly"})
	assert.NotContains(t, args, "-mod=mod")

	args = ensureGoModFlag("npm", []string{"view"})
	assert.NotContains(t, args, "-mod=mod")
}

// TestCloneOutdatedCfg tests the behavior of cloneOutdatedCfg.
//
// It verifies:
//   - Clone has same field values as original
//   - Clone is a deep copy (modifying clone doesn't affect original)
func TestCloneOutdatedCfg(t *testing.T) {
	original := &config.OutdatedCfg{
		Commands:               "npm view {{package}}",
		ExcludeVersions:        []string{"1.0.0"},
		ExcludeVersionPatterns: []string{"alpha"},
		Env:                    map[string]string{"KEY": "value"},
	}

	cloned := cloneOutdatedCfg(original)
	assert.Equal(t, original.Commands, cloned.Commands)
	assert.Equal(t, original.Env, cloned.Env)

	// Verify it's a deep copy
	cloned.Env["KEY"] = "modified"
	assert.NotEqual(t, original.Env["KEY"], cloned.Env["KEY"])
}

// TestExtractExitCode tests the behavior of ExtractExitCode.
//
// It verifies:
//   - Returns empty for nil error
//   - Returns empty for non-exit error
//   - Returns exit code for ExitError
func TestExtractExitCode(t *testing.T) {
	t.Run("nil error returns empty", func(t *testing.T) {
		assert.Equal(t, "", ExtractExitCode(nil))
	})

	t.Run("non-exit error returns empty", func(t *testing.T) {
		assert.Equal(t, "", ExtractExitCode(errors.New("some error")))
	})

	t.Run("exit error returns code", func(t *testing.T) {
		// Run a command that exits with code 1
		cmd := exec.Command("sh", "-c", "exit 1")
		err := cmd.Run()
		require.Error(t, err)
		assert.Equal(t, "1", ExtractExitCode(err))
	})

	t.Run("exit error with different code", func(t *testing.T) {
		// Run a command that exits with code 42
		cmd := exec.Command("sh", "-c", "exit 42")
		err := cmd.Run()
		require.Error(t, err)
		assert.Equal(t, "42", ExtractExitCode(err))
	})
}

// TestNormalizeOutdatedError tests the behavior of normalizeOutdatedError.
//
// It verifies:
//   - Returns nil for nil error
//   - Returns original error for non-dotnet commands
//   - Converts dotnet "No assets file" error to UnsupportedError
//   - Converts dotnet "Found more than one project" error to UnsupportedError
//   - Returns original error for other dotnet errors
func TestNormalizeOutdatedError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.Nil(t, normalizeOutdatedError(nil, "dotnet"))
	})

	t.Run("non-dotnet command returns original error", func(t *testing.T) {
		err := errors.New("some error")
		result := normalizeOutdatedError(err, "npm")
		assert.Equal(t, err, result)
	})

	t.Run("dotnet with No assets file becomes UnsupportedError", func(t *testing.T) {
		err := errors.New("No assets file was found for project")
		result := normalizeOutdatedError(err, "dotnet")
		assert.True(t, pkgerrors.IsUnsupported(result))
	})

	t.Run("dotnet with Found more than one project becomes UnsupportedError", func(t *testing.T) {
		err := errors.New("Found more than one project in directory")
		result := normalizeOutdatedError(err, "DOTNET")
		assert.True(t, pkgerrors.IsUnsupported(result))
	})

	t.Run("dotnet with other error returns original", func(t *testing.T) {
		err := errors.New("package not found")
		result := normalizeOutdatedError(err, "dotnet")
		assert.Equal(t, err, result)
		assert.False(t, pkgerrors.IsUnsupported(result))
	})
}

// TestResolveOutdatedCfg tests the behavior of resolveOutdatedCfg.
//
// It verifies:
//   - Missing rule returns error
//   - Nil outdated config returns unsupported error
//   - Basic config cloned correctly
//   - Package override versioning applied
//   - Package override exclude versions applied
//   - Package override exclude patterns applied
//   - Package override timeout applied
//   - NoTimeout flag clears timeout
func TestResolveOutdatedCfg(t *testing.T) {
	t.Run("missing rule returns error", func(t *testing.T) {
		pkg := formats.Package{Rule: "unknown"}
		cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{}}
		_, err := resolveOutdatedCfg(pkg, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule configuration missing")
	})

	t.Run("nil outdated config returns unsupported error", func(t *testing.T) {
		pkg := formats.Package{Rule: "npm"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Outdated: nil},
			},
		}
		_, err := resolveOutdatedCfg(pkg, cfg)
		assert.Error(t, err)
		assert.True(t, pkgerrors.IsUnsupported(err))
	})

	t.Run("basic config cloned correctly", func(t *testing.T) {
		pkg := formats.Package{Rule: "npm", Name: "lodash"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:       "npm view {{package}} versions --json",
						TimeoutSeconds: 30,
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		assert.Equal(t, "npm view {{package}} versions --json", result.Commands)
		assert.Equal(t, 30, result.TimeoutSeconds)
	})

	t.Run("package override versioning applied", func(t *testing.T) {
		versioningCfg := &config.VersioningCfg{Format: "numeric"}
		pkg := formats.Package{Rule: "npm", Name: "lodash"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands: "npm view {{package}} versions --json",
					},
					PackageOverrides: map[string]config.PackageOverrideCfg{
						"lodash": {
							Outdated: &config.OutdatedOverrideCfg{
								Versioning: versioningCfg,
							},
						},
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		assert.Equal(t, "numeric", result.Versioning.Format)
	})

	t.Run("package override exclude versions applied", func(t *testing.T) {
		pkg := formats.Package{Rule: "npm", Name: "axios"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:        "npm view {{package}} versions --json",
						ExcludeVersions: []string{"1.0.0"},
					},
					PackageOverrides: map[string]config.PackageOverrideCfg{
						"axios": {
							Outdated: &config.OutdatedOverrideCfg{
								ExcludeVersions: []string{"2.0.0-beta"},
							},
						},
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		assert.Equal(t, []string{"2.0.0-beta"}, result.ExcludeVersions)
	})

	t.Run("package override exclude patterns applied", func(t *testing.T) {
		pkg := formats.Package{Rule: "npm", Name: "react"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands: "npm view {{package}} versions --json",
					},
					PackageOverrides: map[string]config.PackageOverrideCfg{
						"react": {
							Outdated: &config.OutdatedOverrideCfg{
								ExcludeVersionPatterns: []string{"^alpha"},
							},
						},
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		// First pattern is from override, followed by default patterns
		assert.Contains(t, result.ExcludeVersionPatterns, "^alpha")
	})

	t.Run("package override timeout applied", func(t *testing.T) {
		timeout := 60
		pkg := formats.Package{Rule: "npm", Name: "webpack"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:       "npm view {{package}} versions --json",
						TimeoutSeconds: 30,
					},
					PackageOverrides: map[string]config.PackageOverrideCfg{
						"webpack": {
							Outdated: &config.OutdatedOverrideCfg{
								TimeoutSeconds: &timeout,
							},
						},
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		assert.Equal(t, 60, result.TimeoutSeconds)
	})

	t.Run("NoTimeout flag clears timeout", func(t *testing.T) {
		pkg := formats.Package{Rule: "npm", Name: "lodash"}
		cfg := &config.Config{
			NoTimeout: true,
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:       "npm view {{package}} versions --json",
						TimeoutSeconds: 30,
					},
				},
			},
		}
		result, err := resolveOutdatedCfg(pkg, cfg)
		require.NoError(t, err)
		assert.Equal(t, 0, result.TimeoutSeconds)
	})
}

// TestExecuteOutdatedCommand tests the behavior of executeOutdatedCommand.
//
// It verifies:
//   - Nil config returns error
//   - Empty commands returns error
//   - Whitespace only commands returns error
//   - Executes simple echo command
//   - Replaces package placeholder
func TestExecuteOutdatedCommand(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := executeOutdatedCommand(context.Background(), nil, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outdated configuration is required")
	})

	t.Run("empty commands returns error", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: ""}
		_, err := executeOutdatedCommand(context.Background(), cfg, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no commands configured")
	})

	t.Run("whitespace only commands returns error", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: "   \n\t  "}
		_, err := executeOutdatedCommand(context.Background(), cfg, "pkg", "1.0.0", "^", ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no commands configured")
	})

	t.Run("executes simple echo command", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: "echo '[\"1.0.0\", \"2.0.0\"]'"}
		output, err := executeOutdatedCommand(context.Background(), cfg, "test-pkg", "1.0.0", "^", ".")
		require.NoError(t, err)
		assert.Contains(t, string(output), "1.0.0")
	})

	t.Run("replaces package placeholder", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: "echo '{{package}}'"}
		output, err := executeOutdatedCommand(context.Background(), cfg, "my-package", "1.0.0", "^", ".")
		require.NoError(t, err)
		assert.Contains(t, string(output), "my-package")
	})
}

// TestRunOutdatedCommand tests the behavior of runOutdatedCommand.
//
// It verifies:
//   - Empty commands returns error
//   - Whitespace only commands returns error
//   - Successful command returns output
//   - Failed command returns normalized error
func TestRunOutdatedCommand(t *testing.T) {
	// Save original function
	originalFunc := execOutdatedFunc
	defer func() { execOutdatedFunc = originalFunc }()

	t.Run("empty commands returns error", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: ""}
		pkg := formats.Package{Name: "test"}
		_, err := runOutdatedCommand(context.Background(), cfg, pkg, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outdated command is empty")
	})

	t.Run("whitespace only commands returns error", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Commands: "   "}
		pkg := formats.Package{Name: "test"}
		_, err := runOutdatedCommand(context.Background(), cfg, pkg, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outdated command is empty")
	})

	t.Run("successful command returns output", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return []byte(`["1.0.0", "2.0.0"]`), nil
		}
		cfg := &config.OutdatedCfg{Commands: "npm view {{package}} versions"}
		pkg := formats.Package{Name: "test", Version: "1.0.0"}
		output, err := runOutdatedCommand(context.Background(), cfg, pkg, ".")
		require.NoError(t, err)
		assert.Equal(t, `["1.0.0", "2.0.0"]`, string(output))
	})

	t.Run("failed command returns normalized error", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return nil, errors.New("command failed")
		}
		cfg := &config.OutdatedCfg{Commands: "npm view {{package}} versions"}
		pkg := formats.Package{Name: "test", Version: "1.0.0"}
		_, err := runOutdatedCommand(context.Background(), cfg, pkg, ".")
		assert.Error(t, err)
	})
}

// TestListNewerVersions tests the behavior of ListNewerVersions.
//
// It verifies:
//   - Nil config returns error
//   - Missing rule returns error
//   - Successful version listing
//   - Excludes versions matching patterns
//   - Invalid versioning strategy returns error
func TestListNewerVersions(t *testing.T) {
	// Save original function
	originalFunc := execOutdatedFunc
	defer func() { execOutdatedFunc = originalFunc }()

	t.Run("nil config returns error", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Rule: "npm"}
		_, err := ListNewerVersions(context.Background(), pkg, nil, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration is required")
	})

	t.Run("missing rule returns error", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Rule: "unknown"}
		cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{}}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rule configuration missing")
	})

	t.Run("successful version listing", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return []byte(`["1.0.0", "1.1.0", "2.0.0"]`), nil
		}

		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0", InstalledVersion: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands: "npm view {{package}} versions --json",
					},
				},
			},
		}
		versions, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		require.NoError(t, err)
		assert.Contains(t, versions, "2.0.0")
		assert.Contains(t, versions, "1.1.0")
	})

	t.Run("excludes versions matching patterns", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return []byte(`["1.0.0", "1.1.0-alpha", "2.0.0"]`), nil
		}

		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0", InstalledVersion: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:               "npm view {{package}} versions --json",
						ExcludeVersionPatterns: []string{"alpha"},
					},
				},
			},
		}
		versions, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		require.NoError(t, err)
		assert.NotContains(t, versions, "1.1.0-alpha")
		assert.Contains(t, versions, "2.0.0")
	})

	t.Run("invalid versioning strategy returns error", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:   "npm view {{package}} versions --json",
						Versioning: &config.VersioningCfg{Format: "invalid"},
					},
				},
			},
		}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
	})
}

// TestCloneStringSlice tests the behavior of cloneStringSlice.
//
// It verifies:
//   - Nil returns nil
//   - Empty slice returns empty slice
//   - Creates deep copy
func TestCloneStringSlice(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := cloneStringSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		result := cloneStringSlice([]string{})
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("creates deep copy", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		cloned := cloneStringSlice(original)
		assert.Equal(t, original, cloned)

		// Modify original - cloned should not change
		original[0] = "modified"
		assert.Equal(t, "a", cloned[0])
	})
}

// TestCloneOutdatedCfgEdgeCases tests edge cases for cloneOutdatedCfg.
//
// It verifies:
//   - Nil returns nil
//   - Clones versioning
//   - Clones env map
//   - Clones extraction
func TestCloneOutdatedCfgEdgeCases(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := cloneOutdatedCfg(nil)
		assert.Nil(t, result)
	})

	t.Run("clones versioning", func(t *testing.T) {
		original := &config.OutdatedCfg{
			Versioning: &config.VersioningCfg{Format: "semver"},
		}
		cloned := cloneOutdatedCfg(original)
		assert.Equal(t, "semver", cloned.Versioning.Format)
	})

	t.Run("clones env map", func(t *testing.T) {
		original := &config.OutdatedCfg{
			Env: map[string]string{"KEY": "value"},
		}
		cloned := cloneOutdatedCfg(original)
		assert.Equal(t, "value", cloned.Env["KEY"])
		// Verify it's a deep copy
		cloned.Env["KEY"] = "modified"
		assert.Equal(t, "value", original.Env["KEY"])
	})

	t.Run("clones extraction", func(t *testing.T) {
		original := &config.OutdatedCfg{
			Extraction: &config.OutdatedExtractionCfg{Pattern: "test"},
		}
		cloned := cloneOutdatedCfg(original)
		assert.Equal(t, "test", cloned.Extraction.Pattern)
		// Verify it's a deep copy
		cloned.Extraction.Pattern = "modified"
		assert.Equal(t, "test", original.Extraction.Pattern)
	})
}

// TestListNewerVersionsErrorPaths tests error paths for ListNewerVersions.
//
// It verifies:
//   - Invalid regex in versioning config returns error
//   - Dotnet command with normalized error
//   - Parse error returns error
//   - Invalid exclude version pattern returns error
func TestListNewerVersionsErrorPaths(t *testing.T) {
	originalFunc := execOutdatedFunc
	defer func() { execOutdatedFunc = originalFunc }()

	t.Run("invalid regex in versioning config returns error", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:   "npm view",
						Versioning: &config.VersioningCfg{Regex: "(invalid regex"},
					},
				},
			},
		}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid version regex")
	})

	t.Run("dotnet command with normalized error", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return nil, errors.New("No assets file was found for project")
		}

		pkg := formats.Package{Name: "Newtonsoft.Json", Rule: "dotnet", Version: "13.0.1"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"dotnet": {
					Outdated: &config.OutdatedCfg{
						Commands: "dotnet list package --outdated",
					},
				},
			},
		}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
		assert.True(t, pkgerrors.IsUnsupported(err))
	})

	t.Run("parse error returns error", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return []byte("not valid json at all {{{"), nil
		}

		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands: "npm view {{package}} versions --json",
						Format:   "json",
					},
				},
			},
		}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
	})

	t.Run("invalid exclude version pattern returns error", func(t *testing.T) {
		execOutdatedFunc = func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
			return []byte(`["1.0.0", "2.0.0"]`), nil
		}

		pkg := formats.Package{Name: "test", Rule: "npm", Version: "1.0.0"}
		cfg := &config.Config{
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Outdated: &config.OutdatedCfg{
						Commands:               "npm view",
						Format:                 "json",
						ExcludeVersionPatterns: []string{"[invalid-regex"},
					},
				},
			},
		}
		_, err := ListNewerVersions(context.Background(), pkg, cfg, ".")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid exclude_version_patterns")
	})
}
