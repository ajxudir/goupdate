package config

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed default.yml
var defaultConfigYAML string

//go:embed template.yml
var templateConfigYAML string

// loadDefaultConfig loads the embedded default configuration.
//
// This unmarshals the embedded default.yml file into a Config structure.
// If unmarshaling fails, returns an empty config with initialized Rules map.
//
// Returns:
//   - *Config: the default configuration
func loadDefaultConfig() *Config {
	var cfg Config
	if err := yaml.Unmarshal([]byte(defaultConfigYAML), &cfg); err == nil {
		return &cfg
	}
	return &Config{Rules: make(map[string]PackageManagerCfg)}
}

// GetDefaultConfig returns the embedded default configuration YAML.
//
// This returns the raw YAML string from the embedded default.yml file.
// Useful for displaying or saving the default configuration.
//
// Returns:
//   - string: the default configuration as YAML
func GetDefaultConfig() string {
	return defaultConfigYAML
}

// GetTemplateConfig returns the embedded template configuration YAML.
//
// This returns the raw YAML string from the embedded template.yml file.
// Useful for generating starter configuration files for users.
//
// Returns:
//   - string: the template configuration as YAML
func GetTemplateConfig() string {
	return templateConfigYAML
}
