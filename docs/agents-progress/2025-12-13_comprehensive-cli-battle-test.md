# Task: Comprehensive CLI Battle Test

**Agent:** Claude
**Date:** 2025-12-13
**Branch:** claude/organize-mock-data-01Y1vCHWXSvHwCvA6nU99JcH
**Status:** In Progress (Phase 8/8)

## Objective

Battle test all CLI commands, parameters, config fields in various combinations to ensure the tool works correctly for all officially supported package managers before release. Validate all output against documentation.

## Test Coverage Matrix

### Commands to Test
- [ ] `scan` - file detection
- [ ] `list` - package parsing + lock resolution
- [ ] `outdated` - version checking
- [ ] `update` - applying updates
- [ ] `config` - configuration management
- [ ] `version` - version info
- [ ] `help` - help system

### Supported Rules (Package Managers)
- [ ] `npm` (js) - package.json / package-lock.json
- [ ] `pnpm` (js) - package.json / pnpm-lock.yaml
- [ ] `yarn` (js) - package.json / yarn.lock
- [ ] `composer` (php) - composer.json / composer.lock
- [ ] `requirements` (python) - requirements*.txt (self-pinning)
- [ ] `pipfile` (python) - Pipfile / Pipfile.lock
- [ ] `mod` (golang) - go.mod / go.sum
- [ ] `msbuild` (dotnet) - *.csproj / packages.lock.json
- [ ] `nuget` (dotnet) - packages.config / packages.lock.json

### Output Formats
- [ ] table (default)
- [ ] json (`--output json`)
- [ ] csv (`--output csv`)
- [ ] xml (`--output xml`)

### Flags to Test
Global: `--config`, `--directory`, `--verbose`, `--help`
list: `--type`, `--package-manager`, `--rule`, `--name`, `--group`, `--output`, `--file`
outdated: `--type`, `--package-manager`, `--rule`, `--name`, `--group`, `--major`, `--minor`, `--patch`, `--no-timeout`, `--skip-preflight`, `--continue-on-fail`, `--output`
update: `--type`, `--package-manager`, `--rule`, `--name`, `--group`, `--major`, `--minor`, `--patch`, `--incremental`, `--dry-run`, `--skip-lock`, `--yes`, `--no-timeout`, `--continue-on-fail`, `--skip-preflight`, `--output`
scan: `--file`, `--directory`, `--config`, `--output`
config: `--show-defaults`, `--show-effective`, `--init`, `--validate`

### Status Values to Verify (per docs/user/cli.md)
List statuses:
- [ ] `LockFound` (🟢)
- [ ] `SelfPinned` (📌)
- [ ] `LockMissing` (🟠)
- [ ] `NotInLock` (🔵)
- [ ] `VersionMissing` (⛔)
- [ ] `NotConfigured` (⚪)
- [ ] `Floating` (⛔)
- [ ] `Ignored` (new)

Outdated statuses:
- [ ] `UpToDate` (🟢)
- [ ] `Outdated` (🟠)
- [ ] `NotConfigured` (⚪)
- [ ] `Failed` (❌)

Update statuses:
- [ ] `UpToDate` (🟢)
- [ ] `Planned` (🟡)
- [ ] `Updated` (🟢)
- [ ] `Failed` (❌)
- [ ] `NotConfigured` (⚪)

## Progress

### Phase 1: Setup
- [ ] Build binary
- [ ] Clone real-world projects
- [ ] Create test environment

### Phase 2: Basic Command Tests
- [ ] Test `version` command
- [ ] Test `help` command
- [ ] Test `config` command variants

### Phase 3: scan Command Tests
- [ ] Test with each package manager testdata
- [ ] Test with examples
- [ ] Test output formats
- [ ] Test --file filter
- [ ] Test invalid directory

### Phase 4: list Command Tests
- [ ] Test with each package manager
- [ ] Test all status values
- [ ] Test filters (--type, --rule, --name, --group)
- [ ] Test output formats
- [ ] Validate column values against docs

### Phase 5: outdated Command Tests
- [ ] Test with npm (real project)
- [ ] Test with pnpm (real project)
- [ ] Test with yarn (real project)
- [ ] Test with composer (real project)
- [ ] Test with mod (real project)
- [ ] Test --major, --minor, --patch flags
- [ ] Test output formats
- [ ] Test --continue-on-fail

### Phase 6: update Command Tests
- [ ] Test --dry-run
- [ ] Test actual updates with --yes
- [ ] Test --patch updates
- [ ] Test --skip-lock
- [ ] Test rollback on failure
- [ ] Validate output values after update

### Phase 7: Edge Cases & Error Handling
- [ ] Invalid config file
- [ ] Missing lock file
- [ ] Corrupted lock file
- [ ] Non-existent directory
- [ ] Empty manifest
- [ ] Network timeout simulation

### Phase 8: Real-World Projects
- [ ] Test expressjs/express (npm)
- [ ] Test laravel/laravel (composer)
- [ ] Test spf13/cobra (go mod)

## Files Modified

(will be updated as issues are found and fixed)

## Issues Found

### Bug 1: Ignored status missing icon (FIXED)
- **Location**: `pkg/display/status.go`, `pkg/constants/statuses.go`
- **Description**: The "Ignored" status was displayed without an icon in table output
- **Fix**: Added `IconIgnored = "🚫"` constant and added case for `InstallStatusIgnored` in `FormatInstallStatus` and `statusIconMap`
- **Files modified**: `pkg/display/status.go`, `pkg/constants/statuses.go`

### Bug 2: Go mod parser doesn't support single-line require (FIXED)
- **Location**: `pkg/config/default.yml` (mod rule extraction pattern)
- **Description**: Go modules with single dependency use single-line format `require pkg version` instead of block format. The regex pattern only matched block format.
- **Fix**: Updated pattern to match both formats: `'(?m)^(?:\s+|require\s+)(?P<n>[\w\.\-\/]+)\s+(?P<version>v[\w\.\-\+]+)'`
- **Files modified**: `pkg/config/default.yml`

### Bug 3: Testdata config validation error (FIXED)
- **Location**: `pkg/testdata/ignored_packages/goupdate.yaml`
- **Description**: Config used `commands` for lock_files but validation required `format` or `extraction`
- **Fix**: Changed to file-based extraction with proper format and pattern
- **Files modified**: `pkg/testdata/ignored_packages/goupdate.yaml`

### Documentation: Added Ignored status to CLI docs (NEW)
- **Location**: `docs/user/cli.md`
- **Description**: Added documentation for the new "Ignored" status with icon 🚫
- **Files modified**: `docs/user/cli.md`

### Observation: Invalid output format falls back silently
- **Description**: When `--output invalid_format` is passed, it silently falls back to table format
- **Severity**: Minor (works correctly, just no warning)
- **Recommendation**: Consider adding a warning for unknown output formats

## Notes

Testing will use:
- Internal testdata from `pkg/testdata/`
- Examples from `examples/`
- Real cloned projects in `/tmp/`

All output values will be validated against:
- docs/user/cli.md (status values, column names)
- docs/user/configuration.md (config fields)
- docs/user/features.md (capabilities)
