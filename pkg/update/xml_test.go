package update

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/stretchr/testify/require"
)

// TestUpdateXMLVersionMissingPackage tests error handling for missing packages.
//
// It verifies:
//   - Returns error when attempting to update a package not present in the manifest
func TestUpdateXMLVersionMissingPackage(t *testing.T) {
	tmpDir := t.TempDir()
	msbuild := filepath.Join(tmpDir, "proj.msbuild")
	require.NoError(t, os.WriteFile(msbuild, []byte(`<Project><ItemGroup><PackageReference Include="Other" Version="1.0.0" /></ItemGroup></Project>`), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"xml": {Manager: "dotnet", Format: "xml", Fields: map[string]string{"ItemGroup/PackageReference": "prod"}, Update: &config.UpdateCfg{}},
	}}

	err := UpdatePackage(formats.Package{Name: "Newtonsoft.Json", Rule: "xml", Source: msbuild}, "1.2.0", cfg, tmpDir, false, true)
	require.Error(t, err)
}

// TestUpdateXMLVersionUnmarshalError tests error handling when XML unmarshaling fails.
//
// It verifies:
//   - Returns error for invalid XML syntax
func TestUpdateXMLVersionUnmarshalError(t *testing.T) {
	cfg := config.PackageManagerCfg{Format: "xml", Fields: map[string]string{"ItemGroup/PackageReference": "prod"}}
	_, err := updateXMLVersion([]byte("not xml at all <"), formats.Package{Name: "demo", Source: "proj.csproj"}, cfg, "1.1.0")
	require.Error(t, err)
}

// TestUpdateXMLVersionMarshalError tests error handling when XML marshaling fails.
//
// It verifies:
//   - Returns error when XML marshaling fails after updates are applied
func TestUpdateXMLVersionMarshalError(t *testing.T) {
	// Test when xml.MarshalIndent fails - hard to trigger with valid XMLNode
	// This tests the error path when XML marshaling fails
	tmpDir := t.TempDir()
	xmlPath := filepath.Join(tmpDir, "test.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(`<Project><ItemGroup><PackageReference Include="Demo" Version="1.0.0" /></ItemGroup></Project>`), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"xml": {Manager: "dotnet", Format: "xml", Fields: map[string]string{"ItemGroup/PackageReference": "prod"}, Update: &config.UpdateCfg{}},
	}}

	// Mock xmlMarshalIndentFunc to return an error
	originalMarshal := xmlMarshalIndentFunc
	xmlMarshalIndentFunc = func(v interface{}, prefix, indent string) ([]byte, error) {
		return nil, errors.New("marshal error")
	}
	t.Cleanup(func() { xmlMarshalIndentFunc = originalMarshal })

	err := UpdatePackage(formats.Package{Name: "Demo", Rule: "xml", Source: xmlPath}, "2.0.0", cfg, tmpDir, false, true)
	require.Error(t, err)
}
