package update

import (
	"fmt"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/formats"
	"gopkg.in/yaml.v3"
)

// yamlUnmarshalFunc is a variable that holds the yaml.Unmarshal function.
// This allows for dependency injection during testing.
var yamlUnmarshalFunc = yaml.Unmarshal

// updateYAMLVersion updates the version of a package in YAML manifest content.
//
// It performs the following operations:
//   - Step 1: Parse YAML content to validate structure
//   - Step 2: Unmarshal into map structure
//   - Step 3: Navigate to dependency fields and update package version
//   - Step 4: Marshal back to YAML format
//
// Parameters:
//   - content: The original YAML file content as bytes
//   - p: The package to update, containing name and constraint information
//   - ruleCfg: Package manager configuration with dependency field paths
//   - target: The target version to update to (without constraint prefix)
//
// Returns:
//   - []byte: Updated YAML content with proper formatting
//   - error: Returns error if YAML is invalid, package not found, or marshaling fails; returns nil on success
func updateYAMLVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
	parser := &formats.YAMLParser{}
	if _, err := parser.Parse(content, &ruleCfg); err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yamlUnmarshalFunc(content, &data); err != nil {
		return nil, err
	}

	found := false
	for field := range ruleCfg.Fields {
		node := formats.GetNestedField(data, field)
		deps, ok := node.(map[string]interface{})
		if !ok {
			continue
		}

		if _, ok := deps[p.Name]; ok {
			found = true
			deps[p.Name] = fmt.Sprintf("%s%s", p.Constraint, target)
		}
	}

	if !found {
		return nil, fmt.Errorf("package %s not found in %s", p.Name, p.Source)
	}

	return yaml.Marshal(data)
}
