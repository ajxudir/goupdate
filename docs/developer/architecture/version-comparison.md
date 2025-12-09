# Version Comparison Architecture

> The version comparison system parses, filters, and selects versions based on constraints, flags, and versioning strategies.

## Table of Contents

- [Key Files](#key-files)
- [Constraint Types](#constraint-types)
- [Version Filtering Flow](#version-filtering-flow)
- [Version Exclusions](#version-exclusions)
- [Filter by Constraint](#filter-by-constraint)
- [Summarize Available Versions](#summarize-available-versions)
- [Select Target Version](#select-target-version)
- [Versioning Strategies](#versioning-strategies)
- [Constraint Normalization](#constraint-normalization)
- [Current Version Resolution](#current-version-resolution)
- [Segment Counting](#segment-counting)
- [Filter Newer Versions](#filter-newer-versions)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/outdated/core.go` | Main version filtering logic |
| `pkg/outdated/versioning.go` | Versioning strategies |
| `pkg/utils/version.go` | Version parsing utilities |

## Constraint Types

| Symbol | Name | Behavior |
|--------|------|----------|
| `^` | Compatible | Same major, newer minor/patch |
| `~` | Patch | Same major.minor, newer patch |
| `>=` | Minimum | Greater than or equal |
| `>` | Greater | Strictly greater |
| `<=` | Maximum | Less than or equal |
| `<` | Less | Strictly less |
| `=` | Exact | Exact match only |
| `*` | Any | No constraint |
| (empty) | Default | Treated as exact match |

## Version Filtering Flow

```
Get Available Versions ──► Apply Version Exclusions ──► Filter by Constraint
                                                               │
                                                               ▼
                           Select Target Version ◄── Summarize Candidates
```

## Version Exclusions

**Location:** `pkg/outdated/core.go:306-357`

```go
func applyVersionExclusions(versions []string, cfg *config.OutdatedCfg) ([]string, error)
```

**Exclusion Types:**

1. **Exact versions:** `exclude_versions: ["1.0.0", "2.0.0"]`
2. **Pattern matching:** `exclude_version_patterns: ["(?i)alpha|beta|rc"]`

**Default Exclusion Pattern:**
```go
productionSafeVersionPattern = "(?i)(?:^|[._\\-/])((?:alpha|beta|rc|canary|dev|snapshot|nightly|preview)(?:[._\\-/]?[0-9A-Za-z]+)*)(?:\\+[^\\s]*)?$"
```

## Filter by Constraint

**Location:** `pkg/outdated/core.go:506-592`

```go
func FilterVersionsByConstraint(p formats.Package, versions []string, flags UpdateSelectionFlags) []string
```

### Flag Override Behavior

| Flag | Constraint Override |
|------|---------------------|
| `--major` | No constraint (all versions) |
| `--minor` | `^` (same major) |
| `--patch` | `~` (same major.minor) |

### Constraint Matching

```go
switch constraint {
case "^":
    // Same major version only
    if semver.Major(reference) == semver.Major(canonical) {
        allowed = append(allowed, raw)
    }
case "~":
    // Same major.minor version only
    if semver.MajorMinor(reference) == semver.MajorMinor(canonical) {
        allowed = append(allowed, raw)
    }
case ">=":
    if semver.Compare(canonical, reference) >= 0 {
        allowed = append(allowed, raw)
    }
// ... etc
}
```

## Summarize Available Versions

**Location:** `pkg/outdated/core.go:475-553`

```go
func SummarizeAvailableVersions(current string, versions []string, cfg *config.VersioningCfg, incremental bool) (string, string, string, error)
```

**Returns:** Best major, minor, and patch candidates.

**Logic:**
1. Parse current version as base
2. For each available version:
   - If major > base.major → major candidate
   - If same major, minor > base.minor → minor candidate
   - If same major.minor, patch > base.patch → patch candidate
   - If same major.minor.patch, use full semver comparison → patch candidate (handles prerelease → stable)
3. With incremental mode: select NEAREST instead of LATEST

**Pre-release Handling:**

When comparing versions with identical major.minor.patch (e.g., `1.0.0-rc03` vs `1.0.0`), the function uses full semver comparison to detect:
- Pre-release to stable transitions: `1.0.0-rc03` → `1.0.0`
- Pre-release to newer pre-release: `1.0.0-alpha` → `1.0.0-beta`

These are categorized as patch updates since the numeric version parts are identical.

## Select Target Version

**Location:** `pkg/outdated/core.go:654-773`

```go
func SelectTargetVersion(major, minor, patch string, flags UpdateSelectionFlags, constraint string, incremental bool) (string, error)
```

**Selection Priority (non-incremental mode):**

| Flag/Constraint | Priority |
|-----------------|----------|
| `--major` | major → minor → patch |
| `--minor` | minor → patch |
| `--patch` | patch only |
| `*` or empty | major → minor → patch |
| `^` | minor → patch |
| `~` | patch only |

**Selection Priority (incremental mode):**

When `incremental=true`, priority is reversed to favor smallest updates:

| Flag/Constraint | Priority |
|-----------------|----------|
| `--major` | patch → minor → major |
| `--minor` | patch → minor |
| `--patch` | patch only |
| `*` or empty | patch → minor → major |
| `^` | patch → minor |
| `~` | patch only |

## Versioning Strategies

**Location:** `pkg/outdated/versioning.go`

### Semver Strategy (Default)

```go
format: semver
```

- Uses `golang.org/x/mod/semver`
- Parses major.minor.patch components
- Standard semantic versioning comparison
- Preserves pre-release identifiers (`1.0.0-rc03` ≠ `1.0.0`)

### Numeric Strategy

```go
format: numeric
```

- For non-semver version numbers
- Simple numeric comparison
- Used when versions are just numbers (e.g., "20231201")

### Ordered Strategy

```go
format: ordered  # aliases: "list", "sorted"
sort: asc|desc
```

- Uses position-based ordering in the returned version list
- For versions that don't follow any numeric pattern (e.g., Debian codenames)
- Respects configured sort order (default: `desc` assumes newest first)

### Regex Strategy

```go
format: regex
regex: "(?P<major>\\d+)\\.(?P<minor>\\d+)\\.(?P<patch>\\d+)"
```

- Custom regex for version extraction
- Named groups: `major`, `minor`, `patch`
- For non-standard version formats

### Multi-Segment Version Support

The versioning system supports versions beyond standard semver:

| Format | Example | Behavior |
|--------|---------|----------|
| 4+ segments | `1.0.0.0`, `1.0.0.1` | First 3 segments used for major/minor/patch; full string for deduplication |
| CalVer | `2024.01.15` | Year=major, month=minor, day=patch |
| Build numbers | `150`, `200` | Treated as major-only (numeric strategy recommended) |

**Key Implementation Details:**

1. **Version Regex** (`pkg/utils/core.go`): Supports any number of `.digit` segments
2. **extractParts** (`versioning.go`): Selects best regex match (most groups, longest) to handle `1.0.0.0`
3. **keyFor** (`versioning.go`): Uses canonical form to prevent incorrect deduplication of similar versions
4. **FilterVersionsByConstraint** (`core.go`): Passes through non-semver versions when no constraint applies

## Constraint Normalization

**Location:** `pkg/outdated/core.go:607-625`

```go
func NormalizeConstraint(constraint string) string
```

**Mappings:**

| Input | Normalized |
|-------|------------|
| `==` | `=` |
| `~=` | `~` |
| `exact` | `=` |
| unsupported | `=` |

## Current Version Resolution

**Location:** `pkg/outdated/core.go:358-366`

```go
func CurrentVersionForOutdated(p formats.Package) string {
    current := strings.TrimSpace(p.InstalledVersion)
    if current != "" && current != "#N/A" {
        return current  // Prefer installed version
    }
    return strings.TrimSpace(p.Version)  // Fall back to declared
}
```

**Priority:**
1. Installed version from lock file
2. Declared version from manifest

## Segment Counting

**Location:** `pkg/outdated/core.go:627-646`

```go
func countConstraintSegments(version string) int
```

For exact matching with partial versions:
- `1` → major only
- `1.2` → major.minor
- `1.2.3` → full version

## Filter Newer Versions

**Location:** `pkg/outdated/core.go:448-495`

```go
func filterNewerVersionsWithStrategy(current string, versions []string, strategy versioningStrategy) []string
```

**Process:**
1. Parse current version as base
2. Filter versions newer than base
3. Deduplicate using strategy key
4. Sort according to strategy

## Testing

**Test File:** `pkg/outdated/outdated_test.go`

Key test scenarios:
- Constraint filtering
- Version exclusions
- Flag overrides
- Incremental selection
- Edge cases (empty versions, invalid formats)

## Related Documentation

- [outdated.md](./outdated.md) - Outdated command architecture
- [update.md](./update.md) - How versions are used for updates
- [incremental-updates.md](./incremental-updates.md) - Incremental version selection
