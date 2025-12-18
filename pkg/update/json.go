package update

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/iancoleman/orderedmap"
)

// jsonUnmarshalFunc is a variable that holds the json.Unmarshal function.
// This allows for dependency injection during testing.
var jsonUnmarshalFunc = json.Unmarshal

// updateJSONVersion updates the version of a package in JSON manifest content.
//
// It performs the following operations:
//   - Step 1: Parse JSON content to validate structure
//   - Step 2: Unmarshal into ordered map to preserve field order
//   - Step 3: Locate and update package version in dependency fields
//   - Step 4: Marshal back to JSON with proper formatting
//
// Parameters:
//   - content: The original JSON file content as bytes
//   - p: The package to update, containing name and constraint information
//   - ruleCfg: Package manager configuration with dependency field names
//   - target: The target version to update to (without constraint prefix)
//
// Returns:
//   - []byte: Updated JSON content with preserved field order and formatting
//   - error: Returns error if JSON is invalid, package not found, or marshaling fails; returns nil on success
func updateJSONVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
	parser := &formats.JSONParser{}
	if _, err := parser.Parse(content, &ruleCfg); err != nil {
		return nil, err
	}

	exists := false
	data := orderedmap.New()
	if unmarshalErr := jsonUnmarshalFunc(content, data); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	for field := range ruleCfg.Fields {
		rawDeps, ok := data.Get(field)
		if !ok {
			continue
		}

		var deps *orderedmap.OrderedMap
		switch v := rawDeps.(type) {
		case orderedmap.OrderedMap:
			copy := v
			deps = &copy
			data.Set(field, deps)
		case map[string]interface{}:
			converted := orderedmap.New()
			for key, val := range v {
				converted.Set(key, val)
			}
			deps = converted
			data.Set(field, deps)
		default:
			continue
		}

		if _, ok := deps.Get(p.Name); ok {
			exists = true
			deps.Set(p.Name, fmt.Sprintf("%s%s", p.Constraint, target))
		}
	}

	if !exists {
		return nil, fmt.Errorf("package %s not found in %s", p.Name, p.Source)
	}

	return marshalJSON(data)
}

// marshalJSON marshals data to JSON format with proper formatting and escape handling.
//
// It performs the following operations:
//   - Step 1: Disable HTML escaping for ordered maps if applicable
//   - Step 2: Create JSON encoder with HTML escaping disabled
//   - Step 3: Apply proper indentation (2 spaces)
//   - Step 4: Trim trailing newline from output
//
// Parameters:
//   - data: The data to marshal, typically an *orderedmap.OrderedMap or compatible type
//
// Returns:
//   - []byte: JSON bytes with proper formatting and no HTML escaping
//   - error: Returns error if encoding fails; returns nil on success
func marshalJSON(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if ordered, ok := data.(*orderedmap.OrderedMap); ok {
		disableOrderedMapEscape(ordered)
	}
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// disableOrderedMapEscape recursively disables HTML escaping for an ordered map and all nested maps.
//
// It performs the following operations:
//   - Step 1: Set EscapeHTML to false on the map
//   - Step 2: Iterate through all keys and normalize escaping for nested values
//
// Parameters:
//   - m: The ordered map to process
//
// Returns:
//   - This function does not return a value; it modifies the map in place
func disableOrderedMapEscape(m *orderedmap.OrderedMap) {
	m.SetEscapeHTML(false)
	for _, key := range m.Keys() {
		val, _ := m.Get(key)
		m.Set(key, normalizeOrderedMapEscaping(val))
	}
}

// normalizeOrderedMapEscaping recursively normalizes HTML escaping for a value of any type.
//
// It performs the following operations:
//   - Step 1: Check the value type (ordered map, slice, or primitive)
//   - Step 2: For ordered maps, disable HTML escaping recursively
//   - Step 3: For slices, normalize each element
//   - Step 4: For primitives, return as-is
//
// Parameters:
//   - val: The value to normalize, can be any type
//
// Returns:
//   - interface{}: The normalized value with HTML escaping disabled for all nested ordered maps
func normalizeOrderedMapEscaping(val interface{}) interface{} {
	switch v := val.(type) {
	case *orderedmap.OrderedMap:
		disableOrderedMapEscape(v)
		return v
	case orderedmap.OrderedMap:
		copy := v
		disableOrderedMapEscape(&copy)
		return &copy
	case []interface{}:
		for i, item := range v {
			v[i] = normalizeOrderedMapEscaping(item)
		}
		return v
	default:
		return val
	}
}
