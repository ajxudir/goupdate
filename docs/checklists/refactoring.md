# Refactoring Checklist

Use this checklist when refactoring code in goupdate.

---

## Phase 1: Planning

- [ ] Refactoring goal clearly defined
- [ ] Scope limited (avoid mixing refactoring with features/fixes)
- [ ] All existing tests pass before starting
- [ ] Identify all affected files

### Refactoring Types
| Type | Scope | Risk |
|------|-------|------|
| Rename | Single symbol | Low |
| Extract function | Single file | Low |
| Move code | Multiple files | Medium |
| Interface change | Package boundary | High |
| Architecture change | Multiple packages | High |

---

## Phase 2: Safety Net

Before refactoring, ensure tests are comprehensive:

- [ ] Coverage checked: `make coverage-func`
- [ ] Coverage >= 95% for affected code
- [ ] Edge cases have tests
- [ ] Integration tests exist for affected functionality

### Add Missing Tests If Needed
- [ ] Additional unit tests added
- [ ] Behavior documented in tests (not just implementation)

---

## Phase 3: Incremental Changes

Refactor in small, testable steps:

### Step Pattern
1. [ ] Make one small change
2. [ ] Run tests: `go test ./...`
3. [ ] Commit if tests pass
4. [ ] Repeat

### Commit Frequency
- Commit after each successful step
- Each commit should leave code in working state
- Use descriptive commit messages

---

## Phase 4: Code Quality

### Naming
- [ ] Names are clear and descriptive
- [ ] Consistent naming conventions
- [ ] No abbreviations (except common ones like `cfg`, `ctx`)

### Structure
- [ ] Functions are focused (single responsibility)
- [ ] No deep nesting (max 3 levels)
- [ ] Error handling is consistent

### Dependencies
- [ ] No circular imports
- [ ] Dependencies flow downward (cmd → pkg)
- [ ] Interfaces used at package boundaries

---

## Phase 5: Verification

### All Tests Pass
- [ ] `go test ./...` - All unit tests
- [ ] `go test -race ./...` - No race conditions
- [ ] `make test-integration` - Integration tests

### Static Analysis
- [ ] `go vet ./...` - No issues
- [ ] `gofmt -s -w .` - Code formatted
- [ ] `make check` - All checks pass

### Coverage
- [ ] Coverage not decreased
- [ ] New code properly covered

---

## Phase 6: Behavior Verification

### Battle Testing
Ensure refactoring didn't change behavior:

```bash
TEST_DIR=$(mktemp -d)
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra
```

- [ ] `goupdate scan` - Same output as before
- [ ] `goupdate list` - Same output as before
- [ ] `goupdate outdated` - Same output as before
- [ ] `goupdate update --dry-run` - Same plan as before
- [ ] All output formats work (table, json, csv, xml)

### Compare Before/After
```bash
# Before refactoring, save outputs:
goupdate list -d $TEST_DIR/cobra > /tmp/before.txt

# After refactoring, compare:
goupdate list -d $TEST_DIR/cobra > /tmp/after.txt
diff /tmp/before.txt /tmp/after.txt
```

- [ ] No unexpected output differences

---

## Phase 7: Documentation

- [ ] Code comments updated
- [ ] Architecture docs updated if structure changed
- [ ] README updated if public API changed

---

## Phase 8: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] Race tests pass: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Commits are logical and reviewable
- [ ] Cleanup test directories

---

## Common Refactoring Patterns

### Extract Function
```go
// Before: Long function with multiple responsibilities
func process() {
    // validation...
    // transformation...
    // output...
}

// After: Focused functions
func process() {
    validate()
    transform()
    output()
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
    db DatabaseReader
}

type DatabaseReader interface {
    Read(id string) (Data, error)
}
```

### Move to Package
When moving code between packages:
1. [ ] Create new location
2. [ ] Copy code (don't move yet)
3. [ ] Update imports in new location
4. [ ] Create forwarding functions in old location
5. [ ] Update all callers
6. [ ] Remove old code

---

## Anti-Patterns to Avoid

- [ ] Don't mix refactoring with bug fixes
- [ ] Don't mix refactoring with new features
- [ ] Don't refactor without tests
- [ ] Don't make large changes without commits
- [ ] Don't change behavior (that's a feature/fix, not refactoring)
