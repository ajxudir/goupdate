# Agent Progress Logs

This directory tracks progress of tasks performed by agentic coding agents (Claude, Codex, Cursor, Copilot, etc.).

## Purpose

- Track work done by agents across sessions
- Provide visibility into multi-step tasks
- Enable continuation of work if session is interrupted
- Document decisions and changes made

## File Naming

```
YYYY-MM-DD_task-name.md
```

## Log Template

```markdown
# Task: [Task Name]

**Agent:** Claude/Codex/Other
**Date:** YYYY-MM-DD
**Branch:** feature/branch-name
**Status:** In Progress / Completed / Blocked

## Objective

Brief description of what was requested.

## Progress

- [x] Step completed
- [x] Another step completed
- [ ] Step pending
- [ ] Final step

## Files Modified

- path/to/file1.go
- path/to/file2.go

## Commits

- `abc1234` - Commit message 1
- `def5678` - Commit message 2

## Notes

Any observations, issues, or decisions made.

## Next Steps

If task is incomplete, what needs to be done next.
```

## Guidelines

1. **One log per task** - Don't mix unrelated work
2. **Update frequently** - Log progress as you go
3. **Be specific** - Include file paths, commit hashes
4. **Document decisions** - Explain why, not just what
5. **Clean up** - Archive completed logs periodically
