# Verbose Logging Guidelines

This document describes how verbose logging is currently implemented, known issues, and guidelines for future development to ensure debug output remains useful without being overwhelming.

## Table of Contents

1. [Current Implementation](#current-implementation)
2. [Identified Issues](#identified-issues)
3. [Recommended Verbosity Levels](#recommended-verbosity-levels)
4. [Logging Guidelines](#logging-guidelines)
5. [Phase-Specific Guidelines](#phase-specific-guidelines)
6. [Priority Fixes](#priority-fixes)
7. [Verification Checklist](#verification-checklist)

---

## Current Implementation

### Verbose Package (`pkg/verbose`)

The verbose logging system uses a global `enabled` flag controlled by CLI flags:

```go
verbose.Enable()      // Turn on verbose output
verbose.Printf(...)   // Only prints if enabled
verbose.Infof(...)    // Informational messages
verbose.Suppress()    // Temporarily suppress output (used during drift checks)
verbose.Unsuppress()  // Restore output
```

### Current Usage

Verbose logging is used throughout the codebase for:
- Configuration loading and validation
- File detection and parsing
- Lock file resolution
- Version checking (outdated)
- Update execution
- System tests
- Drift detection

---

## Identified Issues

### Summary by Command

| Command | Current DEBUG Lines | Target | Reduction Needed |
|---------|---------------------|--------|------------------|
| `scan` | ~80 | ~15-20 | ~75% |
| `list` | ~150 | ~30-40 | ~75% |
| `update` | ~1,200+ | ~150-200 | ~85% |

### Issue Categories

#### 1. Duplicate Log Statements

The same information is logged multiple times:

| Issue | Example | Fix |
|-------|---------|-----|
| Planning update logged twice | `Planning update for axios: incremental=false...` appears 2× | Remove duplicate call |
| Version candidates logged twice | `Version candidates: major=#N/A, minor=1.13.2...` appears 2× | Remove duplicate call |
| Version summarization errors duplicated | Error logged 2× per affected package | Log only once |
| Floating check repeated | Same `"*" is pure wildcard` logged 3-4× for same package | Log once per package |

#### 2. Multi-Line Sequences That Should Be Consolidated

Current patterns that span 3-6 lines should be single lines:

**Conflict Resolution (5 lines → 1 line):**
```
# Current
[DEBUG] Conflict: file "package.json" matched by multiple rules: [npm yarn pnpm]
[DEBUG] Prioritized rule order for "package.json": [npm pnpm yarn]
[DEBUG] Rule "npm" selected: lock file found in .
[DEBUG] Conflict resolved: selected rule "npm" for "package.json"
[DEBUG] Resolved 1 file conflicts

# Target
[DEBUG] Conflict: package.json matched [npm, yarn, pnpm] → selected npm (lock file found)
```

**Pattern Extraction (6 lines → 1 line):**
```
# Current
[DEBUG] Lock extraction: applying extraction pattern(s) to composer.lock
[DEBUG] Pattern extraction: applying 1 pattern(s)
[DEBUG] Pattern extraction: applying pattern 1/1
[DEBUG] Pattern extraction: pattern 1 matched 123 entries
[DEBUG] Pattern extraction: total 123 matches from all patterns
[DEBUG] Lock extraction: pattern matched 123 entries

# Target
[DEBUG] Lock extraction: composer.lock → 123 packages extracted
```

**Drift Checks (3 lines → 1 line):**
```
# Current
[DEBUG] Pre-update drift check: verifying axios is still at 1.7.9
[DEBUG] Pre-update drift check: axios - current=1.7.9, expected=1.7.9
[DEBUG] Pre-update drift check PASSED: axios at expected version 1.7.9

# Target
[DEBUG] Drift check: axios at expected version 1.7.9 ✓
```

**System Tests (4 lines → 1 line):**
```
# Current
[DEBUG] Running system test "composer":
composer install --no-interaction --prefer-dist --no-scripts

[DEBUG] Executing shell command: /bin/zsh -l -c "composer install..."
[DEBUG] Working directory: .

# Target
[DEBUG] Running system test "composer": composer install --no-interaction --prefer-dist --no-scripts
```

#### 3. Redundant Per-Package Lists

Full lists are logged when only summaries are needed:

| Pattern | Lines | Fix |
|---------|-------|-----|
| Every parsed package listed | 35+ lines | Summary: "Parsed 35 packages from composer.json" |
| Every installed version logged | 35+ lines | Already have summary count |
| Full baseline listing | 35 lines | Summary: "Baseline captured for 35 packages" |
| Package processing order | 35 lines | Only list packages being updated |

#### 4. Low-Value Metadata

Remove these as they don't aid debugging:

| Pattern | Example | Action |
|---------|---------|--------|
| File size | `[DEBUG] File size: 578 bytes` | Remove |
| Format metadata | `[DEBUG] Format: json, Fields config: name="", version=""` | Remove |
| Output length | `[DEBUG] Lock file command output length: 2102 bytes` | Remove |
| Default working directory | `[DEBUG] Working directory: .` | Omit when `.` |

#### 5. Repeated Per-Item Logs

Log once instead of per-item:

| Pattern | Current | Target |
|---------|---------|--------|
| Command found in PATH | Logged 35× for same command | Log once per unique command |
| Using outdated config | Logged 35× | Only if override/unexpected |
| Preflight per package | 35 lines | Log unique commands only |

---

## Recommended Verbosity Levels

### Proposed Multi-Level System

| Level | Flag | Content |
|-------|------|---------|
| **Quiet** | `-q` | Errors only |
| **Normal** | (none) | Summary table, warnings, final results |
| **Verbose** | `-v` / `--verbose` | + Key decisions, update actions, test results, conflict resolutions |
| **Debug** | `-vv` | + Shell commands, drift checks, version selection, per-package lock resolution |
| **Trace** | `-vvv` | + Full version lists, all parsed packages, all tags, pattern details |

### What Goes Where

#### Normal (no flag)
- Final summary table
- Warnings (floating constraints, unsupported packages)
- Error messages
- Progress indicators

#### Verbose (`-v`)
- Config loaded: `Config loaded from .goupdate.yml (extends: default)`
- Rule summary: `Extended from "default": 9 rules configured`
- File detection: `Detected 2 package files: composer.json, package.json`
- Conflict resolution (single line): `Conflict: package.json → selected npm`
- Update decisions: `axios: 1.7.9 → 1.13.2 (minor update)`
- System test results: `System test "composer" passed (9.1s)`
- Group updates: `Updating group "vite": 3 packages atomically`

#### Debug (`-vv`)
- Shell commands being executed
- Drift check results
- Version selection logic: `Selected 1.13.2 (minor scope, 5 candidates)`
- Lock resolution per-file: `composer.lock → 136 packages extracted`
- Per-package lock resolution (installed versions)
- Preflight validation per unique command

#### Trace (`-vvv`)
- Full version lists: `All tags for axios: [1.13.2, 1.13.1, ...]`
- All parsed packages listed
- Pattern matching details
- Exclusion reasons per version
- Full baseline per package

---

## Logging Guidelines

### General Principles

1. **Summary over enumeration**: Log counts, not lists (at verbose level)
2. **One line per concept**: Consolidate related logs into single statements
3. **Log decisions, not steps**: Focus on outcomes, not intermediate steps
4. **Fail loudly, succeed quietly**: More detail on failures, less on success
5. **Progressive disclosure**: More detail at higher verbosity levels

### Formatting Standards

```go
// Good - concise, informative
verbose.Printf("Lock extraction: %s → %d packages\n", lockFile, count)

// Bad - spread across multiple calls
verbose.Printf("Lock extraction: reading %s\n", lockFile)
verbose.Printf("Lock extraction: applying patterns\n")
verbose.Printf("Lock extraction: found %d\n", count)

// Good - single-line summary
verbose.Printf("Rule conflicts resolved: %d files → npm:1, composer:1\n", conflicts)

// Bad - per-conflict logging
for _, c := range conflicts {
    verbose.Printf("Conflict: %s matched by %v\n", c.File, c.Rules)
    verbose.Printf("Selected: %s\n", c.Selected)
}

// Good - conditional detail
if verbose.IsDebug() {
    for _, pkg := range packages {
        verbose.Printf("  %s @ %s\n", pkg.Name, pkg.Version)
    }
} else {
    verbose.Printf("Parsed %d packages from %s\n", len(packages), file)
}
```

### Anti-Patterns to Avoid

```go
// DON'T: Log default values
verbose.Printf("Working directory: .\n")  // Omit when default

// DON'T: Log same info multiple ways
verbose.Printf("Found 35 packages\n")
for _, p := range packages {
    verbose.Printf("  - %s\n", p.Name)  // This duplicates the count
}

// DON'T: Log intermediate steps
verbose.Printf("Starting validation\n")
verbose.Printf("Validating step 1\n")
verbose.Printf("Validating step 2\n")
verbose.Printf("Validation complete\n")  // Just log result

// DON'T: Repeat same log per iteration
for _, pkg := range packages {
    verbose.Printf("Checking command: npm\n")  // Same command each time
}
```

---

## Phase-Specific Guidelines

### Configuration Phase

**Current issues:** 16+ lines for validation, 9 lines for rule setup

**Target output:**
```
[DEBUG] Config loaded: .goupdate.yml (extends: default, 9 rules)
[DEBUG] Config validation passed (2 custom rules, 5 groups)
```

**Implementation notes:**
- Collect rule additions/merges, log summary
- Only log validation details on failure
- Group validation: single line with counts

### File Detection Phase

**Current issues:** 2 lines per rule (even no-matches), 5 lines per conflict

**Target output:**
```
[DEBUG] File detection: 4/9 rules matched files
[DEBUG] Conflict: package.json matched [npm, yarn, pnpm] → selected npm (lock file)
[DEBUG] Detected 2 package files: composer.json, package.json
```

**Implementation notes:**
- Count no-match rules, log summary
- Single-line conflict resolution
- Omit individual file listings (available at trace level)

### File Parsing Phase

**Current issues:** 35+ lines listing every package

**Target output:**
```
[DEBUG] Parsed composer.json: 22 packages (15 prod, 7 dev)
[DEBUG] Parsed package.json: 13 packages (9 prod, 4 dev)
```

**Implementation notes:**
- Remove file size and format metadata
- Package list only at trace level
- Include prod/dev breakdown if useful

### Lock Resolution Phase

**Current issues:** 6 lines per lock file, 35 lines for installed versions

**Target output:**
```
[DEBUG] Lock resolution: composer.lock → 136 packages
[DEBUG] Lock resolution: package-lock.json → 13 packages (via npm ls)
```

**Implementation notes:**
- Single line per lock file with extraction method
- Per-package installed versions only at debug level
- Remove output length logging

### Version Checking Phase (update)

**Current issues:** Full tag lists (hundreds per package), duplicate planning logs

**Target output:**
```
[DEBUG] axios: 47 versions available, 5 newer, target: 1.13.2 (minor)
[DEBUG] laravel/framework: 1232 versions → excluded 43 → target: none (major only)
```

**Implementation notes:**
- Full tag list only at trace level
- Consolidate planning + candidates + selection into one line
- Remove duplicate logging calls
- "Using outdated config" only when overridden

### Update Execution Phase

**Current issues:** 3 lines per drift check, command template shown with replacements

**Target output:**
```
[DEBUG] Drift check: axios at 1.7.9 ✓
[DEBUG] Executing: npm install axios@1.13.2
[DEBUG] Updated axios: 1.7.9 → 1.13.2
```

**Implementation notes:**
- Single-line drift checks
- Show final command only (not template + replacements)
- Consolidate lock complete + updated messages

### System Tests Phase

**Current issues:** Multi-line command display, repeated for pre/post

**Target output:**
```
[DEBUG] System test (preflight) "composer": passed (9.1s)
[DEBUG] System test (after axios) "npm-build": passed (14.0s)
```

**Implementation notes:**
- Inline command on single line if short
- Only show full command at debug level
- Working directory only if non-default

---

## Priority Fixes

These changes provide the highest noise reduction with minimal code changes:

### Priority 1: Remove Duplicates (Quick Wins)

1. `pkg/outdated/core.go`: Remove duplicate "Planning update" log
2. `pkg/outdated/core.go`: Remove duplicate "Version candidates" log
3. `pkg/outdated/core.go`: Remove duplicate version summarization error

### Priority 2: Consolidate Multi-Line Sequences

1. Conflict resolution in `pkg/packages/detection.go`
2. Pattern extraction in `pkg/lock/extraction.go`
3. Drift checks in `pkg/update/execution.go`
4. System test output in `pkg/systemtest/runner.go`

### Priority 3: Summary Over Enumeration

1. Config validation: summary only, details on failure
2. Rule setup: count rules added/merged
3. Parsed packages: count only, list at trace
4. Installed versions: already have lock extraction count

### Priority 4: Remove Low-Value Logs

1. Remove file size logging in parsers
2. Remove format metadata logging
3. Remove output length from lock commands
4. Omit "Working directory: ." when default

### Priority 5: Log Once Per Unique Item

1. Preflight: log each unique command once
2. Floating check: log once per package
3. "Using outdated config": only when overridden

---

## Verification Checklist

After implementing verbose logging changes, verify:

### Functional Verification

- [ ] `goupdate scan -d testdata/laravel-app --verbose` shows < 20 DEBUG lines
- [ ] `goupdate list -d testdata/laravel-app --verbose` shows < 40 DEBUG lines
- [ ] `goupdate update -d /path/to/project --verbose --dry-run` shows < 200 DEBUG lines
- [ ] All key decisions are still visible in verbose output
- [ ] Error conditions show full detail

### Content Verification

- [ ] No duplicate log lines for same event
- [ ] Multi-line sequences consolidated to single lines
- [ ] Package lists show counts, not full enumerations
- [ ] "Working directory: ." not logged
- [ ] File size/format metadata not logged
- [ ] Conflict resolution is single line
- [ ] Drift checks are single line
- [ ] System tests show inline commands

### Regression Checks

- [ ] Failed operations show sufficient detail to debug
- [ ] `--verbose` still provides useful information for troubleshooting
- [ ] No information loss that would prevent issue diagnosis
- [ ] Timing information preserved where relevant

### Test Commands

```bash
# Count verbose lines for scan
goupdate scan -d testdata/laravel-app --verbose 2>&1 | grep -c "^\[DEBUG\]"

# Count verbose lines for list
goupdate list -d testdata/laravel-app --verbose 2>&1 | grep -c "^\[DEBUG\]"

# Count verbose lines for update dry-run
goupdate update -d /tmp/test-project --verbose --dry-run -y 2>&1 | grep -c "^\[DEBUG\]"

# Verify no duplicates (should return 0)
goupdate update -d /tmp/test-project --verbose --dry-run -y 2>&1 | sort | uniq -d | wc -l
```

---

## Future Considerations

### Multi-Level Verbosity

Consider implementing `-v`, `-vv`, `-vvv` levels:

```go
type VerboseLevel int

const (
    LevelQuiet VerboseLevel = iota
    LevelNormal
    LevelVerbose  // -v
    LevelDebug    // -vv
    LevelTrace    // -vvv
)

func AtLevel(level VerboseLevel) bool {
    return currentLevel >= level
}

// Usage
if verbose.AtLevel(LevelTrace) {
    verbose.Printf("All tags: %v\n", allTags)
} else if verbose.AtLevel(LevelDebug) {
    verbose.Printf("Found %d tags, %d after exclusions\n", len(allTags), len(filtered))
}
```

### Structured Logging

For future tooling integration, consider JSON-structured debug output:

```go
verbose.Structured(map[string]interface{}{
    "phase": "lock_resolution",
    "file": "composer.lock",
    "packages": 136,
    "duration_ms": 45,
})
```

### Log Categories

Group logs by category for filtering:

```go
verbose.Category("config").Printf("Loaded %s\n", configFile)
verbose.Category("detection").Printf("Found %d files\n", count)
verbose.Category("update").Printf("Updated %s\n", pkg)

// CLI: --verbose=config,update
```
