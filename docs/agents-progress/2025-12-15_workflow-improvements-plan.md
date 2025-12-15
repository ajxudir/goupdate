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
| 8 | Continue on partial failure - create PR but skip auto-merge | High | Medium |
| 9 | Tag reviewers only on partial failure (avoid spam) | Medium | Low |

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

## Feature 8: Continue on Partial Failure

### Problem
Currently, if some packages fail to update, the workflow may fail entirely. Users want:
- Continue updating remaining packages even if some fail
- Still create a PR with successful updates
- Skip auto-merge when there are partial failures (requires manual review)

### Solution

#### 8a. Track partial failure state

The `_goupdate-update` action already uses `--continue-on-fail`. We need to:
1. Detect partial failure (exit code 1 = partial, exit code 0 = success, exit code 2+ = complete failure)
2. Pass this state to subsequent jobs

```yaml
# In _goupdate-update action, add output for partial failure
outputs:
  partial-failure:
    description: 'Whether some packages failed to update'
    value: ${{ steps.update.outputs.partial_failure }}
```

```bash
# In update step
if [[ "$EXIT_CODE" -eq 1 ]]; then
  echo "partial_failure=true" >> "$GITHUB_OUTPUT"
  echo "::warning::Some packages failed to update. PR will be created but auto-merge is disabled."
else
  echo "partial_failure=false" >> "$GITHUB_OUTPUT"
fi
```

#### 8b. Skip auto-merge on partial failure

```yaml
# In merge-pr job condition
merge-pr:
  if: |
    needs.prepare.outputs.auto-merge-updates == 'true' &&
    needs.apply-updates.outputs.partial-failure != 'true' &&  # NEW: Skip on partial
    needs.apply-updates.outputs.pr-number != '' &&
    needs.apply-updates.outputs.has-changes == 'true'
```

#### 8c. Add PR label for partial failure

```yaml
# In apply-updates job, after PR creation
- name: Label PR on partial failure
  if: steps.goupdate.outputs.partial-failure == 'true'
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}
    PR_NUMBER: ${{ steps.pr.outputs.pr-number }}
  run: |
    gh pr edit "$PR_NUMBER" --add-label "partial-failure"
    echo "::warning::PR created with partial failures. Manual review required."
```

### Implementation Files
- `.github/actions/_goupdate-update/action.yml` - Add partial-failure output
- `.github/workflows/auto-update.yml` - Update merge-pr condition, add label step

---

## Feature 9: Tag Reviewers Only on Partial Failure

### Problem
Users don't want to be tagged on every successful auto-update (spam), but DO want to be notified when something fails and needs attention.

### Solution

#### 9a. Add conditional reviewer variables

```yaml
env:
  # PR Reviewer Configuration
  PR_REVIEWERS: ''                      # Always tag (leave empty for no spam)
  PR_FAILURE_REVIEWERS: 'user1,user2'   # Tag only on partial failure
  PR_ASSIGNEES: ''                      # Always assign
  PR_FAILURE_ASSIGNEES: ''              # Assign only on partial failure
```

#### 9b. Conditional tagging logic

```yaml
# In apply-updates job, after PR creation
- name: Add reviewers/assignees
  if: steps.goupdate.outputs.has-changes == 'true'
  env:
    GH_TOKEN: ${{ steps.app-token.outputs.token }}
    PR_NUMBER: ${{ steps.pr.outputs.pr-number }}
    PARTIAL_FAILURE: ${{ steps.goupdate.outputs.partial-failure }}
    PR_REVIEWERS: ${{ env.PR_REVIEWERS }}
    PR_FAILURE_REVIEWERS: ${{ env.PR_FAILURE_REVIEWERS }}
    PR_ASSIGNEES: ${{ env.PR_ASSIGNEES }}
    PR_FAILURE_ASSIGNEES: ${{ env.PR_FAILURE_ASSIGNEES }}
  run: |
    # Always add configured reviewers (if any)
    if [ -n "$PR_REVIEWERS" ]; then
      gh pr edit "$PR_NUMBER" --add-reviewer "$PR_REVIEWERS"
    fi

    # Add failure reviewers only on partial failure
    if [ "$PARTIAL_FAILURE" = "true" ] && [ -n "$PR_FAILURE_REVIEWERS" ]; then
      gh pr edit "$PR_NUMBER" --add-reviewer "$PR_FAILURE_REVIEWERS"
      echo "::notice::Tagged failure reviewers due to partial update failure"
    fi

    # Same logic for assignees
    if [ -n "$PR_ASSIGNEES" ]; then
      gh pr edit "$PR_NUMBER" --add-assignee "$PR_ASSIGNEES"
    fi

    if [ "$PARTIAL_FAILURE" = "true" ] && [ -n "$PR_FAILURE_ASSIGNEES" ]; then
      gh pr edit "$PR_NUMBER" --add-assignee "$PR_FAILURE_ASSIGNEES"
    fi
```

#### 9c. Update PR title/body on failure

```yaml
# Modify PR title to indicate failure
- name: Generate messages
  id: messages
  run: |
    # ... existing logic ...

    # Add failure indicator to title if partial failure
    if [ "$PARTIAL_FAILURE" = "true" ]; then
      TITLE="⚠️ $TITLE [PARTIAL FAILURE]"
    fi
    echo "pr_title=$TITLE" >> $GITHUB_OUTPUT
```

### Implementation Files
- `.github/workflows/auto-update.yml` - Add failure reviewer variables and tagging logic

---

## Implementation Order

### Phase 1: Workflow Variables (Quick Wins)
1. Add `major` option to workflow dispatch (#2)
2. Add `DEFAULT_UPDATE_TYPE` variable (#3)
3. Add `AUTO_MERGE_MAJOR` variable (#5)
4. Add `CREATE_PR_FOR_MAJOR` variable (#7)
5. Update branch naming for major (#4)

**Estimated effort:** 1-2 hours

### Phase 2: Partial Failure Handling (Critical)
6. Add `partial-failure` output to `_goupdate-update` action (#8)
7. Skip auto-merge on partial failure (#8)
8. Add PR label for partial failures (#8)
9. Add failure-specific reviewer/assignee variables (#9)
10. Implement conditional tagging logic (#9)

**Estimated effort:** 1-2 hours

### Phase 3: PR Reviewer Configuration
11. Add reviewer/assignee variables (#6)
12. Update `_gh-pr` action

**Estimated effort:** 30 minutes

### Phase 4: CLI Summary Improvements
13. Add `--summary` flag to CLI commands (#1)
14. Enhance JSON output with better summary
15. Update actions to use new summary output

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

  # NEW: PR Reviewer Configuration (always)
  PR_REVIEWERS: ''                          # Always tag (leave empty for no spam)
  PR_ASSIGNEES: ''                          # Always assign
  PR_MAJOR_REVIEWERS: ''                    # Reviewers for major PRs
  PR_MAJOR_ASSIGNEES: ''                    # Assignees for major PRs

  # NEW: PR Reviewer Configuration (on failure only - avoids spam)
  PR_FAILURE_REVIEWERS: ''                  # Tag ONLY on partial failure
  PR_FAILURE_ASSIGNEES: ''                  # Assign ONLY on partial failure

  # Existing
  PR_TITLE: 'GoUpdate: Auto update - {type} ({date})'
  COMMIT_MESSAGE: 'GoUpdate: Auto update - {type} ({date})'
```

---

## Behavior Matrix

| Scenario | Auto-Merge | Reviewers Tagged | PR Label |
|----------|------------|------------------|----------|
| All updates succeed (minor/patch) | Yes (if enabled) | `PR_REVIEWERS` only | None |
| All updates succeed (major) | `AUTO_MERGE_MAJOR` | `PR_REVIEWERS` + `PR_MAJOR_REVIEWERS` | None |
| Partial failure (some packages fail) | **No** | `PR_REVIEWERS` + `PR_FAILURE_REVIEWERS` | `partial-failure` |
| Complete failure (all fail) | N/A | N/A | No PR created |

---

## Notes

- All new variables have sensible defaults to maintain backward compatibility
- Existing workflows will continue to work without changes
- Major updates remain opt-in (not in scheduled runs unless DEFAULT_UPDATE_TYPE is changed)
- Auto-merge for major is disabled by default for safety
- **Partial failure handling**: PR is still created but auto-merge is skipped, failure reviewers are tagged
- **Spam prevention**: Use `PR_FAILURE_REVIEWERS` instead of `PR_REVIEWERS` to only be notified on failures
