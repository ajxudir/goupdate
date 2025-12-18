package packages

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/warnings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDynamicParser(t *testing.T) {
	parser := NewDynamicParser()
	assert.NotNil(t, parser)
}

func TestDynamicParserParseFile(t *testing.T) {
	parser := NewDynamicParser()
	tmpDir := t.TempDir()

	// Test JSON file
	jsonFile := filepath.Join(tmpDir, "test.json")
	jsonContent := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	require.NoError(t, err)

	cfg := &config.PackageManagerCfg{
		Manager: "test",
		Format:  "json",
		Fields: map[string]string{
			"dependencies": "prod",
		},
	}

	result, err := parser.ParseFile(jsonFile, cfg)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, jsonFile, result.Source)
	assert.Len(t, result.Packages, 1)
	assert.Equal(t, "test", result.Packages[0].Name)

	// Test parser error
	badJSONFile := filepath.Join(tmpDir, "invalid.json")
	badContent := `{"dependencies": {"test": "1.0.0",}`
	require.NoError(t, os.WriteFile(badJSONFile, []byte(badContent), 0o644))

	_, err = parser.ParseFile(badJSONFile, cfg)
	assert.Error(t, err)

	// Test file read error
	_, err = parser.ParseFile("/nonexistent/file", cfg)
	assert.Error(t, err)

	// Test unsupported format
	cfg.Format = "unsupported"
	_, err = parser.ParseFile(jsonFile, cfg)
	assert.Error(t, err)
}

func TestDynamicParserParseFileValidations(t *testing.T) {
	parser := NewDynamicParser()
	jsonFile := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(jsonFile, []byte(`{}`), 0o644))

	_, err := parser.ParseFile(jsonFile, nil)
	assert.Error(t, err)

	_, err = parser.ParseFile(jsonFile, &config.PackageManagerCfg{Format: "", Fields: map[string]string{"dependencies": "prod"}})
	assert.Error(t, err)

	_, err = parser.ParseFile(jsonFile, &config.PackageManagerCfg{Format: "json", Fields: map[string]string{}})
	assert.Error(t, err)
}

func TestDetectFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte("{}"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "go.mod"), []byte("module test"), 0644))

	cfg, err := config.LoadConfig("", tmpDir)
	require.NoError(t, err)

	detected, err := DetectFiles(cfg, tmpDir)
	require.NoError(t, err)

	assert.Contains(t, detected, "npm")
	assert.NotContains(t, detected, "pnpm")
	assert.NotContains(t, detected, "yarn")
	assert.Contains(t, detected, "composer")
	assert.Contains(t, detected, "mod")
}

func TestDetectFilesPrefersLockfileMatches(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies":{"react":"^18.2.0"}}`), 0o644))

	cfg, err := config.LoadConfig("", tmpDir)
	require.NoError(t, err)

	t.Run("pnpm lockfile", func(t *testing.T) {
		lockPath := filepath.Join(tmpDir, "pnpm-lock.yaml")
		lockContent := "lockfileVersion: '6.0'\npackages:\n  /react@18.2.0:\n    resolution:\n      integrity: sha512-test"
		require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0o644))
		t.Cleanup(func() { _ = os.Remove(lockPath) })

		detected, err := DetectFiles(cfg, tmpDir)
		require.NoError(t, err)

		assert.Contains(t, detected, "pnpm")
		assert.NotContains(t, detected, "npm")
		assert.NotContains(t, detected, "yarn")
	})

	t.Run("yarn lockfile", func(t *testing.T) {
		lockPath := filepath.Join(tmpDir, "yarn.lock")
		lockContent := "# yarn lockfile v1\nreact@^18.2.0:\n  version \"18.2.0\"\n  resolved \"https://registry.npmjs.org/react/-/react-18.2.0.tgz\""
		require.NoError(t, os.WriteFile(lockPath, []byte(lockContent), 0o644))
		t.Cleanup(func() { _ = os.Remove(lockPath) })

		detected, err := DetectFiles(cfg, tmpDir)
		require.NoError(t, err)

		assert.Contains(t, detected, "yarn")
		assert.NotContains(t, detected, "npm")
		assert.NotContains(t, detected, "pnpm")
	})
}

func TestResolveRuleForFilePrefersLocksAndPriority(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := filepath.Join(tmpDir, "package.json")
	require.NoError(t, os.WriteFile(manifest, []byte(`{}`), 0o644))

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm":  {LockFiles: []config.LockFileCfg{{Files: []string{"**/package-lock.json"}}}},
		"pnpm": {LockFiles: []config.LockFileCfg{{Files: []string{"**/pnpm-lock.yaml"}}}},
		"yarn": {LockFiles: []config.LockFileCfg{{Files: []string{"**/yarn.lock"}}}},
	}}

	pnpmLock := filepath.Join(tmpDir, "pnpm-lock.yaml")
	require.NoError(t, os.WriteFile(pnpmLock, []byte("lockfileVersion: '6.0'"), 0o644))
	assert.Equal(t, "pnpm", ResolveRuleForFile(cfg, manifest, []string{"npm", "pnpm", "yarn"}))

	require.NoError(t, os.Remove(pnpmLock))
	yarnLock := filepath.Join(tmpDir, "yarn.lock")
	require.NoError(t, os.WriteFile(yarnLock, []byte("# yarn lockfile v1"), 0o644))
	assert.Equal(t, "yarn", ResolveRuleForFile(cfg, manifest, []string{"yarn", "npm"}))

	require.NoError(t, os.Remove(yarnLock))
	assert.Equal(t, "npm", ResolveRuleForFile(cfg, manifest, []string{"npm", "pnpm"}))
}

func TestPrioritizeRulesOrdersKnownManagersFirst(t *testing.T) {
	ordered := prioritizeRules([]string{"custom", "yarn", "npm"})
	assert.Equal(t, []string{"npm", "yarn", "custom"}, ordered)

	ordered = prioritizeRules([]string{"custom", "pnpm"})
	assert.Equal(t, []string{"pnpm", "custom"}, ordered)

	ordered = prioritizeRules([]string{"npm", "custom"})
	assert.Equal(t, []string{"npm", "custom"}, ordered)

	ordered = prioritizeRules([]string{"zeta", "alpha"})
	assert.Equal(t, []string{"alpha", "zeta"}, ordered)
}

func TestRemoveFileFiltersMatches(t *testing.T) {
	files := []string{"a", "b", "a"}
	assert.Equal(t, []string{"b"}, removeFile(files, "a"))
}

func TestDetectFilesWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalWD) }()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0o644))

	cfg := &config.Config{
		WorkingDir: "",
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Include: []string{"package.json"},
				Format:  "json",
				Fields:  map[string]string{"dependencies": "prod"},
			},
		},
	}

	detected, err := DetectFiles(cfg, "")
	require.NoError(t, err)

	if assert.Contains(t, detected, "npm") {
		assert.Equal(t, []string{"package.json"}, detected["npm"])
	}
}

func TestDetectFilesValidations(t *testing.T) {
	baseDir := t.TempDir()

	_, err := DetectFiles(nil, baseDir)
	assert.Error(t, err)

	_, err = DetectFiles(&config.Config{Rules: map[string]config.PackageManagerCfg{}}, baseDir)
	assert.Error(t, err)
}

func TestDetectForRule(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte("{}"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "excluded"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "excluded", "test.json"), []byte("{}"), 0644))

	rule := config.PackageManagerCfg{
		Include: []string{"*.json"},
		Exclude: []string{"excluded/*"},
	}

	files, err := detectForRule(tmpDir, rule)
	assert.NoError(t, err)
	assert.Contains(t, files, filepath.Join(tmpDir, "test.json"))
	assert.NotContains(t, files, filepath.Join(tmpDir, "excluded", "test.json"))
}

func TestDetectForRuleWithMissingBase(t *testing.T) {
	nonexistent := filepath.Join(t.TempDir(), "missing")

	_, err := detectForRule(nonexistent, config.PackageManagerCfg{Include: []string{"*.json"}})
	assert.Error(t, err)
}

func TestDetectForRuleWithFileBase(t *testing.T) {
	tmpDir := t.TempDir()
	fileBase := filepath.Join(tmpDir, "file-base")
	require.NoError(t, os.WriteFile(fileBase, []byte("{}"), 0o644))

	_, err := detectForRule(fileBase, config.PackageManagerCfg{Include: []string{"*.json"}})
	assert.Error(t, err)
}

func TestDetectForRuleSkipsUnreadablePaths(t *testing.T) {
	tmpDir := t.TempDir()
	readable := filepath.Join(tmpDir, "readable.json")
	require.NoError(t, os.WriteFile(readable, []byte("{}"), 0o644))

	unreadableDir := filepath.Join(tmpDir, "blocked")
	require.NoError(t, os.MkdirAll(unreadableDir, 0o755))
	defer func() { _ = os.Chmod(unreadableDir, 0o755) }()
	require.NoError(t, os.Chmod(unreadableDir, 0o000))

	files, err := detectForRule(tmpDir, config.PackageManagerCfg{Include: []string{"*.json"}})
	require.NoError(t, err)
	assert.Contains(t, files, readable)
	assert.NotContains(t, files, filepath.Join(unreadableDir, "ignored.json"))
}

func TestDetectFilesWarnsOnEmptyInclude(t *testing.T) {
	baseDir := t.TempDir()
	buf := &bytes.Buffer{}
	restore := warnings.SetWarningWriter(buf)
	defer restore()

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"npm": {Include: nil}}}

	detected, err := DetectFiles(cfg, baseDir)
	require.NoError(t, err)
	assert.Empty(t, detected)
	assert.Contains(t, buf.String(), "has no include patterns")
}

func TestDetectFilesPropagatesErrors(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "missing")
	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{"pkg": {Include: []string{"*.json"}}}}

	_, err := DetectFiles(cfg, baseDir)
	assert.Error(t, err)
}

func TestDetectFilesUsesWorkingDir(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "nested")
	require.NoError(t, os.MkdirAll(subDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(subDir, "package.json"), []byte(`{}`), 0o644))

	cfg := &config.Config{
		WorkingDir: subDir,
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Include: []string{"**/package.json"},
				Format:  "json",
				Fields:  map[string]string{"dependencies": "prod"},
			},
		},
	}

	detected, err := DetectFiles(cfg, "")
	require.NoError(t, err)

	files := detected["npm"]
	if assert.Len(t, files, 1) {
		assert.Contains(t, filepath.ToSlash(files[0]), "nested/package.json")
	}
}

func TestDetectFilesSkipsDisabledRules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files for both rules
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "composer.json"), []byte(`{}`), 0o644))

	enabled := true
	disabled := false

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Enabled: &enabled,
				Include: []string{"**/package.json"},
				Format:  "json",
				Fields:  map[string]string{"dependencies": "prod"},
			},
			"composer": {
				Enabled: &disabled,
				Include: []string{"**/composer.json"},
				Format:  "json",
				Fields:  map[string]string{"require": "prod"},
			},
		},
	}

	detected, err := DetectFiles(cfg, tmpDir)
	require.NoError(t, err)

	// npm should be detected (enabled)
	assert.Contains(t, detected, "npm")

	// composer should NOT be detected (disabled)
	assert.NotContains(t, detected, "composer")
}

func TestDetectFilesDefaultsToEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{}`), 0o644))

	// Rule without Enabled field should default to enabled
	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				// Enabled is nil (not specified)
				Include: []string{"**/package.json"},
				Format:  "json",
				Fields:  map[string]string{"dependencies": "prod"},
			},
		},
	}

	detected, err := DetectFiles(cfg, tmpDir)
	require.NoError(t, err)

	// npm should be detected since Enabled defaults to true
	assert.Contains(t, detected, "npm")
}

func TestIsEnabled(t *testing.T) {
	enabled := true
	disabled := false

	t.Run("nil defaults to true", func(t *testing.T) {
		rule := config.PackageManagerCfg{Enabled: nil}
		assert.True(t, rule.IsEnabled())
	})

	t.Run("explicit true", func(t *testing.T) {
		rule := config.PackageManagerCfg{Enabled: &enabled}
		assert.True(t, rule.IsEnabled())
	})

	t.Run("explicit false", func(t *testing.T) {
		rule := config.PackageManagerCfg{Enabled: &disabled}
		assert.False(t, rule.IsEnabled())
	})
}

func TestDetectForRuleSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real file
	realFile := filepath.Join(tmpDir, "real.json")
	require.NoError(t, os.WriteFile(realFile, []byte("{}"), 0644))

	// Create a subdirectory for symlinks
	symlinkDir := filepath.Join(tmpDir, "links")
	require.NoError(t, os.MkdirAll(symlinkDir, 0755))

	t.Run("detects valid symlink to file", func(t *testing.T) {
		symlink := filepath.Join(symlinkDir, "valid_link.json")
		require.NoError(t, os.Symlink(realFile, symlink))
		t.Cleanup(func() { _ = os.Remove(symlink) })

		rule := config.PackageManagerCfg{Include: []string{"**/*.json"}}
		files, err := detectForRule(tmpDir, rule)
		require.NoError(t, err)
		assert.Contains(t, files, symlink)
	})

	t.Run("skips broken symlink", func(t *testing.T) {
		brokenSymlink := filepath.Join(symlinkDir, "broken.json")
		require.NoError(t, os.Symlink("/nonexistent/file.json", brokenSymlink))
		t.Cleanup(func() { _ = os.Remove(brokenSymlink) })

		buf := &bytes.Buffer{}
		restore := warnings.SetWarningWriter(buf)
		defer restore()

		rule := config.PackageManagerCfg{Include: []string{"**/*.json"}}
		files, err := detectForRule(tmpDir, rule)
		require.NoError(t, err)

		// Broken symlink should not be in the results
		assert.NotContains(t, files, brokenSymlink)
		// Should warn about broken symlink
		assert.Contains(t, buf.String(), "skipping broken symlink")
	})

	t.Run("skips symlink to directory", func(t *testing.T) {
		// Create a directory to link to
		targetDir := filepath.Join(tmpDir, "target_dir")
		require.NoError(t, os.MkdirAll(targetDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(targetDir, "inner.json"), []byte("{}"), 0644))

		// Create symlink to directory
		dirSymlink := filepath.Join(symlinkDir, "dir_link.json")
		require.NoError(t, os.Symlink(targetDir, dirSymlink))
		t.Cleanup(func() { _ = os.Remove(dirSymlink) })

		rule := config.PackageManagerCfg{Include: []string{"links/*.json"}}
		files, err := detectForRule(tmpDir, rule)
		require.NoError(t, err)

		// Symlink to directory should not be treated as a file match
		assert.NotContains(t, files, dirSymlink)
	})
}
