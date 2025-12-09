# Configuration System Architecture

> The configuration system handles loading, merging, validation, and resolution of goupdate settings.

## Table of Contents

- [Key Files](#key-files)
- [Configuration Structure](#configuration-structure)
- [Config Loading Flow](#config-loading-flow)
- [Extends System](#extends-system)
- [Config Merging](#config-merging)
- [Default Configuration](#default-configuration)
- [Package Overrides](#package-overrides)
- [Group Configuration](#group-configuration)
- [Incremental Updates](#incremental-updates)
- [Latest Mapping](#latest-mapping)
- [Lock File Configuration](#lock-file-configuration)
- [Outdated Configuration](#outdated-configuration)
- [Update Configuration](#update-configuration)
- [Config Resolution](#config-resolution)
- [Validation](#validation)
- [Disabling Rules](#disabling-rules)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/config/model.go` | Configuration structure definitions |
| `pkg/config/load.go` | Config loading and extends processing |
| `pkg/config/merge.go` | Config merging logic |
| `pkg/config/defaults.go` | Embedded default configuration |
| `pkg/config/default.yml` | Default rule definitions |
| `pkg/config/groups.go` | Group parsing and validation |
| `pkg/config/incremental.go` | Incremental update pattern matching |
| `pkg/config/latest_mapping.go` | Version token normalization |

## Configuration Structure

### Root Config

```go
type Config struct {
    Extends         []string                     // Configs to inherit from
    WorkingDir      string                       // Base directory for operations
    Rules           map[string]PackageManagerCfg // Rule definitions
    ExcludeVersions []string                     // Global version exclusion patterns
    Groups          map[string]GroupCfg          // Global package groups
    Incremental     []string                     // Packages to update incrementally
    NoTimeout       bool                         // Runtime flag (from --no-timeout)
}
```

### Rule Configuration

```go
type PackageManagerCfg struct {
    Enabled           *bool                          // Enable/disable rule (default: true)
    Manager           string                         // Package manager type (js, php, python, etc.)
    Include           []string                       // Glob patterns to include
    Exclude           []string                       // Glob patterns to exclude
    Groups            map[string]GroupCfg            // Rule-level package groups
    Format            string                         // File format: json, yaml, xml, raw
    Fields            map[string]string              // Field → type mapping
    Ignore            []string                       // Packages to ignore
    ExcludeVersions   []string                       // Version exclusion patterns
    ConstraintMapping map[string]string              // Symbol → constraint type mapping
    LatestMapping     *LatestMappingCfg              // Version token normalization
    PackageOverrides  map[string]PackageOverrideCfg  // Per-package overrides
    Extraction        *ExtractionCfg                 // Version extraction config
    Outdated          *OutdatedCfg                   // Outdated checking config
    Update            *UpdateCfg                     // Update execution config
    LockFiles         []LockFileCfg                  // Lock file definitions
    SelfPinning       bool                           // Manifest is its own lock
    Incremental       []string                       // Rule-level incremental packages
}
```

## Config Loading Flow

```
CLI Flag --config ──► Load Config File ──► Process Extends ──► Validate Groups
                                                  │
                                                  ▼
                                          Merge with Inherited

No --config specified ──► Try .goupdate.yml ──► Fall back to Defaults
```

### Loading Logic

**Location:** `pkg/config/load.go:12-60`

```go
func LoadConfig(configPath, workDir string) (*Config, error) {
    if configPath != "" {
        // 1. Load specified config
        cfg = loadConfigFile(configPath)
        // 2. Process extends
        cfg = processExtends(cfg, filepath.Dir(configPath))
    } else {
        // 1. Try .goupdate.yml in working directory
        localConfig := filepath.Join(workDir, ".goupdate.yml")
        if exists(localConfig) {
            cfg = loadConfigFile(localConfig)
            cfg = processExtends(cfg, workDir)
        }
        // 2. Fall back to defaults
        if cfg == nil {
            cfg = loadDefaultConfig()
        }
    }

    // 3. Validate group membership
    validateGroupMembership(cfg)

    return cfg, nil
}
```

## Extends System

### Supported Values

| Value | Behavior |
|-------|----------|
| `default` | Inherit embedded default configuration |
| `path/to/file.yml` | Inherit from specified file (relative or absolute) |

### Extends Processing

**Location:** `pkg/config/load.go:80-157`

```go
func processExtendsWithStack(cfg *Config, baseDir string, stack map[string]bool) (*Config, error)
```

**Features:**
1. **Cyclic detection**: Tracks visited files to prevent infinite loops
2. **Recursive processing**: Extended configs can also extend other configs
3. **Order matters**: Earlier extends are merged first, later ones override

**Example:**

```yaml
# .goupdate.yml
extends:
  - default           # Start with defaults
  - ./company.yml     # Override with company settings
  - ./project.yml     # Project-specific overrides
```

## Config Merging

### Merge Strategy

**Location:** `pkg/config/merge.go`

| Field Type | Merge Behavior |
|------------|----------------|
| Scalar (string, bool, int) | Override replaces base |
| Map (Rules, Groups) | Deep merge, custom overrides base |
| Slice (ExcludeVersions, Incremental) | Append with deduplication |
| Pointer (Outdated, Update) | Override replaces base |

### Rule Merging

```go
func mergeRules(base, custom PackageManagerCfg) PackageManagerCfg {
    merged := base

    if custom.Manager != "" {
        merged.Manager = custom.Manager
    }
    if len(custom.Include) > 0 {
        merged.Include = custom.Include  // Full replacement
    }
    if custom.Outdated != nil {
        merged.Outdated = custom.Outdated  // Full replacement
    }
    // ... etc
}
```

### Version Pattern Merging

```go
func mergeVersionPatterns(base, override []string) []string {
    if override == nil {
        return base  // Keep base if no override
    }
    if len(override) == 0 {
        return []string{}  // Explicit empty clears all
    }
    // Append with deduplication
    combined := append([]string{}, base...)
    for _, pattern := range override {
        if !seen[pattern] {
            combined = append(combined, pattern)
        }
    }
    return combined
}
```

## Default Configuration

### Embedded Defaults

**Location:** `pkg/config/defaults.go`

```go
//go:embed default.yml
var defaultConfigYAML string

func loadDefaultConfig() *Config {
    yaml.Unmarshal([]byte(defaultConfigYAML), &cfg)
    return &cfg
}
```

### Default Rules

The embedded `default.yml` includes:

| Rule | Manager | Format | Description |
|------|---------|--------|-------------|
| `npm` | js | json | Node.js package.json |
| `pnpm` | js | json | pnpm (extends npm) |
| `yarn` | js | json | Yarn (extends npm) |
| `composer` | php | json | PHP Composer |
| `requirements` | python | raw | Python requirements.txt |
| `pipfile` | python | raw | Python Pipfile |
| `mod` | golang | raw | Go modules |
| `msbuild` | dotnet | xml | .NET csproj/vbproj |
| `nuget` | dotnet | xml | NuGet packages.config |

## Package Overrides

### Override Structure

```go
type PackageOverrideCfg struct {
    Ignore     bool                  // Skip this package entirely
    Constraint *string               // Override constraint type
    Version    string                // Override declared version
    Outdated   *OutdatedOverrideCfg  // Override outdated config
    Update     *UpdateOverrideCfg    // Override update config
}
```

### Example Usage

```yaml
rules:
  npm:
    package_overrides:
      lodash:
        constraint: "~"  # Use patch-only updates
      internal-lib:
        ignore: true     # Skip entirely
      legacy-pkg:
        outdated:
          timeout_seconds: 60  # Slow registry
```

## Group Configuration

### Group Syntax Options

**Sequence format:**
```yaml
groups:
  core:
    - react
    - react-dom
```

**Map format with packages:**
```yaml
groups:
  core:
    packages:
      - react
      - react-dom
```

**Object format with name:**
```yaml
groups:
  core:
    - name: react
    - name: react-dom
```

### Group Validation

**Location:** `pkg/config/groups.go:87-131`

```go
func validateGroupMembership(cfg *Config) error
```

**Checks:**
- No package assigned to multiple groups within a rule
- Reports conflicts with group names

## Incremental Updates

### Pattern Matching

**Location:** `pkg/config/incremental.go`

```go
func ShouldUpdateIncrementally(p PackageRef, cfg *Config) (bool, error)
```

**Pattern sources (in order):**
1. Rule-level `incremental`
2. Rule-level `incremental_packages` (legacy)
3. Global `incremental`
4. Global `incremental_packages` (legacy)

**Pattern types:**
- Literal: `lodash` matches exactly "lodash"
- Regex: `@company/.*` matches all @company packages

### Example Configuration

```yaml
# Global incremental packages
incremental:
  - "@company/.*"    # Regex: all company packages
  - legacy-database  # Literal: specific package

rules:
  npm:
    incremental:
      - react   # Rule-specific
```

## Latest Mapping

### Purpose

Normalizes version tokens to consistent values:
- Empty string → `*`
- `latest` → `*`
- Custom mappings per package manager

### Configuration

```yaml
latest_mapping:
  default:
    latest: "*"
    stable: "*"
  packages:
    special-pkg:
      edge: "*"
```

## Lock File Configuration

### Structure

```go
type LockFileCfg struct {
    Files      []string       // Glob patterns
    Format     string         // Format: json, yaml, pnpm-lock, yarn-lock, raw
    Extraction *ExtractionCfg // How to extract versions
}
```

### Special Formats

| Format | Description |
|--------|-------------|
| `pnpm-lock` | Native pnpm-lock.yaml parser |
| `yarn-lock` | Native yarn.lock parser |
| `json` | Generic JSON with regex extraction |
| `raw` | Generic regex-based extraction |

## Outdated Configuration

```go
type OutdatedCfg struct {
    Commands               string                   // Shell commands to run
    Env                    map[string]string        // Environment variables
    Format                 string                   // Output format: json, yaml, raw
    Extraction             *OutdatedExtractionCfg   // How to extract versions
    Versioning             *VersioningCfg           // Version parsing config
    ExcludeVersions        []string                 // Specific versions to exclude
    ExcludeVersionPatterns []string                 // Version exclusion patterns
    TimeoutSeconds         int                      // Command timeout
}
```

## Update Configuration

```go
type UpdateCfg struct {
    Commands   string            // Lock/install commands (run after manifest version is updated)
    Env            map[string]string // Environment variables
    LockScope      string            // "group" or "package"
    Group          string            // Group identifier template
    TimeoutSeconds int               // Command timeout
}
```

## Config Resolution

### For Outdated Commands

```go
func ResolveOutdatedCfg(p formats.Package, cfg *config.Config) (*config.OutdatedCfg, error)
```

**Resolution order:**
1. Start with rule-level outdated config
2. Apply package override if exists

### For Update Commands

```go
func ResolveUpdateCfg(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error)
```

**Resolution order:**
1. Start with rule-level update config
2. Apply package override if exists

## Validation

### Group Membership Validation

Prevents packages from being in multiple groups:

```go
if len(groups) > 1 {
    return fmt.Errorf("rule %s has packages assigned to multiple groups: %s",
        ruleName, strings.Join(conflicts, "; "))
}
```

### Rule Enabled Check

```go
func (p *PackageManagerCfg) IsEnabled() bool {
    if p.Enabled == nil {
        return true  // Default enabled
    }
    return *p.Enabled
}
```

## Disabling Rules

When extending a configuration (e.g., `extends: [default]`), you may want to disable certain inherited rules without removing them entirely. Use the `enabled` field:

```yaml
extends:
  - default

rules:
  # Disable msbuild rule from defaults (still inherits but won't run)
  msbuild:
    enabled: false

  # Disable nuget rule from defaults
  nuget:
    enabled: false

  # npm remains enabled (inherited from default)
  # composer remains enabled (inherited from default)
```

**Behavior:**

| Field Value | Effect |
|-------------|--------|
| `enabled: true` | Rule is active (same as not specifying) |
| `enabled: false` | Rule is skipped in detection, parsing, and updates |
| Not specified | Defaults to `true` (enabled) |

**Where `enabled` is checked:**

1. **File detection** (`pkg/packages/detect.go`) - Skips disabled rules when scanning
2. **Specific file parsing** (`cmd/list.go`) - Skips disabled rules when matching files
3. **Config display** (`cmd/config.go --show-effective`) - Shows "Enabled: false" for disabled rules

**Example use cases:**

1. **Focus on specific ecosystems:**
   ```yaml
   extends: [default]
   rules:
     mod: { enabled: false }      # Skip Go modules
     msbuild: { enabled: false }  # Skip .NET projects
     nuget: { enabled: false }
   ```

2. **Custom replacement:**
   ```yaml
   extends: [default]
   rules:
     # Disable default npm rule
     npm: { enabled: false }
     # Use custom npm rule with different settings
     npm-custom:
       manager: js
       include: ["**/package.json"]
       # ... custom configuration
   ```

## Related Documentation

- [lock-resolution.md](./lock-resolution.md) - Lock file parsing
- [outdated.md](./outdated.md) - Outdated configuration usage
- [update.md](./update.md) - Update configuration usage
- [floating-constraints.md](./floating-constraints.md) - Floating constraint handling
