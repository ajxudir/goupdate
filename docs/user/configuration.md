# Configuration

This guide explains how to control discovery, parsing, and lock-file handling with YAML configuration.

## Quick Start

```bash
# 1. See what's available by default
goupdate config --show-defaults

# 2. Create a config template
goupdate config --init

# 3. Verify your config works
goupdate scan                    # Check file detection
goupdate list                    # Check package parsing
goupdate config --show-effective # See merged config
```

**Most users only need 3-5 lines of config** — see [Simple Examples](#simple-customization-examples) below.

---

## Table of Contents

- [Quick Start](#quick-start)
- [File Locations](#file-locations)
- [Simple Customization Examples](#simple-customization-examples)
- [Configuration Reference](#configuration-reference)
  - [Top-Level Options](#top-level-options)
  - [Rule Options](#rule-options)
  - [Outdated Options](#outdated-options)
  - [Update Options](#update-options)
- [Version Comparison](#version-comparison-for-outdated)
- [Adding New Package Managers](#customizing-and-adding-rules)
- [Environment Variables](#environment-variables)

---

## File Locations

- **Defaults:** Embedded `pkg/default.yml` defines every supported rule. View them with `goupdate config --show-defaults`.
- **Project overrides:** `.goupdate.yml` in the working directory is loaded automatically. Use `--config` to point at another path.
- **Extends:** The `extends` field lets you layer multiple configs (including `default`) to compose rules from shared snippets.

## Simple customization examples

Most users only need a few lines of config. These examples show minimal configurations for common use cases.

### Group packages for coordinated updates

```yaml
# .goupdate.yml
extends: [default]
rules:
  npm:
    groups:
      frontend:
        - react
        - react-dom
        - vite
      backend:
        - express
        - mongodb
```

### Default pre-release filtering

When you extend `default`, the following version patterns are **automatically excluded**:

| Pattern | Matches |
|---------|---------|
| `(?i)[._-]alpha` | `1.0.0-alpha`, `2.0.0_alpha1` |
| `(?i)[._-]beta` | `1.0.0-beta`, `2.0.0.beta2` |
| `(?i)[._-]rc` | `1.0.0-rc1`, `2.0.0_rc.2` |
| `(?i)[._-]canary` | `1.0.0-canary` |
| `(?i)[._-]dev` | `1.0.0-dev`, `2.0.0.dev1` |
| `(?i)[._-]snapshot` | `1.0.0-SNAPSHOT` |
| `(?i)[._-]nightly` | `1.0.0-nightly.20231201` |
| `(?i)[._-]preview` | `1.0.0-preview` |
| `(?i)[._-]next` | `1.0.0-next` |

This ensures stable versions are recommended by default. To allow pre-releases:

```yaml
extends: [default]
rules:
  npm:
    exclude_versions: []  # Clear all default patterns
```

### Add custom version filters

Replace default filtering with your own patterns:

```yaml
extends: [default]
rules:
  npm:
    exclude_versions:
      - "(?i)beta"      # Must re-include if needed
      - "(?i)preview"   # Must re-include if needed
      - "(?i)alpha"     # Must re-include if needed
      - "(?i)unstable"  # Custom pattern
```

> **Note:** List fields like `exclude_versions`, `ignore`, and `include` are **completely overwritten** when specified in extending configs. To keep default patterns, you must explicitly re-include them.

### Ignore specific packages

```yaml
extends: [default]
rules:
  npm:
    ignore:
      - eslint
      - prettier
```

### Per-package overrides

```yaml
extends: [default]
rules:
  npm:
    package_overrides:
      react:
        constraint: "~"  # Use tilde constraint for react
      lodash:
        ignore: true     # Never update lodash
```

### Combine multiple customizations

```yaml
extends: [default]
rules:
  npm:
    groups:
      critical:
        - express
        - helmet
    exclude_versions:
      - "(?i)rc"
    ignore:
      - typescript
```

All examples above use `extends: [default]` to inherit built-in rules, then override only the specific fields needed.

**Merge behavior:**
- **List fields** (`include`, `exclude`, `ignore`, `exclude_versions`, `incremental`): **Overwritten** completely when specified
- **Map fields** (`rules`, `groups`, `package_overrides`, `fields`): **Merged** by key, with extending config values taking priority
- **System tests** (`system_tests.tests`): **Merged** by test `name`, allowing override of specific tests

To clear an inherited list, use an empty array `[]`. To remove a single item from a list, you must specify the complete list you want.

---

## Configuration Reference

### Top-Level Options

| Option | Type | Description |
|--------|------|-------------|
| `extends` | `[]string` | Inherit from other configs (use `default` for built-ins) |
| `working_dir` | `string` | Base directory for file discovery (default: `.`) |
| `rules` | `map` | Package manager definitions (see below) |
| `system_tests` | `object` | System test configuration (see [System Tests](./system-tests.md)) |

### Top-level schema

```yaml
working_dir: "./subdir" # optional
extends: ["default", "../shared-rules.yml"]
rules:
  npm: &js_rule
    manager: js
    include: ["**/package.json"]
    exclude: ["**/node_modules/**"]
    format: json
    fields: { name: "name", version: "version" }
    ignore: ["node"]
    exclude_versions:
      - "(?i)preview"
    groups:
      runtime:
        - service-core
        - service-api
    constraint_mapping: { caret: "^", tilde: "~" }
    latest_mapping:
      default: { latest: "*" }
      react: ["next", "rc"]
    package_overrides:
      react:
        constraint: "^"
      tslib:
        ignore: true
    extraction:
      path: "dependencies"
      name_attr: "name"
      version_attr: "version"
    lock_files:
      - files: ["package-lock.json"]
        format: json
        extraction:
          pattern: '(?m)"(?P<n>[^"]+)":\s*\{[^}]*"version":\s*"(?P<version>[^"]+)"'
    incremental:
      - "service-.*"
  pnpm:
    <<: *js_rule
    lock_files:
      - files: ["pnpm-lock.yaml"]
        commands: |
          pnpm ls --json --depth=0 2>/dev/null || exit 0
        timeout_seconds: 60
  yarn:
    <<: *js_rule
    lock_files:
      - files: ["yarn.lock"]
        commands: |
          yarn list --json --depth=0 2>/dev/null || exit 0
        timeout_seconds: 60
```

The bundled defaults ship three JavaScript rules (npm, pnpm, and yarn) that share manifest parsing while mapping to their respective lock files.

- **working_dir:** Default root when no `--directory` flag is provided. The loader in `pkg/config.go` ensures discovery and parsing run from this directory so excludes and includes resolve correctly.
- **extends:** Ordered list of other config files or `default`. Each file is loaded relative to the current config file path and processed in sequence before the local rules are applied. List fields are overwritten (not merged), while map fields merge by key.
- **rules:** Map of rule keys to package manager definitions. Keys are used in output tables to identify which parser handled a file. Rule fields hold rollout `groups` and rule-scoped `exclude_versions` so package-manager-specific names and filters do not collide. Legacy top-level `groups` and `default_exclude_version_patterns` still load for backward compatibility, but rule definitions override them when set.

### Rule Options

Each rule under `rules:` controls discovery, parsing, and lock-file handling:

#### Core Options

| Option | Type | Description | Example |
|--------|------|-------------|---------|
| `enabled` | `bool` | Enable or disable this rule (defaults to true) | `false` |
| `manager` | `string` | Package manager identifier for `--package-manager` filter | `js`, `python`, `golang` |
| `include` | `[]string` | Glob patterns to find manifest files | `["**/package.json"]` |
| `exclude` | `[]string` | Glob patterns to skip | `["**/node_modules/**"]` |
| `format` | `string` | Parser format | `json`, `yaml`, `xml`, `raw` |
| `fields` | `map` | Field mappings for package extraction | `{ name: "name", version: "version" }` |
| `self_pinning` | `bool` | Manifest file is its own lock file (e.g., requirements.txt) | `true` |

#### Filtering Options

| Option | Type | Description | Example |
|--------|------|-------------|---------|
| `ignore` | `[]string` | Package names to exclude from reports | `["eslint", "prettier"]` |
| `exclude_versions` | `[]string` | Regex patterns to filter versions | `["(?i)beta", "(?i)rc"]` |
| `groups` | `map` | Named package groups for coordinated updates | See example below |
| `packages` | `map` | Per-package update settings (e.g., `with_all_dependencies`) | See example below |
| `incremental` | `[]string` | Packages requiring step-by-step updates | `["react", "service-.*"]` |

**Groups example (with all dependencies option):**
```yaml
groups:
  frontend:
    - react
    - react-dom
  # Group with transitive dependency flag for composer
  laravel:
    with_all_dependencies: true
    packages:
      - laravel/framework
      - laravel/sanctum
```

**Per-package settings example (Composer with_all_dependencies):**
```yaml
# For composer packages that need transitive dependencies updated
packages:
  sentry/sentry-laravel:
    with_all_dependencies: true
  intervention/image:
    with_all_dependencies: true
```

The `with_all_dependencies` option adds the `-W` flag to composer update commands, which updates transitive dependencies. This is required for packages like Laravel framework that have tightly-coupled sub-packages.

#### Override Options

| Option | Type | Description |
|--------|------|-------------|
| `constraint_mapping` | `map` | Normalize constraint tokens | `{ caret: "^", tilde: "~" }` |
| `latest_mapping` | `map` | Map "latest" indicators to canonical values | `{ default: { latest: "*" } }` |
| `package_overrides` | `map` | Per-package customization (ignore, constraint, etc.) | See example below |

**Package overrides example:**
```yaml
package_overrides:
  react:
    constraint: "~"     # Use tilde constraint
  lodash:
    ignore: true        # Never update
```

#### Extraction Options (for nested structures)

| Option | Type | Description | Example |
|--------|------|-------------|---------|
| `extraction.path` | `string` | XPath-style path to package nodes | `Project/ItemGroup/PackageReference` |
| `extraction.pattern` | `string` | Regex pattern for raw format extraction | `(?P<n>[\w-]+)==(?P<version>[\d.]+)` |
| `extraction.name_attr` | `string` | Attribute containing package name | `Include`, `id` |
| `extraction.version_attr` | `string` | Attribute containing version | `Version`, `version` |
| `extraction.name_element` | `string` | Element name containing package name (XML) | `Package` |
| `extraction.version_element` | `string` | Element name containing version (XML) | `Version` |
| `extraction.dev_attr` | `string` | Attribute indicating dev dependency | `developmentDependency` |
| `extraction.dev_value` | `string` | Attribute value marking dev dependency | `true` |
| `extraction.dev_element` | `string` | Element name indicating dev dependency (XML) | `PrivateAssets` |
| `extraction.dev_element_value` | `string` | Element text value marking dev dependency | `all` |

#### Lock File Options

Configure how installed versions are extracted from lock files. Use EITHER file-based parsing (format + extraction) OR command-based parsing (commands).

| Option | Type | Description |
|--------|------|-------------|
| `lock_files[].files` | `[]string` | Lock file patterns (for detection and rule conflict resolution) |
| `lock_files[].format` | `string` | Lock file format for file-based parsing: `json`, `raw` |
| `lock_files[].extraction.pattern` | `string` | Regex pattern with named groups `(?P<n>...)` and `(?P<version>...)` |
| `lock_files[].commands` | `string` | Shell command to extract versions (alternative to file parsing) |
| `lock_files[].env` | `map` | Environment variables for commands |
| `lock_files[].timeout_seconds` | `int` | Command timeout (default: 60) |
| `lock_files[].command_extraction` | `map` | Configure how to parse command output |

**File-based extraction (uses regex on lock file content):**
```yaml
lock_files:
  - files: ["**/composer.lock"]
    format: json
    extraction:
      pattern: '(?s)"name":\s*"(?P<n>[^"]+)"\s*,\s*"version":\s*"(?P<version>[^"]+)"'
```

**Command-based extraction (runs command and parses output):**

For complex lock files or when maximum compatibility is needed across versions:
```yaml
lock_files:
  - files: ["**/package-lock.json"]  # For detection/rule conflict resolution
    commands: |
      npm ls --json --package-lock-only 2>/dev/null || exit 0
    timeout_seconds: 60
```

When `commands` is set, `format` and `extraction` are ignored. The command output should be JSON:
- Object format: `{"package-name": "version", ...}`
- Array format: `[{"name": "package-name", "version": "1.0.0"}, ...]`
- npm ls format: `{"dependencies": {"pkg": {"version": "ver"}}}`

Available placeholders: `{{lock_file}}`, `{{base_dir}}`

### Outdated Options

Configure how `goupdate outdated` queries for available versions under `rules.<name>.outdated`:

| Option | Type | Description |
|--------|------|-------------|
| `commands` | `string` | Shell command to get versions (supports `{{package}}`, `{{version}}` placeholders) |
| `format` | `string` | Output format: `json`, `yaml`, or `raw` |
| `extraction.json_key` | `string` | Dot-path to version array in JSON |
| `extraction.yaml_key` | `string` | Dot-path to version array in YAML |
| `extraction.pattern` | `string` | Regex for raw output (use `(?P<version>...)`) |
| `env` | `map` | Environment variables for command |
| `exclude_versions` | `[]string` | Exact versions to exclude |
| `exclude_version_patterns` | `[]string` | Regex patterns to exclude |
| `timeout_seconds` | `int` | Command timeout |

**Example:**
```yaml
outdated:
  commands: |
    curl -s "https://registry.npmjs.org/{{package}}" |
    jq -r '.versions | keys[]'
  format: raw
  extraction:
    pattern: "^(?P<version>[0-9]+\\.[0-9]+\\.[0-9]+)$"
  exclude_version_patterns:
    - "(?i)beta"
    - "(?i)alpha"
```

### Update Options

Configure how `goupdate update` applies changes under `rules.<name>.update`:

| Option | Type | Description |
|--------|------|-------------|
| `commands` | `string` | Command to regenerate lock files |
| `env` | `map` | Environment variables for command |
| `group` | `string` | Assign packages to a named group for atomic updates |
| `timeout_seconds` | `int` | Command timeout |

**Example:**
```yaml
update:
  commands: |
    npm install --package-lock-only --ignore-scripts
  group: npm-deps  # Group packages for atomic lock command execution
```

## Lock-file resolution

For each rule with `lock_files` defined, `pkg/lock/resolve.go` attempts to read the configured files. The result is attached to every package as `InstallStatus` and `InstalledVersion`:

| Status | Icon | Description |
|--------|------|-------------|
| `LockFound` | 🟢 | Lock file contains the package with its version |
| `SelfPinned` | 📌 | Manifest is its own lock file (e.g., requirements.txt) |
| `NotInLock` | 🔵 | Lock file exists but package not found |
| `LockMissing` | 🟠 | No configured lock file found in the directory |
| `NotConfigured` | ⚪ | Rule has no `lock_files` configuration |

Rules without `lock_files` default to `NotConfigured` for installed version lookups.

If a manifest declares a latest/unspecified version but no installed version can be resolved (missing lock file, package not in lock, or unconfigured lock support), `goupdate` emits a warning because it cannot determine whether an update is required.

## Customizing and adding rules

- Start from `goupdate config --show-defaults` to copy an existing rule and adjust `include`, `format`, or extraction fields.
- To add a new package manager, define a new key under `rules` with the fields above. As long as a parser format exists (or you add one under `pkg/formats.go`), the CLI will automatically pick it up during discovery and `list` reporting.
- Keep shared patterns (like common excludes) in a base file and reference it via `extends` to avoid duplication across repos.

---

## Version Comparison for `outdated`

`goupdate outdated` only prints releases newer than the currently installed (or declared) version. By default, the comparison is flexible: `v1.0.0` and `1.0.0` are treated the same, missing patch numbers are assumed to be `0` (so `15.4` becomes `15.4.0`), and numeric segments are pulled out of tags like `alpine-15-4` or `redis7.2.3-alpine` without additional configuration.

When package managers use different conventions, you can control how versions are interpreted via the `versioning` block under each rule’s `outdated` settings:

```yaml
rules:
  npm:
    manager: js
    include: ["**/package.json"]
    fields: { dependencies: prod }
    outdated:
      commands: |
        gh release list --repo vercel/next.js --limit 20 --json tagName
      format: json
      extraction:
        json_key: "tagName"
      versioning:
        format: regex
        regex: "(?i)next(?:js)?[-v]?(?P<major>\\d+)(?:[._-](?P<minor>\\d+))?(?:[._-](?P<patch>\\d+))?"
        sort: desc
```

- **format:** Choose `semver` (default), `numeric` (treats the full number as the major component, useful for Moodle plugin versions), `regex` (use a custom capture for `major`, `minor`, and `patch`), or `ordered`/`list` (respect the order returned by the command without parsing numbers—handy for git hashes or date-sorted tags). Unknown formats raise an error during filtering.
- **regex:** Optional override for extracting numeric segments. Named groups are preferred, and missing pieces default to `0` so partial tags (like `alpine-15.4`) still compare correctly.
- **sort:** `desc` (default) assumes the first entry is the newest. Set `asc` when the upstream command returns oldest-first so entries after the current version are considered newer.

**Supported Version Formats:**

| Format | Example | Notes |
|--------|---------|-------|
| Standard semver | `1.2.3`, `v1.2.3` | Default handling |
| Pre-release | `1.0.0-alpha`, `1.0.0-rc03` | Correctly distinguished from stable releases |
| 4+ segments | `1.0.0.0`, `1.0.0.1` | All segments preserved for deduplication |
| CalVer | `2024.01.15` | Year=major, month=minor, day=patch |
| Build metadata | `1.0.0+build.123` | Metadata preserved but ignored in comparison |

**Pre-release Filtering:**

Pre-release versions like `1.0.0-rc03` are treated as **distinct** from their stable counterparts (`1.0.0`). When you're on a pre-release and a stable version is available, it will be shown as an update. Default exclusion patterns filter out pre-releases from recommendations unless you explicitly clear them.

Additional examples:

- **Git hashes sorted by time:**

  ```yaml
  versioning:
    format: ordered
    sort: desc # rely on the command to return newest hashes first
  ```

- **Moodle-style numeric releases:**

  ```yaml
  versioning:
    format: numeric
  ```

These settings allow the `outdated` command to keep filtering behavior consistent even when upstream package managers expose tags, hashes, or unconventional numeric schemes.

---

## Checklist

Before committing your config:

- [ ] Run `goupdate scan` — files match intended rules
- [ ] Run `goupdate list` — packages parse correctly
- [ ] Run `goupdate config --show-effective` — merged config looks right
- [ ] Include/exclude patterns are scoped narrowly
- [ ] Lock file definitions match manifest rules

## Environment Variables

goupdate respects several environment variables for shell execution and package manager configuration.

### Shell Execution

| Variable | Description | Default |
|----------|-------------|---------|
| `SHELL` | Shell used to execute commands (lock commands, outdated commands, system tests) | `sh` (Unix), `cmd.exe` (Windows) |

Commands are executed via your shell to support aliases and shell configurations. On Unix systems, commands run with `$SHELL -l -c "command"` to load login profiles.

### Package Manager Configuration

Configure these environment variables for your package manager before running goupdate:

| Variable | Package Manager | Description |
|----------|-----------------|-------------|
| `GOPRIVATE` | Go | Comma-separated list of private module paths |
| `GOPROXY` | Go | Module proxy URL (e.g., `direct` for private repos) |
| `GONOPROXY` | Go | Modules to fetch directly without proxy |
| `GONOSUMDB` | Go | Modules to skip checksum database verification |
| `NPM_TOKEN` | npm | Authentication token for private registries |
| `COMPOSER_AUTH` | Composer | JSON string with authentication credentials |

**Example for private Go modules:**
```bash
export GOPRIVATE="github.com/myorg/*,gitlab.com/mycompany/*"
export GOPROXY="https://proxy.golang.org,direct"
goupdate outdated
```

**Example for private npm packages:**
```bash
# Configure via .npmrc (preferred) or environment
echo "//registry.npmjs.org/:_authToken=${NPM_TOKEN}" >> ~/.npmrc
goupdate update
```

### Command Environment

You can pass additional environment variables to commands via config:

```yaml
rules:
  npm:
    outdated:
      env:
        CI: "true"
        NODE_ENV: "production"
      commands: |
        npm view {{package}} versions --json
    update:
      env:
        CI: "true"
      commands: |
        npm install --package-lock-only
```

Environment variables in config values are expanded using `$VAR` or `${VAR}` syntax.

---

## Security Settings

Security settings provide guardrails to prevent potential vulnerabilities. These settings can **only** be configured from the root config file (not from imported configs via `extends`).

```yaml
security:
  # Max config file size (default: 10MB = 10485760 bytes)
  max_config_file_size: 10485760

  # Max regex pattern length to prevent ReDoS attacks (default: 1000)
  max_regex_complexity: 1000

  # Allow complex regex patterns that might be vulnerable to ReDoS
  # WARNING: Only enable if you trust all regex patterns in your config
  allow_complex_regex: false

  # Allow ".." in extends paths (default: false for security)
  allow_path_traversal: false

  # Allow absolute paths in extends (default: false)
  allow_absolute_paths: false
```

### When to Adjust Security Settings

**Increase `max_config_file_size`** when:
- You have very large generated configs
- Your config includes many inline regex patterns or extensive documentation

**Increase `max_regex_complexity`** when:
- You need longer regex patterns for complex version extraction
- Error message shows "pattern length X exceeds maximum Y"

**Enable `allow_complex_regex`** when:
- You need patterns that trigger ReDoS warnings (e.g., patterns with nested quantifiers)
- You trust all regex patterns in your config chain
- **WARNING**: Only enable if necessary; this disables protection against regex denial-of-service

**Enable `allow_path_traversal`** or `allow_absolute_paths` when:
- You need to reference configs in parent directories or absolute paths
- Use case: Corporate compliance configs stored in `/etc/goupdate/` or `../shared/`

### Error Messages and Resolution

When you encounter security-related errors:

| Error | Resolution |
|-------|------------|
| `pattern length X exceeds maximum Y` | Add `security.max_regex_complexity: X` (or larger) |
| `nested quantifiers detected` | Add `security.allow_complex_regex: true` if pattern is safe |
| `config file exceeds maximum size` | Add `security.max_config_file_size: SIZE_IN_BYTES` |
| `path traversal not allowed` | Add `security.allow_path_traversal: true` |
| `absolute paths not allowed` | Add `security.allow_absolute_paths: true` |

---

## Related Documentation

- [CLI Reference](./cli.md) - Command usage and flags
- [Features Overview](./features.md) - Capabilities and supported ecosystems
- [System Tests Guide](./system-tests.md) - Automated testing during updates
- [Architecture Documentation](./architecture/) - Internals for contributors
