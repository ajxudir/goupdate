# Plan: Multi-Pattern Extraction with Version Detection (Updated)

**Date:** 2025-12-12
**Status:** Awaiting Approval
**Branch:** claude/organize-mock-data-01Y1vCHWXSvHwCvA6nU99JcH

---

## Overview

Implement a **reusable** multi-pattern extraction system that can be used across all config areas that use regex patterns. The key behavior:

- **If `detect` is NOT set** → Pattern is ALWAYS applied (match all)
- **If `detect` IS set** → Pattern only activates if `detect` matches the content

This allows conditional pattern activation while maintaining backwards compatibility.

---

## Part 1: All Places Where Multi-Pattern Can Be Reused

### Analysis of Config Fields Using Patterns

| Location | Current Field | Use Case | Multi-Pattern Benefit |
|----------|---------------|----------|----------------------|
| `lock_files[].extraction.pattern` | Single pattern | Lock file version parsing | Different patterns for v6/v7/v8/v9 |
| `lock_files[].command_extraction.pattern` | Single pattern | Command output parsing | Different output formats |
| `outdated.extraction.pattern` | Single pattern | Registry response parsing | Different registry formats |
| `outdated.exclude_version_patterns[]` | Pattern array (all match) | Version exclusion | Conditional exclusions per PM |
| `exclude_versions[]` (global) | Pattern array (all match) | Global exclusions | Conditional exclusions |
| `rule.extraction.pattern` | Single pattern | Manifest parsing | Format variations |
| `outdated.versioning.regex` | Single pattern | Version component extraction | Different version schemes |

### Code Locations Using Patterns

| File | Function | Pattern Field | Lines |
|------|----------|---------------|-------|
| `pkg/lock/resolve.go` | `extractVersionsFromLock()` | `extraction.pattern` | 297-301 |
| `pkg/lock/resolve.go` | Lock command extraction | `command_extraction.pattern` | 427, 641 |
| `pkg/outdated/parsers.go` | `parseRawWithExtraction()` | `extraction.pattern` | 189 |
| `pkg/outdated/core.go` | `applyExclusions()` | `exclude_version_patterns[]` | 429 |
| `pkg/formats/raw.go` | `Parse()` | `extraction.pattern` | 42-43 |
| `pkg/update/raw.go` | `updateDeclaredVersion()` | `extraction.pattern` | 37-42 |

---

## Part 2: Unified Pattern Config Schema

### New Reusable Struct

```go
// PatternCfg defines a single pattern with optional conditional detection.
// This struct is reusable across all extraction and exclusion configs.
type PatternCfg struct {
    // Name is a descriptive identifier for debugging/logging.
    Name string `yaml:"name,omitempty"`

    // Detect is a regex that must match the content for this pattern to activate.
    // If empty, the pattern is ALWAYS applied (no condition).
    // If set, pattern only activates when detect matches.
    Detect string `yaml:"detect,omitempty"`

    // Pattern is the extraction/matching regex.
    Pattern string `yaml:"pattern"`
}
```

### Updated ExtractionCfg

```go
type ExtractionCfg struct {
    // Pattern is a single regex pattern (backwards compatible).
    // Used when Patterns array is empty.
    Pattern string `yaml:"pattern,omitempty"`

    // Patterns is an array of conditional patterns.
    // Behavior:
    //   - Patterns WITHOUT detect: Always applied, results combined
    //   - Patterns WITH detect: Only applied if detect matches content
    //   - First pattern WITH detect that matches wins (exclusive)
    //   - Patterns without detect are additive
    Patterns []PatternCfg `yaml:"patterns,omitempty"`

    // ... existing XML fields unchanged
    Path           string `yaml:"path,omitempty"`
    NameAttr       string `yaml:"name_attr,omitempty"`
    // ...
}
```

### Updated Exclude Version Patterns

```go
type OutdatedCfg struct {
    // ExcludeVersionPatterns lists regex patterns for versions to exclude.
    // Can be simple strings (backwards compatible) or PatternCfg objects.
    // Behavior with PatternCfg:
    //   - Without detect: Pattern always applied
    //   - With detect: Pattern only applied if detect matches package name/version
    ExcludeVersionPatterns []interface{} `yaml:"exclude_version_patterns,omitempty"`

    // ... rest unchanged
}
```

---

## Part 3: Pattern Selection Logic

### Algorithm

```go
// SelectPatterns returns all applicable patterns for the given content.
//
// Logic:
//   1. If Patterns array is empty, return single Pattern field
//   2. For patterns WITH detect field:
//      - Check if detect matches content
//      - First matching detect wins (exclusive for that pattern type)
//   3. For patterns WITHOUT detect field:
//      - Always included (additive)
//   4. Return combined list of applicable patterns
func SelectPatterns(content string, cfg *ExtractionCfg) []string {
    if len(cfg.Patterns) == 0 {
        if cfg.Pattern != "" {
            return []string{cfg.Pattern}
        }
        return nil
    }

    var result []string
    var foundDetectMatch bool

    // First pass: find pattern with matching detect (exclusive)
    for _, p := range cfg.Patterns {
        if p.Detect != "" {
            if matchesDetect(content, p.Detect) && !foundDetectMatch {
                result = append(result, p.Pattern)
                foundDetectMatch = true
                break // Only first detect match wins
            }
        }
    }

    // Second pass: add all patterns without detect (additive)
    for _, p := range cfg.Patterns {
        if p.Detect == "" {
            result = append(result, p.Pattern)
        }
    }

    // Fallback to single pattern if nothing matched
    if len(result) == 0 && cfg.Pattern != "" {
        return []string{cfg.Pattern}
    }

    return result
}
```

---

## Part 4: Example Configurations

### Lock File Extraction (Version-Specific)

```yaml
pnpm:
  lock_files:
    - files: ["**/pnpm-lock.yaml"]
      format: raw
      extraction:
        patterns:
          # Version-specific patterns (exclusive - first match wins)
          - name: "v9"
            detect: "lockfileVersion:\\s*'9"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          - name: "v6_v7_v8"
            detect: "lockfileVersion:\\s*'[678]"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          - name: "v5_legacy"
            detect: "lockfileVersion:\\s*5"
            pattern: '(?m)^\s{4}(?P<n>[@\w\-\.\/]+):\s+(?P<version>[\d\.]+)'
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

### Exclude Version Patterns (Conditional)

```yaml
# Global exclusions with conditional patterns
exclude_versions:
  # Always applied (no detect)
  - "(?i)[._-]alpha"
  - "(?i)[._-]beta"
  - "(?i)[._-]rc"

rules:
  npm:
    outdated:
      exclude_version_patterns:
        # Always applied
        - pattern: "(?i)[._-]canary"

        # Only for scoped packages
        - name: "scoped_next"
          detect: "^@"  # Matches package names starting with @
          pattern: "(?i)[._-]next"

        # Only for specific package
        - name: "react_experimental"
          detect: "^react$"
          pattern: "(?i)experimental"
```

---

## Part 5: Implementation Plan

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

Add `Patterns []PatternCfg` field to `ExtractionCfg`.

### Phase 3: Add Pattern Selection Utility (~80 lines)

**File:** `pkg/utils/patterns.go` (NEW)

```go
package utils

// SelectExtractionPatterns returns applicable patterns based on content.
func SelectExtractionPatterns(content string, singlePattern string, patterns []PatternCfg) []string

// matchesDetect checks if content matches a detect regex.
func matchesDetect(content, detectPattern string) bool
```

### Phase 4: Update Lock File Extraction (~40 lines)

**File:** `pkg/lock/resolve.go`

Update `extractVersionsFromLock()` to use `SelectExtractionPatterns()`.

### Phase 5: Update default.yml (~100 lines)

**File:** `pkg/config/default.yml`

Add multi-pattern configs for:
- pnpm (v5, v6, v7, v8, v9)
- yarn (classic, berry)
- npm (v1, v2, v3) - if needed beyond command-based

### Phase 6: Add Missing Testdata (~300 lines)

| Directory | Version | Files |
|-----------|---------|-------|
| `pkg/testdata/npm_v3/` | npm v3 | package.json, package-lock.json |
| `pkg/testdata/pnpm_v7/` | pnpm v7 | package.json, pnpm-lock.yaml |
| `pkg/testdata/pnpm_v8/` | pnpm v8 | package.json, pnpm-lock.yaml |
| `pkg/testdata/pnpm_v9/` | pnpm v9 | (rename existing pnpm/) |

### Phase 7: Add Integration Tests (~200 lines)

**File:** `pkg/lock/integration_test.go`

```go
// Test version-specific pattern selection
func TestIntegration_NPM_LockfileV3(t *testing.T)
func TestIntegration_PNPM_LockfileV7(t *testing.T)
func TestIntegration_PNPM_LockfileV8(t *testing.T)
func TestIntegration_PNPM_LockfileV9(t *testing.T)

// Test pattern detection logic
func TestIntegration_PatternDetection_PNPM(t *testing.T)
func TestIntegration_PatternDetection_Yarn(t *testing.T)
```

### Phase 8: Add Unit Tests (~150 lines)

**File:** `pkg/utils/patterns_test.go` (NEW)

```go
func TestSelectExtractionPatterns_SinglePattern(t *testing.T)
func TestSelectExtractionPatterns_WithDetect_FirstMatch(t *testing.T)
func TestSelectExtractionPatterns_WithoutDetect_AllMatch(t *testing.T)
func TestSelectExtractionPatterns_Mixed(t *testing.T)
func TestSelectExtractionPatterns_NoMatch_Fallback(t *testing.T)
func TestMatchesDetect(t *testing.T)
```

---

## Part 6: Lock File Version Detection

### Detection Regex Patterns

| Package Manager | Version | Detection Regex | Example in File |
|-----------------|---------|-----------------|-----------------|
| **npm v1** | 1 | `"lockfileVersion":\s*1[,\s}]` | `"lockfileVersion": 1` |
| **npm v2** | 2 | `"lockfileVersion":\s*2[,\s}]` | `"lockfileVersion": 2` |
| **npm v3** | 3 | `"lockfileVersion":\s*3[,\s}]` | `"lockfileVersion": 3` |
| **pnpm v5** | 5 | `lockfileVersion:\s*5` | `lockfileVersion: 5.x` |
| **pnpm v6** | 6 | `lockfileVersion:\s*'6` | `lockfileVersion: '6.0'` |
| **pnpm v7** | 7 | `lockfileVersion:\s*'7` | `lockfileVersion: '7.0'` |
| **pnpm v8** | 8 | `lockfileVersion:\s*'8` | `lockfileVersion: '8.0'` |
| **pnpm v9** | 9 | `lockfileVersion:\s*'9` | `lockfileVersion: '9.0'` |
| **yarn classic** | v1 | `#\s*yarn lockfile v1` | `# yarn lockfile v1` |
| **yarn berry** | v2+ | `__metadata:\s*\n\s+version:` | `__metadata:\n  version: 8` |

---

## Part 7: Backwards Compatibility

### Existing Configs Still Work

```yaml
# OLD: Single pattern (still works)
extraction:
  pattern: '(?m)^...'

# NEW: Multi-pattern with detection
extraction:
  patterns:
    - name: "v9"
      detect: "lockfileVersion:\\s*'9"
      pattern: '...'
```

### Migration Path

1. Existing `pattern` field continues to work as fallback
2. `patterns` array is optional - only add when needed
3. No breaking changes to existing `.goupdate.yml` files

---

## Part 8: Future Extensibility

### Where Else This Can Be Used

1. **Outdated extraction** - Different registry response formats
2. **Exclude version patterns** - Conditional exclusions per package
3. **Manifest parsing** - Different file format variations
4. **Version regex** - CalVer vs SemVer detection

### Example: Conditional Exclude Patterns

```yaml
rules:
  npm:
    outdated:
      exclude_version_patterns:
        # Always exclude alpha/beta
        - pattern: "(?i)[._-]alpha"
        - pattern: "(?i)[._-]beta"

        # Only exclude 'next' for @types packages
        - name: "types_next"
          detect: "^@types/"
          pattern: "(?i)[._-]next"
```

---

## Part 9: Summary

### Files to Create/Modify

| File | Action | Lines |
|------|--------|-------|
| `pkg/config/model.go` | Add PatternCfg, update ExtractionCfg | ~50 |
| `pkg/utils/patterns.go` | NEW - Pattern selection logic | ~80 |
| `pkg/utils/patterns_test.go` | NEW - Unit tests | ~150 |
| `pkg/lock/resolve.go` | Update to use pattern selection | ~40 |
| `pkg/config/default.yml` | Add multi-pattern configs | ~100 |
| `pkg/testdata/npm_v3/*` | NEW - npm v3 testdata | ~50 |
| `pkg/testdata/pnpm_v7/*` | NEW - pnpm v7 testdata | ~50 |
| `pkg/testdata/pnpm_v8/*` | NEW - pnpm v8 testdata | ~50 |
| `pkg/lock/integration_test.go` | Add version-specific tests | ~200 |
| **Total** | | **~770 lines** |

### Key Design Decisions

1. **Reusable `PatternCfg` struct** - Can be used across all config areas
2. **Detect field behavior:**
   - NOT set → Pattern ALWAYS applies (additive)
   - IS set → Pattern only applies if detect matches (exclusive)
3. **First detect match wins** - Prevents duplicate results
4. **Fallback to single pattern** - Backwards compatible

---

## Awaiting Approval

Please confirm:

1. **Is the detect behavior correct?**
   - Without detect = always apply (additive)
   - With detect = only if matches (first match wins)

2. **Testdata scope:**
   - npm v3
   - pnpm v6, v7, v8, v9 (skip deprecated v5?)

3. **Should exclude_version_patterns also support PatternCfg?**
   - Would enable conditional exclusions per package

4. **Any additional requirements?**
