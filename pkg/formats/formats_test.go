package formats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/user/goupdate/pkg/config"
)

// strPtr returns a pointer to the given string.
//
// This is a test helper function for creating string pointers.
//
// Parameters:
//   - s: String value
//
// Returns:
//   - *string: Pointer to the string
func strPtr(s string) *string {
	return &s
}

// TestPackageGetters tests the GetName and GetRule methods of Package.
//
// It verifies:
//   - GetName returns the package name
//   - GetRule returns the rule name
func TestPackageGetters(t *testing.T) {
	pkg := Package{
		Name: "lodash",
		Rule: "npm",
	}

	assert.Equal(t, "lodash", pkg.GetName())
	assert.Equal(t, "npm", pkg.GetRule())
}

// TestGetFormatParser tests the behavior of GetFormatParser.
//
// It verifies:
//   - Valid format names return appropriate parsers
//   - Empty format returns error
//   - Whitespace-only format returns error
//   - Unknown format returns error
func TestGetFormatParser(t *testing.T) {
	tests := []struct {
		format string
		isNil  bool
		hasErr bool
	}{
		{"json", false, false},
		{"yaml", false, false},
		{"xml", false, false},
		{"raw", false, false},
		{"unknown", true, true},
		{"", true, true},     // Empty format should return error
		{"   ", true, true},  // Whitespace-only format should return error
		{"\t\n", true, true}, // Tab/newline should return error
	}

	for _, tt := range tests {
		parser, err := GetFormatParser(tt.format)
		if tt.hasErr {
			assert.Error(t, err)
			assert.Nil(t, parser)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, parser)
		}
	}
}

// TestShouldIgnorePackage tests the behavior of shouldIgnorePackage.
//
// It verifies:
//   - Packages matching ignore patterns are ignored
//   - Packages with ignore override are ignored
//   - Non-ignored packages return false
//   - Nil config doesn't cause errors
func TestShouldIgnorePackage(t *testing.T) {
	cfg := &config.PackageManagerCfg{
		Ignore: []string{"php", "ext-*"},
	}

	assert.True(t, shouldIgnorePackage("php", cfg))
	assert.True(t, shouldIgnorePackage("ext-json", cfg))
	assert.False(t, shouldIgnorePackage("symfony", cfg))

	// Test with nil config
	assert.False(t, shouldIgnorePackage("any", nil))

	// Test with package_overrides ignore
	cfg.PackageOverrides = map[string]config.PackageOverrideCfg{
		"lodash": {Ignore: true},
		"react":  {Constraint: strPtr("~")},
	}
	assert.True(t, shouldIgnorePackage("lodash", cfg))
	assert.False(t, shouldIgnorePackage("react", cfg))
}

// TestGetNestedField tests the behavior of GetNestedField.
//
// It verifies:
//   - Nested field access with dot notation works
//   - Simple field access works
//   - Non-existent fields return nil
//   - Invalid paths return nil
//   - Maps with interface{} keys are supported
func TestGetNestedField(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
		},
		"simple": "direct",
	}

	result := GetNestedField(data, "level1.level2.level3")
	assert.Equal(t, "value", result)

	result = GetNestedField(data, "simple")
	assert.Equal(t, "direct", result)

	result = GetNestedField(data, "notfound")
	assert.Nil(t, result)

	result = GetNestedField(data, "level1.notfound")
	assert.Nil(t, result)

	result = GetNestedField(data, "simple.value")
	assert.Nil(t, result)

	dataWithInterfaceKeys := map[string]interface{}{
		"root": map[interface{}]interface{}{"child": "value"},
	}
	result = GetNestedField(dataWithInterfaceKeys, "root.child")
	assert.Equal(t, "value", result)
}
