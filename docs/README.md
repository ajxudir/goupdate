# Documentation

This directory contains all documentation for goupdate, organized by audience.

## Structure

```
docs/
├── user/            # User-facing documentation
├── developer/       # Developer & contributor documentation
├── checklists/      # Task-specific checklists
├── agents-progress/ # Agent task progress logs
└── internal/        # Internal testing documents
```

## User Documentation (`user/`)

Documentation for goupdate users:

| File | Description |
|------|-------------|
| [cli.md](user/cli.md) | Command-line interface reference |
| [configuration.md](user/configuration.md) | Configuration file format and options |
| [features.md](user/features.md) | Feature overview |
| [actions.md](user/actions.md) | GitHub Actions integration |
| [system-tests.md](user/system-tests.md) | Running tests after updates |
| [private-registries.md](user/private-registries.md) | Working with private package registries |
| [comparison.md](user/comparison.md) | Comparison with other tools |
| [troubleshooting.md](user/troubleshooting.md) | Common issues and solutions |

## Developer Documentation (`developer/`)

Documentation for contributors and maintainers:

| File | Description |
|------|-------------|
| [architecture/](developer/architecture/) | Codebase architecture documentation |
| [testing.md](developer/testing.md) | Test suite overview and guidelines |
| [releasing.md](developer/releasing.md) | Release process and versioning |

## Checklists (`checklists/`)

Task-specific checklists for development:

| File | Description |
|------|-------------|
| [README.md](checklists/README.md) | Checklist selection guide |
| [feature-development.md](checklists/feature-development.md) | Adding new features |
| [bug-fix.md](checklists/bug-fix.md) | Fixing bugs |
| [refactoring.md](checklists/refactoring.md) | Refactoring code |
| [test-improvement.md](checklists/test-improvement.md) | Improving test coverage |
| [test-battle.md](checklists/test-battle.md) | CLI battle testing |
| [test-chaos.md](checklists/test-chaos.md) | Chaos testing coverage |

## Agent Progress (`agents-progress/`)

Progress logs for AI coding agents (Claude, Codex, etc.):

| File | Description |
|------|-------------|
| [README.md](agents-progress/README.md) | Log format and guidelines |
| `YYYY-MM-DD_task-name.md` | Individual task logs |

## Internal Documentation (`internal/`)

Internal tracking documents (not user-facing):

| File | Description |
|------|-------------|
| [testing-progress.md](internal/testing-progress.md) | Testing coverage progress |
| [chaos-testing.md](internal/chaos-testing.md) | Detailed chaos testing plan |
