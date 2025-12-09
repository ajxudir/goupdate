# Chaos Engineering Test Plan

This document provides a comprehensive feature inventory and chaos engineering test plan to systematically verify that the test suite catches breakages when features are deliberately disabled.

## Table of Contents

- [Feature Inventory](#feature-inventory)
- [Chaos Engineering Test Plan](#chaos-engineering-test-plan-1)
- [Phase 1: Core Utilities](#phase-1-core-utilities-low-risk)
- [Phase 2: Configuration](#phase-2-configuration-medium-risk)
- [Phase 3: File Parsing](#phase-3-file-parsing-medium-risk)
- [Phase 4: Package Detection](#phase-4-package-detection-medium-risk)
- [Phase 5: Lock File Resolution](#phase-5-lock-file-resolution-high-risk)
- [Phase 6: Outdated Logic](#phase-6-outdated-logic-high-risk)
- [Phase 7: Update Logic](#phase-7-update-logic-high-risk)
- [Phase 8: Preflight Validation](#phase-8-preflight-validation-medium-risk)
- [Phase 9: CLI Commands](#phase-9-cli-commands-high-risk)
- [Phase 10: Integration Tests](#phase-10-integration-tests-high-risk)
- [Execution Script Template](#execution-script-template)
- [Execution Results](#execution-results-2025-11-27)
- [Manual Battle Testing](#manual-battle-testing)
- [Notes](#notes)

---

## Feature Inventory

### 1. CLI Commands

| Command | Description | Test File |
|---------|-------------|-----------|
| `scan` | Scan directory for package files | `cmd/scan_test.go` |
| `list` (alias: `ls`) | List all detected packages | `cmd/list_test.go` |
| `outdated` | Check for outdated packages | `cmd/outdated_test.go` |
| `update` | Update packages to newer versions | `cmd/update_test.go` |
| `config` | Manage configuration | `cmd/config_test.go` |

### 2. CLI Flags by Command

#### Global/Common Flags
| Flag | Description | Commands |
|------|-------------|----------|
| `--directory`, `-d` | Working directory | all |
| `--config`, `-c` | Config file path | all |
| `--type`, `-t` | Filter by type (all/prod/dev) | list, outdated, update |
| `--package-manager`, `-p` | Filter by package manager | list, outdated, update |
| `--rule` | Filter by rule name | list, outdated, update |

#### Outdated/Update Flags
| Flag | Description | Commands |
|------|-------------|----------|
| `--major` | Allow major version upgrades | outdated, update |
| `--minor` | Allow minor version upgrades | outdated, update |
| `--patch` | Allow patch version upgrades only | outdated, update |
| `--no-timeout` | Disable command timeouts | outdated, update |
| `--skip-preflight` | Skip pre-flight command validation | outdated, update |

#### Update-specific Flags
| Flag | Description |
|------|-------------|
| `--dry-run` | Plan without making changes |
| `--skip-lock` | Skip lock file command execution |
| `--yes`, `-y` | Skip confirmation prompt |
| `--continue-on-fail` | Continue after failures |

#### Config Flags
| Flag | Description |
|------|-------------|
| `--show-defaults` | Show default configuration |
| `--show-effective` | Show effective (merged) configuration |
| `--init` | Create config template |

### 3. Package Managers Supported

| Manager | Rule Name | Lock File | Extraction |
|---------|-----------|-----------|------------|
| npm | npm | package-lock.json | command |
| pnpm | pnpm | pnpm-lock.yaml | command |
| yarn | yarn | yarn.lock | command |
| Go modules | mod | go.sum | raw |
| Composer | composer | composer.lock | json |
| pip | requirements | - | raw |
| Pipenv | pipenv | Pipfile.lock | json |
| MSBuild | msbuild | packages.lock.json | json |
| NuGet | nuget | packages.lock.json | json |

### 4. Configuration Options (model.go)

#### Top-Level Config
| Field | Type | Description |
|-------|------|-------------|
| `extends` | `[]string` | Extend from preset configs |
| `working_dir` | `string` | Base working directory |
| `rules` | `map[string]PackageManagerCfg` | Rule definitions |
| `exclude_versions` | `[]string` | Global version exclusions |
| `groups` | `map[string]GroupCfg` | Global package groups |
| `incremental` | `*IncrementalCfg` | Incremental update settings |
| `no_timeout` | `bool` | Runtime timeout disable |

#### PackageManagerCfg Fields
| Field | Type | Description |
|-------|------|-------------|
| `manager` | `string` | Manager identifier |
| `include` | `[]string` | Glob patterns to include |
| `exclude` | `[]string` | Glob patterns to exclude |
| `groups` | `map[string]GroupCfg` | Package groups |
| `format` | `string` | File format (json/yaml/xml/raw) |
| `fields` | `*FieldsCfg` | Field extraction config |
| `ignore` | `[]string` | Packages to ignore |
| `exclude_versions` | `[]string` | Version exclusions |
| `constraint_mapping` | `*ConstraintMappingCfg` | Version constraint mapping |
| `latest_mapping` | `*LatestMappingCfg` | Latest version indicator mapping |
| `package_overrides` | `map[string]*PackageOverride` | Per-package overrides |
| `extraction` | `*ExtractionCfg` | Regex extraction patterns |
| `outdated` | `*OutdatedCfg` | Outdated check config |
| `update` | `*UpdateCfg` | Update execution config |
| `lock_files` | `[]LockFileCfg` | Lock file definitions |
| `incremental` | `*IncrementalCfg` | Incremental settings |

#### OutdatedCfg Fields
| Field | Type | Description |
|-------|------|-------------|
| `commands` | `string` | Multi-line command script |
| `command` (legacy) | `string` | Single command |
| `args` (legacy) | `[]string` | Command arguments |
| `env` | `map[string]string` | Environment variables |
| `format` | `string` | Output format |
| `extraction` | `*ExtractionCfg` | Version extraction |
| `versioning` | `*VersioningCfg` | Versioning scheme |
| `exclude_versions` | `[]string` | Excluded versions |
| `exclude_version_patterns` | `[]string` | Excluded patterns |
| `timeout_seconds` | `int` | Command timeout |

#### UpdateCfg Fields
| Field | Type | Description |
|-------|------|-------------|
| `commands` | `string` | Multi-line lock command script |
| `lock_command` (legacy) | `string` | Single lock command |
| `lock_args` (legacy) | `[]string` | Lock command arguments |
| `env` | `map[string]string` | Environment variables |
| `group` | `string` | Default group name |
| `timeout_seconds` | `int` | Command timeout |

### 5. Core Packages/Modules

| Package | Purpose | Test File |
|---------|---------|-----------|
| `pkg/config` | Configuration loading/merging | `pkg/config/*_test.go` |
| `pkg/formats` | File format parsing | `pkg/formats/formats_test.go` |
| `pkg/packages` | Package detection | `pkg/packages/packages_test.go` |
| `pkg/lock` | Lock file resolution | `pkg/lock/lockfile_test.go`, `integration_test.go` |
| `pkg/outdated` | Version comparison | `pkg/outdated/outdated_test.go` |
| `pkg/update` | Package updating | `pkg/update/update_test.go` |
| `pkg/preflight` | Pre-flight validation | `pkg/preflight/preflight_test.go` |
| `pkg/cmdexec` | Command execution | `pkg/cmdexec/cmdexec_test.go` |
| `pkg/warnings` | Warning collection | `pkg/warnings/core_test.go` |
| `pkg/utils` | Utilities (version, display) | `pkg/utils/*_test.go` |

### 6. Installation Status Types

| Status | Description |
|--------|-------------|
| `LockFound` | Package found in lock file |
| `SelfPinned` | Manifest is its own lock (e.g., requirements.txt) |
| `LockMissing` | Lock file doesn't exist |
| `NotInLock` | Package not in lock file |
| `VersionMissing` | Version is wildcard with no lock |
| `NotConfigured` | No lock file config for rule |
| `Floating` | Floating constraint (cannot auto-update) |

### 7. Version Constraints

| Constraint | Description |
|------------|-------------|
| `^` | Compatible (major locked) |
| `~` | Approximate (minor locked) |
| `>=` | Greater than or equal |
| `<=` | Less than or equal |
| `>` | Greater than |
| `<` | Less than |
| `=` | Exact match |
| `*` | Any version |

---

## Chaos Engineering Test Plan

### Methodology

For each feature:
1. **Identify**: Document what the feature does
2. **Break**: Comment out or modify the feature code
3. **Test**: Run `go test ./...`
4. **Verify**: Check if tests catch the breakage
5. **Fix**: If tests pass (bad), add new tests
6. **Restore**: Revert the deliberate breakage

### Test Execution Order

Execute tests in order of complexity (simple to complex):

---

## Phase 1: Core Utilities (Low Risk)

### Test 1.1: Version Parsing Breakage
**File**: `pkg/utils/version.go`
**Function**: `ParseSemver` or similar version parsing
**Break Method**: Return empty version always
**Expected**: `pkg/utils/version_test.go` should fail

### Test 1.2: Display Utilities
**File**: `pkg/utils/display.go`
**Function**: `FormatTable` or display helpers
**Break Method**: Return empty string
**Expected**: Display tests should fail

### Test 1.3: Warning Collection
**File**: `pkg/warnings/core.go`
**Function**: Warning collection and dedup
**Break Method**: Don't collect warnings
**Expected**: `pkg/warnings/core_test.go` should fail

---

## Phase 2: Configuration (Medium Risk)

### Test 2.1: Config Loading
**File**: `pkg/config/load.go`
**Function**: `LoadConfig`
**Break Method**: Return nil config always
**Expected**: `pkg/config/load_test.go` and cmd tests should fail

### Test 2.2: Config Merging
**File**: `pkg/config/merge.go`
**Function**: Rule merging logic
**Break Method**: Don't merge rules
**Expected**: `pkg/config/merge_test.go` should fail

### Test 2.3: Default Config
**File**: `pkg/config/defaults.go`
**Function**: `GetDefaultConfig`
**Break Method**: Return empty defaults
**Expected**: Most tests relying on defaults should fail

### Test 2.4: Group Assignment
**File**: `pkg/config/groups.go`
**Function**: `AssignPackageGroup`
**Break Method**: Never assign groups
**Expected**: `pkg/config/groups_test.go` should fail

### Test 2.5: Latest Mapping
**File**: `pkg/config/latest_mapping.go`
**Function**: `IsLatestIndicator`
**Break Method**: Always return false
**Expected**: `pkg/config/latest_mapping_test.go` should fail

### Test 2.6: Incremental Config
**File**: `pkg/config/incremental.go`
**Function**: Incremental mode logic
**Break Method**: Disable incremental support
**Expected**: `pkg/config/incremental_test.go` should fail

---

## Phase 3: File Parsing (Medium Risk)

### Test 3.1: JSON Parser
**File**: `pkg/formats/json.go`
**Function**: `ParseJSON`
**Break Method**: Return empty packages
**Expected**: `pkg/formats/formats_test.go` should fail

### Test 3.2: YAML Parser
**File**: `pkg/formats/yaml.go`
**Function**: `ParseYAML`
**Break Method**: Return empty packages
**Expected**: Format tests should fail

### Test 3.3: XML Parser
**File**: `pkg/formats/xml.go`
**Function**: `ParseXML`
**Break Method**: Return empty packages
**Expected**: XML parsing tests should fail

### Test 3.4: Raw Parser
**File**: `pkg/formats/raw.go`
**Function**: `ParseRaw`
**Break Method**: Return empty packages
**Expected**: Raw format tests should fail

### Test 3.5: PNPM Lock Parser
**File**: `pkg/formats/pnpm.go`
**Function**: `ExtractPnpmLockVersions`
**Break Method**: Return empty map
**Expected**: Integration tests should fail

### Test 3.6: Yarn Lock Parser
**File**: `pkg/formats/yarn.go`
**Function**: `ExtractYarnLockVersions`
**Break Method**: Return empty map
**Expected**: Integration tests should fail

---

## Phase 4: Package Detection (Medium Risk)

### Test 4.1: File Detection
**File**: `pkg/packages/detect.go`
**Function**: `DetectFiles`
**Break Method**: Return no files
**Expected**: `pkg/packages/packages_test.go` should fail

### Test 4.2: Dynamic Parser
**File**: `pkg/packages/parser.go`
**Function**: `ParseFile`
**Break Method**: Return empty result
**Expected**: Parser tests should fail

---

## Phase 5: Lock File Resolution (High Risk)

### Test 5.1: Apply Installed Versions
**File**: `pkg/lock/resolve.go`
**Function**: `ApplyInstalledVersions`
**Break Method**: Don't enrich packages
**Expected**: `pkg/lock/lockfile_test.go` and integration tests should fail

### Test 5.2: Lock Status Assignment
**File**: `pkg/lock/status.go`
**Function**: Status constants
**Break Method**: Change status values
**Expected**: Tests checking status should fail

### Test 5.3: Version Extraction from Lock
**File**: `pkg/lock/resolve.go`
**Function**: `extractVersionsFromLock`
**Break Method**: Return empty map
**Expected**: Integration tests should fail

---

## Phase 6: Outdated Logic (High Risk)

### Test 6.1: List Newer Versions
**File**: `pkg/outdated/core.go`
**Function**: `ListNewerVersions`
**Break Method**: Return empty slice
**Expected**: `pkg/outdated/outdated_test.go` should fail

### Test 6.2: Version Filtering
**File**: `pkg/outdated/core.go`
**Function**: `FilterNewerVersions`
**Break Method**: Return all versions (no filtering)
**Expected**: Outdated tests should fail

### Test 6.3: Version Summarization
**File**: `pkg/outdated/core.go`
**Function**: `SummarizeAvailableVersions`
**Break Method**: Return #N/A always
**Expected**: Tests should fail

### Test 6.4: Constraint Filtering
**File**: `pkg/outdated/core.go`
**Function**: `FilterVersionsByConstraint`
**Break Method**: Ignore constraints
**Expected**: Constraint tests should fail

### Test 6.5: Version Exclusions
**File**: `pkg/outdated/core.go`
**Function**: `applyVersionExclusions`
**Break Method**: Don't exclude versions
**Expected**: Exclusion tests should fail

### Test 6.6: Versioning Strategy
**File**: `pkg/outdated/versioning.go`
**Function**: `newVersioningStrategy`
**Break Method**: Return nil strategy
**Expected**: Versioning tests should fail

### Test 6.7: Command Execution
**File**: `pkg/outdated/exec.go`
**Function**: `execOutdated`
**Break Method**: Return error always
**Expected**: Command execution tests should fail

---

## Phase 7: Update Logic (High Risk)

### Test 7.1: Update Package
**File**: `pkg/update/core.go`
**Function**: `UpdatePackage`
**Break Method**: Don't update
**Expected**: `pkg/update/update_test.go` should fail

### Test 7.2: JSON Version Update
**File**: `pkg/update/json.go`
**Function**: `updateJSONVersion`
**Break Method**: Don't modify content
**Expected**: JSON update tests should fail

### Test 7.3: YAML Version Update
**File**: `pkg/update/yaml.go`
**Function**: `updateYAMLVersion`
**Break Method**: Don't modify content
**Expected**: YAML update tests should fail

### Test 7.4: XML Version Update
**File**: `pkg/update/xml.go`
**Function**: `updateXMLVersion`
**Break Method**: Don't modify content
**Expected**: XML update tests should fail

### Test 7.5: Raw Version Update
**File**: `pkg/update/raw.go`
**Function**: `updateRawVersion`
**Break Method**: Don't modify content
**Expected**: Raw update tests should fail

### Test 7.6: Rollback Logic
**File**: `pkg/update/rollback.go`
**Function**: Rollback implementation
**Break Method**: Don't restore original
**Expected**: Rollback tests should fail

### Test 7.7: Group Scope Detection
**File**: `pkg/update/core.go`
**Function**: `IsGroupScope`
**Break Method**: Return false always
**Expected**: Group scope tests should fail

### Test 7.8: Lock Command Execution
**File**: `pkg/update/exec.go`
**Function**: `execCommand`
**Break Method**: Return error always
**Expected**: Lock command tests should fail

---

## Phase 8: Preflight Validation (Medium Risk)

### Test 8.1: Command Validation
**File**: `pkg/preflight/preflight.go`
**Function**: `ValidatePackages`
**Break Method**: Always return success
**Expected**: `pkg/preflight/preflight_test.go` should fail

### Test 8.2: Resolution Hints
**File**: `pkg/preflight/preflight.go`
**Function**: `GetResolutionHint`
**Break Method**: Return empty hints
**Expected**: Hint tests should fail

---

## Phase 9: CLI Commands (High Risk)

### Test 9.1: List Command Type Filter
**File**: `cmd/list.go`
**Function**: `filterPackagesWithFilters`
**Break Method**: Ignore type filter
**Expected**: `cmd/list_test.go` should fail

### Test 9.2: List Command PM Filter
**File**: `cmd/list.go`
**Function**: `filterPackagesWithFilters`
**Break Method**: Ignore PM filter
**Expected**: Filter tests should fail

### Test 9.3: Outdated Major/Minor/Patch Selection
**File**: `cmd/outdated.go`
**Function**: Target version selection
**Break Method**: Ignore flags
**Expected**: `cmd/outdated_test.go` should fail

### Test 9.4: Update Dry Run
**File**: `cmd/update.go`
**Function**: `runUpdate`
**Break Method**: Always write files
**Expected**: `cmd/update_test.go` should fail

### Test 9.5: Update Skip Lock
**File**: `cmd/update.go`
**Function**: Skip lock logic
**Break Method**: Always run lock
**Expected**: Skip lock tests should fail

### Test 9.6: Config Show Defaults
**File**: `cmd/config.go`
**Function**: `runConfig`
**Break Method**: Don't show defaults
**Expected**: `cmd/config_test.go` should fail

### Test 9.7: Config Init
**File**: `cmd/config.go`
**Function**: `createConfigTemplate`
**Break Method**: Don't create file
**Expected**: Init tests should fail

---

## Phase 10: Integration Tests (High Risk)

### Test 10.1: NPM Integration
**File**: `pkg/lock/integration_test.go`
**Function**: `TestIntegration_NPM`
**Break Method**: Corrupt npm testdata
**Expected**: Integration test should fail

### Test 10.2: Go Mod Integration
**File**: `pkg/lock/integration_test.go`
**Function**: `TestIntegration_GoMod`
**Break Method**: Corrupt mod testdata
**Expected**: Integration test should fail

### Test 10.3: Composer Integration
**File**: `pkg/lock/integration_test.go`
**Function**: `TestIntegration_Composer`
**Break Method**: Corrupt composer testdata
**Expected**: Integration test should fail

### Test 10.4: Lock Not Found
**File**: `pkg/lock/integration_test.go`
**Function**: `TestIntegration_LockNotFound`
**Break Method**: Create fake lock file
**Expected**: Integration test should fail

### Test 10.5: Lock Missing Package
**File**: `pkg/lock/integration_test.go`
**Function**: `TestIntegration_LockMissing`
**Break Method**: Add package to lock
**Expected**: Integration test should fail

---

## Execution Script Template

```bash
#!/bin/bash
# Chaos test execution helper

FEATURE="$1"
FILE="$2"
BACKUP="${FILE}.backup"

echo "=== Chaos Test: $FEATURE ==="

# Backup
cp "$FILE" "$BACKUP"

# Break (modify this per test)
# sed -i 's/return result/return nil/' "$FILE"

# Test
go test ./... 2>&1 | tee /tmp/chaos_test.log

# Check result
if grep -q "FAIL" /tmp/chaos_test.log; then
    echo "✅ PASS: Tests caught the breakage"
else
    echo "❌ FAIL: Tests did NOT catch the breakage"
    echo "Action: Add new tests for this feature"
fi

# Restore
mv "$BACKUP" "$FILE"
```

---

## Execution Results (2025-11-27)

| Test ID | Feature | File | Status | Tests Caught? | Action Taken |
|---------|---------|------|--------|---------------|---------------|
| 1.1 | IsLatestIndicator | pkg/utils/version.go | ✅ Done | YES (2 tests) | None needed |
| 2.3 | loadDefaultConfig | pkg/config/defaults.go | ✅ Done | YES (many) | None needed |
| 2.4 | validateGroupMembership | pkg/config/groups.go | ✅ Done | YES (2 tests) | None needed |
| 3.1 | JSONParser.Parse | pkg/formats/json.go | ✅ Done | YES (10+ tests) | None needed |
| 5.1 | ApplyInstalledVersions | pkg/lock/resolve.go | ✅ Done | YES (15+ tests) | None needed |
| 6.3 | SummarizeAvailableVersions | pkg/outdated/core.go | ✅ Done | YES (3 tests) | None needed |
| 6.4 | FilterVersionsByConstraint | pkg/outdated/core.go | ✅ Done | YES (3 tests) | None needed |
| 7.2 | updateJSONVersion | pkg/update/json.go | ✅ Done | YES (5 tests) | None needed |
| 8.1 | ValidatePackages | pkg/preflight/preflight.go | ✅ Done | **NO → Fixed** | Added 2 new tests |
| 9.1 | matchesTypeFilter | cmd/list.go | ✅ Done | YES (1 test) | None needed |

### Summary
- **10 chaos tests executed**
- **9 tests caught breakages** (90% coverage)
- **1 gap identified and fixed** (ValidatePackages)
- **2 new tests added** to fix gap

### Test Coverage by Package
- **pkg/config**: Group validation ✅
- **pkg/formats**: JSON parser ✅
- **pkg/lock**: Installed versions ✅
- **pkg/outdated**: Version summarization, constraint filtering ✅
- **pkg/update**: JSON version update ✅
- **pkg/preflight**: Command validation ✅ (after fix)
- **pkg/utils**: Latest indicator ✅
- **cmd**: Type filtering ✅

### Gap Fixed: Preflight Validation
The original test `TestValidatePackages` only checked that the function didn't panic, but didn't verify it actually detects missing commands. Added:
- `TestValidatePackagesDetectsMissingCommands`: Verifies `ValidatePackages` detects missing commands in config
- `TestValidateRulesDetectsMissingCommands`: Verifies `ValidateRules` detects missing commands

---

---

## Manual Battle Testing

This section documents manual testing procedures for real-world validation. These tests verify that the CLI works correctly with actual package ecosystems and real file changes.

### Prerequisites

- Clean git working directory
- Access to real projects with package manifests (npm, Go, Python, etc.)
- Network access for registry queries

### Test Matrix: Commands × Flags

| Command | Required Flags to Test | Expected Behavior |
|---------|------------------------|-------------------|
| `scan` | `-d`, `-c`, `--rule` | Lists detected manifest files |
| `list` | `-d`, `-t`, `-p`, `--rule` | Shows all packages with filters |
| `outdated` | `-d`, `--major/minor/patch`, `--no-timeout`, `--skip-preflight` | Shows version updates available |
| `update` | `-d`, `--dry-run`, `--skip-lock`, `-y`, `--continue-on-fail` | Modifies manifest files |
| `config` | `--show-defaults`, `--show-effective`, `--init` | Configuration management |

### Battle Test 1: NPM Project

```bash
# Create test environment
git clone https://github.com/vercel/next.js /tmp/test-npm
cd /tmp/test-npm

# Non-destructive tests
goupdate scan
goupdate list --type prod
goupdate list --type dev
goupdate outdated --major
goupdate outdated --minor
goupdate outdated --patch
goupdate update --dry-run

# DESTRUCTIVE TEST (creates actual changes)
git checkout -b test-update
goupdate update --patch --skip-lock -y  # Patch-only for safety
git diff                                 # Verify changes
git checkout main                        # Revert
git branch -D test-update
```

### Battle Test 2: Go Project

```bash
# Create test environment
git clone https://github.com/spf13/cobra /tmp/test-go
cd /tmp/test-go

# Non-destructive tests
goupdate scan
goupdate list
goupdate outdated
goupdate update --dry-run

# DESTRUCTIVE TEST
git checkout -b test-update
goupdate update --minor --skip-lock -y
git diff go.mod                          # Verify version changes
git checkout main
git branch -D test-update
```

### Battle Test 3: Python Project

```bash
# Create test environment
git clone https://github.com/psf/requests /tmp/test-python
cd /tmp/test-python

# Create config for requirements files
cat > .goupdate.yml << 'EOF'
extends: [default]
rules:
  requirements:
    manager: python
    include: ["**/requirements*.txt"]
    format: raw
    extraction:
      pattern: '^(?P<n>[\w\-\.]+)(?:[ \t]*(?P<constraint>[><=~!]+)[ \t]*(?P<version>[\w\.\-\+]+))?'
EOF

# Non-destructive tests
goupdate scan
goupdate list
goupdate update --dry-run

# DESTRUCTIVE TEST
git checkout -b test-update
goupdate update --patch --skip-lock -y
git diff
git checkout main
git branch -D test-update
```

### Battle Test 4: Custom Config Options

```bash
# Test extends
cat > /tmp/test-extends/.goupdate.yml << 'EOF'
extends: [default]
rules:
  npm:
    groups:
      core: [react, react-dom]
    exclude_versions:
      - "(?i)beta"
    ignore:
      - "@types/*"
EOF
goupdate outdated -d /tmp/test-extends

# Test incremental
cat > /tmp/test-incremental/.goupdate.yml << 'EOF'
extends: [default]
rules:
  npm:
    incremental:
      - react
      - typescript
EOF
goupdate outdated -d /tmp/test-incremental

# Test package overrides
cat > /tmp/test-overrides/.goupdate.yml << 'EOF'
extends: [default]
rules:
  npm:
    package_overrides:
      lodash:
        ignore: true
      react:
        constraint: "~"
EOF
goupdate list -d /tmp/test-overrides
```

### Destructive Command Checklist

**IMPORTANT**: These tests modify actual files. Always use a test branch.

| Test | Command | Verification |
|------|---------|--------------|
| Patch update | `goupdate update --patch -y` | `git diff` shows only patch version bumps |
| Minor update | `goupdate update --minor -y` | `git diff` shows minor version bumps |
| Major update | `goupdate update --major -y` | `git diff` shows major version bumps |
| Skip lock | `goupdate update --skip-lock -y` | No lock command executed |
| Continue on fail | `goupdate update --continue-on-fail -y` | Continues after first failure |
| Single package | `goupdate update react -y` | Only specified package updated |

### Post-Destructive Test Validation

After running destructive tests, verify:

1. **File integrity**: Changed files are valid JSON/YAML/XML
2. **Version format**: New versions maintain constraint format (^, ~, etc.)
3. **No corruption**: No partial writes or truncated content
4. **Rollback works**: `git checkout -- .` restores original state

### Battle Test Results Template

| Date | Project | Commands Tested | Destructive? | Result | Notes |
|------|---------|-----------------|--------------|--------|-------|
| YYYY-MM-DD | project-name | list, outdated | No | ✅ Pass | |
| YYYY-MM-DD | project-name | update --patch | Yes | ✅ Pass | Verified git diff |

---

## Notes

1. Always run `go build ./...` after modifications to catch compile errors
2. Some breakages may cause panic rather than test failures - this is acceptable
3. If a feature has no test coverage, document and add tests
4. Run tests in isolation to avoid interference
5. Use git stash to preserve working state between tests
6. **For destructive tests**: Always create a test branch and never push changes to main
