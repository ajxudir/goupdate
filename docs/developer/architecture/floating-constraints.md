# Floating Constraints (Not Supported)

> Floating constraints are version specifiers that express user intent to stay within a range. **These constraints are not supported for automatic updates** because they cannot be reliably updated without changing user intent.

## Table of Contents

- [What Are Floating Constraints?](#what-are-floating-constraints)
- [Why Floating Constraints Are Not Supported](#why-floating-constraints-are-not-supported)
- [Key File](#key-file)
- [Detection Logic](#detection-logic)
- [How Floating Constraints Are Handled](#how-floating-constraints-are-handled)
- [Status Display](#status-display)
- [User Resolution Options](#user-resolution-options)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## What Are Floating Constraints?

Floating constraints are version specifications that allow flexibility within a defined range:

| Type | Examples | Description |
|------|----------|-------------|
| Wildcards | `5.*`, `5.4.*`, `1.x` | Match any version within segment |
| Pure wildcard | `*` | Match any version (latest) |
| NuGet ranges | `[8.0.0,9.0.0)`, `(1.0,2.0]` | Explicit version ranges |
| Compound | `>=1.0.0 <2.0.0` | Multiple constraints combined |
| OR constraints | `^2.0\|^3.0` | Either constraint matches |

## Why Floating Constraints Are Not Supported

### The Core Problem

When a package has a floating constraint like `"lodash": "5.*"`:

1. **User Intent**: The user wants ANY 5.x version
2. **Automatic Update Problem**: There's no way to "update" a floating constraint
   - Changing `5.*` to `5.2.1` destroys the user's intent
   - Keeping `5.*` means the manifest isn't changed (so what are we updating?)
3. **Lock File Only**: The only sensible action is to run the lock command to get the latest within range
4. **No Per-Package Support**: Most package managers don't have a reliable per-package lock update command

### Package Manager Limitations

| Package Manager | Per-Package Lock Update | Status |
|-----------------|------------------------|--------|
| npm | `npm update <pkg>` exists but modifies manifest | Not reliable |
| pnpm | `pnpm update <pkg>` modifies manifest | Not reliable |
| yarn | `yarn up <pkg>` modifies manifest | Not reliable |
| composer | `composer update <pkg>` works | Limited use |
| pipenv | `pipenv update <pkg>` works | Limited use |
| dotnet | No per-package command | Not supported |
| go | `go mod tidy` updates all | Not supported |

Since most package managers don't have reliable per-package lock-only update commands, floating constraints are marked as unsupported.

## Key File

| File | Purpose |
|------|---------|
| `pkg/utils/version.go` | `IsFloatingConstraint` detection |
| `pkg/lock/resolve.go` | Sets `InstallStatusFloating` status |

## Detection Logic

**Location:** `pkg/utils/version.go:94-134`

```go
func IsFloatingConstraint(version string) bool {
    version = strings.TrimSpace(version)
    if version == "" {
        return false
    }

    // Pure wildcard "*" is floating
    if version == "*" {
        return true
    }

    // Wildcards: "5.*", "5.4.*", "8.x", "1.x.x"
    if strings.Contains(version, ".*") || strings.Contains(version, ".x") {
        return true
    }

    // Trailing wildcard: "5*"
    if strings.HasSuffix(version, "*") && version != "*" {
        return true
    }

    // NuGet/MSBuild ranges: "[8.0.0,9.0.0)", "(1.0,2.0]"
    if strings.HasPrefix(version, "[") || strings.HasPrefix(version, "(") {
        return true
    }

    // Compound: ">=1.0.0 <2.0.0"
    hasMin := strings.Contains(version, ">=") || strings.Contains(version, ">")
    hasMax := strings.Contains(version, "<=") || strings.Contains(version, "<")
    if hasMin && hasMax {
        return true
    }

    // OR constraints: "^2.0|^3.0"
    if strings.Contains(version, "|") {
        return true
    }

    return false
}
```

## How Floating Constraints Are Handled

### In Lock Resolution

**Location:** `pkg/lock/resolve.go`

Packages with floating constraints are marked with `InstallStatusFloating`:

```go
// Mark packages with floating constraints (5.*, >=8.0.0, [8.0.0,9.0.0), etc.)
for idx := range packages {
    if utils.IsFloatingConstraint(packages[idx].Version) {
        packages[idx].InstallStatus = InstallStatusFloating
    }
}
```

### In Outdated Command

**Location:** `cmd/outdated.go:238`

The `deriveOutdatedStatus` function preserves the `Floating` status:

```go
func deriveOutdatedStatus(res outdatedResult) string {
    // Preserve Floating status - these packages cannot be processed automatically
    if res.pkg.InstallStatus == lock.InstallStatusFloating {
        return lock.InstallStatusFloating
    }
    // ... rest of status derivation
}
```

This ensures consistency across commands:
- A package showing `⛔ Floating` in `list` will also show `⛔ Floating` in `outdated`
- The status is NOT changed to `🟢 UpToDate` even when no newer versions are found
- This makes it clear that these packages require manual intervention

### In Update Command

Floating constraints are skipped with an unsupported message:

```go
if utils.IsFloatingConstraint(p.Version) {
    res.status = lock.InstallStatusFloating
    unsupported.Add(p, fmt.Sprintf(
        "floating constraint '%s' cannot be updated automatically; "+
        "remove the constraint or update manually", p.Version))
    continue
}
```

## Status Display

```
RULE    PM      TYPE  VERSION  STATUS       NAME
msbuild dotnet  prod  8.*      ⛔ Floating  Newtonsoft.Json

⛔ msbuild (dotnet): Floating constraint '8.*' cannot be updated automatically; remove the constraint or update manually. (1 package)
```

## User Resolution Options

When a user has floating constraints, they have two options:

### Option 1: Remove the Floating Constraint

Change the manifest to use an exact version:

```diff
- "lodash": "5.*"
+ "lodash": "5.2.1"
```

Then run `goupdate update` normally.

### Option 2: Update Manually

1. Run the package manager's update command directly:
   ```bash
   npm update lodash
   composer update vendor/package
   ```
2. The lock file will be updated to the latest version within the constraint
3. The floating constraint remains in the manifest

## Testing

**Test File:** `pkg/utils/version_test.go`

```go
func TestIsFloatingConstraint(t *testing.T) {
    tests := []struct {
        version  string
        expected bool
    }{
        // Not floating
        {"", false},
        {"1.0.0", false},
        {"^1.0.0", false},  // Semver constraint, not floating
        {"~1.0.0", false},

        // Floating - wildcards
        {"*", true},
        {"5.*", true},
        {"5.4.*", true},
        {"8.x", true},

        // Floating - ranges
        {"[8.0.0,9.0.0)", true},
        {"(1.0,2.0]", true},

        // Floating - compound
        {">=1.0.0 <2.0.0", true},
        {">=3.0,<4.0", true},

        // Floating - OR
        {"^2.0|^3.0", true},
    }
}
```

## Related Documentation

- [outdated.md](./outdated.md) - Outdated command architecture (status handling)
- [update.md](./update.md) - Full update command architecture
- [configuration.md](./configuration.md) - Configuration structure
- [lock-resolution.md](./lock-resolution.md) - How lock files are resolved
