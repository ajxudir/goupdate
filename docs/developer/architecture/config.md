# Config Command Architecture

> The `config` command manages configuration files - showing defaults, effective config, validating configuration, and creating templates.

## Table of Contents

- [Command Overview](#command-overview)
- [Key Files](#key-files)
- [Subcommands](#subcommands)
- [Validation](#validation)
- [Preflight Config Validation](#preflight-config-validation)
- [Template Structure](#template-structure)
- [Working Directory Resolution](#working-directory-resolution)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Command Overview

```bash
goupdate config [flags]

Flags:
      --show-defaults    Show default configuration
      --show-effective   Show effective configuration
      --init             Create .goupdate.yml template
      --validate         Validate configuration file (rejects unknown fields)
  -c, --config           Config file path to validate (default: .goupdate.yml)
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/config.go` | Command definition and handlers |
| `pkg/config/defaults.go` | Embedded default configuration |
| `pkg/config/default.yml` | Default rule definitions |
| `pkg/config/validate.go` | Configuration validation logic |

## Subcommands

### Show Defaults

```bash
goupdate config --show-defaults
```

**Behavior:**
1. Returns embedded `default.yml` content
2. Shows all built-in rules (npm, composer, requirements, etc.)
3. Useful for understanding available configuration options

**Location:** `cmd/config.go:41-47`

```go
if configShowDefaultsFlag {
    defaults := config.GetDefaultConfig()
    fmt.Println("Default configuration:")
    fmt.Println(defaults)
    return nil
}
```

### Show Effective

```bash
goupdate config --show-effective
```

**Behavior:**
1. Loads config using standard resolution (extends, merging)
2. Displays working directory
3. Lists all resolved rules with their settings

**Location:** `cmd/config.go:49-71`

```go
if configShowEffectiveFlag {
    cfg, err := loadConfigFunc("", workDir)
    // Display working directory, rule count
    for key, rule := range cfg.Rules {
        // Display: Manager, Include, Exclude
    }
}
```

### Init Template

```bash
goupdate config --init
```

**Behavior:**
1. Check if `.goupdate.yml` already exists
2. If exists, return error
3. Create comprehensive template with examples

**Location:** `cmd/config.go:76-289`

**Template Contents:**
- Global settings (extends, working_dir, exclude_versions)
- Commented examples for each package manager
- Placeholder documentation
- Lock scope explanations

## Validation

```bash
goupdate config --validate
goupdate config --validate -c ./custom-config.yml
```

**Behavior:**
1. Read config file (default: `.goupdate.yml` in current directory)
2. Parse with strict YAML mode (rejects unknown fields)
3. Validate field values and constraints
4. Report errors and warnings

**Location:** `cmd/config.go` (validateConfigFile function)

**Validation checks:**
- Unknown fields (typos, unsupported options)
- YAML syntax errors
- Empty required values
- Invalid patterns

**Output Examples:**

```
✅ Configuration valid: .goupdate.yml
```

```
❌ Configuration validation failed for: .goupdate.yml

  ERROR: unknown field in config: manager_typo
  WARNING: rules.npm.outdated.commands: missing {{package}} placeholder

💡 See docs/configuration.md for valid configuration options
```

## Preflight Config Validation

All commands (`scan`, `list`, `outdated`, `update`) automatically validate configuration before running.

**Location:** `cmd/config.go` (loadAndValidateConfig function)

```go
func loadAndValidateConfig(configPath, workDir string) (*config.Config, error) {
    // Validate config file if it exists
    if data, err := readFileFunc(configPath); err == nil {
        result := config.ValidateConfigFile(data)
        if result.HasErrors() {
            return nil, NewExitError(ExitConfigError, ...)
        }
    }
    // Load config normally after validation passes
    return loadConfigFunc(configPath, workDir)
}
```

**Benefits:**
- Catches config errors early before any operations
- Provides helpful error messages with hints
- Returns exit code 3 (ExitConfigError) for scripting
- Works for both explicit `-c` configs and local `.goupdate.yml`

## Template Structure

```yaml
# Extends - inherit from other configs
extends:
  - default
  - ./base-config.yml

# Global version exclusion patterns
exclude_versions:
  - "(?i)alpha|beta|rc"

# Global incremental packages
incremental:
  - react
  - typescript

# Global package groups
groups:
  react-ecosystem:
    packages: [react, react-dom]

# Package manager rules
rules:
  npm:
    manager: js
    include: ["**/package.json"]
    # ... full configuration
```

## Working Directory Resolution

**Location:** `cmd/config.go:292-302`

```go
func resolveWorkingDir(flagValue string, cfg *config.Config) string {
    if flagValue != "" && flagValue != "." {
        return flagValue  // CLI flag takes priority
    }
    if cfg != nil && cfg.WorkingDir != "" {
        return cfg.WorkingDir  // Config value next
    }
    return "."  // Default to current directory
}
```

**Priority:**
1. `--directory` flag value
2. Config `working_dir` field
3. Current directory (`.`)

## Testing

**Mocking:**

```go
var (
    loadConfigFunc = config.LoadConfig
    writeFileFunc  = os.WriteFile
)

// In tests:
loadConfigFunc = func(path, dir string) (*config.Config, error) {
    return &config.Config{...}, nil
}
```

## Related Documentation

- [configuration.md](./configuration.md) - Config loading and merging
- [README.md](./README.md) - Architecture overview
