package update

import (
	"encoding/xml"
	"fmt"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
	"github.com/user/goupdate/pkg/utils"
)

// xmlMarshalIndentFunc is a variable that holds the xml.MarshalIndent function.
// This allows for dependency injection during testing.
var xmlMarshalIndentFunc = xml.MarshalIndent

// updateXMLVersion updates the version of a package in XML manifest content.
//
// It performs the following operations:
//   - Step 1: Unmarshal XML content into a structured node tree
//   - Step 2: Locate package nodes matching the target package name
//   - Step 3: Update version attributes for matching nodes
//   - Step 4: Marshal the updated tree back to XML format
//
// Parameters:
//   - content: The original XML file content as bytes
//   - p: The package to update, containing name and constraint information
//   - ruleCfg: Package manager configuration with field paths and extraction rules
//   - target: The target version to update to (without constraint prefix)
//
// Returns:
//   - []byte: Updated XML content with proper indentation
//   - error: Returns error if XML is invalid, package not found, or marshaling fails; returns nil on success
func updateXMLVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
	var root utils.XMLNode
	if err := xml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("invalid XML: %w", err)
	}

	updated := false
	versionValue := fmt.Sprintf("%s%s", p.Constraint, target)

	updateNodes := func(nodes []*utils.XMLNode, nameAttr, versionAttr string) {
		for _, node := range nodes {
			if utils.GetXMLAttr(node, nameAttr) != p.Name {
				continue
			}

			for i := range node.Attrs {
				if node.Attrs[i].Name.Local == versionAttr {
					node.Attrs[i].Value = versionValue
					updated = true
					break
				}
			}
		}
	}

	for field := range ruleCfg.Fields {
		nameAttr := "id"
		versionAttr := "version"
		nodes := utils.FindXMLNodes(&root, field)
		if ruleCfg.Extraction != nil {
			if ruleCfg.Extraction.Path != "" {
				nodes = utils.FindXMLNodes(&root, ruleCfg.Extraction.Path)
			}
			if ruleCfg.Extraction.NameAttr != "" {
				nameAttr = ruleCfg.Extraction.NameAttr
			}
			if ruleCfg.Extraction.VersionAttr != "" {
				versionAttr = ruleCfg.Extraction.VersionAttr
			}
		}

		if (ruleCfg.Manager == "nuget" || ruleCfg.Manager == "dotnet") && ruleCfg.Extraction == nil {
			nameAttr = "Include"
			versionAttr = "Version"
		}

		updateNodes(nodes, nameAttr, versionAttr)
	}

	if ruleCfg.Manager == "nuget" || ruleCfg.Manager == "dotnet" {
		refs := utils.FindXMLNodes(&root, "ItemGroup/PackageReference")
		updateNodes(refs, "Include", "Version")
	}

	if !updated {
		return nil, fmt.Errorf("package %s not found in %s", p.Name, p.Source)
	}

	output, err := xmlMarshalIndentFunc(root, "", "  ")
	if err != nil {
		return nil, err
	}

	return output, nil
}
