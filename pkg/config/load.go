// Package config handles configuration loading, validation, and merging for goupdate.
// It supports YAML-based configuration files with inheritance (extends), rule-based
// package manager definitions, and package-specific overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/goupdate/pkg/verbose"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from the specified path or defaults.
//
// If configPath is provided, it loads that specific config file.
// Otherwise, it looks for .goupdate.yml in the working directory.
// If no config is found, it returns the built-in default configuration.
// Supports config inheritance via the extends mechanism.
//
// Parameters:
//   - configPath: path to the config file, or empty to use defaults
//   - workDir: working directory for the configuration
//
// Returns:
//   - *Config: the loaded and merged configuration
//   - error: any error encountered during loading or validation
func LoadConfig(configPath, workDir string) (*Config, error) {
	var cfg *Config
	var extended []string

	if configPath != "" {
		verbose.Infof("Loading config from: %s", configPath)
		// Load specified config
		loaded, err := loadConfigFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		cfg = loaded
		cfg.SetRootConfig(true) // Mark as root config for security policy enforcement
		extended = cfg.Extends

		// Process extends with security settings from root config
		cfg, err = processExtendsSecure(cfg, filepath.Dir(configPath), cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to process extends: %w", err)
		}
		verbose.ConfigLoaded(configPath, extended)
	} else {
		// Try .goupdate.yml in working directory
		localConfig := filepath.Join(workDir, ".goupdate.yml")
		if _, err := os.Stat(localConfig); err == nil {
			verbose.Infof("Found local config: %s", localConfig)
			loaded, err := loadConfigFile(localConfig)
			if err == nil {
				cfg = loaded
				cfg.SetRootConfig(true) // Mark as root config
				extended = cfg.Extends
				// Process extends with security settings from root config
				cfg, err = processExtendsSecure(cfg, workDir, cfg)
				if err != nil {
					return nil, fmt.Errorf("failed to process extends: %w", err)
				}
				verbose.ConfigLoaded(localConfig, extended)
			}
		}

		if cfg == nil {
			verbose.Info("Using built-in default configuration")
			// Use defaults
			cfg = loadDefaultConfig()
			cfg.SetRootConfig(true)
		}
	}

	if workDir != "" {
		cfg.WorkingDir = workDir
	} else if cfg.WorkingDir == "" {
		cfg.WorkingDir = "."
	}

	if err := validateGroupMembership(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadConfigFileWithLimit loads a config file with a configurable size limit.
//
// This enforces a maximum file size to prevent memory exhaustion attacks.
// The size limit can be configured via security settings in the root config.
//
// Parameters:
//   - path: path to the config file
//   - maxSize: maximum allowed file size in bytes
//
// Returns:
//   - *Config: the loaded configuration
//   - error: error if file is too large, not found, or has invalid YAML
func loadConfigFileWithLimit(path string, maxSize int64) (*Config, error) {
	// Check file size before reading to prevent memory exhaustion
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxSize {
		return nil, fmt.Errorf("config file too large: %d bytes (max %d bytes)\n\n"+
			"💡 To increase this limit, add to your root config:\n"+
			"   security:\n"+
			"     max_config_file_size: %d  # or larger value in bytes",
			info.Size(), maxSize, info.Size()*2)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return loadConfigData(data)
}

// loadConfigFile loads a config file with the default size limit.
//
// This is a convenience wrapper around loadConfigFileWithLimit using the
// default maximum file size of 10MB.
//
// Parameters:
//   - path: path to the config file
//
// Returns:
//   - *Config: the loaded configuration
//   - error: error if file cannot be loaded or parsed
func loadConfigFile(path string) (*Config, error) {
	return loadConfigFileWithLimit(path, DefaultMaxConfigFileSize)
}

// loadConfigData parses YAML configuration data.
//
// This unmarshals the YAML data into a Config struct and initializes
// empty maps as needed.
//
// Parameters:
//   - data: YAML configuration data as bytes
//
// Returns:
//   - *Config: the parsed configuration
//   - error: error if YAML is invalid or malformed
func loadConfigData(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if cfg.Rules == nil {
		cfg.Rules = make(map[string]PackageManagerCfg)
	}

	return &cfg, nil
}

// LoadConfigFileStrict loads a config file and validates for unknown fields.
//
// This is more strict than LoadConfig - it will return an error if the config
// contains any unknown fields or validation issues. Useful for catching typos
// and configuration errors early.
//
// Parameters:
//   - path: path to the config file
//
// Returns:
//   - *Config: the loaded configuration
//   - error: error if file has unknown fields, validation errors, or invalid YAML
func LoadConfigFileStrict(path string) (*Config, error) {
	// Check file size before reading to prevent memory exhaustion
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > DefaultMaxConfigFileSize {
		return nil, fmt.Errorf("config file too large: %d bytes (max %d bytes)\n\n"+
			"💡 To increase this limit, add to your root config:\n"+
			"   security:\n"+
			"     max_config_file_size: %d  # or larger value in bytes",
			info.Size(), DefaultMaxConfigFileSize, info.Size()*2)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Validate for unknown fields
	result := ValidateConfigFile(data)
	if result.HasErrors() {
		return nil, fmt.Errorf("%s", result.ErrorMessages())
	}

	return loadConfigData(data)
}

// processExtends processes the extends inheritance chain.
//
// This is a convenience wrapper that uses the config itself as the root config
// for security policy enforcement.
//
// Parameters:
//   - cfg: the configuration to process
//   - baseDir: base directory for resolving relative paths
//
// Returns:
//   - *Config: the merged configuration after processing extends
//   - error: error if extends chain has cycles or files cannot be loaded
func processExtends(cfg *Config, baseDir string) (*Config, error) {
	return processExtendsSecure(cfg, baseDir, cfg)
}

// processExtendsSecure processes extends with security policy enforcement from root config.
//
// This handles the extends inheritance chain while enforcing security policies
// (path traversal, absolute paths, file size limits) from the root configuration.
//
// Parameters:
//   - cfg: the configuration to process
//   - baseDir: base directory for resolving relative paths
//   - rootCfg: the root configuration containing security settings
//
// Returns:
//   - *Config: the merged configuration after processing extends
//   - error: error if security policies are violated or extends chain is invalid
func processExtendsSecure(cfg *Config, baseDir string, rootCfg *Config) (*Config, error) {
	return processExtendsWithStackSecure(cfg, baseDir, make(map[string]bool), rootCfg)
}

// validateExtendPath checks if an extend path is allowed based on security settings.
//
// This enforces security policies for extends paths:
//   - Path traversal (..) is blocked by default
//   - Absolute paths are blocked by default
//
// Returns an error with helpful message if the path is not allowed.
//
// Parameters:
//   - extend: the extend path to validate
//   - baseDir: base directory (unused, kept for future use)
//   - rootCfg: the root configuration containing security settings
//
// Returns:
//   - error: error if path violates security policy, nil if allowed
func validateExtendPath(extend string, baseDir string, rootCfg *Config) error {
	// Check for path traversal (../)
	if strings.Contains(extend, "..") {
		if !rootCfg.AllowsPathTraversal() {
			return fmt.Errorf("path traversal not allowed in extends: '%s' - "+
				"to allow, add security.allow_path_traversal: true to your root config",
				extend)
		}
	}

	// Check for absolute paths
	if filepath.IsAbs(extend) {
		if !rootCfg.AllowsAbsolutePaths() {
			return fmt.Errorf("absolute paths not allowed in extends: '%s' - "+
				"to allow, add security.allow_absolute_paths: true to your root config",
				extend)
		}
	}

	return nil
}

// processExtendsWithStackSecure processes extends with cycle detection and security enforcement.
//
// This recursively processes the extends chain, merging configurations in order.
// It maintains a stack to detect circular dependencies and enforces security
// policies from the root configuration.
//
// Parameters:
//   - cfg: the configuration to process
//   - baseDir: base directory for resolving relative paths
//   - stack: map tracking visited configs to detect cycles
//   - rootCfg: the root configuration containing security settings
//
// Returns:
//   - *Config: the merged configuration after processing all extends
//   - error: error if cycle detected, security policy violated, or file cannot be loaded
func processExtendsWithStackSecure(cfg *Config, baseDir string, stack map[string]bool, rootCfg *Config) (*Config, error) {
	if len(cfg.Extends) == 0 {
		return cfg, nil
	}

	// Start with empty base
	base := &Config{
		Rules: make(map[string]PackageManagerCfg),
	}

	// Get max file size from root config
	maxFileSize := rootCfg.GetMaxConfigFileSize()

	// Process extends in order
	for _, extend := range cfg.Extends {
		var (
			extendCfg  *Config
			extendKey  string
			cleanupKey bool
		)

		if extend == "default" {
			extendKey = "__default__"
			if stack[extendKey] {
				return nil, fmt.Errorf("cyclic extends detected at %s", extend)
			}
			stack[extendKey] = true
			cleanupKey = true
			extendCfg = loadDefaultConfig()
		} else {
			// Validate path security before processing
			if err := validateExtendPath(extend, baseDir, rootCfg); err != nil {
				return nil, err
			}

			// Resolve relative path
			extendPath := extend
			if !filepath.IsAbs(extendPath) {
				extendPath = filepath.Join(baseDir, extend)
			}

			absPath, absErr := filepath.Abs(extendPath)
			if absErr != nil {
				return nil, fmt.Errorf("failed to resolve extend path '%s': %w", extend, absErr)
			}

			if _, statErr := os.Stat(absPath); statErr != nil {
				return nil, fmt.Errorf("failed to resolve extend '%s': %w", extend, statErr)
			}

			extendKey = absPath
			if stack[extendKey] {
				return nil, fmt.Errorf("cyclic extends detected at %s", extendPath)
			}

			stack[extendKey] = true
			cleanupKey = true

			// Load with configurable size limit from root config
			loaded, err := loadConfigFileWithLimit(extendPath, maxFileSize)
			if err != nil {
				return nil, fmt.Errorf("failed to load extend '%s': %w", extend, err)
			}

			// Recursively process extends in the extended config (using root config's security settings)
			loaded, err = processExtendsWithStackSecure(loaded, filepath.Dir(extendPath), stack, rootCfg)
			if err != nil {
				return nil, err
			}

			extendCfg = loaded
		}

		base = mergeConfigs(base, extendCfg)
		verbose.Printf("Extended from %q: merged %d rules\n", extend, len(extendCfg.Rules))

		if cleanupKey {
			delete(stack, extendKey)
		}
	}

	// Merge current config on top
	result := mergeConfigs(base, cfg)
	result.Extends = nil // Clear extends after processing

	verbose.Printf("Config extends complete: total %d rules configured\n", len(result.Rules))

	return result, nil
}
