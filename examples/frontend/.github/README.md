# Frontend Auto Update Workflow

Drop-in GitHub Actions workflow for automated dependency updates in frontend projects.

Based on [matematikk-mooc/frontend](https://github.com/matematikk-mooc/frontend) project.

## Quick Start

1. Copy this `.github` folder to your repository
2. Copy the `.goupdate.yml` file to your repository root
3. Edit the `env` section in `.github/workflows/auto-update.yml`
4. Push to your repository

## Configuration

### Enable Package Managers

Enable only the package managers your project uses. This example defaults to pnpm.

```yaml
env:
  # Package Managers - Enable the ones your project uses
  ENABLE_NPM: 'false'
  ENABLE_YARN: 'false'
  ENABLE_PNPM: 'true'      # Frontend uses pnpm
  ENABLE_COMPOSER: 'false'
  ENABLE_GO: 'false'
```

### Language Versions

```yaml
env:
  NODE_VERSION: '20'      # For npm, yarn, pnpm
  PHP_VERSION: '8.2'      # For composer
  GO_VERSION: '1.24'      # For go mod
```

## Example Configurations

### Vue.js with pnpm (this example)
```yaml
ENABLE_PNPM: 'true'
NODE_VERSION: '20'
```

### React with npm
```yaml
ENABLE_NPM: 'true'
NODE_VERSION: '20'
TEST_COMMAND: 'npm test'
```

### Laravel + Vue (PHP + pnpm)
```yaml
ENABLE_PNPM: 'true'
ENABLE_COMPOSER: 'true'
NODE_VERSION: '20'
PHP_VERSION: '8.2'
TEST_COMMAND: 'composer test && pnpm test'
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
