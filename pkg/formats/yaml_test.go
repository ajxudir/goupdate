package formats

import (
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYAMLParser tests the behavior of YAMLParser.Parse.
//
// It verifies:
//   - Valid YAML with array-based dependencies is parsed correctly
//   - Package names and versions are extracted
//   - Package types are assigned correctly
//   - Invalid YAML returns an error
func TestYAMLParser(t *testing.T) {
	parser := &YAMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "generic-yaml",
		Fields: map[string]string{
			"dependencies": "prod",
		},
	}

	content := []byte(`
dependencies:
  - name: postgresql
    version: 12.5.8
  - name: redis
    version: 17.11.3
`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Len(t, packages, 2)

	// Check postgresql
	var pg Package
	for _, p := range packages {
		if p.Name == "postgresql" {
			pg = p
		}
	}
	assert.Equal(t, "12.5.8", pg.Version)
	assert.Equal(t, "prod", pg.Type)

	// Test invalid YAML
	_, err = parser.Parse([]byte("invalid: yaml: bad"), cfg)
	assert.Error(t, err)
}

// TestYAMLParserEdgeCases tests edge cases in YAMLParser.
//
// It verifies:
//   - Missing fields are skipped without errors
//   - Non-string versions are formatted correctly
//   - Array entries that aren't maps are skipped
//   - Nested field paths (dot notation) are supported
//   - Constraint mapping works correctly
//   - Ignored packages are filtered out
//   - Maps with interface{} keys are normalized
//   - Array entries require both name and version
func TestYAMLParserEdgeCases(t *testing.T) {
	parser := &YAMLParser{}

	t.Run("missing field is skipped", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields: map[string]string{
				"nonexistent": "prod",
			},
		}

		packages, err := parser.Parse([]byte("dependencies: {}"), cfg)
		require.NoError(t, err)
		assert.Empty(t, packages)
	})

	t.Run("non-string version is formatted", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields: map[string]string{
				"dependencies": "prod",
			},
		}

		content := []byte("dependencies:\n  redis: 123")
		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "123", packages[0].Version)
	})

	t.Run("list entries must be maps", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields: map[string]string{
				"dependencies": "prod",
			},
		}

		content := []byte("dependencies:\n  - name: redis\n    version: 1.0.0\n  - invalid-entry")
		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "redis", packages[0].Name)
	})

	t.Run("nested field paths are supported", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields: map[string]string{
				"chart.dependencies": "prod",
			},
		}

		content := []byte("chart:\n  dependencies:\n    app: 1.2.3")
		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "app", packages[0].Name)
	})

	t.Run("constraint mapping and ignored packages", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields: map[string]string{
				"dependencies": "prod",
			},
			Ignore:            []string{"skipme"},
			ConstraintMapping: map[string]string{"^": "~"},
		}

		content := []byte("dependencies:\n  redis: ^1.0.0\n  skipme: ^2.0.0")
		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 2)

		pkgMap := make(map[string]Package)
		for _, pkg := range packages {
			pkgMap[pkg.Name] = pkg
		}

		// redis: constraint mapped, not ignored
		assert.Equal(t, "~", pkgMap["redis"].Constraint)
		assert.Equal(t, "", pkgMap["redis"].IgnoreReason)

		// skipme: marked as ignored (but still included for visibility)
		assert.Equal(t, "matches ignore pattern 'skipme'", pkgMap["skipme"].IgnoreReason)
	})

	t.Run("map with interface keys is normalized", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager:    "oci",
			Fields:     map[string]string{"dependencies": "prod"},
			Extraction: &config.ExtractionCfg{Pattern: `image:\s*(?P<n>[\w\.-/]+):(?P<version>[\w\.-]+)`},
		}

		content := []byte("dependencies:\n  service:\n    image: service:2.0\n    1: ignored")
		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 1)
		assert.Equal(t, "service", packages[0].Name)
		assert.Equal(t, "2.0", packages[0].Version)
	})

	t.Run("list entries require names and versions", func(t *testing.T) {
		cfg := &config.PackageManagerCfg{
			Manager: "generic-yaml",
			Fields:  map[string]string{"dependencies": "prod"},
			Ignore:  []string{"skipme"},
		}

		content := []byte(`dependencies:
  - name: ""
    version: ""
  - name: skipme
    version: 1.0.0
  - name: valid
    version: 2.0.0`)

		packages, err := parser.Parse(content, cfg)
		require.NoError(t, err)
		require.Len(t, packages, 2) // empty name is skipped, skipme and valid are included

		pkgMap := make(map[string]Package)
		for _, pkg := range packages {
			pkgMap[pkg.Name] = pkg
		}

		// valid: not ignored
		assert.Equal(t, "valid", pkgMap["valid"].Name)
		assert.Equal(t, "2.0.0", pkgMap["valid"].Version)
		assert.Equal(t, "", pkgMap["valid"].IgnoreReason)

		// skipme: marked as ignored (but still included for visibility)
		assert.Equal(t, "matches ignore pattern 'skipme'", pkgMap["skipme"].IgnoreReason)
	})
}

// TestYAMLParserImageExtraction tests container image extraction in YAMLParser.
//
// It verifies:
//   - Docker image specifications are parsed correctly
//   - Image name and tag are extracted using regex patterns
//   - Images without tags get "*" as version
func TestYAMLParserImageExtraction(t *testing.T) {
	parser := &YAMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "oci",
		Fields: map[string]string{
			"services": "prod",
		},
		Extraction: &config.ExtractionCfg{Pattern: `image:\s*(?P<n>[\w\.-/]+):(?P<version>[\w\.-]+)`},
	}

	content := []byte(`
services:
  redis:
    image: redis:7.2.3
  app:
    image: myapp
`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Len(t, packages, 2)

	packageMap := map[string]Package{}
	for _, p := range packages {
		packageMap[p.Name] = p
	}

	redis := packageMap["redis"]
	assert.Equal(t, "redis", redis.Name)
	assert.Equal(t, "7.2.3", redis.Version)

	app := packageMap["myapp"]
	assert.Equal(t, "myapp", app.Name)
	assert.Equal(t, "*", app.Version)
}

// TestParseImageFromMapVariations tests parseImageFromMap variations.
//
// It verifies:
//   - Image string with tag is parsed correctly
//   - Non-string image values return empty version
//   - Missing image key returns defaults
//   - Fallback to colon-splitting when no extraction pattern configured
func TestParseImageFromMapVariations(t *testing.T) {
	cfg := &config.PackageManagerCfg{Extraction: &config.ExtractionCfg{Pattern: `image:\s*(?P<name>[\w\.-/]+):(?P<version>[\w\.-]+)`}}

	version, name := parseImageFromMap(map[string]interface{}{"image": "redis:7"}, "service", "", cfg)
	assert.Equal(t, "7", version)
	assert.Equal(t, "redis", name)

	version, name = parseImageFromMap(map[string]interface{}{"image": 123}, "service", "", cfg)
	assert.Equal(t, "", version)
	assert.Equal(t, "service", name)

	version, name = parseImageFromMap(map[string]interface{}{}, "service", "default", cfg)
	assert.Equal(t, "default", version)
	assert.Equal(t, "service", name)

	// No extraction configured; falls back to splitting the image value
	version, name = parseImageFromMap(map[string]interface{}{"image": "nginx:1.25"}, "service", "", &config.PackageManagerCfg{})
	assert.Equal(t, "1.25", version)
	assert.Equal(t, "nginx", name)
}

// TestYAMLParserWithOverrides tests package overrides in YAMLParser.
//
// It verifies:
//   - Version overrides are applied correctly
//   - Constraint overrides are applied correctly
//   - Non-overridden packages retain original values
func TestYAMLParserWithOverrides(t *testing.T) {
	parser := &YAMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "generic-yaml",
		Fields: map[string]string{
			"dependencies": "prod",
		},
		PackageOverrides: map[string]config.PackageOverrideCfg{
			"postgresql": {
				Version:    "13.0.0",
				Constraint: strPtr("~"),
			},
		},
	}

	content := []byte(`dependencies:
  - name: postgresql
    version: ^12.5.8
  - name: redis
    version: ~17.11.3`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	assert.Equal(t, "13.0.0", pkgMap["postgresql"].Version)
	assert.Equal(t, "~", pkgMap["postgresql"].Constraint)
	assert.Equal(t, "17.11.3", pkgMap["redis"].Version)
}
