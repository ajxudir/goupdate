# Frontend Auto Update Workflow

Drop-in GitHub Actions workflow for automated dependency updates in pnpm/npm/yarn projects.

Based on [matematikk-mooc/frontend](https://github.com/matematikk-mooc/frontend) project.

## Quick Start

1. Copy this `.github` folder to your repository
2. Copy the `.goupdate.yml` file to your repository root
3. Edit the `env` section in `.github/workflows/auto-update.yml`
4. Push to your repository

## Configuration

Edit environment variables in the workflow:

```yaml
env:
  # Package manager: npm, yarn, pnpm
  PACKAGE_MANAGER: 'pnpm'

  # Node.js version
  NODE_VERSION: '20'

  # Branch settings
  UPDATE_BRANCH: 'goupdate/auto-update'
  TARGET_BRANCH: 'stage-updates'

  # Test command (set to empty to skip)
  TEST_COMMAND: ''

  # Packages to exclude (comma-separated)
  EXCLUDE_PACKAGES: ''
```

## Package Manager Support

| Manager | Files | Setup |
|---------|-------|-------|
| `npm` | package.json, package-lock.json | `npm ci` |
| `yarn` | package.json, yarn.lock | corepack + `yarn install --frozen-lockfile` |
| `pnpm` | package.json, pnpm-lock.yaml | corepack + `pnpm install --frozen-lockfile` |

## Update Policy

- **Patch/Minor**: Applied automatically
- **Major**: Alerts only, does not block other updates

## Manual Trigger

Go to Actions → Auto Update Dependencies → Run workflow:
- **check-only**: Only check for updates
- **update**: Apply updates and create PR

## Files

```
.github/
├── actions/
│   ├── _goupdate-install/   # Download goupdate
│   ├── _goupdate-check/     # Check for updates
│   ├── _goupdate-update/    # Apply updates
│   ├── _gh-pr/              # Create PRs
│   └── _git-branch/         # Branch management
└── workflows/
    └── auto-update.yml      # Main workflow
```
