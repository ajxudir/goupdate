package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom YAML unmarshaling for GroupCfg.
//
// This allows groups to be specified in multiple formats:
//   - Simple list: ["pkg1", "pkg2"]
//   - Map with packages: {packages: ["pkg1", "pkg2"]}
//   - Map with settings: {with_all_dependencies: true, packages: ["pkg1", "pkg2"]}
//   - Map with per-package settings: {packages: [{name: "pkg1", with_all_dependencies: true}]}
//
// Parameters:
//   - value: the YAML node to unmarshal
//
// Returns:
//   - error: error if YAML structure is invalid
func (g *GroupCfg) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		packages, pkgSettings, err := parseGroupSequenceWithSettings(value.Content)
		if err != nil {
			return err
		}
		g.Packages = packages
		g.PackageSettings = pkgSettings
		return nil
	case yaml.MappingNode:
		if len(value.Content)%2 != 0 {
			return fmt.Errorf("group mapping entries must be key/value pairs")
		}

		packages := make([]string, 0)
		pkgSettings := make(map[string]PackageSettings)
		for i := 0; i < len(value.Content); i += 2 {
			key := strings.TrimSpace(value.Content[i].Value)
			node := value.Content[i+1]

			switch key {
			case "packages", "members":
				if node.Kind != yaml.SequenceNode {
					return fmt.Errorf("group %s must be a sequence", key)
				}
				parsed, settings, err := parseGroupSequenceWithSettings(node.Content)
				if err != nil {
					return err
				}
				packages = append(packages, parsed...)
				for k, v := range settings {
					pkgSettings[k] = v
				}
			case "with_all_dependencies":
				if node.Kind == yaml.ScalarNode {
					g.WithAllDependencies = node.Value == "true"
				}
			default:
				return fmt.Errorf("unsupported group key %q", key)
			}
		}

		g.Packages = packages
		g.PackageSettings = pkgSettings
		return nil
	default:
		return fmt.Errorf("group configuration must be a sequence or map")
	}
}

// parseGroupSequenceWithSettings parses a YAML sequence into package names and settings.
//
// This handles multiple entry formats:
//   - Simple string: "pkg1"
//   - Map with name only: {name: "pkg1"}
//   - Map with settings: {name: "pkg1", with_all_dependencies: true}
//
// Parameters:
//   - nodes: YAML nodes representing the sequence items
//
// Returns:
//   - []string: list of package names
//   - map[string]PackageSettings: per-package settings (may be empty)
//   - error: error if a node has invalid structure or missing name
func parseGroupSequenceWithSettings(nodes []*yaml.Node) ([]string, map[string]PackageSettings, error) {
	packages := make([]string, 0, len(nodes))
	pkgSettings := make(map[string]PackageSettings)

	for _, item := range nodes {
		switch item.Kind {
		case yaml.ScalarNode:
			name := strings.TrimSpace(item.Value)
			if name == "" {
				continue
			}
			packages = append(packages, name)
		case yaml.MappingNode:
			var entry struct {
				Name                string `yaml:"name"`
				WithAllDependencies bool   `yaml:"with_all_dependencies"`
			}

			if err := item.Decode(&entry); err != nil {
				return nil, nil, fmt.Errorf("failed to decode group member: %w", err)
			}

			name := strings.TrimSpace(entry.Name)
			if name == "" {
				return nil, nil, fmt.Errorf("group member is missing a name")
			}

			packages = append(packages, name)

			// Store settings if any non-default values
			if entry.WithAllDependencies {
				pkgSettings[name] = PackageSettings{
					WithAllDependencies: entry.WithAllDependencies,
				}
			}
		default:
			return nil, nil, fmt.Errorf("group entries must be package names")
		}
	}

	return packages, pkgSettings, nil
}

// validateGroupMembership validates that packages are not assigned to multiple groups.
//
// This checks all rules in the configuration to ensure each package appears
// in at most one group per rule. Packages in multiple groups would create
// ambiguity during group updates.
//
// Parameters:
//   - cfg: the configuration to validate
//
// Returns:
//   - error: error if any package is assigned to multiple groups
func validateGroupMembership(cfg *Config) error {
	for ruleName, rule := range cfg.Rules {
		if len(rule.Groups) == 0 {
			continue
		}

		packages := make(map[string]map[string]struct{})
		for groupName, group := range rule.Groups {
			for _, pkg := range group.Packages {
				name := strings.TrimSpace(pkg)
				if name == "" {
					continue
				}

				if _, exists := packages[name]; !exists {
					packages[name] = make(map[string]struct{})
				}

				packages[name][groupName] = struct{}{}
			}
		}

		conflicts := make([]string, 0)
		for pkg, groups := range packages {
			if len(groups) < 2 {
				continue
			}

			groupNames := make([]string, 0, len(groups))
			for group := range groups {
				groupNames = append(groupNames, group)
			}
			sort.Strings(groupNames)

			conflicts = append(conflicts, fmt.Sprintf("%s (%s)", pkg, strings.Join(groupNames, ", ")))
		}

		if len(conflicts) > 0 {
			sort.Strings(conflicts)
			return fmt.Errorf("rule %s has packages assigned to multiple groups: %s", ruleName, strings.Join(conflicts, "; "))
		}
	}

	return nil
}
