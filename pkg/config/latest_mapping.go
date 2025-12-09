package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom YAML unmarshaling for LatestMappingCfg.
//
// This handles multiple YAML formats for latest_mapping configuration:
//   - Simple key-value: {latest: "*"}
//   - Sequences: {stable: ["1.x", "v1"]}
//   - Package-specific mappings
//
// Parameters:
//   - value: the YAML node to unmarshal
//
// Returns:
//   - error: error if YAML structure is invalid
func (l *LatestMappingCfg) UnmarshalYAML(value *yaml.Node) error {
	l.Default = make(map[string]string)
	l.Packages = make(map[string]map[string]string)

	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("latest_mapping must be a mapping")
	}

	for i := 0; i < len(value.Content); i += 2 {
		key := strings.TrimSpace(value.Content[i].Value)
		valNode := value.Content[i+1]

		switch key {
		case "default":
			if err := l.parseDefault(valNode); err != nil {
				return err
			}
		case "packages":
			if err := l.parsePackages(valNode); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown field %q in latest_mapping", key)
		}
	}

	return nil
}

// parseDefault parses the default section of latest_mapping.
func (l *LatestMappingCfg) parseDefault(node *yaml.Node) error {
	switch node.Kind {
	case yaml.MappingNode:
		for j := 0; j < len(node.Content); j += 2 {
			token := strings.TrimSpace(node.Content[j].Value)
			valNode := node.Content[j+1]
			if valNode.Kind != yaml.ScalarNode {
				return fmt.Errorf("default.%s must be a string", token)
			}
			l.Default[normalizeLatestKey(token)] = strings.TrimSpace(valNode.Value)
		}
	case yaml.SequenceNode:
		parsed, err := parseLatestSequence(node.Content)
		if err != nil {
			return fmt.Errorf("default: %w", err)
		}
		mergeLatestMap(l.Default, parsed)
	case yaml.ScalarNode:
		// Handle empty string case - just leave Default empty
		if strings.TrimSpace(node.Value) != "" {
			return fmt.Errorf("default must be a mapping or sequence")
		}
	default:
		return fmt.Errorf("default must be a mapping or sequence")
	}
	return nil
}

// parsePackages parses the packages section of latest_mapping.
func (l *LatestMappingCfg) parsePackages(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("packages must be a mapping")
	}

	for j := 0; j < len(node.Content); j += 2 {
		pkgName := strings.TrimSpace(node.Content[j].Value)
		valNode := node.Content[j+1]

		pkgMapping, err := l.parsePackageMapping(valNode, pkgName)
		if err != nil {
			return err
		}
		l.Packages[pkgName] = pkgMapping
	}
	return nil
}

// parsePackageMapping parses a single package's mapping configuration.
func (l *LatestMappingCfg) parsePackageMapping(node *yaml.Node, pkgName string) (map[string]string, error) {
	result := make(map[string]string)

	switch node.Kind {
	case yaml.MappingNode:
		for j := 0; j < len(node.Content); j += 2 {
			token := strings.TrimSpace(node.Content[j].Value)
			valNode := node.Content[j+1]
			if valNode.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("packages.%s.%s must be a string", pkgName, token)
			}
			result[normalizeLatestKey(token)] = strings.TrimSpace(valNode.Value)
		}
	case yaml.SequenceNode:
		parsed, err := parseLatestSequence(node.Content)
		if err != nil {
			return nil, fmt.Errorf("packages.%s: %w", pkgName, err)
		}
		mergeLatestMap(result, parsed)
	default:
		return nil, fmt.Errorf("packages.%s must be a mapping or sequence", pkgName)
	}

	return result, nil
}

// parseLatestSequence parses a sequence of tokens into a mapping.
// For sequences, all elements except the last are tokens (keys),
// and the last element is the pattern (value).
// If there's only one element, it's a token with an empty pattern.
func parseLatestSequence(content []*yaml.Node) (map[string]string, error) {
	result := make(map[string]string)

	if len(content) == 0 {
		return result, nil
	}

	// Validate all items are scalars
	for _, item := range content {
		if item.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("sequences must contain strings")
		}
	}

	if len(content) == 1 {
		// Single element: token with empty pattern
		result[normalizeLatestKey(content[0].Value)] = ""
	} else {
		// Multiple elements: last is pattern, others are tokens
		pattern := strings.TrimSpace(content[len(content)-1].Value)
		for _, item := range content[:len(content)-1] {
			result[normalizeLatestKey(item.Value)] = pattern
		}
	}

	return result, nil
}

// mergeLatestMap merges source mappings into destination.
//
// This copies all entries from src to dest, overwriting existing keys.
//
// Parameters:
//   - dest: the destination map to merge into
//   - src: the source map to merge from
func mergeLatestMap(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

// mergeLatestMappingCfg merges two latest mapping configurations.
//
// This creates a new configuration with mappings from both base and override.
// Override mappings take precedence for conflicting keys.
//
// Parameters:
//   - base: the base latest mapping configuration
//   - override: the override latest mapping configuration
//
// Returns:
//   - *LatestMappingCfg: merged configuration, or nil if both inputs are nil
func mergeLatestMappingCfg(base, override *LatestMappingCfg) *LatestMappingCfg {
	if base == nil && override == nil {
		return nil
	}

	merged := &LatestMappingCfg{
		Default:  make(map[string]string),
		Packages: make(map[string]map[string]string),
	}

	if base != nil {
		mergeLatestMap(merged.Default, base.Default)
		for name, mapping := range base.Packages {
			merged.Packages[name] = make(map[string]string)
			mergeLatestMap(merged.Packages[name], mapping)
		}
	}

	if override != nil {
		mergeLatestMap(merged.Default, override.Default)
		for name, mapping := range override.Packages {
			if existing, ok := merged.Packages[name]; ok {
				mergeLatestMap(existing, mapping)
			} else {
				merged.Packages[name] = make(map[string]string)
				mergeLatestMap(merged.Packages[name], mapping)
			}
		}
	}

	if len(merged.Default) == 0 {
		merged.Default = nil
	}
	if len(merged.Packages) == 0 {
		merged.Packages = nil
	}

	return merged
}

// normalizeLatestKey normalizes a version token key.
//
// This converts keys to lowercase and trims whitespace for consistent
// case-insensitive matching.
//
// Parameters:
//   - key: the key to normalize
//
// Returns:
//   - string: normalized key (lowercase, trimmed)
func normalizeLatestKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
