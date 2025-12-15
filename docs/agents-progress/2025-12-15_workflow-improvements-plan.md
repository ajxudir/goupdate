# Workflow and CLI Improvements Plan

**Date:** 2025-12-15
**Status:** Planning
**Branch:** claude/sync-workflow-actions-3xoEJ

## Overview

This plan addresses user feedback to improve the goupdate workflow and CLI for better usability in GitHub Actions workflows.

---

## Feature Requests Summary

| # | Feature | Priority | Complexity |
|---|---------|----------|------------|
| 1 | Human-readable CLI summaries | High | Medium |
| 2 | Major updates support in workflow dispatch | High | Low |
| 3 | Configurable default update scope for scheduled runs | Medium | Low |
| 4 | Separate branch handling for major updates | Medium | Medium |
| 5 | Auto-merge disabled by default for major | Medium | Low |
| 6 | PR reviewer/assignee configuration variables | Medium | Low |
| 7 | Option to disable PR creation for major (crash with warning) | Low | Low |

---

## Feature 1: Human-Readable CLI Summaries

### Problem
Current CLI uses JSON output for workflows, requiring complex jq parsing. Users want human-readable summaries like:
```
Summary: 33 updated, 19 up-to-date
         (13 have major updates still available)
```

### Solution

#### 1a. Add `--summary` flag to commands
Add a `--summary` flag that outputs a concise human-readable summary suitable for GitHub Actions.

**Example output for `outdated --summary`:**
```
Outdated packages: 45 total
- Major: 13 packages
- Minor: 22 packages
- Patch: 10 packages

Summary: 32 can auto-update (minor/patch), 13 require manual review (major only)
```

**Example output for `update --summary`:**
```
Update complete: 33 packages updated

Summary: 33 updated, 19 up-to-date
         (13 have major updates still available)
```

#### 1b. Enhance JSON output with summary section
Already exists: `summary.has_major`, `summary.has_minor`, etc.

Add additional fields:
```json
{
  "summary": {
    "total": 45,
    "to_update": 32,
    "up_to_date": 19,
    "has_major": 13,
    "has_minor": 22,
    "has_patch": 10,
    "major_only_count": 5,
    "auto_updatable": 27,
    "human_summary": "32 can auto-update, 13 require manual review"
  }
}
```

### Implementation Files
- `cmd/outdated.go` - Add `--summary` flag
- `cmd/update.go` - Add `--summary` flag
- `pkg/output/types.go` - Add summary generation
- `pkg/output/summary.go` - New file for summary formatting

---

## Feature 2: Major Updates Support in Workflow

### Problem
Current workflow dispatch only offers `minor` and `patch` options. Users need `major` for intentional upgrades.

### Solution

#### 2a. Add `major` option to workflow dispatch

```yaml
# auto-update.yml
inputs:
  update-type:
    description: 'Dependency update type'
    required: false
    default: 'minor'
    type: choice
    options:
      - minor   # Default, safe for automated weekly runs
      - patch
      - major   # NEW: Requires explicit selection
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add major option

---

## Feature 3: Configurable Default Update Scope

### Problem
Scheduled weekly runs always use `minor`. Users want to configure this.

### Solution

#### 3a. Add `DEFAULT_UPDATE_TYPE` environment variable

```yaml
env:
  # Update scope configuration
  DEFAULT_UPDATE_TYPE: 'minor'  # Options: patch, minor, major
```

#### 3b. Update schedule handler to use this variable

```yaml
case "$EVENT" in
  schedule)
    # Use configured default instead of hardcoded 'minor'
    echo "update_type=$DEFAULT_UPDATE_TYPE" >> $GITHUB_OUTPUT
    ;;
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add DEFAULT_UPDATE_TYPE env var

---

## Feature 4: Separate Branch for Major Updates

### Problem
Users want major updates on a separate branch (`goupdate/auto-update-major`) to keep them isolated.

### Solution

#### 4a. Update branch naming logic

```yaml
# In prepare job config step
case "$UPDATE_TYPE" in
  major)
    echo "update_branch_name=${BRANCH_PREFIX}/auto-update-major" >> $GITHUB_OUTPUT
    ;;
  minor)
    echo "update_branch_name=${BRANCH_PREFIX}/auto-update-minor" >> $GITHUB_OUTPUT
    ;;
  patch)
    echo "update_branch_name=${BRANCH_PREFIX}/auto-update-patch" >> $GITHUB_OUTPUT
    ;;
esac
```

#### 4b. Separate target branch for major (optional)
Add `MAJOR_UPDATE_TARGET_BRANCH` variable:

```yaml
env:
  UPDATE_TARGET_BRANCH: 'stage'           # Target for minor/patch
  MAJOR_UPDATE_TARGET_BRANCH: 'stage'     # Target for major (same by default)
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Update branch naming logic

---

## Feature 5: Auto-Merge Disabled by Default for Major

### Problem
Major updates may have breaking changes. Auto-merge should be disabled by default for major.

### Solution

#### 5a. Add `AUTO_MERGE_MAJOR` variable

```yaml
env:
  AUTO_MERGE_UPDATES: 'true'    # For minor/patch
  AUTO_MERGE_MAJOR: 'false'     # NEW: Disabled by default for major
```

#### 5b. Update merge-pr job condition

```yaml
# In apply-updates job, determine auto-merge based on update type
- name: Determine auto-merge setting
  id: auto-merge
  run: |
    UPDATE_TYPE="${{ needs.prepare.outputs.update-type }}"
    if [ "$UPDATE_TYPE" = "major" ]; then
      echo "enabled=${{ env.AUTO_MERGE_MAJOR }}" >> $GITHUB_OUTPUT
    else
      echo "enabled=${{ env.AUTO_MERGE_UPDATES }}" >> $GITHUB_OUTPUT
    fi
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add AUTO_MERGE_MAJOR variable and logic

---

## Feature 6: PR Reviewer/Assignee Configuration

### Problem
Users want to automatically tag reviewers/assignees on PRs.

### Solution

#### 6a. Add reviewer/assignee variables

```yaml
env:
  # PR Configuration
  PR_REVIEWERS: ''           # Comma-separated: 'user1,user2'
  PR_MAJOR_REVIEWERS: ''     # Reviewers for major PRs (empty = use PR_REVIEWERS)
  PR_ASSIGNEES: ''           # Comma-separated: 'user1,user2'
  PR_MAJOR_ASSIGNEES: ''     # Assignees for major PRs
```

#### 6b. Update _gh-pr action to support reviewers/assignees

```yaml
# In _gh-pr/action.yml
inputs:
  reviewers:
    description: 'Comma-separated list of reviewers'
    required: false
    default: ''
  assignees:
    description: 'Comma-separated list of assignees'
    required: false
    default: ''
```

```bash
# In _gh-pr action script
if [ -n "$REVIEWERS" ]; then
  gh pr edit "$PR_NUMBER" --add-reviewer "$REVIEWERS"
fi
if [ -n "$ASSIGNEES" ]; then
  gh pr edit "$PR_NUMBER" --add-assignee "$ASSIGNEES"
fi
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add reviewer/assignee variables
- `.github/actions/_gh-pr/action.yml` - Add reviewer/assignee inputs

---

## Feature 7: Disable PR Creation for Major (Crash with Warning)

### Problem
Some users prefer to not create PRs for major updates, keeping current behavior of failing the workflow.

### Solution

#### 7a. Add `CREATE_PR_FOR_MAJOR` variable

```yaml
env:
  CREATE_PR_FOR_MAJOR: 'true'   # Set to 'false' to skip PR and fail with warning
```

#### 7b. Update logic in apply-updates job

```yaml
- name: Check major PR creation policy
  if: needs.prepare.outputs.update-type == 'major' && env.CREATE_PR_FOR_MAJOR != 'true'
  run: |
    echo "::error::Major updates detected but CREATE_PR_FOR_MAJOR is disabled"
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "⚠️  MAJOR UPDATES REQUIRE MANUAL HANDLING"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo "Major updates were found but PR creation is disabled."
    echo "To enable PR creation for major updates, set CREATE_PR_FOR_MAJOR=true"
    echo ""
    exit 1
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add CREATE_PR_FOR_MAJOR variable

---

## Implementation Order

### Phase 1: Workflow Variables (Quick Wins)
1. Add `major` option to workflow dispatch (#2)
2. Add `DEFAULT_UPDATE_TYPE` variable (#3)
3. Add `AUTO_MERGE_MAJOR` variable (#5)
4. Add `CREATE_PR_FOR_MAJOR` variable (#7)
5. Update branch naming for major (#4)

**Estimated effort:** 1-2 hours

### Phase 2: PR Reviewer Configuration
6. Add reviewer/assignee variables (#6)
7. Update `_gh-pr` action

**Estimated effort:** 30 minutes

### Phase 3: CLI Summary Improvements
8. Add `--summary` flag to CLI commands (#1)
9. Enhance JSON output with better summary
10. Update actions to use new summary output

**Estimated effort:** 2-3 hours

---

## Configuration Summary (Final State)

```yaml
# .github/workflows/auto-update.yml
env:
  # Existing
  DEFAULT_BRANCH: 'main'
  BRANCH_PREFIX: 'goupdate'
  UPDATE_TARGET_BRANCH: 'stage'
  AUTO_MERGE_UPDATES: 'true'

  # NEW: Update Scope Configuration
  DEFAULT_UPDATE_TYPE: 'minor'              # Default for scheduled runs

  # NEW: Major Update Settings
  AUTO_MERGE_MAJOR: 'false'                 # Disabled by default
  CREATE_PR_FOR_MAJOR: 'true'               # Set false to fail instead
  MAJOR_UPDATE_TARGET_BRANCH: 'stage'       # Target branch for major

  # NEW: PR Configuration
  PR_REVIEWERS: ''                          # Default reviewers
  PR_MAJOR_REVIEWERS: ''                    # Reviewers for major PRs
  PR_ASSIGNEES: ''                          # Default assignees
  PR_MAJOR_ASSIGNEES: ''                    # Assignees for major PRs

  # Existing
  PR_TITLE: 'GoUpdate: Auto update - {type} ({date})'
  COMMIT_MESSAGE: 'GoUpdate: Auto update - {type} ({date})'
```

---

## Notes

- All new variables have sensible defaults to maintain backward compatibility
- Existing workflows will continue to work without changes
- Major updates remain opt-in (not in scheduled runs unless DEFAULT_UPDATE_TYPE is changed)
- Auto-merge for major is disabled by default for safety
