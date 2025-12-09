# Frontend Example (pnpm)

Example pnpm-based Vue.js frontend project with automated dependency updates.

Based on [matematikk-mooc/frontend](https://github.com/matematikk-mooc/frontend) project structure.

## Quick Start

1. Copy the `.github` folder to your repository
2. Copy `.goupdate.yml` to your repository root
3. Edit the `env` section in `.github/workflows/auto-update.yml`
4. Push to your repository

## Configuration

### Workflow Configuration

Edit environment variables in `.github/workflows/auto-update.yml`:

```yaml
env:
  # Package manager
  PACKAGE_MANAGER: 'pnpm'

  # Node.js version
  NODE_VERSION: '20'

  # Branch settings
  UPDATE_BRANCH: 'goupdate/auto-update'
  TARGET_BRANCH: 'stage-updates'

  # Test command (set to empty to skip)
  TEST_COMMAND: 'pnpm run test'

  # Packages to exclude (comma-separated)
  EXCLUDE_PACKAGES: ''
```

### GoUpdate Configuration

Edit `.goupdate.yml` for package grouping and system tests:

```yaml
rules:
  pnpm:
    # Group related packages
    groups:
      vue:
        - vue
        - vuex
        - vue-router

    # Update incrementally for stability
    incremental:
      - vue
      - vuex

    # Skip packages
    ignore:
      - "@types/*"

# Run tests after updates
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: build
      commands: pnpm run build
```

## Features

- **Weekly scheduled updates** - Runs every Monday at UTC midnight
- **Manual trigger** - Check-only or full update modes
- **Grouped updates** - Vue ecosystem, build tools, and testing packages updated together
- **Incremental updates** - Critical packages like Vue are updated one version at a time
- **System tests** - Lint, build, and test validation after updates
- **PR automation** - Creates PRs with detailed update summaries

## Manual Trigger

Go to Actions → Auto Update Dependencies → Run workflow:
- **check-only**: Only check for updates, no changes
- **update**: Apply updates and create PR

## Files

```
.github/
├── actions/
│   ├── _goupdate-install/   # Download goupdate binary
│   ├── _goupdate-check/     # Check for available updates
│   ├── _goupdate-update/    # Apply dependency updates
│   ├── _gh-pr/              # Create pull requests
│   └── _git-branch/         # Git branch management
└── workflows/
    └── auto-update.yml      # Main workflow

.goupdate.yml                # Update configuration
package.json                 # Project dependencies
```

## Update Policy

- **Patch/Minor**: Applied automatically
- **Major**: Alerts only, does not block other updates

If a package has both major and patch available, the patch is applied and major is reported.
