package output

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteScanResult_JSON tests the behavior of WriteScanResult with JSON format.
//
// It verifies:
//   - Writes valid JSON that can be unmarshaled back
//   - Summary and files are correctly serialized
func TestWriteScanResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	result := &ScanResult{
		Summary: ScanSummary{
			Directory:    "/test",
			TotalEntries: 2,
			UniqueFiles:  2,
			RulesMatched: 1,
		},
		Files: []ScanEntry{
			{Rule: "npm", PM: "js", Format: "json", File: "package.json"},
			{Rule: "npm", PM: "js", Format: "json", File: "other/package.json"},
		},
	}

	err := WriteScanResult(&buf, FormatJSON, result)
	require.NoError(t, err)

	var parsed ScanResult
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "/test", parsed.Summary.Directory)
	assert.Equal(t, 2, parsed.Summary.TotalEntries)
	assert.Len(t, parsed.Files, 2)
}

// TestWriteScanResult_XML tests the behavior of WriteScanResult with XML format.
//
// It verifies:
//   - Writes XML with proper header
//   - Contains scanResult root element and summary data
func TestWriteScanResult_XML(t *testing.T) {
	var buf bytes.Buffer
	result := &ScanResult{
		Summary: ScanSummary{
			Directory:    "/test",
			TotalEntries: 1,
			UniqueFiles:  1,
			RulesMatched: 1,
		},
		Files: []ScanEntry{
			{Rule: "npm", PM: "js", Format: "json", File: "package.json"},
		},
	}

	err := WriteScanResult(&buf, FormatXML, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<?xml version=")
	assert.Contains(t, output, "<scanResult>")
	assert.Contains(t, output, "<directory>/test</directory>")
}

// TestWriteScanResult_CSV tests the behavior of WriteScanResult with CSV format.
//
// It verifies:
//   - Writes CSV with header and data rows
func TestWriteScanResult_CSV(t *testing.T) {
	var buf bytes.Buffer
	result := &ScanResult{
		Files: []ScanEntry{
			{Rule: "npm", PM: "js", Format: "json", File: "package.json"},
			{Rule: "composer", PM: "php", Format: "json", File: "composer.json"},
		},
	}

	err := WriteScanResult(&buf, FormatCSV, result)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3) // header + 2 rows
	assert.Contains(t, lines[0], "RULE")
	assert.Contains(t, lines[1], "npm")
	assert.Contains(t, lines[2], "composer")
}

// TestWriteListResult_JSON tests the behavior of WriteListResult with JSON format.
//
// It verifies:
//   - Writes valid JSON with summary, packages, and warnings
func TestWriteListResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	result := &ListResult{
		Summary: ListSummary{
			TotalPackages: 2,
		},
		Packages: []ListPackage{
			{Rule: "npm", PM: "js", Type: "prod", Name: "express", Version: "4.18.0"},
			{Rule: "npm", PM: "js", Type: "dev", Name: "jest", Version: "29.0.0"},
		},
		Warnings: []string{"warning1"},
	}

	err := WriteListResult(&buf, FormatJSON, result)
	require.NoError(t, err)

	var parsed ListResult
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, 2, parsed.Summary.TotalPackages)
	assert.Len(t, parsed.Packages, 2)
	assert.Len(t, parsed.Warnings, 1)
}

// TestWriteListResult_CSV tests the behavior of WriteListResult with CSV format.
//
// It verifies:
//   - Writes CSV with package columns
func TestWriteListResult_CSV(t *testing.T) {
	var buf bytes.Buffer
	result := &ListResult{
		Packages: []ListPackage{
			{Rule: "npm", PM: "js", Type: "prod", Name: "express", Version: "4.18.0", Status: "LockFound"},
		},
	}

	err := WriteListResult(&buf, FormatCSV, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "RULE,PM,TYPE,CONSTRAINT,VERSION,INSTALLED,STATUS,GROUP,NAME")
	assert.Contains(t, output, "npm")
	assert.Contains(t, output, "express")
}

// TestWriteOutdatedResult_JSON tests the behavior of WriteOutdatedResult with JSON format.
//
// It verifies:
//   - Writes valid JSON with summary and package data
func TestWriteOutdatedResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	result := &OutdatedResult{
		Summary: OutdatedSummary{
			TotalPackages:    3,
			OutdatedPackages: 1,
			UpToDatePackages: 2,
			FailedPackages:   0,
			HasMajor:         1,
			HasMinor:         2,
			HasPatch:         3,
		},
		Packages: []OutdatedPackage{
			{Rule: "npm", PM: "js", Name: "express", Version: "4.17.0", Major: "5.0.0", Status: "Outdated"},
			{Rule: "npm", PM: "js", Name: "lodash", Version: "4.17.21", Status: "UpToDate"},
		},
	}

	err := WriteOutdatedResult(&buf, FormatJSON, result)
	require.NoError(t, err)

	var parsed OutdatedResult
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, 3, parsed.Summary.TotalPackages)
	assert.Equal(t, 1, parsed.Summary.OutdatedPackages)
	assert.Equal(t, 1, parsed.Summary.HasMajor)
	assert.Equal(t, 2, parsed.Summary.HasMinor)
	assert.Equal(t, 3, parsed.Summary.HasPatch)
	assert.Len(t, parsed.Packages, 2)
}

// TestWriteOutdatedResult_XML tests the behavior of WriteOutdatedResult with XML format.
//
// It verifies:
//   - Writes XML with outdatedResult root element
func TestWriteOutdatedResult_XML(t *testing.T) {
	var buf bytes.Buffer
	result := &OutdatedResult{
		Summary: OutdatedSummary{
			TotalPackages:    1,
			OutdatedPackages: 1,
		},
		Packages: []OutdatedPackage{
			{Rule: "npm", PM: "js", Name: "express", Status: "Outdated"},
		},
	}

	err := WriteOutdatedResult(&buf, FormatXML, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<outdatedResult>")
	assert.Contains(t, output, "<totalPackages>1</totalPackages>")
}

// TestWriteUpdateResult_JSON tests the behavior of WriteUpdateResult with JSON format.
//
// It verifies:
//   - Writes valid JSON with dry run flag and package updates
func TestWriteUpdateResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	result := &UpdateResult{
		Summary: UpdateSummary{
			TotalPackages:   2,
			UpdatedPackages: 1,
			FailedPackages:  0,
			DryRun:          true,
		},
		Packages: []UpdatePackage{
			{Rule: "npm", PM: "js", Name: "express", Version: "4.17.0", Target: "4.18.0", Status: "Planned"},
			{Rule: "npm", PM: "js", Name: "lodash", Version: "4.17.21", Target: "#N/A", Status: "UpToDate"},
		},
	}

	err := WriteUpdateResult(&buf, FormatJSON, result)
	require.NoError(t, err)

	var parsed UpdateResult
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.Summary.DryRun)
	assert.Equal(t, 1, parsed.Summary.UpdatedPackages)
	assert.Len(t, parsed.Packages, 2)
}

// TestWriteUpdateResult_CSV tests the behavior of WriteUpdateResult with CSV format.
//
// It verifies:
//   - Writes CSV with update columns including target version
func TestWriteUpdateResult_CSV(t *testing.T) {
	var buf bytes.Buffer
	result := &UpdateResult{
		Packages: []UpdatePackage{
			{Rule: "npm", PM: "js", Name: "express", Version: "4.17.0", Target: "4.18.0", Status: "Updated"},
		},
	}

	err := WriteUpdateResult(&buf, FormatCSV, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "RULE,PM,TYPE,CONSTRAINT,VERSION,INSTALLED,TARGET,STATUS,GROUP,NAME,ERROR")
	assert.Contains(t, output, "express")
	assert.Contains(t, output, "4.18.0")
}

// TestWriteResult_UnsupportedFormat tests the behavior of Write functions with unsupported format.
//
// It verifies:
//   - Returns error for table format on all Write functions
func TestWriteResult_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer

	err := WriteScanResult(&buf, FormatTable, &ScanResult{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")

	err = WriteListResult(&buf, FormatTable, &ListResult{})
	assert.Error(t, err)

	err = WriteOutdatedResult(&buf, FormatTable, &OutdatedResult{})
	assert.Error(t, err)

	err = WriteUpdateResult(&buf, FormatTable, &UpdateResult{})
	assert.Error(t, err)
}

// TestScanResultXMLStructure tests the behavior of ScanResult XML marshaling.
//
// It verifies:
//   - XML structure with proper root element and nested elements
func TestScanResultXMLStructure(t *testing.T) {
	result := &ScanResult{
		Summary: ScanSummary{
			Directory:    "/test",
			TotalEntries: 1,
			UniqueFiles:  1,
			RulesMatched: 1,
		},
		Files: []ScanEntry{
			{Rule: "npm", PM: "js", Format: "json", File: "package.json"},
		},
	}

	data, err := xml.MarshalIndent(result, "", "  ")
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "<scanResult>")
	assert.Contains(t, output, "<summary>")
	assert.Contains(t, output, "<files>")
	assert.Contains(t, output, "<file>")
}

// TestWriteOutdatedResult_CSV tests the behavior of WriteOutdatedResult with CSV format.
//
// It verifies:
//   - Writes CSV with all outdated package columns including major/minor/patch versions
func TestWriteOutdatedResult_CSV(t *testing.T) {
	var buf bytes.Buffer
	result := &OutdatedResult{
		Packages: []OutdatedPackage{
			{
				Rule:             "npm",
				PM:               "js",
				Type:             "prod",
				Constraint:       "^",
				Version:          "4.17.0",
				InstalledVersion: "4.17.0",
				Major:            "5.0.0",
				Minor:            "4.18.0",
				Patch:            "#N/A",
				Status:           "Outdated",
				Group:            "core",
				Name:             "express",
				Error:            "",
			},
		},
	}

	err := WriteOutdatedResult(&buf, FormatCSV, result)
	require.NoError(t, err)

	output := buf.String()
	// Verify header
	assert.Contains(t, output, "RULE,PM,TYPE,CONSTRAINT,VERSION,INSTALLED,MAJOR,MINOR,PATCH,STATUS,GROUP,NAME,ERROR")
	// Verify data
	assert.Contains(t, output, "npm")
	assert.Contains(t, output, "express")
	assert.Contains(t, output, "5.0.0")
}

// TestWriteListResult_XML tests the behavior of WriteListResult with XML format.
//
// It verifies:
//   - Writes XML with listResult root element
func TestWriteListResult_XML(t *testing.T) {
	var buf bytes.Buffer
	result := &ListResult{
		Summary: ListSummary{
			TotalPackages: 1,
		},
		Packages: []ListPackage{
			{Rule: "npm", PM: "js", Type: "prod", Name: "express", Version: "4.18.0"},
		},
	}

	err := WriteListResult(&buf, FormatXML, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<?xml version=")
	assert.Contains(t, output, "<listResult>")
	assert.Contains(t, output, "<totalPackages>1</totalPackages>")
}

// TestWriteUpdateResult_XML tests the behavior of WriteUpdateResult with XML format.
//
// It verifies:
//   - Writes XML with updateResult root element
func TestWriteUpdateResult_XML(t *testing.T) {
	var buf bytes.Buffer
	result := &UpdateResult{
		Summary: UpdateSummary{
			TotalPackages:   1,
			UpdatedPackages: 1,
		},
		Packages: []UpdatePackage{
			{Rule: "npm", PM: "js", Name: "express", Version: "4.17.0", Target: "4.18.0", Status: "Updated"},
		},
	}

	err := WriteUpdateResult(&buf, FormatXML, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "<?xml version=")
	assert.Contains(t, output, "<updateResult>")
}
