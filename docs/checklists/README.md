# Checklists

Task-specific checklists for goupdate development.

## Available Checklists

| Checklist | When to Use |
|-----------|-------------|
| [Feature Development](feature-development.md) | Adding new features |
| [Bug Fix](bug-fix.md) | Fixing bugs |
| [Refactoring](refactoring.md) | Refactoring code |
| [Test Improvement](test-improvement.md) | Adding/improving tests |
| [Battle Testing](../testing-checklist.md) | CLI validation on real projects |

## Quick Selection Guide

```
Is this a new feature?
  └─ Yes → feature-development.md

Is this fixing a bug?
  └─ Yes → bug-fix.md

Is this refactoring existing code?
  └─ Yes → refactoring.md

Is this improving test coverage?
  └─ Yes → test-improvement.md

Is this validating the CLI works?
  └─ Yes → ../testing-checklist.md
```

## Common Across All Tasks

Regardless of task type, always:

1. **Plan first** - Identify scope and dependencies
2. **Test before commit** - `go test ./...`
3. **Check for races** - `go test -race ./...`
4. **Verify coverage** - `make coverage-func`
5. **Battle test** - Test on real projects
6. **Document** - Update progress report

## Checklist Usage in CLAUDE.md

These checklists are referenced in `/CLAUDE.md` and should be used based on task type:

```markdown
## Task Type Detection

When starting a task, identify the type:
- "Add X feature" → Use feature-development.md
- "Fix X bug" → Use bug-fix.md
- "Refactor X" → Use refactoring.md
- "Add tests for X" → Use test-improvement.md
- "Battle test" → Use testing-checklist.md
```
