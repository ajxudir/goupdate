# Feature Development Checklist

Use this checklist when adding new features to goupdate.

---

## Phase 1: Planning

- [ ] Feature requirements documented
- [ ] Identify affected packages (cmd/, pkg/*)
- [ ] Check for existing similar functionality to extend
- [ ] Plan testdata requirements (real files, not mocks)
- [ ] Identify config fields needed (if any)

---

## Phase 2: Implementation

### Code Structure
- [ ] Follow existing package patterns
- [ ] Use appropriate package location:
  - `cmd/` - CLI commands and flags
  - `pkg/config/` - Configuration handling
  - `pkg/formats/` - Package file parsers (json, yaml, xml, raw)
  - `pkg/lock/` - Lock file resolution
  - `pkg/outdated/` - Version checking
  - `pkg/update/` - Update execution
  - `pkg/packages/` - Package detection

### Flags & Configuration
- [ ] New flags registered with cobra
- [ ] Flag defaults set appropriately
- [ ] Config fields added to `pkg/config/model.go`
- [ ] Config validation added if needed
- [ ] Flag save/restore pattern in tests (prevent pollution)

### Output Formats
- [ ] Table output implemented (default)
- [ ] JSON output implemented (`--output json`)
- [ ] CSV output implemented (`--output csv`)
- [ ] XML output implemented (`--output xml`)

---

## Phase 3: Testing

### Unit Tests
- [ ] Test file created: `*_test.go`
- [ ] Happy path tests
- [ ] Edge case tests
- [ ] Error handling tests
- [ ] Flag save/restore using `t.Cleanup()`

### Integration Tests
- [ ] Integration test created (if affects file parsing)
- [ ] Uses real testdata (no mocks)
- [ ] Test function named `Test*Integration*`

### Testdata
- [ ] Real package files added to `pkg/testdata/`
- [ ] Edge cases in `pkg/testdata/*/_edge-cases/`
- [ ] Lock files included where applicable
- [ ] README.md updated if new ecosystem

### Coverage
- [ ] `make coverage-func` - Check coverage
- [ ] New code >= 90% covered
- [ ] No decrease in overall coverage

---

## Phase 4: Quality Checks

### Static Analysis
- [ ] `go vet ./...` - No issues
- [ ] `gofmt -s -w .` - Code formatted
- [ ] `make check` - All checks pass

### Race Detection
- [ ] `go test -race ./...` - No races
- [ ] Concurrent access protected (if applicable)

### Chaos Testing
- [ ] Edge cases tested (empty, null, unicode)
- [ ] Invalid input handled gracefully
- [ ] No panics on malformed data

---

## Phase 5: Documentation

### Code Documentation
- [ ] Package-level doc comments
- [ ] Exported function doc comments
- [ ] Complex logic commented

### User Documentation
- [ ] `docs/user/cli.md` - CLI help updated
- [ ] `docs/user/configuration.md` - Config options documented
- [ ] `docs/user/features.md` - Feature documented

### Architecture Documentation
- [ ] `docs/developer/architecture/` - Architecture docs updated
- [ ] Design decisions documented

---

## Phase 6: Battle Testing

### CLI Testing (MANDATORY)
```bash
TEST_DIR=$(mktemp -d)
git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra
```

- [ ] `goupdate scan` - Feature works in scan
- [ ] `goupdate list` - Feature works in list
- [ ] `goupdate outdated` - Feature works in outdated
- [ ] `goupdate update --dry-run` - Dry run works
- [ ] **`goupdate update -y`** - ACTUAL update works (not just dry-run!)
- [ ] `git diff` - Changes verified
- [ ] All output formats tested (table, json, csv, xml)

### Error Scenarios
- [ ] Invalid input handled
- [ ] Missing dependencies handled
- [ ] Network errors handled (if applicable)

---

## Phase 7: Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] Race tests pass: `go test -race ./...`
- [ ] Coverage maintained: `make coverage-func`
- [ ] Clean git status
- [ ] Commit with descriptive message
- [ ] Progress report updated: `docs/agents-progress/`

---

## Cleanup

- [ ] Remove test directories: `rm -rf $TEST_DIR`
- [ ] No debug code left
- [ ] No TODO comments for this feature
