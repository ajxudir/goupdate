package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestLatestMappingUnmarshalHandlesPackageScopesAndArrays tests the behavior of LatestMappingCfg unmarshaling with package scopes and arrays.
//
// It verifies:
//   - Package-specific mappings are parsed correctly
//   - Array values are parsed correctly
//   - Default and package mappings coexist
func TestLatestMappingUnmarshalHandlesPackageScopesAndArrays(t *testing.T) {
	yamlContent := `
latest_mapping:
  default:
    latest: "*"
    stable: "^"
  packages:
    react:
      latest: ">="
    vue:
      - latest
      - "*"
`
	var cfg PackageManagerCfg
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &cfg))

	assert.NotNil(t, cfg.LatestMapping)

	assert.NotNil(t, cfg.LatestMapping.Default)
	assert.Equal(t, "*", cfg.LatestMapping.Default["latest"])
	assert.Equal(t, "^", cfg.LatestMapping.Default["stable"])

	assert.NotNil(t, cfg.LatestMapping.Packages)
	assert.Equal(t, ">=", cfg.LatestMapping.Packages["react"]["latest"])

	assert.Equal(t, "*", cfg.LatestMapping.Packages["vue"]["latest"])
}

// TestLatestMappingUnmarshalCoversEmptyAndSequenceMappings tests the behavior of LatestMappingCfg unmarshaling with empty and sequence mappings.
//
// It verifies:
//   - Empty values are parsed correctly
//   - Single-item sequences are parsed correctly
func TestLatestMappingUnmarshalCoversEmptyAndSequenceMappings(t *testing.T) {
	yamlContent := `
latest_mapping:
  default:
    ""
  packages:
    mypackage:
      - latest
`
	var cfg PackageManagerCfg
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &cfg))

	assert.NotNil(t, cfg.LatestMapping)
	assert.NotNil(t, cfg.LatestMapping.Default)
	assert.NotNil(t, cfg.LatestMapping.Packages["mypackage"])
	assert.Equal(t, "", cfg.LatestMapping.Packages["mypackage"]["latest"])
}

// TestLatestMappingUnmarshalDefaultSequence tests the behavior of LatestMappingCfg unmarshaling with default as a sequence.
//
// It verifies:
//   - Default sequence with multiple elements is parsed correctly
//   - Default sequence with single element is parsed correctly
func TestLatestMappingUnmarshalDefaultSequence(t *testing.T) {
	t.Run("default sequence with multiple elements", func(t *testing.T) {
		yamlContent := `
latest_mapping:
  default:
    - latest
    - stable
    - "*"
`
		var cfg PackageManagerCfg
		require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &cfg))

		assert.NotNil(t, cfg.LatestMapping)
		assert.NotNil(t, cfg.LatestMapping.Default)
		assert.Equal(t, "*", cfg.LatestMapping.Default["latest"])
		assert.Equal(t, "*", cfg.LatestMapping.Default["stable"])
	})

	t.Run("default sequence with single element", func(t *testing.T) {
		yamlContent := `
latest_mapping:
  default:
    - latest
`
		var cfg PackageManagerCfg
		require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &cfg))

		assert.NotNil(t, cfg.LatestMapping)
		assert.NotNil(t, cfg.LatestMapping.Default)
		assert.Equal(t, "", cfg.LatestMapping.Default["latest"])
	})

	t.Run("default sequence with invalid non-scalar", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "default"},
				{Kind: yaml.SequenceNode, Content: []*yaml.Node{
					{Kind: yaml.MappingNode},
				}},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})
}

// TestMergeLatestMappingCfg tests the behavior of merging LatestMappingCfg.
//
// It verifies:
//   - Base and child default mappings are merged
//   - Package mappings are combined from both configs
//   - Nil configs are handled correctly
func TestMergeLatestMappingCfg(t *testing.T) {
	base := &LatestMappingCfg{
		Default: map[string]string{
			"latest": "*",
		},
		Packages: map[string]map[string]string{
			"react": {"latest": ">="},
		},
	}

	child := &LatestMappingCfg{
		Default: map[string]string{
			"stable": "^",
		},
		Packages: map[string]map[string]string{
			"vue": {"latest": "*"},
		},
	}

	merged := mergeLatestMappingCfg(base, child)

	assert.Equal(t, "*", merged.Default["latest"])
	assert.Equal(t, "^", merged.Default["stable"])

	assert.Equal(t, ">=", merged.Packages["react"]["latest"])
	assert.Equal(t, "*", merged.Packages["vue"]["latest"])

	assert.Nil(t, mergeLatestMappingCfg(nil, nil))

	result := mergeLatestMappingCfg(base, nil)
	assert.Equal(t, base, result)

	result = mergeLatestMappingCfg(nil, child)
	assert.Equal(t, child, result)
}

// TestLatestMappingUnmarshalValidation tests the behavior of LatestMappingCfg unmarshaling with validation.
//
// It verifies:
//   - Invalid default value types return error
//   - Invalid packages structure returns error
//   - Invalid package value types return error
//   - Invalid default node type returns error
//   - Invalid packages node type returns error
//   - Invalid top-level field returns error
//   - Invalid top-level node type returns error
func TestLatestMappingUnmarshalValidation(t *testing.T) {
	t.Run("invalid default value type", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "default"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "latest"},
					{Kind: yaml.SequenceNode},
				}},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("invalid packages structure", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "packages"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "react"},
					{Kind: yaml.ScalarNode, Value: "not-a-mapping"},
				}},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "packages.react")
	})

	t.Run("invalid package value type", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "packages"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "vue"},
					{Kind: yaml.MappingNode, Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "latest"},
						{Kind: yaml.MappingNode},
					}},
				}},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vue.latest")
	})

	t.Run("invalid default node type", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "default"},
				{Kind: yaml.ScalarNode, Value: "invalid"},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default")
	})

	t.Run("invalid packages node type", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "packages"},
				{Kind: yaml.ScalarNode, Value: "invalid"},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "packages")
	})

	t.Run("invalid top-level field", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "unknown"},
				{Kind: yaml.MappingNode},
			},
		}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("invalid top-level node type", func(t *testing.T) {
		var lm LatestMappingCfg
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: "invalid"}
		err := lm.UnmarshalYAML(node)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mapping")
	})
}
