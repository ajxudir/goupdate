# Releasing Guide

This document describes the automated release workflow for goupdate.

## Overview

This project includes **reusable GitHub Actions** that can be copied to other Go projects. The workflows and composite actions in `.github/` are designed to be modular and configurable.

**What's included:**
- Automated dependency updates with PR-based workflow
- Release candidate (RC) creation with GoReleaser binaries
- Multi-arch Docker Hub image builds
- Configurable via environment variables

See [Reusing in Other Projects](#reusing-in-other-projects) for setup instructions and [Configuration Options](#configuration-options) for all available settings.

## Release Strategy

The release process uses GitHub's native release functionality with a two-branch strategy:

- **`stage` branch** - Development/staging, receives auto-updates and prereleases
- **`main` branch** - Production only, stable releases triggered by pushing version tags

## Branching Strategy

```
DEVELOPMENT (stage branch)
═══════════════════════════════════════════════════════════════════

Feature/Fix PR        Weekly Schedule
     │                      │
     ▼                      │
PR Workflow runs            │
(test + lint)               │
     │                      │
     ▼                      ▼
Merge to stage         Auto Update runs
     │                 (check deps)
     │                      │
     │                      ▼
     │              Create PR with updates
     │              (auto-merge enabled)
     │                      │
     └──────────────────────┤
                            ▼
                  Release workflow triggers
                  (push to stage)
                            │
                            ▼
         Prepare (generate RC tag) → Test
                            │
                ┌───────────┴───────────┐
                ▼                       ▼
           GoReleaser              Docker build
        (create prerelease)       (parallel)
        (_stage-YYYYMMDD-rcN)


                     ⬇️ When ready to release ⬇️


PRODUCTION (main branch)
═══════════════════════════════════════════════════════════════════

1. Merge stage → main
2. Create and push a tag:
   git tag v1.0.0 && git push origin v1.0.0
                │
                ▼
    Release workflow triggers
    → Prepare (use pushed tag) → Test
                │
    ┌───────────┴───────────┐
    ▼                       ▼
GoReleaser              Docker build
(create release with     (parallel)
 multi-arch binaries)
→ Push Docker images (latest, vX.Y.Z, vX.Y, vX)
```

## Workflows

### PR Workflow (`pr.yml`)

**Trigger:** Pull requests to `main` or `stage`

**Purpose:** Validate code before merge

**Jobs:**
1. Run tests
2. Run linter

### Auto Update (`auto-update.yml`)

**Trigger:**
- Weekly (Mondays at 00:00 UTC)
- Manual dispatch

**Purpose:** Check for dependency updates and create PRs

**Behavior by Trigger:**

| Trigger | Dependency Check | Apply Updates | Create PR |
|---------|------------------|---------------|-----------|
| **Schedule** (weekly) | ✅ Yes | ✅ If updates found | ✅ With auto-merge |
| **Manual: check-only** | ✅ Yes | ❌ No | ❌ No |
| **Manual: update** | ✅ Yes | ✅ If updates found | ✅ With auto-merge |

**Jobs:**
1. **Prepare** - Determine actions based on trigger
2. **Check Updates** - Check for dependency updates
3. **Apply Updates** - Apply updates and create PR (if updates found)
4. **Summary** - Generate workflow summary

**Dependency Update Flow:**
1. Check for available updates
2. Create feature branch (`goupdate/auto-update-minor`)
3. Apply updates with test validation
4. Create PR with auto-merge enabled
5. When PR merges to stage, Release workflow creates RC

### Release (`release.yml`)

**Trigger:**
- Push to `stage` branch (creates prerelease with binaries)
- Push tag matching `v*` (builds stable release with binaries)

**Purpose:** Create releases with GoReleaser binaries and Docker images

**Behavior by Trigger:**

| Trigger | GoReleaser | Docker Hub |
|---------|------------|------------|
| **Push to stage** | ✅ Creates prerelease with binaries | ✅ Builds RC image |
| **Tag push (v*)** | ✅ Creates release with binaries | ✅ Builds versioned images |

**Jobs:**
1. **Prepare** - Determine trigger type and generate tag
2. **Test** - Run tests
3. **GoReleaser** - Build binaries and create/update release (parallel with Docker)
4. **Docker** - Build and push Docker Hub images (parallel with GoReleaser)
5. **Summary** - Generate release summary

**Push to Stage Flow:**
1. Prepare generates RC tag (`_stage-YYYYMMDD-rcN`)
2. Run tests
3. GoReleaser creates prerelease with binaries (parallel with Docker)
4. Docker builds image with RC tags (parallel with GoReleaser)

**Tag Push Flow:**
1. Prepare uses pushed tag (e.g., `v1.2.3`)
2. Run tests
3. GoReleaser creates release with multi-arch binaries (parallel with Docker)
4. Docker builds with version tags (parallel with GoReleaser)

For prereleases: Docker tags are `_stage-YYYYMMDD-rcN` and `rc-latest`
For stable: Docker tags are `vX.Y.Z`, `X.Y`, `X`, and `latest`

## Tag Naming

### Release Candidate Tags

RC tags use the format: `_stage-YYYYMMDD-rcN`

- `_stage` - Underscore prefix ensures RCs sort **below** any real version
- `YYYYMMDD` - Date of creation
- `rcN` - Sequence number (rc1, rc2, rc3...) for multiple RCs on same day

Examples:
- `_stage-20241203-rc1` - First RC on Dec 3, 2024
- `_stage-20241203-rc2` - Second RC on Dec 3, 2024

### Stable Version Tags

Stable versions follow semantic versioning: `vX.Y.Z`

- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features)
- `v1.1.1` - Patch release (bug fixes)

### Version Detection

The application detects its version type at runtime:

| Version | Type | Warning |
|---------|------|---------|
| `dev` | Development build | "Development build: unreleased version" |
| `_stage-*` | Staging build | "Staging build from stage branch" |
| `v1.0.0` | Stable | No warning |

## Dependency Update Logic

The auto-update workflow applies updates intelligently:

| Update Type | Action |
|-------------|--------|
| Patch (1.0.0 → 1.0.1) | ✅ Auto-apply |
| Minor (1.0.0 → 1.1.0) | ✅ Auto-apply |
| Major (1.0.0 → 2.0.0) | ⚠️ Check for minor first |
| Major only available | ❌ Fail with notification |

## Required Secrets

Configure in GitHub Settings → Secrets → Actions:

| Secret | Purpose | Required |
|--------|---------|----------|
| `DOCKERHUB_USERNAME` | Docker Hub username | Only if using Docker |
| `DOCKERHUB_TOKEN` | Docker Hub access token | Only if using Docker |

`GITHUB_TOKEN` is automatically provided.

**Note:** If `BUILD_DOCKER: 'true'` but Docker secrets are missing, the workflow will **fail with clear instructions**. Set `BUILD_DOCKER: 'false'` to disable Docker builds entirely.

### Secrets and Environment Setup

**Required for core workflow (dependency checks, GoReleaser):**
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

**Only required for Docker Hub builds:**
- `DOCKERHUB_USERNAME` - Your Docker Hub username
- `DOCKERHUB_TOKEN` - Your Docker Hub access token

To skip Docker builds: Set `BUILD_DOCKER: 'false'` in release.yml.

## Reusing in Other Projects

The workflows and actions in this repository are designed to be **portable and reusable** across Go projects. They use composite actions (prefixed with `_`) that encapsulate common CI/CD tasks.

### Project Structure

```
.github/
├── actions/                    # Reusable composite actions
│   ├── _go-setup/             # Go environment setup with caching
│   ├── _go-test/              # Go test runner with options
│   ├── _goupdate/             # Dependency check and update
│   ├── _gh-release/           # GitHub release creation (non-Go projects)
│   ├── _gh-pr/                # PR creation with auto-merge
│   ├── _dockerhub/            # Docker Hub multi-arch builds
│   └── _goreleaser/           # GoReleaser builds (Go projects)
└── workflows/                  # Workflow definitions
    ├── auto-update.yml        # Dependency updates via PR
    ├── release.yml            # GoReleaser + Docker builds
    └── pr.yml                 # PR validation
```

### Quick Start

1. Copy `.github/actions/` directory to your project
2. Copy `.github/workflows/` directory to your project
3. Modify the `CONFIGURATION` section in each workflow (see [Configuration Options](#configuration-options))
4. Create a `stage` branch (or change `STAGE_BRANCH` in `release.yml`)

### Configuration Options

#### Auto Update Workflow (`auto-update.yml`)

```yaml
env:
  GO_VERSION: '1.24'                          # Go version to use
  UPDATE_BRANCH_NAME: 'goupdate/auto-update-minor'  # Branch for update PRs
  UPDATE_TARGET_BRANCH: 'stage'               # Target branch for PRs
  AUTO_MERGE_UPDATES: 'true'                  # Enable auto-merge for PRs
  DELETE_BRANCH_ON_MERGE: 'true'              # Delete branch after merge
  PR_TITLE: 'GoUpdate: Auto update - {type} ({date})'  # PR title template
  PR_REVIEWERS: ''                            # Reviewers for PRs
  COMMIT_MESSAGE: 'GoUpdate: Auto update - {type} ({date})'  # Commit message template
  TEST_COMMAND: 'make test'                   # Custom test command
  EXCLUDE_PACKAGES: ''                        # Packages to skip
```

**Message Placeholders (for PR_TITLE and COMMIT_MESSAGE):**
- `{date}` - Current date (YYYY-MM-DD)
- `{type}` - Update type based on workflow input:
  - `Minor` - Minor and patch updates only (default for scheduled runs)
  - `Patch` - Patch updates only
  - `All` - All updates including major (use with caution)

#### Release Workflow (`release.yml`)

```yaml
env:
  GO_VERSION: '1.24'                # Go version to use
  STAGE_BRANCH: 'stage'             # Branch that triggers prereleases
  RC_TAG_PREFIX: '_stage'           # Prefix for RC tags
  DOCKER_IMAGE_NAME: 'myapp'        # Docker Hub image name
  USE_GORELEASER: 'true'            # Set to 'false' to skip GoReleaser
  BUILD_DOCKER: 'true'              # Set to 'false' to skip Docker
  TEST_COMMAND: 'make test'         # Custom test command
```

**Important:** If `BUILD_DOCKER: 'true'` but Docker secrets are missing, the workflow will **fail with clear instructions** on how to fix it (either add secrets or disable Docker builds). This prevents silent failures.

### Branch Setup

If using a different development branch name:

1. In `release.yml`:
   - Change `branches: [stage]` under `push:` trigger
   - Update `STAGE_BRANCH: 'stage'` to your branch name

2. In `auto-update.yml`:
   - Update `UPDATE_TARGET_BRANCH: 'stage'` to your branch name

3. In `pr.yml`:
   - Add your branch to `branches: [main, stage]`

### Reusable Actions

Actions are prefixed with `_` to indicate they are internal composite actions. Each action is self-contained and can be used independently.

| Action | Purpose | Key Options |
|--------|---------|-------------|
| `_go-setup` | Set up Go with caching | `go-version`, `skip-download`, `skip-verify` |
| `_go-test` | Run Go tests with options | `race`, `coverage`, `timeout` |
| `_goupdate` | Check and apply dependency updates | `mode`, `update-type`, `exclude-packages` |
| `_goreleaser` | GoReleaser builds with prerelease support | `prerelease`, `tag`, `tag-prefix`, `latest` |
| `_gh-release` | Create GitHub releases (non-Go projects) | `prerelease`, `draft`, `tag`, `latest` |
| `_gh-pr` | Create PRs with auto-merge | `auto-merge`, `merge-method`, `delete-branch` |
| `_dockerhub` | Build multi-arch Docker Hub images | `platforms`, `provenance`, `sbom` |

**For complete documentation of all inputs, outputs, and examples, see [actions.md](actions.md).**

#### Using Actions Directly

You can use these actions in your own workflows:

```yaml
# Example: Just run tests with race detection
- uses: ./.github/actions/_go-test
  with:
    race: 'true'
    timeout: '10m'

# Example: Check for updates without applying
- uses: ./.github/actions/_goupdate
  with:
    mode: 'check'

# Example: GoReleaser prerelease (auto-generates tag)
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'true'
    tag-prefix: '_stage'

# Example: GoReleaser stable release (marked as latest)
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    tag: 'v1.0.0'
    latest: 'true'

# Example: Create a prerelease (non-Go projects)
- uses: ./.github/actions/_gh-release
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'true'

# Example: Create a stable release (non-Go projects)
- uses: ./.github/actions/_gh-release
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'false'
    tag: 'v1.0.0'
    latest: 'true'
```

See each action's `action.yml` for full documentation of available inputs and outputs.

## Complete CI/CD Pipeline

This project uses its own reusable actions for CI/CD (dogfooding). The complete pipeline includes:

1. **Auto Update** - Weekly dependency checks with auto-PR creation
2. **PR Validation** - Tests and linting on all PRs
3. **Release** - Prereleases on stage, stable releases via tags

### Pipeline Flow Diagram

```
AUTOMATED PIPELINE
═══════════════════════════════════════════════════════════════════

Weekly Schedule (Monday 00:00 UTC)
        │
        ▼
AUTO UPDATE WORKFLOW (.github/workflows/auto-update.yml)
  1. Check for dependency updates
  2. Apply updates + run tests
  3. Create PR → stage (auto-merge)
        │
        ▼
PR WORKFLOW (.github/workflows/pr.yml)
  • Run tests
  • Run linter
        │
        ▼ (auto-merge when checks pass)

RELEASE WORKFLOW (.github/workflows/release.yml)
  Trigger: push to stage
  1. Generate RC tag (_stage-YYYYMMDD)
  2. Run tests
  3. GoReleaser → prerelease
  4. Docker → rc-latest tag


                    ⬇️ Manual: When ready for production ⬇️


PRODUCTION RELEASE
═══════════════════════════════════════════════════════════════════

1. Merge stage → main
2. Create a stable release using one of these methods:
   • CLI: git tag v1.2.3 && git push origin v1.2.3
   • GitHub UI: Releases → Create new release → tag vX.Y.Z on main
        │
        ▼
RELEASE WORKFLOW (.github/workflows/release.yml)
  Trigger: tag push (v*)
  1. Run tests
  2. GoReleaser → stable release
     • Multi-arch binaries
     • Marked as "latest"
  3. Docker → latest + version tags
```

### Auto Update Workflow

The auto-update workflow (`auto-update.yml`) automates dependency management:

**Triggers:**
- Weekly schedule (configurable cron)
- Manual dispatch with mode selection

**Modes:**
| Mode | Description |
|------|-------------|
| `check-only` | Check for updates without applying |
| `update` | Check, apply updates, and create PR |

**Configuration:**
```yaml
env:
  GO_VERSION: '1.24'
  UPDATE_BRANCH_NAME: 'goupdate/auto-update-minor'
  UPDATE_TARGET_BRANCH: 'stage'
  AUTO_MERGE_UPDATES: 'true'
  DELETE_BRANCH_ON_MERGE: 'true'
  PR_TITLE: 'GoUpdate: Auto update - {type} ({date})'
  TEST_COMMAND: 'make test'
```

**Flow:**
1. Check stage branch for dependency updates
2. Create update branch with applied changes
3. Run tests to validate updates
4. Create PR to stage with auto-merge enabled
5. When PR merges → Release workflow creates prerelease

### How This Project Uses the Pipeline

goupdate uses its own actions (dogfooding):

1. **Weekly**: `auto-update.yml` checks for Go dependency updates
2. **Auto-PR**: Creates PR to `stage`
3. **Auto-merge**: PR merges when tests pass
4. **Prerelease**: `release.yml` creates `_stage-YYYYMMDD-rcN` release
5. **Docker**: Builds `rc-latest` image for testing
6. **Production**: Manual merge to `main` + tag push creates stable release

## Setting Up CI/CD for Your Project

This section provides a complete guide for setting up automated releases in your own project using the reusable actions.

### Step-by-Step Setup

#### 1. Copy the Actions

```bash
# Copy the actions directory to your project
cp -r .github/actions/ /path/to/your-project/.github/actions/
```

#### 2. Create Your Release Workflow

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    branches: [stage]  # Prerelease on push to stage
    tags:
      - 'v*'           # Stable release on version tags

env:
  STAGE_BRANCH: 'stage'
  RC_TAG_PREFIX: '_stage'

permissions:
  contents: write

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: make test  # Or: npm test, go test, etc.

  release:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Determine release type
        id: config
        run: |
          if [ "${{ github.ref_type }}" = "tag" ]; then
            echo "is_prerelease=false" >> $GITHUB_OUTPUT
            echo "is_stable=true" >> $GITHUB_OUTPUT
            echo "tag=${{ github.ref_name }}" >> $GITHUB_OUTPUT
          else
            echo "is_prerelease=true" >> $GITHUB_OUTPUT
            echo "is_stable=false" >> $GITHUB_OUTPUT
          fi

      # For Go projects: use GoReleaser
      - name: Release with GoReleaser
        uses: ./.github/actions/_goreleaser
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          tag: ${{ steps.config.outputs.tag }}
          tag-prefix: ${{ env.RC_TAG_PREFIX }}
          prerelease: ${{ steps.config.outputs.is_prerelease }}
          latest: ${{ steps.config.outputs.is_stable }}

      # For non-Go projects: use gh-release
      # - name: Create Release
      #   uses: ./.github/actions/_gh-release
      #   with:
      #     github-token: ${{ secrets.GITHUB_TOKEN }}
      #     tag: ${{ steps.config.outputs.tag }}
      #     tag-prefix: ${{ env.RC_TAG_PREFIX }}
      #     prerelease: ${{ steps.config.outputs.is_prerelease }}
      #     latest: ${{ steps.config.outputs.is_stable }}
```

#### 3. Create a Stage Branch

```bash
git checkout -b stage
git push -u origin stage
```

#### 4. Configure Branch Protection (Optional)

In GitHub repository settings:
- Protect `main` branch: require PR reviews
- Protect `stage` branch: require status checks to pass

#### 5. Add Auto Update Workflow (Optional)

For automated dependency updates, create `.github/workflows/auto-update.yml`:

```yaml
name: Auto Update

on:
  schedule:
    - cron: '0 0 * * 1'  # Weekly on Monday
  workflow_dispatch:
    inputs:
      mode:
        description: 'Operation mode'
        type: choice
        options:
          - check-only
          - update

env:
  UPDATE_BRANCH_NAME: 'deps/auto-update'
  UPDATE_TARGET_BRANCH: 'stage'
  AUTO_MERGE_UPDATES: 'true'
  TEST_COMMAND: 'make test'  # Or: npm test, go test, etc.

permissions:
  contents: write
  pull-requests: write

jobs:
  check-updates:
    runs-on: ubuntu-latest
    outputs:
      has-updates: ${{ steps.check.outputs.has-updates }}
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ env.UPDATE_TARGET_BRANCH }}

      # For Go projects
      - uses: ./.github/actions/_goupdate
        id: check
        with:
          mode: 'check'

  apply-updates:
    needs: check-updates
    if: |
      github.event.inputs.mode != 'check-only' &&
      needs.check-updates.outputs.has-updates == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          ref: ${{ env.UPDATE_TARGET_BRANCH }}
          fetch-depth: 0

      - name: Create update branch
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git checkout -b "${{ env.UPDATE_BRANCH_NAME }}"

      - uses: ./.github/actions/_goupdate
        id: update
        with:
          mode: 'update'
          test-command: ${{ env.TEST_COMMAND }}

      - name: Commit and push
        if: steps.update.outputs.has-changes == 'true'
        run: |
          git add -A
          git commit -m "chore(deps): update dependencies"
          git push -u origin "${{ env.UPDATE_BRANCH_NAME }}"

      - uses: ./.github/actions/_gh-pr
        if: steps.update.outputs.has-changes == 'true'
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          title: 'chore(deps): Update dependencies'
          base: ${{ env.UPDATE_TARGET_BRANCH }}
          head: ${{ env.UPDATE_BRANCH_NAME }}
          auto-merge: ${{ env.AUTO_MERGE_UPDATES }}
          delete-branch: 'true'
```

This creates a complete automation loop:
- Weekly check for updates
- Auto-PR to stage branch
- Auto-merge when tests pass
- Release workflow creates prerelease

### Prerelease via Draft Release (Alternative Flow)

If you prefer creating releases via GitHub UI instead of pushing tags:

1. Create a **draft prerelease** on main branch via GitHub UI
2. Set it as prerelease
3. When ready, push a version tag - GoReleaser will:
   - Update the existing release in place
   - Remove the prerelease flag
   - Mark it as latest

This workflow is configured by default when using `latest: ${{ steps.config.outputs.is_stable }}`.

## Platform Portability

The release actions use standard CLI tools that work on any CI/CD platform. While the examples use GitHub Actions YAML syntax, the underlying commands are portable.

### Core Commands Used

| Action | Core Commands |
|--------|---------------|
| `_goreleaser` | `goreleaser release`, `git tag`, `git push` |
| `_gh-release` | `gh release create`, `git tag`, `git push` |
| `_dockerhub` | `docker buildx build`, `docker push` |
| `_goupdate` | `goupdate update`, `git commit` |

### GitLab CI Example

```yaml
# .gitlab-ci.yml
stages:
  - test
  - release

variables:
  RC_TAG_PREFIX: "_stage"

test:
  stage: test
  script:
    - make test

release:prerelease:
  stage: release
  only:
    - stage
  script:
    - TODAY=$(date -u +%Y%m%d)
    - TAG="${RC_TAG_PREFIX}-${TODAY}-rc1"
    - git tag -a "$TAG" -m "Release Candidate $TAG"
    - git push origin "$TAG"
    - goreleaser release --clean

release:stable:
  stage: release
  only:
    - tags
  script:
    - goreleaser release --clean
  variables:
    GORELEASER_MAKE_LATEST: "true"
```

### Bitbucket Pipelines Example

```yaml
# bitbucket-pipelines.yml
pipelines:
  branches:
    stage:
      - step:
          name: Test
          script:
            - make test
      - step:
          name: Prerelease
          script:
            - TODAY=$(date -u +%Y%m%d)
            - TAG="_stage-${TODAY}-rc1"
            - git tag -a "$TAG" -m "Release Candidate $TAG"
            - git push origin "$TAG"
            - goreleaser release --clean

  tags:
    'v*':
      - step:
          name: Test
          script:
            - make test
      - step:
          name: Release
          script:
            - export GORELEASER_MAKE_LATEST=true
            - goreleaser release --clean
```

### Azure Pipelines Example

```yaml
# azure-pipelines.yml
trigger:
  branches:
    include:
      - stage
  tags:
    include:
      - 'v*'

stages:
  - stage: Test
    jobs:
      - job: Test
        pool:
          vmImage: 'ubuntu-latest'
        steps:
          - script: make test

  - stage: Release
    dependsOn: Test
    jobs:
      - job: Prerelease
        condition: eq(variables['Build.SourceBranch'], 'refs/heads/stage')
        steps:
          - script: |
              TODAY=$(date -u +%Y%m%d)
              TAG="_stage-${TODAY}-rc1"
              git tag -a "$TAG" -m "Release Candidate $TAG"
              git push origin "$TAG"
              goreleaser release --clean

      - job: StableRelease
        condition: startsWith(variables['Build.SourceBranch'], 'refs/tags/v')
        steps:
          - script: |
              export GORELEASER_MAKE_LATEST=true
              goreleaser release --clean
```

### Key Environment Variables

These environment variables work with GoReleaser across all platforms:

| Variable | Description |
|----------|-------------|
| `GORELEASER_CURRENT_TAG` | Override the release tag |
| `GORELEASER_MAKE_LATEST` | Mark release as latest (true/false) |
| `GORELEASER_RELEASE_DRAFT` | Create as draft (true/false) |
| `GORELEASER_RELEASE_PRERELEASE` | Mark as prerelease (true/false) |
| `GITHUB_TOKEN` | GitHub authentication token |

### Non-Go Projects

For projects without Go, use platform-native release commands:

```bash
# Create release with GitHub CLI (works on any CI)
gh release create "v1.0.0" \
  --title "v1.0.0" \
  --generate-notes \
  --latest

# Create prerelease
gh release create "_stage-20241203-rc1" \
  --prerelease \
  --generate-notes
```

## Troubleshooting

### RC Not Created on Push

If RC wasn't created when pushing to stage:
- Check that the Release workflow ran
- Check test results - tests must pass
- Check GoReleaser logs for errors

### PR Not Created

If dependency update PR wasn't created:
- No updates were available
- All available updates are major-only (requires manual review)
- Tests failed after applying updates

### Docker Build Failed

If Docker builds fail, check:
- `BUILD_DOCKER` is set to `'true'`
- `DOCKERHUB_USERNAME` secret is configured
- `DOCKERHUB_TOKEN` secret is configured

The workflow provides clear error messages with instructions when secrets are missing.

### GoReleaser Failed

If GoReleaser fails:
- Check `.goreleaser.yml` exists and is valid
- Check test results passed
- Review GoReleaser logs for build errors

### Auto-Merge Not Working

If auto-merge doesn't enable:
- Ensure "Allow auto-merge" is enabled in repository settings
- The `GITHUB_TOKEN` needs sufficient permissions
- PR must pass all required status checks first

### Tag Already Exists

The workflow will fail if the target version tag already exists. Choose a different version number.
