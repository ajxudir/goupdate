// Package constants provides centralized string constants used throughout the application.
package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStatusConstants tests the behavior of status constants.
//
// It verifies:
//   - Status constants have the expected string values
//   - Prevents accidental changes to status constant values
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StatusUpToDate", StatusUpToDate, "UpToDate"},
		{"StatusUpdated", StatusUpdated, "Updated"},
		{"StatusPlanned", StatusPlanned, "Planned"},
		{"StatusFailed", StatusFailed, "Failed"},
		{"StatusConfigError", StatusConfigError, "ConfigError"},
		{"StatusSummarizeError", StatusSummarizeError, "SummarizeError"},
		{"StatusOutdated", StatusOutdated, "Outdated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant, "constant %s has unexpected value", tt.name)
		})
	}
}

// TestPlaceholderConstants tests the behavior of placeholder constants.
//
// It verifies:
//   - Placeholder constants have the expected string values
//   - PlaceholderNA and PlaceholderWildcard are correctly defined
func TestPlaceholderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"PlaceholderNA", PlaceholderNA, "#N/A"},
		{"PlaceholderWildcard", PlaceholderWildcard, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.constant, "constant %s has unexpected value", tt.name)
		})
	}
}

// TestFilterConstants tests the behavior of filter constants.
//
// It verifies:
//   - FilterAll has the expected "all" value
func TestFilterConstants(t *testing.T) {
	assert.Equal(t, "all", FilterAll, "FilterAll should be 'all'")
}

// TestIconConstants tests the behavior of icon constants.
//
// It verifies:
//   - All icon constants are non-empty strings
//   - Icons are properly defined for use in CLI output
func TestIconConstants(t *testing.T) {
	icons := []struct {
		name     string
		constant string
	}{
		{"IconSuccess", IconSuccess},
		{"IconWarning", IconWarning},
		{"IconError", IconError},
		{"IconInfo", IconInfo},
		{"IconNotConfigured", IconNotConfigured},
		{"IconBlocked", IconBlocked},
		{"IconPinned", IconPinned},
		{"IconPending", IconPending},
		{"IconCheckmark", IconCheckmark},
		{"IconCross", IconCross},
		{"IconWarn", IconWarn},
		{"IconCheckmarkBox", IconCheckmarkBox},
		{"IconLightbulb", IconLightbulb},
	}

	for _, icon := range icons {
		t.Run(icon.name, func(t *testing.T) {
			assert.NotEmpty(t, icon.constant, "icon %s should not be empty", icon.name)
		})
	}
}

// TestValidationConstants tests the behavior of validation status constants.
//
// It verifies:
//   - ValidationValid contains the substring "valid"
//   - ValidationInvalid contains the substring "invalid"
func TestValidationConstants(t *testing.T) {
	assert.Contains(t, ValidationValid, "valid", "ValidationValid should contain 'valid'")
	assert.Contains(t, ValidationInvalid, "invalid", "ValidationInvalid should contain 'invalid'")
}

// TestIconsAreDistinct tests the behavior of icon uniqueness.
//
// It verifies:
//   - All status icons have distinct values
//   - No two icons share the same visual representation
func TestIconsAreDistinct(t *testing.T) {
	icons := map[string]string{
		"IconSuccess":       IconSuccess,
		"IconWarning":       IconWarning,
		"IconError":         IconError,
		"IconInfo":          IconInfo,
		"IconNotConfigured": IconNotConfigured,
		"IconBlocked":       IconBlocked,
		"IconPending":       IconPending,
	}

	// Check that all status icons are different
	seen := make(map[string]string)
	for name, icon := range icons {
		if existingName, exists := seen[icon]; exists {
			t.Errorf("Icon %s has same value as %s: %s", name, existingName, icon)
		}
		seen[icon] = name
	}
}
