# GoUpdate GitHub Actions Example

Drop-in GitHub Actions workflow for automated dependency updates.

## Quick Start

1. Copy the `.github` folder to your repository
2. Edit the `env` section in `.github/workflows/auto-update.yml`
3. Push to your repository

## Configuration

Edit the environment variables in the workflow:

```yaml
env:
  # Goupdate source (change to your org's fork if needed)
  GOUPDATE_REPO: 'ajxudir/goupdate'

  # Package manager: npm, yarn, pnpm, composer, mod
  PACKAGE_MANAGER: 'npm'

  # Language version (uncomment the one you need)
  NODE_VERSION: '20'
  # PHP_VERSION: '8.2'
  # GO_VERSION: '1.24'

  # Branch settings
  UPDATE_BRANCH: 'goupdate/auto-update'
  TARGET_BRANCH: 'stage-updates'

  # Test command (set to empty to skip)
  TEST_COMMAND: 'npm test'

  # Packages to exclude (comma-separated)
  EXCLUDE_PACKAGES: ''
```

## Supported Package Managers

| Manager | Files |
|---------|-------|
| `npm` | package.json, package-lock.json |
| `yarn` | package.json, yarn.lock |
| `pnpm` | package.json, pnpm-lock.yaml |
| `composer` | composer.json, composer.lock |
| `mod` | go.mod, go.sum |

## Update Policy

- **Patch/Minor**: Applied automatically
- **Major**: Alerts only, does not block other updates

If a package has both major and patch available, the patch is applied and major is reported.

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
│   └── _gh-pr/              # Create PRs
└── workflows/
    └── auto-update.yml      # Main workflow
```
