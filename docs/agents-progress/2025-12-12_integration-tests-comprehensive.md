# Task: Comprehensive Integration Tests for goupdate

**Agent:** Claude
**Date:** 2025-12-12
**Status:** Planning
**Branch:** claude/review-config-tests-LwFiQ

## Objective

Add comprehensive integration tests for all officially supported package managers (9 total) and all goupdate commands (scan, list, outdated, update, config) to catch configuration and integration bugs before release. This includes:

1. Real integration tests using actual package manifest files
2. Mock data for registry responses and command output
3. Battle testing on real-world projects
4. Chaos engineering to validate test coverage
5. Documentation and infrastructure updates

## Current State

### Already Completed (from merged branch)
- ✅ `pkg/config/commands_test.go` - Configuration validation tests
- ✅ `pkg/lock/integration_test.go` - NPM, PNPM, GoMod, Composer integration tests
- ✅ `pkg/testdata/pnpm/` - PNPM testdata added

### Coverage Status
| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| pkg/config | 100% | 80% | ✅ |
| pkg/formats | 100% | 85% | ✅ |
| pkg/lock | 93.4% | 80% | ✅ |
| pkg/outdated | 76.6% | 80% | ⚠️ Below target |
| pkg/update | 86.7% | 80% | ✅ |
| cmd | 77.6% | 70% | ✅ |

### Package Manager Support Matrix
| Manager | Testdata | Integration Test | Missing |
|---------|----------|------------------|---------|
| npm | ✅ | ✅ | - |
| pnpm | ✅ | ✅ | - |
| yarn | ⚠️ edge-cases only | ❌ | Full testdata + integration test |
| composer | ✅ | ✅ | - |
| mod | ✅ | ✅ | - |
| requirements | ✅ | ❌ | Integration test |
| pipfile | ✅ | ❌ | Integration test |
| msbuild | ✅ | ❌ | Integration test |
| nuget | ✅ | ❌ | Integration test |

---

## PHASE 1: IMPLEMENTATION (Get it working)

Following AGENTS.md Section 0: Complete the task first, then add tests/coverage.

### 1.1: Foundation - Directory Structure

Create new directory structure:
```
pkg/
├── testdata/                    # Real package files (already exists)
│   ├── yarn/                    # NEW - proper structure
│   ├── npm/                     # ✅ Exists
│   ├── pnpm/                    # ✅ Exists
│   ├── composer/                # ✅ Exists
│   ├── mod/                     # ✅ Exists
│   ├── requirements/            # ✅ Exists
│   ├── pipfile/                 # ✅ Exists
│   ├── msbuild/                 # ✅ Exists
│   └── nuget/                   # ✅ Exists
├── testdata_errors/             # Real files that trigger errors (exists)
│   └── _version-mismatch/       # NEW - drift detection scenarios
├── mocksdata/                   # NEW: Mock responses (registry, commands)
│   ├── README.md                # Documentation for mocksdata
│   ├── registry/                # Mock registry responses
│   │   ├── npm/                 # npm view output samples
│   │   ├── composer/            # composer show output samples
│   │   ├── gomod/               # go list -m -versions output
│   │   ├── pypi/                # pip index versions output
│   │   └── nuget/               # dotnet package search output
│   └── commands/                # Mock command output samples
│       ├── pnpm-ls/             # pnpm ls --json output
│       ├── npm-ls/              # npm ls --json output
│       ├── yarn-list/           # yarn list --json output
│       └── composer-show/       # composer show output
└── mocksdata_errors/            # NEW: Mock responses that trigger errors
    ├── README.md                # Documentation for error cases
    ├── registry/                # Invalid/error registry responses
    │   ├── timeout/             # Timeout scenarios
    │   ├── not-found/           # 404 responses
    │   └── malformed/           # Invalid JSON
    └── commands/                # Invalid command outputs
        ├── invalid-json/
        └── partial-output/
```

**Tasks:**
- [ ] 1.1.1: Create `pkg/mocksdata/` directory structure
- [ ] 1.1.2: Create `pkg/mocksdata_errors/` directory structure
- [ ] 1.1.3: Add `pkg/mocksdata/README.md` documentation
- [ ] 1.1.4: Add `pkg/mocksdata_errors/README.md` documentation

### 1.2: Testdata Completion

**1.2.1: Complete yarn testdata**
```
pkg/testdata/yarn/
├── package.json          # Real dependencies (lodash, express, typescript)
├── yarn.lock             # Real yarn.lock v1 format
└── .goupdate.yml         # Override for offline testing
```

**1.2.2: Add .goupdate.yml overrides for offline testing**
Verify each testdata directory has offline-compatible config:
- [ ] `pkg/testdata/npm/.goupdate.yml`
- [ ] `pkg/testdata/pnpm/.goupdate.yml`
- [ ] `pkg/testdata/yarn/.goupdate.yml` (NEW)
- [ ] `pkg/testdata/composer/.goupdate.yml`
- [ ] `pkg/testdata/mod/.goupdate.yml`
- [ ] `pkg/testdata/requirements/.goupdate.yml`
- [ ] `pkg/testdata/pipfile/.goupdate.yml`
- [ ] `pkg/testdata/msbuild/.goupdate.yml`
- [ ] `pkg/testdata/nuget/.goupdate.yml`

**1.2.3: Add version mismatch testdata for drift detection**
```
pkg/testdata_errors/_version-mismatch/
├── npm/
│   ├── package.json              # lodash: "^4.17.0"
│   ├── package-lock.json         # lodash: "4.17.21"
│   └── .goupdate.yml
└── composer/
    ├── composer.json             # monolog: "^2.0"
    ├── composer.lock             # monolog: "2.3.5"
    └── .goupdate.yml
```

### 1.3: Mocksdata - Registry Responses

Capture real registry output for testing outdated command:

**1.3.1: NPM registry responses**
```bash
# Capture real output
npm view lodash versions --json > pkg/mocksdata/registry/npm/lodash.json
npm view express versions --json > pkg/mocksdata/registry/npm/express.json
```

Files to create:
- [ ] `pkg/mocksdata/registry/npm/lodash.json`
- [ ] `pkg/mocksdata/registry/npm/express.json`

**1.3.2: Composer registry responses**
```bash
composer show monolog/monolog --all --format=json > pkg/mocksdata/registry/composer/monolog.json
```

Files to create:
- [ ] `pkg/mocksdata/registry/composer/monolog.json`

**1.3.3: Go module registry responses**
```bash
go list -m -versions github.com/spf13/cobra > pkg/mocksdata/registry/gomod/cobra.txt
```

Files to create:
- [ ] `pkg/mocksdata/registry/gomod/cobra.txt`

**1.3.4: PyPI registry responses**
```bash
curl https://pypi.org/pypi/requests/json > pkg/mocksdata/registry/pypi/requests.json
```

Files to create:
- [ ] `pkg/mocksdata/registry/pypi/requests.json`

**1.3.5: NuGet registry responses**
```bash
curl "https://api.nuget.org/v3-flatcontainer/newtonsoft.json/index.json" > pkg/mocksdata/registry/nuget/newtonsoft.json
```

Files to create:
- [ ] `pkg/mocksdata/registry/nuget/newtonsoft.json`

### 1.4: Mocksdata - Command Output

Capture real command output for testing lock resolution:

**1.4.1: pnpm ls output**
```bash
cd pkg/testdata/pnpm && pnpm install && pnpm ls --json --depth=0 > ../../mocksdata/commands/pnpm-ls/standard.json
```

Files to create:
- [ ] `pkg/mocksdata/commands/pnpm-ls/standard.json`

**1.4.2: npm ls output**
```bash
cd pkg/testdata/npm && npm install && npm ls --json > ../../mocksdata/commands/npm-ls/standard.json
```

Files to create:
- [ ] `pkg/mocksdata/commands/npm-ls/standard.json`

**1.4.3: yarn list output**
```bash
cd pkg/testdata/yarn && yarn install && yarn list --json > ../../mocksdata/commands/yarn-list/standard.json
```

Files to create:
- [ ] `pkg/mocksdata/commands/yarn-list/standard.json`

**1.4.4: composer show output**
```bash
cd pkg/testdata/composer && composer install && composer show --format=json > ../../mocksdata/commands/composer-show/standard.json
```

Files to create:
- [ ] `pkg/mocksdata/commands/composer-show/standard.json`

### 1.5: Mocksdata Errors

Create error scenarios for testing error handling:

**1.5.1: Registry errors**
- [ ] `pkg/mocksdata_errors/registry/not-found/package.json` - 404 response
- [ ] `pkg/mocksdata_errors/registry/malformed/response.json` - Truncated JSON
- [ ] `pkg/mocksdata_errors/registry/timeout/README.md` - Document timeout testing

**1.5.2: Command output errors**
- [ ] `pkg/mocksdata_errors/commands/invalid-json/pnpm-ls.json` - Invalid JSON
- [ ] `pkg/mocksdata_errors/commands/partial-output/npm-ls.json` - Incomplete output

---

## PHASE 2: TESTING (After Phase 1 complete)

Following AGENTS.md Section 0: Write tests after implementation is complete.

### 2.1: Lock Resolution Tests

**File:** `pkg/lock/integration_test.go` (+150 lines)

Add integration tests for remaining package managers:

- [ ] `TestIntegration_Yarn` - Parse yarn.lock
- [ ] `TestIntegration_Requirements` - Self-pinning manifest
- [ ] `TestIntegration_Pipfile` - Parse Pipfile.lock
- [ ] `TestIntegration_MSBuild` - Parse packages.lock.json
- [ ] `TestIntegration_NuGet` - Parse packages.lock.json

**File:** `pkg/lock/command_output_test.go` (NEW ~200 lines)

Test JSON parsing of command output using mocksdata:

- [ ] `TestParseLockCommand_PNPM_Output` - Parse pnpm ls --json
- [ ] `TestParseLockCommand_NPM_Output` - Parse npm ls --json
- [ ] `TestParseLockCommand_Yarn_Output` - Parse yarn list --json
- [ ] `TestParseLockCommand_Composer_Output` - Parse composer show
- [ ] `TestParseLockCommand_InvalidJSON` - Error handling

### 2.2: Package Parsing Tests

**File:** `pkg/packages/integration_test.go` (NEW ~250 lines)

Test manifest parsing using real testdata:

- [ ] `TestParse_NPM_PackageJSON`
- [ ] `TestParse_Yarn_PackageJSON`
- [ ] `TestParse_Composer_ComposerJSON`
- [ ] `TestParse_GoMod`
- [ ] `TestParse_Requirements`
- [ ] `TestParse_Pipfile`
- [ ] `TestParse_MSBuild_Csproj`
- [ ] `TestParse_NuGet_PackagesConfig`

**File:** `pkg/packages/format_preservation_test.go` (NEW ~150 lines)

- [ ] `TestParse_JSON_PreservesOrder`
- [ ] `TestParse_YAML_PreservesComments`
- [ ] `TestParse_XML_PreservesNamespaces`

### 2.3: Update Command Tests

**File:** `pkg/update/manifest_integration_test.go` (NEW ~350 lines)

Test manifest file modifications (copy to temp dir):

JavaScript:
- [ ] `TestUpdate_NPM_ModifiesPackageJSON`
- [ ] `TestUpdate_Yarn_ModifiesPackageJSON`
- [ ] `TestUpdate_PNPM_ModifiesPackageJSON`

PHP:
- [ ] `TestUpdate_Composer_ModifiesComposerJSON`

Go:
- [ ] `TestUpdate_GoMod_ModifiesGoMod`

Python:
- [ ] `TestUpdate_Requirements_ModifiesFile`
- [ ] `TestUpdate_Pipfile_ModifiesPipfile`

.NET:
- [ ] `TestUpdate_MSBuild_ModifiesCsproj`
- [ ] `TestUpdate_NuGet_ModifiesPackagesConfig`

**File:** `pkg/update/display_sync_test.go` (NEW ~150 lines)

Test VERSION/INSTALLED synchronization (from docblock requirement):

- [ ] `TestUpdate_DisplayValueSync_AfterUpdate`
- [ ] `TestUpdate_DetectsDrift_VersionMismatch`

**File:** `pkg/update/rollback_integration_test.go` (NEW ~150 lines)

- [ ] `TestUpdate_Rollback_RestoresManifest`
- [ ] `TestUpdate_Rollback_OnLockFailure`
- [ ] `TestUpdate_Rollback_PreservesPermissions`

### 2.4: Outdated Command Tests

**File:** `pkg/outdated/registry_parsing_test.go` (NEW ~250 lines)

Test registry output parsing using mocksdata/registry:

- [ ] `TestOutdated_NPM_RegistryParsing`
- [ ] `TestOutdated_Composer_RegistryParsing`
- [ ] `TestOutdated_GoMod_RegistryParsing`
- [ ] `TestOutdated_PyPI_RegistryParsing`
- [ ] `TestOutdated_NuGet_RegistryParsing`

**File:** `pkg/outdated/registry_errors_test.go` (NEW ~100 lines)

Test error handling using mocksdata_errors/registry:

- [ ] `TestOutdated_PackageNotFound_ReturnsError`
- [ ] `TestOutdated_MalformedResponse_ReturnsError`

### 2.5: Scan Command Tests

**File:** `cmd/scan_integration_test.go` (NEW ~200 lines)

- [ ] `TestScan_DetectsAllManifestTypes` - Scan pkg/testdata/
- [ ] `TestScan_RespectsExcludePatterns`
- [ ] `TestScan_IdentifiesCorrectPackageManager`
- [ ] `TestScan_HandlesNestedDirectories`

### 2.6: Config Command Tests

**File:** `pkg/config/integration_test.go` (NEW ~200 lines)

- [ ] `TestConfig_LoadsAllExamples` - Load examples/*/.goupdate.yml
- [ ] `TestConfig_ExtendsDefault` - Verify extends: [default]
- [ ] `TestConfig_RuleInheritance`
- [ ] `TestConfig_GroupValidation`

### 2.7: List Command Tests

**File:** `cmd/list_integration_test.go` (NEW ~200 lines)

- [ ] `TestList_AllPackageManagers`
- [ ] `TestList_FilterByType` - --type prod/dev
- [ ] `TestList_FilterByGroup`
- [ ] `TestList_OutputFormats` - --output json/csv/xml

### 2.8: System Test Integration

**File:** `pkg/systemtest/integration_test.go` (NEW ~200 lines)

- [ ] `TestSystemTest_Preflight_BlocksUpdate`
- [ ] `TestSystemTest_AfterEach_RunsPerPackage`
- [ ] `TestSystemTest_AfterAll_RunsOnce`
- [ ] `TestSystemTest_ContinueOnFail`

### 2.9: Real Package Manager Tests (Optional)

**File:** `tests/real_pm_test.go` (NEW ~400 lines)

Only run when package managers are installed (skip if not available):

- [ ] `TestRealPM_NPM_Install` - Skip if no npm
- [ ] `TestRealPM_PNPM_Install` - Skip if no pnpm
- [ ] `TestRealPM_Yarn_Install` - Skip if no yarn
- [ ] `TestRealPM_Composer_Update` - Skip if no composer
- [ ] `TestRealPM_GoMod_Tidy` - Skip if no go
- [ ] `TestRealPM_Pip_Install` - Skip if no pip
- [ ] `TestRealPM_Dotnet_Restore` - Skip if no dotnet

**Total new test lines:** ~2,950 lines across 15 files

---

## PHASE 3: BATTLE TESTING

Following AGENTS.md Section 7: Battle Testing Procedures

### 3.1: Clone Real-World Projects

```bash
# JavaScript ecosystems
git clone --depth 1 https://github.com/expressjs/express.git /tmp/goupdate-test/express
git clone --depth 1 https://github.com/nuxt/nuxt.git /tmp/goupdate-test/nuxt
git clone --depth 1 https://github.com/facebook/react.git /tmp/goupdate-test/react

# Go projects
git clone --depth 1 https://github.com/spf13/cobra.git /tmp/goupdate-test/cobra

# PHP projects
git clone --depth 1 https://github.com/laravel/laravel.git /tmp/goupdate-test/laravel

# Python projects
git clone --depth 1 https://github.com/django/django.git /tmp/goupdate-test/django
git clone --depth 1 https://github.com/psf/requests.git /tmp/goupdate-test/requests
```

### 3.2: Test All Commands Systematically

For each project, run:

```bash
PROJECT=/tmp/goupdate-test/express

# 1. Scan - verify file detection
goupdate scan -d $PROJECT

# 2. List - verify package parsing and lock resolution
goupdate list -d $PROJECT
goupdate list -d $PROJECT --type prod
goupdate list -d $PROJECT --type dev
goupdate list -d $PROJECT -p js

# 3. Outdated - verify version fetching
goupdate outdated -d $PROJECT
goupdate outdated -d $PROJECT --major
goupdate outdated -d $PROJECT --minor
goupdate outdated -d $PROJECT --patch

# 4. Update - test with dry-run first
goupdate update -d $PROJECT --dry-run
goupdate update -d $PROJECT --dry-run --patch

# 5. Actual update (with rollback capability)
goupdate update -d $PROJECT --patch
git -C $PROJECT diff  # Review changes
git -C $PROJECT checkout .  # Rollback
```

### 3.3: Battle Test Checklist

Document results in `docs/battle-testing-results.md`:

| Project | Scan | List | Outdated | Update --dry-run | Update | Issues Found |
|---------|------|------|----------|------------------|--------|--------------|
| express | ✅ | ✅ | ✅ | ✅ | ✅ | None |
| nuxt | ... | ... | ... | ... | ... | ... |
| react | ... | ... | ... | ... | ... | ... |
| cobra | ... | ... | ... | ... | ... | ... |
| laravel | ... | ... | ... | ... | ... | ... |
| django | ... | ... | ... | ... | ... | ... |
| requests | ... | ... | ... | ... | ... | ... |

**What to look for:**
- [ ] Invalid output formats or misaligned tables
- [ ] Incorrect status values
- [ ] Missing or wrong version numbers
- [ ] Errors that should be warnings (or vice versa)
- [ ] Crashes or panics
- [ ] Package manager detection failures
- [ ] Lock file resolution failures

---

## PHASE 4: CHAOS ENGINEERING

Following AGENTS.md Section 8: Chaos Engineering

### 4.1: Chaos Test Methodology

1. **Inventory** all features, commands, flags, config options
2. **Break** each feature deliberately (return empty, skip logic, etc.)
3. **Test** by running `go test ./...`
4. **Verify** tests catch the breakage
5. **Fix** by adding tests if breakage wasn't caught
6. **Restore** original code

### 4.2: Chaos Test Targets

**Lock resolution functions:**
- [ ] Break `ResolveLock()` - return empty map
- [ ] Break `ParseLockFile()` - return nil
- [ ] Break `ExecuteLockCommand()` - skip execution
- [ ] Break JSON parsing - return empty result

**Package parsing functions:**
- [ ] Break `ParsePackages()` - return empty list
- [ ] Break format detection - always return "unknown"
- [ ] Break dependency extraction - skip dependencies

**Update functions:**
- [ ] Break version update - skip file write
- [ ] Break rollback - skip restore
- [ ] Break lock regeneration - skip command execution

**Outdated functions:**
- [ ] Break registry fetching - return empty versions
- [ ] Break version comparison - always return "up-to-date"
- [ ] Break constraint validation - skip validation

**Scan functions:**
- [ ] Break file detection - return empty list
- [ ] Break PM identification - always return "unknown"

### 4.3: Chaos Test Documentation

Document results in `docs/chaos-testing-results.md`:

| Test ID | Feature | File | Line | Break Pattern | Tests Caught? | Action |
|---------|---------|------|------|---------------|---------------|--------|
| 1.1 | ResolveLock | pkg/lock/resolve.go | 45 | return nil | YES/NO | None/Added test |
| 1.2 | ParsePackages | pkg/packages/parse.go | 78 | return []Package{} | YES/NO | None/Added test |
| ... | ... | ... | ... | ... | ... | ... |

### 4.4: Break Patterns

Examples:
```go
// Return empty
func SomeFunction() []string {
    return nil // CHAOS TEST: Return empty
}

// Skip validation
func Validate(x string) error {
    return nil // CHAOS TEST: Skip validation
}

// Always return false
func IsValid(x string) bool {
    return false // CHAOS TEST: Always return false
}
```

---

## PHASE 5: VALIDATION & POLISH

### 5.1: Pre-release Validation

```bash
# 1. All tests pass
go test ./...

# 2. Race detection
go test -race ./...

# 3. Static analysis
go vet ./...

# 4. Coverage check
go test -cover ./pkg/... ./cmd/...

# 5. Build verification
go build ./...

# 6. Makefile targets
make test
make coverage
make coverage-func
make check
```

### 5.2: Coverage Validation

Verify coverage targets are met (from AGENTS.md Section 9):

| Package | Target | Current | Status |
|---------|--------|---------|--------|
| pkg/config | 80% | 100% | ✅ |
| pkg/formats | 85% | 100% | ✅ |
| pkg/lock | 80% | 93.4% | ✅ |
| pkg/outdated | 80% | 76.6% | ⚠️ Need improvement |
| pkg/update | 80% | 86.7% | ✅ |
| pkg/preflight | 75% | TBD | - |
| cmd | 70% | 77.6% | ✅ |

**Action:** Improve `pkg/outdated` coverage to meet 80% target.

### 5.3: Documentation Updates

Update all affected documentation (AGENTS.md Section 6):

- [ ] `docs/testing.md` - Add integration test guidelines
- [ ] `docs/developer/testing.md` - Update with new test structure
- [ ] `pkg/testdata/README.md` - Document testdata structure
- [ ] `pkg/mocksdata/README.md` - Document mocksdata purpose
- [ ] `docs/battle-testing-results.md` - Battle test results
- [ ] `docs/chaos-testing-results.md` - Chaos test results
- [ ] `README.md` - Update if needed

### 5.4: Makefile Updates

Update `Makefile` with new test targets:

```makefile
test-unit:         # Fast, no external deps
	go test -race -v -short ./...

test-integration:  # Uses testdata + mocksdata
	go test -race -v -run Integration ./...

test-real-pm:      # Uses actual package managers (skips if not installed)
	go test -race -v ./tests/...

test-all:          # Everything
	go test -race -v ./...

test-coverage:     # Coverage report
	go test -cover ./pkg/... ./cmd/...
```

---

## DELIVERABLES

### Code
1. ✅ New directory structure (`mocksdata/`, `mocksdata_errors/`)
2. ✅ Complete testdata for all 9 package managers
3. ✅ ~2,950 lines of integration tests across 15 files
4. ✅ Real PM tests with skip conditions
5. ✅ Updated Makefile with new test targets

### Testing & Validation
6. ✅ Battle testing results on 7+ real-world projects
7. ✅ Chaos engineering validation of test coverage
8. ✅ 100% code coverage maintained
9. ✅ All tests passing (unit + integration + race)

### Documentation
10. ✅ `pkg/testdata/README.md` - Testdata structure guide
11. ✅ `pkg/mocksdata/README.md` - Mocksdata usage guide
12. ✅ `docs/battle-testing-results.md` - Battle test results
13. ✅ `docs/chaos-testing-results.md` - Chaos test results
14. ✅ `docs/testing.md` - Updated integration test guidelines
15. ✅ This agent progress log

---

## RISKS & MITIGATION

### Risk 1: Tests may reveal existing bugs
**Mitigation:** Fix bugs as discovered, add regression tests

### Risk 2: Coverage may drop during implementation
**Mitigation:** Following TASK-FIRST workflow - add tests after implementation

### Risk 3: Real PM tests may fail in CI
**Mitigation:** Use skip conditions when package managers not available

### Risk 4: Battle testing may reveal config issues
**Mitigation:** Fix config issues in default.yml and document

---

## SUCCESS CRITERIA

- [ ] All 9 package managers have complete testdata
- [ ] All 9 package managers have integration tests
- [ ] All commands (scan, list, outdated, update, config) have integration tests
- [ ] Battle tested on 7+ real-world projects with no critical issues
- [ ] Chaos testing validates 100% test coverage
- [ ] All tests pass: `go test ./...`
- [ ] Race detector clean: `go test -race ./...`
- [ ] Coverage targets met for all packages
- [ ] Documentation complete and up-to-date

---

## NEXT STEPS

**Awaiting approval to proceed with:**
1. Phase 1: Implementation (directory structure, testdata, mocksdata)
2. Phase 2: Testing (integration tests for all PMs and commands)
3. Phase 3: Battle Testing (7+ real-world projects)
4. Phase 4: Chaos Engineering (validate test coverage)
5. Phase 5: Validation & Polish (coverage, docs, Makefile)

**Estimated effort:** ~2-3 days full implementation + testing + validation

---

## NOTES

- Following AGENTS.md TASK-FIRST WORKFLOW: Implementation first, tests after
- Using real package files in testdata, mocks only for registry/command output
- Battle testing and chaos engineering will validate quality before merge
- All changes on branch: `claude/review-config-tests-LwFiQ`
