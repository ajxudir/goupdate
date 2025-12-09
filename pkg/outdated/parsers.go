package outdated

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ajxudir/goupdate/pkg/config"
)

// UTF-8 BOM bytes (EF BB BF)
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// stripBOM removes UTF-8 BOM from the beginning of output if present.
//
// The UTF-8 BOM (Byte Order Mark) is a sequence of bytes (EF BB BF) that some
// tools add to the beginning of text output. This function detects and removes it.
//
// Parameters:
//   - output: Raw bytes that may start with a UTF-8 BOM
//
// Returns:
//   - []byte: The output with BOM removed if present, or unchanged output otherwise
func stripBOM(output []byte) []byte {
	if bytes.HasPrefix(output, utf8BOM) {
		return output[len(utf8BOM):]
	}
	return output
}

// parseAvailableVersionsForPackage parses command output to extract available versions.
//
// It uses the Format field to determine parsing strategy and Extraction for configuration.
// Supports JSON, YAML, and raw text formats with customizable extraction rules.
//
// Parameters:
//   - pkgName: The package name being processed (currently unused but reserved for future use)
//   - cfg: Outdated configuration containing format type and extraction rules
//   - output: Raw command output bytes to parse
//
// Returns:
//   - []string: List of extracted version strings
//   - error: When config is nil, format is unsupported, or parsing fails; returns nil on success
func parseAvailableVersionsForPackage(pkgName string, cfg *config.OutdatedCfg, output []byte) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("outdated configuration is required")
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = "json" // Default to JSON for most package registries
	}

	switch format {
	case "json":
		return parseJSONWithExtraction(cfg.Extraction, output)
	case "yaml":
		return parseYAMLWithExtraction(cfg.Extraction, output)
	case "raw":
		return parseRawWithExtraction(cfg.Extraction, output)
	default:
		return nil, fmt.Errorf("unsupported output format: %s (supported: json, yaml, raw)", format)
	}
}

// parseJSONWithExtraction extracts versions from JSON output using extraction config.
//
// This is a convenience wrapper around parseJSONVersions that extracts the JSON key
// from the extraction configuration.
//
// Parameters:
//   - extraction: Extraction configuration containing the JSONKey path; may be nil
//   - output: Raw JSON bytes to parse
//
// Returns:
//   - []string: Extracted version strings
//   - error: When JSON parsing fails; returns nil on success
func parseJSONWithExtraction(extraction *config.OutdatedExtractionCfg, output []byte) ([]string, error) {
	key := ""
	if extraction != nil {
		key = extraction.JSONKey
	}
	return parseJSONVersions(key, output)
}

// parseYAMLWithExtraction extracts versions from YAML output using extraction config.
//
// It performs the following operations:
//   - Step 1: Strips UTF-8 BOM if present
//   - Step 2: Unmarshals YAML into a generic structure
//   - Step 3: Navigates to the specified key using dot notation
//   - Step 4: Extracts versions from the target node
//
// Parameters:
//   - extraction: Extraction configuration containing the YAMLKey path; may be nil
//   - output: Raw YAML bytes to parse
//
// Returns:
//   - []string: Extracted version strings
//   - error: When YAML parsing fails or key path is invalid; returns nil on success
func parseYAMLWithExtraction(extraction *config.OutdatedExtractionCfg, output []byte) ([]string, error) {
	// Strip BOM if present
	output = stripBOM(output)

	var data interface{}
	if err := yaml.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML output: %w", err)
	}

	key := ""
	if extraction != nil {
		key = extraction.YAMLKey
	}

	node := data
	if key != "" {
		parts := strings.Split(key, ".")
		for _, part := range parts {
			switch v := node.(type) {
			case map[string]interface{}:
				node = v[part]
			case map[interface{}]interface{}:
				node = v[part]
			default:
				return nil, fmt.Errorf("yaml key %s not found", key)
			}
		}
	}

	return extractVersionsFromNode(node)
}

// extractVersionsFromNode converts a YAML/JSON node to a list of version strings.
//
// It handles multiple node types:
//   - []interface{}: Converts each element to string
//   - []string: Returns as-is
//   - string: Returns as single-element slice
//
// Parameters:
//   - node: The parsed YAML/JSON node to extract versions from
//
// Returns:
//   - []string: Extracted version strings with empty strings filtered out
//   - error: When node type is not supported; returns nil on success
func extractVersionsFromNode(node interface{}) ([]string, error) {
	switch v := node.(type) {
	case []interface{}:
		versions := make([]string, 0, len(v))
		for _, entry := range v {
			str := strings.TrimSpace(fmt.Sprint(entry))
			if str != "" {
				versions = append(versions, str)
			}
		}
		return versions, nil
	case []string:
		return v, nil
	case string:
		// Single version string
		if v != "" {
			return []string{v}, nil
		}
		return []string{}, nil
	default:
		return nil, fmt.Errorf("expected array or string, got %T", node)
	}
}

// parseRawWithExtraction extracts versions from raw output using regex pattern.
//
// If no pattern is provided in the extraction config, uses a default pattern that
// matches common version formats (e.g., "1.2.3", "v1.2.3-alpha").
//
// Parameters:
//   - extraction: Extraction configuration containing the regex Pattern; may be nil
//   - output: Raw text bytes to parse
//
// Returns:
//   - []string: Extracted version strings
//   - error: When regex compilation or parsing fails; returns nil on success
func parseRawWithExtraction(extraction *config.OutdatedExtractionCfg, output []byte) ([]string, error) {
	pattern := ""
	if extraction != nil {
		pattern = extraction.Pattern
	}

	if pattern == "" {
		// Default pattern: one version per line
		pattern = `(?m)^[\s]*(?P<version>v?[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[._-][a-zA-Z0-9]+)*)[\s]*$`
	}

	return parseRegexVersions(pattern, output)
}

// Core parsing functions

// parseJSONVersions parses JSON output and extracts version strings using optional key path.
//
// It performs the following operations:
//   - Strips UTF-8 BOM if present
//   - Unmarshals JSON into a generic structure
//   - Navigates to the specified key using dot notation
//   - Handles both array formats and object formats (keys as versions)
//
// Parameters:
//   - key: Dot-separated path to version data (e.g., "versions" or "data.versions"); empty to use root
//   - output: Raw JSON bytes from command execution
//
// Returns:
//   - []string: Extracted version strings
//   - error: When JSON parsing fails or key path is invalid; returns nil on success
func parseJSONVersions(key string, output []byte) ([]string, error) {
	// Strip BOM if present (common with Windows dotnet CLI output)
	output = stripBOM(output)

	var payload any
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JSON output: %w", err)
	}

	node := payload
	if key != "" {
		parts := strings.Split(key, ".")
		for _, part := range parts {
			currentMap, ok := node.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("json key %s not found", key)
			}
			node = currentMap[part]
		}
	}

	// Handle map[string]any where keys are versions (like npm registry)
	if keyMap, ok := node.(map[string]any); ok {
		versions := make([]string, 0, len(keyMap))
		for k := range keyMap {
			versions = append(versions, k)
		}
		return versions, nil
	}

	rawSlice, ok := node.([]any)
	if !ok {
		return nil, fmt.Errorf("json key %s did not resolve to an array or object", key)
	}

	versions := make([]string, 0, len(rawSlice))
	for _, entry := range rawSlice {
		str := strings.TrimSpace(fmt.Sprint(entry))
		if str != "" {
			versions = append(versions, str)
		}
	}

	return versions, nil
}

// parseRegexVersions extracts versions from raw output using a regex pattern.
//
// It performs the following operations:
//   - Compiles the provided regex pattern
//   - Finds all matches in the output
//   - Extracts version from named "version" group or first capture group
//   - Deduplicates versions while preserving order
//
// Parameters:
//   - pattern: Regular expression pattern with optional "version" named group
//   - output: Raw output bytes to search for version patterns
//
// Returns:
//   - []string: Extracted and deduplicated version strings
//   - error: When regex compilation fails; returns nil on success
func parseRegexVersions(pattern string, output []byte) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid extraction pattern: %w", err)
	}

	matches := re.FindAllStringSubmatch(string(output), -1)
	if len(matches) == 0 {
		return []string{}, nil
	}

	versions := make([]string, 0, len(matches))
	seen := make(map[string]struct{})

	for _, match := range matches {
		version := ""

		if idx := re.SubexpIndex("version"); idx >= 0 && idx < len(match) {
			version = match[idx]
		} else if len(match) > 1 {
			version = match[1]
		} else if len(match) > 0 {
			version = match[0]
		}

		version = strings.TrimSpace(version)
		if version != "" {
			if _, exists := seen[version]; !exists {
				versions = append(versions, version)
				seen[version] = struct{}{}
			}
		}
	}

	return versions, nil
}
