# Bug Fix Checklist

Use this checklist when fixing bugs in goupdate.

---

## Phase 1: Investigation

- [ ] Bug reproduced locally
- [ ] Root cause identified
- [ ] Affected code located
- [ ] Impact scope understood (which commands/features affected)

### Reproduction Steps
```bash
# Document exact steps to reproduce:
# 1.
# 2.
# 3.
```

### Root Cause
```
# Document root cause:
```

---

## Phase 2: Test First (TDD)

Before fixing, write a test that fails:

- [ ] Test written that reproduces the bug
- [ ] Test fails before fix (confirms reproduction)
- [ ] Test covers the exact scenario reported
- [ ] Edge cases identified for additional tests

### Test Location
| Bug Type | Test File Location |
|----------|-------------------|
| CLI command | `cmd/*_test.go` |
| Config parsing | `pkg/config/*_test.go` |
| Format parsing | `pkg/formats/*_test.go` |
| Lock resolution | `pkg/lock/*_test.go` |
| Version checking | `pkg/outdated/*_test.go` |
| Update logic | `pkg/update/*_test.go` |

---

## Phase 3: Fix Implementation

- [ ] Minimal change to fix the bug
- [ ] No unrelated changes included
- [ ] Existing tests still pass
- [ ] New test now passes

### Fix Principles
- Fix the bug, not the symptom
- Don't refactor while fixing (separate PR)
- Keep changes focused and reviewable

---

## Phase 4: Regression Testing

### Unit Tests
- [ ] All existing tests pass: `go test ./...`
- [ ] New regression test passes
- [ ] Related tests reviewed for completeness

### Race Detection
- [ ] `go test -race ./...` - No races introduced

### Coverage
- [ ] Coverage not decreased
- [ ] Bug scenario now covered

---

## Phase 5: Verification

### Reproduce Original Scenario
- [ ] Original bug no longer occurs
- [ ] Fix works in all output formats (table, json, csv, xml)

### Edge Cases
- [ ] Similar scenarios tested
- [ ] Boundary conditions checked
- [ ] Null/empty inputs handled

---

## Phase 6: Battle Testing

Test the fix on real projects:

```bash
TEST_DIR=$(mktemp -d)
git clone --depth 1 <project-that-triggered-bug> $TEST_DIR/test
```

- [ ] Bug scenario verified fixed
- [ ] No regressions in related functionality
- [ ] All commands still work: scan, list, outdated, update

---

## Phase 7: Documentation

- [ ] Bug documented in commit message
- [ ] If user-facing: docs updated if needed
- [ ] Progress report: `docs/agents-progress/YYYY-MM-DD_bugfix-description.md`

### Commit Message Format
```
fix: Brief description of fix

Root cause: [explanation]
Fix: [what was changed]

Closes #XXX (if applicable)
```

---

## Phase 8: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] Race tests pass: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Commit with descriptive message
- [ ] Cleanup test directories

---

## Bug Categories Quick Reference

| Category | Common Causes | Where to Look |
|----------|--------------|---------------|
| Test pollution | Package-level flags not restored | `cmd/*_test.go` t.Cleanup() |
| Race condition | Concurrent map/slice access | Add mutex or use sync types |
| Parsing error | Format edge cases | `pkg/formats/`, testdata |
| Config error | Validation missing | `pkg/config/validate.go` |
| Update failure | Lock file mismatch | `pkg/lock/`, `pkg/update/` |
