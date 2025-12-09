# Private Package Registries

This guide covers how to configure authentication for private package registries when using goupdate.

## Overview

goupdate uses your existing package manager configurations for authentication. It does not store or manage credentials directly - instead, it relies on each package manager's native authentication mechanisms.

## npm / Node.js

### Using .npmrc

Create or edit `~/.npmrc` (global) or `.npmrc` in your project:

```ini
# For npm private registry
//registry.npmjs.org/:_authToken=your-auth-token

# For GitHub Packages
//npm.pkg.github.com/:_authToken=your-github-token
@your-org:registry=https://npm.pkg.github.com/

# For Azure Artifacts
//pkgs.dev.azure.com/your-org/_packaging/your-feed/npm/registry/:_authToken=your-token
```

### Environment Variables

For CI/CD environments, use environment variables:

```bash
# npm token
export NPM_TOKEN=your-auth-token

# In .npmrc, reference the environment variable
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
```

### Verifying Authentication

Test that authentication works before running goupdate:

```bash
npm whoami --registry https://registry.npmjs.org/
```

## Composer / PHP

### Using auth.json

Create `auth.json` in your project root or `~/.composer/auth.json`:

```json
{
    "http-basic": {
        "repo.packagist.com": {
            "username": "your-username",
            "password": "your-password"
        }
    },
    "github-oauth": {
        "github.com": "your-github-token"
    },
    "gitlab-token": {
        "gitlab.com": "your-gitlab-token"
    }
}
```

### Environment Variables

```bash
export COMPOSER_AUTH='{"github-oauth": {"github.com": "your-token"}}'
```

### Private Packagist

For Private Packagist:

```json
{
    "http-basic": {
        "repo.packagist.com": {
            "username": "token",
            "password": "your-private-packagist-token"
        }
    }
}
```

## pip / Python

### Using pip.conf

Create `~/.pip/pip.conf` (Unix) or `%APPDATA%\pip\pip.ini` (Windows):

```ini
[global]
index-url = https://username:password@your-private-pypi.com/simple/
extra-index-url = https://pypi.org/simple/

# Or use a trusted host
trusted-host = your-private-pypi.com
```

### Environment Variables

```bash
export PIP_INDEX_URL=https://username:password@your-private-pypi.com/simple/
export PIP_EXTRA_INDEX_URL=https://pypi.org/simple/
```

### Using keyring

For more secure credential storage:

```bash
pip install keyring
keyring set your-private-pypi.com username
```

### Poetry

For Poetry projects, use `poetry config`:

```bash
poetry config repositories.private https://your-private-pypi.com/simple/
poetry config http-basic.private username password
```

## Go Modules

### GOPRIVATE

Set `GOPRIVATE` for private modules:

```bash
export GOPRIVATE=github.com/your-org/*,gitlab.com/your-company/*
```

### Git Configuration

Configure git credentials for HTTPS:

```bash
# Using credential helper
git config --global credential.helper store

# Or configure URL rewriting
git config --global url."https://oauth2:${GITLAB_TOKEN}@gitlab.com/".insteadOf "https://gitlab.com/"
```

### Using .netrc

Create `~/.netrc`:

```
machine github.com
login your-username
password your-token

machine gitlab.com
login oauth2
password your-gitlab-token
```

## NuGet / .NET

### Using nuget.config

Create `nuget.config` in your solution root:

```xml
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <packageSources>
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" />
    <add key="private" value="https://your-private-nuget.com/v3/index.json" />
  </packageSources>
  <packageSourceCredentials>
    <private>
      <add key="Username" value="your-username" />
      <add key="ClearTextPassword" value="your-password" />
    </private>
  </packageSourceCredentials>
</configuration>
```

### Azure Artifacts

For Azure Artifacts:

```xml
<packageSources>
  <add key="azure" value="https://pkgs.dev.azure.com/your-org/_packaging/your-feed/nuget/v3/index.json" />
</packageSources>
<packageSourceCredentials>
  <azure>
    <add key="Username" value="any" />
    <add key="ClearTextPassword" value="your-pat-token" />
  </azure>
</packageSourceCredentials>
```

## CI/CD Best Practices

### GitHub Actions

```yaml
jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup npm authentication
        run: |
          echo "//registry.npmjs.org/:_authToken=${{ secrets.NPM_TOKEN }}" >> .npmrc

      - name: Run goupdate
        run: goupdate update --dry-run
        env:
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
```

### GitLab CI

```yaml
update-dependencies:
  script:
    - echo "//registry.npmjs.org/:_authToken=${NPM_TOKEN}" >> .npmrc
    - goupdate update --dry-run
  variables:
    NPM_TOKEN: $NPM_TOKEN  # Set in CI/CD settings
```

### Azure Pipelines

```yaml
steps:
  - task: npmAuthenticate@0
    inputs:
      workingFile: .npmrc

  - script: goupdate update --dry-run
    displayName: 'Check for updates'
```

## Troubleshooting

### Authentication Errors

**Symptom**: `401 Unauthorized` or `403 Forbidden` errors

**Solutions**:
1. Verify your credentials are correct
2. Check token expiration
3. Ensure the token has required scopes (read packages)
4. Test authentication with the native package manager first

### Certificate Errors

**Symptom**: `SSL certificate problem` or `unable to verify the first certificate`

**Solutions**:
1. Install CA certificates for your private registry
2. Use `--insecure` flag (not recommended for production)
3. Configure trusted hosts in package manager config

### Proxy Issues

**Symptom**: Connection timeouts or `ECONNREFUSED`

**Solutions**:
1. Configure proxy settings:
   ```bash
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   export NO_PROXY=localhost,127.0.0.1,.example.com
   ```
2. Add your private registry to `NO_PROXY` if behind the firewall

### Token Scope Issues

**Symptom**: Can read packages but goupdate shows no updates

**Solutions**:
1. Ensure token has `read:packages` scope
2. For some registries, you need `repo` scope as well
3. Check if the package is published to the correct registry

## Security Recommendations

1. **Never commit credentials** to version control
2. **Use environment variables** for tokens in CI/CD
3. **Rotate tokens regularly** (at least quarterly)
4. **Use minimal permissions** - only grant read access if write isn't needed
5. **Consider credential helpers** for local development
6. **Audit token usage** in your registry's access logs

## See Also

- [Configuration Reference](./configuration.md)
- [Troubleshooting](./troubleshooting.md)
- [GitHub Actions Integration](./actions.md)
