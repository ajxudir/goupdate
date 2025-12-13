# Testing Checklist Template

Use this checklist when battle testing or validating new features.
Copy this template to your progress report and check off items as completed.

## Pre-Testing Setup

- [ ] Build latest binary: `go build -o /tmp/goupdate .`
- [ ] Create isolated test directory: `TEST_DIR=$(mktemp -d)`
- [ ] Clone test projects (run in parallel):
  ```bash
  git clone --depth 1 https://github.com/spf13/cobra.git $TEST_DIR/cobra &
  git clone --depth 1 https://github.com/expressjs/express.git $TEST_DIR/express &
  wait
  ```

## Unit Tests

- [ ] `go test ./...` - All packages pass
- [ ] `go test -race ./...` - No data races detected
- [ ] `make test-unit` - Unit tests with race flag
- [ ] `make test-integration` - Integration tests pass

## Coverage

- [ ] `make coverage-func` - Check coverage percentages
- [ ] Coverage >= 97% overall
- [ ] No decrease in coverage for modified packages

## Static Analysis

- [ ] `go vet ./...` - No issues
- [ ] `make check` - Linters pass

## CLI Battle Testing

### Scan Command
- [ ] `goupdate scan -d $TEST_DIR/cobra` - Go project detected
- [ ] `goupdate scan -d $TEST_DIR/express` - JS project detected
- [ ] `goupdate scan --output json` - JSON format works
- [ ] `goupdate scan --output csv` - CSV format works
- [ ] `goupdate scan --output xml` - XML format works

### List Command
- [ ] `goupdate list -d $TEST_DIR/cobra` - Lists packages
- [ ] `goupdate list --type prod` - Filter by type works
- [ ] `goupdate list --type dev` - Filter by type works
- [ ] `goupdate list -p golang` - Filter by PM works
- [ ] `goupdate list --output json` - JSON format works

### Outdated Command
- [ ] `goupdate outdated -d $TEST_DIR/cobra` - Shows outdated versions
- [ ] `goupdate outdated --major` - Major filter works
- [ ] `goupdate outdated --minor` - Minor filter works
- [ ] `goupdate outdated --patch` - Patch filter works
- [ ] `goupdate outdated --output json` - JSON format works

### Update Command (CRITICAL)
- [ ] `goupdate update --dry-run` - Dry run shows plan
- [ ] **ACTUAL UPDATE**: `goupdate update --patch -y` - Performs update
- [ ] `git diff` - Verify manifest file modified correctly
- [ ] `git checkout .` - Rollback successful

## Output Format Verification

| Format | scan | list | outdated | update |
|--------|------|------|----------|--------|
| table  | [ ]  | [ ]  | [ ]      | [ ]    |
| json   | [ ]  | [ ]  | [ ]      | [ ]    |
| csv    | [ ]  | [ ]  | [ ]      | [ ]    |
| xml    | [ ]  | [ ]  | [ ]      | [ ]    |

## Error Handling

- [ ] Invalid path returns clear error
- [ ] No packages found shows informative message
- [ ] Network timeout handled gracefully
- [ ] Invalid config shows helpful error

## Workflow Commands (GitHub Actions)

- [ ] `make test-unit` - Same as CI
- [ ] `make test-integration` - Same as CI
- [ ] `make coverage-func` - Same as CI
- [ ] `go build ./...` - Build verification

## Documentation

- [ ] Examples in docs/*.md still work
- [ ] CLI --help output accurate
- [ ] Progress report updated in docs/agents-progress/

## Final Verification

- [ ] All tests pass: `go test ./... -count=1`
- [ ] Clean git status (no untracked test files)
- [ ] Changes committed with descriptive message
- [ ] Pushed to correct branch

## Issues Found

Document any issues discovered during testing:

| Issue | Severity | Status | Notes |
|-------|----------|--------|-------|
|       |          |        |       |

## Test Environment

- **Date**: YYYY-MM-DD
- **Go Version**:
- **OS**:
- **Test Projects**: cobra, express
- **Tester**: Claude/Human
