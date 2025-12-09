# goupdate

Scan, list, and update dependencies across npm, Go, Composer, pip, and .NET from one CLI. Open-source, runs locally, no cloud services or git required.

## Table of Contents

- [What is goupdate?](#what-is-goupdate)
- [Feature Comparison](#feature-comparison)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Supported Ecosystems](#supported-ecosystems)
- [Commands](#commands)
  - [config](#config)
  - [scan](#scan)
  - [list](#list)
  - [outdated](#outdated)
  - [update](#update)
- [CLI Reference](#cli-reference)
- [Configuration](#configuration)
- [Common Use Cases](#common-use-cases)
  - [Autonomous Updates with System Tests](#autonomous-updates-with-system-tests)
  - [Incremental Updates for Safe Migrations](#incremental-updates-for-safe-migrations)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)
- [Documentation](#documentation)
- [Releasing (Maintainers)](#releasing-maintainers)

## What is goupdate?

**goupdate** is a command-line tool that helps developers manage software dependencies across multiple programming languages and package managers. Instead of using separate tools for each ecosystem (npm for JavaScript, pip for Python, composer for PHP, etc.), goupdate provides a single, unified interface to:

- **Discover** all dependency files in your project (package.json, go.mod, requirements.txt, etc.)
- **List** what versions you've declared vs. what's actually installed
- **Check** which packages have newer versions available
- **Update** dependencies safely with automatic rollback on failure

Unlike cloud-based alternatives (Dependabot, Renovate), goupdate runs entirely on your machine. It doesn't need access to your git repository, doesn't require accounts or API keys, and works on air-gapped servers or CI environments without internet access to third-party services.

## Feature Comparison

| Feature | goupdate | Dependabot | Renovate |
|---------|:--------:|:----------:|:--------:|
| Open source | Yes | Yes | Yes |
| Free (public repos) | Yes | Yes | Yes |
| Free (private repos) | Yes | Yes | Limited* |
| All features free | Yes | Yes | No* |
| No code access required | Yes | No | No |
| CLI tool | Yes | No | Limited** |
| No vendor lock-in | Yes | GitHub only | Self-host option |
| Works without git | Yes | No | No |
| Runs locally | Yes | No | No |
| Config inheritance | Yes | No | Yes |
| Custom PM via config | Yes | No | Yes |
| Package grouping | Yes | Yes | Yes |
| Atomic rollback | Yes | N/A | N/A |
| SBOM/Audit reports | Yes | No | No |

*Renovate hosted service (Mend.io) has usage limits for private repos; some features require paid plans.

**Renovate CLI only supports dry-run mode locally; full update functionality requires a git repository and is designed for CI/server environments.

goupdate is designed for **local control** - you decide when and how updates happen. It runs entirely on your machine without requiring access to your git repository or source code. It queries package registries (npm, PyPI, etc.) to check for updates but requires no cloud services, accounts, or external code access.

goupdate is built with Go for cross-platform compatibility. You only need the binary for your system architecture - no PHP, Node.js, or other runtimes required.

See [docs/comparison.md](docs/comparison.md) for detailed feature comparison.

## Installation

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/ajxudir/goupdate/releases):

```bash
# Linux (amd64)
curl -Lo goupdate.tar.gz https://github.com/ajxudir/goupdate/releases/latest/download/goupdate_linux_amd64.tar.gz
tar -xzf goupdate.tar.gz
chmod +x goupdate
sudo mv goupdate /usr/local/bin/

# macOS (arm64)
curl -Lo goupdate.tar.gz https://github.com/ajxudir/goupdate/releases/latest/download/goupdate_darwin_arm64.tar.gz
tar -xzf goupdate.tar.gz
chmod +x goupdate
sudo mv goupdate /usr/local/bin/
```

### Build from Source

```bash
go install github.com/ajxudir/goupdate@latest
# or
git clone https://github.com/ajxudir/goupdate && cd goupdate
make build && sudo make install
```

### Docker

Run without installing Go or any dependencies:

```bash
git clone https://github.com/ajxudir/goupdate && cd goupdate
docker run -v $(pwd)/pkg/testdata:/workspace ajxudir/goupdate:latest outdated
```

See [Dockerfile](Dockerfile) and [docker-compose.yml](docker-compose.yml) for configuration options.

## Quick Start

```bash
goupdate scan                    # Discover package files
goupdate list                    # Show declared vs installed versions
goupdate outdated                # Find packages with newer versions
goupdate update --dry-run        # Preview updates
goupdate update --yes            # Apply updates (skip confirmation)
```

## Supported Ecosystems

| Ecosystem | Rule | Manifest | Lock File |
|-----------|------|----------|-----------|
| **JavaScript** | `npm` | `package.json` | `package-lock.json` |
| **JavaScript** | `pnpm` | `package.json` | `pnpm-lock.yaml` |
| **JavaScript** | `yarn` | `package.json` | `yarn.lock` |
| **Go** | `mod` | `go.mod` | `go.sum` |
| **PHP** | `composer` | `composer.json` | `composer.lock` |
| **Python** | `requirements` | `requirements.txt` | - |
| **Python** | `pipfile` | `Pipfile` | `Pipfile.lock` |
| **.NET** | `msbuild` | `*.csproj` | `packages.lock.json` |
| **.NET** | `nuget` | `packages.config` | `packages.lock.json` |

Need a different package manager? Add it via [configuration](docs/configuration.md) - no code required. See [examples/ruby-api/](examples/ruby-api/) for a custom Bundler setup.

## Commands

### config

Manage and validate configuration:

```bash
goupdate config --show-defaults    # Show built-in defaults
goupdate config --show-effective   # Show merged config (defaults + local)
goupdate config --init             # Create .goupdate.yml template
goupdate config --validate         # Validate config (rejects unknown fields)
goupdate config --validate -c ./custom.yml  # Validate specific file
```

**Tip:** Validate your config before running other commands. All commands (`scan`, `list`, `outdated`, `update`) also perform preflight validation automatically.

### scan

Discover package files in your project:

```bash
$ goupdate scan
RULE  PM  FORMAT  FILE
----  --  ------  ------------
npm   js  json    package.json

Total entries: 1
Unique files: 1
Rules matched: 1
```

### list

Show declared dependencies with installed versions from lock files:

```bash
$ goupdate list
RULE  PM  TYPE  CONSTRAINT      VERSION  INSTALLED  STATUS        GROUP    NAME
----  --  ----  --------------  -------  ---------  ------------  -------  --------
npm   js  prod  Compatible (^)  5.1.0    5.1.1      🟢 LockFound  backend  helmet
npm   js  prod  Compatible (^)  4.8.0    4.17.2     🟢 LockFound  backend  mongodb
npm   js  dev   Compatible (^)  3.2.0    3.2.11     🟢 LockFound  frontend vite
npm   js  prod  Compatible (^)  17.0.0   17.0.2     🟢 LockFound  frontend react
```

Filter by type, rule, name, or group:

```bash
goupdate list --type prod        # Production dependencies only
goupdate list --type dev         # Development dependencies only
goupdate list --rule npm         # Only npm packages
goupdate list -r npm,composer    # Multiple rules (comma-separated)
goupdate list --name lodash      # Filter by package name
goupdate list -n react,express   # Multiple packages (comma-separated)
goupdate list --group backend    # Filter by group
goupdate list -g frontend        # Filter by group (shorthand)
```

Status indicators: `🟢 LockFound` (version resolved), `🟠 LockMissing` (no lock file), `🔵 NotInLock` (not in lock file), `⚪ NotConfigured` (lock file not supported for this rule).

See [docs/cli.md](docs/cli.md) for all options.

### outdated

Check for available updates by querying package registries:

```bash
$ goupdate outdated
RULE  PM  TYPE  CONSTRAINT      VERSION  INSTALLED  MAJOR  MINOR  PATCH  STATUS        NAME
----  --  ----  --------------  -------  ---------  -----  -----  -----  ------------  ------
npm   js  prod  Compatible (^)  4.17.1   4.17.3     5.0.0  4.18   4.17.4 🟠 Outdated   lodash
npm   js  prod  Compatible (^)  2.4.0    2.4.7      #N/A   #N/A   #N/A   🟢 UpToDate   winston
npm   js  dev   Exact (=)       2.8.8    2.8.8      3.0.0  2.9    2.8.9  🟠 Outdated   prettier
```

#### Version Scope Flags

Control which updates to show with `--major`, `--minor`, `--patch`:

```bash
goupdate outdated --patch        # Only patch updates (1.2.3 -> 1.2.4)
goupdate outdated --minor        # Minor and patch (1.2.0 -> 1.3.0)
goupdate outdated --major        # All updates including major (1.0 -> 2.0)
```

These flags respect your constraint configuration. A package with `^1.0.0` (Compatible) constraint will only show updates within `1.x.x` unless `--major` is specified. Constraints are **never modified** - only the version number is updated.

#### Incremental Updates

For systems with database migrations or breaking changes that require step-by-step upgrades, use the `incremental` config option to select the **nearest** available version instead of the latest:

```yaml
rules:
  npm:
    incremental: [prisma, typeorm, sequelize]
```

When incremental is enabled for a package, goupdate selects the nearest version in each category:
- **Major**: nearest next major (e.g., 2.0.0 instead of 5.0.0)
- **Minor**: nearest next minor (e.g., 1.1.0 instead of 1.9.0)
- **Patch**: nearest next patch (e.g., 1.0.1 instead of 1.0.15)

This respects your existing constraint - if using `^` (Compatible), you get the nearest minor/patch within your major version.

For complete control over available versions, override the `outdated.commands` to filter versions at the source before goupdate sees them:

```yaml
rules:
  npm:
    package_overrides:
      prisma:
        outdated:
          commands: |
            npm view {{package}} versions --json | jq '[.[] | select(startswith("5."))]'
```

See [docs/cli.md](docs/cli.md) for all options.

### update

Apply dependency updates to manifest files. The update command validates commands, shows a detailed plan, asks for confirmation, applies updates, and reports results:

```bash
$ goupdate update --patch

Update Plan
══════════════════════════════════════════════════════════════════════

Will update (--patch scope):
  react                18.2.0 → 18.2.5  (major: 19.0.0, minor: 18.3.0 available)
  react-dom            18.2.0 → 18.2.5  (major: 19.0.0, minor: 18.3.0 available)
  axios                1.3.0 → 1.3.4  (minor: 1.6.0 available)
  lodash               4.17.0 → 4.17.21  (fully updated to latest)

Up to date (other updates available):
  zustand              4.3.0  (major: 5.0.0 available)

Summary: 4 to update, 2 up-to-date
         (3 have major, 3 have minor available)

4 package(s) will be updated. Continue? [y/N]: y

RULE  PM  TYPE  CONSTRAINT       VERSION  INSTALLED  TARGET   STATUS       GROUP  NAME
----  --  ----  ---------------  -------  ---------  -------  -----------  -----  ---------
npm   js  prod  Patch (--patch)  1.3.0    1.3.0      1.3.4    🟢 Updated          axios
npm   js  prod  Patch (--patch)  4.18.2   4.18.2     #N/A     🟢 UpToDate         express
npm   js  prod  Patch (--patch)  4.17.0   4.17.0     4.17.21  🟢 Updated          lodash
npm   js  prod  Patch (--patch)  18.2.0   18.2.0     18.2.5   🟢 Updated          react
npm   js  prod  Patch (--patch)  18.2.0   18.2.0     18.2.5   🟢 Updated          react-dom
npm   js  prod  Patch (--patch)  4.3.0    4.3.0      #N/A     🟢 UpToDate         zustand

Total packages: 6

Update Summary
══════════════════════════════════════════════════════════════════════

Successfully updated:
  react                18.2.0 → 18.2.5  (major: 19.0.0, minor: 18.3.0 available)
  react-dom            18.2.0 → 18.2.5  (major: 19.0.0, minor: 18.3.0 available)
  axios                1.3.0 → 1.3.4  (minor: 1.6.0 available)
  lodash               4.17.0 → 4.17.21  (fully updated to latest)

Up to date (other updates available):
  zustand              4.3.0  (major: 5.0.0 available)

Summary: 4 updated, 2 up-to-date
         (3 have major, 3 have minor updates still available)
```

The update process:
1. **Preflight check**: Validates package manager commands are available
2. **Update plan**: Shows what will be updated with available versions info
3. **Confirmation prompt**: Asks before proceeding (unless `--yes` or `--dry-run`)
4. **Apply updates**: Updates version in manifest files (package.json, go.mod, etc.)
5. **Run lock commands**: Executes `npm install`, `go mod tidy`, etc.
6. **Verify and report**: Shows final status for each package

#### Rollback Behavior

If an update fails, goupdate automatically reverts the manifest file changes for that package:

- **Grouped packages**: All packages in a group are rolled back together if any fails
- **Individual packages**: Only the failed package is rolled back
- **With `--continue-on-fail`**: Failed packages are rolled back, successful updates are kept, processing continues to remaining packages

```bash
goupdate update --yes            # Skip confirmation
goupdate update --patch --yes    # Only patch updates
goupdate update --skip-lock      # Skip lock file regeneration
goupdate update --dry-run        # Preview without making changes
```

See [docs/cli.md](docs/cli.md) for all options.

## CLI Reference

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to config file (default: `.goupdate.yml`) |
| `--directory` | `-d` | Working directory (default: `.`) |
| `--help` | `-h` | Show help |

### Filter Flags (list, outdated, update)

| Flag | Short | Description |
|------|-------|-------------|
| `--type` | `-t` | Filter by type: `all`, `prod`, `dev` |
| `--package-manager` | `-p` | Filter by package manager |
| `--rule` | `-r` | Filter by rule name (comma-separated) |
| `--name` | `-n` | Filter by package name (comma-separated) |
| `--group` | `-g` | Filter by group (comma-separated) |

### Version Flags (outdated, update)

| Flag | Description |
|------|-------------|
| `--major` | Include major version updates |
| `--minor` | Include minor version updates (default scope) |
| `--patch` | Restrict to patch updates only |

### Update Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | | Preview changes without applying |
| `--yes` | `-y` | Skip confirmation prompt |
| `--skip-lock` | | Skip lock file regeneration |
| `--continue-on-fail` | | Continue after package failures |
| `--skip-preflight` | | Skip command validation |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Partial failure (with `--continue-on-fail`) |
| `2` | Complete failure |
| `3` | Configuration error |

See [docs/cli.md](docs/cli.md) for complete reference.

## Configuration

Create `.goupdate.yml` in your project root:

```yaml
extends: [default]  # Inherit built-in package manager rules

rules:
  npm:
    # Group related packages for atomic updates
    groups:
      react: [react, react-dom, @types/react]
      testing: [jest, @testing-library/react]

    # Exclude unstable versions (regex patterns)
    exclude_versions:
      - "(?i)alpha|beta|rc|canary"

    # Ignore certain packages from updates
    ignore:
      - "@types/*"
      - "eslint-*"

    # Select nearest version instead of latest (for migrations)
    incremental:
      - prisma
      - typeorm
```

### Configuration Options

| Option | Purpose | Example |
|--------|---------|---------|
| `extends` | Inherit from other configs | `[default]` |
| `groups` | Group packages for atomic updates | `{react: [react, react-dom]}` |
| `ignore` | Skip packages matching patterns | `["@types/*"]` |
| `exclude_versions` | Filter out version strings (regex) | `["(?i)beta"]` |
| `incremental` | Select nearest version instead of latest | `["prisma"]` |

See [docs/configuration.md](docs/configuration.md) for the full schema.

## Common Use Cases

### Autonomous Updates with System Tests

**System tests are the foundation of safe automation.** When your test suite validates that your application works correctly, you can confidently automate dependency updates - and any future automation - knowing that breaking changes will be caught before reaching production.

This test-driven approach creates a virtuous cycle:
- **Better tests** → more confidence in automation
- **More automation** → less manual maintenance
- **Less maintenance** → more time for features and better tests

The investment in comprehensive system tests pays dividends across all automation, not just dependency updates. The same test suite that validates updates also protects against regressions from new features, refactoring, and infrastructure changes.

```yaml
# .goupdate.yml
extends: [default]

system_tests:
  run_preflight: true    # Verify app works before updates
  run_mode: after_all    # Run tests after all updates (or after_each for max safety)
  tests:
    - name: unit-tests
      commands: npm test
      timeout_seconds: 120
    - name: e2e-tests
      commands: npm run e2e
      timeout_seconds: 300
```

**How it works:**

1. goupdate applies a dependency update
2. Lock file is regenerated (`npm install`, `go mod tidy`, etc.)
3. Your test command runs automatically
4. If tests **pass** → update is kept
5. If tests **fail** → update is rolled back, next package continues

This creates an autonomous update cycle:

```
AUTONOMOUS UPDATE CYCLE
═══════════════════════════════════════════════════════════════════

Weekly Schedule (or push to stage)
        │
        ▼
Check for minor/patch updates
        │
        ▼
Apply update + Run system tests ◄──────┐
        │                              │
  ┌─────┴─────┐                        │
  │           │                        │
Pass        Fail                       │
  │           │                        │
  ▼           ▼                        │
Keep      Rollback ────────────────────┘
Update    (try next package)
  │
  ▼
Create prerelease for final testing
        │
        ▼
Manual review + merge to production
```

**Benefits:**

- **Zero manual work** after initial setup
- **Security updates** applied automatically within hours
- **Breaking changes** caught before they reach production
- **Developers focus** on features, not dependency maintenance
- **Audit trail** via git commits and release notes
- **Foundation for future automation** - the same tests enable safe automation of other tasks (refactoring, migrations, infrastructure changes)

> **Note on test coverage:** Tests won't catch every problem on day one - and that's okay. The key is that when an issue does slip through, you add a test for it. Over time, your test suite grows to catch an ever-wider range of problems automatically. Instead of playing whack-a-mole with recurring issues, each fix permanently prevents that class of problem. This compounds: a year of adding regression tests means a year's worth of issues that will never waste your time again.

**Example: Full automation setup**

```yaml
# .goupdate.yml - Production-ready configuration
extends: [default]

# System tests validate updates automatically
system_tests:
  run_preflight: true
  run_mode: after_all
  stop_on_fail: true
  tests:
    - name: lint
      commands: npm run lint
      timeout_seconds: 60
      continue_on_fail: true  # Lint issues don't block updates

    - name: unit-tests
      commands: npm test
      timeout_seconds: 180

    - name: build
      commands: npm run build
      timeout_seconds: 300

    - name: e2e-tests
      commands: npm run e2e
      timeout_seconds: 600

rules:
  npm:
    # Group related packages for atomic updates
    groups:
      react: [react, react-dom, @types/react]
      prisma: [prisma, @prisma/client]

    # Step-by-step for database migrations
    incremental: [prisma, typeorm]

    # Skip unstable versions (regex patterns)
    exclude_versions:
      - "(?i)alpha|beta|rc|canary"
```

With this setup:
- Minor/patch updates are applied and tested automatically
- Grouped packages update together (React ecosystem stays in sync)
- Prisma updates one version at a time (for safe migrations)
- If tests fail, the update is rolled back and logged
- Successful updates create a prerelease for final human review

### Incremental Updates for Safe Migrations

For packages with database migrations or breaking changes between versions, use incremental mode to update **one version at a time**:

```yaml
rules:
  npm:
    incremental: [prisma, typeorm, sequelize]
```

**Without incremental** (default): `prisma@2.0.0` → `prisma@5.0.0` (latest)
**With incremental**: `prisma@2.0.0` → `prisma@3.0.0` → `prisma@4.0.0` → `prisma@5.0.0`

Each step runs your test suite. If version 4.0.0 breaks tests, you stay on 3.0.0 and get notified - no manual debugging of multi-version jumps.

### CI/CD Pipeline

Apply updates non-interactively with proper error handling:

```bash
#!/bin/bash
# Apply patch updates, continue if individual packages fail
goupdate update --patch --yes --continue-on-fail

case $? in
  0) echo "All updates applied successfully" ;;
  1) echo "Some updates applied, some failed - check logs" ;;
  2) echo "All updates failed" ;;
  3) echo "Configuration error" ;;
esac
```

When a package update fails, goupdate **automatically rolls back** that package's changes before continuing to the next. With `--continue-on-fail`, successful updates are kept while failed ones are reverted.

For production-only updates:

```bash
goupdate update --patch --type prod --yes --continue-on-fail
```

### Audit Reports & EU CRA Compliance

The **EU Cyber Resilience Act (CRA)** requires manufacturers to maintain a **Software Bill of Materials (SBOM)** - a formal inventory of software components and dependencies.

goupdate helps with CRA compliance by generating dependency reports:

```bash
# Generate SBOM-style dependency inventory
goupdate list > sbom-dependencies.txt

# Track which packages need updates
goupdate outdated > security-audit.txt

# Machine-readable format for compliance tools
goupdate list --type prod > production-dependencies.txt
```

Use in CI to maintain up-to-date dependency records for audits and compliance reviews. Future OpenTelemetry integration will enable exporting this data to centralized systems for organization-wide visibility.

### Server Without Git

Update dependencies on a deployment server:

```bash
goupdate update -d /var/www/app --config /etc/goupdate/policy.yml --yes
```

### Centralized Policy

Share configuration across multiple projects:

```yaml
# /etc/goupdate/company-policy.yml
extends: [default]
rules:
  npm:
    exclude_versions: ["(?i)alpha|beta|rc"]
    groups:
      security: [lodash, express, helmet]
```

Then in each project:

```yaml
# .goupdate.yml
extends:
  - /etc/goupdate/company-policy.yml
```

### Private Registry Authentication

Private registry access is configured through native package manager tools - no credentials are stored in goupdate configuration. Configure authentication as you normally would:

- **npm**: Use `.npmrc` or `npm login`
- **Composer**: Use `composer config` or `auth.json`
- **Go**: Use `GOPRIVATE` and `git config`
- **pip**: Use `pip.conf` or environment variables

This approach keeps secrets management within your existing tooling and avoids an extra layer of abstraction.

### Future: OpenTelemetry Support

Future releases will include OpenTelemetry support for ingesting package data across projects organization-wide. You configure your own OTEL server - **no data is collected by third parties or routed through cloud services**. All processing runs on your infrastructure.

## Troubleshooting

### Configuration validation errors

If you see configuration validation errors when running commands:

```
configuration validation failed for .goupdate.yml:
  - unknown field in config: manager_typo

💡 Run 'goupdate config --validate' for details, or see docs/configuration.md
```

This means your config file has unknown fields (likely typos). Use the validate command to check your config:

```bash
goupdate config --validate              # Check .goupdate.yml
goupdate config --validate -c ./my.yml  # Check specific file
```

**For detailed schema information**, add the `--verbose` flag:

```bash
goupdate config --validate --verbose
```

This shows:
- Valid field names for each config section
- Suggestions for common typos (e.g., "did you mean 'commands'?")
- Documentation references for fixing the issue

### "Command not found" errors

goupdate validates that package manager commands exist before running. If you see errors like:

```
Pre-flight validation failed:
  - command not found: npm
    Resolution: Install Node.js: https://nodejs.org/
```

Install the required package manager or use `--skip-preflight` to bypass validation.

### Lock file shows "NotConfigured"

If installed versions show as `⚪ NotConfigured`, the lock file format is not configured for that rule. You can add lock file support via configuration:

```yaml
rules:
  my-rule:
    lock_files:
      - files: ["my-lock.json"]
        format: json
        extraction:
          pattern: '"(?P<n>[^"]+)":\s*"(?P<version>[^"]+)"'
```

See [docs/configuration.md](docs/configuration.md) for lock file configuration details.

### Updates not appearing

1. Check if the package is in your `ignore` list
2. Check if versions are filtered by `exclude_versions`
3. Try `--major` flag for major version updates
4. Verify the package registry is accessible

### Grouped updates failing

When updates fail for grouped packages, goupdate automatically rolls back all manifest changes for that group. Use `--continue-on-fail` to proceed with remaining groups after a failure.

## Examples

| Example | Framework | Demonstrates |
|---------|-----------|--------------|
| [react-app](examples/react-app/) | React/Vite | Package groups, incremental updates |
| [django-app](examples/django-app/) | Django | Python groups, ignoring packages |
| [go-cli](examples/go-cli/) | Go CLI | Go module handling |
| [laravel-app](examples/laravel-app/) | Laravel | Composer package groups |
| [ruby-api](examples/ruby-api/) | Ruby/Rails | **Custom package manager** via config |

## Documentation

- [CLI reference](docs/cli.md) - All commands, flags, and exit codes
- [Configuration guide](docs/configuration.md) - YAML schema and options
- [Features overview](docs/features.md) - Capabilities and supported ecosystems
- [Testing guide](docs/testing.md) - Running tests and coverage
- [Tool comparison](docs/comparison.md) - vs Dependabot and Renovate
- [Architecture](docs/architecture/) - Internals and data flow for contributors
- [Docker usage](Dockerfile) - Container deployment options
- [Releasing guide](docs/releasing.md) - CI/CD workflows, setup guide for your project, and platform portability
- [GitHub Actions reference](docs/actions.md) - Complete reference for all reusable actions (inputs, outputs, examples)

## Releasing (Maintainers)

This project includes **reusable GitHub Actions** for Go projects. The workflows handle dependency updates, release candidates, GoReleaser builds, and multi-arch Docker images.

**Reusable actions** (copy `.github/actions/` to your project):
- `_go-setup` - Go environment with caching
- `_go-test` - Go test runner with options
- `_goupdate` - Dependency check and update
- `_gh-release` - GitHub release creation
- `_gh-pr` - PR creation with auto-merge
- `_dockerhub` - Multi-arch Docker builds (customizable image name, registry, platforms)
- `_goreleaser` - GoReleaser binary builds

See [docs/actions.md](docs/actions.md) for complete reference of all inputs, outputs, and examples.

See [docs/releasing.md](docs/releasing.md) for workflow documentation including:
- Configuration options for each workflow
- How to customize for your project
- Examples of using actions directly
- Troubleshooting guide

### Branching Strategy

```
stage branch (development/staging)
══════════════════════════════════════════════════════════════
├── Receives feature PRs and auto-updates
├── Creates prereleases (_stage-YYYYMMDD-rcN)
└── Tests run automatically on push

                        ↓
              (manual merge when ready)
                        ↓

main branch (production)
══════════════════════════════════════════════════════════════
├── Stable releases only (vX.Y.Z)
└── Create release via tag push or GitHub UI
```

### Release Flow

1. **Development happens on `stage`:**
   - PRs merged to `stage` branch
   - Weekly dependency updates (minor/patch only)
   - Tests run, then prerelease is created

2. **Promote to production:**
   - Merge `stage` → `main`
   - Create a stable release using one of these methods:
     - **Command Line:** `git tag v1.2.3 && git push origin v1.2.3`
     - **GitHub UI:** Go to Releases → Create new release → Create tag `vX.Y.Z` on `main`
   - Binaries and Docker images built automatically

### Required Setup

#### GitHub App Setup (Required for Auto-Update)

The auto-update workflow **requires** a GitHub App for authentication. The workflow will fail immediately if `GOUPDATE_APP_ID` and `GOUPDATE_APP_PRIVATE_KEY` secrets are not configured.

| Secret | Description |
|--------|-------------|
| `GOUPDATE_APP_ID` | GitHub App ID (numeric) |
| `GOUPDATE_APP_PRIVATE_KEY` | GitHub App private key (PEM format) |

**Why use a GitHub App?**

GitHub App authentication provides several advantages over Personal Access Tokens:
- **Not tied to any user account** - survives employee turnover
- **Works across org repos** - install once, use everywhere
- **Higher API rate limits** - 5,000 requests/hour vs 1,000 for PATs
- **Short-lived tokens** - more secure, auto-rotated
- **Fine-grained permissions** - only grant what's needed

The auto-update workflow uses the GitHub App to:
- **Trigger PR workflows**: GitHub prevents workflows created with `GITHUB_TOKEN` from triggering other workflows
- **Check CI status**: Read check runs to verify all tests pass before merging
- **Merge PRs**: Merge the PR after all checks pass
- **Trigger release workflow**: Start the release process after merge

**Setup Instructions:**

1. **Create a GitHub App:**
   - Go to **GitHub Settings > Developer settings > GitHub Apps**
   - Click **"New GitHub App"**
   - Set app name: `GoUpdate-<YourOrg>` (e.g., `GoUpdate-Acme` - GitHub App names must be globally unique)
   - Set Homepage URL to your repository URL
   - Uncheck **"Webhook > Active"** (not needed)
   - Under **"Repository permissions"**, set:
     - **Checks**: Read (to check CI status)
     - **Contents**: Read and write (to push branches)
     - **Pull requests**: Read and write (to create/merge PRs)
     - **Workflows**: Read and write (to trigger workflows)
   - Under **"Where can this GitHub App be installed?"**, select "Only on this account"
   - Click **"Create GitHub App"**

2. **Note the App ID:**
   - After creation, you'll see the App ID on the app settings page
   - Save this number for the `APP_ID` secret

3. **Generate a private key:**
   - Scroll down to **"Private keys"**
   - Click **"Generate a private key"**
   - A `.pem` file will be downloaded
   - Save the entire contents of this file for the `APP_PRIVATE_KEY` secret

4. **Install the app on your repository:**
   - Go to **"Install App"** in the left sidebar
   - Click **"Install"** next to your account/organization
   - Select "Only select repositories" and choose your repo
   - Click **"Install"**

5. **Add secrets to your repository:**
   - Go to your repo > **Settings > Secrets and variables > Actions**
   - Click **"New repository secret"**
   - Add `GOUPDATE_APP_ID` with the numeric App ID
   - Add `GOUPDATE_APP_PRIVATE_KEY` with the entire contents of the `.pem` file (including `-----BEGIN RSA PRIVATE KEY-----` and `-----END RSA PRIVATE KEY-----`)

> **Note:** Without these secrets, the auto-update workflow will fail at startup with setup instructions.

#### Secrets (DockerHub)

For Docker image publishing (optional - builds will skip if not configured):

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Your Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token (not password) |

**To create a Docker Hub token:**
1. Log in to [Docker Hub](https://hub.docker.com/)
2. Go to Account Settings > Security > Access Tokens
3. Click "New Access Token" with Read/Write permissions
4. Add the token as a repository secret

The workflow summary explains how to enable Docker if secrets are missing.

## License

MIT License - see [LICENSE](LICENSE) file.
