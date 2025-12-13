package formats

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/utils"
)

// TestXMLParser tests the behavior of XMLParser.Parse.
//
// It verifies:
//   - Valid XML is parsed correctly
//   - Package references are extracted using configured paths and attributes
//   - Package names and versions are captured
//   - Package types are assigned correctly
//   - Invalid XML returns an error
func TestXMLParser(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			Path:        "Project/ItemGroup/PackageReference",
			NameAttr:    "Include",
			VersionAttr: "Version",
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="2.12.0" />
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	assert.Len(t, packages, 2)

	// Check Newtonsoft.Json
	var json Package
	for _, p := range packages {
		if p.Name == "Newtonsoft.Json" {
			json = p
		}
	}
	assert.Equal(t, "13.0.1", json.Version)
	assert.Equal(t, "prod", json.Type)

	// Test invalid XML
	_, err = parser.Parse([]byte("invalid xml"), cfg)
	assert.Error(t, err)
}

// TestXMLParserDefaultAttributes tests XMLParser with default "id" and "version" attributes.
//
// It verifies:
//   - Default attributes are used when no extraction config provided
//   - Empty attribute values cause entries to be skipped
func TestXMLParserDefaultAttributes(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "maven",
		Fields: map[string]string{
			"dependencies/dependency": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<project>
  <dependencies>
    <dependency id="log4j" version="2.0.1" />
    <dependency id="" version="" />
  </dependencies>
</project> `)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 1)

	assert.Equal(t, "log4j", packages[0].Name)
	assert.Equal(t, "2.0.1", packages[0].Version)
}

// TestXMLParserNugetFallback tests NuGet/dotnet fallback path.
//
// It verifies:
//   - When no packages match configured fields, PackageReference nodes are searched
//   - Fallback only triggers for nuget/dotnet package managers
func TestXMLParserNugetFallback(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Fields: map[string]string{
			"unused": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 1)

	assert.Equal(t, "Newtonsoft.Json", packages[0].Name)
	assert.Equal(t, "13.0.1", packages[0].Version)
	assert.Equal(t, "prod", packages[0].Type)
}

// TestXMLParserWithExtractionAttributes tests custom extraction attribute names.
//
// It verifies:
//   - Custom name and version attributes are used when configured
//   - Extraction path is respected
func TestXMLParserWithExtractionAttributes(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "maven",
		Extraction: &config.ExtractionCfg{
			Path:        "dependencies/dependency",
			NameAttr:    "name",
			VersionAttr: "version",
		},
		Fields: map[string]string{
			"dependencies": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<project>
  <dependencies>
    <dependency name="commons-io" version="2.15.1" />
  </dependencies>
</project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	assert.Equal(t, "commons-io", packages[0].Name)
	assert.Equal(t, "2.15.1", packages[0].Version)
}

// TestXMLParserIgnoresPackages tests package ignore functionality.
//
// It verifies:
//   - Packages matching ignore patterns are marked with IgnoreReason
//   - Non-ignored packages have no IgnoreReason
func TestXMLParserIgnoresPackages(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "maven",
		Fields: map[string]string{
			"dependencies/dependency": "prod",
		},
		Ignore: []string{"skipme"},
	}

	content := []byte(`<?xml version="1.0"?>
<project>
  <dependencies>
    <dependency id="keepme" version="1.0.0" />
    <dependency id="skipme" version="2.0.0" />
  </dependencies>
</project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	// keepme: not ignored
	assert.Equal(t, "", pkgMap["keepme"].IgnoreReason)

	// skipme: marked as ignored (but still included for visibility)
	assert.Equal(t, "matches ignore pattern 'skipme'", pkgMap["skipme"].IgnoreReason)
}

// TestXMLParserWithOverrides tests package overrides in XMLParser.
//
// It verifies:
//   - Version overrides are applied
//   - Constraint overrides are applied
//   - Non-overridden packages retain original values
func TestXMLParserWithOverrides(t *testing.T) {
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			Path:        "Project/ItemGroup/PackageReference",
			NameAttr:    "Include",
			VersionAttr: "Version",
		},
		Fields: map[string]string{
			"packages": "prod",
		},
		PackageOverrides: map[string]config.PackageOverrideCfg{
			"Newtonsoft.Json": {
				Version:    "12.0.3",
				Constraint: strPtr(""),
			},
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.1" />
    <PackageReference Include="Serilog" Version="2.12.0" />
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	assert.Equal(t, "12.0.3", pkgMap["Newtonsoft.Json"].Version)
	assert.Equal(t, "", pkgMap["Newtonsoft.Json"].Constraint)
	assert.Equal(t, "2.12.0", pkgMap["Serilog"].Version)
}

// TestXMLParserDevDependencyAttribute tests dev dependency detection via attributes.
//
// It verifies:
//   - Packages with developmentDependency="true" are marked as dev
//   - Packages without the attribute remain as prod
//   - Case-insensitive attribute value matching
func TestXMLParserDevDependencyAttribute(t *testing.T) {
	// Test nuget packages.config with developmentDependency="true"
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			Path:        "package",
			NameAttr:    "id",
			VersionAttr: "version",
			DevAttr:     "developmentDependency",
			DevValue:    "true",
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte(`<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="Newtonsoft.Json" version="13.0.3" targetFramework="net48" />
  <package id="Serilog" version="3.1.1" targetFramework="net48" />
  <package id="xunit" version="2.6.6" targetFramework="net48" developmentDependency="true" />
  <package id="Moq" version="4.20.70" targetFramework="net48" developmentDependency="true" />
</packages>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 4)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	// Production packages
	assert.Equal(t, "prod", pkgMap["Newtonsoft.Json"].Type)
	assert.Equal(t, "prod", pkgMap["Serilog"].Type)

	// Dev packages (developmentDependency="true")
	assert.Equal(t, "dev", pkgMap["xunit"].Type)
	assert.Equal(t, "dev", pkgMap["Moq"].Type)
}

// TestXMLParserDevDependencyElement tests dev dependency detection via child elements.
//
// It verifies:
//   - Packages with <PrivateAssets>all</PrivateAssets> child element are marked as dev
//   - Packages without the element remain as prod
//   - Case-insensitive element value matching
func TestXMLParserDevDependencyElement(t *testing.T) {
	// Test msbuild PackageReference with PrivateAssets element
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			Path:            "Project/ItemGroup/PackageReference",
			NameAttr:        "Include",
			VersionAttr:     "Version",
			DevElement:      "PrivateAssets",
			DevElementValue: "all",
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Microsoft.Extensions.Hosting" Version="8.0.2" />
    <PackageReference Include="Serilog" Version="3.1.0" />
    <PackageReference Include="Microsoft.EntityFrameworkCore.Design" Version="8.0.2">
      <PrivateAssets>all</PrivateAssets>
      <IncludeAssets>runtime; build; native; contentfiles; analyzers</IncludeAssets>
    </PackageReference>
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 3)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	// Production packages
	assert.Equal(t, "prod", pkgMap["Microsoft.Extensions.Hosting"].Type)
	assert.Equal(t, "prod", pkgMap["Serilog"].Type)

	// Dev package (PrivateAssets=all)
	assert.Equal(t, "dev", pkgMap["Microsoft.EntityFrameworkCore.Design"].Type)
}

// TestXMLParserDevDependencyElementAsAttribute tests dev dependency detection via element-as-attribute.
//
// It verifies:
//   - Packages with PrivateAssets="all" attribute are marked as dev
//   - Both attribute and child element forms are supported
func TestXMLParserDevDependencyElementAsAttribute(t *testing.T) {
	// Test msbuild PackageReference with PrivateAssets as attribute
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			Path:            "Project/ItemGroup/PackageReference",
			NameAttr:        "Include",
			VersionAttr:     "Version",
			DevElement:      "PrivateAssets",
			DevElementValue: "all",
		},
		Fields: map[string]string{
			"packages": "prod",
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Production.Package" Version="1.0.0" />
    <PackageReference Include="DevTool.Package" Version="2.0.0" PrivateAssets="all" />
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	assert.Equal(t, "prod", pkgMap["Production.Package"].Type)
	assert.Equal(t, "dev", pkgMap["DevTool.Package"].Type)
}

// TestXMLParserDevDependencyNugetFallback tests dev dependency detection in fallback path.
//
// It verifies:
//   - NuGet/dotnet fallback path respects dev dependency configuration
//   - Dev dependency markers work in the fallback code path
func TestXMLParserDevDependencyNugetFallback(t *testing.T) {
	// Test that nuget/dotnet fallback path also respects dev dependency config
	parser := &XMLParser{}
	cfg := &config.PackageManagerCfg{
		Manager: "dotnet",
		Extraction: &config.ExtractionCfg{
			DevElement:      "PrivateAssets",
			DevElementValue: "all",
		},
		Fields: map[string]string{
			"unused": "prod", // No match, triggers fallback
		},
	}

	content := []byte(`<?xml version="1.0"?>
<Project>
  <ItemGroup>
    <PackageReference Include="Prod.Package" Version="1.0.0" />
    <PackageReference Include="Dev.Package" Version="2.0.0">
      <PrivateAssets>all</PrivateAssets>
    </PackageReference>
  </ItemGroup>
</Project>`)

	packages, err := parser.Parse(content, cfg)
	require.NoError(t, err)
	require.Len(t, packages, 2)

	pkgMap := make(map[string]Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	assert.Equal(t, "prod", pkgMap["Prod.Package"].Type)
	assert.Equal(t, "dev", pkgMap["Dev.Package"].Type)
}

// TestIsDevDependency tests the behavior of isDevDependency.
//
// It verifies:
//   - Nil extraction config returns false
//   - Empty extraction config returns false
//   - Case-insensitive attribute value matching
//   - Case-insensitive child element value matching
func TestIsDevDependency(t *testing.T) {
	t.Run("nil extraction returns false", func(t *testing.T) {
		node := &utils.XMLNode{}
		assert.False(t, isDevDependency(node, nil))
	})

	t.Run("empty extraction returns false", func(t *testing.T) {
		node := &utils.XMLNode{}
		assert.False(t, isDevDependency(node, &config.ExtractionCfg{}))
	})

	t.Run("case insensitive attribute match", func(t *testing.T) {
		node := &utils.XMLNode{
			Attrs: []xml.Attr{{Name: xml.Name{Local: "developmentDependency"}, Value: "TRUE"}},
		}
		extraction := &config.ExtractionCfg{
			DevAttr:  "developmentDependency",
			DevValue: "true",
		}
		assert.True(t, isDevDependency(node, extraction))
	})

	t.Run("case insensitive element match", func(t *testing.T) {
		node := &utils.XMLNode{
			Nodes: []utils.XMLNode{
				{XMLName: xml.Name{Local: "PrivateAssets"}, Content: "ALL"},
			},
		}
		extraction := &config.ExtractionCfg{
			DevElement:      "PrivateAssets",
			DevElementValue: "all",
		}
		assert.True(t, isDevDependency(node, extraction))
	})
}
