package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/goupdate/pkg/config"
	"github.com/user/goupdate/pkg/errors"
	"github.com/user/goupdate/pkg/verbose"
)

// TestExecute tests the behavior of ExecuteTest with --help flag.
//
// It verifies:
//   - Help flag returns successfully
func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"goupdate", "--help"}
	err := ExecuteTest()
	assert.NoError(t, err)
}

// TestExecuteWrapper tests the behavior of Execute wrapper function.
//
// It verifies:
//   - Execute function covers the success path
func TestExecuteWrapper(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Ensure Execute covers the success path without exiting
	os.Args = []string{"goupdate", "--help"}
	Execute()
}

// TestExecuteError tests the behavior of Execute with invalid commands.
//
// It verifies:
//   - Unknown commands return ExitFailure code
//   - Error messages are written to stderr
func TestExecuteError(t *testing.T) {
	oldExit := exitFunc
	oldArgs := os.Args
	rootCmd.SilenceErrors = false
	rootCmd.SilenceUsage = true

	defer func() {
		exitFunc = oldExit
		os.Args = oldArgs
		rootCmd.SilenceErrors = false
		rootCmd.SilenceUsage = false
		rootCmd.SetArgs(nil)
	}()

	exitCode := 0
	exitFunc = func(code int) { exitCode = code }
	rootCmd.SetArgs([]string{"unknown"})
	os.Args = []string{"goupdate", "unknown"}

	output := captureStderr(t, Execute)

	assert.Equal(t, errors.ExitFailure, exitCode) // Unknown command returns errors.ExitFailure (2)
	assert.Contains(t, output, "Error: unknown command")
}

// TestConfigCommand tests the behavior of config command with various flags.
//
// It verifies:
//   - Config --show-defaults displays default configuration
//   - Config --init creates new config file
//   - Config --show-effective shows effective configuration
//   - Init fails when config file already exists
//   - Help is shown when no flags are set
func TestConfigCommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	t.Run("show-defaults", func(t *testing.T) {
		oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
		defer func() {
			configInitFlag = oldInit
			configShowDefaultsFlag = oldDefaults
			configShowEffectiveFlag = oldEffective
		}()

		configShowDefaultsFlag = false
		configInitFlag = false
		configShowEffectiveFlag = false
		os.Args = []string{"goupdate", "config", "--show-defaults"}
		err := ExecuteTest()
		assert.NoError(t, err)
	})

	t.Run("init", func(t *testing.T) {
		oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
		defer func() {
			configInitFlag = oldInit
			configShowDefaultsFlag = oldDefaults
			configShowEffectiveFlag = oldEffective
		}()

		configShowDefaultsFlag = false
		configInitFlag = false
		configShowEffectiveFlag = false
		oldWD, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWD) }()

		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))

		os.Args = []string{"goupdate", "config", "--init"}
		err = ExecuteTest()
		assert.NoError(t, err)

		_, err = os.Stat(".goupdate.yml")
		assert.NoError(t, err)
	})

	t.Run("show-effective", func(t *testing.T) {
		oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
		defer func() {
			configInitFlag = oldInit
			configShowDefaultsFlag = oldDefaults
			configShowEffectiveFlag = oldEffective
		}()

		configShowDefaultsFlag = false
		configInitFlag = false
		configShowEffectiveFlag = false
		oldWD, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWD) }()

		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))

		os.Args = []string{"goupdate", "config", "--show-effective"}
		err = ExecuteTest()
		assert.NoError(t, err)
	})

	t.Run("init fails when exists", func(t *testing.T) {
		oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
		defer func() {
			configInitFlag = oldInit
			configShowDefaultsFlag = oldDefaults
			configShowEffectiveFlag = oldEffective
		}()

		configShowDefaultsFlag = false
		configInitFlag = true
		configShowEffectiveFlag = false
		oldWD, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWD) }()

		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		require.NoError(t, os.WriteFile(".goupdate.yml", []byte("rules: {}"), 0644))

		err = runConfig(nil, nil)
		assert.Error(t, err)
	})

	t.Run("help path", func(t *testing.T) {
		oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
		defer func() {
			configInitFlag = oldInit
			configShowDefaultsFlag = oldDefaults
			configShowEffectiveFlag = oldEffective
		}()

		configShowDefaultsFlag = false
		configInitFlag = false
		configShowEffectiveFlag = false
		oldWD, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWD) }()

		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))

		// Reset flags so no branch triggers and help path is used
		configInitFlag = false
		configShowDefaultsFlag = false
		configShowEffectiveFlag = false

		err = runConfig(&cobra.Command{}, nil)
		assert.NoError(t, err)
	})
}

// TestRunConfigEffectiveError tests the behavior of runConfig when loading fails.
//
// It verifies:
//   - Config load failure returns appropriate error
func TestRunConfigEffectiveError(t *testing.T) {
	oldLoad := loadConfigFunc
	defer func() { loadConfigFunc = oldLoad }()

	loadConfigFunc = func(configPath, baseDir string) (*config.Config, error) {
		return nil, fmt.Errorf("load failure")
	}

	oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
	defer func() {
		configInitFlag = oldInit
		configShowDefaultsFlag = oldDefaults
		configShowEffectiveFlag = oldEffective
	}()

	configInitFlag = false
	configShowDefaultsFlag = false
	configShowEffectiveFlag = true
	oldWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWD) }()

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	err = runConfig(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config")
}

// TestCreateConfigTemplateWriteError tests the behavior of createConfigTemplate with write errors.
//
// It verifies:
//   - Write errors are properly handled and reported
func TestCreateConfigTemplateWriteError(t *testing.T) {
	oldWrite := writeFileFunc
	defer func() { writeFileFunc = oldWrite }()

	writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
		return fmt.Errorf("write failure")
	}

	oldWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWD) }()

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	err = createConfigTemplate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create config file")
}

// TestResolveWorkingDir tests the behavior of resolveWorkingDir.
//
// It verifies:
//   - Explicit working directory takes precedence
//   - Config working directory is used when no explicit directory
//   - Falls back to current directory when config has no working directory
func TestResolveWorkingDir(t *testing.T) {
	cfg := &config.Config{WorkingDir: "/cfg"}

	assert.Equal(t, "/explicit", resolveWorkingDir("/explicit", cfg))
	assert.Equal(t, "/cfg", resolveWorkingDir("", cfg))
	assert.Equal(t, ".", resolveWorkingDir("", &config.Config{}))
}

// captureStdout is a test helper that captures stdout during function execution.
//
// Parameters:
//   - t: The testing instance
//   - fn: The function to execute while capturing stdout
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// captureStderr is a test helper that captures stderr during function execution.
//
// Parameters:
//   - t: The testing instance
//   - fn: The function to execute while capturing stderr
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// TestLoadAndValidateConfig tests the behavior of loadAndValidateConfig.
//
// It verifies:
//   - Valid config files are loaded successfully
//   - Config files with unknown fields are rejected
//   - Missing config files return appropriate errors
//   - Local config files with unknown fields are rejected
//   - Default config is used when no local config exists
//   - Valid local config files are loaded successfully
//   - Config load failures after validation are handled
func TestLoadAndValidateConfig(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    include: ["**/package.json"]
`), 0644)
		require.NoError(t, err)

		cfg, err := loadAndValidateConfig(configPath, tmpDir)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("config file with unknown field", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    unknown_field: invalid
`), 0644)
		require.NoError(t, err)

		cfg, err := loadAndValidateConfig(configPath, tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration validation failed")
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("config file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/nonexistent.yml"

		cfg, err := loadAndValidateConfig(configPath, tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("local config with unknown field", func(t *testing.T) {
		tmpDir := t.TempDir()
		localConfig := tmpDir + "/.goupdate.yml"
		err := os.WriteFile(localConfig, []byte(`
rules:
  custom:
    typo_field: oops
`), 0644)
		require.NoError(t, err)

		cfg, err := loadAndValidateConfig("", tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration validation failed")
	})

	t.Run("no local config uses defaults", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg, err := loadAndValidateConfig("", tmpDir)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("valid local config", func(t *testing.T) {
		tmpDir := t.TempDir()
		localConfig := tmpDir + "/.goupdate.yml"
		err := os.WriteFile(localConfig, []byte(`
extends:
  - default
rules:
  npm:
    enabled: false
`), 0644)
		require.NoError(t, err)

		cfg, err := loadAndValidateConfig("", tmpDir)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("config load failure after validation", func(t *testing.T) {
		oldLoad := loadConfigFunc
		defer func() { loadConfigFunc = oldLoad }()

		loadConfigFunc = func(configPath, baseDir string) (*config.Config, error) {
			return nil, fmt.Errorf("simulated load failure")
		}

		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`rules: {}`), 0644)
		require.NoError(t, err)

		cfg, err := loadAndValidateConfig(configPath, tmpDir)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to load config")
	})
}

// TestLoadAndValidateConfigExitCode tests the behavior of loadAndValidateConfig exit codes.
//
// It verifies:
//   - Config validation errors return ExitConfigError code
func TestLoadAndValidateConfigExitCode(t *testing.T) {
	t.Run("returns config error exit code", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    invalid_key: true
`), 0644)
		require.NoError(t, err)

		_, err = loadAndValidateConfig(configPath, tmpDir)
		assert.Error(t, err)

		// Verify it returns an ExitError with correct code
		code := errors.GetExitCode(err)
		assert.Equal(t, errors.ExitConfigError, code)
	})
}

// TestRunConfigShowEffectiveWithSystemTests tests the behavior of runConfig --show-effective with system tests.
//
// It verifies:
//   - Effective config includes system tests configuration
func TestRunConfigShowEffectiveWithSystemTests(t *testing.T) {
	oldLoad := loadConfigFunc
	defer func() { loadConfigFunc = oldLoad }()

	runPreflight := true
	stopOnFail := false
	loadConfigFunc = func(configPath, baseDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: baseDir,
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Include: []string{"**/package.json"}},
			},
			SystemTests: &config.SystemTestsCfg{
				RunMode:      "after_each",
				RunPreflight: &runPreflight,
				StopOnFail:   &stopOnFail,
				Tests: []config.SystemTestCfg{
					{Name: "unit-tests", Commands: "go test ./...", TimeoutSeconds: 60, ContinueOnFail: true},
					{Name: "integration", Commands: "make test"},
				},
			},
		}, nil
	}

	oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
	defer func() {
		configInitFlag = oldInit
		configShowDefaultsFlag = oldDefaults
		configShowEffectiveFlag = oldEffective
	}()

	configInitFlag = false
	configShowDefaultsFlag = false
	configShowEffectiveFlag = true

	output := captureStdout(t, func() {
		err := runConfig(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Effective configuration")
	assert.Contains(t, output, "System Tests:")
	assert.Contains(t, output, "Run Preflight: true")
	assert.Contains(t, output, "Run Mode: after_each")
	assert.Contains(t, output, "Stop on Fail: false")
	assert.Contains(t, output, "unit-tests")
	assert.Contains(t, output, "Timeout: 60s")
	assert.Contains(t, output, "Continue on Fail: true")
	assert.Contains(t, output, "integration")
}

// TestRunConfigShowEffectiveWithDisabledRule tests the behavior of runConfig --show-effective with disabled rules.
//
// It verifies:
//   - Effective config shows disabled rules
func TestRunConfigShowEffectiveWithDisabledRule(t *testing.T) {
	oldLoad := loadConfigFunc
	defer func() { loadConfigFunc = oldLoad }()

	enabled := false
	loadConfigFunc = func(configPath, baseDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: baseDir,
			Rules: map[string]config.PackageManagerCfg{
				"npm": {Manager: "js", Include: []string{"**/package.json"}, Enabled: &enabled},
			},
		}, nil
	}

	oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
	defer func() {
		configInitFlag = oldInit
		configShowDefaultsFlag = oldDefaults
		configShowEffectiveFlag = oldEffective
	}()

	configInitFlag = false
	configShowDefaultsFlag = false
	configShowEffectiveFlag = true

	output := captureStdout(t, func() {
		err := runConfig(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Enabled: false")
}

// TestRunConfigShowEffectiveWithExclude tests the behavior of runConfig --show-effective with exclude patterns.
//
// It verifies:
//   - Effective config shows exclude patterns
func TestRunConfigShowEffectiveWithExclude(t *testing.T) {
	oldLoad := loadConfigFunc
	defer func() { loadConfigFunc = oldLoad }()

	loadConfigFunc = func(configPath, baseDir string) (*config.Config, error) {
		return &config.Config{
			WorkingDir: baseDir,
			Rules: map[string]config.PackageManagerCfg{
				"npm": {
					Manager: "js",
					Include: []string{"**/package.json"},
					Exclude: []string{"**/node_modules/**"},
				},
			},
		}, nil
	}

	oldInit, oldDefaults, oldEffective := configInitFlag, configShowDefaultsFlag, configShowEffectiveFlag
	defer func() {
		configInitFlag = oldInit
		configShowDefaultsFlag = oldDefaults
		configShowEffectiveFlag = oldEffective
	}()

	configInitFlag = false
	configShowDefaultsFlag = false
	configShowEffectiveFlag = true

	output := captureStdout(t, func() {
		err := runConfig(nil, nil)
		assert.NoError(t, err)
	})

	assert.Contains(t, output, "Exclude:")
	assert.Contains(t, output, "**/node_modules/**")
}

// TestConfigValidateCommand tests the behavior of config validate command.
//
// It verifies:
//   - Valid config files pass validation
//   - Invalid config files are rejected with appropriate error
//   - Missing config files are handled properly
func TestConfigValidateCommand(t *testing.T) {
	t.Run("validates valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    include: ["**/package.json"]
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.NoError(t, err)
		assert.Contains(t, output, "Configuration valid")
	})

	t.Run("rejects config with unknown fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    typo_field: invalid
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.Error(t, err)
		assert.Contains(t, output, "validation failed")
	})

	t.Run("uses default location when no path specified", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldWD, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWD) }()
		require.NoError(t, os.Chdir(tmpDir))

		err = os.WriteFile(".goupdate.yml", []byte(`
rules:
  npm:
    manager: js
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = ""
		configValidateFlag = true

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.NoError(t, err)
		assert.Contains(t, output, "Configuration valid")
	})

	t.Run("reports file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/nonexistent.yml"

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		err := runConfig(nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("shows warnings with errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		// Create a config with semantic error (missing test name) AND warning (empty group)
		// This covers lines 181-185 in validateConfigFile where warnings are printed alongside errors
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    groups:
      empty_group:
        packages: []
system_tests:
  tests:
    - commands: "exit 0"
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.Error(t, err)
		assert.Contains(t, output, "validation failed")
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "WARNING")
		assert.Contains(t, output, "empty_group")
	})

	t.Run("shows only warnings when valid", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		// system_tests with empty tests generates a warning
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
system_tests:
  run_mode: after_all
  tests: []
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.NoError(t, err)
		assert.Contains(t, output, "valid with warnings")
	})

	t.Run("shows verbose errors when flag set", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := tmpDir + "/config.yml"
		err := os.WriteFile(configPath, []byte(`
rules:
  npm:
    manager: js
    typo_field: invalid
`), 0644)
		require.NoError(t, err)

		oldPath := configPathFlag
		oldValidate := configValidateFlag
		defer func() {
			configPathFlag = oldPath
			configValidateFlag = oldValidate
		}()

		configPathFlag = configPath
		configValidateFlag = true

		// Enable verbose mode
		verbose.Enable()
		defer verbose.Disable()

		output := captureStdout(t, func() {
			err = runConfig(nil, nil)
		})

		assert.Error(t, err)
		assert.Contains(t, output, "validation failed")
		// When verbose is enabled, shouldn't show hint to run with --verbose
		assert.NotContains(t, output, "Run with --verbose for detailed")
	})
}
