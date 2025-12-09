package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/utils"
)

// TestUpdateRawVersionCaseInsensitiveMatch tests case-insensitive package name matching.
//
// It verifies:
//   - Matches package names case-insensitively (e.g., "Demo" matches "demo")
func TestUpdateRawVersionCaseInsensitiveMatch(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "raw", Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<name>\w+)==(?P<version>[\d\.]+)`}}
	content := []byte("Demo==1.0.0\n")
	updated, err := updateRawVersion(content, formats.Package{Name: "demo", Constraint: "==", Source: "reqs.txt"}, cfg, "2.0.0")
	require.NoError(t, err)
	assert.Contains(t, string(updated), "==2.0.0")
}

// TestUpdateRawVersionMissingPackage tests error handling for missing packages.
//
// It verifies:
//   - Returns error when attempting to update a package not present in the file
func TestUpdateRawVersionMissingPackage(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "raw", Extraction: &config.ExtractionCfg{Pattern: `(?m)^(?P<n>\w+)==(?P<version>[\d\.]+)`}}
	_, err := updateRawVersion([]byte("other==1.0.0\n"), formats.Package{Name: "demo", Source: "reqs.txt"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateRawVersionExtractionError tests error handling when regex extraction fails.
//
// It verifies:
//   - Returns error for invalid regex patterns
func TestUpdateRawVersionExtractionError(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "raw", Extraction: &config.ExtractionCfg{Pattern: "("}}
	_, err := updateRawVersion([]byte("demo 1.0.0"), formats.Package{Name: "demo"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateRawVersionDuplicateVersionsSafety tests safety when multiple packages share the same version.
//
// It verifies:
//   - Only updates the target package when multiple packages have the same version
//   - Does not accidentally update other packages with the same version number
func TestUpdateRawVersionDuplicateVersionsSafety(t *testing.T) {
	// Test that when multiple packages have the same version number,
	// only the target package's version is updated (not the first occurrence)
	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<name>[\w\-]+)==(?P<version>[\d\.]+)`,
		},
	}

	// Both packages have version 1.0.0 - flask comes second
	content := []byte("requests==1.0.0\nflask==1.0.0\ndjango==2.0.0\n")

	// Update flask (the second package with 1.0.0)
	updated, err := updateRawVersion(content, formats.Package{
		Name:       "flask",
		Constraint: "==",
		Source:     "requirements.txt",
	}, cfg, "3.0.0")

	require.NoError(t, err)
	result := string(updated)

	// Flask should be updated to 3.0.0
	assert.Contains(t, result, "flask==3.0.0", "flask should be updated to 3.0.0")

	// Requests should still be 1.0.0 (not accidentally updated)
	assert.Contains(t, result, "requests==1.0.0", "requests should remain at 1.0.0")

	// Django should be unchanged
	assert.Contains(t, result, "django==2.0.0", "django should remain at 2.0.0")
}

// TestUpdateRawVersionMultipleSameVersions tests updates with three packages sharing the same version.
//
// It verifies:
//   - Only updates the target package when multiple packages have identical versions
//   - Other packages remain unchanged
func TestUpdateRawVersionMultipleSameVersions(t *testing.T) {
	// Test with three packages sharing the same version
	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<n>[\w\-]+)==(?P<version>[\d\.]+)`,
		},
	}

	content := []byte("alpha==1.0.0\nbeta==1.0.0\ngamma==1.0.0\n")

	// Update beta (the middle package)
	updated, err := updateRawVersion(content, formats.Package{
		Name:       "beta",
		Constraint: "==",
		Source:     "requirements.txt",
	}, cfg, "2.0.0")

	require.NoError(t, err)
	result := string(updated)

	// Only beta should be updated
	assert.Contains(t, result, "alpha==1.0.0", "alpha should remain at 1.0.0")
	assert.Contains(t, result, "beta==2.0.0", "beta should be updated to 2.0.0")
	assert.Contains(t, result, "gamma==1.0.0", "gamma should remain at 1.0.0")
}

// TestUpdateRawVersionWithVersionAlt tests the version_alt fallback mechanism.
//
// It verifies:
//   - Uses version_alt capture group when version group is not available
func TestUpdateRawVersionWithVersionAlt(t *testing.T) {
	// Test version_alt fallback
	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<name>\w+)\s+(?P<version_alt>[\d\.]+)`,
		},
	}
	content := []byte("demo 1.0.0\n")
	updated, err := updateRawVersion(content, formats.Package{Name: "demo", Constraint: "", Source: "reqs.txt"}, cfg, "2.0.0")
	require.NoError(t, err)
	assert.Contains(t, string(updated), "2.0.0")
}

// TestUpdateRawVersionNoVersionGroup tests error handling when version group is missing.
//
// It verifies:
//   - Returns error when regex pattern doesn't capture a version group
func TestUpdateRawVersionNoVersionGroup(t *testing.T) {
	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<name>\w+)`,
		},
	}
	content := []byte("demo\n")
	_, err := updateRawVersion(content, formats.Package{Name: "demo", Source: "reqs.txt"}, cfg, "2.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no version found")
}

// TestUpdateRawVersionInvalidVersionIndices tests bounds checking for version indices.
//
// It verifies:
//   - Returns error when version indices are out of bounds
//   - Prevents panics from invalid index positions
func TestUpdateRawVersionInvalidVersionIndices(t *testing.T) {
	// Test bounds check for invalid version indices
	originalExtract := extractAllMatchesWithIndexFunc
	defer func() { extractAllMatchesWithIndexFunc = originalExtract }()

	// Mock extraction to return invalid indices
	extractAllMatchesWithIndexFunc = func(pattern, text string) ([]utils.MatchWithIndex, error) {
		return []utils.MatchWithIndex{
			{
				Groups:     map[string]string{"name": "demo", "version": "1.0.0"},
				GroupIndex: map[string][2]int{"name": {0, 4}, "version": {100, 200}}, // Invalid indices beyond text length
			},
		}, nil
	}

	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<name>\w+)==(?P<version>[\d\.]+)`,
		},
	}
	content := []byte("demo==1.0.0\n")
	_, err := updateRawVersion(content, formats.Package{Name: "demo", Source: "reqs.txt"}, cfg, "2.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid version position")
}

// TestUpdateRawVersionWithConstraintGroup tests handling of separate constraint capture groups.
//
// It verifies:
//   - Correctly updates version when constraint is captured in a separate group
//   - Does not include constraint in the version replacement
func TestUpdateRawVersionWithConstraintGroup(t *testing.T) {
	// Test when constraint is captured separately
	cfg := config.PackageManagerCfg{
		Format: "raw",
		Extraction: &config.ExtractionCfg{
			Pattern: `(?m)^(?P<name>\w+)(?P<constraint>[><=~^]+)(?P<version>[\d\.]+)`,
		},
	}
	content := []byte("demo>=1.0.0\n")
	updated, err := updateRawVersion(content, formats.Package{Name: "demo", Constraint: ">=", Source: "reqs.txt"}, cfg, "2.0.0")
	require.NoError(t, err)
	assert.Contains(t, string(updated), "demo>=2.0.0")
}

// TestUpdateRawVersionNoExtractionPattern tests error handling when extraction pattern is missing.
//
// It verifies:
//   - Returns UnsupportedError when no extraction pattern is configured
func TestUpdateRawVersionNoExtractionPattern(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "raw"}
	_, err := updateRawVersion([]byte("demo 1.0.0"), formats.Package{Name: "demo"}, cfg, "1.1.0")
	require.Error(t, err)
	assert.True(t, errors.IsUnsupported(err))
}
