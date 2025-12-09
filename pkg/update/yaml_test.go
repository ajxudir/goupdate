package update

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// TestUpdateYAMLVersionUnmarshalError tests error handling when YAML unmarshaling fails.
//
// It verifies:
//   - Returns error when YAML unmarshaling fails
func TestUpdateYAMLVersionUnmarshalError(t *testing.T) {
	originalUnmarshal := yamlUnmarshalFunc
	yamlUnmarshalFunc = func([]byte, interface{}) error { return errors.New("yaml fail") }
	t.Cleanup(func() { yamlUnmarshalFunc = originalUnmarshal })

	cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"dependencies": "prod"}}
	_, err := updateYAMLVersion([]byte("dependencies:\n  demo: 1.0.0\n"), formats.Package{Name: "demo", Constraint: "^", Source: "deps.yaml"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateYAMLVersionMissingPackage tests error handling for missing packages.
//
// It verifies:
//   - Returns error when attempting to update a package not present in the manifest
func TestUpdateYAMLVersionMissingPackage(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"dependencies": "prod"}}
	_, err := updateYAMLVersion([]byte("dependencies:\n  other: 1.0.0\n"), formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateYAMLVersionFieldNotMap tests handling of non-map field types.
//
// It verifies:
//   - Skips fields that exist but are not map types
func TestUpdateYAMLVersionFieldNotMap(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"dependencies": "prod"}}
	// Make dependencies a string instead of map
	content := []byte("dependencies: \"not a map\"\n")
	_, err := updateYAMLVersion(content, formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateYAMLVersionNestedFields tests handling of nested YAML fields.
//
// It verifies:
//   - Correctly navigates and updates packages in nested YAML structures
//   - Supports dot-notation field paths (e.g., "level1.level2.deps")
//   - Returns error for non-existent nested paths
//   - Handles single level fields correctly
func TestUpdateYAMLVersionNestedFields(t *testing.T) {
	t.Run("handles deeply nested fields", func(t *testing.T) {
		cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"level1.level2.deps": "prod"}}
		content := []byte(`level1:
  level2:
    deps:
      demo: 1.0.0
`)
		updated, err := updateYAMLVersion(content, formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "2.0.0")
		require.NoError(t, err)
		assert.Contains(t, string(updated), "2.0.0")
	})

	t.Run("returns error for non-existent nested path", func(t *testing.T) {
		cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"missing.path.deps": "prod"}}
		content := []byte(`other: value`)
		_, err := updateYAMLVersion(content, formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "2.0.0")
		require.Error(t, err)
	})

	t.Run("handles single level field", func(t *testing.T) {
		cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"deps": "prod"}}
		content := []byte(`deps:
  demo: 1.0.0
`)
		updated, err := updateYAMLVersion(content, formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "3.0.0")
		require.NoError(t, err)
		assert.Contains(t, string(updated), "3.0.0")
	})
}

// TestUpdateYAMLVersionParserError tests error handling when YAML parsing fails.
//
// It verifies:
//   - Returns parser error for invalid YAML syntax
func TestUpdateYAMLVersionParserError(t *testing.T) {
	// Test when YAML parser.Parse fails
	cfg := config.PackageManagerCfg{Format: "yaml", Fields: map[string]string{"invalid/path": "prod"}}
	_, err := updateYAMLVersion([]byte("not: yaml: content:"), formats.Package{Name: "demo", Source: "deps.yaml"}, cfg, "1.1.0")
	require.Error(t, err)
}
