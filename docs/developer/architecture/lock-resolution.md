# Lock File Resolution Architecture

> Lock file resolution maps declared package versions to installed versions by parsing lock files.

## Table of Contents

- [Key Files](#key-files)
- [Install Statuses](#install-statuses)
- [Resolution Flow](#resolution-flow)
- [Main Function](#main-function)
- [Self-Pinning](#self-pinning)
- [Lock File Parsing](#lock-file-parsing)
- [Scope Resolution](#scope-resolution)
- [Lock File Configuration](#lock-file-configuration)
- [Package Name Normalization](#package-name-normalization)
- [Warning Handling](#warning-handling)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/lock/resolve.go` | Main resolution logic |
| `pkg/lock/status.go` | Install status constants |

## Install Statuses

**Location:** `pkg/lock/status.go`

| Status | Description | Icon |
|--------|-------------|------|
| `LockFound` | Package found in lock file with version | 🟢 |
| `SelfPinned` | Manifest is self-pinning (e.g., requirements.txt) | 📌 |
| `NotInLock` | Package declared but not in lock file | 🔵 |
| `LockMissing` | No lock file found for this rule | 🟠 |
| `VersionMissing` | Wildcard version with no installed version | ⛔ |
| `NotConfigured` | Rule has no lock file configuration | ⚪ |
| `Floating` | Floating constraint (e.g., `5.*`, `>=1.0.0 <2.0.0`) | ⛔ |

> **Note:** The ⛔ icon indicates the package cannot be processed for updates. The ⚪ icon indicates missing configuration.

## Resolution Flow

```
Group packages by rule ──► Check rule has lock_files ──► Self-pinning?
                                    │                         │
                                    ▼                         ▼
                           Find lock files            Use declared
                           by glob pattern            as installed
                                    │
                                    ▼
                           Extract versions ──► Match packages to versions
```

## Main Function

**Location:** `pkg/lock/resolve.go:21-129`

```go
func ApplyInstalledVersions(packages []formats.Package, cfg *config.Config, baseDir string) ([]formats.Package, error)
```

### Processing Steps

1. **Group packages by rule**
   ```go
   ruleIndexes := make(map[string][]int)
   for idx := range packages {
       ruleIndexes[packages[idx].Rule] = append(ruleIndexes[packages[idx].Rule], idx)
   }
   ```

2. **Check lock file configuration**
   ```go
   if len(ruleCfg.LockFiles) == 0 {
       // Check if rule uses self-pinning
       if ruleCfg.SelfPinning {
           // Use declared version as installed
           packages[idx].InstalledVersion = version
           packages[idx].InstallStatus = InstallStatusSelfPinned
       } else {
           // No lock support
           packages[idx].InstallStatus = InstallStatusNotConfigured
       }
   }
   ```

3. **Scope by source directory**
   - Each package's source file determines its scope
   - Lock files are searched relative to the source's directory

4. **Resolve installed versions**
   ```go
   installed, foundLock, err := resolveInstalledVersions(scopeDir, ruleCfg.LockFiles)
   ```

5. **Match packages to versions**
   ```go
   if version, ok := installed[name]; ok && version != "" {
       packages[idx].InstalledVersion = version
       packages[idx].InstallStatus = InstallStatusLockFound
   } else {
       packages[idx].InstallStatus = InstallStatusNotInLock
   }
   ```

## Self-Pinning

For manifests that ARE their lock files (e.g., `requirements.txt`):

```go
if ruleCfg.SelfPinning {
    version := strings.TrimSpace(packages[idx].Version)
    if version == "" || version == "*" {
        // Wildcard versions can't be self-pinned
        packages[idx].InstalledVersion = "#N/A"
        packages[idx].InstallStatus = InstallStatusVersionMissing
    } else {
        packages[idx].InstalledVersion = version
        packages[idx].InstallStatus = InstallStatusSelfPinned
    }
}
```

**Configuration:**
```yaml
rules:
  requirements:
    self_pinning: true
```

## Lock File Parsing

### Version Extraction

**Location:** `pkg/lock/resolve.go:194-233`

```go
func extractVersionsFromLock(path string, cfg *config.LockFileCfg) (map[string]string, error)
```

**Extraction Methods:**

For lock files, versions are extracted using:

1. **Command-based extraction** (preferred for npm/pnpm/yarn):
   ```yaml
   lock_files:
     - files: ["package-lock.json"]
       commands: |
         npm ls --json --package-lock-only 2>/dev/null || exit 0
   ```

2. **Regex-based extraction** (for other formats):
   ```yaml
   lock_files:
     - files: ["go.sum"]
       format: raw
       extraction:
         pattern: '(?m)^(?P<n>\S+)\s+(?P<version>v[^\s]+)'
   ```

### Regex-based Extraction

For other lock file formats:

```go
if cfg.Extraction.Pattern == "" {
    return nil, fmt.Errorf("lock file extraction pattern missing")
}

matches, err := utils.ExtractAllMatches(cfg.Extraction.Pattern, content)
```

**Required named groups:**
- `name` or `n` - Package name
- `version` - Installed version

**Example pattern for package-lock.json:**
```regex
"(?P<n>[^"]+)":\s*\{[^}]*"version":\s*"(?P<version>[^"]+)"
```

## Scope Resolution

Lock files are searched relative to the package's source directory:

```go
scopeDir := baseDir
if packages[idx].Source != "" {
    scopeDir = filepath.Dir(packages[idx].Source)
}

if scopeDir == "" {
    scopeDir = cfg.WorkingDir
}
if scopeDir == "" {
    scopeDir = "."
}
```

This supports monorepo structures where each package.json has its own lock file.

## Lock File Configuration

```yaml
rules:
  npm:
    lock_files:
      - files: ["**/package-lock.json"]
        format: json
        extraction:
          pattern: '(?m)"(?P<n>[^"]+)":\s*\{[^}]*"version":\s*"(?P<version>[^"]+)"'
```

### Fields

| Field | Description |
|-------|-------------|
| `files` | Glob patterns to find lock files |
| `format` | `json`, `yaml`, `raw` |
| `commands` | Shell command for extracting versions (preferred) |
| `extraction.pattern` | Regex for extracting name/version |

## Package Name Normalization

**Location:** `pkg/lock/resolve.go:235-254`

```go
func normalizeLockPackageName(name, alt string) string
```

**Normalizations:**
- Trim whitespace
- Remove `node_modules/` prefix
- Remove `/go.mod` suffix
- Use `n` group if `name` is empty

## Warning Handling

When a package has a "latest" indicator (`*`, `latest`) but no installed version:

```go
func issueLatestWarning(pkg formats.Package, ruleCfg config.PackageManagerCfg, seen map[string]struct{})
```

The warning is deduplicated by `rule:name` key and relies on the `NotConfigured` status display rather than emitting noisy warnings.

## Testing

**Test File:** `pkg/lock/lockfile_test.go`, `pkg/lock/integration_test.go`

Key test scenarios:
- Lock file found with matching packages
- Lock file missing
- Package not in lock file
- Self-pinning manifests
- Multiple lock files in different directories

## Related Documentation

- [configuration.md](./configuration.md) - Lock file configuration
- [list.md](./list.md) - How list command uses lock resolution
- [outdated.md](./outdated.md) - How outdated command uses lock resolution
- [self-pinning.md](./self-pinning.md) - Self-pinning manifest details
