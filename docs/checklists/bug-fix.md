# Bug Fix Checklist

Use this checklist when fixing bugs in goupdate.
**Parallel execution** - run independent verification tasks simultaneously.

---

## Quick Reference

### Common Bug Locations
| Bug Type | Location | Common Cause |
|----------|----------|--------------|
| Test pollution | `cmd/*_test.go` | Package-level flags not restored |
| Race condition | `pkg/*` | Concurrent map/slice access |
| Parsing error | `pkg/formats/` | Format edge cases, regex errors |
| Config error | `pkg/config/` | Validation missing, merge logic |
| Lock resolution | `pkg/lock/` | Lock file format changes |
| Update failure | `pkg/update/` | Constraint format, file write |
| Filter bug | `pkg/filtering/` | Case sensitivity, empty values |
| Output bug | `pkg/output/` | Format rendering, escaping |

### CLI Flags to Check
| Command | Critical Flags |
|---------|---------------|
| scan | `-d`, `-o`, `-f` |
| list | `-d`, `-o`, `-t`, `-p`, `-r`, `-n`, `-g` |
| outdated | All list + `--major`, `--minor`, `--patch` |
| update | All outdated + `--dry-run`, `-y`, `--skip-lock` |

---

## Phase 1: Investigation (Parallel)

Run these simultaneously:

```bash
# Terminal 1: Reproduce bug
goupdate <command-that-fails>

# Terminal 2: Check recent changes
git log --oneline -20

# Terminal 3: Run tests
go test ./...
```

| Task | Status |
|------|--------|
| Bug reproduced locally | [ ] |
| Root cause identified | [ ] |
| Affected code located | [ ] |
| Impact scope understood | [ ] |
| Related tests identified | [ ] |

### Reproduction Steps
```bash
# Document exact commands:
# 1.
# 2.
# 3.
```

### Root Cause Analysis
| Question | Answer |
|----------|--------|
| Which function fails? | |
| What input triggers it? | |
| What's the expected behavior? | |
| What's the actual behavior? | |
| When did it start? (commit) | |

---

## Phase 2: Test First (TDD)

**Write a failing test BEFORE fixing:**

```go
func TestBugFix_IssueXXX(t *testing.T) {
    // Arrange: Setup that reproduces the bug

    // Act: Execute the buggy code path

    // Assert: This should FAIL before the fix
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

| Task | Status |
|------|--------|
| Test reproduces bug | [ ] |
| Test FAILS before fix | [ ] |
| Test covers exact scenario | [ ] |
| Edge cases identified | [ ] |

### Test Locations by Bug Type
| Bug Type | Test File |
|----------|-----------|
| CLI scan | `cmd/scan_test.go` |
| CLI list | `cmd/list_test.go` |
| CLI outdated | `cmd/outdated_test.go` |
| CLI update | `cmd/update_test.go` |
| Config loading | `pkg/config/loader_test.go` |
| Config validation | `pkg/config/validation_test.go` |
| JSON parsing | `pkg/formats/json_test.go` |
| XML parsing | `pkg/formats/xml_test.go` |
| Raw/Regex parsing | `pkg/formats/raw_test.go` |
| Lock resolution | `pkg/lock/*_test.go` |
| Version checking | `pkg/outdated/*_test.go` |
| Update logic | `pkg/update/*_test.go` |
| Filtering | `pkg/filtering/*_test.go` |
| Output formatting | `pkg/output/*_test.go` |

---

## Phase 3: Fix Implementation

| Principle | Status |
|-----------|--------|
| Minimal change only | [ ] |
| No unrelated changes | [ ] |
| Existing tests pass | [ ] |
| New test passes | [ ] |
| Fix addresses root cause (not symptom) | [ ] |

### Common Fix Patterns

**Test Pollution Fix:**
```go
func TestSomething(t *testing.T) {
    // Save original
    orig := updateRuleFlag
    t.Cleanup(func() { updateRuleFlag = orig })

    // Test code
}
```

**Race Condition Fix:**
```go
// Add mutex protection
var mu sync.Mutex
mu.Lock()
defer mu.Unlock()
// Critical section
```

**Nil Check Fix:**
```go
if result != nil && len(result) > 0 {
    // Safe access
}
```

---

## Phase 4: Regression Testing (Parallel)

Run all simultaneously:

```bash
# Terminal 1: Unit tests
go test ./... -count=1

# Terminal 2: Race detection
go test -race ./...

# Terminal 3: Coverage
make coverage-func

# Terminal 4: Integration tests
make test-integration
```

| Test | Command | Status |
|------|---------|--------|
| Unit tests | `go test ./...` | [ ] |
| Race detection | `go test -race ./...` | [ ] |
| Coverage | `make coverage-func` | [ ] |
| Integration | `make test-integration` | [ ] |
| New test passes | `go test -v -run TestBugFix` | [ ] |

### Chaos Tests (Error Handling Verification)

If bug is error-related, verify chaos tests still pass:

| Test | Command | Status |
|------|---------|--------|
| Update chaos | `go test -v ./pkg/update -run Chaos` | [ ] |
| Outdated chaos | `go test -v ./pkg/outdated -run Chaos` | [ ] |
| Config chaos | `go test -v ./pkg/config -run Chaos` | [ ] |
| Edge cases | `go test -v ./cmd -run EdgeCase` | [ ] |

---

## Phase 5: Verification (Parallel)

### 5A: Original Scenario Fixed

| Check | Status |
|-------|--------|
| Bug no longer occurs | [ ] |
| All output formats work | [ ] |
| All filters work | [ ] |
| Error handling correct | [ ] |

### 5B: Edge Cases (Parallel)

Test related scenarios:
```bash
# Terminal 1: Empty inputs
goupdate list -d /empty/dir

# Terminal 2: Invalid inputs
goupdate list -d /nonexistent

# Terminal 3: Special characters
goupdate list -n "package@1.0.0"
```

| Edge Case | Status |
|-----------|--------|
| Empty/null inputs | [ ] |
| Invalid inputs | [ ] |
| Unicode/special chars | [ ] |
| Large datasets | [ ] |
| Boundary conditions | [ ] |

---

## Phase 6: Battle Testing

### Setup
```bash
export TEST_DIR=$(mktemp -d)
git clone --depth 1 <project-that-triggered-bug> $TEST_DIR/test
```

### Test All Commands
| Test | Status |
|------|--------|
| Bug scenario fixed | [ ] |
| scan works | [ ] |
| list works | [ ] |
| outdated works | [ ] |
| **ACTUAL update** works | [ ] |
| All output formats | [ ] |

---

## Phase 7: Documentation

| Task | Status |
|------|--------|
| Commit message documents fix | [ ] |
| User docs updated (if user-facing) | [ ] |
| Progress report created | [ ] |

### Commit Message Format
```
fix: Brief description

Root cause: [explanation of why it broke]
Fix: [what was changed to fix it]

Closes #XXX (if applicable)
```

---

## Phase 8: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] No races: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Cleanup: `rm -rf $TEST_DIR`

---

## Common Bug Categories

### Test Pollution
| Flag | Variable | Commands Affected |
|------|----------|-------------------|
| `-o/--output` | `outputFlag`, `updateOutputFlag`, `scanOutputFlag` | All |
| `-d/--directory` | `dirFlag`, `updateDirFlag` | All |
| `-r/--rule` | `ruleFlag`, `updateRuleFlag` | list, outdated, update |
| `-t/--type` | `typeFlag`, `updateTypeFlag` | list, outdated, update |
| `-p/--package-manager` | `pmFlag`, `updatePMFlag` | list, outdated, update |
| `--dry-run` | `dryRunFlag`, `updateDryRunFlag` | update |
| `--major/--minor/--patch` | `majorFlag`, `minorFlag`, `patchFlag` | outdated, update |
| `-y/--yes` | `yesFlag`, `updateYesFlag` | update |

### Race Conditions
| Component | Shared Resource | Fix |
|-----------|-----------------|-----|
| Config | Global config | Use sync.Once |
| Output | Stdout buffer | Use mutex |
| Temp files | File handles | Use unique names |
| Package list | Slice/map | Use sync.Mutex |

### Format Parsing
| Format | Common Issues | Check |
|--------|--------------|-------|
| JSON | Nested objects, arrays | Path extraction |
| XML | Namespaces, attributes | XPath correctness |
| YAML | Indentation, lists | Parser settings |
| Raw | Regex edge cases | Pattern matching |

---

## Parallel Execution Summary

### Can Run in Parallel
- Phase 1: Investigation tasks
- Phase 4: All regression tests
- Phase 5B: Edge case testing
- Phase 6: Format testing

### Must Run Sequentially
- Phase 2: Test must fail before fix
- Phase 3: Fix after test written
- Phase 8: After all verification
