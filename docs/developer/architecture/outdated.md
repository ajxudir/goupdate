# Outdated Command Architecture

> The `outdated` command compares installed versions against available versions from registries.

## Table of Contents

- [Command Overview](#command-overview)
- [Key Files](#key-files)
- [Data Flow](#data-flow)
- [Core Functions](#core-functions)
- [Version Exclusions](#version-exclusions)
- [Versioning Strategies](#versioning-strategies)
- [Status Handling](#status-handling)
- [Output Format](#output-format)
- [Error Handling](#error-handling)
- [Pre-flight Validation](#pre-flight-validation)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Command Overview

```bash
goupdate outdated [file...] [flags]

Flags:
  -t, --type string              Filter by type (comma-separated): all,prod,dev (default "all")
  -p, --package-manager string   Filter by package manager (comma-separated, default "all")
  -r, --rule string              Filter by rule (comma-separated, default "all")
  -n, --name string              Filter by package name (comma-separated)
  -g, --group string             Filter by group (comma-separated)
  -c, --config string            Config file path
  -d, --directory string         Directory to scan (default ".")
      --major                    Allow major, minor, and patch comparisons
      --minor                    Allow minor and patch comparisons
      --patch                    Restrict comparisons to patch scope
      --no-timeout               Disable command timeouts
      --skip-preflight           Skip pre-flight command validation
      --continue-on-fail         Continue after failures (exit code 1)
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/outdated.go` | Command definition, result handling |
| `pkg/outdated/core.go` | Version fetching and filtering |
| `pkg/outdated/versioning.go` | Version parsing and comparison |
| `pkg/outdated/exec.go` | Command execution |
| `pkg/outdated/parsers.go` | Output parsing (JSON, YAML, raw) |
| `pkg/preflight/preflight.go` | Command availability validation |

## Data Flow

```
Preflight Validate ──► Fetch Versions ──► Filter Versions ──► Summarize Results ──► Display Results
```

### Step-by-Step Flow

1. **Load & Prepare** (same as list command)
   - Load config, detect files, parse packages
   - Filter by type/pm/rule
   - Apply installed versions

2. **Pre-flight Validation** (`preflight.ValidatePackages`)
   ```go
   if !outdatedSkipPreflight {
       validation := preflight.ValidatePackages(packages, cfg)
       if validation.HasErrors() {
           return NewExitError(ExitConfigError, ...)
       }
   }
   ```
   - Validates all commands in `outdated.commands` exist in PATH
   - Provides installation hints for missing commands

3. **Fetch Versions** (per package)
   ```go
   versions, err := listNewerVersionsFunc(context.Background(), p, cfg, workDir)
   ```

4. **Filter by Constraint** (`outdated.FilterVersionsByConstraint`)
   ```go
   filtered := outdated.FilterVersionsByConstraint(p, versions, selection)
   ```

5. **Summarize Available** (`outdated.SummarizeAvailableVersions`)
   ```go
   major, minor, patch, err := outdated.SummarizeAvailableVersions(...)
   ```

6. **Select Target** (`outdated.SelectTargetVersion`)
   ```go
   target, _ := outdated.SelectTargetVersion(major, minor, patch, selection, constraint, incremental)
   ```

7. **Display Results** (live output as each package completes)

## Core Functions

### `outdated.ListNewerVersions`

**Location:** `pkg/outdated/core.go:35`

```go
func ListNewerVersions(p formats.Package, cfg *config.Config, baseDir string) ([]string, error)
```

**Steps:**

1. Resolve effective outdated config (with overrides)
2. Create versioning strategy
3. Run outdated command
4. Parse output based on format
5. Apply version exclusions
6. Filter to newer versions only

### `outdated.FilterVersionsByConstraint`

**Location:** `pkg/outdated/core.go:506`

```go
func FilterVersionsByConstraint(p formats.Package, versions []string, flags UpdateSelectionFlags) []string
```

**Flag Override Logic:**

| Flag | Effective Constraint | Meaning |
|------|---------------------|---------|
| `--major` | `""` (none) | Allow any version |
| `--minor` | `"^"` | Same major only |
| `--patch` | `"~"` | Same major.minor only |
| (none) | Use package's constraint | Respect declared constraint |

**Constraint Filtering:**

| Constraint | Filter Rule |
|------------|-------------|
| `^` (Compatible) | Same major version |
| `~` (Patch) | Same major.minor |
| `>=` (Minimum) | Greater or equal |
| `>` (Greater) | Strictly greater |
| `<=` (Maximum) | Less or equal |
| `<` (Less) | Strictly less |
| `=` (Exact) | Matches segment count |
| `*` (Any) | No filter |

### `outdated.SummarizeAvailableVersions`

**Location:** `pkg/outdated/core.go:369`

```go
func SummarizeAvailableVersions(current string, versions []string, cfg *config.VersioningCfg, incremental bool) (string, string, string, error)
```

**Returns:**
- `major` - Best available in newer major
- `minor` - Best available in same major, newer minor
- `patch` - Best available in same major.minor, newer patch

**Incremental Mode:**

When `incremental: true` for a package:
- Returns LOWEST matching version (one step at a time)
- Instead of latest 3.0.0, returns first available after current

### `outdated.SelectTargetVersion`

**Location:** `pkg/outdated/core.go:654`

```go
func SelectTargetVersion(major, minor, patch string, flags UpdateSelectionFlags, constraint string, incremental bool) (string, error)
```

**Selection Priority (non-incremental mode):**

| Condition | Priority |
|-----------|----------|
| `--major` flag | major → minor → patch |
| `--minor` flag | minor → patch |
| `--patch` flag | patch only |
| `*` constraint | major → minor → patch |
| `^` constraint | minor → patch |
| `~` constraint | patch only |

**Selection Priority (incremental mode):**

When `incremental=true`, the priority is reversed to favor smallest updates:

| Condition | Priority |
|-----------|----------|
| `--major` flag | patch → minor → major |
| `--minor` flag | patch → minor |
| `--patch` flag | patch only |
| `*` constraint | patch → minor → major |
| `^` constraint | patch → minor |
| `~` constraint | patch only |

## Version Exclusions

### Pattern-Based Exclusion

**Location:** `pkg/outdated/core.go:305-356`

```go
func applyVersionExclusions(versions []string, cfg *config.OutdatedCfg) ([]string, error)
```

**Exclusion Sources (in order):**
1. Rule-level `exclude_versions` (accepts regex patterns)
2. `outdated.exclude_version_patterns` (if configured)
3. Default pattern (pre-release filter)

**Default Pre-release Filter:**
```regex
(?i)(?:^|[._\-/])((?:alpha|beta|rc|canary|dev|snapshot|nightly|preview)(?:[._\-/]?[0-9A-Za-z]+)*)(?:\+[^\s]*)?$
```

Excludes: `1.0.0-alpha`, `2.0.0-beta.1`, `3.0.0-rc1`, etc.

## Versioning Strategies

**Location:** `pkg/outdated/versioning.go`

| Format | Description | Use Case |
|--------|-------------|----------|
| `semver` | Standard semantic versioning | npm, composer |
| `numeric` | Major version only | Simple version schemes |
| `regex` | Custom regex with groups | Non-standard formats |
| `ordered` | Position in list = order | No version parsing needed |

**Custom Regex Example:**
```yaml
outdated:
  versioning:
    format: regex
    regex: '(?P<major>\d+)_(?P<minor>\d+)_(?P<patch>\d+)'
```

## Status Handling

### `deriveOutdatedStatus`

**Location:** `cmd/outdated.go:238`

```go
func deriveOutdatedStatus(res outdatedResult) string {
    // Preserve Floating status - these packages cannot be processed automatically
    if res.pkg.InstallStatus == lock.InstallStatusFloating {
        return lock.InstallStatusFloating
    }

    if res.err != nil {
        if code := outdated.ExtractExitCode(res.err); code != "" {
            return fmt.Sprintf("Failed(%s)", code)
        }
        return "Failed"
    }

    if res.major != "#N/A" || res.minor != "#N/A" || res.patch != "#N/A" {
        return "Outdated"
    }

    return "UpToDate"
}
```

**Status Values:**

| Status | Icon | Meaning |
|--------|------|---------|
| `Floating` | ⛔ | Floating constraint - cannot be processed |
| `Outdated` | 🟠 | Newer versions available |
| `UpToDate` | 🟢 | No newer versions found |
| `NotConfigured` | ⚪ | No outdated config for rule |
| `Failed` | 🔴 | Command execution failed |
| `Failed(N)` | 🔴 | Command failed with exit code N |

**Important:** Packages with `InstallStatusFloating` from the list phase preserve their status. This ensures consistency across commands - a package showing `⛔ Floating` in `list` will also show `⛔ Floating` in `outdated`, not `🟢 UpToDate`.

## Output Format

```
RULE       PM      TYPE  CONSTRAINT  VERSION  INSTALLED  MAJOR   MINOR   PATCH   STATUS        GROUP  NAME
----       --      ----  ----------  -------  ---------  -----   -----   -----   ------        -----  ----
npm        js      prod  Compatible  ^4.0.0   4.17.20    5.0.0   4.18.0  4.17.21 🟠 Outdated         lodash
npm        js      prod  Major (*)   *        4.4.3      #N/A    #N/A    #N/A    ⛔ Floating         debug
composer   php     prod  Patch       ~2.0.0   2.0.5      3.0.0   2.1.0   2.0.6   🟠 Outdated         monolog

Total packages: 3

⛔ npm (js): Floating constraint '*' - update manually or remove constraint. (1 package)
⛔ requirements (python): No rule configuration available for this package. (3 packages)

❌ express (js/npm): command timed out
  💡 Command took too long: Use --no-timeout flag or increase timeout in config (timeout_seconds)
```

## Error Handling

### Command Execution Errors

**Location:** `pkg/outdated/core.go:237-259`

```go
if normalized := normalizeOutdatedError(err, commandName); normalized != err {
    return nil, normalized
}
```

**Normalized Errors:**

| Pattern | Mapped To |
|---------|-----------|
| "No assets file was found" (dotnet) | UnsupportedError |
| "Found more than one project" (dotnet) | UnsupportedError |

### Error Hints

**Location:** `cmd/exitcodes.go:80-157`

Common error patterns with resolution hints:
- `"command timed out"` → "Use --no-timeout flag..."
- `"ENOTFOUND"` → "Check network connectivity..."
- `"401"` → "Configure authentication..."

## Pre-flight Validation

**Location:** `pkg/preflight/preflight.go`

```go
func ValidatePackages(packages []formats.Package, cfg *config.Config) *ValidateResult
```

**Checks:**
1. Extracts command names from `outdated.commands`
2. Checks each command exists via `exec.LookPath`
3. Falls back to shell check for aliases
4. Returns errors with installation hints

**Command Resolution Hints:**
```go
var CommandResolutionHints = map[string]string{
    "npm":    "Install Node.js: https://nodejs.org/",
    "go":     "Install Go: https://go.dev/dl/",
    "dotnet": "Install .NET SDK: https://dotnet.microsoft.com/download",
    // ... more
}
```

## Testing

**Test File:** `cmd/outdated_test.go`

Key test functions:
- `TestRunOutdatedEmpty` - No packages
- `TestRunOutdatedWithPackages` - Normal flow
- `TestRunOutdatedVersionFiltering` - Constraint filtering
- `TestOutdatedPreflightValidation` - Command validation

**Mocking:**

```go
var listNewerVersionsFunc = outdated.ListNewerVersions

// In tests:
listNewerVersionsFunc = func(ctx context.Context, p formats.Package, cfg *config.Config, baseDir string) ([]string, error) {
    return []string{"1.1.0", "2.0.0"}, nil
}
```

## Related Documentation

- [floating-constraints.md](./floating-constraints.md) - Floating constraint handling
- [version-comparison.md](./version-comparison.md) - Version parsing details
- [command-execution.md](./command-execution.md) - Command execution
- [incremental-updates.md](./incremental-updates.md) - Incremental mode
- [configuration.md](./configuration.md) - Config structure
