# Scan Command Architecture

> The `scan` command discovers package manifest files that match configured rules.

## Table of Contents

- [Command Overview](#command-overview)
- [Key Files](#key-files)
- [Data Flow](#data-flow)
- [Core Functions](#core-functions)
- [Output Format](#output-format)
- [Special Handling](#special-handling)
- [Testing](#testing)
- [Error Handling](#error-handling)
- [Related Documentation](#related-documentation)

---

## Command Overview

```bash
goupdate scan [flags]

Flags:
  -d, --directory string   Directory to scan (default ".")
  -c, --config string      Config file path
  -o, --output string      Output format: json, csv, xml (default: table)
  -f, --file string        Filter by file path patterns (comma-separated, supports globs)
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/scan.go` | Command definition and output formatting |
| `pkg/packages/detect.go` | File detection and rule matching |
| `pkg/utils/core.go` | Glob pattern matching utilities |

## Data Flow

```
Load Config ──► Detect Files ──► Validate Files ──► Print Results
```

### Step-by-Step Flow

1. **Load Configuration** (`loadConfigFunc`)
   ```go
   cfg, err := loadConfigFunc(scanConfigFlag, scanDirFlag)
   ```
   - Loads user config merged with defaults
   - Resolves `extends` chains

2. **Resolve Working Directory** (`resolveWorkingDir`)
   ```go
   workDir := resolveWorkingDir(scanDirFlag, cfg)
   cfg.WorkingDir = workDir
   ```
   - Priority: CLI flag > config `working_dir` > current directory

3. **Detect Files** (`detectFilesFunc`)
   ```go
   detected, err := detectFilesFunc(cfg, workDir)
   ```
   - Walks directory tree
   - Matches files against include/exclude patterns
   - Returns `map[string][]string` (rule → files)

4. **Validate Files** (`validateFile`)
   ```go
   status, errMsg := validateFile(parser, file, &ruleCfg)
   ```
   - Attempts to parse each detected file
   - Returns status (`🟢 valid` or `❌ invalid`) and error message
   - Continues processing even if some files fail validation

5. **Print Results** (`printScannedFiles`)
   - Formats output as table with columns: RULE, PM, FORMAT, FILE, STATUS
   - Shows totals for entries, unique files, rules matched, valid/invalid files

## Core Functions

### `packages.DetectFiles`

**Location:** `pkg/packages/detect.go:15`

```go
func DetectFiles(cfg *config.Config, baseDir string) (map[string][]string, error)
```

**Behavior:**
1. Iterates over all rules in config
2. Skips disabled rules (`enabled: false`)
3. Walks directory tree with `filepath.Walk`
4. Matches each file against rule's `include` and `exclude` patterns
5. Resolves conflicts when multiple rules match the same file

**Rule Conflict Resolution:**

When multiple rules match the same file (e.g., `package.json` matching both `npm` and `pnpm`):

```go
func selectRuleForFile(cfg *config.Config, file string, rules []string) string
```

Priority order:
1. Rule with existing lock file in same directory
2. Priority map: `npm` > `pnpm` > `yarn`
3. Alphabetical order

### `utils.MatchPatterns`

**Location:** `pkg/utils/core.go`

```go
func MatchPatterns(path string, include, exclude []string) bool
```

**Behavior:**
1. If path matches ANY exclude pattern → return `false`
2. If path matches ANY include pattern → return `true`
3. Otherwise → return `false`

**Pattern Syntax:**
- `**/` - Match any directory depth
- `*` - Match any characters except `/`
- `?` - Match single character

## Output Format

### Table Output (Default)

```
Scanned package files in ./

RULE       PM      FORMAT  FILE             STATUS
----       --      ------  ----             ------
composer   php     json    composer.json    🟢 valid
mod        golang  raw     go.mod           🟢 valid
npm        js      json    package.json     ❌ invalid

Total entries: 3
Unique files: 3
Rules matched: 3
Valid files: 2
Invalid files: 1
```

### Status Indicators

The STATUS column shows file validation results using standardized emoji icons:

| Status | Icon | Description |
|--------|------|-------------|
| Valid | 🟢 | File parsed successfully |
| Invalid | ❌ | File has parsing errors |

### Structured Output

Use `-o json`, `-o csv`, or `-o xml` for structured output that includes error details:

```json
{
  "summary": {
    "directory": "./",
    "total_entries": 3,
    "unique_files": 3,
    "rules_matched": 3,
    "valid_files": 2,
    "invalid_files": 1
  },
  "files": [
    {
      "rule": "npm",
      "pm": "js",
      "format": "json",
      "file": "package.json",
      "status": "❌ invalid",
      "error": "invalid JSON: unexpected end of JSON input"
    }
  ]
}
```

## Special Handling

### File Filtering

Use the `--file` flag to filter results by file path patterns:

```bash
# Include only go.mod files
goupdate scan --file "go.mod"

# Include multiple patterns
goupdate scan --file "go.mod,package.json"

# Exclude patterns (prefix with !)
goupdate scan --file "!**/testdata/**,!**/examples/**"

# Combined include and exclude
goupdate scan --file "go.mod,!**/testdata/**"
```

**Pattern Behavior:**
- If include patterns exist, files must match at least one
- Files matching any exclude pattern are rejected
- Patterns support glob syntax (`*`, `**`, `?`)

### Disabled Rules

Rules with `enabled: false` are skipped during detection:

```yaml
rules:
  npm:
    enabled: false  # This rule will be skipped
    include: ["**/package.json"]
```

### Empty Include Patterns

Rules without `include` patterns emit a warning and are skipped:

```go
if len(rule.Include) == 0 {
    warnings.Warnf("⚠️ rule %s has no include patterns; skipping detection\n", ruleKey)
    continue
}
```

## Testing

**Test File:** `cmd/scan_test.go`

Key test functions:
- `TestRunScanEmptyDir` - No files found
- `TestRunScanWithFiles` - Normal detection
- `TestRunScanMultipleRules` - Multiple rules matching

**Mocking:**

```go
var detectFilesFunc = packages.DetectFiles

// In tests:
detectFilesFunc = func(cfg *config.Config, workDir string) (map[string][]string, error) {
    return map[string][]string{"npm": {"package.json"}}, nil
}
```

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| "failed to load config" | Invalid config file | Check YAML syntax |
| "no package manager rules configured" | Empty rules map | Add rules to config |
| "failed to access base directory" | Directory doesn't exist | Verify path |

## Related Documentation

- [configuration.md](./configuration.md) - Config loading details
- [list.md](./list.md) - Uses same detection logic
