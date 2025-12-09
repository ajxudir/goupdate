# Utility Commands Architecture

> Documentation for the version, help, and config commands that support the main CLI functionality.

## Table of Contents

- [Overview](#overview)
- [version Command](#version-command)
- [help Command](#help-command)
- [config Command](#config-command)
- [Config Validation](#config-validation)
- [Key Files](#key-files)
- [Testing](#testing)

## Overview

The utility commands provide supporting functionality for the main dependency management commands:

```
goupdate CLI
═══════════════════════════════════════════════════════════════════

Main Commands              Utility Commands
├── scan                   ├── version
├── list                   ├── help
├── outdated               └── config
└── update                     ├── --show-defaults
                               ├── --show-effective
                               ├── --init
                               └── --validate
```

## version Command

**Location:** `cmd/version.go`

Prints version and build information for goupdate.

### Build-Time Variables

```go
var (
    Version   = "dev"     // Set via -ldflags
    BuildTime = ""        // Set via -ldflags
    GitCommit = ""        // Set via -ldflags
)
```

### Output Format

```
goupdate version 1.0.0
  Build time: 2024-01-15T10:30:00Z
  Git commit: abc1234
  Go version: go1.21.0
  OS/Arch:    linux/amd64
```

### Makefile Integration

```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

LDFLAGS := -X github.com/ajxudir/goupdate/cmd.Version=$(VERSION) \
           -X github.com/ajxudir/goupdate/cmd.BuildTime=$(BUILD_TIME) \
           -X github.com/ajxudir/goupdate/cmd.GitCommit=$(GIT_COMMIT)

build:
    go build -ldflags="$(LDFLAGS)" -o goupdate .
```

## help Command

**Location:** Built into Cobra framework

The help command is automatically provided by the Cobra CLI framework.

### Usage Patterns

```bash
goupdate help              # Show all commands
goupdate help <command>    # Show command-specific help
goupdate <command> --help  # Alternative syntax
goupdate <command> -h      # Short form
```

### Customization

Help text is defined in each command's Cobra definition:

```go
var updateCmd = &cobra.Command{
    Use:   "update [file...]",
    Short: "Apply dependency updates",
    Long:  `Plan and apply dependency updates using rule-level configuration.`,
    // ...
}
```

## config Command

**Location:** `cmd/config.go`

Manages configuration display, creation, and validation.

### Subcommands (via flags)

| Flag | Description | Implementation |
|------|-------------|----------------|
| `--show-defaults` | Print embedded defaults | `config.GetDefaultConfig()` |
| `--show-effective` | Show merged config | `config.LoadConfig()` |
| `--init` | Create template | `createConfigTemplate()` |
| `--validate` | Validate config | `config.ValidateConfigFile()` |

### Data Flow

```
CLI --validate ───▶ cmd/config.go validateFile() ───▶ pkg/config/validate.go
                                                              │
                                                              ▼
                                                      ValidationResult
                                                        - Errors
                                                        - Warnings
```

## Config Validation

**Location:** `pkg/config/validate.go`

### Validation Features

1. **Unknown Field Detection**
   - Uses `yaml.NewDecoder().KnownFields(true)` for strict parsing
   - Extracts field name, line number, and type context from errors

2. **Schema Hints**
   - Maps unknown fields to valid alternatives
   - Suggests corrections for common typos
   - Provides documentation references

3. **Verbose Mode**
   - Enabled via `--verbose` flag
   - Shows valid keys, expected types, and doc sections

### Schema Information

```go
var configSchema = map[string]schemaInfo{
    "Config": {
        fields: "extends, working_dir, rules, exclude_versions, groups, incremental",
        doc:    "configuration",
    },
    "PackageManagerCfg": {
        fields: "enabled, manager, include, exclude, ...",
        doc:    "rules",
    },
    // ... more types
}
```

### Common Typo Detection

```go
var commonTypos = map[string]map[string]string{
    "Config": {
        "rule":        "rules",
        "extend":      "extends",
        "working-dir": "working_dir",
    },
    // ... more typos
}
```

### ValidationError Structure

```go
type ValidationError struct {
    Field      string // Field path (e.g., "rules.npm.outdated")
    Message    string // Error description
    Expected   string // Expected type or schema hint
    ValidKeys  string // Valid keys for this context
    DocSection string // Documentation section reference
}
```

### Output Examples

**Standard output:**
```
❌ Configuration validation failed for: .goupdate.yml

  ERROR: unknown field 'command'

💡 Run with --verbose for detailed schema information
💡 See docs/configuration.md for valid configuration options
```

**Verbose output:**
```
❌ Configuration validation failed for: .goupdate.yml

  ERROR: unknown field 'command' (line 15) (did you mean 'commands'?)
    Valid keys: commands, env, format, extraction, versioning, ...
    📖 See: docs/configuration.md#outdated

💡 See docs/configuration.md for valid configuration options
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/version.go` | Version command implementation |
| `cmd/config.go` | Config command implementation |
| `pkg/config/validate.go` | Config validation logic |
| `pkg/config/model.go` | Config struct definitions (for schema reference) |
| `pkg/verbose/verbose.go` | Verbose output utilities |

## Testing

### Version Tests

```bash
go test -v ./cmd/... -run TestVersion
```

### Config Validation Tests

```bash
go test -v ./pkg/config/... -run TestValidate
```

### Test Files

| Test File | Coverage |
|-----------|----------|
| `cmd/version_test.go` | Version output format |
| `cmd/config_test.go` | Config command flags, validation |
| `pkg/config/validate_test.go` | Validation logic, schema hints |

### Test Scenarios

1. **Valid configuration** - Should pass without errors
2. **Unknown fields** - Should detect and suggest corrections
3. **Type mismatches** - Should report expected types
4. **YAML syntax errors** - Should provide line numbers
5. **Missing required fields** - Should list what's missing
