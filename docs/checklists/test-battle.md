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

## Phase 8: Testdata Verification

Verify all testdata directories are valid and parseable:

### 8A: Package Manager Testdata (`pkg/testdata/`)

| PM | Directory | scan | list | Files Present |
|----|-----------|:----:|:----:|---------------|
| npm | `pkg/testdata/npm/` | [ ] | [ ] | package.json, package-lock.json |
| npm v1 | `pkg/testdata/npm_v1/` | [ ] | [ ] | lockfileVersion 1 |
| npm v2 | `pkg/testdata/npm_v2/` | [ ] | [ ] | lockfileVersion 2 |
| npm v3 | `pkg/testdata/npm_v3/` | [ ] | [ ] | lockfileVersion 3 |
| pnpm | `pkg/testdata/pnpm/` | [ ] | [ ] | pnpm-lock.yaml |
| pnpm v6-v9 | `pkg/testdata/pnpm_v6-v9/` | [ ] | [ ] | Multiple versions |
| yarn | `pkg/testdata/yarn/` | [ ] | [ ] | yarn.lock (classic) |
| yarn berry | `pkg/testdata/yarn_berry/` | [ ] | [ ] | yarn.lock (berry) |
| composer | `pkg/testdata/composer/` | [ ] | [ ] | composer.json, composer.lock |
| mod | `pkg/testdata/mod/` | [ ] | [ ] | go.mod, go.sum |
| pipfile | `pkg/testdata/pipfile/` | [ ] | [ ] | Pipfile, Pipfile.lock |
| requirements | `pkg/testdata/requirements/` | [ ] | [ ] | requirements.txt |
| msbuild | `pkg/testdata/msbuild/` | [ ] | [ ] | .csproj |
| nuget | `pkg/testdata/nuget/` | [ ] | [ ] | packages.config |

```bash
# Verify all testdata directories parse correctly (parallel)
for dir in npm npm_v1 npm_v2 npm_v3 pnpm yarn composer mod pipfile requirements msbuild nuget; do
    $GOUPDATE list -d pkg/testdata/$dir -o json 2>&1 | head -5 &
done
wait
```

### 8B: Special Testdata

| Directory | Purpose | Status |
|-----------|---------|--------|
| `pkg/testdata/groups/` | Group filtering tests | [ ] |
| `pkg/testdata/incremental/` | Incremental update tests | [ ] |
| `pkg/testdata/ignored_packages/` | Package ignore tests | [ ] |

---

## Phase 9: Edge Cases Testing

Test edge case scenarios from `_edge-cases/` directories:

### 9A: No Lock File Scenarios

| PM | Path | Expected Behavior | Status |
|----|------|-------------------|--------|
| npm | `npm/_edge-cases/no-lock/` | LockMissing status | [ ] |
| composer | `composer/_edge-cases/no-lock/` | LockMissing status | [ ] |
| mod | `mod/_edge-cases/no-lock/` | LockMissing status | [ ] |
| pipfile | `pipfile/_edge-cases/no-lock/` | LockMissing status | [ ] |
| msbuild | `msbuild/_edge-cases/no-lock/` | LockMissing status | [ ] |
| nuget | `nuget/_edge-cases/no-lock/` | LockMissing status | [ ] |

```bash
# Test all no-lock scenarios (parallel)
$GOUPDATE list -d pkg/testdata/npm/_edge-cases/no-lock -o json &
$GOUPDATE list -d pkg/testdata/composer/_edge-cases/no-lock -o json &
$GOUPDATE list -d pkg/testdata/mod/_edge-cases/no-lock -o json &
wait
```

### 9B: Prerelease Version Scenarios

| PM | Path | Expected Behavior | Status |
|----|------|-------------------|--------|
| npm | `npm/_edge-cases/prerelease/` | Prerelease versions detected | [ ] |

### 9C: Edge Case Test Files

| Test File | Purpose | Status |
|-----------|---------|--------|
| `cmd/edge_cases_test.go` | Security, network, output edge cases | [ ] |
| `cmd/context_cancellation_test.go` | Context cancellation handling | [ ] |

---

## Phase 10: Examples Testing

Test with real-world example projects from `examples/`:

### 10A: Example Projects (Parallel)

```bash
# Test all example projects in parallel
for project in go-cli react-app django-app laravel-app kpas-api kpas-frontend ruby-api; do
    $GOUPDATE scan -d examples/$project &
done
wait
```

| Project | PM | scan | list | outdated | Status |
|---------|----| :---:|:----:|:--------:|--------|
| go-cli | mod | [ ] | [ ] | [ ] | Go CLI app |
| react-app | npm | [ ] | [ ] | [ ] | React frontend |
| django-app | pip | [ ] | [ ] | [ ] | Django backend |
| laravel-app | composer | [ ] | [ ] | [ ] | Laravel app |
| kpas-api | npm | [ ] | [ ] | [ ] | Node.js API |
| kpas-frontend | npm | [ ] | [ ] | [ ] | Frontend SPA |
| ruby-api | bundler | [ ] | [ ] | [ ] | Ruby API |
| github-workflows | - | [ ] | - | - | CI examples |

---

## Phase 11: Chaos & Integration Tests

Verify comprehensive test suites pass:

### 11A: Chaos Tests

| Test File | Lines | Purpose | Status |
|-----------|-------|---------|--------|
| `pkg/update/chaos_test.go` | 811 | Filesystem errors, rollback, concurrent access | [ ] |
| `pkg/outdated/chaos_versioning_test.go` | 849 | Version parsing chaos | [ ] |
| `pkg/config/chaos_config_test.go` | 841 | Config loading/validation chaos | [ ] |

```bash
# Run chaos tests
go test -v ./pkg/update -run Chaos
go test -v ./pkg/outdated -run Chaos
go test -v ./pkg/config -run Chaos
```

### 11B: Integration Tests

| Test File | Purpose | Requires | Status |
|-----------|---------|----------|--------|
| `cmd/update_integration_test.go` | Real PM execution | go, npm installed | [ ] |
| `cmd/output_format_integration_test.go` | All format outputs | - | [ ] |

```bash
# Run integration tests (requires actual package managers)
go test -v ./cmd -run Integration
```

### 11C: E2E Tests

| Test File | Purpose | Status |
|-----------|---------|--------|
| `cmd/e2e_test.go` | Exit codes, GitHub Actions compat | [ ] |

```bash
# Run E2E tests
go test -v ./cmd -run E2E
```

### 11D: Exit Code Verification

| Scenario | Expected Code | Status |
|----------|---------------|--------|
| Success | 0 | [ ] |
| Partial failure | 1 | [ ] |
| Complete failure | 2 | [ ] |
| Config error | 3 | [ ] |

---

## Phase 12: Error Testdata (`testdata_errors/`)

Test error handling with intentionally malformed files:

### 12A: Config Errors (`_config-errors/`)

| Scenario | Path | Expected | Status |
|----------|------|----------|--------|
| Invalid YAML | `_config-errors/invalid-yaml/` | Parse error | [ ] |
| Cyclic extends (direct) | `_config-errors/cyclic-extends-direct/` | Cycle error | [ ] |
| Cyclic extends (indirect) | `_config-errors/cyclic-extends-indirect/` | Cycle error | [ ] |
| Duplicate groups | `_config-errors/duplicate-groups/` | Validation error | [ ] |
| Empty extends | `_config-errors/empty-extends/` | Validation error | [ ] |
| Empty rules | `_config-errors/empty-rules/` | Validation error | [ ] |
| Unknown extends | `_config-errors/unknown-extends/` | Not found error | [ ] |
| Type mismatch (include) | `_config-errors/type-mismatch-include/` | Type error | [ ] |
| Type mismatch (rules) | `_config-errors/type-mismatch-rules/` | Type error | [ ] |
| Path traversal | `_config-errors/path-traversal/` | Security error | [ ] |

### 12B: Invalid Syntax (`_invalid-syntax/`)

| PM | Path | Expected | Status |
|----|------|----------|--------|
| npm | `_invalid-syntax/npm/` | JSON parse error | [ ] |
| pnpm | `_invalid-syntax/pnpm/` | YAML parse error | [ ] |
| yarn | `_invalid-syntax/yarn/` | Parse error | [ ] |
| composer | `_invalid-syntax/composer/` | JSON parse error | [ ] |
| mod | `_invalid-syntax/mod/` | go.mod parse error | [ ] |
| requirements | `_invalid-syntax/requirements/` | Parse error | [ ] |
| pipfile | `_invalid-syntax/pipfile/` | TOML parse error | [ ] |
| msbuild | `_invalid-syntax/msbuild/` | XML parse error | [ ] |
| nuget | `_invalid-syntax/nuget/` | XML parse error | [ ] |

### 12C: Lock File Errors

| Scenario | Path | Expected | Status |
|----------|------|----------|--------|
| Lock errors | `_lock-errors/` | Parse error | [ ] |
| Lock missing | `_lock-missing/` | LockMissing status | [ ] |
| Lock not found | `_lock-not-found/` | Not found error | [ ] |
| Lock scenarios | `_lock-scenarios/` | Multi-lock handling | [ ] |

### 12D: Malformed Files

| Type | Path | Expected | Status |
|------|------|----------|--------|
| Malformed JSON | `malformed-json/` | JSON parse error | [ ] |
| Malformed XML | `malformed-xml/` | XML parse error | [ ] |
| Malformed structure | `_malformed/` | Structure error | [ ] |

```bash
# Test error handling (parallel across categories)
$GOUPDATE list -d pkg/testdata_errors/malformed-json 2>&1 | grep -i error &
$GOUPDATE list -d pkg/testdata_errors/_invalid-syntax/npm 2>&1 | grep -i error &
$GOUPDATE config -d pkg/testdata_errors/_config-errors/invalid-yaml --validate 2>&1 &
wait
```

---

## Phase 13: Mock Error Scenarios (`mocksdata_errors/`)

Test mock-dependent error scenarios:

| Scenario | Path | Expected | Status |
|----------|------|----------|--------|
| Command timeout | `mocksdata_errors/command-timeout/` | Timeout error | [ ] |
| Invalid command | `mocksdata_errors/invalid-command/` | Command error | [ ] |
| Package not found | `mocksdata_errors/package-not-found/` | Not found error | [ ] |

---

## Phase 14: Config File Validation

Test `.goupdate.yml` files in examples and root:

### 14A: Root Config

| Test | Command | Status |
|------|---------|--------|
| Validate root config | `config -d . --validate` | [ ] |
| Show effective config | `config -d . --show-effective` | [ ] |
| Init new config | `config --init` (in temp dir) | [ ] |

### 14B: Example Configs

| Project | Command | Status |
|---------|---------|--------|
| go-cli | `config -d examples/go-cli --validate` | [ ] |
| react-app | `config -d examples/react-app --validate` | [ ] |
| django-app | `config -d examples/django-app --validate` | [ ] |
| laravel-app | `config -d examples/laravel-app --validate` | [ ] |
| kpas-api | `config -d examples/kpas-api --validate` | [ ] |
| kpas-frontend | `config -d examples/kpas-frontend --validate` | [ ] |
| ruby-api | `config -d examples/ruby-api --validate` | [ ] |

```bash
# Validate all example configs (parallel)
for project in go-cli react-app django-app laravel-app kpas-api kpas-frontend ruby-api; do
    $GOUPDATE config -d examples/$project --validate &
done
wait
```

---

## Phase 15: GitHub Actions Compatibility

Verify commands work as used in GitHub Actions:

### 15A: Action Commands (from `.github/actions/`)

| Action | Command Pattern | Status |
|--------|-----------------|--------|
| goupdate-check | `goupdate outdated -o json` | [ ] |
| goupdate-check | `goupdate outdated -o json --verbose` | [ ] |
| goupdate-update | `goupdate update --minor --continue-on-fail -y` | [ ] |
| goupdate-update | `goupdate update --patch --continue-on-fail -y` | [ ] |
| goupdate-update | `goupdate update --major --continue-on-fail -y` | [ ] |
| goupdate-update | `goupdate update --verbose --continue-on-fail -y --dry-run` | [ ] |

### 15B: Makefile Targets

| Target | Command | Status |
|--------|---------|--------|
| init | `make init` | [ ] |
| build | `make build` | [ ] |
| build-dev | `make build-dev` | [ ] |
| test | `make test` | [ ] |
| test-ci | `make test-ci` | [ ] |
| test-unit | `make test-unit` | [ ] |
| test-integration | `make test-integration` | [ ] |
| test-e2e | `make test-e2e` | [ ] |
| coverage | `make coverage` | [ ] |
| coverage-func | `make coverage-func` | [ ] |
| coverage-html | `make coverage-html` | [ ] |
| vet | `make vet` | [ ] |
| fmt | `make fmt` | [ ] |
| check | `make check` | [ ] |
| clean | `make clean` | [ ] |

---

## Phase 16: Help Output

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

## Phase 17: Final Verification

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
- Phase 8: Testdata verification (all directories)
- Phase 9: Edge cases testing
- Phase 10: Examples testing (all projects)
- Phase 11A: Chaos tests (independent test files)
- Phase 12: Error testdata testing (all categories)
- Phase 14: Config file validation (all example projects)
- Phase 15A: GitHub Action commands

### Must Run Sequentially
- Phase 2D.2: Actual updates (per project: update → verify → rollback)
- Phase 11B: Integration tests (may require PM installations)
- Phase 15B: Makefile targets (some modify files)
- Phase 17: Final verification (after all other phases)

### Collision Prevention
- Use unique `$TEST_DIR` per session
- Never run actual updates on same project simultaneously
- Run `git status` before commits

---

## Session Log

| Date | Tester | Go Version | OS | Branch | Result |
|------|--------|------------|----| -------|--------|
| | | | | | |
