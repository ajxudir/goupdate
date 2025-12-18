package formats

import (
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRawParser tests the behavior of RawParser.Parse.
//
// It verifies:
//   - Regex pattern extracts package names, versions, and constraints
//   - Multiple packages are parsed from text
//   - Different constraint operators are captured correctly
//   - No extraction pattern results in empty package list
func TestRawParser(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "pip",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-]+)\s*(?P<constraint>[<>=~]+)?\s*(?P<version>[\d\.]+)`,
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte("flask>=2.0.0\ndjango~=3.2\nrequests==1.0.0")

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Len(t, packages, 3)

	// Check flask
	var flask Package
	for _, p := range packages {
		if p.Name == "flask" {
			flask = p
		}
	}
	assert.Equal(t, ">=", flask.Constraint)
	assert.Equal(t, "2.0.0", flask.Version)

	// Test with no extraction config
	cfg.Extraction.Pattern = ""
	packages, err = parser.Parse(content, cfg)
	assert.NoError(t, err)
	assert.Empty(t, packages)
}

// TestRawParserRequirementsMissingVersion tests packages without version specs.
//
// It verifies:
//   - Packages without versions get "*" as version
//   - Packages with versions are parsed correctly
//   - Alternative version group name (version_alt) works
func TestRawParserRequirementsMissingVersion(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "python",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-\.]+)(?:[ \t]*(?P<constraint>[><=~!]+)[ \t]*(?P<version>[\w\.\-\+]+)|[ \t]+(?P<version_alt>[\w\.\-\+]+))?`,
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte("flask>=2.0.0\npandas\npytest>=7.0.0\n")

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	versions := map[string]string{}
	for _, pkg := range packages {
		versions[pkg.Name] = pkg.Version
	}

	assert.Equal(t, "*", versions["pandas"])
	assert.Equal(t, "2.0.0", versions["flask"])
	assert.Equal(t, "7.0.0", versions["pytest"])
}

// TestRawParserRequirementsWhitespaceVersion tests whitespace-separated versions.
//
// It verifies:
//   - Versions separated by whitespace are captured
//   - Works alongside operator-based version specs
func TestRawParserRequirementsWhitespaceVersion(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "python",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-\.]+)(?:[ \t]*(?P<constraint>[><=~!]+)[ \t]*(?P<version>[\w\.\-\+]+)|[ \t]+(?P<version_alt>[\w\.\-\+]+))?`,
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte("django 4.2\nrequests==2.28.1\n")

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	versions := map[string]string{}
	for _, pkg := range packages {
		versions[pkg.Name] = pkg.Version
	}

	assert.Equal(t, "4.2", versions["django"])
	assert.Equal(t, "2.28.1", versions["requests"])
}

// TestRawParserPipfileSections tests INI-style section parsing (Pipfile).
//
// It verifies:
//   - Sections are extracted correctly from INI-style files
//   - Packages from different sections get correct types (prod/dev)
//   - Section headers don't match the regex pattern
//   - Non-package lines in sections are ignored
func TestRawParserPipfileSections(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "python",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-\.]+)\s*=\s*"(?P<constraint>[><=~!]+)?\s*(?P<version>[\w\.\-\+\*]+)?"`,
		},
		Fields: map[string]string{
			"packages":     "prod",
			"dev-packages": "dev",
		},
	}

	content := []byte(`[[source]]
url = "https://pypi.org/simple"
name = "pypi"

[packages]
flask = ">=2.0.0"
django = "~=4.2"
requests = "*"

[dev-packages]
pytest = ">=7.0.0"
black = "==22.10.0"`)

	packagesSection := extractSection(string(content), "packages")
	require.Contains(t, packagesSection, "flask = \">=2.0.0\"")
	require.NotContains(t, packagesSection, "[dev-packages]")
	t.Logf("packagesSection=%q", packagesSection)

	devSection := extractSection(string(content), "dev-packages")
	require.Contains(t, devSection, "pytest = \">=7.0.0\"")
	require.NotContains(t, devSection, "[packages]")
	t.Logf("devSection=%q", devSection)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 5)

	packageTypes := map[string]string{}
	for _, pkg := range packages {
		packageTypes[pkg.Name] = pkg.Type
	}

	assert.NotContains(t, packageTypes, "url")
	assert.NotContains(t, packageTypes, "name")
	assert.Equal(t, "prod", packageTypes["flask"])
	assert.Equal(t, "prod", packageTypes["django"])
	assert.Equal(t, "prod", packageTypes["requests"])
	assert.Equal(t, "dev", packageTypes["pytest"])
	assert.Equal(t, "dev", packageTypes["black"])
}

// TestRawParserPipfileMissingSection tests handling of missing sections.
//
// It verifies:
//   - Missing sections don't cause errors
//   - Only packages from present sections are returned
func TestRawParserPipfileMissingSection(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "python",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-\.]+)\s*=\s*"(?P<constraint>[><=~!]+)?\s*(?P<version>[\w\.\-\+\*]+)?"`,
		},
		Fields: map[string]string{
			"packages":     "prod",
			"dev-packages": "dev",
		},
	}

	content := []byte(`[packages]
flask = ">=2.0.0"
django = "~=4.2"`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	for _, pkg := range packages {
		assert.Equal(t, "prod", pkg.Type)
	}
}

// TestRawParserWithOverrides tests package overrides in RawParser.
//
// It verifies:
//   - Version overrides are applied
//   - Constraint overrides are applied
//   - Non-overridden packages retain original values
func TestRawParserWithOverrides(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "pip",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-]+)\s*(?P<constraint>[<>=~]+)?\s*(?P<version>[\d\.]+)`,
		},
		Fields: map[string]string{
			"packages": "prod",
		},
		PackageOverrides: map[string]config.PackageOverrideCfg{
			"flask": {
				Version:    "1.1.4",
				Constraint: strPtr(""),
			},
		},
	}

	content := []byte("flask>=2.0.0\ndjango~=3.2\nrequests==1.0.0")

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	assert.Equal(t, "1.1.4", pkgMap["flask"].Version)
	assert.Equal(t, "", pkgMap["flask"].Constraint)
	assert.Equal(t, "3.2", pkgMap["django"].Version)
}

// TestRawParserIgnoresPackages tests package ignore functionality.
//
// It verifies:
//   - Packages matching ignore patterns are marked with IgnoreReason
//   - Non-ignored packages have no IgnoreReason
func TestRawParserIgnoresPackages(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager:    "pip",
		Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>[\w\-]+)\s*(?P<constraint>[<>=~]+)?\s*(?P<version>[\d\.]+)`},
		Fields:     map[string]string{"packages": "prod"},
		Ignore:     []string{"skipme"},
	}

	content := []byte("skipme>=1.0.0\nkeepme==2.0.0")

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	// skipme: marked as ignored (but still included for visibility)
	assert.Equal(t, "matches ignore pattern 'skipme'", pkgMap["skipme"].IgnoreReason)

	// keepme: not ignored
	assert.Equal(t, "", pkgMap["keepme"].IgnoreReason)
}

// TestRawParserConstraintMapping tests constraint mapping in RawParser.
//
// It verifies:
//   - Constraint mapping is applied to parsed packages
//   - Configured mappings transform constraints correctly
func TestRawParserConstraintMapping(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager:           "pip",
		Extraction:        &config.ExtractionCfg{Pattern: `(?m)^(?P<n>[\w\-]+)\s*(?P<constraint>[<>=~]+)?\s*(?P<version>[\d\.]+)`},
		Fields:            map[string]string{"packages": "prod"},
		ConstraintMapping: map[string]string{">=": ">>="},
	}

	packages, err := parser.Parse([]byte("pkg>=1.0.0"), cfg)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.Equal(t, ">>=", packages[0].Constraint)
}

// TestRawParserInvalidPattern tests handling of invalid regex patterns.
//
// It verifies:
//   - Invalid regex patterns return an error
func TestRawParserInvalidPattern(t *testing.T) {
	parser := &RawParser{}
	cfg := &config.PackageManagerCfg{
		Manager:    "pip",
		Extraction: &config.ExtractionCfg{Pattern: "("},
		Fields:     map[string]string{"packages": "prod"},
	}

	_, err := parser.Parse([]byte("package"), cfg)
	assert.Error(t, err)
}
