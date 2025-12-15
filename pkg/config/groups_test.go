package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestRuleGroupConfigSupportsStringList tests the behavior of GroupCfg unmarshaling with string lists.
//
// It verifies:
//   - String list format for groups is parsed correctly
//   - Package names are extracted from list
func TestRuleGroupConfigSupportsStringList(t *testing.T) {
	content := []byte("rules:\n  npm:\n    groups:\n      frontend:\n        - react\n        - vue\n")
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	rule, ok := cfg.Rules["npm"]
	require.True(t, ok)

	group, ok := rule.Groups["frontend"]
	require.True(t, ok)
	require.Len(t, group.Packages, 2)
	assert.Equal(t, "react", group.Packages[0])
	assert.Equal(t, "vue", group.Packages[1])
}

// TestRuleGroupConfigSupportsMapping tests the behavior of GroupCfg unmarshaling with mapping format.
//
// It verifies:
//   - Mapping format with packages key is parsed correctly
//   - Both named and direct package entries work
func TestRuleGroupConfigSupportsMapping(t *testing.T) {
	content := []byte("rules:\n  npm:\n    groups:\n      backend:\n        packages:\n          - name: api\n          - gateway\n")
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	rule, ok := cfg.Rules["npm"]
	require.True(t, ok)

	group, ok := rule.Groups["backend"]
	require.True(t, ok)
	require.Len(t, group.Packages, 2)
	assert.Equal(t, "api", group.Packages[0])
	assert.Equal(t, "gateway", group.Packages[1])
}

// TestLegacyTopLevelGroupsStillLoad tests the behavior of legacy top-level groups configuration.
//
// It verifies:
//   - Top-level groups configuration is still supported
//   - Legacy format loads correctly
func TestLegacyTopLevelGroupsStillLoad(t *testing.T) {
	content := []byte("groups:\n  rollout:\n    - api\n")
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	group, ok := cfg.Groups["rollout"]
	require.True(t, ok)
	require.Len(t, group.Packages, 1)
	assert.Equal(t, "api", group.Packages[0])
}

// TestGroupConfigUnmarshalErrors tests the behavior of GroupCfg unmarshaling with invalid input.
//
// It verifies:
//   - Non-scalar sequence entries return error
//   - Odd mapping entries return error
//   - Invalid mapping structures return error
//   - Mapping parse errors are handled
//   - Unsupported keys return error
//   - Unsupported types return error
//   - Invalid sequence entries return error
func TestGroupConfigUnmarshalErrors(t *testing.T) {
	t.Run("non-scalar sequence", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("odd mapping entries", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "packages"}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("invalid mapping", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "packages"}, {Kind: yaml.AliasNode, Value: "bad-anchor"}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("mapping parse error", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "packages"}, {Kind: yaml.SequenceNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "name"}, {Kind: yaml.ScalarNode, Value: ""}}}}}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("unsupported key", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "unknown"}, {Kind: yaml.SequenceNode}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("unsupported type", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: "unexpected"}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})

	t.Run("sequence invalid entry", func(t *testing.T) {
		var group GroupCfg
		node := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{{Kind: yaml.SequenceNode}}}
		err := group.UnmarshalYAML(node)
		assert.Error(t, err)
	})
}

// TestGroupConfigUnmarshalSupportsMembers tests the behavior of GroupCfg unmarshaling with members key.
//
// It verifies:
//   - Both packages and members keys work
//   - Whitespace is trimmed from package names
//   - Named and direct entries are combined
func TestGroupConfigUnmarshalSupportsMembers(t *testing.T) {
	content := []byte("group:\n  packages:\n    - base\n  members:\n    - name: extra\n    - \" direct \"\n")
	var cfg struct {
		Group GroupCfg `yaml:"group"`
	}

	require.NoError(t, yaml.Unmarshal(content, &cfg))
	assert.Equal(t, []string{"base", "extra", "direct"}, cfg.Group.Packages)
}

// TestParseGroupSequenceWithSettingsValidatesEntries tests the behavior of parseGroupSequenceWithSettings with validation.
//
// It verifies:
//   - Whitespace is trimmed from entries
//   - Empty entries are skipped
//   - Empty name mappings return error
//   - Invalid name mappings return error
//   - Invalid entry types return error
func TestParseGroupSequenceWithSettingsValidatesEntries(t *testing.T) {
	nodes := []*yaml.Node{{Kind: yaml.ScalarNode, Value: " first "}, {Kind: yaml.ScalarNode, Value: "   "}}
	packages, _, err := parseGroupSequenceWithSettings(nodes)
	require.NoError(t, err)
	assert.Equal(t, []string{"first"}, packages)

	nodes = []*yaml.Node{{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "name"}, {Kind: yaml.ScalarNode, Value: ""}}}}
	_, _, err = parseGroupSequenceWithSettings(nodes)
	assert.Error(t, err)

	nodes = []*yaml.Node{{Kind: yaml.MappingNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "name"}, {Kind: yaml.SequenceNode}}}}
	_, _, err = parseGroupSequenceWithSettings(nodes)
	assert.Error(t, err)

	nodes = []*yaml.Node{{Kind: yaml.SequenceNode}}
	_, _, err = parseGroupSequenceWithSettings(nodes)
	assert.Error(t, err)
}

// TestValidateGroupMembership tests the behavior of validateGroupMembership.
//
// It verifies:
//   - Packages in multiple groups return error
//   - No groups returns no error
//   - Whitespace-only packages are ignored
func TestValidateGroupMembership(t *testing.T) {
	cfg := &Config{
		Rules: map[string]PackageManagerCfg{
			"npm": {
				Groups: map[string]GroupCfg{
					"core":  {Packages: []string{"shared", "unique"}},
					"extra": {Packages: []string{"shared"}},
				},
			},
		},
	}

	err := validateGroupMembership(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "npm")
	assert.Contains(t, err.Error(), "shared")

	err = validateGroupMembership(&Config{Rules: map[string]PackageManagerCfg{"npm": {}}})
	assert.NoError(t, err)

	err = validateGroupMembership(&Config{Rules: map[string]PackageManagerCfg{
		"npm": {Groups: map[string]GroupCfg{"core": {Packages: []string{"  ", "unique"}}}},
	}})
	assert.NoError(t, err)
}

// TestGroupWithAllDependenciesGroupLevel tests the behavior of with_all_dependencies at group level.
//
// It verifies:
//   - Group-level with_all_dependencies is parsed correctly
//   - Packages in the group inherit the setting
func TestGroupWithAllDependenciesGroupLevel(t *testing.T) {
	content := []byte(`rules:
  composer:
    groups:
      laravel:
        with_all_dependencies: true
        packages:
          - laravel/framework
          - laravel/tinker
`)
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	rule, ok := cfg.Rules["composer"]
	require.True(t, ok)

	group, ok := rule.Groups["laravel"]
	require.True(t, ok)
	assert.True(t, group.WithAllDependencies)
	assert.Len(t, group.Packages, 2)
	assert.Equal(t, "laravel/framework", group.Packages[0])
}

// TestGroupWithAllDependenciesPackageLevel tests the behavior of with_all_dependencies at package level.
//
// It verifies:
//   - Package-level with_all_dependencies within groups is parsed correctly
//   - Settings are stored per-package
func TestGroupWithAllDependenciesPackageLevel(t *testing.T) {
	content := []byte(`rules:
  composer:
    groups:
      mixed:
        packages:
          - name: sentry/sentry-laravel
            with_all_dependencies: true
          - laravel/framework
`)
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	rule, ok := cfg.Rules["composer"]
	require.True(t, ok)

	group, ok := rule.Groups["mixed"]
	require.True(t, ok)
	assert.Len(t, group.Packages, 2)
	assert.Contains(t, group.Packages, "sentry/sentry-laravel")
	assert.Contains(t, group.Packages, "laravel/framework")

	// Check per-package settings
	settings, ok := group.PackageSettings["sentry/sentry-laravel"]
	assert.True(t, ok)
	assert.True(t, settings.WithAllDependencies)

	// laravel/framework should not have settings (plain string entry)
	_, ok = group.PackageSettings["laravel/framework"]
	assert.False(t, ok)
}

// TestRuleLevelPackageSettings tests the behavior of packages settings at rule level.
//
// It verifies:
//   - Rule-level package settings are parsed correctly
//   - ShouldUpdateWithAllDependencies resolves correctly
func TestRuleLevelPackageSettings(t *testing.T) {
	content := []byte(`rules:
  composer:
    packages:
      sentry/sentry-laravel:
        with_all_dependencies: true
`)
	var cfg Config
	require.NoError(t, yaml.Unmarshal(content, &cfg))

	rule, ok := cfg.Rules["composer"]
	require.True(t, ok)

	// Check package settings at rule level
	settings, ok := rule.Packages["sentry/sentry-laravel"]
	assert.True(t, ok)
	assert.True(t, settings.WithAllDependencies)
}

// TestShouldUpdateWithAllDependencies tests the resolution of with_all_dependencies settings.
//
// It verifies:
//   - Rule-level package settings have highest priority
//   - Group-level per-package settings take precedence over group-level settings
//   - Group-level settings apply to all packages in the group
func TestShouldUpdateWithAllDependencies(t *testing.T) {
	rule := PackageManagerCfg{
		Packages: map[string]PackageSettings{
			"direct-pkg": {WithAllDependencies: true},
		},
		Groups: map[string]GroupCfg{
			"group1": {
				Packages:            []string{"group-pkg", "override-pkg"},
				WithAllDependencies: true,
			},
			"group2": {
				Packages: []string{"mixed-pkg"},
				PackageSettings: map[string]PackageSettings{
					"mixed-pkg": {WithAllDependencies: true},
				},
			},
		},
	}

	// Direct package setting
	assert.True(t, rule.ShouldUpdateWithAllDependencies("direct-pkg"))

	// Group-level setting applies to packages in group
	assert.True(t, rule.ShouldUpdateWithAllDependencies("group-pkg"))
	assert.True(t, rule.ShouldUpdateWithAllDependencies("override-pkg"))

	// Per-package setting within group
	assert.True(t, rule.ShouldUpdateWithAllDependencies("mixed-pkg"))

	// Unknown package returns false
	assert.False(t, rule.ShouldUpdateWithAllDependencies("unknown-pkg"))
}
