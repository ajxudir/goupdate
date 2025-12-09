package outdated

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/goupdate/pkg/config"
)

// TestParseJSONVersions tests the behavior of parseJSONVersions.
//
// It verifies:
//   - Parsing JSON array at root level
//   - Parsing JSON array at nested key path
func TestParseJSONVersions(t *testing.T) {
	output := []byte(`["1.0.0", "1.1.0", "2.0.0"]`)
	versions, err := parseJSONVersions("", output)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, versions)

	output = []byte(`{"versions": ["1.0.0", "2.0.0"]}`)
	versions, err = parseJSONVersions("versions", output)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
}

// TestParseJSONVersionsWithBOM tests the behavior of parseJSONVersions with UTF-8 BOM.
//
// It verifies:
//   - BOM is stripped from JSON before parsing
//   - Works with both root array and nested key extraction
func TestParseJSONVersionsWithBOM(t *testing.T) {
	// UTF-8 BOM (EF BB BF) followed by JSON - common with Windows dotnet CLI output
	output := []byte{0xEF, 0xBB, 0xBF}
	output = append(output, []byte(`["1.0.0", "2.0.0"]`)...)
	versions, err := parseJSONVersions("", output)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)

	// With key extraction
	output = []byte{0xEF, 0xBB, 0xBF}
	output = append(output, []byte(`{"versions": ["3.0.0", "4.0.0"]}`)...)
	versions, err = parseJSONVersions("versions", output)
	require.NoError(t, err)
	assert.Equal(t, []string{"3.0.0", "4.0.0"}, versions)
}

// TestParseRegexVersions tests the behavior of parseRegexVersions.
//
// It verifies:
//   - Extracts versions using named capture group
//   - Matches multiple version strings in input
func TestParseRegexVersions(t *testing.T) {
	output := []byte("v1.0.0\nv1.1.0\nv2.0.0")
	versions, err := parseRegexVersions(`v(?P<version>\d+\.\d+\.\d+)`, output)
	require.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0", "2.0.0"}, versions)
}

// TestParseAvailableVersionsForPackage tests the behavior of parseAvailableVersionsForPackage.
//
// It verifies:
//   - Nil config returns error
//   - Default format is JSON
//   - YAML format parsing works correctly
//   - Raw format parsing works correctly
//   - Unsupported format returns error
func TestParseAvailableVersionsForPackage(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := parseAvailableVersionsForPackage("test", nil, []byte("{}"))
		assert.Error(t, err)
	})

	t.Run("json format default", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Format: ""}
		output := []byte(`["1.0.0", "2.0.0"]`)
		versions, err := parseAvailableVersionsForPackage("test", cfg, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("yaml format", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Format: "yaml"}
		output := []byte("- 1.0.0\n- 2.0.0")
		versions, err := parseAvailableVersionsForPackage("test", cfg, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("raw format", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Format: "raw"}
		output := []byte("1.0.0\n2.0.0")
		versions, err := parseAvailableVersionsForPackage("test", cfg, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("unsupported format", func(t *testing.T) {
		cfg := &config.OutdatedCfg{Format: "xml"}
		_, err := parseAvailableVersionsForPackage("test", cfg, []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported output format")
	})
}

// TestParseJSONWithExtraction tests the behavior of parseJSONWithExtraction.
//
// It verifies:
//   - Nil extraction config uses root array
//   - JSONKey is properly extracted from config
func TestParseJSONWithExtraction(t *testing.T) {
	t.Run("nil extraction uses root array", func(t *testing.T) {
		output := []byte(`["1.0.0", "2.0.0"]`)
		versions, err := parseJSONWithExtraction(nil, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("with json key", func(t *testing.T) {
		extraction := &config.OutdatedExtractionCfg{JSONKey: "versions"}
		output := []byte(`{"versions": ["1.0.0", "2.0.0"]}`)
		versions, err := parseJSONWithExtraction(extraction, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})
}

// TestParseYAMLWithExtraction tests the behavior of parseYAMLWithExtraction.
//
// It verifies:
//   - Nil extraction config uses root array
//   - YAMLKey navigation works with dot notation
//   - Returns error for unsupported node types
//   - Returns error when key path is not found
//   - Handles nested YAML keys with map[interface{}]interface{}
func TestParseYAMLWithExtraction(t *testing.T) {
	t.Run("nil extraction uses root array", func(t *testing.T) {
		output := []byte("- 1.0.0\n- 2.0.0")
		versions, err := parseYAMLWithExtraction(nil, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("with yaml key", func(t *testing.T) {
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "versions"}
		output := []byte("versions:\n  - 1.0.0\n  - 2.0.0")
		versions, err := parseYAMLWithExtraction(extraction, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("unsupported type in node", func(t *testing.T) {
		output := []byte("key: 123")
		_, err := parseYAMLWithExtraction(nil, output)
		assert.Error(t, err) // int is not a supported node type
	})

	t.Run("key not found", func(t *testing.T) {
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "missing.key"}
		output := []byte("other: value")
		_, err := parseYAMLWithExtraction(extraction, output)
		assert.Error(t, err)
	})

	t.Run("nested yaml key with map interface keys", func(t *testing.T) {
		// YAML unmarshals nested maps to map[interface{}]interface{} not map[string]interface{}
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "data.versions"}
		output := []byte("data:\n  versions:\n    - 1.0.0\n    - 2.0.0")
		versions, err := parseYAMLWithExtraction(extraction, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})
}

// TestExtractVersionsFromNode tests the behavior of extractVersionsFromNode.
//
// It verifies:
//   - Extracts versions from []interface{} slice
//   - Extracts versions from []string slice
//   - Handles single string value
//   - Returns empty array for empty string
//   - Returns error for unsupported types
func TestExtractVersionsFromNode(t *testing.T) {
	t.Run("interface slice", func(t *testing.T) {
		node := []interface{}{"1.0.0", "2.0.0"}
		versions, err := extractVersionsFromNode(node)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("string slice", func(t *testing.T) {
		node := []string{"1.0.0", "2.0.0"}
		versions, err := extractVersionsFromNode(node)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("single string", func(t *testing.T) {
		node := "1.0.0"
		versions, err := extractVersionsFromNode(node)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0"}, versions)
	})

	t.Run("empty string", func(t *testing.T) {
		node := ""
		versions, err := extractVersionsFromNode(node)
		require.NoError(t, err)
		assert.Empty(t, versions)
	})

	t.Run("unsupported type", func(t *testing.T) {
		node := 123
		_, err := extractVersionsFromNode(node)
		assert.Error(t, err)
	})
}

// TestParseRawWithExtraction tests the behavior of parseRawWithExtraction.
//
// It verifies:
//   - Nil extraction uses default regex pattern
//   - Custom pattern from extraction config works
//   - Empty pattern falls back to default
func TestParseRawWithExtraction(t *testing.T) {
	t.Run("nil extraction uses default pattern", func(t *testing.T) {
		output := []byte("1.0.0\n2.0.0")
		versions, err := parseRawWithExtraction(nil, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("with custom pattern", func(t *testing.T) {
		extraction := &config.OutdatedExtractionCfg{Pattern: `v(?P<version>\d+\.\d+\.\d+)`}
		output := []byte("v1.0.0\nv2.0.0")
		versions, err := parseRawWithExtraction(extraction, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("empty pattern uses default", func(t *testing.T) {
		extraction := &config.OutdatedExtractionCfg{Pattern: ""}
		output := []byte("1.0.0")
		versions, err := parseRawWithExtraction(extraction, output)
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0"}, versions)
	})
}

// TestParseJSONVersionsEdgeCases tests edge cases for parseJSONVersions.
//
// It verifies:
//   - Returns error when nested key is not found
//   - Returns error when key resolves to non-array/non-object
//   - Extracts versions from map keys when value is object
//   - Returns error for invalid JSON
func TestParseJSONVersionsEdgeCases(t *testing.T) {
	t.Run("nested key not found", func(t *testing.T) {
		output := []byte(`{"data": {"missing": []}}`)
		_, err := parseJSONVersions("data.versions", output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json key")
	})

	t.Run("key resolves to non-array non-object", func(t *testing.T) {
		output := []byte(`{"versions": "not-an-array"}`)
		_, err := parseJSONVersions("versions", output)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "did not resolve to an array or object")
	})

	t.Run("map keys used as versions", func(t *testing.T) {
		output := []byte(`{"1.0.0": {}, "2.0.0": {}}`)
		versions, err := parseJSONVersions("", output)
		require.NoError(t, err)
		assert.Len(t, versions, 2)
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		output := []byte(`{invalid json`)
		_, err := parseJSONVersions("", output)
		assert.Error(t, err)
	})
}

// TestParseRegexVersionsEdgeCases tests edge cases for parseRegexVersions.
//
// It verifies:
//   - Returns error for invalid regex pattern
//   - Returns empty array when no matches found
//   - Uses first capture group when no named group exists
//   - Uses full match when no capture groups exist
//   - Deduplicates version strings
func TestParseRegexVersionsEdgeCases(t *testing.T) {
	t.Run("invalid regex pattern", func(t *testing.T) {
		_, err := parseRegexVersions("[invalid", []byte("1.0.0"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid extraction pattern")
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		versions, err := parseRegexVersions(`\d+\.\d+\.\d+`, []byte("no versions here"))
		require.NoError(t, err)
		assert.Empty(t, versions)
	})

	t.Run("pattern without named groups uses first capture", func(t *testing.T) {
		versions, err := parseRegexVersions(`v(\d+\.\d+\.\d+)`, []byte("v1.0.0\nv2.0.0"))
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("pattern without capture groups uses full match", func(t *testing.T) {
		versions, err := parseRegexVersions(`\d+\.\d+\.\d+`, []byte("1.0.0\n2.0.0"))
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})

	t.Run("deduplicates versions", func(t *testing.T) {
		versions, err := parseRegexVersions(`\d+\.\d+\.\d+`, []byte("1.0.0\n1.0.0\n2.0.0"))
		require.NoError(t, err)
		assert.Equal(t, []string{"1.0.0", "2.0.0"}, versions)
	})
}

// TestParseYAMLWithExtractionErrors tests error cases for parseYAMLWithExtraction.
//
// It verifies:
//   - Returns error for invalid YAML
//   - Traverses nested map[string]interface{} keys correctly
//   - Returns error when key not found in non-map node
func TestParseYAMLWithExtractionErrors(t *testing.T) {
	t.Run("invalid YAML returns error", func(t *testing.T) {
		_, err := parseYAMLWithExtraction(&config.OutdatedExtractionCfg{}, []byte("not: valid: yaml: [["))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
	})

	t.Run("traverses nested map[string]interface{} keys", func(t *testing.T) {
		yamlContent := []byte(`
versions:
  available:
    - "1.0.0"
    - "2.0.0"
`)
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "versions.available"}
		versions, err := parseYAMLWithExtraction(extraction, yamlContent)
		require.NoError(t, err)
		assert.Contains(t, versions, "1.0.0")
		assert.Contains(t, versions, "2.0.0")
	})

	t.Run("yaml key not found in non-map node", func(t *testing.T) {
		yamlContent := []byte(`
versions: "just a string"
`)
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "versions.nested"}
		_, err := parseYAMLWithExtraction(extraction, yamlContent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestParseJSONVersionsKeyNotFound tests key path traversal failures in parseJSONVersions.
//
// It verifies:
//   - Returns error when traversing non-object nodes
func TestParseJSONVersionsKeyNotFound(t *testing.T) {
	t.Run("json key path traversal fails on non-object", func(t *testing.T) {
		jsonContent := []byte(`{"data": "string value"}`)
		_, err := parseJSONVersions("data.nested", jsonContent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestParseYAMLWithExtractionMapStringInterface tests YAML parsing with different map types.
//
// It verifies:
//   - Correctly traverses map[string]interface{} structures
//   - Handles map[interface{}]interface{} with mixed key types
func TestParseYAMLWithExtractionMapStringInterface(t *testing.T) {
	t.Run("traverses map[string]interface{} correctly", func(t *testing.T) {
		// YAML that produces map[string]interface{} for nested keys
		yamlContent := []byte(`
data:
  versions:
    - "1.0.0"
    - "2.0.0"
`)
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "data.versions"}
		versions, err := parseYAMLWithExtraction(extraction, yamlContent)
		require.NoError(t, err)
		assert.Contains(t, versions, "1.0.0")
		assert.Contains(t, versions, "2.0.0")
	})

	t.Run("traverses map[interface{}]interface{} with mixed keys", func(t *testing.T) {
		// YAML with boolean keys at root produces map[interface{}]interface{}
		// But we can still access nested string keys
		yamlContent := []byte(`
true: ignored
data:
  versions:
    - "1.0.0"
    - "2.0.0"
`)
		extraction := &config.OutdatedExtractionCfg{YAMLKey: "data.versions"}
		versions, err := parseYAMLWithExtraction(extraction, yamlContent)
		require.NoError(t, err)
		assert.Contains(t, versions, "1.0.0")
		assert.Contains(t, versions, "2.0.0")
	})
}
