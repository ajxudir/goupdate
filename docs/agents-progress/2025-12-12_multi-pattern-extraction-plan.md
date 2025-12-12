# Plan: Multi-Pattern Extraction with Version Detection

**Date:** 2025-12-12
**Status:** Awaiting Approval
**Branch:** claude/organize-mock-data-01Y1vCHWXSvHwCvA6nU99JcH

---

## Overview

Implement multi-pattern extraction support for lock files, allowing multiple regex patterns with version detection to handle different lock file format versions cleanly.

---

## Part 1: Lock File Version Detection Fields

### Version Detection Patterns by Format

| Package Manager | Lock File | Version Field | Detection Pattern |
|-----------------|-----------|---------------|-------------------|
| **npm v1** | package-lock.json | `"lockfileVersion": 1` | `"lockfileVersion":\s*1[,\s}]` |
| **npm v2** | package-lock.json | `"lockfileVersion": 2` | `"lockfileVersion":\s*2[,\s}]` |
| **npm v3** | package-lock.json | `"lockfileVersion": 3` | `"lockfileVersion":\s*3[,\s}]` |
| **pnpm v5** | pnpm-lock.yaml | `lockfileVersion: 5.x` | `lockfileVersion:\s*'?5` |
| **pnpm v6** | pnpm-lock.yaml | `lockfileVersion: '6.0'` | `lockfileVersion:\s*'6` |
| **pnpm v7** | pnpm-lock.yaml | `lockfileVersion: '7.0'` | `lockfileVersion:\s*'7` |
| **pnpm v8** | pnpm-lock.yaml | `lockfileVersion: '8.0'` | `lockfileVersion:\s*'8` |
| **pnpm v9** | pnpm-lock.yaml | `lockfileVersion: '9.0'` | `lockfileVersion:\s*'9` |
| **yarn classic** | yarn.lock | `# yarn lockfile v1` | `#\s*yarn lockfile v1` |
| **yarn berry** | yarn.lock | `__metadata:\n  version:` | `__metadata:\s*\n\s+version:` |

---

## Part 2: New Config Schema

### Current Schema (Single Pattern)

```yaml
lock_files:
  - files: ["**/pnpm-lock.yaml"]
    format: raw
    extraction:
      pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:...'
```

### Proposed Schema (Multi-Pattern with Detection)

```yaml
lock_files:
  - files: ["**/pnpm-lock.yaml"]
    format: raw
    extraction:
      # Multiple patterns with version detection
      patterns:
        - name: "v9"
          detect: "lockfileVersion:\\s*'9"    # Regex to detect this format
          pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

        - name: "v6_v7_v8"
          detect: "lockfileVersion:\\s*'[678]"
          pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

        - name: "v5"
          detect: "lockfileVersion:\\s*'?5"
          pattern: '...different pattern for v5...'

      # Fallback pattern if no detection matches (backwards compatible)
      pattern: '...default pattern...'
```

### Schema Definition Changes

**File:** `pkg/config/model.go`

```go
// ExtractionPatternCfg defines a single extraction pattern with optional detection.
type ExtractionPatternCfg struct {
    // Name is a descriptive name for this pattern (for debugging/logging).
    Name string `yaml:"name,omitempty"`

    // Detect is a regex pattern that must match the file content for this pattern to activate.
    // Only one pattern can activate per file. First matching detection wins.
    Detect string `yaml:"detect,omitempty"`

    // Pattern is the extraction regex with named groups (n/name, version).
    Pattern string `yaml:"pattern"`
}

// ExtractionCfg holds configuration for version extraction from files.
type ExtractionCfg struct {
    // Pattern is a single regex pattern (backwards compatible, used as fallback).
    Pattern string `yaml:"pattern,omitempty"`

    // Patterns is an array of patterns with version detection.
    // Only one pattern activates per file based on the Detect field.
    Patterns []ExtractionPatternCfg `yaml:"patterns,omitempty"`

    // ... existing fields (Path, NameAttr, etc.)
}
```

---

## Part 3: Implementation Plan

### Phase 1: Schema & Model Changes (~50 lines)

**Files to modify:**
- `pkg/config/model.go` - Add `ExtractionPatternCfg` struct, update `ExtractionCfg`

**Changes:**
```go
// Add new struct
type ExtractionPatternCfg struct {
    Name    string `yaml:"name,omitempty"`
    Detect  string `yaml:"detect,omitempty"`
    Pattern string `yaml:"pattern"`
}

// Update existing struct
type ExtractionCfg struct {
    Pattern  string                  `yaml:"pattern,omitempty"`
    Patterns []ExtractionPatternCfg  `yaml:"patterns,omitempty"`  // NEW
    // ... rest unchanged
}
```

### Phase 2: Pattern Selection Logic (~80 lines)

**Files to modify:**
- `pkg/lock/resolve.go` - Add pattern selection logic in `extractVersionsFromLock()`

**New function:**
```go
// selectExtractionPattern selects the appropriate pattern based on file content.
//
// It performs the following:
//   - If Patterns is empty, returns the single Pattern field (backwards compatible)
//   - Iterates through Patterns array in order
//   - For each pattern with a Detect field, checks if content matches
//   - Returns first pattern where Detect matches
//   - If no Detect matches, returns first pattern without Detect field
//   - If still no match, returns empty string (will use fallback Pattern)
func selectExtractionPattern(content string, cfg *ExtractionCfg) (string, string) {
    if len(cfg.Patterns) == 0 {
        return cfg.Pattern, ""
    }

    for _, p := range cfg.Patterns {
        if p.Detect == "" {
            continue // Skip patterns without detect for now
        }
        re, err := regexp.Compile(p.Detect)
        if err != nil {
            continue
        }
        if re.MatchString(content) {
            return p.Pattern, p.Name
        }
    }

    // Try first pattern without Detect (default)
    for _, p := range cfg.Patterns {
        if p.Detect == "" {
            return p.Pattern, p.Name
        }
    }

    // Fallback to single pattern field
    return cfg.Pattern, ""
}
```

### Phase 3: Update default.yml with Multi-Pattern Support (~100 lines)

**File to modify:**
- `pkg/config/default.yml`

**Changes for pnpm:**
```yaml
pnpm:
  lock_files:
    - files: ["**/pnpm-lock.yaml"]
      format: raw
      extraction:
        patterns:
          - name: "v9"
            detect: "lockfileVersion:\\s*'9"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          - name: "v6_v7_v8"
            detect: "lockfileVersion:\\s*'[678]"
            pattern: '(?m)^\s{6}''?(?P<n>[@\w\-\.\/]+)''?:\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'

          - name: "v5_fallback"
            # No detect = fallback for older formats
            pattern: '(?m)^\s+(?P<n>[@\w\-\.\/]+):\s+(?P<version>[\d\.]+)'
```

**Changes for yarn:**
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

### Phase 4: Add Missing Testdata (~200 lines of test files)

**New directories to create:**

| Directory | Version | Content |
|-----------|---------|---------|
| `pkg/testdata/npm_v3/` | npm v3 | package.json + package-lock.json (lockfileVersion: 3) |
| `pkg/testdata/pnpm_v7/` | pnpm v7 | package.json + pnpm-lock.yaml (lockfileVersion: '7.0') |
| `pkg/testdata/pnpm_v8/` | pnpm v8 | package.json + pnpm-lock.yaml (lockfileVersion: '8.0') |
| `pkg/testdata/pnpm_v9/` | pnpm v9 | Already exists as `pkg/testdata/pnpm/` - rename for consistency |

**Note:** pnpm v5 is deprecated and auto-converts, so we'll skip it.

### Phase 5: Add Integration Tests (~150 lines)

**File to modify:**
- `pkg/lock/integration_test.go`

**New tests:**
```go
// TestIntegration_NPM_LockfileV3 tests npm lockfileVersion 3 (npm 9+ format).
func TestIntegration_NPM_LockfileV3(t *testing.T) { ... }

// TestIntegration_PNPM_LockfileV7 tests pnpm lockfileVersion 7.0.
func TestIntegration_PNPM_LockfileV7(t *testing.T) { ... }

// TestIntegration_PNPM_LockfileV8 tests pnpm lockfileVersion 8.0.
func TestIntegration_PNPM_LockfileV8(t *testing.T) { ... }

// TestIntegration_PNPM_LockfileV9 tests pnpm lockfileVersion 9.0.
func TestIntegration_PNPM_LockfileV9(t *testing.T) { ... }

// TestIntegration_PatternDetection tests that correct pattern is selected.
func TestIntegration_PatternDetection(t *testing.T) { ... }
```

### Phase 6: Unit Tests for Pattern Selection (~100 lines)

**New file:**
- `pkg/lock/pattern_selection_test.go`

**Tests:**
```go
func TestSelectExtractionPattern_SinglePattern(t *testing.T) { ... }
func TestSelectExtractionPattern_MultiPattern_FirstMatch(t *testing.T) { ... }
func TestSelectExtractionPattern_MultiPattern_SecondMatch(t *testing.T) { ... }
func TestSelectExtractionPattern_MultiPattern_NoMatch_Fallback(t *testing.T) { ... }
func TestSelectExtractionPattern_EmptyPatterns_UsesSinglePattern(t *testing.T) { ... }
```

---

## Part 4: Testdata Structure

### npm v3 Testdata

**File:** `pkg/testdata/npm_v3/package-lock.json`
```json
{
  "name": "test-npm-v3-lockfile",
  "version": "1.0.0",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
    "": {
      "name": "test-npm-v3-lockfile",
      "version": "1.0.0",
      "dependencies": {
        "lodash": "^4.17.21",
        "express": "~4.18.2"
      }
    },
    "node_modules/lodash": {
      "version": "4.17.21"
    },
    "node_modules/express": {
      "version": "4.18.3"
    }
  }
}
```

### pnpm v7/v8/v9 Testdata

All three versions use the same `importers` structure, just different `lockfileVersion` values:

**File:** `pkg/testdata/pnpm_v7/pnpm-lock.yaml`
```yaml
lockfileVersion: '7.0'

settings:
  autoInstallPeers: true

importers:
  .:
    dependencies:
      lodash:
        specifier: ^4.17.21
        version: 4.17.21
      express:
        specifier: ~4.18.2
        version: 4.18.3
```

---

## Part 5: Benefits of This Approach

### 1. Version-Specific Pattern Selection
- Only one pattern activates per file
- No duplicate results
- Guaranteed correct regex for file version

### 2. Self-Documenting
- Each pattern has a name
- Detection regex is explicit
- Easy to understand which pattern applies

### 3. Backwards Compatible
- Single `pattern` field still works
- `patterns` array is optional
- Existing configs don't break

### 4. Easy to Extend
- Adding new version = add new pattern entry
- No code changes needed for new versions
- Users can add custom patterns in their `.goupdate.yml`

### 5. Better Debugging
- Logs can show which pattern was selected
- Easier to troubleshoot extraction issues

---

## Part 6: Implementation Order

| Step | Description | Files | Lines | Time Est. |
|------|-------------|-------|-------|-----------|
| 1 | Add `ExtractionPatternCfg` struct | model.go | ~20 | 10 min |
| 2 | Implement `selectExtractionPattern()` | resolve.go | ~50 | 20 min |
| 3 | Update `extractVersionsFromLock()` to use selection | resolve.go | ~20 | 10 min |
| 4 | Create npm_v3 testdata | testdata/npm_v3/* | ~50 | 15 min |
| 5 | Create pnpm_v7, v8, v9 testdata | testdata/pnpm_v*/* | ~100 | 30 min |
| 6 | Update default.yml with multi-pattern | default.yml | ~60 | 20 min |
| 7 | Add integration tests | integration_test.go | ~120 | 30 min |
| 8 | Add unit tests for pattern selection | pattern_selection_test.go | ~80 | 20 min |
| 9 | Run full test suite | - | - | 10 min |
| 10 | Update progress report | docs/ | ~50 | 10 min |

**Total Estimated: ~550 lines, ~3 hours**

---

## Part 7: Validation Criteria

- [ ] All existing integration tests still pass (backwards compatible)
- [ ] New `TestIntegration_NPM_LockfileV3` passes
- [ ] New `TestIntegration_PNPM_LockfileV7` passes
- [ ] New `TestIntegration_PNPM_LockfileV8` passes
- [ ] New `TestIntegration_PNPM_LockfileV9` passes
- [ ] `TestIntegration_PatternDetection` verifies correct pattern selection
- [ ] Unit tests for `selectExtractionPattern` all pass
- [ ] Race detector clean: `go test -race ./pkg/lock/...`
- [ ] Coverage maintained ≥95% for pkg/lock

---

## Part 8: Example Usage in User Config

Users can override patterns in their `.goupdate.yml`:

```yaml
extends: [default]

rules:
  pnpm:
    lock_files:
      - files: ["**/pnpm-lock.yaml"]
        format: raw
        extraction:
          patterns:
            # Custom pattern for their specific pnpm setup
            - name: "custom_monorepo"
              detect: "lockfileVersion:\\s*'9"
              pattern: '(?m)^\s{4}''(?P<n>[@\w\-\.\/]+)'':\s*\n\s+specifier:[^\n]+\n\s+version:\s*(?P<version>[\d\.]+)'
```

---

## Awaiting Approval

Please review this plan and confirm:

1. **Schema design** - Is the `patterns` array with `name`, `detect`, `pattern` fields acceptable?
2. **Detection approach** - Is "first matching detection wins" the right logic?
3. **Testdata scope** - npm v3 + pnpm v7/v8/v9 (skip deprecated v5)?
4. **Any additional requirements?**
