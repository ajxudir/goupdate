package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsRules is used to test JS package manager rules
var jsRules = []string{"npm", "pnpm", "yarn"}

// TestLoadConfigComplete tests the behavior of LoadConfig with various scenarios.
//
// It verifies:
//   - Default config loads successfully with working directory
//   - Custom config files are loaded correctly
//   - Nonexistent config files return an error
//   - Default config fallback works with invalid default YAML
func TestLoadConfigComplete(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("default config", func(t *testing.T) {
		cfg, err := LoadConfig("", tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, tmpDir, cfg.WorkingDir)
		assert.Greater(t, len(cfg.Rules), 5)
		assert.NotEmpty(t, cfg.ExcludeVersions)
	})

	t.Run("custom config", func(t *testing.T) {
		configFile := filepath.Join(tmpDir, ".goupdate.yml")
		content := `rules:
  custom-rule:
    manager: custom
    include: ["*.custom"]
    format: raw
    fields:
      packages: prod`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfig(configFile, tmpDir)
		require.NoError(t, err)
		assert.Contains(t, cfg.Rules, "custom-rule")
	})

	t.Run("nonexistent config", func(t *testing.T) {
		cfg, err := LoadConfig("/nonexistent/config.yml", tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("default config fallback", func(t *testing.T) {
		original := defaultConfigYAML
		defaultConfigYAML = "invalid: ["
		defer func() { defaultConfigYAML = original }()

		cfg := loadDefaultConfig()
		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.Rules)
	})
}

// TestLoadConfigLocalConfigExtendsError tests the behavior of LoadConfig when extends references a missing file.
//
// It verifies:
//   - Loading config with missing extends file returns an error
func TestLoadConfigLocalConfigExtendsError(t *testing.T) {
	tmpDir := t.TempDir()

	content := "extends: ['missing.yml']\n"
	err := os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(content), 0644)
	require.NoError(t, err)

	cfg, loadErr := LoadConfig("", tmpDir)
	assert.Error(t, loadErr)
	assert.Nil(t, cfg)
}

// TestLoadConfigLocalConfigSuccess tests the behavior of LoadConfig with a local .goupdate.yml file.
//
// It verifies:
//   - Local .goupdate.yml file is found and loaded when config path is empty
func TestLoadConfigLocalConfigSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid local .goupdate.yml config file
	content := `rules:
  local-rule:
    manager: local
    include: ["*.local"]
    format: raw
    fields:
      packages: local
`
	err := os.WriteFile(filepath.Join(tmpDir, ".goupdate.yml"), []byte(content), 0644)
	require.NoError(t, err)

	// Load with empty configPath - should find and load .goupdate.yml
	cfg, loadErr := LoadConfig("", tmpDir)
	require.NoError(t, loadErr)
	assert.NotNil(t, cfg)
	assert.Contains(t, cfg.Rules, "local-rule")
}

// TestLoadConfigDefaultWorkingDir tests the behavior of LoadConfig with default working directory.
//
// It verifies:
//   - Working directory is set correctly when not specified in config
func TestLoadConfigDefaultWorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")

	err := os.WriteFile(configFile, []byte("rules: {}\n"), 0644)
	require.NoError(t, err)

	cfg, loadErr := LoadConfig(configFile, "")
	require.NoError(t, loadErr)
	assert.Equal(t, ".", cfg.WorkingDir)
}

// TestLoadConfigFileInvalidYAML tests the behavior of LoadConfig with invalid YAML.
//
// It verifies:
//   - Invalid YAML returns an error with helpful message
func TestLoadConfigFileInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yml")

	err := os.WriteFile(invalidFile, []byte("invalid: ["), 0644)
	require.NoError(t, err)

	cfg, loadErr := loadConfigFile(invalidFile)
	assert.Error(t, loadErr)
	assert.Nil(t, cfg)
}

// TestProcessExtendsErrorInNestedConfig tests the behavior of processExtends with errors in nested config.
//
// It verifies:
//   - Errors in nested extended configs are properly reported
func TestProcessExtendsErrorInNestedConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// child config references a missing file to trigger an error on recursive processing
	childConfig := filepath.Join(tmpDir, "child.yml")
	err := os.WriteFile(childConfig, []byte("extends: ['missing.yml']\n"), 0644)
	require.NoError(t, err)

	rootConfig := Config{Extends: []string{"child.yml"}}

	cfg, procErr := processExtends(&rootConfig, tmpDir)
	assert.Nil(t, cfg)
	assert.Error(t, procErr)
}

// TestProcessExtendsDetectsCycle tests the behavior of processExtends with circular dependencies.
//
// It verifies:
//   - Circular extends dependencies are detected and return an error
func TestProcessExtendsDetectsCycle(t *testing.T) {
	tmpDir := t.TempDir()

	parentConfig := filepath.Join(tmpDir, "parent.yml")
	childConfig := filepath.Join(tmpDir, "child.yml")

	require.NoError(t, os.WriteFile(parentConfig, []byte("extends: ['child.yml']\n"), 0644))
	require.NoError(t, os.WriteFile(childConfig, []byte("extends: ['parent.yml']\n"), 0644))

	rootCfg := Config{Extends: []string{"parent.yml"}}

	result, err := processExtends(&rootCfg, tmpDir)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic extends")
}

// TestProcessExtendsDetectsDefaultCycle tests the behavior of processExtends with cycles in default config.
//
// It verifies:
//   - Circular dependencies to default config are detected and return an error
func TestProcessExtendsDetectsDefaultCycle(t *testing.T) {
	cfg := &Config{Extends: []string{"default"}}
	stack := map[string]bool{"__default__": true}

	result, err := processExtendsWithStackSecure(cfg, ".", stack, cfg)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic extends")
}

// TestProcessExtendsWithInvalidPath tests the behavior of processExtends with invalid file paths.
//
// It verifies:
//   - Invalid extends paths return an error
func TestProcessExtendsWithInvalidPath(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()

	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()
	require.NoError(t, os.RemoveAll(tmpDir))

	cfg := &Config{Extends: []string{"child.yml"}}

	result, procErr := processExtendsWithStackSecure(cfg, ".", make(map[string]bool), cfg)
	assert.Nil(t, result)
	require.Error(t, procErr)
	assert.Contains(t, procErr.Error(), "failed to resolve extend")
}

// TestProcessExtendsWithInvalidYaml tests the behavior of processExtends with invalid YAML in extended file.
//
// It verifies:
//   - Invalid YAML in extended file returns an error
func TestProcessExtendsWithInvalidYaml(t *testing.T) {
	tmpDir := t.TempDir()

	invalidContent := "rules: ["
	invalidPath := filepath.Join(tmpDir, "invalid.yml")
	require.NoError(t, os.WriteFile(invalidPath, []byte(invalidContent), 0644))

	cfg := &Config{Extends: []string{"invalid.yml"}}

	result, err := processExtendsWithStackSecure(cfg, tmpDir, make(map[string]bool), cfg)
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load extend")
	assert.Contains(t, err.Error(), "invalid.yml")
}

// TestProcessExtendsWithStackNoExtends tests the behavior of processExtends when config has no extends.
//
// It verifies:
//   - Configs without extends field load successfully
func TestProcessExtendsWithStackNoExtends(t *testing.T) {
	original := &Config{Rules: map[string]PackageManagerCfg{"pkg": {Manager: "js"}}}

	result, err := processExtendsWithStackSecure(original, ".", make(map[string]bool), original)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

// TestConfigExtends tests the behavior of config extension mechanism.
//
// It verifies:
//   - Single extends work correctly
//   - Multiple extends are processed in order
//   - Nested extends work correctly
func TestConfigExtends(t *testing.T) {
	t.Run("extend default", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `extends: ["default"]

rules:
  custom-scanner:
    manager: scanner
    include: ["**/*.scan"]
    format: json

  npm:
    ignore: ["eslint-*", "babel-*"]`

		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		cfg, err := LoadConfig(configPath, tmpDir)
		require.NoError(t, err)

		// Should have default rules
		assert.Contains(t, cfg.Rules, "composer")

		// Should have custom rule
		assert.Contains(t, cfg.Rules, "custom-scanner")

		// Should have override on npm
		npmRule := cfg.Rules["npm"]
		assert.Contains(t, npmRule.Ignore, "eslint-*")
		assert.Contains(t, npmRule.Ignore, "babel-*")
	})

	t.Run("extend base config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base config
		baseContent := `rules:
  custom-rule-base:
    manager: custom
    include: ["**/*.custom"]
    format: raw`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "base.yml"), []byte(baseContent), 0644))

		// Create team config extending base
		teamContent := `extends: ["base.yml"]

rules:
  npm:
    exclude: ["vendor/**"]`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "team.yml"), []byte(teamContent), 0644))

		cfg, err := LoadConfig(filepath.Join(tmpDir, "team.yml"), tmpDir)
		require.NoError(t, err)

		// Should have inherited rule from base
		assert.Contains(t, cfg.Rules, "custom-rule-base")

		// Should have team-specific config
		teamRule := cfg.Rules["npm"]
		assert.Contains(t, teamRule.Exclude, "vendor/**")
	})

	t.Run("chain multiple extends", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create base
		baseContent := `rules:
  custom-rule-base:
    manager: custom
    include: ["**/*.custom"]
    format: raw`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "base.yml"), []byte(baseContent), 0644))

		// Create chain extending base
		chainContent := `extends: ["base.yml"]

rules:
  custom-rule-chain:
    manager: chain
    include: ["**/*.chain"]
    format: raw`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "chain.yml"), []byte(chainContent), 0644))

		cfg, err := LoadConfig(filepath.Join(tmpDir, "chain.yml"), tmpDir)
		require.NoError(t, err)

		// Should have rules from both base and chain
		assert.Contains(t, cfg.Rules, "custom-rule-base")
		assert.Contains(t, cfg.Rules, "custom-rule-chain")
	})

	t.Run("local config with extends", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create team config
		teamContent := `rules:
  npm:
    exclude: ["vendor/**"]`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "team.yml"), []byte(teamContent), 0644))

		// Create local config
		localContent := `extends: ["default", "team.yml"]

rules:
  custom-scanner:
    manager: scanner
    include: ["**/*.scan"]
    format: json

  npm:
    exclude: ["test/**"]`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "local.yml"), []byte(localContent), 0644))

		cfg, err := LoadConfig(filepath.Join(tmpDir, "local.yml"), tmpDir)
		require.NoError(t, err)

		// Should have custom rule
		assert.Contains(t, cfg.Rules, "custom-scanner")
		assert.Equal(t, "scanner", cfg.Rules["custom-scanner"].Manager)

		// Local override should win
		assert.Contains(t, cfg.Rules["npm"].Exclude, "test/**")
	})

	t.Run("extends preserves working dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Create team config in parent
		teamContent := `rules:
  custom-rule-base:
    manager: custom
    include: ["**/*.custom"]
    format: raw`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "team.yml"), []byte(teamContent), 0644))

		// Create config in subdir extending parent (with security enabled for path traversal)
		configContent := `security:
  allow_path_traversal: true
extends: ["../team.yml"]`
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "config.yml"), []byte(configContent), 0644))

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(subDir))
		defer func() { _ = os.Chdir(originalDir) }()

		cfg, err := LoadConfig("config.yml", subDir)
		require.NoError(t, err)

		// Should have inherited rule
		assert.Contains(t, cfg.Rules, "custom-rule-base")
	})
}

// TestConfigExtendsErrors tests the behavior of config extension error cases.
//
// It verifies:
//   - Missing extended files return an error
//   - Invalid YAML in extended files returns an error
func TestConfigExtendsErrors(t *testing.T) {
	t.Run("missing extend file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `extends: ["nonexistent.yml"]`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0644))

		_, err := LoadConfig(configPath, tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent.yml")
	})
}

// TestConfigInSubdirectory tests the behavior of LoadConfig with config in subdirectory.
//
// It verifies:
//   - Config files in subdirectories are loaded correctly
//   - Relative paths in extends work from subdirectory
func TestConfigInSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "npm", "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	// Create team config in root
	teamContent := `rules:
  custom-rule-base:
    manager: custom
    include: ["**/*.custom"]
    format: raw`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "team.yml"), []byte(teamContent), 0644))

	// Create config in subdir (with security enabled for path traversal)
	configContent := `security:
  allow_path_traversal: true
extends: ["../../team.yml"]

rules:
  custom-scanner:
    manager: scanner
    include: ["**/*.scan"]
    format: json

  npm:
    exclude: ["test/**"]`
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "config.yml"), []byte(configContent), 0644))

	cfg, err := LoadConfig(filepath.Join(subDir, "config.yml"), subDir)
	require.NoError(t, err)

	assert.Contains(t, cfg.Rules, "custom-scanner")
	assert.Contains(t, cfg.Rules["npm"].Exclude, "test/**")
}

// TestLoadConfigErrorsOnDuplicateGroupMembers tests the behavior of LoadConfig with duplicate group members.
//
// It verifies:
//   - Duplicate group members return an error
func TestLoadConfigErrorsOnDuplicateGroupMembers(t *testing.T) {
	configRoot, _ := filepath.Abs("../testdata_errors/_config-errors/duplicate-groups")

	_, err := LoadConfig(filepath.Join(configRoot, ".goupdate.yml"), configRoot)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "react")
	assert.Condition(t, func() bool {
		for _, rule := range jsRules {
			if strings.Contains(err.Error(), rule) {
				return true
			}
		}

		return false
	})
}

// TestLoadConfigFileSizeLimit tests the behavior of LoadConfig with file size limits.
//
// It verifies:
//   - Files exceeding max size return an error
//   - Custom max size from security config is respected
func TestLoadConfigFileSizeLimit(t *testing.T) {
	t.Run("rejects oversized config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "large.yml")

		// Create a file that exceeds the 10MB limit
		// We'll create a file with repeated content
		content := strings.Repeat("x: y\n", 2*1024*1024+1) // ~10MB+

		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Verify the file is larger than the limit
		info, err := os.Stat(configFile)
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(10*1024*1024))

		// Loading should fail with size error
		cfg, err := loadConfigFile(configFile)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "config file too large")
	})

	t.Run("accepts config file at size limit boundary", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "small.yml")

		// Create a small valid config file
		content := "rules: {}\n"
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Loading should succeed
		cfg, err := loadConfigFile(configFile)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("LoadConfigFileStrict also enforces size limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "large.yml")

		// Create a file that exceeds the 10MB limit
		content := strings.Repeat("x: y\n", 2*1024*1024+1)

		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		cfg, err := LoadConfigFileStrict(configFile)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "config file too large")
	})
}

// TestProcessExtendsPathTraversal tests the behavior of processExtends with path traversal attempts.
//
// It verifies:
//   - Path traversal (..) is rejected by default
//   - Absolute paths are rejected by default
//   - Security config allows path traversal when enabled
//   - Security config allows absolute paths when enabled
func TestProcessExtendsPathTraversal(t *testing.T) {
	t.Run("allows legitimate parent directory extends", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Create parent config
		parentContent := `rules:
  custom:
    manager: test
    include: ["*.test"]
    format: raw
    fields:
      packages: prod`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "parent.yml"), []byte(parentContent), 0644))

		// Create child config that extends parent (with security enabled for path traversal)
		childContent := `security:
  allow_path_traversal: true
extends: ["../parent.yml"]
rules:
  child-rule:
    manager: child
    include: ["*.child"]
    format: raw
    fields:
      packages: prod`
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "child.yml"), []byte(childContent), 0644))

		cfg, err := LoadConfig(filepath.Join(subDir, "child.yml"), subDir)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Contains(t, cfg.Rules, "custom")
		assert.Contains(t, cfg.Rules, "child-rule")
	})

	t.Run("fails on nonexistent path traversal", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Create child config that tries to extend nonexistent file outside scope
		// Enable path traversal to test that the file itself doesn't exist
		childContent := `security:
  allow_path_traversal: true
extends: ["../../../nonexistent.yml"]`
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "child.yml"), []byte(childContent), 0644))

		cfg, err := LoadConfig(filepath.Join(subDir, "child.yml"), subDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to resolve extend")
	})

	t.Run("resolves absolute paths correctly", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a config with absolute path extends
		absPath := filepath.Join(tmpDir, "absolute.yml")
		absContent := `rules:
  abs-rule:
    manager: abs
    include: ["*.abs"]
    format: raw
    fields:
      packages: prod`
		require.NoError(t, os.WriteFile(absPath, []byte(absContent), 0644))

		// Create config that extends via absolute path (with security enabled)
		mainContent := "security:\n  allow_absolute_paths: true\nextends: [\"" + absPath + "\"]"
		mainPath := filepath.Join(tmpDir, "main.yml")
		require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

		cfg, err := LoadConfig(mainPath, tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Contains(t, cfg.Rules, "abs-rule")
	})
}

// TestSecurityDefaultsBlock tests the behavior of security settings in extended configs.
//
// It verifies:
//   - Security settings in extended configs are ignored
//   - Only root config security settings are applied
func TestSecurityDefaultsBlock(t *testing.T) {
	t.Run("path traversal blocked by default with helpful message", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		// Create parent config
		parentContent := `rules:
  test:
    manager: test
    include: ["*.test"]
    format: raw
    fields:
      packages: prod`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "parent.yml"), []byte(parentContent), 0644))

		// Create child config without enabling path traversal
		childContent := `extends: ["../parent.yml"]`
		require.NoError(t, os.WriteFile(filepath.Join(subDir, "child.yml"), []byte(childContent), 0644))

		cfg, err := LoadConfig(filepath.Join(subDir, "child.yml"), subDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "path traversal not allowed")
		assert.Contains(t, err.Error(), "allow_path_traversal: true")
	})

	t.Run("absolute paths blocked by default with helpful message", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create target config
		absPath := filepath.Join(tmpDir, "target.yml")
		targetContent := `rules:
  test:
    manager: test
    include: ["*.test"]
    format: raw
    fields:
      packages: prod`
		require.NoError(t, os.WriteFile(absPath, []byte(targetContent), 0644))

		// Create main config without enabling absolute paths
		mainContent := "extends: [\"" + absPath + "\"]"
		mainPath := filepath.Join(tmpDir, "main.yml")
		require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0644))

		cfg, err := LoadConfig(mainPath, tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "absolute paths not allowed")
		assert.Contains(t, err.Error(), "allow_absolute_paths: true")
	})
}

// TestLoadConfigFileStrict tests the behavior of LoadConfigFileStrict.
//
// It verifies:
//   - Valid config loads successfully
//   - Warnings are treated as errors in strict mode
func TestLoadConfigFileStrict(t *testing.T) {
	t.Run("valid config loads successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "valid.yml")
		content := `rules:
  npm:
    manager: js
    include: ["**/package.json"]
    format: json`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		cfg, err := LoadConfigFileStrict(configFile)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Contains(t, cfg.Rules, "npm")
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "invalid.yml")
		content := `rules:
  npm:
    badfield: value`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		cfg, err := LoadConfigFileStrict(configFile)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		cfg, err := LoadConfigFileStrict("/nonexistent/path/config.yml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})
}
