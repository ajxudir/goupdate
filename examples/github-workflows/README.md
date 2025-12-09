# GoUpdate GitHub Actions Example

Drop-in GitHub Actions workflow for automated dependency updates across multiple package managers.

## Quick Start

1. Copy the `.github` folder to your repository
2. Edit the `env` section in `.github/workflows/auto-update.yml`
3. Push to your repository

## Configuration

### Enable Package Managers

Enable only the package managers your project uses. Only the required tools will be installed.

```yaml
env:
  # Package Managers - Enable the ones your project uses
  # Set to 'true' to enable, 'false' to disable
  ENABLE_NPM: 'true'
  ENABLE_YARN: 'false'
  ENABLE_PNPM: 'false'
  ENABLE_COMPOSER: 'false'
  ENABLE_GO: 'false'
```

### Language Versions

Configure versions for the enabled package managers:

```yaml
env:
  NODE_VERSION: '20'      # For npm, yarn, pnpm
  PHP_VERSION: '8.2'      # For composer
  GO_VERSION: '1.24'      # For go mod
```

### Other Settings

```yaml
env:
  # Branch settings
  UPDATE_BRANCH: 'goupdate/auto-update'
  TARGET_BRANCH: 'stage-updates'

  # PR title template ({date} and {type} are replaced)
  PR_TITLE: 'chore(deps): Auto update - {type} ({date})'

  # Test command (set to empty to skip)
  TEST_COMMAND: 'npm test'

  # Packages to exclude (comma-separated)
  EXCLUDE_PACKAGES: ''
```

## Example Configurations

### Node.js (npm)
```yaml
ENABLE_NPM: 'true'
NODE_VERSION: '20'
TEST_COMMAND: 'npm test'
```

### Node.js (pnpm)
```yaml
ENABLE_PNPM: 'true'
NODE_VERSION: '20'
TEST_COMMAND: 'pnpm test'
```

### Laravel (PHP + npm)
```yaml
ENABLE_NPM: 'true'
ENABLE_COMPOSER: 'true'
NODE_VERSION: '20'
PHP_VERSION: '8.2'
TEST_COMMAND: 'composer test && npm test'
```

### Full-stack (Go + pnpm)
```yaml
ENABLE_PNPM: 'true'
ENABLE_GO: 'true'
NODE_VERSION: '20'
GO_VERSION: '1.24'
TEST_COMMAND: 'go test ./... && pnpm test'
```

## Supported Package Managers

| Flag | Manager | Language | Files |
|------|---------|----------|-------|
| `ENABLE_NPM` | npm | Node.js | package.json, package-lock.json |
| `ENABLE_YARN` | yarn | Node.js | package.json, yarn.lock |
| `ENABLE_PNPM` | pnpm | Node.js | package.json, pnpm-lock.yaml |
| `ENABLE_COMPOSER` | composer | PHP | composer.json, composer.lock |
| `ENABLE_GO` | go mod | Go | go.mod, go.sum |

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
│   ├── _goupdate-install/   # Download goupdate binary
│   ├── _goupdate-check/     # Check for updates
│   ├── _goupdate-update/    # Apply updates
│   ├── _gh-pr/              # Create pull requests
│   └── _git-branch/         # Branch management
└── workflows/
    └── auto-update.yml      # Main workflow
```
