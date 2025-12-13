# GoUpdate - Claude Code Instructions

## Project Overview
GoUpdate is a CLI tool for scanning, listing, and updating package dependencies across multiple ecosystems (npm, Go mod, pip, composer, etc.).

## Task-Specific Checklists

**Select checklist based on task type:**

| Task Type | Checklist |
|-----------|-----------|
| Adding new feature | `docs/checklists/feature-development.md` |
| Fixing a bug | `docs/checklists/bug-fix.md` |
| Refactoring code | `docs/checklists/refactoring.md` |
| Improving tests | `docs/checklists/test-improvement.md` |
| Validating CLI | `docs/testing-checklist.md` |

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

### 1. Plan First (Complex Tasks)
Before coding, identify:
- What files need modification
- Which tasks are independent (can run in parallel)
- Which tasks have dependencies (must run sequentially)
- Potential merge conflicts

### 2. Parallel Execution Strategy
Run independent operations simultaneously:
```bash
# Example: Run tests while cloning test projects
go test ./... &
git clone --depth 1 https://github.com/spf13/cobra.git /tmp/cobra &
wait
```

### 3. Testing Sequence
- `make test` before committing
- Battle test on real projects (use `docs/testing-checklist.md`)
- **CRITICAL**: Test actual updates, not just `--dry-run`

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
Use checklist: `docs/testing-checklist.md`
1. Clone real project to isolated `/tmp/test-*` directory
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
- See `AGENTS.md` for detailed procedures

## Parallel Work

For independent tasks, use separate directories or git worktrees:
```bash
# Separate temp directories for parallel battle testing
TEST_DIR_1=$(mktemp -d)
TEST_DIR_2=$(mktemp -d)

# Or git worktrees for parallel feature development
git worktree add ../goupdate-feature-a feature-a
git worktree add ../goupdate-feature-b feature-b
```

## Critical Warnings

- NEVER use `git reset --hard` or `git push --force` without explicit permission
- NEVER skip pre-commit hooks (`--no-verify`)
- NEVER commit credentials or .env files
- Test pollution: Always restore package-level flags in tests
