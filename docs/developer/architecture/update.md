# Update Command Architecture

> The `update` command plans and applies version updates by combining constraint-aware selections with configured lock commands.

## Table of Contents

- [Command Overview](#command-overview)
- [Key Files](#key-files)
- [Data Flow](#data-flow)
- [Update Execution Modes](#update-execution-modes)
- [Config Resolution](#config-resolution)
- [Group Management](#group-management)
- [Manifest Update Formats](#manifest-update-formats)
- [Validation & Rollback](#validation--rollback)
- [Update Statuses](#update-statuses)
- [Error Handling](#error-handling)
- [Command Execution](#command-execution)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Command Overview

```bash
goupdate update [file...] [flags]

Flags:
  -t, --type string              Filter by type (comma-separated): all,prod,dev (default "all")
  -p, --package-manager string   Filter by package manager (comma-separated, default "all")
  -r, --rule string              Filter by rule (comma-separated, default "all")
  -n, --name string              Filter by package name (comma-separated)
  -g, --group string             Filter by group (comma-separated)
  -c, --config string            Config file path
  -d, --directory string         Directory to scan (default ".")
      --major                    Force major upgrades (cascade to minor/patch)
      --minor                    Force minor upgrades (cascade to patch)
      --patch                    Force patch upgrades only
      --dry-run                  Plan updates without writing files
      --skip-lock                Skip running lock/install command
  -y, --yes                      Skip confirmation prompt
      --no-timeout               Disable command timeouts
      --continue-on-fail         Continue after failures (exit code 1)
      --skip-preflight           Skip pre-flight command validation
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/update.go` | Command definition, planning, group processing |
| `pkg/update/core.go` | Main update logic, file writing |
| `pkg/update/registry.go` | Format updater registry (extensibility) |
| `pkg/update/resolve.go` | Config resolution with overrides |
| `pkg/update/group.go` | Group key resolution |
| `pkg/update/exec.go` | Lock command execution |
| `pkg/update/errors.go` | Update error types |
| `pkg/update/json.go` | JSON manifest updates |
| `pkg/update/yaml.go` | YAML manifest updates |
| `pkg/update/xml.go` | XML manifest updates |
| `pkg/update/raw.go` | Raw/regex-based updates |
| `pkg/update/rollback.go` | Rollback utilities |

## Data Flow

```
Load & Prepare ──► Resolve Configs ──► Plan Updates ──► Preview & Confirm ──► Execute Updates
                                                                                      │
                                                                                      ▼
                   Rollback On Error ◄── Validate Changes ◄── Run Lock Commands ◄── Update Manifest
```

### Step-by-Step Flow

1. **Load & Prepare** (same as list/outdated commands)
   - Load config, detect files, parse packages
   - Filter by type/pm/rule
   - Apply installed versions
   - Apply package groups

2. **Resolve Update Configs** (`update.ResolveUpdateCfg`)
   ```go
   for _, p := range packages {
       cfgForPkg, cfgErr := resolveUpdateCfgFunc(p, cfg)
       resolvedPkg.Group = update.NormalizeUpdateGroup(cfgForPkg, p)
   }
   ```
   - Resolve effective config per package
   - Apply package overrides
   - Normalize group identifiers

3. **Pre-flight Validation** (unless `--skip-preflight`)
   - Validates all commands in `update.commands` exist
   - Returns errors with installation hints

4. **Plan Updates** (per package)
   - Skip floating constraints (marked as unsupported)
   - Check for exact constraints (skip versioning)
   - Fetch available versions
   - Filter by constraint
   - Select target version

5. **Preview & Confirm** (unless `--yes` or `--dry-run`)
   ```
   Update Plan
   ══════════════════════════════════════════════════════════════════════

   Will update (constraint scope):
     lodash               ^4.17.20 → 4.18.0  (major: 5.0.0 available)
     express              ^4.18.0 → 4.19.0   (fully updated to latest)

   Up to date (other updates available):
     zustand              ^4.3.0  (major: 5.0.0 available)

   Summary: 2 to update, 1 up-to-date
            (2 have major available)

   2 package(s) will be updated. Continue? [y/N]:
   ```

   **Summary sections explained:**
   - **Will update**: Packages that will be updated with their version transition
   - **Up to date (other updates available)**: Packages at latest within current scope but have updates in other scopes (actionable - use `--major` to update)
   - Packages that are "fully up to date" (no updates anywhere) are **not listed** - they don't require action

   **After execution**, a similar summary is shown:
   ```
   Update Summary
   ══════════════════════════════════════════════════════════════════════

   Successfully updated:
     lodash               ^4.17.20 → 4.18.0  (major: 5.0.0 available)
     express              ^4.18.0 → 4.19.0   (fully updated to latest)

   Up to date (other updates available):
     zustand              ^4.3.0  (major: 5.0.0 available)

   Summary: 2 updated, 1 up-to-date
            (2 have major updates still available)
   ```

   The "Up to date" section only appears when there are packages with remaining updates available. Fully up-to-date packages are counted in the summary but not listed individually.

6. **Execute Updates** (per group or package)
   - Group-level lock: update manifests first, then run lock once
   - Package-level lock: run lock per package with validation

7. **Validate Changes** (`validateUpdatedPackage`)
   - Re-parse manifest to verify version written
   - Check installed version matches target (drift detection)

8. **Rollback on Error** (unless `--continue-on-fail`)
   - Restore original manifest content
   - Re-run lock command with original version

## Update Execution Modes

### Per-Package Lock (Default)

For each package:
1. Read original manifest content (for rollback)
2. Run lock command (pre-update)
3. Update declared version in manifest
4. Run lock command (post-update)
5. Validate changes
6. Rollback if validation fails

```go
func UpdatePackage(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error
```

### Group-Level Lock

When multiple packages share the same group (via the `group` field):
1. Update all declared versions in the group
2. Run lock command once for the entire group
3. Validate all packages in the group
4. Rollback entire group on any failure

```go
if useGroupLock && !dryRun && !updateSkipLockRun {
    // Update declared versions only (skipLock=true)
    for _, plan := range plans {
        updatePackageFunc(plan.res.pkg, plan.res.target, cfg, workDir, dryRun, true)
    }

    // Run lock command once for the entire group
    update.RunGroupLockCommand(groupUpdateCfg, workDir)

    // Validate all applied updates
    for _, plan := range applied {
        validateUpdatedPackage(plan, reloadList, baseline)
    }
}
```

### Floating Constraints (Not Supported)

Floating constraints like `5.*`, `>=8.0.0`, `[8.0.0,9.0.0)` are **not supported** for automatic updates because:

1. Changing the manifest destroys user intent (e.g., changing `5.*` to `5.2.1`)
2. Most package managers don't have reliable per-package lock-only update commands
3. The only sensible action is running the lock command, but that updates all packages

**Handling:**

```go
if utils.IsFloatingConstraint(p.Version) {
    res.status = lock.InstallStatusFloating
    unsupported.Add(p, fmt.Sprintf(
        "floating constraint '%s' cannot be updated automatically; "+
        "remove the constraint or update manually", p.Version))
    continue
}
```

**User Options:**
1. Remove the floating constraint and use an exact version
2. Run the package manager's update command manually

## Config Resolution

**Location:** `pkg/update/resolve.go`

```go
func ResolveUpdateCfg(p formats.Package, cfg *config.Config) (*config.UpdateCfg, error)
```

**Resolution Order:**
1. Get rule-level `update` config
2. Apply package override from `package_overrides[name].update`

**Override Fields:**

| Field | Description |
|-------|-------------|
| `commands` | Lock/install command to run after manifest update |
| `env` | Environment variables |
| `group` | Group identifier for atomic updates |
| `timeout_seconds` | Command timeout |

## Group Management

**Location:** `pkg/update/group.go`

### Group Key Resolution

```go
func UpdateGroupKey(cfg *config.UpdateCfg, pkg formats.Package) string
```

**Priority:**
1. Package's pre-assigned group (`pkg.Group`)
2. Resolved group from config (`cfg.Group`)
3. Package name (fallback for ungrouped)

### Group Templates

The `group` field supports placeholders:
- `{{package}}` - Package name
- `{{rule}}` - Rule name
- `{{type}}` - Package type (prod/dev)

Example:
```yaml
update:
  group: "{{rule}}-deps"  # Groups all packages in same rule together
```

## Format Updater Registry

**Location:** `pkg/update/registry.go`

The format updater registry provides an extensible architecture for handling different manifest formats:

### Interface

```go
type FormatUpdater interface {
    UpdateVersion(content []byte, pkg formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error)
}
```

### Registry Operations

```go
// Register a custom format updater
RegisterFormatUpdater("custom", FormatUpdaterFunc(myCustomUpdater))

// Get updater for a format
updater := GetFormatUpdater("json")

// List all registered formats
formats := ListRegisteredFormats() // ["json", "yaml", "xml", "raw"]
```

### Built-in Updaters

The following updaters are registered at initialization:

| Format | Updater Function | Description |
|--------|------------------|-------------|
| `json` | `updateJSONVersion` | JSON manifest updates |
| `yaml` | `updateYAMLVersion` | YAML manifest updates |
| `xml` | `updateXMLVersion` | XML/MSBuild updates |
| `raw` | `updateRawVersion` | Regex-based updates |

### Adding Custom Formats

To add support for a new manifest format:

```go
func init() {
    update.RegisterFormatUpdater("myformat", update.FormatUpdaterFunc(updateMyFormat))
}

func updateMyFormat(content []byte, pkg formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error) {
    // Custom update logic
    return updatedContent, nil
}
```

## Manifest Update Formats

### JSON Updates

**Location:** `pkg/update/json.go`

```go
func updateJSONVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error)
```

- Uses `orderedmap` to preserve key order
- Finds package in configured `fields` (e.g., `dependencies`, `devDependencies`)
- Replaces version with `constraint + target`
- Preserves formatting (indentation, no HTML escaping)

### YAML Updates

**Location:** `pkg/update/yaml.go`

- Uses `gopkg.in/yaml.v3` to preserve comments and formatting
- Navigates to version field via configured path
- Updates scalar node value

### XML Updates

**Location:** `pkg/update/xml.go`

- Uses Go's `encoding/xml` parser
- Navigates using configured XPath-like extraction
- Updates element or attribute value

### Raw/Regex Updates

**Location:** `pkg/update/raw.go`

```go
func updateRawVersion(content []byte, p formats.Package, ruleCfg config.PackageManagerCfg, target string) ([]byte, error)
```

- Uses regex with named groups from `extraction.pattern`
- Finds match for package name
- Replaces version at exact captured position
- Handles constraint prefix if captured separately

## Validation & Rollback

### Version Validation

**Location:** `cmd/update.go:629-666`

```go
func validateUpdatedPackage(plan *plannedUpdate, reloadList func() ([]formats.Package, error), baseline map[string]versionSnapshot) error
```

**Checks:**
1. Package still exists after update
2. Declared version matches target
3. Installed version matches target (drift detection)

### Baseline Snapshots

Before updates, capture current state:
```go
type versionSnapshot struct {
    version   string
    installed string
}

baseline := snapshotVersions(packages)
```

### Rollback Strategy

**Location:** `cmd/update.go:565-578`

```go
func rollbackPlans(plans []*plannedUpdate, cfg *config.Config, workDir string, failures *[]error, groupErr error)
```

**When triggered:**
- Validation failure
- Lock command failure
- Group-level failure (unless `--continue-on-fail`)

**Actions:**
1. Restore original manifest version
2. Re-run lock command
3. Mark all group packages as failed

## Update Statuses

| Status | Emoji | Description |
|--------|-------|-------------|
| `Updated` | 🟢 | Successfully updated |
| `Planned` | 🟡 | Will update (dry-run mode) |
| `UpToDate` | 🟢 | Already at target version |
| `Floating` | ⛔ | Floating constraint (cannot auto-update) |
| `Failed` | ❌ | Update failed |
| `ConfigError` | ❌ | Configuration problem |
| `SummarizeError` | ❌ | Version summarization failed |
| `NotConfigured` | ⚪ | Rule doesn't support updates |

## Error Handling

### UpdateUnsupportedError

**Location:** `pkg/update/errors.go`

```go
type UpdateUnsupportedError struct {
    Reason string
}

func IsUpdateUnsupported(err error) bool {
    var target *UpdateUnsupportedError
    return errors.As(err, &target)
}
```

**Common Reasons:**
- "update configuration missing for {rule}"
- "lock update missing for {rule}"
- "no commands configured"
- "no package_update_commands configured for floating constraints"
- "updates not supported for format {format}"

### Error Hints

Errors are enhanced with actionable hints:
```go
fmt.Printf("❌ %s\n", EnhanceErrorWithHint(err))
```

Common hints include:
- Timeout errors → "Use --no-timeout flag..."
- Network errors → "Check network connectivity..."
- Auth errors → "Configure authentication..."

## Command Execution

### Lock Commands

**Location:** `pkg/update/exec.go`

```go
func executeUpdateCommand(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error)
```

**Placeholders:**
- `{{package}}` - Package name
- `{{version}}` - Target version
- `{{constraint}}` - Package constraint

### Package Update Commands

For floating constraints:
```go
func executePackageUpdateCommand(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error)
```

Uses `package_update_commands` instead of `commands`.

## Testing

**Test File:** `cmd/update_test.go`

Key test functions:
- `TestRunUpdateWithPackages` - Normal update flow
- `TestUpdateValidation` - Validation checks
- `TestUpdateRollback` - Rollback on failure
- `TestUpdateGroupLevelLock` - Group-level locking
- `TestFloatingConstraintInGroupShowsError` - Floating + group rejection
- `TestFloatingConstraintWithPackageUpdateCommandsShowsFloating` - Floating support

**Mocking:**

```go
var updatePackageFunc = update.UpdatePackage
var resolveUpdateCfgFunc = update.ResolveUpdateCfg

// In tests:
updatePackageFunc = func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun, skipLock bool) error {
    return nil
}
```

## Related Documentation

- [outdated.md](./outdated.md) - Version fetching and selection
- [lock-resolution.md](./lock-resolution.md) - Lock file parsing
- [floating-constraints.md](./floating-constraints.md) - Floating constraint details
- [groups.md](./groups.md) - Package grouping
- [command-execution.md](./command-execution.md) - Shell command execution
