package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDefaultConfig tests the behavior of GetDefaultConfig.
//
// It verifies:
//   - Default config YAML contains rules
//   - Contains npm and composer configurations
func TestGetDefaultConfig(t *testing.T) {
	yml := GetDefaultConfig()
	assert.Contains(t, yml, "rules:")
	assert.Contains(t, yml, "npm")
	assert.Contains(t, yml, "composer")
}

// TestGetTemplateConfig tests the behavior of GetTemplateConfig.
//
// It verifies:
//   - Template config YAML is not empty
//   - Contains sample configuration with rules
func TestGetTemplateConfig(t *testing.T) {
	yml := GetTemplateConfig()
	assert.NotEmpty(t, yml)
	// Template should contain sample configuration
	assert.Contains(t, yml, "rules:")
}

// TestDefaultConfigIncludesUpdateCommands tests the behavior of default config update commands.
//
// It verifies:
//   - JS rules use new Commands format
//   - Composer uses general install command without package-specific placeholders
//   - Go mod uses go mod tidy without package-specific placeholders
//   - Dotnet uses general restore command without package-specific placeholders
func TestDefaultConfigIncludesUpdateCommands(t *testing.T) {
	cfg := loadDefaultConfig()

	// Test JS rules use new Commands format
	for _, ruleName := range jsRules {
		npm := cfg.Rules[ruleName]
		require.NotNil(t, npm.Update)
		assert.NotEmpty(t, npm.Update.Commands, "rule %s should have Commands", ruleName)
	}

	// Test composer uses general install command (no package-specific args)
	composer := cfg.Rules["composer"]
	require.NotNil(t, composer.Update)
	assert.Contains(t, composer.Update.Commands, "composer install")
	// Should not have package-specific placeholders
	assert.NotContains(t, composer.Update.Commands, "{{package}}")

	// Test Go mod uses go mod tidy (not go get with package-specific args)
	goMod := cfg.Rules["mod"]
	require.NotNil(t, goMod.Update)
	assert.Contains(t, goMod.Update.Commands, "go mod tidy")
	// Should not have package-specific placeholders
	assert.NotContains(t, goMod.Update.Commands, "{{package}}")

	// Test dotnet uses general restore command
	dotnet := cfg.Rules["msbuild"]
	require.NotNil(t, dotnet.Update)
	assert.Contains(t, dotnet.Update.Commands, "dotnet restore")
	// Should not have package-specific placeholders
	assert.NotContains(t, dotnet.Update.Commands, "{{package}}")
}
