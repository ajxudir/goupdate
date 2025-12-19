package testutil

import (
	"github.com/ajxudir/goupdate/pkg/config"
)

// ConfigBuilder provides a fluent API for building test configurations.
//
// Use this builder to construct Config objects for testing purposes
// without needing to set all required fields manually.
type ConfigBuilder struct {
	cfg config.Config
}

// NewConfig creates a new ConfigBuilder with default values.
//
// Initializes a builder with working directory set to "." and an
// empty rules map ready for configuration.
//
// Returns:
//   - *ConfigBuilder: New builder instance ready for method chaining
func NewConfig() *ConfigBuilder {
	return &ConfigBuilder{
		cfg: config.Config{
			WorkingDir: ".",
			Rules:      make(map[string]config.PackageManagerCfg),
		},
	}
}

// WithWorkingDir sets the working directory for the configuration.
//
// Parameters:
//   - dir: Path to the working directory
//
// Returns:
//   - *ConfigBuilder: Self for method chaining
func (b *ConfigBuilder) WithWorkingDir(dir string) *ConfigBuilder {
	b.cfg.WorkingDir = dir
	return b
}

// WithRule adds a rule to the configuration.
//
// Parameters:
//   - name: Rule identifier (e.g., "npm", "pip", "mod")
//   - rule: Package manager configuration for this rule
//
// Returns:
//   - *ConfigBuilder: Self for method chaining
func (b *ConfigBuilder) WithRule(name string, rule config.PackageManagerCfg) *ConfigBuilder {
	if b.cfg.Rules == nil {
		b.cfg.Rules = make(map[string]config.PackageManagerCfg)
	}
	b.cfg.Rules[name] = rule
	return b
}

// Build returns the built configuration.
//
// Returns a pointer to the constructed Config. The builder can be
// reused after calling Build.
//
// Returns:
//   - *config.Config: Pointer to the built configuration
func (b *ConfigBuilder) Build() *config.Config {
	return &b.cfg
}

// NPMRule creates a typical NPM rule configuration.
//
// Returns a pre-configured rule for NPM/JavaScript packages with
// standard fields, update commands, and outdated commands.
//
// Returns:
//   - config.PackageManagerCfg: NPM rule configuration
func NPMRule() config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Manager: "js",
		Format:  "json",
		Fields:  map[string]string{"dependencies": "prod", "devDependencies": "dev"},
		Update: &config.UpdateCfg{
			Commands: "npm install {{package}}@{{version}}",
		},
		Outdated: &config.OutdatedCfg{
			Commands: "npm view {{package}} versions --json",
		},
	}
}

// GoModRule creates a typical Go module rule configuration.
//
// Returns a pre-configured rule for Go module packages with standard
// raw format, update commands, and outdated commands.
//
// Returns:
//   - config.PackageManagerCfg: Go module rule configuration
func GoModRule() config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Manager: "go",
		Format:  "raw",
		Update: &config.UpdateCfg{
			Commands: "go get {{package}}@{{version}}",
		},
		Outdated: &config.OutdatedCfg{
			Commands: "go list -m -versions {{package}}",
		},
	}
}

// NuGetRule creates a typical NuGet rule configuration.
//
// Returns a pre-configured rule for .NET/NuGet packages with standard
// XML format, update commands, and outdated commands.
//
// Returns:
//   - config.PackageManagerCfg: NuGet rule configuration
func NuGetRule() config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Manager: "dotnet",
		Format:  "xml",
		Fields:  map[string]string{"ItemGroup/PackageReference": "prod"},
		Update: &config.UpdateCfg{
			Commands: "dotnet add package {{package}} --version {{version}}",
		},
		Outdated: &config.OutdatedCfg{
			Commands: "dotnet list package --outdated",
		},
	}
}

// SimpleRule creates a minimal rule with just an update command.
//
// Useful for tests that only need update functionality without
// full package manager configuration.
//
// Parameters:
//   - updateCmd: Command template to use for updates
//
// Returns:
//   - config.PackageManagerCfg: Minimal rule configuration
func SimpleRule(updateCmd string) config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Format: "json",
		Fields: map[string]string{"dependencies": "prod"},
		Update: &config.UpdateCfg{
			Commands: updateCmd,
		},
	}
}

// RuleWithGroup creates a rule with update group configured.
//
// Useful for tests that need to verify group-based update behavior.
//
// Parameters:
//   - updateCmd: Command template to use for updates
//   - group: Group name to assign to packages matching this rule
//
// Returns:
//   - config.PackageManagerCfg: Rule configuration with group
func RuleWithGroup(updateCmd, group string) config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Format: "json",
		Fields: map[string]string{"dependencies": "prod"},
		Update: &config.UpdateCfg{
			Commands: updateCmd,
			Group:    group,
		},
	}
}

// ComposerRule creates a typical Composer (PHP) rule configuration.
//
// Returns a pre-configured rule for Composer/PHP packages with
// standard fields, update commands, and outdated commands.
//
// Returns:
//   - config.PackageManagerCfg: Composer rule configuration
func ComposerRule() config.PackageManagerCfg {
	return config.PackageManagerCfg{
		Manager: "php",
		Format:  "json",
		Fields:  map[string]string{"require": "prod", "require-dev": "dev"},
		Update: &config.UpdateCfg{
			Commands: "composer require {{package}}:{{version}}",
		},
		Outdated: &config.OutdatedCfg{
			Commands: "composer show {{package}} --all --format=json",
		},
	}
}
