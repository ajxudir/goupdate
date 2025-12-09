package update

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/iancoleman/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
)

// TestUpdateJSONVersionPreservesInequalityEncoding tests JSON inequality operator encoding.
//
// It verifies:
//   - Inequality operators like >= are not HTML-escaped in JSON output
//   - Version updates preserve constraint operators in their original form
func TestUpdateJSONVersionPreservesInequalityEncoding(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod"}}
	content := []byte(`{"dependencies":{"axios":">=1.5.0","demo":"^1.0.0"}}`)

	updated, err := updateJSONVersion(content, formats.Package{Name: "axios", Constraint: ">=", Source: "package.json"}, cfg, "1.13.2")
	require.NoError(t, err)

	updatedStr := string(updated)
	assert.NotContains(t, updatedStr, "\\u003e=")
	assert.Contains(t, updatedStr, ">=1.13.2")
}

// TestUpdateJSONVersionPreservesOrdering tests JSON field ordering preservation.
//
// It verifies:
//   - Field order is preserved when updating package versions
//   - Package order within dependencies sections is maintained
func TestUpdateJSONVersionPreservesOrdering(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod", "devDependencies": "dev"}}
	content := []byte(`{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.2",
    "axios": "=1.5.0"
  },
  "devDependencies": {
    "eslint": "=8.47.0"
  }
}`)

	updated, err := updateJSONVersion(content, formats.Package{Name: "axios", Constraint: "=", Source: "package.json"}, cfg, "1.6.0")
	require.NoError(t, err)

	expected := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.2",
    "axios": "=1.6.0"
  },
  "devDependencies": {
    "eslint": "=8.47.0"
  }
}`

	assert.Equal(t, expected, string(updated))
}

// TestUpdateJSONVersionHandlesMapVariants tests handling of different map type representations.
//
// It verifies:
//   - Updates packages in OrderedMap dependencies
//   - Updates packages in standard map[string]interface{} dependencies
//   - Skips non-map fields gracefully
func TestUpdateJSONVersionHandlesMapVariants(t *testing.T) {
	jsonContent := []byte(`{"dependencies":{"demo":"=1.0.0"},"devDependencies":{"demo":">=1.0.0"},"scripts":{}}`)
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod", "devDependencies": "dev", "scripts": "scripts"}}

	originalUnmarshal := jsonUnmarshalFunc
	jsonUnmarshalFunc = func(_ []byte, value interface{}) error {
		data := value.(*orderedmap.OrderedMap)

		deps := orderedmap.New()
		deps.Set("demo", "=1.0.0")
		data.Set("dependencies", *deps)

		data.Set("devDependencies", map[string]interface{}{"demo": ">=1.0.0"})
		data.Set("scripts", "not-a-map")
		return nil
	}
	t.Cleanup(func() { jsonUnmarshalFunc = originalUnmarshal })

	updated, err := updateJSONVersion(jsonContent, formats.Package{Name: "demo", Constraint: "=", Source: "package.json"}, cfg, "2.0.0")
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(updated, &data))

	assert.Equal(t, "=2.0.0", data["dependencies"].(map[string]interface{})["demo"])
	assert.Equal(t, "=2.0.0", data["devDependencies"].(map[string]interface{})["demo"])
}

// TestUpdateJSONVersionMissingPackage tests error handling for missing packages.
//
// It verifies:
//   - Returns error when attempting to update a package not present in the manifest
func TestUpdateJSONVersionMissingPackage(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod"}}
	_, err := updateJSONVersion([]byte(`{"dependencies":{"other":"1.0.0"}}`), formats.Package{Name: "demo", Constraint: "^", Source: "package.json"}, cfg, "1.2.0")
	require.Error(t, err)
}

// TestMarshalJSONEncodeError tests error handling in marshalJSON.
//
// It verifies:
//   - Function doesn't panic with valid ordered map inputs
func TestMarshalJSONEncodeError(t *testing.T) {
	// Test with unmarshalable value - this is hard to trigger normally
	// as orderedmap handles most cases
	assert.NotPanics(t, func() {
		_, _ = marshalJSON(orderedmap.New())
	})
}

// TestNormalizeOrderedMapEscaping tests the normalization of HTML escaping in ordered maps.
//
// It verifies:
//   - Handles pointer to OrderedMap correctly
//   - Handles value OrderedMap correctly
//   - Processes slices of interfaces recursively
//   - Passes through non-map types unchanged
func TestNormalizeOrderedMapEscaping(t *testing.T) {
	t.Run("pointer to OrderedMap", func(t *testing.T) {
		om := orderedmap.New()
		om.Set("key", "value")
		result := normalizeOrderedMapEscaping(om)
		assert.NotNil(t, result)
	})

	t.Run("value OrderedMap", func(t *testing.T) {
		om := orderedmap.New()
		om.Set("key", "value")
		result := normalizeOrderedMapEscaping(*om)
		assert.NotNil(t, result)
	})

	t.Run("slice of interfaces", func(t *testing.T) {
		om := orderedmap.New()
		om.Set("key", "value")
		slice := []interface{}{om, "string", 123}
		result := normalizeOrderedMapEscaping(slice)
		assert.NotNil(t, result)
	})

	t.Run("default type passthrough", func(t *testing.T) {
		str := "hello"
		result := normalizeOrderedMapEscaping(str)
		assert.Equal(t, str, result)
	})
}

// TestUpdateJSONVersionUnmarshalError tests error handling when JSON unmarshaling fails.
//
// It verifies:
//   - Returns error when JSON unmarshaling fails
//   - Error message contains the unmarshal failure details
func TestUpdateJSONVersionUnmarshalError(t *testing.T) {
	originalUnmarshal := jsonUnmarshalFunc
	jsonUnmarshalFunc = func([]byte, interface{}) error { return errors.New("unmarshal fail") }
	t.Cleanup(func() { jsonUnmarshalFunc = originalUnmarshal })

	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod"}}
	_, err := updateJSONVersion([]byte(`{"dependencies":{"demo":"1.0.0"}}`), formats.Package{Name: "demo", Source: "package.json"}, cfg, "1.1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal fail")
}

// TestUpdateJSONVersionOrderedMapValue tests handling of OrderedMap returned as value.
//
// It verifies:
//   - Updates packages correctly when dependencies are OrderedMap values (not pointers)
func TestUpdateJSONVersionOrderedMapValue(t *testing.T) {
	jsonContent := []byte(`{"dependencies":{"demo":"=1.0.0"}}`)
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod"}}

	originalUnmarshal := jsonUnmarshalFunc
	jsonUnmarshalFunc = func(_ []byte, value interface{}) error {
		data := value.(*orderedmap.OrderedMap)

		// Return OrderedMap as value (not pointer) to trigger deps = v case
		deps := orderedmap.New()
		deps.Set("demo", "=1.0.0")
		data.Set("dependencies", *deps) // value, not pointer

		return nil
	}
	t.Cleanup(func() { jsonUnmarshalFunc = originalUnmarshal })

	updated, err := updateJSONVersion(jsonContent, formats.Package{Name: "demo", Constraint: "=", Source: "package.json"}, cfg, "2.0.0")
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(updated, &data))
	assert.Equal(t, "=2.0.0", data["dependencies"].(map[string]interface{})["demo"])
}

// TestUpdateJSONVersionFieldNotInData tests handling of missing fields in JSON data.
//
// It verifies:
//   - Updates successfully when some configured fields don't exist in the data
//   - Ignores missing fields gracefully
func TestUpdateJSONVersionFieldNotInData(t *testing.T) {
	// Test when config specifies a field that doesn't exist in the JSON data
	jsonContent := []byte(`{"dependencies":{"demo":"1.0.0"}}`)
	// Config includes "devDependencies" field but data only has "dependencies"
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod", "devDependencies": "dev"}}

	updated, err := updateJSONVersion(jsonContent, formats.Package{Name: "demo", Constraint: "", Source: "package.json"}, cfg, "2.0.0")
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(updated, &data))
	assert.Equal(t, "2.0.0", data["dependencies"].(map[string]interface{})["demo"])
}

// TestUpdateJSONVersionFieldNotMapType tests handling of non-map field types.
//
// It verifies:
//   - Skips fields that exist but are not map types
//   - Still updates other valid dependency fields
func TestUpdateJSONVersionFieldNotMapType(t *testing.T) {
	// Test when a field exists but its value is not a map (hits default: continue)
	jsonContent := []byte(`{"dependencies":{"demo":"1.0.0"},"version":"1.0.0"}`)
	// Config includes "version" field which is a string, not a map
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod", "version": "meta"}}

	updated, err := updateJSONVersion(jsonContent, formats.Package{Name: "demo", Constraint: "", Source: "package.json"}, cfg, "2.0.0")
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(updated, &data))
	assert.Equal(t, "2.0.0", data["dependencies"].(map[string]interface{})["demo"])
}

// TestUpdateJSONVersionParserError tests error handling when JSON parsing fails.
//
// It verifies:
//   - Returns parser error for invalid JSON syntax
func TestUpdateJSONVersionParserError(t *testing.T) {
	// Test when JSON parser.Parse fails due to bad extraction config
	cfg := config.PackageManagerCfg{Format: "json", Fields: map[string]string{"dependencies": "prod"}}

	originalUnmarshal := jsonUnmarshalFunc
	jsonUnmarshalFunc = func([]byte, interface{}) error { return errors.New("parse fail") }
	t.Cleanup(func() { jsonUnmarshalFunc = originalUnmarshal })

	_, err := updateJSONVersion([]byte(`{"dependencies":{"demo":"1.0.0"}}`), formats.Package{Name: "demo", Source: "package.json"}, cfg, "1.1.0")
	require.Error(t, err)
}
