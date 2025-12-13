# Testing Checklist Template

Use this checklist when battle testing or validating new features.
Copy this template to your progress report and check off items as completed.

---

## Phase 1: Setup (Parallelizable)

Run these tasks simultaneously:

```bash
# Terminal 1: Build binary
go build -o /tmp/goupdate .

# Terminal 2: Clone test projects (parallel)
TEST_DIR=$(mktemp -d)
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra &
git clone --depth 1 https://github.com/expressjs/express.git $TEST_DIR/express &
wait
```

- [ ] Binary built successfully
- [ ] Test directory created: `$TEST_DIR`
- [ ] cobra (Go) project cloned
- [ ] express (JS) project cloned

---

## Phase 2: Automated Tests (Run in Parallel)

These can run simultaneously in separate terminals:

### Group A: Unit Tests
```bash
go test ./... -count=1
```
- [ ] All packages pass
- [ ] No test pollution errors

### Group B: Race Detection
```bash
go test -race ./...
```
- [ ] No data races detected

### Group C: Coverage
```bash
make coverage-func
```
- [ ] Coverage >= 97% overall
- [ ] No decrease in modified packages

### Group D: Static Analysis
```bash
go vet ./... && make check
```
- [ ] No vet issues
- [ ] Linters pass

---

## Phase 3: CLI Battle Testing

### 3A: Scan Command (All Formats)
| Test | Command | Status |
|------|---------|--------|
| Go project | `goupdate scan -d $TEST_DIR/cobra` | [ ] |
| JS project | `goupdate scan -d $TEST_DIR/express` | [ ] |
| JSON output | `goupdate scan -d $TEST_DIR/cobra --output json` | [ ] |
| CSV output | `goupdate scan -d $TEST_DIR/cobra --output csv` | [ ] |
| XML output | `goupdate scan -d $TEST_DIR/cobra --output xml` | [ ] |

### 3B: List Command
| Test | Command | Status |
|------|---------|--------|
| Basic | `goupdate list -d $TEST_DIR/cobra` | [ ] |
| Filter prod | `goupdate list -d $TEST_DIR/cobra --type prod` | [ ] |
| Filter dev | `goupdate list -d $TEST_DIR/express --type dev` | [ ] |
| Filter PM | `goupdate list -d $TEST_DIR/cobra -p golang` | [ ] |
| JSON output | `goupdate list -d $TEST_DIR/cobra --output json` | [ ] |

### 3C: Outdated Command
| Test | Command | Status |
|------|---------|--------|
| Basic | `goupdate outdated -d $TEST_DIR/cobra` | [ ] |
| Major filter | `goupdate outdated -d $TEST_DIR/cobra --major` | [ ] |
| Minor filter | `goupdate outdated -d $TEST_DIR/cobra --minor` | [ ] |
| Patch filter | `goupdate outdated -d $TEST_DIR/cobra --patch` | [ ] |
| JSON output | `goupdate outdated -d $TEST_DIR/cobra --output json` | [ ] |

### 3D: Update Command (CRITICAL - Sequential)
**These MUST run in order:**

1. [ ] Dry run: `goupdate update -d $TEST_DIR/cobra --dry-run`
2. [ ] **ACTUAL UPDATE**: `goupdate update -d $TEST_DIR/cobra --patch -y`
3. [ ] Verify changes: `git -C $TEST_DIR/cobra diff`
4. [ ] Confirm manifest modified correctly
5. [ ] Rollback: `git -C $TEST_DIR/cobra checkout .`

---

## Phase 4: Error Handling

| Scenario | Expected | Status |
|----------|----------|--------|
| Invalid path | Clear error message | [ ] |
| No packages found | Informative message | [ ] |
| Invalid config | Helpful error | [ ] |
| Network timeout | Graceful handling | [ ] |

---

## Phase 5: Workflow Parity (CI Commands)

Verify these match GitHub Actions behavior:

- [ ] `make test-unit` - Passes with -race flag
- [ ] `make test-integration` - Integration tests pass
- [ ] `make coverage-func` - Coverage report generated
- [ ] `go build ./...` - Build succeeds

---

## Phase 6: Documentation

- [ ] Examples in `docs/*.md` still work
- [ ] CLI `--help` output accurate
- [ ] Progress report updated

---

## Phase 7: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] Clean git status (no untracked test files)
- [ ] Changes committed with descriptive message
- [ ] Pushed to correct branch
- [ ] Cleanup: `rm -rf $TEST_DIR`

---

## Output Format Matrix

Quick verification grid for all commands and formats:

| Format | scan | list | outdated | update |
|--------|:----:|:----:|:--------:|:------:|
| table  | [ ]  | [ ]  | [ ]      | [ ]    |
| json   | [ ]  | [ ]  | [ ]      | [ ]    |
| csv    | [ ]  | [ ]  | [ ]      | [ ]    |
| xml    | [ ]  | [ ]  | [ ]      | [ ]    |

---

## Issues Found

| Issue | Severity | Status | Notes |
|-------|----------|--------|-------|
|       |          |        |       |

---

## Test Environment

| Field | Value |
|-------|-------|
| Date | YYYY-MM-DD |
| Go Version | `go version` |
| OS | `uname -a` |
| Test Projects | cobra, express |
| Tester | Claude / Human |
| Branch | |
| Commit | |

---

## Parallel Execution Summary

**Can run in parallel:**
- Phase 1: Setup tasks (build + clone)
- Phase 2: All test groups (A, B, C, D)
- Phase 3A-3C: Scan, List, Outdated testing

**Must run sequentially:**
- Phase 3D: Update command (dry-run → actual → verify → rollback)
- Phase 7: Final verification (after all other phases)

**Collision avoidance:**
- Use separate `$TEST_DIR` for each parallel session
- Never run actual updates on the same project simultaneously
- Run `git status` before commits to catch untracked files
