# Plan: Multi-Pattern Extraction with Version Detection (Updated v2)

**Date:** 2025-12-12
**Status:** Awaiting Approval
**Branch:** claude/organize-mock-data-01Y1vCHWXSvHwCvA6nU99JcH

---

## Overview

Implement a **reusable** multi-pattern extraction system with corrected behavior:

- **If `detect` is NOT set** → Pattern is ALWAYS applied (default = true)
- **If `detect` IS set** → Pattern only activates if `detect` matches
- **Multiple patterns CAN match** → All matching patterns run (NOT exclusive)
- **Results are combined** → All matching patterns contribute results

---

## Part 1: Corrected Detect Logic

### Previous (Wrong) Logic
```
First detect match wins (exclusive) ❌
```

### Corrected Logic
```
All patterns with matching detect run (additive) ✅
Patterns without detect always run (default = true) ✅
```

### Algorithm

```go
// SelectPatterns returns ALL applicable patterns for the given content.
//
// Logic:
//   1. If Patterns array is empty, return single Pattern field
//   2. For each pattern in Patterns:
//      - If Detect is empty → ALWAYS include (default = true)
//      - If Detect is set → Include ONLY if detect matches content
//   3. Return ALL matching patterns (not just first)
//   4. Fallback to single pattern field if nothing matched
func SelectPatterns(content string, cfg *ExtractionCfg) []string {
    if len(cfg.Patterns) == 0 {
        if cfg.Pattern != "" {
            return []string{cfg.Pattern}
        }
        return nil
    }

    var result []string

    for _, p := range cfg.Patterns {
        if p.Detect == "" {
            // No detect = always include (default true)
            result = append(result, p.Pattern)
        } else if matchesDetect(content, p.Detect) {
            // Detect set and matches = include
            result = append(result, p.Pattern)
        }
        // Detect set but doesn't match = skip
    }

    // Fallback to single pattern if nothing matched
    if len(result) == 0 && cfg.Pattern != "" {
        return []string{cfg.Pattern}
    }

    return result
}
```

### Example Behavior

```yaml
patterns:
  - name: "v9_specific"
    detect: "lockfileVersion:\\s*'9"  # Only runs for v9
    pattern: '..v9 pattern..'

  - name: "common_fallback"
    # No detect = always runs
    pattern: '..common pattern..'

  - name: "v6_v7_v8"
    detect: "lockfileVersion:\\s*'[678]"  # Only for v6/v7/v8
    pattern: '..v6-8 pattern..'
```

**For v9 file:** Runs `v9_specific` + `common_fallback` (2 patterns)
**For v6 file:** Runs `v6_v7_v8` + `common_fallback` (2 patterns)
**For v5 file:** Runs `common_fallback` only (1 pattern)

---

## Part 2: Package Exclusion Status (NEW)

### Current Behavior (Problem)
Ignored packages are completely filtered out - they don't appear in results.

### New Behavior (Solution)
Ignored packages appear in results with status explaining why they're excluded.

### New Status Constant

**File:** `pkg/lock/status.go`

```go
const (
    // ... existing statuses ...

    // InstallStatusIgnored indicates the package is excluded by config.
    // The package matches an ignore pattern or has ignore: true in package_overrides.
    InstallStatusIgnored = "Ignored"
)
```

### Display Format

```
┌──────────────────┬─────────┬───────────┬──────────────────┐
│ Package          │ Version │ Installed │ Status           │
├──────────────────┼─────────┼───────────┼──────────────────┤
│ lodash           │ ^4.17.21│ 4.17.21   │ ✓ LockFound      │
│ eslint           │ ^8.0.0  │ -         │ ⊘ Ignored        │ ← NEW
│ babel-core       │ ^7.0.0  │ -         │ ⊘ Ignored        │ ← NEW
└──────────────────┴─────────┴───────────┴──────────────────┘
```

### Verbose Output

When `--verbose` is enabled:

```
[DEBUG] Package 'eslint' ignored: matches pattern 'eslint-*' in ignore config
[DEBUG] Package 'babel-core' ignored: package_overrides.ignore = true
```

### Implementation

**File:** `pkg/formats/helpers.go`

```go
// getIgnoreReason returns the reason a package is ignored, or empty if not ignored.
func getIgnoreReason(name string, cfg *config.PackageManagerCfg) string {
    if cfg == nil {
        return ""
    }

    for _, pattern := range cfg.Ignore {
        if matched, _ := regexp.MatchString(pattern, name); matched {
            return fmt.Sprintf("matches ignore pattern '%s'", pattern)
        }
    }

    if override, exists := cfg.PackageOverrides[name]; exists && override.Ignore {
        return "package_overrides.ignore = true"
    }

    return ""
}
```

**File:** `pkg/formats/json.go`, `pkg/formats/raw.go`, etc.

Instead of:
```go
if shouldIgnorePackage(name, cfg) {
    continue  // Skip completely
}
```

Change to:
```go
if reason := getIgnoreReason(name, cfg); reason != "" {
    pkg.InstallStatus = lock.InstallStatusIgnored
    pkg.IgnoreReason = reason  // New field
    verbose.PackageFiltered(name, reason)
    // Continue adding to result, don't skip
}
```

---

## Part 3: All Config Fields That Can Use Multi-Pattern

### Complete Analysis

| Field | Type | Current | Multi-Pattern Benefit |
|-------|------|---------|----------------------|
| **Extraction** ||||
| `extraction.pattern` | string | Single | Version-specific manifest parsing |
| `lock_files[].extraction.pattern` | string | Single | Lock file version support |
| `lock_files[].command_extraction.pattern` | string | Single | Command output format variations |
| `outdated.extraction.pattern` | string | Single | Registry response format variations |
| **Version Exclusion** ||||
| `exclude_versions` (global) | []string | Array (all match) | Conditional exclusions by package type |
| `rule.exclude_versions` | []string | Array (all match) | Rule-specific conditional exclusions |
| `outdated.exclude_version_patterns` | []string | Array (all match) | Conditional per-package exclusions |
| **Package Filtering** ||||
| `ignore` | []string | Array (all match) | Conditional ignore by package type |
| **Versioning** ||||
| `versioning.regex` | string | Single | SemVer vs CalVer detection |

### Fields NOT Suitable for Multi-Pattern

| Field | Reason |
|-------|--------|
| `include`, `exclude` | Glob patterns, not regex |
| `files` | Glob patterns, not regex |
| `constraint_mapping` | Key-value mapping, not regex |

---

## Part 4: Unified PatternCfg Schema

### Reusable Struct

```go
// PatternCfg defines a conditional pattern for extraction or exclusion.
// This struct is reusable across all config areas that use regex patterns.
type PatternCfg struct {
    // Name is a descriptive identifier for debugging/logging.
    Name string `yaml:"name,omitempty"`

    // Detect is a regex that must match content for this pattern to activate.
    // If empty (default), the pattern is ALWAYS applied.
    // If set, pattern only activates when detect matches.
    // Multiple patterns with detect can all match and run.
    Detect string `yaml:"detect,omitempty"`

    // Pattern is the extraction/matching regex.
    Pattern string `yaml:"pattern"`
}
```

### Updated ExtractionCfg

```go
type ExtractionCfg struct {
    // Pattern is a single regex pattern (backwards compatible).
    Pattern string `yaml:"pattern,omitempty"`

    // Patterns is an array of conditional patterns (NEW).
    // All patterns with matching detect (or no detect) are applied.
    // Results from all matching patterns are combined.
    Patterns []PatternCfg `yaml:"patterns,omitempty"`

    // ... existing XML fields unchanged
}
```

### Updated OutdatedCfg

```go
type OutdatedCfg struct {
    // ExcludeVersionPatterns can be:
    // - Simple strings (backwards compatible): ["alpha", "beta"]
    // - PatternCfg objects (NEW): [{detect: "^@types/", pattern: "next"}]
    // Behavior:
    //   - Without detect: Pattern always applied
    //   - With detect: Pattern only applied if detect matches package name
    ExcludeVersionPatterns []interface{} `yaml:"exclude_version_patterns,omitempty"`
}
```

---

## Part 5: Example Configurations

### Lock File Extraction (Multiple Versions)

```yaml
pnpm:
  lock_files:
    - files: ["**/pnpm-lock.yaml"]
      format: raw
      extraction:
        patterns:
          # v9-specific pattern (runs for v9 only)
          - name: "v9"
            detect: "lockfileVersion:\\s*'9"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          # v6/v7/v8 pattern (runs for those versions)
          - name: "v6_v7_v8"
            detect: "lockfileVersion:\\s*'[678]"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          # Fallback for unknown versions (always runs if nothing else matches)
          - name: "fallback"
            # No detect = always runs
            pattern: '(?m)^\s+(?P<n>[@\w\-\.\/]+):\s+(?P<version>[\d\.]+)'
```

### Yarn Classic vs Berry

```yaml
yarn:
  lock_files:
    - files: ["**/yarn.lock"]
      format: raw
      extraction:
        patterns:
          - name: "berry"
            detect: "__metadata:\\s*\\n\\s+version:"
            pattern: '(?m)^"(?P<n>@?[\w\-\.\/]+)@npm:[^"]+":.*\n\s+version:\s*(?P<version>[\d\.]+)'

          - name: "classic"
            detect: "#\\s*yarn lockfile v1"
            pattern: '(?m)^"?(?P<n>@?[\w\-\.\/]+)@[^:]+:\\s*\n\\s+version\\s+"(?P<version>[^"]+)"'
```

### Conditional Exclude Version Patterns

```yaml
rules:
  npm:
    outdated:
      exclude_version_patterns:
        # Always exclude alpha/beta (no detect = always runs)
        - pattern: "(?i)[._-]alpha"
        - pattern: "(?i)[._-]beta"
        - pattern: "(?i)[._-]rc"

        # Only exclude 'next' for @types packages
        - name: "types_next"
          detect: "^@types/"
          pattern: "(?i)[._-]next"

        # Only exclude 'experimental' for react packages
        - name: "react_experimental"
          detect: "^react"
          pattern: "(?i)experimental"

        # Only exclude 'canary' for Next.js
        - name: "nextjs_canary"
          detect: "^next$"
          pattern: "(?i)canary"
```

---

## Part 6: Implementation Plan

### Phase 1: Add PatternCfg Struct (~30 lines)

**File:** `pkg/config/model.go`

```go
// PatternCfg defines a conditional pattern for extraction or exclusion.
type PatternCfg struct {
    Name    string `yaml:"name,omitempty"`
    Detect  string `yaml:"detect,omitempty"`
    Pattern string `yaml:"pattern"`
}
```

### Phase 2: Update ExtractionCfg (~20 lines)

**File:** `pkg/config/model.go`

Add `Patterns []PatternCfg` to ExtractionCfg, LockCommandExtractionCfg, OutdatedExtractionCfg.

### Phase 3: Add Pattern Selection Utility (~100 lines)

**File:** `pkg/utils/patterns.go` (NEW)

```go
package utils

import "regexp"

// SelectExtractionPatterns returns all applicable patterns based on content.
// All patterns with matching detect (or no detect) are included.
func SelectExtractionPatterns(content string, singlePattern string, patterns []PatternCfg) []string

// MatchesDetect checks if content matches a detect regex.
func MatchesDetect(content, detectPattern string) bool
```

### Phase 4: Add Ignored Status (~80 lines)

**File:** `pkg/lock/status.go`
- Add `InstallStatusIgnored = "Ignored"`

**File:** `pkg/formats/model.go`
- Add `IgnoreReason string` field to Package struct

**File:** `pkg/formats/helpers.go`
- Add `getIgnoreReason()` function

**File:** `pkg/display/status.go`
- Add formatting for Ignored status

### Phase 5: Update Format Parsers (~60 lines)

**Files:** `pkg/formats/json.go`, `pkg/formats/raw.go`, `pkg/formats/yaml.go`, `pkg/formats/xml.go`

Instead of skipping ignored packages, set status and include them.

### Phase 6: Update Lock File Extraction (~40 lines)

**File:** `pkg/lock/resolve.go`

Update `extractVersionsFromLock()` to use `SelectExtractionPatterns()`.

### Phase 7: Update default.yml (~100 lines)

**File:** `pkg/config/default.yml`

Add multi-pattern configs for pnpm (v6-v9), yarn (classic/berry).

### Phase 8: Add Testdata (~350 lines)

| Directory | Version | Files |
|-----------|---------|-------|
| `pkg/testdata/npm_v3/` | npm v3 | package.json, package-lock.json |
| `pkg/testdata/pnpm_v7/` | pnpm v7 | package.json, pnpm-lock.yaml |
| `pkg/testdata/pnpm_v8/` | pnpm v8 | package.json, pnpm-lock.yaml |
| `pkg/testdata/pnpm_v9/` | pnpm v9 | package.json, pnpm-lock.yaml |
| `pkg/testdata/ignored_packages/` | Ignored test | package.json with ignored packages |

### Phase 9: Add Integration Tests (~250 lines)

**File:** `pkg/lock/integration_test.go`

```go
// Version-specific tests
func TestIntegration_NPM_LockfileV3(t *testing.T)
func TestIntegration_PNPM_LockfileV7(t *testing.T)
func TestIntegration_PNPM_LockfileV8(t *testing.T)
func TestIntegration_PNPM_LockfileV9(t *testing.T)

// Pattern detection tests
func TestIntegration_PatternDetection_PNPM(t *testing.T)
func TestIntegration_PatternDetection_Yarn(t *testing.T)

// Ignored packages test (NEW)
func TestIntegration_IgnoredPackages_ShowStatus(t *testing.T)
func TestIntegration_IgnoredPackages_VerboseReason(t *testing.T)
```

### Phase 10: Add Unit Tests (~150 lines)

**File:** `pkg/utils/patterns_test.go` (NEW)

```go
func TestSelectPatterns_AllMatchingRun(t *testing.T)
func TestSelectPatterns_NoDetect_AlwaysRuns(t *testing.T)
func TestSelectPatterns_WithDetect_OnlyIfMatches(t *testing.T)
func TestSelectPatterns_Combined_Results(t *testing.T)
func TestMatchesDetect(t *testing.T)
```

**File:** `pkg/formats/helpers_test.go`

```go
func TestGetIgnoreReason_MatchesPattern(t *testing.T)
func TestGetIgnoreReason_OverrideIgnore(t *testing.T)
func TestGetIgnoreReason_NotIgnored(t *testing.T)
```

---

## Part 7: Lock File Version Detection

### Detection Patterns

| Package Manager | Version | Detection Regex | Example |
|-----------------|---------|-----------------|---------|
| **npm v1** | 1 | `"lockfileVersion":\s*1[,\s}]` | `"lockfileVersion": 1` |
| **npm v2** | 2 | `"lockfileVersion":\s*2[,\s}]` | `"lockfileVersion": 2` |
| **npm v3** | 3 | `"lockfileVersion":\s*3[,\s}]` | `"lockfileVersion": 3` |
| **pnpm v6** | 6 | `lockfileVersion:\s*'6` | `lockfileVersion: '6.0'` |
| **pnpm v7** | 7 | `lockfileVersion:\s*'7` | `lockfileVersion: '7.0'` |
| **pnpm v8** | 8 | `lockfileVersion:\s*'8` | `lockfileVersion: '8.0'` |
| **pnpm v9** | 9 | `lockfileVersion:\s*'9` | `lockfileVersion: '9.0'` |
| **yarn classic** | v1 | `#\s*yarn lockfile v1` | `# yarn lockfile v1` |
| **yarn berry** | v2+ | `__metadata:\s*\n\s+version:` | `__metadata:\n  version: 8` |

---

## Part 8: Summary

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `pkg/config/model.go` | Add PatternCfg, update structs | ~50 |
| `pkg/utils/patterns.go` | NEW - Pattern selection | ~100 |
| `pkg/utils/patterns_test.go` | NEW - Unit tests | ~150 |
| `pkg/lock/status.go` | Add InstallStatusIgnored | ~5 |
| `pkg/formats/model.go` | Add IgnoreReason field | ~5 |
| `pkg/formats/helpers.go` | Add getIgnoreReason() | ~30 |
| `pkg/formats/*.go` | Update parsers for ignored status | ~60 |
| `pkg/display/status.go` | Format Ignored status | ~10 |
| `pkg/lock/resolve.go` | Use pattern selection | ~40 |
| `pkg/config/default.yml` | Multi-pattern configs | ~100 |
| `pkg/testdata/*` | New testdata dirs | ~350 |
| `pkg/lock/integration_test.go` | New tests | ~250 |
| **Total** | | **~1150 lines** |

### Key Design Decisions

1. **All matching patterns run** - NOT first match wins
2. **No detect = always runs** - Default is true
3. **Ignored packages show in results** - With status explaining why
4. **Verbose shows reason** - When `--verbose` enabled
5. **Backwards compatible** - Single `pattern` field still works

---

## Part 9: Documentation Updates

### Add to docs/status-reference.md (or similar)

```markdown
## Package Status Values

| Status | Icon | Description |
|--------|------|-------------|
| LockFound | ✓ | Version found in lock file |
| NotInLock | ℹ | Package not in lock file |
| LockMissing | ⚠ | Lock file doesn't exist |
| Floating | ⊗ | Floating constraint (*, >=x) |
| NotConfigured | ⊘ | No lock file config for rule |
| SelfPinned | 📌 | Manifest is self-pinning |
| VersionMissing | ⊗ | No concrete version found |
| **Ignored** | ⊘ | **Package excluded by config** |

### Ignored Status

Packages with `Ignored` status are excluded from processing due to:
- Matching an `ignore` pattern in the rule config
- Having `ignore: true` in `package_overrides`

Use `--verbose` to see the specific reason each package is ignored.
```

---

## Awaiting Approval

Please confirm:

1. **Corrected detect logic?**
   - All matching patterns run (additive, not exclusive)
   - No detect = always runs (default true)

2. **Ignored status implementation?**
   - Show ignored packages in results with status
   - Verbose output shows reason

3. **Testdata scope:**
   - npm v3
   - pnpm v6, v7, v8, v9
   - Ignored packages testdata

4. **Any additional requirements?**
