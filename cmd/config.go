package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/spf13/cobra"
)

var (
	configShowDefaultsFlag  bool
	configShowEffectiveFlag bool
	configInitFlag          bool
	configValidateFlag      bool
	configPathFlag          string
)

var (
	loadConfigFunc = config.LoadConfig
	writeFileFunc  = os.WriteFile
	readFileFunc   = os.ReadFile
)

// loadConfigWithoutValidation loads the configuration without strict schema validation.
// This is used by the scan command where we want to detect files even if some configs
// in subdirectories are malformed (e.g., test fixtures).
func loadConfigWithoutValidation(configPath, workDir string) (*config.Config, error) {
	cfg, err := loadConfigFunc(configPath, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

// loadAndValidateConfig loads the configuration and validates it for unknown fields.
//
// This provides preflight validation to catch configuration errors early,
// ensuring users are notified of typos or deprecated options before processing.
//
// Parameters:
//   - configPath: Path to custom config file, or empty for default location
//   - workDir: Working directory to search for default config
//
// Returns:
//   - *config.Config: Loaded and validated configuration
//   - error: Validation or load error with details
func loadAndValidateConfig(configPath, workDir string) (*config.Config, error) {
	// If a custom config path is specified, validate it first
	if configPath != "" {
		data, err := readFileFunc(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file '%s': %w", configPath, err)
		}

		result := config.ValidateConfigFile(data)
		if result.HasErrors() {
			var errBuilder strings.Builder
			errBuilder.WriteString(fmt.Sprintf("configuration validation failed for %s:\n", configPath))
			for _, e := range result.Errors {
				errBuilder.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
			}
			errBuilder.WriteString("\n💡 Run 'goupdate config --validate' for details, or see docs/configuration.md")
			verbose.Infof("Exit code %d (config error): configuration validation failed for custom config %s", errors.ExitConfigError, configPath)
			return nil, errors.NewExitError(errors.ExitConfigError, fmt.Errorf("%s", errBuilder.String()))
		}
	} else {
		// Check for .goupdate.yml in workDir and validate if it exists
		localConfig := workDir + "/.goupdate.yml"
		if data, err := readFileFunc(localConfig); err == nil {
			result := config.ValidateConfigFile(data)
			if result.HasErrors() {
				var errBuilder strings.Builder
				errBuilder.WriteString(fmt.Sprintf("configuration validation failed for %s:\n", localConfig))
				for _, e := range result.Errors {
					errBuilder.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
				}
				errBuilder.WriteString("\n💡 Run 'goupdate config --validate' for details, or see docs/configuration.md")
				verbose.Infof("Exit code %d (config error): configuration validation failed for local config %s", errors.ExitConfigError, localConfig)
				return nil, errors.NewExitError(errors.ExitConfigError, fmt.Errorf("%s", errBuilder.String()))
			}
		}
	}

	// Load the config normally
	cfg, err := loadConfigFunc(configPath, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or create configuration",
	Long:  `Show or create configuration files.`,
	RunE:  runConfig,
}

func init() {
	configCmd.Flags().BoolVar(&configShowDefaultsFlag, "show-defaults", false, "Show default configuration")
	configCmd.Flags().BoolVar(&configShowEffectiveFlag, "show-effective", false, "Show effective configuration")
	configCmd.Flags().BoolVar(&configInitFlag, "init", false, "Create .goupdate.yml template")
	configCmd.Flags().BoolVar(&configValidateFlag, "validate", false, "Validate configuration file (rejects unknown fields)")
	configCmd.Flags().StringVarP(&configPathFlag, "config", "c", "", "Config file path to validate")
}

// runConfig executes the config command with the specified flags.
//
// Behavior depends on flags:
//   - --init: Creates a .goupdate.yml template file
//   - --validate: Validates the configuration file for schema errors
//   - --show-defaults: Displays the default configuration
//   - --show-effective: Displays the effective merged configuration
//
// Parameters:
//   - cmd: Cobra command instance
//   - args: Command line arguments
//
// Returns:
//   - error: Returns error on validation or file operation failure
func runConfig(cmd *cobra.Command, args []string) error {
	if configInitFlag {
		return createConfigTemplate()
	}

	if configValidateFlag {
		return validateConfigFile()
	}

	if configShowDefaultsFlag {
		defaults := config.GetDefaultConfig()
		fmt.Println("Default configuration:")
		fmt.Println()
		fmt.Println(defaults)
		return nil
	}

	if configShowEffectiveFlag {
		workDir, _ := os.Getwd()
		cfg, err := loadConfigFunc("", workDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("Effective configuration:")
		fmt.Println()
		fmt.Printf("Working Directory: %s\n", cfg.WorkingDir)
		fmt.Printf("Rules: %d\n\n", len(cfg.Rules))

		for key, rule := range cfg.Rules {
			fmt.Printf("Rule: %s\n", key)
			if !rule.IsEnabled() {
				fmt.Printf("  Enabled: false\n")
			}
			fmt.Printf("  Manager: %s\n", rule.Manager)
			fmt.Printf("  Include: %s\n", strings.Join(rule.Include, ", "))
			if len(rule.Exclude) > 0 {
				fmt.Printf("  Exclude: %s\n", strings.Join(rule.Exclude, ", "))
			}
			fmt.Println()
		}

		// Show system_tests configuration if present
		if cfg.SystemTests != nil && len(cfg.SystemTests.Tests) > 0 {
			fmt.Println("System Tests:")
			fmt.Printf("  Run Preflight: %v\n", cfg.SystemTests.IsRunPreflight())
			fmt.Printf("  Run Mode: %s\n", cfg.SystemTests.GetRunMode())
			fmt.Printf("  Stop on Fail: %v\n", cfg.SystemTests.IsStopOnFail())
			fmt.Printf("  Tests: %d\n", len(cfg.SystemTests.Tests))
			for _, test := range cfg.SystemTests.Tests {
				fmt.Printf("    - %s\n", test.Name)
				if test.TimeoutSeconds > 0 {
					fmt.Printf("      Timeout: %ds\n", test.TimeoutSeconds)
				}
				if test.ContinueOnFail {
					fmt.Printf("      Continue on Fail: true\n")
				}
			}
			fmt.Println()
		}
		return nil
	}

	return cmd.Help()
}

// validateConfigFile validates the configuration file at the specified path.
//
// If no path is specified via --config flag, validates .goupdate.yml in the
// current working directory. Reports validation errors and warnings.
//
// Returns:
//   - error: Returns ExitError with ExitConfigError code on validation failure
func validateConfigFile() error {
	configPath := configPathFlag
	if configPath == "" {
		// Try default location
		workDir, _ := os.Getwd()
		configPath = workDir + "/.goupdate.yml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file '%s': %w", configPath, err)
	}

	result := config.ValidateConfigFile(data)

	if result.HasErrors() {
		fmt.Printf("%s Configuration validation failed for: %s\n\n", constants.IconError, configPath)

		// Use verbose errors when --verbose flag is set
		if verbose.IsEnabled() {
			for _, e := range result.Errors {
				fmt.Printf("  ERROR: %s\n", e.VerboseError())
			}
		} else {
			for _, e := range result.Errors {
				fmt.Printf("  ERROR: %s\n", e.Error())
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println()
			for _, w := range result.Warnings {
				fmt.Printf("  WARNING: %s\n", w)
			}
		}
		fmt.Println()
		if !verbose.IsEnabled() {
			fmt.Printf("%s Run with --verbose for detailed schema information\n", constants.IconLightbulb)
		}
		fmt.Printf("%s See docs/configuration.md for valid configuration options\n", constants.IconLightbulb)
		verbose.Infof("Exit code %d (config error): configuration validation failed for %s", errors.ExitConfigError, configPath)
		return errors.NewExitError(errors.ExitConfigError, fmt.Errorf("configuration validation failed"))
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("%s Configuration valid with warnings: %s\n\n", constants.IconWarn, configPath)
		for _, w := range result.Warnings {
			fmt.Printf("  WARNING: %s\n", w)
		}
		fmt.Println()
	} else {
		fmt.Printf("%s Configuration valid: %s\n", constants.IconCheckmarkBox, configPath)
	}

	return nil
}

// createConfigTemplate creates a new .goupdate.yml template file.
//
// The template is created in the current directory. Fails if a config
// file already exists at that location.
//
// Returns:
//   - error: Returns error if file exists or cannot be created
func createConfigTemplate() error {
	configPath := ".goupdate.yml"
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists: %s", configPath)
	}

	// Use embedded template from pkg/config/template.yml
	template := config.GetTemplateConfig()

	// Use 0600 permissions for config files (owner read/write only) for security
	if err := writeFileFunc(configPath, []byte(template), 0600); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("Created configuration template: %s\n", configPath)
	return nil
}

// resolveWorkingDir determines the working directory to use.
//
// Priority order:
//  1. Flag value (if specified and not ".")
//  2. Config WorkingDir (if specified)
//  3. Current directory (".")
//
// Parameters:
//   - flagValue: Value from --dir flag
//   - cfg: Loaded configuration (may be nil)
//
// Returns:
//   - string: Resolved working directory path
func resolveWorkingDir(flagValue string, cfg *config.Config) string {
	if flagValue != "" && flagValue != "." {
		return flagValue
	}

	if cfg != nil && cfg.WorkingDir != "" {
		return cfg.WorkingDir
	}

	return "."
}
