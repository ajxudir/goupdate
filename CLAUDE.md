# GoUpdate - Claude Code Instructions

## Project Overview
GoUpdate is a CLI tool for scanning, listing, and updating package dependencies across multiple ecosystems (npm, Go mod, pip, composer, etc.).

## Quick Reference

### Build & Test Commands
```bash
make build          # Build binary
make test           # Run all tests
make test-unit      # Unit tests with -race flag
make test-integration # Integration tests only
make coverage-func  # Coverage report
make check          # Run linters
go test ./...       # Fast test run
```

### CLI Commands
```bash
goupdate scan -d <path>      # Detect package files
goupdate list -d <path>      # List packages
goupdate outdated -d <path>  # Show outdated packages
goupdate update -d <path>    # Update packages
```

## IMPORTANT: Task Workflow

1. **Plan first** - For complex tasks, plan the approach before coding
2. **Run tests** - Always run `make test` before committing
3. **Check coverage** - Maintain 100% branch coverage on modified code
4. **Battle test** - Test CLI on real projects (not just unit tests)

## Code Conventions

- Go 1.21+ required
- Use `t.Cleanup()` for test teardown (not defer for flag restoration)
- Package-level flags must be saved/restored in tests to prevent pollution
- Real testdata only - no mock registries or fake version catalogs

## Testing Requirements

### Before Committing
- `go test ./...` must pass
- `go test -race ./...` must pass (no data races)
- Coverage must not decrease

### Battle Testing (MANDATORY for new features)
1. Clone real project: `git clone --depth 1 <repo> /tmp/test`
2. Test all commands: scan, list, outdated, update
3. **CRITICAL**: Test actual updates (not just `--dry-run`)
4. Verify changes with `git diff`
5. Test all output formats: table, json, csv, xml

## File Structure

```
cmd/           # CLI commands
pkg/           # Core packages
  config/      # Configuration loading
  formats/     # Package file parsers
  lock/        # Lock file handling
  outdated/    # Version checking
  update/      # Update logic
testdata/      # Test fixtures (real files only)
docs/          # Documentation
```

## Progress Tracking

- Log progress in `docs/agents-progress/YYYY-MM-DD_task-name.md`
- Use the checklist at `docs/testing-checklist.md` for validation

## Parallel Work

For independent tasks, use separate git worktrees:
```bash
git worktree add ../goupdate-feature-a feature-a
git worktree add ../goupdate-feature-b feature-b
```

## Critical Warnings

- NEVER use `git reset --hard` or `git push --force` without explicit permission
- NEVER skip pre-commit hooks (`--no-verify`)
- NEVER commit credentials or .env files
- Test pollution: Always restore package-level flags in tests
