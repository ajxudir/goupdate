package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMergeVersionPatterns tests the behavior of mergeVersionPatterns.
//
// It verifies:
//   - Nil override returns base patterns
//   - Empty override clears patterns
//   - Override replaces base completely
func TestMergeVersionPatterns(t *testing.T) {
	base := []string{"a", "b"}

	t.Run("nil override", func(t *testing.T) {
		merged := mergeVersionPatterns(base, nil)
		assert.Equal(t, base, merged)
	})

	t.Run("empty override clears", func(t *testing.T) {
		merged := mergeVersionPatterns(base, []string{})
		assert.Empty(t, merged)
	})

	t.Run("override replaces base completely", func(t *testing.T) {
		merged := mergeVersionPatterns(base, []string{"b", "c"})
		assert.Equal(t, []string{"b", "c"}, merged)
	})
}

// TestMergeStringLists tests the behavior of mergeStringLists.
//
// It verifies:
//   - Nil override returns base list
//   - Empty override clears list
//   - Override replaces base completely
func TestMergeStringLists(t *testing.T) {
	base := []string{"a", "b"}

	t.Run("nil override returns base", func(t *testing.T) {
		merged := mergeStringLists(base, nil)
		assert.Equal(t, base, merged)
	})

	t.Run("empty override clears list", func(t *testing.T) {
		merged := mergeStringLists(base, []string{})
		assert.Empty(t, merged)
	})

	t.Run("override replaces base completely", func(t *testing.T) {
		merged := mergeStringLists(base, []string{"b", "c"})
		assert.Equal(t, []string{"b", "c"}, merged)
	})
}

// TestMergeConfigs tests the behavior of mergeConfigs.
//
// It verifies:
//   - Working directory is properly merged
//   - Rules are merged correctly
//   - Exclude versions are merged
//   - Groups are merged correctly
//   - Incremental lists are merged
func TestMergeConfigs(t *testing.T) {
	base := &Config{
		WorkingDir: "/base",
		Rules: map[string]PackageManagerCfg{
			"npm": {Manager: "js"},
		},
	}

	overlay := &Config{
		Rules: map[string]PackageManagerCfg{
			"custom": {Manager: "custom"},
		},
	}

	result := mergeConfigs(base, overlay)
	assert.Equal(t, "/base", result.WorkingDir)
	assert.Contains(t, result.Rules, "npm")
	assert.Contains(t, result.Rules, "custom")
}

// TestMergeConfigsNilCustom tests the behavior of mergeConfigs with nil custom config.
//
// It verifies:
//   - Nil custom config returns base config
func TestMergeConfigsNilCustom(t *testing.T) {
	base := &Config{Rules: map[string]PackageManagerCfg{"npm": {Manager: "js"}}}

	result := mergeConfigs(base, nil)
	assert.Equal(t, base, result)
}

// TestMergeRules tests the behavior of mergeRules.
//
// It verifies:
//   - Rule properties are merged correctly
//   - Nil rules return base rules
//   - Disabled rules are preserved
func TestMergeRules(t *testing.T) {
	base := PackageManagerCfg{
		Manager:           "npm",
		Include:           []string{"base"},
		Exclude:           []string{"ignore"},
		Format:            "json",
		Fields:            map[string]string{"a": "prod"},
		Ignore:            []string{"skip"},
		ConstraintMapping: map[string]string{"~=": "~"},
		LatestMapping:     &LatestMappingCfg{Default: map[string]string{"latest": "*"}},
		PackageOverrides:  map[string]PackageOverrideCfg{"pkg": {Ignore: true}},
		Extraction:        &ExtractionCfg{Pattern: "base"},
		LockFiles:         []LockFileCfg{{Files: []string{"base.lock"}}},
	}

	custom := PackageManagerCfg{
		Manager:           "yarn",
		Include:           []string{"custom"},
		Exclude:           []string{"custom-ex"},
		Format:            "raw",
		Fields:            map[string]string{"b": "dev"},
		Ignore:            []string{"custom-ignore"},
		ConstraintMapping: map[string]string{"==": "="},
		LatestMapping:     &LatestMappingCfg{Packages: map[string]map[string]string{"pkg": {"rolling": "*"}}},
		PackageOverrides:  map[string]PackageOverrideCfg{"pkg": {Version: "1.0.0"}},
		Extraction:        &ExtractionCfg{Pattern: "custom"},
		Outdated:          &OutdatedCfg{Commands: "echo {{package}}", TimeoutSeconds: 30},
		Update:            &UpdateCfg{Commands: "cmd {{package}}", TimeoutSeconds: 45},
		LockFiles:         []LockFileCfg{{Files: []string{"custom.lock"}}},
		Metadata:          map[string]interface{}{"k": "v"},
	}

	merged := mergeRules(base, custom)

	assert.Equal(t, "yarn", merged.Manager)
	// List fields are now overwritten by custom (not merged)
	assert.Equal(t, []string{"custom"}, merged.Include)
	assert.Equal(t, []string{"custom-ex"}, merged.Exclude)
	assert.Equal(t, "raw", merged.Format)
	assert.Equal(t, map[string]string{"b": "dev"}, merged.Fields)
	assert.Equal(t, []string{"custom-ignore"}, merged.Ignore)
	assert.Equal(t, map[string]string{"==": "="}, merged.ConstraintMapping)
	assert.Equal(t, &LatestMappingCfg{Default: map[string]string{"latest": "*"}, Packages: map[string]map[string]string{"pkg": {"rolling": "*"}}}, merged.LatestMapping)
	assert.Equal(t, map[string]PackageOverrideCfg{"pkg": {Version: "1.0.0"}}, merged.PackageOverrides)
	assert.Equal(t, &ExtractionCfg{Pattern: "custom"}, merged.Extraction)
	assert.Equal(t, &OutdatedCfg{Commands: "echo {{package}}", TimeoutSeconds: 30}, merged.Outdated)
	assert.Equal(t, &UpdateCfg{Commands: "cmd {{package}}", TimeoutSeconds: 45}, merged.Update)
	// LockFiles are merged by first file pattern
	assert.Equal(t, []LockFileCfg{{Files: []string{"base.lock"}}, {Files: []string{"custom.lock"}}}, merged.LockFiles)
	assert.Equal(t, map[string]interface{}{"k": "v"}, merged.Metadata)
}

// TestConfigMerging tests the behavior of config merging scenarios.
//
// It verifies:
//   - Base and custom configs merge correctly
//   - String fields are merged
//   - Lists are merged
//   - Maps are merged
func TestConfigMerging(t *testing.T) {
	t.Run("merge preserves base when overlay empty", func(t *testing.T) {
		base := Config{
			Rules: map[string]PackageManagerCfg{
				"test": {Manager: "js"},
			},
		}
		overlay := Config{}

		mergeConfigs(&base, &overlay)

		assert.Equal(t, "js", base.Rules["test"].Manager)
	})

	t.Run("overlay overwrites base lists", func(t *testing.T) {
		base := Config{
			Rules: map[string]PackageManagerCfg{
				"test": {Manager: "js", Include: []string{"*.json"}},
			},
		}
		overlay := Config{
			Rules: map[string]PackageManagerCfg{
				"test": {Include: []string{"*.txt"}},
			},
		}

		result := mergeConfigs(&base, &overlay)

		// Overlay's Include completely replaces base (overwrite, not merge)
		assert.Equal(t, []string{"*.txt"}, result.Rules["test"].Include)
	})

	t.Run("nil custom returns base", func(t *testing.T) {
		base := &Config{WorkingDir: "base"}

		result := mergeConfigs(base, nil)

		assert.Same(t, base, result)
	})

	t.Run("merges groups and global fields", func(t *testing.T) {
		base := Config{
			Rules: map[string]PackageManagerCfg{
				"node": {
					Groups: map[string]GroupCfg{
						"core":  {Packages: []string{"left"}},
						"extra": {Packages: []string{"base"}},
					},
				},
			},
			Groups: map[string]GroupCfg{
				"shared": {Packages: []string{"root"}},
			},
			ExcludeVersions: []string{"1"},
			Incremental:     []string{"pkg-a"},
		}

		overlay := Config{
			Rules: map[string]PackageManagerCfg{
				"node": {
					Groups: map[string]GroupCfg{
						"core": {Packages: []string{"right"}},
						"new":  {Packages: []string{"added"}},
					},
				},
			},
			Groups: map[string]GroupCfg{
				"shared": {Packages: []string{"override"}},
			},
			ExcludeVersions: []string{"1", "3"},
			Incremental:     []string{"pkg-a", "pkg-c"},
		}

		result := mergeConfigs(&base, &overlay)

		assert.Equal(t, []string{"right"}, result.Rules["node"].Groups["core"].Packages)
		assert.Equal(t, []string{"added"}, result.Rules["node"].Groups["new"].Packages)
		assert.Equal(t, []string{"override"}, result.Groups["shared"].Packages)
		// List fields are overwritten, not merged
		assert.Equal(t, []string{"1", "3"}, result.ExcludeVersions)
		assert.Equal(t, []string{"pkg-a", "pkg-c"}, result.Incremental)
	})

	t.Run("adds new rules and groups", func(t *testing.T) {
		base := Config{
			Rules: map[string]PackageManagerCfg{
				"base": {Manager: "npm"},
			},
			Groups: map[string]GroupCfg{
				"existing": {Packages: []string{"a"}},
			},
		}

		overlay := Config{
			Rules: map[string]PackageManagerCfg{
				"base": {Manager: "pnpm"},
				"new":  {Manager: "python"},
			},
			Groups: map[string]GroupCfg{
				"existing": {Packages: []string{"b"}},
				"extra":    {Packages: []string{"c"}},
			},
		}

		result := mergeConfigs(&base, &overlay)

		assert.Equal(t, "pnpm", result.Rules["base"].Manager)
		assert.Equal(t, "python", result.Rules["new"].Manager)
		assert.Equal(t, []string{"b"}, result.Groups["existing"].Packages)
		assert.Equal(t, []string{"c"}, result.Groups["extra"].Packages)
	})
}

// TestMergeGroupMaps tests the behavior of mergeGroupMaps.
//
// It verifies:
//   - Group maps are merged correctly
//   - Nil maps are handled
//   - Group packages are combined
func TestMergeGroupMaps(t *testing.T) {
	t.Run("nil maps return nil", func(t *testing.T) {
		assert.Nil(t, mergeGroupMaps(nil, nil))
	})

	t.Run("merges override with base", func(t *testing.T) {
		base := map[string]GroupCfg{
			"existing": {Packages: []string{"a"}},
		}
		override := map[string]GroupCfg{
			"existing": {Packages: []string{"b"}},
			"new":      {Packages: []string{"c"}},
		}

		result := mergeGroupMaps(base, override)

		assert.Equal(t, []string{"b"}, result["existing"].Packages)
		assert.Equal(t, []string{"c"}, result["new"].Packages)
	})
}

// TestMergeStringListsOverride tests the behavior of mergeStringLists with override scenarios.
//
// It verifies:
//   - Non-empty override completely replaces base
func TestMergeStringListsOverride(t *testing.T) {
	base := []string{"a", "b"}
	override := []string{"b", "c"}

	result := mergeStringLists(base, override)

	// Override replaces base completely (not merged)
	assert.Equal(t, []string{"b", "c"}, result)
}

// TestMergeRulesOverride tests the behavior of mergeRules with override scenarios.
//
// It verifies:
//   - Override fields replace base fields
//   - String fields are overridden
//   - List fields are overridden
func TestMergeRulesOverride(t *testing.T) {
	base := PackageManagerCfg{
		Manager:           "npm",
		Include:           []string{"base"},
		Exclude:           []string{"base-ex"},
		Groups:            map[string]GroupCfg{"shared": {Packages: []string{"left"}}},
		Format:            "json",
		Fields:            map[string]string{"base": "field"},
		Ignore:            []string{"ignore-base"},
		ExcludeVersions:   []string{"1"},
		ConstraintMapping: map[string]string{"^": "base"},
		LatestMapping:     &LatestMappingCfg{Default: map[string]string{"base": "1.0.0"}},
		PackageOverrides:  map[string]PackageOverrideCfg{"pkg": {Version: "1.0.0"}},
		Extraction:        &ExtractionCfg{Pattern: "base"},
		Outdated:          &OutdatedCfg{Commands: "base {{package}}"},
		Update:            &UpdateCfg{Commands: "base {{package}}"},
		LockFiles:         []LockFileCfg{{Files: []string{"base.lock"}, Format: "json"}},
		Metadata:          map[string]interface{}{"base": true},
		Incremental:       []string{"pkg-a"},
	}

	custom := PackageManagerCfg{
		Manager:           "pnpm",
		Include:           []string{"custom"},
		Exclude:           []string{"custom-ex"},
		Groups:            map[string]GroupCfg{"shared": {Packages: []string{"right"}}, "new": {Packages: []string{"added"}}},
		Format:            "yaml",
		Fields:            map[string]string{"base": "override"},
		Ignore:            []string{"ignore-custom"},
		ExcludeVersions:   []string{"2"},
		ConstraintMapping: map[string]string{"~": "override"},
		LatestMapping:     &LatestMappingCfg{Default: map[string]string{"custom": "2.0.0"}},
		PackageOverrides:  map[string]PackageOverrideCfg{"pkg": {Version: "2.0.0", Ignore: true}},
		Extraction:        &ExtractionCfg{Pattern: "custom"},
		Outdated:          &OutdatedCfg{Commands: "custom {{package}}"},
		Update:            &UpdateCfg{Commands: "custom {{package}}"},
		LockFiles:         []LockFileCfg{{Files: []string{"custom.lock"}, Format: "yaml"}},
		Metadata:          map[string]interface{}{"custom": "meta"},
		Incremental:       []string{"pkg-b"},
	}

	result := mergeRules(base, custom)

	assert.Equal(t, "pnpm", result.Manager)
	// List fields are now overwritten by custom (not merged)
	assert.Equal(t, []string{"custom"}, result.Include)
	assert.Equal(t, []string{"custom-ex"}, result.Exclude)
	assert.Equal(t, []string{"right"}, result.Groups["shared"].Packages)
	assert.Equal(t, []string{"added"}, result.Groups["new"].Packages)
	assert.Equal(t, "yaml", result.Format)
	assert.Equal(t, map[string]string{"base": "override"}, result.Fields)
	assert.Equal(t, []string{"ignore-custom"}, result.Ignore)
	assert.Equal(t, []string{"2"}, result.ExcludeVersions)
	assert.Equal(t, map[string]string{"~": "override"}, result.ConstraintMapping)
	assert.Equal(t, map[string]string{"base": "1.0.0", "custom": "2.0.0"}, result.LatestMapping.Default)
	assert.Equal(t, map[string]PackageOverrideCfg{"pkg": {Version: "2.0.0", Ignore: true}}, result.PackageOverrides)
	assert.Equal(t, "custom", result.Extraction.Pattern)
	assert.Equal(t, "custom {{package}}", result.Outdated.Commands)
	assert.Equal(t, "custom {{package}}", result.Update.Commands)
	// LockFiles are merged by first file pattern
	assert.Equal(t, []LockFileCfg{{Files: []string{"base.lock"}, Format: "json"}, {Files: []string{"custom.lock"}, Format: "yaml"}}, result.LockFiles)
	assert.Equal(t, map[string]interface{}{"custom": "meta"}, result.Metadata)
	assert.Equal(t, []string{"pkg-b"}, result.Incremental)
}

// TestRuleGroupMergePrefersOverride tests the behavior of rule group merging with override preference.
//
// It verifies:
//   - Override group packages replace base group packages
func TestRuleGroupMergePrefersOverride(t *testing.T) {
	base := PackageManagerCfg{Groups: map[string]GroupCfg{"shared": {Packages: []string{"base"}}}}
	override := PackageManagerCfg{Groups: map[string]GroupCfg{"shared": {Packages: []string{"override"}}, "extra": {Packages: []string{"new"}}}}

	merged := mergeRules(base, override)

	assert.Contains(t, merged.Groups, "shared")
	assert.Len(t, merged.Groups["shared"].Packages, 1)
	assert.Equal(t, "override", merged.Groups["shared"].Packages[0])

	assert.Contains(t, merged.Groups, "extra")
	assert.Len(t, merged.Groups["extra"].Packages, 1)
	assert.Equal(t, "new", merged.Groups["extra"].Packages[0])
}

// TestMergeConfigsSystemTests tests the behavior of merging system tests configuration.
//
// It verifies:
//   - Override system tests replace base system tests
//   - Nil override preserves base system tests
//   - Nil base accepts override system tests
func TestMergeConfigsSystemTests(t *testing.T) {
	t.Run("preserves base system_tests when custom is nil", func(t *testing.T) {
		runPreflight := true
		base := &Config{
			Rules: map[string]PackageManagerCfg{},
			SystemTests: &SystemTestsCfg{
				RunPreflight: &runPreflight,
				RunMode:      "after_all",
				Tests: []SystemTestCfg{
					{Name: "base-test", Commands: "echo base"},
				},
			},
		}
		custom := &Config{
			Rules: map[string]PackageManagerCfg{},
		}

		result := mergeConfigs(base, custom)

		assert.NotNil(t, result.SystemTests)
		assert.Equal(t, "after_all", result.SystemTests.RunMode)
		assert.Len(t, result.SystemTests.Tests, 1)
		assert.Equal(t, "base-test", result.SystemTests.Tests[0].Name)
	})

	t.Run("custom system_tests merges tests by name", func(t *testing.T) {
		runPreflightBase := true
		runPreflightCustom := false
		base := &Config{
			Rules: map[string]PackageManagerCfg{},
			SystemTests: &SystemTestsCfg{
				RunPreflight: &runPreflightBase,
				RunMode:      "after_all",
				Tests: []SystemTestCfg{
					{Name: "base-test", Commands: "echo base"},
					{Name: "shared-test", Commands: "echo shared-base"},
				},
			},
		}
		custom := &Config{
			Rules: map[string]PackageManagerCfg{},
			SystemTests: &SystemTestsCfg{
				RunPreflight: &runPreflightCustom,
				RunMode:      "after_each",
				Tests: []SystemTestCfg{
					{Name: "shared-test", Commands: "echo shared-custom"},
					{Name: "custom-test", Commands: "echo custom"},
				},
			},
		}

		result := mergeConfigs(base, custom)

		assert.NotNil(t, result.SystemTests)
		assert.Equal(t, "after_each", result.SystemTests.RunMode)
		assert.False(t, *result.SystemTests.RunPreflight)
		// Tests are merged by name: base-test kept, shared-test overridden, custom-test added
		assert.Len(t, result.SystemTests.Tests, 3)
		assert.Equal(t, "base-test", result.SystemTests.Tests[0].Name)
		assert.Equal(t, "echo base", result.SystemTests.Tests[0].Commands)
		assert.Equal(t, "shared-test", result.SystemTests.Tests[1].Name)
		assert.Equal(t, "echo shared-custom", result.SystemTests.Tests[1].Commands)
		assert.Equal(t, "custom-test", result.SystemTests.Tests[2].Name)
		assert.Equal(t, "echo custom", result.SystemTests.Tests[2].Commands)
	})

	t.Run("nil base system_tests with custom system_tests", func(t *testing.T) {
		runPreflight := true
		base := &Config{
			Rules: map[string]PackageManagerCfg{},
		}
		custom := &Config{
			Rules: map[string]PackageManagerCfg{},
			SystemTests: &SystemTestsCfg{
				RunPreflight: &runPreflight,
				RunMode:      "after_each",
				Tests: []SystemTestCfg{
					{Name: "new-test", Commands: "echo new"},
				},
			},
		}

		result := mergeConfigs(base, custom)

		assert.NotNil(t, result.SystemTests)
		assert.Equal(t, "after_each", result.SystemTests.RunMode)
		assert.Len(t, result.SystemTests.Tests, 1)
		assert.Equal(t, "new-test", result.SystemTests.Tests[0].Name)
	})

	t.Run("both nil system_tests", func(t *testing.T) {
		base := &Config{
			Rules: map[string]PackageManagerCfg{},
		}
		custom := &Config{
			Rules: map[string]PackageManagerCfg{},
		}

		result := mergeConfigs(base, custom)

		assert.Nil(t, result.SystemTests)
	})
}

// TestMergeLockFiles tests the behavior of mergeLockFiles.
//
// It verifies:
//   - Lock file configs are merged correctly
//   - Nil slices are handled
//   - Override replaces base lock files
func TestMergeLockFiles(t *testing.T) {
	t.Run("nil override returns base", func(t *testing.T) {
		base := []LockFileCfg{{Files: []string{"base.lock"}}}
		result := mergeLockFiles(base, nil)
		assert.Equal(t, base, result)
	})

	t.Run("empty override clears list", func(t *testing.T) {
		base := []LockFileCfg{{Files: []string{"base.lock"}}}
		result := mergeLockFiles(base, []LockFileCfg{})
		assert.Empty(t, result)
	})

	t.Run("merges distinct lock files", func(t *testing.T) {
		base := []LockFileCfg{{Files: []string{"base.lock"}, Format: "json"}}
		override := []LockFileCfg{{Files: []string{"custom.lock"}, Format: "yaml"}}
		result := mergeLockFiles(base, override)
		assert.Len(t, result, 2)
		assert.Equal(t, "base.lock", result[0].Files[0])
		assert.Equal(t, "custom.lock", result[1].Files[0])
	})

	t.Run("override replaces by first file pattern", func(t *testing.T) {
		base := []LockFileCfg{{Files: []string{"package-lock.json"}, Format: "json"}}
		override := []LockFileCfg{{Files: []string{"package-lock.json"}, Format: "yaml"}}
		result := mergeLockFiles(base, override)
		assert.Len(t, result, 1)
		assert.Equal(t, "yaml", result[0].Format)
	})
}

// TestMergeListsClearWithEmptySlice tests the behavior of list merging with empty slices.
//
// It verifies:
//   - Empty slice override clears the list
func TestMergeListsClearWithEmptySlice(t *testing.T) {
	t.Run("empty Include clears base", func(t *testing.T) {
		base := PackageManagerCfg{Include: []string{"base"}}
		custom := PackageManagerCfg{Include: []string{}}
		result := mergeRules(base, custom)
		assert.Empty(t, result.Include)
	})

	t.Run("empty Exclude clears base", func(t *testing.T) {
		base := PackageManagerCfg{Exclude: []string{"base"}}
		custom := PackageManagerCfg{Exclude: []string{}}
		result := mergeRules(base, custom)
		assert.Empty(t, result.Exclude)
	})

	t.Run("empty Ignore clears base", func(t *testing.T) {
		base := PackageManagerCfg{Ignore: []string{"base"}}
		custom := PackageManagerCfg{Ignore: []string{}}
		result := mergeRules(base, custom)
		assert.Empty(t, result.Ignore)
	})

	t.Run("nil Include preserves base", func(t *testing.T) {
		base := PackageManagerCfg{Include: []string{"base"}}
		custom := PackageManagerCfg{}
		result := mergeRules(base, custom)
		assert.Equal(t, []string{"base"}, result.Include)
	})
}

// TestMergeRulesEnabled tests the behavior of merging enabled field in rules.
//
// It verifies:
//   - Enabled field is properly merged from base and override
//   - Nil enabled preserves base enabled value
//   - Explicit enabled value overrides base
func TestMergeRulesEnabled(t *testing.T) {
	t.Run("enabled false overrides base", func(t *testing.T) {
		base := PackageManagerCfg{
			Manager: "js",
			Include: []string{"**/package.json"},
		}
		enabled := false
		custom := PackageManagerCfg{
			Enabled: &enabled,
		}

		result := mergeRules(base, custom)

		assert.NotNil(t, result.Enabled)
		assert.False(t, *result.Enabled)
		// Other fields should be preserved from base
		assert.Equal(t, "js", result.Manager)
		assert.Equal(t, []string{"**/package.json"}, result.Include)
	})

	t.Run("enabled true overrides base", func(t *testing.T) {
		enabledBase := false
		base := PackageManagerCfg{
			Enabled: &enabledBase,
			Manager: "js",
		}
		enabledCustom := true
		custom := PackageManagerCfg{
			Enabled: &enabledCustom,
		}

		result := mergeRules(base, custom)

		assert.NotNil(t, result.Enabled)
		assert.True(t, *result.Enabled)
	})

	t.Run("nil enabled preserves base", func(t *testing.T) {
		enabledBase := true
		base := PackageManagerCfg{
			Enabled: &enabledBase,
			Manager: "js",
		}
		custom := PackageManagerCfg{
			Manager: "pnpm",
		}

		result := mergeRules(base, custom)

		assert.NotNil(t, result.Enabled)
		assert.True(t, *result.Enabled)
		assert.Equal(t, "pnpm", result.Manager)
	})
}

// TestMergePackageSettings tests the behavior of mergePackageSettings.
//
// It verifies:
//   - Custom settings override base settings
//   - Base settings are preserved when not overridden
//   - Nil base results in copy of custom
func TestMergePackageSettings(t *testing.T) {
	t.Run("custom overrides base", func(t *testing.T) {
		base := map[string]PackageSettings{
			"pkg-a": {WithAllDependencies: false},
			"pkg-b": {WithAllDependencies: true},
		}
		custom := map[string]PackageSettings{
			"pkg-a": {WithAllDependencies: true}, // override
			"pkg-c": {WithAllDependencies: true}, // new package
		}

		result := mergePackageSettings(base, custom)

		assert.Len(t, result, 3)
		assert.True(t, result["pkg-a"].WithAllDependencies, "custom should override base")
		assert.True(t, result["pkg-b"].WithAllDependencies, "base should be preserved")
		assert.True(t, result["pkg-c"].WithAllDependencies, "new package should be added")
	})

	t.Run("nil base returns copy of custom", func(t *testing.T) {
		custom := map[string]PackageSettings{
			"pkg-a": {WithAllDependencies: true},
		}

		result := mergePackageSettings(nil, custom)

		assert.Len(t, result, 1)
		assert.True(t, result["pkg-a"].WithAllDependencies)
	})
}

// TestMergeRulesPackageSettings tests that mergeRules handles Packages field.
//
// It verifies:
//   - Packages field is merged correctly from base and custom rules
func TestMergeRulesPackageSettings(t *testing.T) {
	base := PackageManagerCfg{
		Manager: "composer",
		Include: []string{"**/composer.json"},
		Packages: map[string]PackageSettings{
			"laravel/framework": {WithAllDependencies: true},
		},
	}
	custom := PackageManagerCfg{
		Packages: map[string]PackageSettings{
			"monolog/monolog": {WithAllDependencies: true},
		},
	}

	result := mergeRules(base, custom)

	assert.Len(t, result.Packages, 2)
	assert.True(t, result.Packages["laravel/framework"].WithAllDependencies)
	assert.True(t, result.Packages["monolog/monolog"].WithAllDependencies)
}
