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

  # Node.js package managers
  ENABLE_NPM: 'false'
  ENABLE_YARN: 'false'
  ENABLE_PNPM: 'true'       # Frontend uses pnpm

  # PHP package manager
  ENABLE_COMPOSER: 'false'

  # Python package managers
  ENABLE_PIP: 'false'         # requirements.txt
  ENABLE_PIPENV: 'false'      # Pipfile

  # Go package manager
  ENABLE_GO: 'false'

  # .NET package manager
  ENABLE_NUGET: 'false'
```

### Language Versions

```yaml
env:
  NODE_VERSION: '20'        # For npm, yarn, pnpm
  PHP_VERSION: '8.2'        # For composer
  PYTHON_VERSION: '3.12'    # For pip, pipenv
  GO_VERSION: '1.24'        # For go mod
  DOTNET_VERSION: '8.0'     # For nuget
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
| `ENABLE_PIP` | pip | Python | requirements.txt |
| `ENABLE_PIPENV` | pipenv | Python | Pipfile, Pipfile.lock |
| `ENABLE_GO` | go mod | Go | go.mod, go.sum |
| `ENABLE_NUGET` | nuget | .NET | *.csproj, packages.config |

## Update Policy

- **Patch/Minor**: Applied automatically
- **Major**: Alerts only, does not block other updates

### Major-Only Updates

When **only** major updates are available (no minor/patch updates), the workflow **fails** intentionally. This triggers a GitHub email notification so you can review and handle them manually.

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
