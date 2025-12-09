# List Command Architecture

> The `list` command shows declared packages with their installed versions resolved from lock files.

## Table of Contents

- [Command Overview](#command-overview)
- [Key Files](#key-files)
- [Data Flow](#data-flow)
- [Core Functions](#core-functions)
- [Output Format](#output-format)
- [Install Status Details](#install-status-details)
- [Special Handling](#special-handling)
- [Testing](#testing)
- [Error Handling](#error-handling)
- [Related Documentation](#related-documentation)

---

## Command Overview

```bash
goupdate list [file...] [flags]

Aliases: ls

Flags:
  -t, --type string              Filter by type (comma-separated): all,prod,dev (default "all")
  -p, --package-manager string   Filter by package manager (comma-separated, default "all")
  -r, --rule string              Filter by rule (comma-separated, default "all")
  -n, --name string              Filter by package name (comma-separated)
  -g, --group string             Filter by group (comma-separated)
  -c, --config string            Config file path
  -d, --directory string         Directory to scan (default ".")
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/list.go` | Command definition, filtering, display |
| `pkg/packages/detect.go` | File detection |
| `pkg/packages/parser.go` | Package extraction from manifests |
| `pkg/lock/resolve.go` | Lock file version resolution |
| `pkg/lock/status.go` | Install status constants |

## Data Flow

```
Config Load ──► Detect/Parse ──► Filter Packages ──► Resolve Installed ──► Display Results
```

### Step-by-Step Flow

1. **Load Configuration**
   ```go
   cfg, err := loadConfigFunc(listConfigFlag, workDir)
   ```

2. **Get Packages** (`getPackagesFunc`)
   ```go
   pkgs, err := getPackagesFunc(cfg, args, workDir)
   ```
   - If files provided as args: parse those specific files
   - Otherwise: detect all matching files and parse them

3. **Filter Packages** (`filterPackagesWithFilters`)
   ```go
   pkgs = filterPackagesWithFilters(pkgs, listTypeFlag, listPMFlag, listRuleFlag, listNameFlag, "")
   ```
   - Applies type filter (prod/dev/all)
   - Applies package manager filter
   - Applies rule filter
   - Applies name filter (comma-separated, case-insensitive)

4. **Apply Installed Versions** (`applyInstalledVersionsFunc`)
   ```go
   pkgs, err = applyInstalledVersionsFunc(pkgs, cfg, workDir)
   ```
   - Resolves each package against lock files
   - Sets `InstalledVersion` and `InstallStatus`

5. **Apply Groups** (`applyPackageGroups`)
   ```go
   pkgs = applyPackageGroups(pkgs, cfg)
   ```
   - Assigns packages to configured groups

6. **Filter by Group** (`filterByGroup`)
   ```go
   pkgs = filterByGroup(pkgs, listGroupFlag)
   ```
   - Filters packages by group (comma-separated, case-insensitive)
   - Applied after groups are assigned

7. **Track Unsupported** (`unsupportedTracker`)
   ```go
   for _, p := range pkgs {
       if shouldTrackUnsupported(p.InstallStatus) {
           unsupported.Add(p, deriveUnsupportedReason(p, cfg, nil, false))
       }
   }
   ```

8. **Display Results** (`printPackages`)

## Core Functions

### `getPackages`

**Location:** `cmd/list.go:193`

```go
func getPackages(cfg *config.Config, args []string, workDir string) ([]formats.Package, error)
```

**Two modes:**

1. **Specific files** (args provided):
   ```go
   parseSpecificFiles(args, cfg, parser)
   ```
   - Finds matching rule for each file
   - Parses file with that rule's config

2. **Auto-detect** (no args):
   ```go
   detectAndParseAll(cfg, parser, workDir)
   ```
   - Runs detection across all rules
   - Parses each detected file

### `lock.ApplyInstalledVersions`

**Location:** `pkg/lock/resolve.go:21`

```go
func ApplyInstalledVersions(packages []formats.Package, cfg *config.Config, baseDir string) ([]formats.Package, error)
```

**Behavior:**

1. Groups packages by rule and source directory (scope)
2. For each scope:
   - Finds lock files matching rule's `lock_files` patterns
   - Extracts version map from lock files
   - Matches package names to versions

**Status Assignment:**

| Condition | Status | Icon |
|-----------|--------|------|
| Version found in lock | `LockFound` | 🟢 |
| Self-pinning rule | `SelfPinned` | 📌 |
| Not in lock file | `NotInLock` | 🔵 |
| Lock file missing | `LockMissing` | 🟠 |
| Wildcard with no version | `VersionMissing` | ⛔ |
| No lock config | `NotConfigured` | ⚪ |
| Floating constraint | `Floating` | ⛔ |

> **Note:** The ⛔ icon indicates the package cannot be processed for updates. The ⚪ icon indicates missing configuration.

### `deriveUnsupportedReason`

**Location:** `cmd/list.go:809`

```go
func deriveUnsupportedReason(p formats.Package, cfg *config.Config, err error, latestMissing bool) string
```

**Returns explanatory message for unsupported packages:**

| Condition | Message |
|-----------|---------|
| Latest indicator with no lock entry | "Declared as 'latest' without a lock file entry..." |
| VersionMissing status | "No concrete version found in manifest or lock file..." |
| Floating constraint | "Floating constraint '...' cannot be updated automatically..." |
| Wildcard with no version | "Version missing for wildcard constraint '*'..." |
| No lock files configured | "No lock file configuration for this rule." |
| Rule missing (wildcard only) | "No rule configuration available for this package." |

## Output Format

```
RULE       PM      TYPE  CONSTRAINT  VERSION  INSTALLED  STATUS        GROUP  NAME
----       --      ----  ----------  -------  ---------  ------        -----  ----
composer   php     prod  Compatible  ^4.0     4.2.0      🟢 LockFound         slim/slim
npm        js      prod  Patch       ~1.0.0   1.0.5      🟢 LockFound  core   lodash
npm        js      dev   Exact       =        3.0.0      🔵 NotInLock         jest

Total packages: 3

⛔ nuget (dotnet): No lock file configuration for this rule. (2 packages)
```

### Conditional GROUP Column

**Location:** `cmd/list.go:705-722`

The GROUP column is only displayed when meaningful:

```go
func shouldShowGroupColumn(rows []listDisplayRow) bool {
    groupCounts := make(map[string]int)
    for _, row := range rows {
        group := strings.TrimSpace(row.pkg.Group)
        if group != "" {
            groupCounts[group]++
        }
    }
    for _, count := range groupCounts {
        if count >= 2 {
            return true
        }
    }
    return false
}
```

**Criteria:**
- At least one group must have 2+ packages assigned
- Single-package groups or no groups → column hidden
- Reduces visual noise when groups aren't being used

## Install Status Details

### Self-Pinning Mode

**Location:** `pkg/lock/resolve.go:47-60`

For rules with `self_pinning: true` (e.g., `requirements.txt`):

```go
if ruleCfg.SelfPinning {
    if version == "" || version == "*" {
        packages[idx].InstalledVersion = "#N/A"
        packages[idx].InstallStatus = InstallStatusVersionMissing
    } else {
        packages[idx].InstalledVersion = version
        packages[idx].InstallStatus = InstallStatusSelfPinned
    }
}
```

The declared version IS the installed version (no separate lock file).

### Unsupported Tracking

**Location:** `cmd/list.go:65-122`

```go
type unsupportedTracker struct {
    rules map[string]*unsupportedRuleInfo
}
```

Groups unsupported packages by rule to reduce noise:
- Instead of 50 messages for 50 packages
- Shows 1 message with count: "⛔ npm (js): ... (50 packages)"

## Special Handling

### Constraint Display

**Location:** `cmd/list.go:723-737`

```go
func formatConstraintDisplay(p formats.Package) string
```

Maps constraint symbols to readable names:
- (empty) → "Major"
- `^` → "Compatible (^)"
- `~` → "Patch (~)"
- `>=` → "Min (>=)"
- `<=` → "Max (<=)"
- `>` → "Above (>)"
- `<` → "Below (<)"
- `=` → "Exact (=)"
- `*` → "Major (*)"
- Unknown → "#N/A" with warning

### Group Assignment

**Location:** `cmd/list.go:377-431`

Priority for group assignment:
1. Rule-level groups (`rules.npm.groups`)
2. Top-level groups (`groups`)
3. Update config group (`rules.npm.update.group`)

## Testing

**Test File:** `cmd/list_test.go`

Key test functions:
- `TestRunListEmptyResults` - No packages found
- `TestRunListWithPackages` - Normal output
- `TestRunListTracksUnsupported` - Unsupported message grouping
- `TestUnsupportedTrackerAndReasons` - Reason derivation

**Mocking:**

```go
var (
    getPackagesFunc            = getPackages
    applyInstalledVersionsFunc = lock.ApplyInstalledVersions
)
```

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| "failed to load config" | Invalid config | Check YAML syntax |
| "no rule config found for file" | File doesn't match any rule | Add rule or check patterns |
| "failed to parse" | Invalid manifest format | Check file syntax |
| "failed to resolve lock files" | Lock parsing error | Check lock file format |

## Related Documentation

- [lock-resolution.md](./lock-resolution.md) - Lock file parsing details
- [configuration.md](./configuration.md) - Config loading
- [self-pinning.md](./self-pinning.md) - Self-pinning rules
- [groups.md](./groups.md) - Package grouping
