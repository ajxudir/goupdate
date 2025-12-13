# Refactoring Checklist

Use this checklist when refactoring code in goupdate.
**Parallel execution** - run independent verification tasks simultaneously.

---

## Quick Reference

### Package Structure
```
goupdate/
├── cmd/                    # CLI commands and flags
│   ├── root.go             # Root command, persistent flags
│   ├── scan.go             # Scan command
│   ├── list.go             # List command
│   ├── outdated.go         # Outdated command
│   ├── update.go           # Update command
│   └── config.go           # Config command
├── pkg/
│   ├── config/             # Configuration
│   │   ├── model.go        # Config structs
│   │   ├── loader.go       # Config loading
│   │   ├── validation.go   # Config validation
│   │   └── default.yml     # Default config
│   ├── formats/            # File format parsers
│   │   ├── json.go         # JSON parser
│   │   ├── xml.go          # XML parser
│   │   ├── raw.go          # Regex parser
│   │   └── yaml.go         # YAML parser
│   ├── lock/               # Lock file resolution
│   ├── outdated/           # Version checking
│   ├── update/             # Update execution
│   ├── filtering/          # Package filtering
│   ├── output/             # Output formatting
│   └── packages/           # Package detection
└── testdata/               # Test fixtures
```

### Refactoring Risk Levels
| Type | Scope | Risk | Example |
|------|-------|------|---------|
| Rename | Single symbol | Low | Rename variable |
| Extract function | Single file | Low | Split large function |
| Extract interface | Package boundary | Medium | Add testability |
| Move code | Multiple files | Medium | Reorganize package |
| Interface change | Package API | High | Change public function |
| Architecture | Multiple packages | High | Change data flow |

---

## Phase 1: Planning

- [ ] Refactoring goal clearly defined
- [ ] Scope limited (no mixing with features/fixes)
- [ ] All existing tests pass: `go test ./...`
- [ ] Affected files identified
- [ ] Risk level assessed
- [ ] Rollback plan clear

### Scope Checklist
| Question | Answer |
|----------|--------|
| What exactly are we changing? | |
| Why is this refactoring needed? | |
| What files are affected? | |
| What's the risk level? | |
| Can we do this incrementally? | |

---

## Phase 2: Safety Net (Parallel)

Before refactoring, verify test coverage:

```bash
# Terminal 1: Coverage report
make coverage-func

# Terminal 2: Run affected tests
go test -v ./pkg/affected/...

# Terminal 3: Check for flaky tests
go test -count=10 ./pkg/affected/...
```

| Check | Status |
|-------|--------|
| Coverage ≥95% for affected code | [ ] |
| All tests pass | [ ] |
| No flaky tests | [ ] |
| Edge cases covered | [ ] |
| Integration tests exist | [ ] |

### Add Missing Tests First
If coverage is insufficient:
- [ ] Add tests for uncovered paths
- [ ] Document behavior in tests
- [ ] Verify tests catch behavior changes

---

## Phase 3: Incremental Changes

**Small steps, frequent commits:**

### Step Pattern
```
1. Make ONE small change
2. Run tests: go test ./...
3. Commit if tests pass
4. Repeat
```

| Step | Change | Tests Pass | Committed |
|------|--------|------------|-----------|
| 1 | | [ ] | [ ] |
| 2 | | [ ] | [ ] |
| 3 | | [ ] | [ ] |
| 4 | | [ ] | [ ] |
| 5 | | [ ] | [ ] |

### Commit Frequently
- Each commit should leave code working
- Commits should be logically atomic
- Use descriptive commit messages

---

## Phase 4: Code Quality

### Naming
| Check | Status |
|-------|--------|
| Names are clear and descriptive | [ ] |
| Consistent naming conventions | [ ] |
| No unexplained abbreviations | [ ] |
| Package names match directory | [ ] |

### Structure
| Check | Status |
|-------|--------|
| Functions are focused (SRP) | [ ] |
| No deep nesting (max 3 levels) | [ ] |
| Error handling consistent | [ ] |
| No duplicate code | [ ] |
| Clear control flow | [ ] |

### Dependencies
| Check | Status |
|-------|--------|
| No circular imports | [ ] |
| Dependencies flow downward | [ ] |
| Interfaces at boundaries | [ ] |
| Minimal coupling | [ ] |

### Package-Level Flags
If refactoring commands:
| Flag Variable | Save/Restore Required |
|---------------|----------------------|
| `outputFlag` | [ ] |
| `dirFlag` | [ ] |
| `ruleFlag` | [ ] |
| `typeFlag` | [ ] |
| `pmFlag` | [ ] |
| `dryRunFlag` | [ ] |

---

## Phase 5: Verification (Parallel)

Run all simultaneously:

```bash
# Terminal 1: Unit tests
go test ./... -count=1

# Terminal 2: Race detection
go test -race ./...

# Terminal 3: Coverage
make coverage-func

# Terminal 4: Static analysis
go vet ./... && make check
```

| Test | Command | Status |
|------|---------|--------|
| Unit tests | `go test ./...` | [ ] |
| Race detection | `go test -race ./...` | [ ] |
| Coverage not decreased | `make coverage-func` | [ ] |
| Static analysis | `go vet ./...` | [ ] |
| Linters | `make check` | [ ] |
| Formatting | `gofmt -s -w .` | [ ] |

---

## Phase 6: Behavior Verification

### Before/After Comparison

```bash
export TEST_DIR=$(mktemp -d)
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra

# BEFORE refactoring (on main branch):
git stash
goupdate list -d $TEST_DIR/cobra > /tmp/before-list.txt
goupdate list -d $TEST_DIR/cobra -o json > /tmp/before-json.txt
goupdate outdated -d $TEST_DIR/cobra > /tmp/before-outdated.txt
git stash pop

# AFTER refactoring:
goupdate list -d $TEST_DIR/cobra > /tmp/after-list.txt
goupdate list -d $TEST_DIR/cobra -o json > /tmp/after-json.txt
goupdate outdated -d $TEST_DIR/cobra > /tmp/after-outdated.txt

# Compare:
diff /tmp/before-list.txt /tmp/after-list.txt
diff /tmp/before-json.txt /tmp/after-json.txt
diff /tmp/before-outdated.txt /tmp/after-outdated.txt
```

| Command | Before == After | Status |
|---------|-----------------|--------|
| `scan` | [ ] | |
| `list` | [ ] | |
| `list -o json` | [ ] | |
| `outdated` | [ ] | |
| `update --dry-run` | [ ] | |

### All Commands Work

| Command | Status |
|---------|--------|
| `scan -d $TEST_DIR/cobra` | [ ] |
| `list -d $TEST_DIR/cobra` | [ ] |
| `outdated -d $TEST_DIR/cobra` | [ ] |
| `update --dry-run -d $TEST_DIR/cobra` | [ ] |
| `config --show-defaults` | [ ] |

### All Output Formats (Parallel)

```bash
goupdate list -d $TEST_DIR/cobra -o table &
goupdate list -d $TEST_DIR/cobra -o json | jq . &
goupdate list -d $TEST_DIR/cobra -o csv &
goupdate list -d $TEST_DIR/cobra -o xml &
wait
```

| Format | Works | Status |
|--------|-------|--------|
| table | [ ] | |
| json | [ ] | |
| csv | [ ] | |
| xml | [ ] | |

---

## Phase 7: Documentation

- [ ] Code comments updated
- [ ] Architecture docs updated if structure changed
- [ ] README updated if API changed
- [ ] No stale TODOs left

---

## Phase 8: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] No races: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Commits are logical and reviewable
- [ ] Cleanup: `rm -rf $TEST_DIR`

---

## Refactoring Patterns

### Extract Function
```go
// Before: Long function
func process() {
    // validation...
    // transformation...
    // output...
}

// After: Focused functions
func process() {
    if err := validate(); err != nil { return err }
    data := transform()
    return output(data)
}
```

### Extract Interface
```go
// Before: Concrete dependency
type Service struct {
    db *Database
}

// After: Interface dependency
type Service struct {
    db DataReader
}

type DataReader interface {
    Read(id string) (Data, error)
}
```

### Move to Package
When relocating code:
1. [ ] Create new location
2. [ ] Copy code (don't delete yet)
3. [ ] Update imports in new location
4. [ ] Create forwarding in old location (temporary)
5. [ ] Update all callers
6. [ ] Remove old code and forwarding
7. [ ] Tests pass at each step

### Rename Symbol
```bash
# Use Go tools for safe renames:
gorename -from 'pkg.OldName' -to 'NewName'
```

---

## Anti-Patterns to Avoid

| Don't | Why |
|-------|-----|
| Mix refactoring with features | Hard to review, risky |
| Mix refactoring with bug fixes | Can't bisect issues |
| Refactor without tests | Can't verify behavior preserved |
| Make large changes without commits | Can't rollback safely |
| Change behavior | That's a feature, not refactoring |
| Skip battle testing | May break real usage |

---

## Parallel Execution Summary

### Can Run in Parallel
- Phase 2: All safety net checks
- Phase 5: All verification tests
- Phase 6: Output format testing
- Before/after comparisons (different terminals)

### Must Run Sequentially
- Phase 3: Incremental changes (one at a time)
- Phase 8: Final verification (after all other phases)
