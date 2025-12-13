# **AGENTS.md — Agentic Coding Guidelines**

> **Note for Claude Code users**: Claude Code reads `CLAUDE.md` at startup, not this file.
> See `/CLAUDE.md` for quick reference instructions that Claude loads automatically.
> This file contains detailed procedures for all agentic coding agents.

These rules apply to **all agentic coding agents** (Claude, Codex, Cursor, Copilot, etc.).
If unsure, agents must **ask** instead of guessing.

## Task-Specific Checklists

Select the appropriate checklist based on task type:

| Task Type | Checklist |
|-----------|-----------|
| Adding new feature | `docs/checklists/feature-development.md` |
| Fixing a bug | `docs/checklists/bug-fix.md` |
| Refactoring code | `docs/checklists/refactoring.md` |
| Improving tests | `docs/checklists/test-improvement.md` |
| Battle testing CLI | `docs/testing-checklist.md` |

See `docs/checklists/README.md` for selection guide.

## Agent Progress Tracking

All agents working on tasks must log their progress in `docs/agents-progress/`.

### Log File Naming
```
docs/agents-progress/
├── YYYY-MM-DD_task-name.md    # Individual task logs
├── YYYY-MM-DD_migration-xyz.md
└── ...
```

### Log Format
```markdown
# Task: [Task Name]
**Agent:** Claude/Codex/Other
**Date:** YYYY-MM-DD
**Status:** In Progress / Completed / Blocked

## Objective
Brief description of what was requested.

## Progress
- [x] Step completed
- [ ] Step pending

## Files Modified
- path/to/file.go

## Notes
Any observations or issues encountered.
```

## Table of Contents

- [0. TASK-FIRST WORKFLOW](#0-task-first-workflow)
- [1. NO DESTRUCTIVE OPERATIONS](#1-no-destructive-operations)
- [2. TESTS + COVERAGE](#2-tests--coverage)
- [3. REAL TESTDATA ONLY](#3-real-testdata-only)
- [4. RULE & PM SEPARATION](#4-rule--pm-separation)
- [5. OUTPUT RULES](#5-output-rules-table--errors)
- [6. DOCS MUST MATCH CODE](#6-docs-must-match-code)
- [7. BATTLE TESTING PROCEDURES](#7-battle-testing-procedures)
- [8. CHAOS ENGINEERING](#8-chaos-engineering)
- [9. QUALITY ASSURANCE TASKS](#9-quality-assurance-tasks)
- [10. DOCUMENTATION TASKS](#10-documentation-tasks)
- [11. COMMON AGENT TASKS](#11-common-agent-tasks)
- [12. TESTDATA MANAGEMENT](#12-testdata-management)

---

## **0. TASK-FIRST WORKFLOW**

**CRITICAL: Complete the task first, then add tests/coverage.**

When working on large migrations, refactors, or new features:

1. **Focus on completion** - Get the feature/migration working first
2. **Skip tests during implementation** - Don't waste time on tests for code that may change
3. **Avoid lint/coverage distractions** - These are business requirements for the final code
4. **Test after completion** - Once the code is stable and working, add tests
5. **Final polish** - Run lint, coverage, and verification as the last step

### Why This Matters

Trying to maintain 100% coverage during a large migration:
- Takes 100x longer
- Wastes effort on tests for code that gets refactored
- Breaks focus on the actual implementation
- Results in poor quality tests written just to pass coverage

### Workflow for Large Tasks

```
Phase 1: Implementation
├── Write the code
├── Get it compiling
├── Get it working
└── Manual verification

Phase 2: Testing (AFTER Phase 1 is complete)
├── Write unit tests
├── Write integration tests
├── Achieve coverage targets
└── Run full test suite

Phase 3: Polish
├── Run linters
├── Fix style issues
├── Update documentation
└── Final PR review
```

### Exception

For small bug fixes or simple features, TDD (test-first) is still preferred.

---

## **1. NO DESTRUCTIVE OPERATIONS**

Agents must **never** run or suggest:

* `git reset --hard`
* `git clean`
* `git checkout .`
* `rm -rf` on project files
* `git push --force`

Unless the user explicitly writes:
**ALLOW_DESTRUCTIVE=true** in the same message.

---

## **2. TESTS + COVERAGE**

* All modified code must be fully tested.
* Keep branch coverage at 100% (e.g., sort/prioritization helpers like `prioritizeRules` must be fully exercised).
* No placeholders ("TODO", "implement", empty tests).
* No removing tests unless feature is removed.
* Run and ensure all pass:

  * `make test`
  * `make coverage`
  * `make coverage-func`
  * `make check`
* Coverage must stay **100%**.

---

## **3. REAL TESTDATA ONLY**

* NO fake registries, mock version catalogs, synthetic JSON maps.
* Use **real** package.json, lockfiles, go.mod, composer.json, etc.
* Main `testdata/<rule>` must contain:

  * Many packages
  * Real outdated versions
  * High constraint variety
  * Only a few UpToDate cases
* All invalid/broken/edge-case files must go in:

  * `testdata/_edgecases/`
  * or other `_invalid/` folders
    Never mix edgecases with main testdata.

---

## **4. RULE & PM SEPARATION**

* `PM` = ecosystem (js, dotnet, python, php, golang)
* `RULE` = file format (npm, pnpm, yarn, composer, msbuild, nuget, pipfile, requirements, mod)
* Never merge rules.
* Behavior must be **configuration-driven**, not hardcoded.

---

## **5. OUTPUT RULES (TABLE + ERRORS)**

* Table output must be clean and aligned.
* Errors/warnings:

  * Only printed **before or after** table.
  * Never during table rendering.
  * Must use prefixes:

    * `⚠️` Warning
    * `❌` Error
* STATUS column must always reflect correct state:

  * `🟠 LockMissing`
  * `🔵 NotInLock`
  * `🔴 VersionMissing`
  * `❌ Failed`, etc.

---

## **6. DOCS MUST MATCH CODE**

If behavior changes, agents must update `docs/*.md` in the same PR.
Docs must never drift out of sync.

---

## **7. BATTLE TESTING PROCEDURES**

When asked to "battle test" or validate the CLI, follow these steps:

### Clone real-world projects for testing:
**IMPORTANT**: Use separate temp directories for each test task to enable parallel testing.

```bash
# Create unique temp directories for isolation
TEST_DIR=$(mktemp -d)

# JavaScript ecosystems (run in parallel)
git clone --depth 1 https://github.com/expressjs/express.git $TEST_DIR/express &
git clone --depth 1 https://github.com/nuxt/nuxt.git $TEST_DIR/nuxt &
git clone --depth 1 https://github.com/facebook/react.git $TEST_DIR/react &

# Go projects
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra &

# PHP projects
git clone --depth 1 https://github.com/laravel/laravel.git $TEST_DIR/laravel &

# Python projects
git clone --depth 1 https://github.com/django/django.git $TEST_DIR/django &

wait  # Wait for all clones to complete
```

### Parallel testing strategy:
Run tests in parallel when possible to speed up validation:
```bash
# Test multiple projects in parallel
goupdate scan -d $TEST_DIR/express &
goupdate scan -d $TEST_DIR/cobra &
goupdate scan -d $TEST_DIR/laravel &
wait

# Test different commands in parallel on same project
goupdate list -d $TEST_DIR/express --type prod &
goupdate list -d $TEST_DIR/express --type dev &
wait
```

### Test all commands systematically:
```bash
# 1. Scan - verify file detection
goupdate scan -d /tmp/project

# 2. List - verify package parsing and lock resolution
goupdate list -d /tmp/project
goupdate list -d /tmp/project --type prod
goupdate list -d /tmp/project --type dev
goupdate list -d /tmp/project -p js

# 3. Outdated - verify version fetching
goupdate outdated -d /tmp/project
goupdate outdated -d /tmp/project --major
goupdate outdated -d /tmp/project --minor
goupdate outdated -d /tmp/project --patch

# 4. Update - test with dry-run first
goupdate update -d /tmp/project --dry-run
goupdate update -d /tmp/project --dry-run --patch

# 5. REQUIRED: Actual update (with rollback capability)
# CRITICAL: Dry-run is NOT sufficient! You MUST test actual updates.
goupdate update -d /tmp/project --patch
git -C /tmp/project diff  # Review changes
git -C /tmp/project checkout .  # Rollback if needed
```

**IMPORTANT**: Step 5 (actual update) is MANDATORY for battle testing.
Testing with `--dry-run` only does NOT validate the update logic.
Always perform actual updates on cloned test projects and verify changes.

### What to look for:
- Invalid output formats or misaligned tables
- Incorrect status values
- Missing or wrong version numbers
- Errors that should be warnings (or vice versa)
- Crashes or panics

### REQUIRED: Test examples and workflow commands
When battle testing, you MUST also verify:

1. **Documentation examples work**: Test all examples in docs/*.md files
2. **Workflow commands work**: Test common user workflows:
   ```bash
   # Full workflow: scan → list → outdated → update
   goupdate scan -d /tmp/project
   goupdate list -d /tmp/project
   goupdate outdated -d /tmp/project
   goupdate update -d /tmp/project --patch -y  # Actual update!
   git -C /tmp/project diff                     # Verify changes
   ```
3. **All output formats work**: Test json, csv, xml for each command
4. **Filter combinations work**: Test --type, --rule, -p flags together

---

## **8. CHAOS ENGINEERING**

When asked to perform chaos engineering or validate test coverage:

### Methodology:
1. **Inventory** all features, commands, flags, config options
2. **Break** each feature deliberately (return empty, skip logic, etc.)
3. **Test** by running `go test ./...`
4. **Verify** tests catch the breakage
5. **Fix** by adding tests if breakage wasn't caught
6. **Restore** original code

### Break patterns:
```go
// Return empty
func SomeFunction() []string {
    return nil // CHAOS TEST: Return empty
}

// Skip validation
func Validate(x string) error {
    return nil // CHAOS TEST: Skip validation
}

// Always return false
func IsValid(x string) bool {
    return false // CHAOS TEST: Always return false
}
```

### Handle unused imports:
When breaking functions, remove unused imports to avoid build failures.

### Document results in chaos-testing.md:
| Test ID | Feature | File | Tests Caught? | Action |
|---------|---------|------|---------------|--------|
| 1.1 | FunctionName | pkg/path/file.go | YES/NO | None/Added test |

### Reference: [chaos-testing.md](docs/chaos-testing.md)

---

## **9. QUALITY ASSURANCE TASKS**

### Pre-release validation:
```bash
# 1. All tests pass
go test ./...

# 2. Race detection
go test -race ./...

# 3. Static analysis
go vet ./...

# 4. Coverage check
go test -cover ./pkg/... ./cmd/...

# 5. Build verification
go build ./...
```

### Coverage targets (see docs/testing.md):
| Package | Target |
|---------|--------|
| pkg/config | 80% |
| pkg/formats | 85% |
| pkg/lock | 80% |
| pkg/outdated | 80% |
| pkg/update | 80% |
| pkg/preflight | 75% |
| cmd | 70% |

---

## **10. DOCUMENTATION TASKS**

When updating documentation:

### Files to update:
- `docs/testing.md` - Test procedures, TDD, coverage
- `docs/cli.md` - Command reference
- `docs/configuration.md` - Config options
- `docs/features.md` - Feature overview
- `README.md` - Quick start, links
- `CHAOS_TEST_PLAN.md` - Chaos test results

### Documentation requirements:
- Update coverage tables with actual numbers
- Include code examples
- Reference test files for each feature
- Keep pre-release checklists current

---

## **11. COMMON AGENT TASKS**

Tasks that agents should be able to handle:

| Task | Description | Key Commands |
|------|-------------|--------------|
| Battle test | Test CLI on real projects | Clone, scan, list, outdated, update |
| Chaos test | Validate test coverage | Break feature, test, fix, restore |
| Add test | Write test for uncovered code | Identify gap, write test, verify |
| Fix bug | Reproduce, fix, add regression test | Write failing test first |
| Add feature | TDD approach | Test → Implement → Refactor |
| Update docs | Keep docs in sync | Update all affected .md files |
| Coverage check | Verify coverage targets | `go test -cover ./...` |
| Release prep | Pre-release validation | Tests, race, vet, coverage |

---

## **12. TESTDATA MANAGEMENT**

### Directory structure:
```
pkg/_testdata/           # Package-specific test fixtures
testdata/                # Shared test fixtures (symlink to _old_codebase/testdata)
examples/                # Example project configurations
```

### When creating test fixtures:
- Use real package files from actual projects
- Include lock files for accurate installed version testing
- Create edge cases in `_testdata/_edgecases/` subdirectory
- Never commit node_modules or vendor directories
