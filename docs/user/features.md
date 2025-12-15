# Features Overview

`goupdate` provides a single, auditable view of dependencies across mixed ecosystems. It reads manifests, consults lock files, and produces normalized reports that highlight declared constraints versus installed versions.

## Table of Contents

- [Key Benefits](#key-benefits)
- [Supported Ecosystems](#supported-ecosystems)
- [Core Capabilities](#core-capabilities)
  - [System Tests](#system-tests)
- [Configuration Features](#configuration-features)
- [Output Formats](#output-formats)
- [How It Works](#how-it-works)
- [Test Coverage](#test-coverage)
- [Related Documentation](#related-documentation)

## Key Benefits

| Feature | Description |
|---------|-------------|
| **Single view** | Normalize manifests and lock files from mixed ecosystems into one report |
| **Enterprise-ready** | YAML-based rules enable shared standards with repo-level flexibility |
| **Fast onboarding** | Ships with defaults covering common package managers |
| **Actionable signals** | Missing lock files or packages surfaced as explicit statuses |

## Supported Ecosystems

| Ecosystem | Rule | Package Manager | Manifest | Lock File |
|-----------|------|-----------------|----------|-----------|
| JavaScript | `npm` | npm | `package.json` | `package-lock.json` |
| JavaScript | `pnpm` | pnpm | `package.json` | `pnpm-lock.yaml` |
| JavaScript | `yarn` | Yarn | `package.json` | `yarn.lock` |
| Go | `mod` | Go modules | `go.mod` | `go.sum` |
| PHP | `composer` | Composer | `composer.json` | `composer.lock` |
| Python | `requirements` | pip | `requirements.txt` | - |
| Python | `pipfile` | Pipenv | `Pipfile` | `Pipfile.lock` |
| .NET | `msbuild` | MSBuild | `*.csproj`, `*.vbproj`, `*.fsproj` | `packages.lock.json` |
| .NET | `nuget` | NuGet | `packages.config` | `packages.lock.json` |

Additional package managers can be added via configuration.

## Core Capabilities

### Discovery

- Walks working directory with include/exclude patterns per rule
- Respects `working_dir` from config or `--directory` flag for monorepo subtrees
- Automatically detects manifest files based on configured patterns

### Parsing and Normalization

| Feature | Description |
|---------|-------------|
| Dynamic parser | Loads each manifest format using appropriate parser |
| Field mapping | Maps fields into consistent package entries |
| Custom extraction | Supports nested structures via YAML configuration |
| Package ignoring | Excludes packages by name to reduce noise |

### Lock File Awareness

| Status | Icon | Meaning |
|--------|------|---------|
| `LockFound` | 🟢 | Package found in lock file with version |
| `SelfPinned` | 📌 | Manifest is its own lock (e.g., requirements.txt) |
| `LockMissing` | 🟠 | Lock file doesn't exist |
| `NotInLock` | 🔵 | Lock file exists but package not found |
| `VersionMissing` | ⛔ | No concrete version available |
| `NotConfigured` | ⚪ | Lock file not supported for this rule |
| `Floating` | ⛔ | Floating constraint cannot auto-update |

### Version Constraint Recognition

| Constraint | npm/yarn | Composer | Go | Python |
|------------|----------|----------|-----|--------|
| Exact | `1.0.0` | `1.0.0` | `v1.0.0` | `==1.0.0` |
| Compatible | `^1.0.0` | `^1.0` | - | `~=1.0` |
| Patch | `~1.0.0` | `~1.0.0` | - | - |
| Min | `>=1.0.0` | `>=1.0` | - | `>=1.0.0` |
| Max | `<2.0.0` | `<2.0` | - | `<2.0.0` |
| Range | `>=1.0 <2.0` | `>=1.0,<2.0` | - | `>=1.0,<2.0` |
| Wildcard | `1.x` / `*` | `1.*` / `*` | - | `*` |

### Pre-flight Validation

Before running `outdated` or `update` commands, goupdate validates that required package manager commands are available:

| Package Manager | Required Commands | Installation Hint |
|-----------------|-------------------|-------------------|
| npm/pnpm/yarn | `npm`, `pnpm`, `yarn` | Install Node.js |
| Go | `go` | Install Go |
| Composer | `composer` | Install Composer |
| Python | `pip`, `pipenv` | Install Python |
| .NET | `nuget`, `dotnet` | Install .NET SDK |

Skip validation with `--skip-preflight` if commands are resolved through other means.

### System Tests

System tests validate application health before, during, and after dependency updates. Configure custom test suites (unit tests, e2e tests, Playwright, etc.) to run automatically:

| Run Mode | Description | Use Case |
|----------|-------------|----------|
| `preflight` | Run before any updates | Ensure app works before changes |
| `after_each` | Run after each package update | Maximum safety, identifies breaking packages |
| `after_all` | Run once after all updates | Fast for many packages |

**Example configuration:**
```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: unit-tests
      commands: npm test
      timeout_seconds: 120
    - name: e2e-tests
      commands: npx playwright test
      timeout_seconds: 300
```

**CLI flags:**
- `--skip-system-tests`: Skip all system tests
- `--system-test-mode <mode>`: Override run mode (after_each, after_all, none)

See [System Tests Documentation](./system-tests.md) for TDD-based automated maintenance patterns.

## Configuration Features

### Extensibility

| Feature | Description |
|---------|-------------|
| `extends` | Inherit from other config files or embedded defaults |
| Rule blocks | Add new package managers by defining rule blocks |
| Groups | Organize packages into rollout cohorts |
| Incremental | Mark dependencies for step-by-step version advances |

### Version Filtering

| Config Key | Purpose |
|------------|---------|
| `exclude_versions` | Regex patterns to filter versions (rule-level) |
| `outdated.exclude_versions` | Exact versions to exclude (outdated command) |
| `outdated.exclude_version_patterns` | Regex patterns for outdated command |
| `incremental` | Force nearest-step updates instead of latest |

## Output Formats

All main commands (`scan`, `list`, `outdated`, `update`) support multiple output formats via the `--output` flag:

| Format | Usage | Use Case |
|--------|-------|----------|
| Table | (default) | Interactive terminal output with live updates |
| CSV | `--output csv` | Spreadsheet analysis, data import |
| JSON | `--output json` | Programmatic processing, CI/CD integration |
| XML | `--output xml` | Enterprise tooling, legacy system integration |

### Structured Output Features

- **Clean Output**: Progress messages are completely suppressed (no stderr noise)
- **Consistent Structure**: All formats include summary, packages, warnings, and errors
- **Pure Stdout**: Only structured data is written to stdout
- **CI-Friendly**: Non-interactive when using structured formats
- **Flag Validation**: `--verbose` is rejected; `update` requires `--yes` or `--dry-run`

## How It Works

```
Load Config ──► Merge defaults + local overrides
     │
     ▼
Discover Files ──► Apply include/exclude patterns
     │
     ▼
Parse Manifests ──► Extract packages using format rules
     │
     ▼
Resolve Locks ──► Read lock files for installed versions
     │
     ▼
Report Results ──► Output tables or structured formats
```

1. **Load configuration:** Merge embedded defaults with local overrides or extended files
2. **Discover manifests:** Apply include/exclude patterns to find files for each rule
3. **Parse packages:** Convert manifests into normalized packages using format rules
4. **Resolve lock files:** Check lock files to record installed versions and status
5. **Report results:** Present data in tables with ecosystem and type filters

## Test Coverage

| Directory | Purpose |
|-----------|---------|
| `pkg/testdata/` | Working test cases for each rule |
| `pkg/_testdata/` | Error cases and edge case scenarios |
| `examples/` | Runnable example projects |

Run tests with:
```bash
go test ./...
make coverage
make coverage-func
```

## Related Documentation

- [CLI Reference](./cli.md) - Command usage and flags
- [Configuration Guide](./configuration.md) - YAML schema and options
- [System Tests Guide](./system-tests.md) - Automated testing during updates
- [Architecture Documentation](./architecture/) - Internals for contributors
- [Testing Guide](./testing.md) - Test-driven development and coverage
