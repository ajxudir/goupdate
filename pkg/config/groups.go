package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ajxudir/goupdate/pkg/verbose"
	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom YAML unmarshaling for GroupCfg.
//
// This allows groups to be specified in two formats:
//   - Simple list: ["pkg1", "pkg2"]
//   - Map with settings: {with_all_dependencies: true, packages: ["pkg1", "pkg2"]}
//
// Parameters:
//   - value: the YAML node to unmarshal
//
// Returns:
//   - error: error if YAML structure is invalid
func (g *GroupCfg) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.SequenceNode:
		packages, err := parseGroupSequence(value.Content)
		if err != nil {
			return err
		}
		g.Packages = packages
		return nil
	case yaml.MappingNode:
		if len(value.Content)%2 != 0 {
			return fmt.Errorf("group mapping entries must be key/value pairs")
		}

		packages := make([]string, 0)
		for i := 0; i < len(value.Content); i += 2 {
			key := strings.TrimSpace(value.Content[i].Value)
			node := value.Content[i+1]

			switch key {
			case "packages", "members":
				if node.Kind != yaml.SequenceNode {
					return fmt.Errorf("group %s must be a sequence", key)
				}
				parsed, err := parseGroupSequence(node.Content)
				if err != nil {
					return err
				}
				packages = append(packages, parsed...)
			case "with_all_dependencies":
				if node.Kind == yaml.ScalarNode {
					g.WithAllDependencies = node.Value == "true"
				}
			default:
				return fmt.Errorf("unsupported group key %q", key)
			}
		}

		g.Packages = packages
		return nil
	default:
		return fmt.Errorf("group configuration must be a sequence or map")
	}
}

// parseGroupSequence parses a YAML sequence into a list of package names.
//
// This handles both simple string entries and map entries with a "name" field.
// Empty strings are skipped.
//
// Parameters:
//   - nodes: YAML nodes representing the sequence items
//
// Returns:
//   - []string: list of package names
//   - error: error if a node has invalid structure or missing name
func parseGroupSequence(nodes []*yaml.Node) ([]string, error) {
	packages := make([]string, 0, len(nodes))

	for _, item := range nodes {
		switch item.Kind {
		case yaml.ScalarNode:
			name := strings.TrimSpace(item.Value)
			if name == "" {
				continue
			}
			packages = append(packages, name)
		case yaml.MappingNode:
			var alias struct {
				Name string `yaml:"name"`
			}

			if err := item.Decode(&alias); err != nil {
				return nil, fmt.Errorf("failed to decode group member: %w", err)
			}

			name := strings.TrimSpace(alias.Name)
			if name == "" {
				return nil, fmt.Errorf("group member is missing a name")
			}

			packages = append(packages, name)
		default:
			return nil, fmt.Errorf("group entries must be package names")
		}
	}

	return packages, nil
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
	verbose.Debugf("Group validation: checking for packages in multiple groups")
	for ruleName, rule := range cfg.Rules {
		if len(rule.Groups) == 0 {
			continue
		}

		verbose.Tracef("Group validation: rule %q has %d groups", ruleName, len(rule.Groups))
		packages := make(map[string]map[string]struct{})
		for groupName, group := range rule.Groups {
			verbose.Tracef("Group validation: group %q has %d packages", groupName, len(group.Packages))
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

			verbose.Printf("Group validation ERROR: package %q is in multiple groups: %v\n", pkg, groupNames)
			conflicts = append(conflicts, fmt.Sprintf("%s (%s)", pkg, strings.Join(groupNames, ", ")))
		}

		if len(conflicts) > 0 {
			sort.Strings(conflicts)
			return fmt.Errorf("rule %s has packages assigned to multiple groups: %s", ruleName, strings.Join(conflicts, "; "))
		}
	}

	verbose.Debugf("Group validation: passed - no conflicts found")
	return nil
}
