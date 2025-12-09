# Self-Pinning Manifests Architecture

> Self-pinning manifests are files where the declared version IS the installed version - there's no separate lock file.

## Table of Contents

- [Purpose](#purpose)
- [Key Files](#key-files)
- [Configuration](#configuration)
- [Lock Resolution Flow](#lock-resolution-flow)
- [Status Values](#status-values)
- [Display](#display)
- [Wildcard Handling](#wildcard-handling)
- [Update Behavior](#update-behavior)
- [Requirements.txt Example](#requirementstxt-example)
- [vs Lock Files](#vs-lock-files)
- [When to Use](#when-to-use)
- [Testing](#testing)
- [Implementation Details](#implementation-details)
- [Related Documentation](#related-documentation)

---

## Purpose

Some package formats don't have separate lock files:
- `requirements.txt` - Versions are pinned directly
- Custom config files - Version is the source of truth

For these, the "installed version" is the same as the "declared version".

## Key Files

| File | Purpose |
|------|---------|
| `pkg/lock/resolve.go` | Self-pinning detection and handling |
| `pkg/lock/status.go` | `SelfPinned` status constant |
| `pkg/config/model.go` | `SelfPinning` field definition |

## Configuration

```yaml
rules:
  requirements:
    manager: python
    include: ["**/requirements*.txt"]
    format: raw
    # This manifest IS its own lock file
    self_pinning: true
```

## Lock Resolution Flow

**Location:** `pkg/lock/resolve.go`

```go
func ApplyInstalledVersions(packages []formats.Package, cfg *config.Config, baseDir string) ([]formats.Package, error) {
    // For self-pinning rules
    if ruleCfg.SelfPinning {
        version := strings.TrimSpace(packages[idx].Version)

        if version == "" || version == "*" {
            // Wildcard versions can't be self-pinned
            packages[idx].InstalledVersion = "#N/A"
            packages[idx].InstallStatus = InstallStatusVersionMissing
        } else {
            // Use declared version as installed
            packages[idx].InstalledVersion = version
            packages[idx].InstallStatus = InstallStatusSelfPinned
        }
        continue
    }

    // Normal lock file resolution for non-self-pinning rules...
}
```

## Status Values

| Status | Meaning |
|--------|---------|
| `SelfPinned` | Manifest is self-pinning, version used as installed |
| `VersionMissing` | Wildcard version in self-pinning manifest |

## Display

In list/outdated output:

```
RULE         PM      TYPE  STATUS         VERSION  INSTALLED  NAME
requirements python  prod  📌 SelfPinned  3.2.1    3.2.1      requests
requirements python  prod  📌 SelfPinned  2.28.0   2.28.0     requests-oauthlib
```

## Wildcard Handling

When a self-pinning manifest has wildcard versions:

```
# requirements.txt
requests    # No version specified
flask>=2.0  # Range, not pinned
```

These get special status because there's no concrete installed version:

```
RULE         PM      STATUS             VERSION  INSTALLED  NAME
requirements python  ⛔ Floating        *        #N/A       requests
requirements python  📌 SelfPinned      >=2.0    >=2.0      flask
```

> **Note:** Pure wildcards (`*`) are marked as `Floating` because they cannot be updated automatically. Range constraints (`>=2.0`) that aren't compound are treated as self-pinned since they represent a specific minimum version.

## Update Behavior

For self-pinning manifests:
1. **No lock command needed** - The manifest IS the lock
2. **Direct update** - Just modify the version in the file
3. **Validation** - Re-parse manifest to verify change

**Configuration:**

```yaml
rules:
  requirements:
    update:
      # No commands needed - version is updated directly
      timeout_seconds: 60
    self_pinning: true
```

## Requirements.txt Example

**Input file:**
```
requests==2.28.0
flask>=2.0.0
numpy
```

**Parsed packages:**

| Name | Version | Status |
|------|---------|--------|
| requests | 2.28.0 | SelfPinned |
| flask | >=2.0.0 | VersionMissing (range) |
| numpy | * | VersionMissing (wildcard) |

## vs Lock Files

| Aspect | Self-Pinning | Lock File |
|--------|--------------|-----------|
| Installed version | = Declared | From lock file |
| Lock command | Not needed | Required |
| Version precision | Must be exact | Can be range |
| Rollback | Just restore file | Restore + lock |

## When to Use

Use `self_pinning: true` when:
1. Format doesn't have separate lock file
2. Declared version IS the installed version
3. Updates modify the manifest directly

Don't use when:
1. Separate lock file exists
2. Declared version is a range/constraint
3. Lock command needed to resolve dependencies

## Testing

**Test scenarios:**
- Exact version → SelfPinned status
- Wildcard version → VersionMissing status
- Range version → VersionMissing status
- Update without lock command

## Implementation Details

### Status Assignment

```go
const (
    InstallStatusSelfPinned = "SelfPinned"
)
```

### Version Check

```go
if version == "" || version == "*" {
    // Can't determine installed version
    packages[idx].InstallStatus = InstallStatusVersionMissing
} else {
    // Declared = Installed
    packages[idx].InstalledVersion = version
    packages[idx].InstallStatus = InstallStatusSelfPinned
}
```

## Related Documentation

- [lock-resolution.md](./lock-resolution.md) - Lock file resolution
- [configuration.md](./configuration.md) - Self-pinning configuration
- [update.md](./update.md) - Update without lock commands
