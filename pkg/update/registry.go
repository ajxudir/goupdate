package update

import (
	"fmt"
	"sync"

	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/errors"
	"github.com/user/goupdate/pkg/formats"
)

// FormatUpdater defines the interface for updating package versions in manifest files.
// Implementations handle format-specific logic (JSON, YAML, XML, Raw).
type FormatUpdater interface {
	// UpdateVersion updates the version of a package in the given content.
	// Returns the updated content or an error.
	UpdateVersion(content []byte, pkg formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error)
}

// FormatUpdaterFunc is a function type that implements FormatUpdater.
type FormatUpdaterFunc func(content []byte, pkg formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error)

// UpdateVersion implements FormatUpdater for FormatUpdaterFunc.
//
// This method allows FormatUpdaterFunc to satisfy the FormatUpdater interface by
// delegating to the underlying function.
//
// Parameters:
//   - content: The manifest file content to update
//   - pkg: The package whose version should be updated
//   - ruleCfg: Package manager configuration with format-specific rules
//   - target: The target version to update to
//
// Returns:
//   - []byte: Updated manifest content
//   - error: Returns error if the underlying function fails; returns nil on success
func (f FormatUpdaterFunc) UpdateVersion(content []byte, pkg formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
	return f(content, pkg, ruleCfg, target)
}

// updaterRegistry holds registered format updaters.
var updaterRegistry = struct {
	sync.RWMutex
	updaters map[string]FormatUpdater
}{
	updaters: make(map[string]FormatUpdater),
}

// RegisterFormatUpdater registers an updater for a specific format.
//
// This allows extending goupdate with custom format handlers beyond the built-in
// JSON, YAML, XML, and raw formats.
//
// Parameters:
//   - format: The format identifier (e.g., "json", "yaml", "custom")
//   - updater: The FormatUpdater implementation to handle this format
//
// Returns:
//   - This function does not return a value; it modifies the registry
func RegisterFormatUpdater(format string, updater FormatUpdater) {
	updaterRegistry.Lock()
	defer updaterRegistry.Unlock()
	updaterRegistry.updaters[format] = updater
}

// GetFormatUpdater returns the updater for the given format.
//
// Parameters:
//   - format: The format identifier to look up (e.g., "json", "yaml")
//
// Returns:
//   - FormatUpdater: The registered updater for this format; returns nil if no updater is registered
func GetFormatUpdater(format string) FormatUpdater {
	updaterRegistry.RLock()
	defer updaterRegistry.RUnlock()
	return updaterRegistry.updaters[format]
}

// ListRegisteredFormats returns a list of all registered format names.
//
// This is useful for error messages and debugging to show which formats are available.
//
// Returns:
//   - []string: List of all registered format identifiers (e.g., ["json", "yaml", "xml", "raw"])
func ListRegisteredFormats() []string {
	updaterRegistry.RLock()
	defer updaterRegistry.RUnlock()
	formats := make([]string, 0, len(updaterRegistry.updaters))
	for format := range updaterRegistry.updaters {
		formats = append(formats, format)
	}
	return formats
}

// init registers the built-in format updaters.
func init() {
	// Register built-in updaters
	RegisterFormatUpdater("json", FormatUpdaterFunc(updateJSONVersion))
	RegisterFormatUpdater("yaml", FormatUpdaterFunc(updateYAMLVersion))
	RegisterFormatUpdater("xml", FormatUpdaterFunc(updateXMLVersion))
	RegisterFormatUpdater("raw", FormatUpdaterFunc(updateRawVersion))
}

// getUpdaterForFormat returns the appropriate updater for the given format.
//
// It performs the following operations:
//   - Step 1: Check the updater registry for the format
//   - Step 2: Return the registered updater if found
//   - Step 3: Return UnsupportedError if no updater is registered
//
// Parameters:
//   - format: The format identifier to look up (e.g., "json", "yaml")
//
// Returns:
//   - FormatUpdater: The updater for this format
//   - error: Returns UnsupportedError if format is not registered; returns nil on success
func getUpdaterForFormat(format string) (FormatUpdater, error) {
	// Try registry first
	if updater := GetFormatUpdater(format); updater != nil {
		return updater, nil
	}

	// Format not registered
	return nil, &errors.UnsupportedError{
		Reason: fmt.Sprintf("updates not supported for format %s; registered formats: %v", format, ListRegisteredFormats()),
	}
}
