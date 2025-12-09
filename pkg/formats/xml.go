package formats

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/utils"
)

// XMLParser parses XML package files (e.g., .csproj, packages.config).
//
// It supports various XML-based package managers including NuGet, .NET, and Maven,
// with configurable extraction paths and attribute mappings.
type XMLParser struct{}

// Parse parses XML content and extracts package dependencies.
//
// It performs the following operations:
//   - Unmarshals the XML content into a node tree
//   - Searches for package nodes using XPath-like patterns from the configuration
//   - Extracts package names and versions from XML attributes
//   - Applies version parsing, constraints, and package overrides
//   - Identifies dev dependencies based on configured markers
//   - Falls back to PackageReference nodes for .NET/NuGet projects
//
// Parameters:
//   - content: The raw bytes of the XML package manifest file
//   - cfg: The package manager configuration with extraction rules and field mappings
//
// Returns:
//   - []Package: A list of parsed packages with names, versions, and dependency types
//   - error: Returns an error if the XML is invalid; returns nil on successful parse
func (p *XMLParser) Parse(content []byte, cfg *config.PackageManagerCfg) ([]Package, error) {
	var root utils.XMLNode
	if err := xml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("invalid XML: %w", err)
	}

	var packages []Package

	for field, pkgType := range cfg.Fields {
		var nodes []*utils.XMLNode
		if cfg.Extraction != nil && cfg.Extraction.Path != "" {
			nodes = utils.FindXMLNodes(&root, cfg.Extraction.Path)
		} else {
			nodes = utils.FindXMLNodes(&root, field)
		}

		for _, node := range nodes {
			var name, version string

			if cfg.Extraction != nil {
				if cfg.Extraction.NameAttr != "" {
					name = utils.GetXMLAttr(node, cfg.Extraction.NameAttr)
				}
				if cfg.Extraction.VersionAttr != "" {
					version = utils.GetXMLAttr(node, cfg.Extraction.VersionAttr)
				}
			} else {
				name = utils.GetXMLAttr(node, "id")
				version = utils.GetXMLAttr(node, "version")
			}

			if name == "" || version == "" {
				continue
			}

			if shouldIgnorePackage(name, cfg) {
				continue
			}

			vInfo := utils.ParseVersion(version)

			// Apply package-specific overrides
			vInfo = utils.ApplyPackageOverride(name, vInfo, cfg)

			vInfo = utils.NormalizeDeclaredVersion(name, vInfo, cfg)

			// Determine package type - check for dev dependency markers
			finalType := pkgType
			if cfg.Extraction != nil && isDevDependency(node, cfg.Extraction) {
				finalType = "dev"
			}

			packages = append(packages, Package{
				Name:        name,
				Version:     vInfo.Version,
				Constraint:  vInfo.Constraint,
				Type:        finalType,
				PackageType: cfg.Manager,
			})
		}
	}

	if len(packages) == 0 && (cfg.Manager == "nuget" || cfg.Manager == "dotnet") {
		packageRefs := utils.FindXMLNodes(&root, "ItemGroup/PackageReference")
		for _, ref := range packageRefs {
			name := utils.GetXMLAttr(ref, "Include")
			version := utils.GetXMLAttr(ref, "Version")

			if name != "" && version != "" && !shouldIgnorePackage(name, cfg) {
				vInfo := utils.ParseVersion(version)

				// Apply package-specific overrides
				vInfo = utils.ApplyPackageOverride(name, vInfo, cfg)

				vInfo = utils.NormalizeDeclaredVersion(name, vInfo, cfg)

				// Determine package type - check for dev dependency markers
				pkgType := "prod"
				if cfg.Extraction != nil && isDevDependency(ref, cfg.Extraction) {
					pkgType = "dev"
				}

				packages = append(packages, Package{
					Name:        name,
					Version:     vInfo.Version,
					Constraint:  vInfo.Constraint,
					Type:        pkgType,
					PackageType: cfg.Manager,
				})
			}
		}
	}

	return packages, nil
}

// isDevDependency checks if an XML node represents a dev dependency based on extraction config.
//
// It performs the following checks:
//   - Checks for dev attributes (e.g., developmentDependency="true" for NuGet packages.config)
//   - Checks for dev elements (e.g., <PrivateAssets>all</PrivateAssets> for MSBuild)
//   - Supports both child element and attribute forms of dev markers
//   - Performs case-insensitive value matching
//
// Parameters:
//   - node: The XML node to check
//   - extraction: The extraction configuration containing dev dependency markers
//
// Returns:
//   - bool: true if the node represents a dev dependency; false otherwise
func isDevDependency(node *utils.XMLNode, extraction *config.ExtractionCfg) bool {
	if extraction == nil {
		return false
	}

	// Check dev attribute (e.g., developmentDependency="true" for nuget packages.config)
	if extraction.DevAttr != "" && extraction.DevValue != "" {
		attrValue := utils.GetXMLAttr(node, extraction.DevAttr)
		if strings.EqualFold(attrValue, extraction.DevValue) {
			return true
		}
	}

	// Check dev element (e.g., <PrivateAssets>all</PrivateAssets> for msbuild)
	if extraction.DevElement != "" {
		// First check as attribute (PrivateAssets="all")
		attrValue := utils.GetXMLAttr(node, extraction.DevElement)
		if attrValue != "" {
			if extraction.DevElementValue == "" || strings.EqualFold(attrValue, extraction.DevElementValue) {
				return true
			}
		}

		// Then check as child element
		for i := range node.Nodes {
			child := &node.Nodes[i]
			if child.XMLName.Local == extraction.DevElement {
				childText := strings.TrimSpace(child.Content)
				if extraction.DevElementValue == "" || strings.EqualFold(childText, extraction.DevElementValue) {
					return true
				}
			}
		}
	}

	return false
}
