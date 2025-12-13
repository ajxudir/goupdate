# Feature Development Checklist

Use this checklist when adding new features to goupdate.
**Parallel execution** - run independent tasks simultaneously.

---

## Quick Reference

### Package Locations
| Component | Package | Purpose |
|-----------|---------|---------|
| Commands | `cmd/` | CLI commands, flags, output |
| Config | `pkg/config/` | Config loading, validation, model |
| Formats | `pkg/formats/` | Package file parsers (json, yaml, xml, raw) |
| Lock | `pkg/lock/` | Lock file resolution |
| Outdated | `pkg/outdated/` | Version checking, comparison |
| Update | `pkg/update/` | File modification, constraint update |
| Filtering | `pkg/filtering/` | Type, PM, rule, name filters |
| Output | `pkg/output/` | Table, JSON, CSV, XML formatting |
| Packages | `pkg/packages/` | Package detection, grouping |

### CLI Commands
| Command | File | Flags |
|---------|------|-------|
| scan | `cmd/scan.go` | `-d`, `-c`, `-o`, `-f`, `--verbose` |
| list | `cmd/list.go` | `-d`, `-c`, `-o`, `-f`, `-t`, `-p`, `-r`, `-n`, `-g` |
| outdated | `cmd/outdated.go` | All list flags + `--major`, `--minor`, `--patch`, `--no-timeout`, `--skip-preflight`, `--continue-on-fail` |
| update | `cmd/update.go` | All outdated flags + `--dry-run`, `--skip-lock`, `-y`, `--incremental`, `--skip-system-tests`, `--system-test-mode` |
| config | `cmd/config.go` | `--show-defaults`, `--show-effective`, `--init`, `--validate` |

---

## Phase 1: Planning (Parallel)

Run these tasks simultaneously:

```bash
# Terminal 1: Analyze existing code
grep -r "similar_feature" ./

# Terminal 2: Check test coverage
make coverage-func

# Terminal 3: Review documentation
ls docs/user/ docs/developer/
```

- [ ] Feature requirements documented
- [ ] Affected packages identified
- [ ] Similar functionality checked
- [ ] Config fields needed identified
- [ ] CLI flags needed identified
- [ ] Output formats affected
- [ ] Filter options affected
- [ ] Package managers affected

---

## Phase 2: Implementation

### 2A: CLI Flags (if adding new flags)

| Requirement | Status |
|-------------|--------|
| Flag registered with cobra | [ ] |
| Short and long form | [ ] |
| Default value set | [ ] |
| Flag added to help text | [ ] |
| Flag validated | [ ] |
| Flag documented in `--help` | [ ] |

**Flag Types:**
```go
// String flag
cmd.Flags().StringVarP(&flagVar, "name", "n", "default", "description")

// Bool flag
cmd.Flags().BoolVarP(&flagVar, "dry-run", "", false, "description")

// StringSlice flag
cmd.Flags().StringSliceVarP(&flagVar, "type", "t", []string{"all"}, "description")
```

### 2B: Config Fields (if adding new config)

| Requirement | File | Status |
|-------------|------|--------|
| Field added to model | `pkg/config/model.go` | [ ] |
| JSON/YAML tags | `pkg/config/model.go` | [ ] |
| Default value | `pkg/config/default.yml` | [ ] |
| Validation rule | `pkg/config/validation.go` | [ ] |
| Merge logic | `pkg/config/loader.go` | [ ] |
| Documentation | `docs/user/configuration.md` | [ ] |

**Config Field Template:**
```go
type Config struct {
    NewField string `yaml:"new_field" json:"new_field"`
}
```

### 2C: Output Formats (MANDATORY)

All commands must support all formats:

| Format | Struct | Render Function | Status |
|--------|--------|-----------------|--------|
| table | `pkg/output/types.go` | `RenderTable()` | [ ] |
| json | `pkg/output/types.go` | `RenderJSON()` | [ ] |
| csv | `pkg/output/types.go` | `RenderCSV()` | [ ] |
| xml | `pkg/output/types.go` | `RenderXML()` | [ ] |

### 2D: Filter Support (if applicable)

| Filter | Flag | Function | Status |
|--------|------|----------|--------|
| Type | `-t` | `FilterByType()` | [ ] |
| PM | `-p` | `FilterByPM()` | [ ] |
| Rule | `-r` | `FilterByRule()` | [ ] |
| Name | `-n` | `FilterByName()` | [ ] |
| Group | `-g` | `FilterByGroup()` | [ ] |
| File | `-f` | `FilterByFile()` | [ ] |

---

## Phase 3: Testing (Parallel)

### 3A: Unit Tests (can run in parallel)

```bash
# Terminal 1: Run tests for new package
go test -v ./pkg/newfeature/...

# Terminal 2: Run related tests
go test -v ./cmd/...
```

| Test Type | Location | Status |
|-----------|----------|--------|
| Happy path | `*_test.go` | [ ] |
| Error paths | `*_test.go` | [ ] |
| Edge cases | `*_test.go` | [ ] |
| Flag validation | `cmd/*_test.go` | [ ] |
| Config validation | `pkg/config/*_test.go` | [ ] |

### 3B: Flag Save/Restore (CRITICAL)

Prevent test pollution:
```go
func TestNewFeature(t *testing.T) {
    // Save all affected flags
    origFlag1 := flag1
    origFlag2 := flag2

    t.Cleanup(func() {
        flag1 = origFlag1
        flag2 = origFlag2
    })

    // Test code
}
```

**Common flags to save:**
- [ ] `outputFlag` / `updateOutputFlag`
- [ ] `dirFlag` / `updateDirFlag`
- [ ] `ruleFlag` / `updateRuleFlag`
- [ ] `typeFlag`
- [ ] `pmFlag`
- [ ] `dryRunFlag`
- [ ] `majorFlag` / `minorFlag` / `patchFlag`

### 3C: Integration Tests

| Requirement | Status |
|-------------|--------|
| Uses real testdata | [ ] |
| Named `Test*Integration*` | [ ] |
| Tests file parsing | [ ] |
| Tests lock resolution | [ ] |
| Tests all PMs if applicable | [ ] |

### 3D: Testdata

| Requirement | Status |
|-------------|--------|
| Real files (not fabricated) | [ ] |
| Located in `pkg/testdata/` | [ ] |
| Edge cases in `_edge-cases/` | [ ] |
| Lock files included | [ ] |
| All formats covered | [ ] |

---

## Phase 4: Quality Checks (Parallel)

Run all simultaneously:
```bash
# Terminal 1
go test ./... -count=1

# Terminal 2
go test -race ./...

# Terminal 3
make coverage-func

# Terminal 4
go vet ./... && make check
```

| Check | Command | Status |
|-------|---------|--------|
| Unit tests | `go test ./...` | [ ] |
| Race detection | `go test -race ./...` | [ ] |
| Coverage ≥97% | `make coverage-func` | [ ] |
| Static analysis | `go vet ./...` | [ ] |
| Linters | `make check` | [ ] |
| Formatting | `gofmt -s -w .` | [ ] |

---

## Phase 5: Documentation (Parallel)

Update docs simultaneously:

```bash
# Check all docs that might need updates
ls docs/user/cli.md docs/user/configuration.md docs/user/features.md docs/developer/
```

| Doc | Location | Status |
|-----|----------|--------|
| CLI reference | `docs/user/cli.md` | [ ] |
| Configuration | `docs/user/configuration.md` | [ ] |
| Features | `docs/user/features.md` | [ ] |
| Architecture | `docs/developer/architecture/` | [ ] |
| Help output | `cmd/*.go` flag descriptions | [ ] |

---

## Phase 6: Battle Testing (MANDATORY)

### 6A: Setup (Parallel)

```bash
export TEST_DIR=$(mktemp -d)
export GOUPDATE=/tmp/goupdate

# Build and clone in parallel
go build -o $GOUPDATE . &
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra &
git clone --depth 1 https://github.com/expressjs/express.git $TEST_DIR/express &
wait
```

### 6B: Test All Commands (Parallel where possible)

| Test | Command | Status |
|------|---------|--------|
| Scan | `$GOUPDATE scan -d $TEST_DIR/cobra` | [ ] |
| List | `$GOUPDATE list -d $TEST_DIR/cobra` | [ ] |
| Outdated | `$GOUPDATE outdated -d $TEST_DIR/cobra` | [ ] |
| Update dry-run | `$GOUPDATE update -d $TEST_DIR/cobra --dry-run` | [ ] |
| **ACTUAL Update** | `$GOUPDATE update -d $TEST_DIR/cobra --patch -y` | [ ] |
| Verify changes | `git -C $TEST_DIR/cobra diff` | [ ] |
| Rollback | `git -C $TEST_DIR/cobra checkout .` | [ ] |

### 6C: Test All Output Formats (Parallel)

```bash
$GOUPDATE list -d $TEST_DIR/cobra -o json | jq . &
$GOUPDATE list -d $TEST_DIR/cobra -o csv &
$GOUPDATE list -d $TEST_DIR/cobra -o xml &
wait
```

| Format | scan | list | outdated | update | Status |
|--------|------|------|----------|--------|--------|
| table | [ ] | [ ] | [ ] | [ ] | |
| json | [ ] | [ ] | [ ] | [ ] | |
| csv | [ ] | [ ] | [ ] | [ ] | |
| xml | [ ] | [ ] | [ ] | [ ] | |

### 6D: Test All Filters (if applicable)

| Filter | Test | Status |
|--------|------|--------|
| `-t prod` | `list -d $TEST_DIR/express -t prod` | [ ] |
| `-t dev` | `list -d $TEST_DIR/express -t dev` | [ ] |
| `-p js` | `list -d $TEST_DIR -p js` | [ ] |
| `-p golang` | `list -d $TEST_DIR -p golang` | [ ] |
| `-r npm` | `list -d $TEST_DIR -r npm` | [ ] |
| `-n package` | `list -d $TEST_DIR -n express` | [ ] |

### 6E: Test Package Managers (Parallel)

| PM | Test Project | scan | list | outdated | update |
|----|--------------|------|------|----------|--------|
| npm | express | [ ] | [ ] | [ ] | [ ] |
| golang | cobra | [ ] | [ ] | [ ] | [ ] |

---

## Phase 7: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] No races: `go test -race ./...`
- [ ] Coverage ≥97%: `make coverage-func`
- [ ] Clean git status
- [ ] Commit with descriptive message
- [ ] Progress report: `docs/agents-progress/YYYY-MM-DD_feature-name.md`
- [ ] Cleanup: `rm -rf $TEST_DIR`

---

## Commit Message Format

```
feat: Brief description of feature

- Added new flag --flag-name for X
- Added config field new_field for Y
- Updated output formats to include Z

Closes #XXX (if applicable)
```

---

## Parallel Execution Summary

### Can Run in Parallel
- Phase 1: All planning tasks
- Phase 3: Unit tests across packages
- Phase 4: All quality checks
- Phase 5: Documentation updates
- Phase 6A: Build + clone
- Phase 6B: Scan, list, outdated (read-only)
- Phase 6C: All format tests

### Must Run Sequentially
- Phase 6B: Actual update → verify → rollback
- Phase 7: Final verification
