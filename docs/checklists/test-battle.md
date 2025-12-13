# Battle Testing Checklist

Comprehensive CLI testing against real-world projects.
**Parallel execution optimized** - run independent tests simultaneously.

---

## Quick Setup Script

```bash
#!/bin/bash
# battle-test-setup.sh - Run once to prepare environment
set -e

export TEST_DIR=$(mktemp -d)
export GOUPDATE=/tmp/goupdate

# Phase 1: Parallel setup
echo "Building binary and cloning test projects..."
go build -o $GOUPDATE . &
BUILD_PID=$!

# Clone diverse ecosystem projects in parallel
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra &       # Go
git clone --depth 1 https://github.com/expressjs/express.git $TEST_DIR/express & # JS (npm)
git clone --depth 1 https://github.com/laravel/laravel.git $TEST_DIR/laravel &   # PHP
git clone --depth 1 https://github.com/pallets/flask.git $TEST_DIR/flask &       # Python
wait

wait $BUILD_PID
echo "Setup complete. TEST_DIR=$TEST_DIR"
```

---

## Phase 1: Automated Tests (Parallel)

Run all four in separate terminals simultaneously:

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
| Unit tests | `go test ./... -count=1` | [ ] |
| Race detection | `go test -race ./...` | [ ] |
| Coverage ≥97% | `make coverage-func` | [ ] |
| Static analysis | `go vet ./... && make check` | [ ] |

---

## Phase 2: Command Testing (Parallel by Command)

### 2A: Scan Command

**Can run all scan tests in parallel** (read-only operation):

```bash
# Run these simultaneously in separate terminals
$GOUPDATE scan -d $TEST_DIR/cobra &
$GOUPDATE scan -d $TEST_DIR/express &
$GOUPDATE scan -d $TEST_DIR/laravel &
$GOUPDATE scan -d $TEST_DIR/flask &
wait
```

| Test | Command | Status |
|------|---------|--------|
| Go project | `scan -d $TEST_DIR/cobra` | [ ] |
| JS project | `scan -d $TEST_DIR/express` | [ ] |
| PHP project | `scan -d $TEST_DIR/laravel` | [ ] |
| Python project | `scan -d $TEST_DIR/flask` | [ ] |
| JSON output | `scan -d $TEST_DIR/cobra -o json` | [ ] |
| CSV output | `scan -d $TEST_DIR/cobra -o csv` | [ ] |
| XML output | `scan -d $TEST_DIR/cobra -o xml` | [ ] |
| File filter | `scan -d $TEST_DIR/express -f "package.json"` | [ ] |
| Verbose | `scan -d $TEST_DIR/cobra --verbose` | [ ] |

---

### 2B: List Command

**Can run all list tests in parallel** (read-only):

| Test | Command | Status |
|------|---------|--------|
| **Basic listing** | | |
| Go packages | `list -d $TEST_DIR/cobra` | [ ] |
| JS packages | `list -d $TEST_DIR/express` | [ ] |
| PHP packages | `list -d $TEST_DIR/laravel` | [ ] |
| Python packages | `list -d $TEST_DIR/flask` | [ ] |
| **Type filters** | | |
| Prod only | `list -d $TEST_DIR/express -t prod` | [ ] |
| Dev only | `list -d $TEST_DIR/express -t dev` | [ ] |
| All types | `list -d $TEST_DIR/express -t all` | [ ] |
| **Package manager filters** | | |
| JS only | `list -d $TEST_DIR -p js` | [ ] |
| Golang only | `list -d $TEST_DIR -p golang` | [ ] |
| PHP only | `list -d $TEST_DIR -p php` | [ ] |
| Python only | `list -d $TEST_DIR -p python` | [ ] |
| Multiple PMs | `list -d $TEST_DIR -p js,golang` | [ ] |
| **Rule filters** | | |
| NPM rule | `list -d $TEST_DIR -r npm` | [ ] |
| Mod rule | `list -d $TEST_DIR -r mod` | [ ] |
| Composer rule | `list -d $TEST_DIR -r composer` | [ ] |
| **Name filters** | | |
| Single package | `list -d $TEST_DIR/express -n express` | [ ] |
| Multiple names | `list -d $TEST_DIR/express -n "accepts,body-parser"` | [ ] |
| **Output formats** | | |
| Table (default) | `list -d $TEST_DIR/cobra` | [ ] |
| JSON | `list -d $TEST_DIR/cobra -o json` | [ ] |
| CSV | `list -d $TEST_DIR/cobra -o csv` | [ ] |
| XML | `list -d $TEST_DIR/cobra -o xml` | [ ] |
| **Combined filters** | | |
| Type + PM | `list -d $TEST_DIR -t prod -p js` | [ ] |
| Type + Rule | `list -d $TEST_DIR -t dev -r npm` | [ ] |
| PM + Name | `list -d $TEST_DIR -p golang -n cobra` | [ ] |

---

### 2C: Outdated Command

**Can run in parallel** (network calls, but read-only):

| Test | Command | Status |
|------|---------|--------|
| **Basic outdated** | | |
| Go packages | `outdated -d $TEST_DIR/cobra` | [ ] |
| JS packages | `outdated -d $TEST_DIR/express` | [ ] |
| PHP packages | `outdated -d $TEST_DIR/laravel` | [ ] |
| **Version scope** | | |
| Major updates | `outdated -d $TEST_DIR/cobra --major` | [ ] |
| Minor updates | `outdated -d $TEST_DIR/cobra --minor` | [ ] |
| Patch updates | `outdated -d $TEST_DIR/cobra --patch` | [ ] |
| **Filters** | | |
| Type filter | `outdated -d $TEST_DIR/express -t prod` | [ ] |
| PM filter | `outdated -d $TEST_DIR -p js` | [ ] |
| Rule filter | `outdated -d $TEST_DIR -r mod` | [ ] |
| Name filter | `outdated -d $TEST_DIR/cobra -n cobra` | [ ] |
| **Special flags** | | |
| No timeout | `outdated -d $TEST_DIR/cobra --no-timeout` | [ ] |
| Skip preflight | `outdated -d $TEST_DIR/cobra --skip-preflight` | [ ] |
| Continue on fail | `outdated -d $TEST_DIR/cobra --continue-on-fail` | [ ] |
| **Output formats** | | |
| JSON | `outdated -d $TEST_DIR/cobra -o json` | [ ] |
| CSV | `outdated -d $TEST_DIR/cobra -o csv` | [ ] |
| XML | `outdated -d $TEST_DIR/cobra -o xml` | [ ] |

---

### 2D: Update Command (SEQUENTIAL - Critical)

**MUST run sequentially per project** (modifies files):

#### 2D.1: Dry Run Tests (Safe - can run in parallel across projects)

```bash
# These can run in parallel on different projects
$GOUPDATE update -d $TEST_DIR/cobra --dry-run &
$GOUPDATE update -d $TEST_DIR/express --dry-run &
wait
```

| Test | Command | Status |
|------|---------|--------|
| Dry run Go | `update -d $TEST_DIR/cobra --dry-run` | [ ] |
| Dry run JS | `update -d $TEST_DIR/express --dry-run` | [ ] |
| Dry run patch | `update -d $TEST_DIR/cobra --dry-run --patch` | [ ] |
| Dry run minor | `update -d $TEST_DIR/cobra --dry-run --minor` | [ ] |
| Dry run major | `update -d $TEST_DIR/cobra --dry-run --major` | [ ] |
| Dry run JSON | `update -d $TEST_DIR/cobra --dry-run -o json` | [ ] |

#### 2D.2: Actual Update Tests (SEQUENTIAL per project)

**CRITICAL: Test actual updates, not just dry-run!**

```bash
# Project 1: Go (cobra)
$GOUPDATE update -d $TEST_DIR/cobra --patch -y
git -C $TEST_DIR/cobra diff          # Verify changes
git -C $TEST_DIR/cobra checkout .    # Rollback

# Project 2: JS (express)
$GOUPDATE update -d $TEST_DIR/express --patch -y
git -C $TEST_DIR/express diff
git -C $TEST_DIR/express checkout .
```

| Test | Commands (Sequential) | Status |
|------|----------------------|--------|
| **Go Project** | | |
| Patch update | `update -d $TEST_DIR/cobra --patch -y` | [ ] |
| Verify changes | `git -C $TEST_DIR/cobra diff` | [ ] |
| Manifest correct | Check go.mod has updated versions | [ ] |
| Rollback | `git -C $TEST_DIR/cobra checkout .` | [ ] |
| **JS Project** | | |
| Patch update | `update -d $TEST_DIR/express --patch -y` | [ ] |
| Verify changes | `git -C $TEST_DIR/express diff` | [ ] |
| package.json correct | Check package.json versions | [ ] |
| Rollback | `git -C $TEST_DIR/express checkout .` | [ ] |
| **Special flags** | | |
| Skip lock | `update -d $TEST_DIR/cobra --patch -y --skip-lock` | [ ] |
| Continue on fail | `update -d $TEST_DIR/cobra --patch -y --continue-on-fail` | [ ] |
| Incremental | `update -d $TEST_DIR/cobra --patch -y --incremental` | [ ] |
| **Filtered updates** | | |
| Single package | `update -d $TEST_DIR/cobra --patch -y -n github.com/spf13/pflag` | [ ] |
| By type | `update -d $TEST_DIR/express --patch -y -t prod` | [ ] |

---

### 2E: Config Command (Parallel - read-only)

| Test | Command | Status |
|------|---------|--------|
| Show defaults | `config --show-defaults` | [ ] |
| Show effective | `config -d $TEST_DIR/cobra --show-effective` | [ ] |
| Validate config | `config -d $TEST_DIR/cobra --validate` | [ ] |
| Init config | `config -d /tmp/test-init --init` | [ ] |
| Custom config | `config -c /path/to/config.yml --show-effective` | [ ] |

---

### 2F: Version Command

| Test | Command | Status |
|------|---------|--------|
| Version flag | `--version` | [ ] |
| Version output | Verify format: `goupdate version X.Y.Z (commit)` | [ ] |

---

## Phase 3: Output Format Matrix

Verify all commands produce valid output in all formats:

| Format | scan | list | outdated | update |
|--------|:----:|:----:|:--------:|:------:|
| table | [ ] | [ ] | [ ] | [ ] |
| json | [ ] | [ ] | [ ] | [ ] |
| csv | [ ] | [ ] | [ ] | [ ] |
| xml | [ ] | [ ] | [ ] | [ ] |

**JSON Validation** (can run in parallel):
```bash
$GOUPDATE scan -d $TEST_DIR/cobra -o json | jq . &
$GOUPDATE list -d $TEST_DIR/cobra -o json | jq . &
$GOUPDATE outdated -d $TEST_DIR/cobra -o json | jq . &
wait
```

---

## Phase 4: Filter Combinations Matrix

Test filter combinations work correctly:

| Type | PM | Rule | Name | Group | Expected |
|------|----|----- |------|-------|----------|
| prod | js | npm | - | - | [ ] JS prod deps |
| dev | js | npm | - | - | [ ] JS dev deps |
| all | golang | mod | - | - | [ ] All Go deps |
| prod | - | - | express | - | [ ] Single package |
| - | js,php | - | - | - | [ ] Multiple PMs |

---

## Phase 5: Package Manager Coverage

Verify each supported PM works:

| PM | Rule | Test Project | scan | list | outdated | update |
|----|------|--------------|:----:|:----:|:--------:|:------:|
| js | npm | express | [ ] | [ ] | [ ] | [ ] |
| js | yarn | (if available) | [ ] | [ ] | [ ] | [ ] |
| js | pnpm | (if available) | [ ] | [ ] | [ ] | [ ] |
| php | composer | laravel | [ ] | [ ] | [ ] | [ ] |
| python | requirements | flask | [ ] | [ ] | [ ] | [ ] |
| golang | mod | cobra | [ ] | [ ] | [ ] | [ ] |
| dotnet | msbuild | (if available) | [ ] | [ ] | [ ] | [ ] |

---

## Phase 6: Error Handling

| Scenario | Command | Expected | Status |
|----------|---------|----------|--------|
| Invalid path | `scan -d /nonexistent` | Error message | [ ] |
| Empty directory | `scan -d /tmp/empty` | No packages found | [ ] |
| Invalid config | `config -c /bad.yml --validate` | Validation error | [ ] |
| Network timeout | `outdated --timeout 1ms` | Timeout error | [ ] |
| Missing lock file | `list -d (project without lock)` | LockMissing status | [ ] |
| Invalid package name | `update -n "###invalid###"` | Error/no match | [ ] |

---

## Phase 7: CI Workflow Parity

Verify make targets match CI:

| Target | Command | Status |
|--------|---------|--------|
| Build | `make build` | [ ] |
| Unit tests | `make test-unit` | [ ] |
| Integration | `make test-integration` | [ ] |
| Coverage | `make coverage-func` | [ ] |
| Lint | `make check` | [ ] |
| Full test | `make test` | [ ] |

---

## Phase 8: Help Output

Verify all commands have accurate help:

| Command | Check | Status |
|---------|-------|--------|
| `goupdate --help` | Root help shows all commands | [ ] |
| `goupdate scan --help` | All scan flags documented | [ ] |
| `goupdate list --help` | All list flags documented | [ ] |
| `goupdate outdated --help` | All outdated flags documented | [ ] |
| `goupdate update --help` | All update flags documented | [ ] |
| `goupdate config --help` | All config flags documented | [ ] |

---

## Phase 9: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] No race conditions: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Cleanup: `rm -rf $TEST_DIR`

---

## Parallel Execution Summary

### Can Run in Parallel
- Phase 1: All automated tests (4 terminals)
- Phase 2A: All scan tests
- Phase 2B: All list tests
- Phase 2C: All outdated tests
- Phase 2D.1: Dry-run updates (across different projects)
- Phase 2E: All config tests
- Phase 3: JSON validation
- Phase 5: PM tests (across different projects)

### Must Run Sequentially
- Phase 2D.2: Actual updates (per project: update → verify → rollback)
- Phase 9: Final verification (after all other phases)

### Collision Prevention
- Use unique `$TEST_DIR` per session
- Never run actual updates on same project simultaneously
- Run `git status` before commits

---

## Session Log

| Date | Tester | Go Version | OS | Branch | Result |
|------|--------|------------|----| -------|--------|
| | | | | | |
