# Incremental Updates Architecture

> Incremental updates select the NEAREST available version rather than the LATEST, enabling one-step-at-a-time upgrades.

## Table of Contents

- [Purpose](#purpose)
- [Key Files](#key-files)
- [Configuration](#configuration)
- [Pattern Matching](#pattern-matching)
- [Pattern Collection Order](#pattern-collection-order)
- [Version Selection](#version-selection)
- [Example](#example)
- [Integration with Update Command](#integration-with-update-command)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Use Cases](#use-cases)
- [Related Documentation](#related-documentation)

---

## Purpose

Some packages have breaking changes between versions. Incremental updates allow:
1. **Gradual upgrades:** Move one major/minor at a time
2. **Safer migrations:** Test each version step
3. **Compatibility:** Avoid skipping versions with migration steps

## Key Files

| File | Purpose |
|------|---------|
| `pkg/config/incremental.go` | Pattern matching logic |
| `pkg/outdated/core.go` | Version selection with incremental mode |

## Configuration

### Global Incremental Packages

```yaml
incremental:
  - react
  - typescript
  - "@company/.*"  # Regex pattern
```

### Rule-Level Incremental

```yaml
rules:
  npm:
    incremental:
      - webpack
      - babel-core
```

### Legacy Syntax

```yaml
incremental_packages:  # Legacy field name
  - react
```

## Pattern Matching

**Location:** `pkg/config/incremental.go:18-46`

```go
func ShouldUpdateIncrementally(p PackageRef, cfg *Config) (bool, error)
```

**Process:**
1. Collect patterns from rule and global config
2. For each pattern:
   - If literal: exact match
   - If regex: pattern match
3. Return true if any pattern matches

### Pattern Detection

**Location:** `pkg/config/incremental.go:72-74`

```go
func usesRegexMeta(pattern string) bool {
    return strings.ContainsAny(pattern, ".*+?{}()|[]^$\\")
}
```

**Pattern Types:**

| Pattern | Type | Matches |
|---------|------|---------|
| `react` | Literal | Exactly "react" |
| `@company/.*` | Regex | Any @company package |
| `typescript` | Literal | Exactly "typescript" |

## Pattern Collection Order

**Location:** `pkg/config/incremental.go:48-62`

```go
func collectIncrementalPatterns(p PackageRef, cfg *Config) []string {
    patterns := make([]string, 0)

    // 1. Rule-level patterns (if package has rule)
    if rule, ok := cfg.Rules[p.GetRule()]; ok {
        patterns = append(patterns, rule.Incremental...)
        patterns = append(patterns, rule.LegacyIncremental...)
    }

    // 2. Global patterns
    patterns = append(patterns, cfg.Incremental...)
    patterns = append(patterns, cfg.LegacyIncremental...)

    return patterns
}
```

**Priority:** Rule-level patterns are checked first.

## Version Selection

**Location:** `pkg/outdated/core.go:368-436`

```go
func SummarizeAvailableVersions(current string, versions []string, cfg *config.VersioningCfg, incremental bool) (string, string, string, error)
```

### Normal Mode (incremental=false)

Selects LATEST version in each category:

```go
isBetterCandidate := func(candidate *parsedVersion, parsed parsedVersion) bool {
    if candidate == nil {
        return true
    }
    return strategy.compare(parsed, *candidate) > 0  // Prefer higher
}
```

### Incremental Mode (incremental=true)

Selects NEAREST version in each category:

```go
isBetterCandidate := func(candidate *parsedVersion, parsed parsedVersion) bool {
    if candidate == nil {
        return true
    }
    return strategy.compare(parsed, *candidate) < 0  // Prefer lower
}
```

## Example

Given package `react` at version `16.8.0` with available versions:
- `16.9.0`, `16.10.0`, `16.14.0`
- `17.0.0`, `17.0.1`, `17.0.2`
- `18.0.0`, `18.1.0`, `18.2.0`

### Normal Mode

| Category | Selected |
|----------|----------|
| Major | `18.2.0` (latest major) |
| Minor | `16.14.0` (latest minor in 16.x) |
| Patch | N/A |

### Incremental Mode

| Category | Selected |
|----------|----------|
| Major | `17.0.0` (nearest major) |
| Minor | `16.9.0` (nearest minor) |
| Patch | N/A |

## Integration with Update Command

**Location:** `cmd/update.go:252-259`

```go
incremental, incrementalErr := config.ShouldUpdateIncrementally(p, cfg)
if incrementalErr != nil {
    res.status = "ConfigError"
    res.err = incrementalErr
    continue
}

major, minor, patch, _ := outdated.SummarizeAvailableVersions(
    outdated.CurrentVersionForOutdated(p), filtered, versioning, incremental)
```

## Error Handling

Invalid regex patterns return an error:

```go
if err := regexp.Compile(pattern); err != nil {
    return false, fmt.Errorf("invalid incremental package pattern %q: %w", pattern, err)
}
```

## Testing

**Test File:** `pkg/config/incremental_test.go`

Key test scenarios:
- Literal pattern matching
- Regex pattern matching
- Rule-level vs global patterns
- Invalid regex handling
- Legacy field support

## Use Cases

### Large Version Jumps

For packages like React that have migration guides between versions:

```yaml
incremental:
  - react
  - react-dom
```

Allows upgrading 16 → 17 → 18 one step at a time.

### Internal Packages

For internal packages with breaking changes:

```yaml
incremental:
  - "@company/.*"
```

### Specific Problem Packages

For packages known to have compatibility issues:

```yaml
rules:
  npm:
    incremental:
      - webpack      # Complex config changes
      - typescript   # Type system changes
```

## Related Documentation

- [version-comparison.md](./version-comparison.md) - Version selection logic
- [update.md](./update.md) - Update command usage
- [configuration.md](./configuration.md) - Configuration options
