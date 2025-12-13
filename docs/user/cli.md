# CLI Commands

The CLI exposes seven commands. All data commands honor `--config` to load an alternate YAML file and `--directory` to override the configured `working_dir` when scanning files.

## Table of Contents

- [Exit Codes](#exit-codes)
- [Quick Reference](#quick-reference)
- [Global Flags](#global-flags)
- [Output Format Flag](#output-format-flag)
- [list](#list)
- [outdated](#outdated)
- [update](#update)
- [scan](#scan)
- [config](#config)
- [version](#version)
- [help](#help)
- [Supported Rules](#supported-rules)
- [Examples](#examples)
- [Output Format Examples](#output-format-examples)
- [Related Documentation](#related-documentation)

## Exit Codes

goupdate uses distinct exit codes for scripting integration:

| Code | Name | Description |
|------|------|-------------|
| `0` | Success | All operations completed successfully |
| `1` | Partial Failure | Some operations failed, some succeeded (use `--continue-on-fail`) |
| `2` | Failure | All operations failed or a critical error occurred |
| `3` | Config Error | Configuration or validation error (missing commands, invalid config) |

### Using Exit Codes in Scripts

```bash
# Check if any updates are available
goupdate outdated --patch
if [ $? -eq 0 ]; then
  echo "All packages are up to date"
elif [ $? -eq 1 ]; then
  echo "Some packages could not be checked (partial failure)"
elif [ $? -eq 2 ]; then
  echo "Outdated packages found or complete failure"
fi

# Apply updates, allow partial success
goupdate update --patch --yes --continue-on-fail
case $? in
  0) echo "All updates applied successfully" ;;
  1) echo "Some updates applied, some failed" ;;
  2) echo "All updates failed" ;;
  3) echo "Configuration error - check your setup" ;;
esac
```

## Quick Reference

| Command | Description | Aliases |
|---------|-------------|---------|
| `list` | Show declared dependencies with installed versions | `ls` |
| `outdated` | Check for available updates | - |
| `update` | Apply dependency updates | - |
| `scan` | Find matching package files | - |
| `config` | Show, validate, or scaffold configuration | - |
| `version` | Print version and build information | - |
| `help` | Show help for any command | - |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to custom config file (default: `.goupdate.yml`) |
| `--directory` | `-d` | Working directory for scanning (default: `.`) |
| `--verbose` | | Enable verbose debug output with troubleshooting hints |
| `--help` | `-h` | Show help for command |

### Verbose Mode

The `--verbose` flag enables detailed debug output that helps troubleshoot issues:

```bash
goupdate list --verbose
goupdate config --validate --verbose
```

When enabled, verbose mode provides:
- Detailed error messages with schema information
- Field type expectations and valid keys for configuration errors
- Documentation references for resolving issues
- Debug output showing internal processing steps

## Output Format Flag

All main commands (`scan`, `list`, `outdated`, `update`) support alternative output formats for scripting and integration:

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output format: `json`, `csv`, `xml` (default: table) |

**Examples:**
```bash
goupdate list --output json
goupdate outdated -o csv
goupdate scan --output xml
```

When using these flags:
- Live table output is replaced with a progress indicator (shown on stderr)
- The structured output is written to stdout after processing completes
- For `update`, confirmation prompts are skipped (like `--yes`)
- Output includes summary statistics, package data, warnings, and errors

## list

Resolve declared constraints, enrich them with installed versions from lock files, and present the results in a table.

```bash
goupdate list [file...]
goupdate ls [file...]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--type` | `-t` | Filter by dependency type (`prod`, `dev`, `all`) | `all` |
| `--package-manager` | `-p` | Filter by package manager name | `all` |
| `--rule` | `-r` | Filter by rule key (comma-separated) | `all` |
| `--name` | `-n` | Filter by package name (comma-separated) | - |
| `--group` | `-g` | Filter by group (comma-separated) | - |
| `--config` | `-c` | Custom config file path | `.goupdate.yml` |
| `--directory` | `-d` | Working directory | `.` |
| `--output` | `-o` | Output format: `json`, `csv`, `xml` | `table` |

### Output Columns

| Column | Description |
|--------|-------------|
| `RULE` | Rule key that matched the file |
| `PM` | Package manager identifier |
| `TYPE` | Dependency type (`prod` or `dev`) |
| `CONSTRAINT` | Version constraint type (e.g., `Compatible (^)`, `Patch (~)`) |
| `VERSION` | Declared version in manifest |
| `INSTALLED` | Version from lock file |
| `STATUS` | Lock file resolution status |
| `GROUP` | Package group (if configured) |
| `NAME` | Package name |

### Status Values

| Status | Icon | Description |
|--------|------|-------------|
| `LockFound` | 🟢 | Package found in lock file |
| `SelfPinned` | 📌 | Manifest is its own lock (e.g., requirements.txt) |
| `LockMissing` | 🟠 | Lock file doesn't exist |
| `NotInLock` | 🔵 | Lock file exists but package not found |
| `VersionMissing` | ⛔ | No concrete version available |
| `NotConfigured` | ⚪ | Lock file not supported for this rule |
| `Floating` | ⛔ | Floating constraint cannot auto-update |
| `Ignored` | 🚫 | Package excluded by ignore pattern or package_overrides |

## outdated

Check for available updates for each package using configured CLI commands.

```bash
goupdate outdated [file...]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--type` | `-t` | Filter by dependency type | `all` |
| `--package-manager` | `-p` | Filter by package manager | `all` |
| `--rule` | `-r` | Filter by rule key (comma-separated) | `all` |
| `--name` | `-n` | Filter by package name (comma-separated) | - |
| `--group` | `-g` | Filter by group (comma-separated) | - |
| `--major` | | Show major updates (lift constraints) | `false` |
| `--minor` | | Show minor updates (pin major) | `false` |
| `--patch` | | Show patch updates (pin major.minor) | `false` |
| `--no-timeout` | | Disable command timeouts | `false` |
| `--skip-preflight` | | Skip command validation | `false` |
| `--continue-on-fail` | | Continue after failures (exit 1 for partial success) | `false` |
| `--config` | `-c` | Custom config file path | `.goupdate.yml` |
| `--directory` | `-d` | Working directory | `.` |
| `--output` | `-o` | Output format: `json`, `csv`, `xml` | `table` |

### Output Columns

| Column | Description |
|--------|-------------|
| `RULE` | Rule key that matched the file |
| `PM` | Package manager identifier |
| `TYPE` | Dependency type |
| `CONSTRAINT` | Version constraint (with override indicator if applicable) |
| `VERSION` | Declared version |
| `INSTALLED` | Currently installed version |
| `MAJOR` | Latest major update available |
| `MINOR` | Latest minor update available |
| `PATCH` | Latest patch update available |
| `STATUS` | Update status |
| `GROUP` | Package group |
| `NAME` | Package name |
| `ERROR` | Error message (if any) |

### Status Values

| Status | Icon | Description |
|--------|------|-------------|
| `UpToDate` | 🟢 | No updates available |
| `Outdated` | 🟠 | Updates available |
| `NotConfigured` | ⚪ | Cannot check updates |
| `Failed` | ❌ | Command failed (with exit code) |

## update

Plan and apply dependency updates using rule-level configuration.

```bash
goupdate update [file...]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--type` | `-t` | Filter by dependency type | `all` |
| `--package-manager` | `-p` | Filter by package manager | `all` |
| `--rule` | `-r` | Filter by rule key (comma-separated) | `all` |
| `--name` | `-n` | Filter by package name (comma-separated) | - |
| `--group` | `-g` | Filter by group (comma-separated) | - |
| `--major` | | Force major upgrades | `false` |
| `--minor` | | Force minor upgrades | `false` |
| `--patch` | | Force patch upgrades | `false` |
| `--incremental` | | Force incremental updates (one version step at a time) | `false` |
| `--dry-run` | | Plan without applying changes | `false` |
| `--skip-lock` | | Skip lock/install commands | `false` |
| `--yes` | `-y` | Skip confirmation prompt | `false` |
| `--no-timeout` | | Disable command timeouts | `false` |
| `--continue-on-fail` | | Continue after failures | `false` |
| `--skip-preflight` | | Skip command validation | `false` |
| `--skip-system-tests` | | Skip all system tests | `false` |
| `--system-test-mode` | | Override system test run mode (`after_each`, `after_all`, `none`) | config value |
| `--config` | `-c` | Custom config file path | `.goupdate.yml` |
| `--directory` | `-d` | Working directory | `.` |
| `--output` | `-o` | Output format: `json`, `csv`, `xml` | `table` |

### Status Values

| Status | Icon | Description |
|--------|------|-------------|
| `UpToDate` | 🟢 | Already at latest |
| `Planned` | 🟡 | Update planned (dry-run) |
| `Updated` | 🟢 | Successfully updated |
| `Failed` | ❌ | Update failed |
| `NotConfigured` | ⚪ | Cannot update |

### Behavior

- Shows preview table with planned updates before confirmation
- Shows confirmation prompt unless `--dry-run` or `--yes` is specified
- Validates baseline with `list` before changes
- Executes lock/install commands after manifest edits
- Runs system tests after updates (if configured)
- Rolls back group on failure (including test failures)
- Honors `incremental` config or `--incremental` flag for step-by-step updates
- Shows final summary with counts and remaining available updates

### System Tests

When `system_tests` is configured, tests run automatically during updates:

| Flag | Effect |
|------|--------|
| `--skip-system-tests` | Skip all tests (preflight and post-update) |
| `--system-test-mode after_each` | Run tests after each package (max safety) |
| `--system-test-mode after_all` | Run tests once after all updates (faster) |
| `--system-test-mode none` | Only run preflight tests |
| `--dry-run` | System tests are skipped |

See [System Tests Guide](./system-tests.md) for configuration details.

### Incremental Mode

When `--incremental` is specified or a package is configured for incremental updates:

- Updates are applied one version step at a time (patch → minor → major)
- If a patch update is available, it's applied first
- Run the command again to apply the next available update
- Respects scope flags: `--minor` allows patch and minor, `--major` allows all
- Useful for testing updates progressively rather than jumping to latest

**Example workflow:**
```bash
# Apply smallest available updates
goupdate update --incremental --yes

# If more updates are available, run again
goupdate update --incremental --yes

# Repeat until fully up-to-date
```

## scan

Walk the working directory and show which files match which rules.

```bash
goupdate scan
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--file` | `-f` | Filter by file path patterns (comma-separated, supports globs) | - |
| `--directory` | `-d` | Directory to scan | `.` |
| `--config` | `-c` | Custom config file | `.goupdate.yml` |
| `--output` | `-o` | Output format: `json`, `csv`, `xml` | `table` |

### Output Columns

| Column | Description |
|--------|-------------|
| `RULE` | Rule key that matched |
| `PM` | Package manager identifier |
| `FORMAT` | File format (json, raw, toml, xml) |
| `FILE` | Relative path to matched file |
| `STATUS` | File validation status (valid/invalid) |

## config

Show configuration details, validate configuration, or scaffold a new `.goupdate.yml`.

```bash
goupdate config [--show-defaults|--show-effective|--init|--validate]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--show-defaults` | | Print embedded default configuration |
| `--show-effective` | | Show merged configuration (defaults + local) |
| `--init` | | Create minimal `.goupdate.yml` template |
| `--validate` | | Validate configuration file (rejects unknown fields) |
| `--config` | `-c` | Config file path to validate (default: `.goupdate.yml`) |

### Configuration Validation

Use `--validate` to verify your configuration file before running commands:

```bash
# Validate default config location (.goupdate.yml)
goupdate config --validate

# Validate specific config file
goupdate config --validate -c ./custom-config.yml
```

**What validation checks:**
- YAML syntax errors
- Unknown fields (typos in field names)
- Invalid field values
- Missing required configurations

**Example output for invalid config:**
```
❌ Configuration validation failed for: .goupdate.yml

  ERROR: unknown field in config: manager_typo
  WARNING: rules.npm.outdated.commands: missing {{package}} placeholder

💡 See docs/configuration.md for valid configuration options
```

**Preflight validation:** All commands (`scan`, `list`, `outdated`, `update`) automatically validate the configuration file before running. This catches configuration errors early with helpful error messages.

**Verbose validation output:** Use `--verbose` for detailed schema information when validation fails:

```bash
goupdate config --validate --verbose
```

Example verbose output for an unknown field:
```
❌ Configuration validation failed for: .goupdate.yml

  ERROR: unknown field 'command' (line 15) (did you mean 'commands'?)
    Valid keys: commands, env, format, extraction, versioning, exclude_versions, exclude_version_patterns, timeout_seconds
    📖 See: docs/configuration.md#outdated

💡 See docs/configuration.md for valid configuration options
```

## version

Print version and build information about goupdate.

```bash
goupdate version
```

### Output

```
goupdate version 1.0.0
  Build time: 2024-01-15T10:30:00Z
  Git commit: abc1234
  Go version: go1.21.0
  OS/Arch:    linux/amd64
```

### Build Information

Version information is embedded at build time using Go ldflags:

```bash
# Build with version information
go build -ldflags="-X github.com/ajxudir/goupdate/cmd.Version=1.0.0 \
                   -X github.com/ajxudir/goupdate/cmd.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
                   -X github.com/ajxudir/goupdate/cmd.GitCommit=$(git rev-parse --short HEAD)"
```

The Makefile automatically sets these values when using `make build`.

## help

Show help for goupdate or any specific command.

```bash
goupdate help
goupdate help <command>
goupdate <command> --help
```

### Examples

```bash
# Show general help
goupdate help

# Show help for update command
goupdate help update
goupdate update --help

# Show help for config command
goupdate help config
```

### Help Output

Running `goupdate help` shows:
- Available commands with descriptions
- Global flags available to all commands
- Usage examples

Running `goupdate help <command>` shows:
- Command-specific description
- All available flags with defaults
- Usage examples

## Supported Rules

| Rule | PM | Description | Manifest | Lock File |
|------|-----|-------------|----------|-----------|
| `npm` | js | Node.js (npm) | `package.json` | `package-lock.json` |
| `pnpm` | js | Node.js (pnpm) | `package.json` | `pnpm-lock.yaml` |
| `yarn` | js | Node.js (yarn) | `package.json` | `yarn.lock` |
| `mod` | golang | Go modules | `go.mod` | `go.sum` |
| `composer` | php | PHP Composer | `composer.json` | `composer.lock` |
| `requirements` | python | Python pip | `requirements.txt` | - |
| `pipfile` | python | Python Pipenv | `Pipfile` | `Pipfile.lock` |
| `msbuild` | dotnet | .NET MSBuild | `*.csproj`, `*.vbproj`, `*.fsproj` | `packages.lock.json` |
| `nuget` | dotnet | .NET NuGet | `packages.config` | `packages.lock.json` |

## Examples

```bash
# List all packages
goupdate list

# List only production dependencies
goupdate list --type=prod

# List packages for a specific rule
goupdate list --rule=npm
goupdate list -r npm

# Filter by package name (comma-separated)
goupdate list --name=lodash,express
goupdate list -n lodash

# Filter by group (comma-separated)
goupdate list --group=core,utils
goupdate list -g core

# Check for outdated packages
goupdate outdated

# Check with major version updates allowed
goupdate outdated --major

# Check specific packages for updates
goupdate outdated --name=react,typescript

# Dry-run update planning
goupdate update --dry-run

# Apply updates without confirmation
goupdate update --yes

# Apply only patch updates
goupdate update --patch --yes

# Update specific packages only
goupdate update --name=lodash --yes

# Update packages in a specific group
goupdate update --group=core --yes

# Scan for package files
goupdate scan

# Show effective configuration
goupdate config --show-effective
```

## Output Format Examples

```bash
# Export package list as JSON
goupdate list --output json > packages.json

# Export outdated packages as CSV for spreadsheet analysis
goupdate outdated --output csv > outdated.csv

# Export scan results as XML
goupdate scan --output xml > scan.xml

# Pipe JSON output to jq for filtering
goupdate outdated --output json | jq '.packages[] | select(.status == "Outdated")'

# Use in CI/CD scripts
if goupdate outdated --output json | jq -e '.summary.outdated_packages > 0' > /dev/null; then
  echo "Updates available"
fi
```

### JSON Output Structure

All JSON outputs follow a consistent structure:

```json
{
  "summary": {
    // Command-specific summary statistics
  },
  "packages": [
    // Array of package objects
  ],
  "warnings": [
    // Optional array of warning messages
  ],
  "errors": [
    // Optional array of error messages
  ]
}
```

### CSV Output Structure

CSV outputs include a header row followed by data rows. All columns from the table output are included.

## Related Documentation

- [Configuration Guide](./configuration.md) - YAML schema and options
- [Features Overview](./features.md) - Capabilities and supported ecosystems
- [System Tests Guide](./system-tests.md) - Automated testing during updates
- [Architecture Documentation](./architecture/) - Internals for contributors
