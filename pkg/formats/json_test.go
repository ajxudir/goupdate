package formats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
)

// TestJSONParser tests the behavior of JSONParser.Parse.
//
// It verifies:
//   - Valid JSON is parsed correctly
//   - Dependencies and devDependencies are extracted
//   - Package types are assigned correctly (prod/dev)
//   - Version constraints are parsed correctly
//   - Invalid JSON returns an error
func TestJSONParser(t *testing.T) {
	parser := &JSONParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "js",
		Fields: map[string]string{
			"dependencies":    "prod",
			"devDependencies": "dev",
		},
	}

	content := []byte(`{
		"dependencies": {
			"express": "^4.0.0",
			"lodash": "~1.2.3"
		},
		"devDependencies": {
			"jest": "29.0.0"
		}
	}`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Len(t, packages, 3)

	// Check specific packages
	var express, jest Package
	for _, p := range packages {
		if p.Name == "express" {
			express = p
		}
		if p.Name == "jest" {
			jest = p
		}
	}

	assert.Equal(t, "prod", express.Type)
	assert.Equal(t, "^", express.Constraint)
	assert.Equal(t, "4.0.0", express.Version)

	assert.Equal(t, "dev", jest.Type)
	assert.Equal(t, "", jest.Constraint)
	assert.Equal(t, "29.0.0", jest.Version)

	// Test invalid JSON
	_, err = parser.Parse([]byte("invalid"), cfg)
	assert.Error(t, err)
}

// TestJSONParserSkipsInvalidEntries tests that JSONParser skips invalid entries.
//
// It verifies:
//   - Dependencies that are not maps are skipped
//   - Dependencies with non-string versions are skipped
//   - No packages are returned when all entries are invalid
func TestJSONParserSkipsInvalidEntries(t *testing.T) {
	parser := &JSONParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "js",
		Fields: map[string]string{
			"dependencies":    "prod",
			"devDependencies": "dev",
		},
	}

	// dependencies is not a map and devDependencies contains a non-string version
	content := []byte(`{
                "dependencies": "not-a-map",
                "devDependencies": {
                        "jest": {"version": "29.0.0"}
                }
        }`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Empty(t, packages)
}

// TestJSONParserConstraintMapping tests constraint mapping in JSONParser.
//
// It verifies:
//   - Constraint mapping is applied to parsed packages
//   - Configured mappings transform constraints correctly
func TestJSONParserConstraintMapping(t *testing.T) {
	parser := &JSONParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "js",
		Fields: map[string]string{
			"dependencies": "prod",
		},
		ConstraintMapping: map[string]string{"~": "~>"},
	}

	packages, err := parser.Parse([]byte(`{"dependencies": {"pkg": "~1.2.3"}}`), cfg)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.Equal(t, "~>", packages[0].Constraint)
}

// TestJSONParserWithOverrides tests package overrides in JSONParser.
//
// It verifies:
//   - Version overrides are applied
//   - Constraint overrides are applied
//   - Both version and constraint can be overridden together
//   - Ignore flag causes packages to be skipped
//   - Non-overridden packages retain original values
func TestJSONParserWithOverrides(t *testing.T) {
	parser := &JSONParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "js",
		Fields: map[string]string{
			"dependencies": "prod",
		},
		PackageOverrides: map[string]config.PackageOverrideCfg{
			"react": {
				Constraint: strPtr(""), // Override to exact
			},
			"vue": {
				Version: "2.7.14", // Override version
			},
			"axios": {
				Version:    "0.27.2",
				Constraint: strPtr("~"), // Override both
			},
			"lodash": {
				Ignore: true, // Ignore package
			},
		},
	}

	content := []byte(`{
		"dependencies": {
			"react": "^18.0.0",
			"vue": "^3.0.0",
			"axios": "^1.5.0",
			"lodash": "^4.17.21",
			"express": "^4.18.2"
		}
	}`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	// react: constraint overridden to exact
	assert.Equal(t, "18.0.0", pkgMap["react"].Version)
	assert.Equal(t, "", pkgMap["react"].Constraint)

	// vue: version overridden
	assert.Equal(t, "2.7.14", pkgMap["vue"].Version)
	assert.Equal(t, "^", pkgMap["vue"].Constraint)

	// axios: both overridden
	assert.Equal(t, "0.27.2", pkgMap["axios"].Version)
	assert.Equal(t, "~", pkgMap["axios"].Constraint)

	// lodash: ignored
	_, exists := pkgMap["lodash"]
	assert.False(t, exists)

	// express: no override, original values
	assert.Equal(t, "4.18.2", pkgMap["express"].Version)
	assert.Equal(t, "^", pkgMap["express"].Constraint)
}
