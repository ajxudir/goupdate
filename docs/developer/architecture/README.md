# GoUpdate Architecture Documentation

> This documentation provides a deep-dive into the architecture, data flows, and implementation details of each command and subsystem. Use this as a reference when implementing new features, debugging issues, or understanding how existing functionality works.

## Table of Contents

- [Quick Reference](#quick-reference)
- [Architecture Overview](#architecture-overview)
- [Documentation Files](#documentation-files)
- [Data Flow: Complete Update Cycle](#data-flow-complete-update-cycle)
- [Key Concepts](#key-concepts)
- [Testing Guidelines](#testing-guidelines)
- [Adding New Features](#adding-new-features)

---

## Quick Reference

| Command | Purpose | Key Files |
|---------|---------|-----------|
| `scan` | Detect package files matching rules | `cmd/scan.go`, `pkg/packages/detect.go` |
| `list` | Show declared vs installed versions | `cmd/list.go`, `pkg/lock/resolve.go` |
| `outdated` | Find packages with newer versions | `cmd/outdated.go`, `pkg/outdated/core.go` |
| `update` | Apply version updates with rollback | `cmd/update.go`, `pkg/update/core.go` |
| `config` | Manage configuration | `cmd/config.go`, `pkg/config/load.go` |
| `version` | Print version and build info | `cmd/version.go` |
| `help` | Show command help | Built into Cobra |

## Architecture Overview

```
CLI Commands
═══════════════════════════════════════════════════════════════════════
   scan         list       outdated      update       config
     │            │            │            │            │
     ▼            ▼            ▼            ▼            ▼

Core Packages
═══════════════════════════════════════════════════════════════════════
 packages       lock       outdated      update       config
 (detect)    (resolve)     (fetch)      (apply)      (load)
     │            │            │            │            │
     ▼            ▼            ▼            ▼            ▼

Supporting Packages
═══════════════════════════════════════════════════════════════════════
 formats       utils       cmdexec     preflight    warnings
 (parsers)   (helpers)   (commands)   (validate)   (logging)
```

## Documentation Files

### Commands
- **[scan.md](./scan.md)** - File detection and pattern matching
- **[list.md](./list.md)** - Package listing with lock file resolution
- **[outdated.md](./outdated.md)** - Version checking and comparison
- **[update.md](./update.md)** - Update execution with validation and rollback
- **[config.md](./config.md)** - Configuration management
- **[utility-commands.md](./utility-commands.md)** - Version, help, and config validation

### Core Systems
- **[configuration.md](./configuration.md)** - Config loading, merging, and defaults
- **[lock-resolution.md](./lock-resolution.md)** - Lock file parsing and version extraction
- **[version-comparison.md](./version-comparison.md)** - Version parsing and filtering
- **[format-parsers.md](./format-parsers.md)** - JSON, YAML, XML, Raw format handling
- **[command-execution.md](./command-execution.md)** - Shell command execution with placeholders

### Special Features
- **[floating-constraints.md](./floating-constraints.md)** - Handling wildcards and ranges
- **[groups.md](./groups.md)** - Package grouping for atomic updates
- **[incremental-updates.md](./incremental-updates.md)** - One-step-at-a-time upgrades
- **[self-pinning.md](./self-pinning.md)** - Manifests that act as lock files
- **[system-tests.md](./system-tests.md)** - Automated validation during updates

## Data Flow: Complete Update Cycle

```
1. CONFIGURATION
   config.LoadConfig() → Merge defaults + user config + extends

2. DETECTION
   packages.DetectFiles() → Find files matching include/exclude patterns

3. PARSING
   packages.DynamicParser.ParseFile() → Extract packages from manifest files

4. LOCK RESOLUTION
   lock.ApplyInstalledVersions() → Match packages to lock file entries

5. VERSION CHECKING
   outdated.ListNewerVersions() → Fetch available versions from registry

6. FILTERING
   outdated.FilterVersionsByConstraint() → Apply constraint rules (^, ~, =)

7. SELECTION
   outdated.SelectTargetVersion() → Pick target based on flags/constraints

8. UPDATE
   update.UpdatePackage() → Modify manifest + run lock command

9. VALIDATION
   validateUpdatedPackage() → Verify changes were applied correctly

10. ROLLBACK (on failure)
    rollbackPlans() → Restore original versions if validation fails
```

## Key Concepts

### Install Status Values

| Status | Icon | Meaning |
|--------|------|---------|
| `LockFound` | 🟢 | Package found in lock file with version |
| `SelfPinned` | 📌 | Manifest is its own lock (e.g., requirements.txt) |
| `NotInLock` | 🔵 | Package not found in lock file |
| `LockMissing` | 🟠 | Lock file doesn't exist |
| `VersionMissing` | ⛔ | No concrete version available |
| `NotConfigured` | ⚪ | No lock file config for this rule |
| `Floating` | ⛔ | Floating constraint (5.*, ranges) cannot auto-update |

> **Note:** ⛔ indicates the package cannot be processed for updates. ⚪ indicates missing configuration.

### Constraint Types

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

### Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | All operations completed |
| 1 | Partial | Some succeeded, some failed (with --continue-on-fail) |
| 2 | Failure | Complete failure or critical error |
| 3 | Config Error | Invalid configuration or validation failure |

## Testing Guidelines

Each command has corresponding test files:
- `cmd/*_test.go` - Command-level integration tests
- `pkg/*/..._test.go` - Package-level unit tests

Key testing patterns:
- Function variables (`loadConfigFunc`, `execCommandFunc`) allow mocking
- `t.TempDir()` for isolated file system tests
- `captureStdout()` for output verification

### Test Data Structure

| Directory | Purpose |
|-----------|---------|
| `pkg/testdata/` | Valid test fixtures for automated testing (no errors) |
| `pkg/_testdata/` | Error test cases for manual error handling validation |

**testdata/** - Contains sample manifests and lock files for all supported package managers:
- `npm/`, `composer/`, `mod/`, `msbuild/`, `nuget/`, `pipfile/`, `requirements/`
- Each has edge-case subfolders like `_edge-cases/no-lock/` for LockMissing status

**_testdata/** - Contains intentionally broken test cases:
- `invalid-command/` - Non-existent commands for error handling
- `malformed-json/` - Invalid JSON for parse error testing
- `malformed-xml/` - Invalid XML for parse error testing
- `command-timeout/` - Commands that exceed timeout

## Adding New Features

1. **Update documentation first** - Add to relevant architecture doc
2. **Add tests** - Write failing tests before implementation
3. **Implement feature** - Follow existing patterns
4. **Update this index** - Keep documentation in sync

## Documentation Style

### Diagrams

When creating ASCII diagrams in documentation:

- **Do not use borders** around diagrams (they often render misaligned)
- Use a title with `═══` separator instead of boxes
- Keep flow arrows simple: `──►`, `│`, `▼`

**Good:**
```
FLOW TITLE
═══════════════════════════════════════════════════════════════════

Step A ──► Step B ──► Step C
               │
               ▼
           Step D
```

**Avoid:**
```
┌─────────────────────────────────────────────────────────────────┐
│                         FLOW TITLE                               │
├─────────────────────────────────────────────────────────────────┤
│   Step A ──► Step B ──► Step C                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

*Last updated: 2025-12-01*
