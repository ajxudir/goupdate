# GitHub Actions Reference

This document provides complete reference documentation for all reusable GitHub Actions included in this project. These actions are located in `.github/actions/` and can be copied to other projects.

## Table of Contents

- [Overview](#overview)
- [Actions](#actions)
  - [_dockerhub](#_dockerhub)
  - [_gh-pr](#_gh-pr)
  - [_gh-release](#_gh-release)
  - [_go-setup](#_go-setup)
  - [_go-test](#_go-test)
  - [_goreleaser](#_goreleaser)
  - [_goupdate](#_goupdate)
- [Examples](#examples)
  - [Custom Docker Image Names](#custom-docker-image-names)
  - [Multi-Registry Docker Builds](#multi-registry-docker-builds)
  - [PR with Auto-Merge](#pr-with-auto-merge)
  - [Dependency Updates with Custom Tests](#dependency-updates-with-custom-tests)

---

## Overview

All actions are composite actions (prefixed with `_`) that encapsulate common CI/CD tasks. Each action is self-contained and can be used independently or combined in workflows.

**Project Structure:**
```
.github/
├── actions/
│   ├── _dockerhub/      # Docker Hub multi-arch builds
│   ├── _gh-pr/          # PR creation with auto-merge
│   ├── _gh-release/     # GitHub release creation
│   ├── _go-setup/       # Go environment setup with caching
│   ├── _go-test/        # Go test runner with options
│   ├── _goreleaser/     # GoReleaser builds
│   └── _goupdate/       # Dependency check and update
└── workflows/
    ├── auto-update.yml  # Dependency updates via PR
    ├── pr.yml           # PR validation
    └── release.yml      # GoReleaser + Docker builds
```

---

## Actions

### _dockerhub

Builds and pushes multi-architecture Docker images to Docker Hub or other container registries.

**Location:** `.github/actions/_dockerhub/`

**Features:**
- Multi-arch builds (amd64, arm64)
- GitHub Actions cache integration
- Provenance and SBOM generation
- Multi-stage build support
- Supports any OCI-compliant registry

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `registry` | Container registry (docker.io, ghcr.io, etc.) | No | `docker.io` |
| `username` | Registry username | **Yes** | - |
| `password` | Registry password or token | **Yes** | - |
| `image-name` | Image name without registry prefix | **Yes** | - |
| `tags` | Image tags (newline or comma separated) | **Yes** | - |
| `platforms` | Target platforms | No | `linux/amd64,linux/arm64` |
| `dockerfile` | Path to Dockerfile | No | `./Dockerfile` |
| `context` | Build context | No | `.` |
| `build-args` | Build arguments (KEY=VALUE, newline separated) | No | `''` |
| `push` | Push to registry | No | `true` |
| `cache` | Use GitHub Actions cache | No | `true` |
| `target` | Multi-stage build target | No | `''` |
| `labels` | Additional OCI labels (KEY=VALUE, newline separated) | No | `''` |
| `provenance` | Generate provenance attestation | No | `false` |
| `sbom` | Generate SBOM (Software Bill of Materials) | No | `false` |
| `load` | Load image to local Docker daemon (for testing, disables multi-arch) | No | `false` |
| `secrets` | Build secrets (id=secret_value, newline separated) | No | `''` |
| `no-cache` | Disable all caching | No | `false` |

#### Outputs

| Output | Description |
|--------|-------------|
| `digest` | Image digest |
| `metadata` | Image metadata JSON |
| `tags` | Applied tags |
| `imageid` | Image ID |

#### Examples

**Basic Docker Hub push:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: myorg/myapp
    tags: |
      latest
      v1.0.0
```

**Custom image name with version tags:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: mycompany/my-custom-app
    tags: |
      ${{ github.sha }}
      ${{ github.ref_name }}
      latest
```

**Push to GitHub Container Registry (ghcr.io):**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
    image-name: ${{ github.repository }}
    tags: latest
```

**Build with custom Dockerfile and build args:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: myorg/myapp
    tags: latest
    dockerfile: ./docker/Dockerfile.prod
    context: ./src
    build-args: |
      NODE_ENV=production
      API_VERSION=v2
```

**Multi-stage build with specific target:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: myorg/myapp
    tags: latest
    target: production
```

**Single platform for testing:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: myorg/myapp
    tags: test
    platforms: linux/amd64
    load: 'true'
    push: 'false'
```

**With provenance and SBOM:**
```yaml
- uses: ./.github/actions/_dockerhub
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}
    image-name: myorg/myapp
    tags: latest
    provenance: 'true'
    sbom: 'true'
```

---

### _gh-pr

Creates GitHub pull requests with auto-merge support.

**Location:** `.github/actions/_gh-pr/`

**Features:**
- Create PRs with assignees, reviewers
- Enable auto-merge with configurable merge method
- Delete branch after merge option
- Support for draft PRs

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `github-token` | GitHub token for creating PR (needs repo and write:discussion permissions for auto-merge) | **Yes** | - |
| `title` | PR title | **Yes** | - |
| `body` | PR body content | No | `''` |
| `base` | Base branch to merge into | No | `main` |
| `head` | Head branch with changes (default: current branch) | No | `''` |
| `assignees` | Assignees (comma-separated usernames) | No | `''` |
| `reviewers` | Reviewers (comma-separated usernames) | No | `''` |
| `team-reviewers` | Team reviewers (comma-separated team slugs) | No | `''` |
| `draft` | Create as draft PR | No | `false` |
| `auto-merge` | Enable auto-merge when checks pass | No | `false` |
| `merge-method` | Merge method: squash, merge, or rebase | No | `squash` |
| `delete-branch` | Delete head branch after merge (requires auto-merge) | No | `false` |

#### Outputs

| Output | Description |
|--------|-------------|
| `pr-number` | Created PR number |
| `pr-url` | URL of the created PR |
| `pr-state` | State of the PR (open, closed, merged) |
| `auto-merge-enabled` | Whether auto-merge was enabled |

#### Examples

**Basic PR:**
```yaml
- uses: ./.github/actions/_gh-pr
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    title: 'feat: Add new feature'
    body: 'This PR adds a new feature.'
```

**PR with auto-merge and squash:**
```yaml
- uses: ./.github/actions/_gh-pr
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    title: 'chore(deps): Update dependencies'
    body: 'Automated dependency update'
    base: stage
    auto-merge: 'true'
    merge-method: squash
    delete-branch: 'true'
```

**PR with reviewers and assignees:**
```yaml
- uses: ./.github/actions/_gh-pr
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    title: 'fix: Critical bug fix'
    reviewers: 'alice,bob'
    assignees: 'carol'
```

---

### _gh-release

Creates GitHub releases with support for both prereleases and stable releases.

**Location:** `.github/actions/_gh-release/`

**Features:**
- Auto-generate prerelease tags (`_stage-YYYYMMDD-rcN`)
- Upload release assets
- Generate release notes from commits
- Create discussion category

#### Tag Generation

When `tag` input is empty and `prerelease` is true:
- Format: `_stage-YYYYMMDD-rcN`
- Underscore prefix ensures RCs sort below real versions
- Sequence number (rc1, rc2, etc.) auto-increments

For stable releases, you must provide an explicit `tag` input.

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `github-token` | GitHub token for creating release | **Yes** | - |
| `prerelease` | Mark as prerelease | No | `true` |
| `draft` | Create as draft release | No | `false` |
| `tag` | Explicit tag name (skips auto-generation for prereleases) | No | `''` |
| `tag-prefix` | Tag prefix for auto-generated prerelease tags | No | `_stage` |
| `body` | Release body content | No | `''` |
| `generate-notes` | Auto-generate release notes from commits | No | `true` |
| `target-commitish` | Target branch or commit SHA (default: current HEAD) | No | `''` |
| `files` | Assets to upload (glob pattern, newline-separated) | No | `''` |
| `discussion-category` | Create discussion in this category | No | `''` |
| `latest` | Mark as latest release (true, false, or auto) | No | `auto` |

**Note:** Release title defaults to the tag name when not specified by GitHub.

#### Outputs

| Output | Description |
|--------|-------------|
| `tag` | The created tag name |
| `release-url` | URL of the created release |
| `release-id` | ID of the created release |
| `date` | Date component of the tag (for prereleases) |
| `sequence` | Sequence number (for prereleases) |

#### Examples

**Create prerelease with auto-generated tag:**
```yaml
- uses: ./.github/actions/_gh-release
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'true'
```

**Create stable release with explicit tag:**
```yaml
- uses: ./.github/actions/_gh-release
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'false'
    tag: 'v1.0.0'
    latest: 'true'
```

**Release with file attachments:**
```yaml
- uses: ./.github/actions/_gh-release
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    tag: 'v1.0.0'
    files: |
      dist/*.tar.gz
      dist/*.zip
      checksums.txt
```

---

### _go-setup

Sets up Go environment with module caching.

**Location:** `.github/actions/_go-setup/`

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `go-version` | Go version to install | No | `1.24` |
| `working-directory` | Working directory | No | `.` |
| `skip-download` | Skip go mod download (for vendored dependencies) | No | `false` |
| `skip-verify` | Skip go mod verify (faster but less safe) | No | `false` |

#### Outputs

| Output | Description |
|--------|-------------|
| `go-version` | Installed Go version |
| `cache-hit` | Whether cache was hit |

#### Examples

**Basic setup:**
```yaml
- uses: ./.github/actions/_go-setup
```

**Custom Go version:**
```yaml
- uses: ./.github/actions/_go-setup
  with:
    go-version: '1.23'
```

**With vendored dependencies:**
```yaml
- uses: ./.github/actions/_go-setup
  with:
    skip-download: 'true'
```

---

### _go-test

Runs Go tests with configurable options.

**Location:** `.github/actions/_go-test/`

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `working-directory` | Working directory | No | `.` |
| `race` | Enable race detector | No | `true` |
| `coverage` | Generate coverage report | No | `false` |
| `coverage-file` | Coverage output file | No | `coverage.out` |
| `packages` | Packages to test | No | `./...` |
| `verbose` | Verbose output | No | `true` |
| `timeout` | Test timeout | No | `10m` |

#### Outputs

| Output | Description |
|--------|-------------|
| `passed` | Whether all tests passed |
| `coverage-file` | Path to coverage file if generated |

#### Examples

**Basic test run:**
```yaml
- uses: ./.github/actions/_go-test
```

**With coverage:**
```yaml
- uses: ./.github/actions/_go-test
  with:
    coverage: 'true'
    coverage-file: 'coverage.out'
```

**Specific packages with longer timeout:**
```yaml
- uses: ./.github/actions/_go-test
  with:
    packages: './pkg/...'
    timeout: '30m'
```

---

### _goreleaser

Runs GoReleaser to build and release Go binaries.

**Location:** `.github/actions/_goreleaser/`

**Features:**
- Creates GitHub releases with multi-arch binaries
- Supports prereleases with auto-generated tags
- Configurable release title and notes
- Snapshot mode for testing

#### Tag Generation

When `tag` input is empty and `prerelease` is true:
- Format: `_stage-YYYYMMDD-rcN`
- Auto-increments sequence number for same day

For stable releases, you must provide an explicit `tag` input.

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `github-token` | GitHub token for releases | **Yes** | - |
| `version` | GoReleaser version | No | `~> v2` |
| `args` | GoReleaser arguments | No | `release --clean` |
| `tag` | Explicit tag to release (auto-generated for prereleases if empty) | No | `''` |
| `tag-prefix` | Tag prefix for auto-generated prerelease tags | No | `_stage` |
| `prerelease` | Mark as prerelease | No | `false` |
| `draft` | Create as draft release | No | `false` |
| `latest` | Mark as latest release (true, false, or auto) | No | `auto` |
| `working-directory` | Working directory | No | `.` |
| `snapshot` | Build snapshot (no release created) | No | `false` |
| `config-file` | Path to GoReleaser config file | No | `.goreleaser.yml` |
| `distribution` | GoReleaser distribution (goreleaser or goreleaser-pro) | No | `goreleaser` |
| `parallelism` | Number of parallel builds (empty for auto) | No | `''` |
| `skip-announce` | Skip announce step | No | `false` |
| `skip-validate` | Skip validation step | No | `false` |

**Note:** Release title defaults to the tag name when not specified.

#### Outputs

| Output | Description |
|--------|-------------|
| `tag` | The release tag |
| `release-url` | URL of the created release |
| `artifacts` | Built artifacts JSON |
| `metadata` | Release metadata JSON |

#### Examples

**Create prerelease with auto-generated tag:**
```yaml
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'true'
```

**Create stable release:**
```yaml
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    tag: 'v1.0.0'
    latest: 'true'
```

**Snapshot build for testing:**
```yaml
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    snapshot: 'true'
```

**Custom tag prefix:**
```yaml
- uses: ./.github/actions/_goreleaser
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    prerelease: 'true'
    tag-prefix: '_rc'
```

---

### _goupdate

Go dependency management using goupdate (check, update, or both).

**Location:** `.github/actions/_goupdate/`

**Features:**
- Check for dependency updates
- Apply updates with test validation
- Auto-commit changes
- Intelligent handling of major-only updates

**Error Handling:**
- Uses `--continue-on-fail` to show all package errors before failing
- Captures exit codes from goupdate commands
- Step fails with non-zero exit code if any errors occurred
- Errors are displayed in workflow output for visibility

**Behavior in goupdate repository:** Builds from source (dogfooding)
**Behavior in other repositories:** Downloads latest release from GitHub

#### Modes

| Mode | Description |
|------|-------------|
| `check` | Only check for updates (no changes) |
| `update` | Apply updates without checking first |
| `check-and-update` | Check and apply updates (default) |

#### Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `mode` | Operation mode: check, update, or check-and-update | No | `check-and-update` |
| `working-directory` | Working directory containing go.mod | No | `.` |
| `update-type` | Type of updates: none (use constraints), patch, minor, or all (includes major) | No | `minor` |
| `test-command` | Test command to run after updates (use "skip" to skip) | No | `go test -race ./...` |
| `fail-on-major-only` | Exit with error if only major updates available | No | `true` |
| `exclude-packages` | Packages to skip (comma-separated) | No | `''` |
| `commit-changes` | Auto-commit after successful update | No | `false` |
| `commit-message` | Commit message template (use {packages} for list) | No | `chore(deps): update Go dependencies` |
| `goupdate-version` | Version to install (latest, or specific tag). Ignored in goupdate repo. | No | `latest` |
| `github-token` | GitHub token for API requests (helps with rate limits) | No | `''` |
| `system-test-mode` | System test run mode: after_each, after_all, or none | No | `''` |
| `verbose` | Enable verbose output (shows debug info and test output on success) | No | `false` |

#### Outputs

| Output | Description |
|--------|-------------|
| `has-updates` | Whether any updates are available |
| `has-major-only` | Whether any packages have ONLY major updates |
| `updates-json` | Full JSON output from goupdate |
| `major-count` | Count of packages with major updates |
| `minor-count` | Count of packages with minor updates |
| `patch-count` | Count of packages with patch updates |
| `summary` | Human-readable summary of updates |
| `major-packages` | List of packages with major-only updates |
| `outdated-output` | Human-readable table of outdated packages |
| `update-output` | Human-readable output from update operation |
| `updated-count` | Number of packages updated |
| `has-changes` | Whether go.mod was modified |
| `updated-packages` | List of updated packages |
| `major-only-error` | Whether exit was due to major-only updates |
| `goupdate-version` | Installed goupdate version |
| `goupdate-source` | Installation source (local-build or release) |

#### Examples

**Check for updates only:**
```yaml
- uses: ./.github/actions/_goupdate
  with:
    mode: 'check'
```

**Apply patch updates:**
```yaml
- uses: ./.github/actions/_goupdate
  with:
    mode: 'update'
    update-type: 'patch'
```

**Full check and update with custom test:**
```yaml
- uses: ./.github/actions/_goupdate
  with:
    mode: 'check-and-update'
    update-type: 'minor'
    test-command: 'make test'
```

**Update with auto-commit:**
```yaml
- uses: ./.github/actions/_goupdate
  with:
    mode: 'update'
    commit-changes: 'true'
    commit-message: 'deps: update {packages}'
```

**Exclude specific packages:**
```yaml
- uses: ./.github/actions/_goupdate
  with:
    mode: 'check-and-update'
    exclude-packages: 'github.com/some/unstable-pkg,github.com/another/pkg'
```

---

## Examples

### Custom Docker Image Names

To customize the Docker image name, use the `image-name` input:

```yaml
jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: ./.github/actions/_dockerhub
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
          image-name: mycompany/my-custom-image-name
          tags: |
            latest
            ${{ github.ref_name }}
```

### Multi-Registry Docker Builds

Push to multiple registries by calling the action multiple times:

```yaml
jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # Push to Docker Hub
      - uses: ./.github/actions/_dockerhub
        with:
          registry: docker.io
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
          image-name: myorg/myapp
          tags: latest

      # Push to GitHub Container Registry
      - uses: ./.github/actions/_dockerhub
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
          image-name: ${{ github.repository }}
          tags: latest
```

### PR with Auto-Merge

Create a PR that auto-merges after checks pass:

```yaml
jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Make changes
        run: |
          # ... your changes ...
          git checkout -b feature/auto-update
          git add .
          git commit -m "chore: automated update"

      - uses: ./.github/actions/_gh-pr
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          title: 'chore: Automated update'
          body: |
            This PR was automatically generated.

            Changes:
            - Updated configuration
            - Applied patches
          base: main
          auto-merge: 'true'
          merge-method: squash
          delete-branch: 'true'
```

### Dependency Updates with Custom Tests

Check and apply dependency updates with custom validation:

```yaml
jobs:
  update-deps:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: ./.github/actions/_go-setup
        with:
          go-version: '1.24'

      - uses: ./.github/actions/_goupdate
        id: update
        with:
          mode: 'check-and-update'
          update-type: 'minor'
          test-command: |
            go test -race ./...
            go build ./...
          commit-changes: 'true'

      - name: Create PR if updates applied
        if: steps.update.outputs.has-changes == 'true'
        uses: ./.github/actions/_gh-pr
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          title: 'chore(deps): Update Go dependencies'
          body: |
            Updated packages: ${{ steps.update.outputs.updated-packages }}

            Summary: ${{ steps.update.outputs.summary }}
          auto-merge: 'true'
```

---

## See Also

- [Releasing Guide](releasing.md) - Complete workflow documentation
- [Configuration Guide](configuration.md) - goupdate YAML configuration
- [CLI Reference](cli.md) - goupdate command-line options
