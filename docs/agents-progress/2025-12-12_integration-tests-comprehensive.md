# Task: Comprehensive Integration Tests for goupdate

**Agent:** Claude
**Date:** 2025-12-12
**Status:** Planning - Awaiting Approval
**Branch:** claude/review-config-tests-LwFiQ

---

## Executive Summary

Add comprehensive integration tests for all 9 officially supported package managers and all goupdate commands to prevent configuration and integration bugs before release. This plan follows AGENTS.md TASK-FIRST WORKFLOW and incorporates insights from:

- ✅ Complete codebase architecture review
- ✅ Existing test patterns analysis (973 tests, integration_test.go pattern)
- ✅ GitHub workflow requirements (95% total coverage, 75% function minimum)
- ✅ Documentation review (testing.md, chaos-testing.md, AGENTS.md)
- ✅ Critical code path analysis (docblocks, error handling)

**Key Insight:** The pnpm --lockfile-only bug would have been caught by:
1. Integration test that verifies lock resolution after update
2. Command execution test that runs pnpm ls --json (would fail if node_modules missing)
3. Real PM test that executes full update cycle

---

## Objective

Implement **defense-in-depth** testing strategy:

1. **Integration Tests** - Test real package files through core functions (pkg/lock/integration_test.go pattern)
2. **Command Execution Tests** - Test actual command execution with testdata (copy to temp, run, verify)
3. **End-to-End Tests** - Test complete CLI workflows (scan → list → outdated → update)
4. **Battle Testing** - Test on real-world projects (Express, React, Laravel, Django, etc.)
5. **Chaos Engineering** - Validate test coverage catches breakages
6. **CI/CD Integration** - Ensure tests run in GitHub Actions

---

## Current State Analysis

### Coverage Status (from make coverage-func)

| Package | Current | Target | Status | Gap |
|---------|---------|--------|--------|-----|
| pkg/config | 100% | 80% | ✅ | None |
| pkg/formats | 100% | 85% | ✅ | None |
| pkg/lock | 93.4% | 80% | ✅ | None |
| **pkg/outdated** | **76.6%** | **80%** | **⚠️** | **Need +3.4%** |
| pkg/update | 86.7% | 80% | ✅ | None |
| pkg/preflight | 84.2% | 75% | ✅ | None |
| pkg/utils | 97.7% | 80% | ✅ | None |
| pkg/packages | 100% | 80% | ✅ | None |
| pkg/warnings | 100% | 80% | ✅ | None |
| pkg/cmdexec | 88.0% | 75% | ✅ | None |
| cmd | 77.6% | 70% | ✅ | None |

**CI Requirement:** Total ≥95%, All functions ≥75%

### Package Manager Testdata Matrix

| Manager | Manifest | Lock File | Testdata | Integration Test | Missing |
|---------|----------|-----------|----------|------------------|---------|
| npm | package.json | package-lock.json | ✅ | ✅ TestIntegration_NPM | None |
| pnpm | package.json | pnpm-lock.yaml | ❌ | ❌ | **testdata + test** |
| yarn | package.json | yarn.lock | ⚠️ errors only | ❌ | **testdata + test** |
| composer | composer.json | composer.lock | ✅ | ✅ TestIntegration_Composer | None |
| mod | go.mod | go.sum | ✅ | ✅ TestIntegration_GoMod | None |
| requirements | requirements*.txt | (self-pinning) | ✅ | ❌ | **test only** |
| pipfile | Pipfile | Pipfile.lock | ✅ | ❌ | **test only** |
| msbuild | *.csproj | packages.lock.json | ✅ | ❌ | **test only** |
| nuget | packages.config | packages.lock.json | ✅ | ❌ | **test only** |

**Priority:** Add missing testdata first (pnpm, yarn), then integration tests for all 9 PMs.

### Existing Test Infrastructure

**Strengths:**
- ✅ 973 test functions across cmd/ and pkg/
- ✅ Integration test pattern established (integration_test.go)
- ✅ Comprehensive testdata_errors/ structure for error scenarios
- ✅ Race detector enabled in all test targets
- ✅ Make targets: test, test-unit, test-e2e, coverage, coverage-func
- ✅ Chaos testing methodology documented (docs/internal/chaos-testing.md)

**Gaps:**
- ❌ No command execution tests (would catch --lockfile-only bug)
- ❌ No end-to-end CLI workflow tests
- ❌ No real PM tests (optional, skip if tool not installed)
- ❌ Missing integration tests for 5 package managers
- ❌ pkg/outdated below 80% coverage target

---

## Critical Code Paths (From Docblock Analysis)

### 1. Lock Resolution Flow (pkg/lock/resolve.go)
```
ApplyInstalledVersions()
├─ Group packages by rule
├─ For each rule:
│  ├─ Check SelfPinning mode → use declared version
│  ├─ resolveInstalledVersions()
│  │  ├─ Command-based: Execute pnpm ls, npm ls, etc.
│  │  └─ File-based: Parse lock file with regex/JSONPath
│  └─ Set InstalledVersion + InstallStatus
└─ Return enriched packages
```

**Test Gap:** No tests that verify command-based lock resolution actually works with real commands.

### 2. Update Flow (pkg/update/core.go, cmd/update.go)
```
UpdatePackage()
├─ Backup manifest atomically
├─ Read manifest file
├─ updateDeclaredVersion() → update version in content
├─ Write to temp file → atomic rename
├─ Run lock command (unless --skip-lock)
│  └─ CRITICAL: Lock command must produce state compatible with lock resolution
└─ Validate: CheckVersionUpdated()
   └─ CRITICAL: VERSION/INSTALLED must match new values (not cached old values)
```

**Test Gap:** No tests that verify display values sync after update (this was explicitly mentioned in testing.md).

### 3. Outdated Flow (pkg/outdated/core.go)
```
ListNewerVersions()
├─ Resolve effective outdated config (base + overrides)
├─ Run command with {{package}}, {{version}} placeholders
├─ Parse output (JSON/YAML/raw regex extraction)
├─ Apply version exclusions (global + rule + package)
├─ Filter newer versions using versioning strategy
└─ FilterVersionsByConstraint() → respect ^, ~, >=, etc.
```

**Test Gap:** No integration tests that use real captured registry output.

---

## IMPLEMENTATION PLAN

Following **AGENTS.md Section 0: TASK-FIRST WORKFLOW**

### Phase Flow
```
Phase 1: Implementation (Get it working)
  ↓
Phase 2: Testing (Add tests after implementation)
  ↓
Phase 3: Battle Testing (Real-world projects)
  ↓
Phase 4: Chaos Engineering (Validate test coverage)
  ↓
Phase 5: Validation & Polish (Coverage, docs, Makefile)
```

---

## PHASE 1: IMPLEMENTATION (Get it working)

**Goal:** Create directory structure and testdata that enables all integration tests.

### 1.1: Complete pnpm Testdata (PRIORITY 1)

**Location:** `pkg/testdata/pnpm/`

**Files to create:**
```
pkg/testdata/pnpm/
├── package.json         # Real dependencies: lodash, express, typescript, prettier
├── pnpm-lock.yaml       # Real pnpm-lock.yaml v9 format
└── .goupdate.yml        # Override for offline testing
```

**package.json content:**
```json
{
  "name": "test-pnpm-project",
  "dependencies": {
    "lodash": "^4.17.0",
    "express": "^4.18.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "prettier": "^3.0.0"
  }
}
```

**.goupdate.yml override** (for offline testing without pnpm command):
```yaml
extends: [default]
rules:
  pnpm:
    lock_files:
      - files: ["**/pnpm-lock.yaml"]
        format: yaml
        extraction:
          pattern: "packages/(?P<n>[^:]+):\\s+version: (?P<version>[^\\s]+)"
```

**How to generate:**
```bash
cd pkg/testdata/pnpm
pnpm install
# Captures real pnpm-lock.yaml
```

### 1.2: Complete yarn Testdata (PRIORITY 2)

**Location:** `pkg/testdata/yarn/`

**Files to create:**
```
pkg/testdata/yarn/
├── package.json         # Same as pnpm for consistency
├── yarn.lock            # Real yarn.lock v1 format
└── .goupdate.yml        # Override for offline testing
```

**.goupdate.yml override:**
```yaml
extends: [default]
rules:
  yarn:
    lock_files:
      - files: ["**/yarn.lock"]
        format: raw
        extraction:
          pattern: '^"?(?P<n>[^@\\s]+)@[^:]+:\\s+version "(?P<version>[^"]+)"'
```

**How to generate:**
```bash
cd pkg/testdata/yarn
yarn install
```

### 1.3: Verify Existing Testdata Has .goupdate.yml Overrides

**Required for offline testing (no network calls in CI):**

Check each directory:
- [ ] `pkg/testdata/npm/.goupdate.yml` - Override npm ls command with regex extraction
- [ ] `pkg/testdata/composer/.goupdate.yml` - Override composer show with file parsing
- [ ] `pkg/testdata/mod/.goupdate.yml` - Verify go.sum regex extraction
- [ ] `pkg/testdata/requirements/.goupdate.yml` - Self-pinning mode
- [ ] `pkg/testdata/pipfile/.goupdate.yml` - File-based lock parsing
- [ ] `pkg/testdata/msbuild/.goupdate.yml` - XML parsing
- [ ] `pkg/testdata/nuget/.goupdate.yml` - XML parsing

**Purpose:** Ensures integration tests run without requiring actual package manager tools in CI.

### 1.4: Capture Real Command Output Samples (NEW)

**Purpose:** Enable testing of command output parsing without running actual commands.

**Location:** `pkg/testdata_samples/` (NEW directory)

**Structure:**
```
pkg/testdata_samples/
├── README.md                           # Explains purpose and how to regenerate
├── lock-commands/                      # Lock resolution command outputs
│   ├── pnpm-ls-standard.json          # pnpm ls --json --depth=0
│   ├── npm-ls-standard.json           # npm ls --json
│   ├── yarn-list-standard.json        # yarn list --json
│   └── composer-show-standard.json    # composer show --format=json
├── outdated-commands/                  # Registry query outputs
│   ├── npm-view-lodash.json           # npm view lodash versions --json
│   ├── npm-view-express.json          # npm view express versions --json
│   ├── composer-show-monolog.json     # composer show monolog/monolog --all --format=json
│   ├── go-list-cobra.txt              # go list -m -versions github.com/spf13/cobra
│   └── pypi-requests.json             # curl https://pypi.org/pypi/requests/json
└── errors/                             # Error scenarios
    ├── invalid-json-pnpm-ls.json      # Truncated JSON
    ├── package-not-found-404.json     # Registry 404 response
    └── malformed-composer-show.json   # Invalid JSON
```

**How to generate:**
```bash
# From real installations
cd pkg/testdata/pnpm
pnpm ls --json --depth=0 > ../testdata_samples/lock-commands/pnpm-ls-standard.json

cd pkg/testdata/npm
npm ls --json > ../testdata_samples/lock-commands/npm-ls-standard.json

# Registry queries
npm view lodash versions --json > pkg/testdata_samples/outdated-commands/npm-view-lodash.json
composer show monolog/monolog --all --format=json > pkg/testdata_samples/outdated-commands/composer-show-monolog.json
```

**Purpose:** Tests can verify JSON parsing logic without network access or tool dependencies.

---

## PHASE 2: TESTING (After Phase 1 complete)

**Goal:** Add integration tests that would have caught the pnpm --lockfile-only bug.

### 2.1: Lock Resolution Integration Tests

**File:** `pkg/lock/integration_test.go` (extend existing, add ~150 lines)

**Tests to add (following existing TestIntegration_NPM pattern):**

```go
// TestIntegration_PNPM tests pnpm lock resolution with real testdata
func TestIntegration_PNPM(t *testing.T) {
    testdataDir, _ := filepath.Abs("../testdata/pnpm")
    cfg, _ := config.LoadConfig("", testdataDir)

    parser := packages.NewDynamicParser()
    result, _ := parser.ParseFile(filepath.Join(testdataDir, "package.json"), &cfg.Rules["pnpm"])

    for i := range result.Packages {
        result.Packages[i].Rule = "pnpm"
    }

    enriched, _ := ApplyInstalledVersions(result.Packages, cfg, testdataDir)

    // Verify lodash, express, typescript, prettier have installed versions
    lookup := make(map[string]string)
    for _, pkg := range enriched {
        lookup[pkg.Name] = pkg.InstalledVersion
    }

    assert.NotEmpty(t, lookup["lodash"], "lodash should have installed version")
    assert.NotEmpty(t, lookup["express"], "express should have installed version")
    assert.NotEmpty(t, lookup["typescript"], "typescript should have installed version")
}

// TestIntegration_Yarn - Same pattern for yarn
// TestIntegration_Requirements - Self-pinning mode
// TestIntegration_Pipfile - Pipfile.lock parsing
// TestIntegration_MSBuild - packages.lock.json XML parsing
// TestIntegration_NuGet - packages.config XML parsing
```

**What this catches:**
- Lock file format changes
- Parsing regex breakage
- InstallStatus assignment logic

**Estimated:** +150 lines (5 new tests × ~30 lines each)

### 2.2: Command Output Parsing Tests (NEW)

**File:** `pkg/lock/command_output_test.go` (NEW ~200 lines)

**Purpose:** Test JSON/raw output parsing using testdata_samples.

```go
func TestParseLockCommand_PNPM_JSON(t *testing.T) {
    // Read sample output
    content, _ := os.ReadFile("../testdata_samples/lock-commands/pnpm-ls-standard.json")

    // Parse using extraction config
    cfg := config.LockCommandExtractionCfg{
        Format: "json",
        JSONKey: "dependencies",
        // ... extraction config
    }

    versions, err := parseCommandOutput(content, &cfg)
    require.NoError(t, err)

    // Verify expected packages extracted
    assert.Contains(t, versions, "lodash")
    assert.Contains(t, versions, "express")
    assert.Regexp(t, `^\d+\.\d+\.\d+$`, versions["lodash"]) // Semver format
}

// Similar tests for:
// - TestParseLockCommand_NPM_JSON
// - TestParseLockCommand_Yarn_JSON
// - TestParseLockCommand_Composer_JSON
// - TestParseLockCommand_InvalidJSON (error handling)
```

**What this catches:**
- JSON extraction path breakage (e.g., dependencies → packages)
- Regex extraction changes
- Invalid JSON handling

**Estimated:** ~200 lines (8 tests × ~25 lines each)

### 2.3: Update Display Value Sync Tests (NEW)

**File:** `pkg/update/display_sync_integration_test.go` (NEW ~150 lines)

**Purpose:** Verify VERSION/INSTALLED columns show NEW values after update (not cached old values).

**From testing.md critical requirement:**
> The VERSION and INSTALLED columns must show the NEW version after update, NOT the old cached version.

```go
func TestUpdate_DisplayValueSync_AfterUpdate(t *testing.T) {
    // Setup temp dir with testdata
    tmpDir := copyTestdataToTemp(t, "npm")

    cfg, _ := config.LoadConfig("", tmpDir)

    // 1. Get initial state
    packages := parseAndResolve(t, tmpDir, cfg)
    oldVersion := findPackage(packages, "lodash").Version
    oldInstalled := findPackage(packages, "lodash").InstalledVersion

    // 2. Update package to new version
    targetVersion := "4.17.21"
    err := update.UpdatePackage(pkg, targetVersion, cfg, tmpDir, false, true) // skipLock=true for test
    require.NoError(t, err)

    // 3. CRITICAL: Re-parse and re-resolve to get fresh state
    updatedPackages := parseAndResolve(t, tmpDir, cfg)
    updatedPkg := findPackage(updatedPackages, "lodash")

    // 4. Verify display values updated
    assert.Equal(t, targetVersion, updatedPkg.Version, "VERSION column must show new version")
    assert.Equal(t, targetVersion, updatedPkg.InstalledVersion, "INSTALLED column must show new version")
    assert.NotEqual(t, oldVersion, updatedPkg.Version, "VERSION must have changed")
    assert.NotEqual(t, oldInstalled, updatedPkg.InstalledVersion, "INSTALLED must have changed")
}
```

**What this catches:**
- Cached value bugs (displaying old values after update)
- File not re-read after update
- Lock file not regenerated correctly

**Estimated:** ~150 lines (3 tests for npm/composer/mod)

### 2.4: pkg/outdated Coverage Improvement

**Goal:** Increase pkg/outdated from 76.6% to ≥80%

**File:** `pkg/outdated/integration_test.go` (NEW ~100 lines)

**Tests using testdata_samples:**

```go
func TestOutdated_NPM_RegistryParsing(t *testing.T) {
    // Read captured npm view output
    content, _ := os.ReadFile("../testdata_samples/outdated-commands/npm-view-lodash.json")

    cfg := config.OutdatedCfg{
        Format: "json",
        Extraction: &config.OutdatedExtractionCfg{
            JSONKey: "versions",
        },
    }

    versions, err := parseRegistryOutput(content, &cfg)
    require.NoError(t, err)

    assert.Contains(t, versions, "4.17.21")
    assert.Greater(t, len(versions), 10, "should have many versions")
}

// Similar for composer, go mod, pypi
```

**What this catches:**
- Registry response format changes
- JSON extraction breakage
- Version parsing issues

**Estimated:** ~100 lines (4 tests × ~25 lines)

### 2.5: End-to-End CLI Workflow Tests (NEW)

**File:** `cmd/e2e_workflow_test.go` (NEW ~300 lines)

**Purpose:** Test complete workflows that span multiple commands.

```go
func TestE2E_ScanListOutdatedUpdate_NPM(t *testing.T) {
    tmpDir := copyTestdataToTemp(t, "npm")

    // 1. Scan should detect package.json
    scanOut := runCommand(t, "scan", "-d", tmpDir)
    assert.Contains(t, scanOut, "package.json")
    assert.Contains(t, scanOut, "npm")

    // 2. List should show packages
    listOut := runCommand(t, "list", "-d", tmpDir)
    assert.Contains(t, listOut, "lodash")
    assert.Contains(t, listOut, "express")

    // 3. Outdated should detect updates
    outdatedOut := runCommand(t, "outdated", "-d", tmpDir, "--patch")
    assert.Contains(t, outdatedOut, "Outdated") // Status column

    // 4. Update dry-run should plan updates
    dryRunOut := runCommand(t, "update", "-d", tmpDir, "--dry-run", "--patch")
    assert.Contains(t, dryRunOut, "lodash")
    assert.NotContains(t, dryRunOut, "Updated") // Should not actually update

    // 5. Verify files not modified
    originalContent := readFile(t, filepath.Join(tmpDir, "package.json"))
    assert.Contains(t, originalContent, "4.17.0") // Original version
}

// Similar for: composer, mod, requirements, pipfile
```

**What this catches:**
- Command integration issues
- Flag handling across commands
- Output format consistency
- Dry-run safety

**Estimated:** ~300 lines (5 PM tests × ~60 lines each)

### 2.6: Real Package Manager Tests (NEW, OPTIONAL)

**File:** `tests/real_pm_test.go` (NEW ~400 lines)

**Purpose:** Test with actual package manager tools (skip if not installed).

```go
func TestRealPM_PNPM_UpdateCycle(t *testing.T) {
    // Skip if pnpm not installed
    if _, err := exec.LookPath("pnpm"); err != nil {
        t.Skip("pnpm not installed, skipping real PM test")
    }

    tmpDir := t.TempDir()

    // 1. Create package.json
    writeFile(t, filepath.Join(tmpDir, "package.json"), `{
        "dependencies": {
            "lodash": "^4.17.0"
        }
    }`)

    // 2. Run pnpm install
    runCmd(t, tmpDir, "pnpm", "install")

    // 3. Verify lock file created
    assert.FileExists(t, filepath.Join(tmpDir, "pnpm-lock.yaml"))

    // 4. Run goupdate update --patch
    runCommand(t, "update", "-d", tmpDir, "-r", "pnpm", "--patch", "-y", "--skip-lock")

    // 5. Verify package.json updated
    content := readFile(t, filepath.Join(tmpDir, "package.json"))
    assert.Regexp(t, `4\.17\.\d{2,}`, content) // Should be 4.17.21 or higher

    // 6. Run pnpm install to regenerate lock
    runCmd(t, tmpDir, "pnpm", "install")

    // 7. CRITICAL: Verify pnpm ls works (catches --lockfile-only bug)
    output := runCmd(t, tmpDir, "pnpm", "ls", "--json", "--depth=0")
    assert.Contains(t, output, "lodash")
    assert.NotContains(t, output, "ERR_PNPM_")
}

// Similar for: npm, yarn, composer, go, pip, dotnet
```

**What this catches:**
- Actual command compatibility issues
- Lock file regeneration correctness
- **The pnpm --lockfile-only bug** (pnpm ls would fail)

**Estimated:** ~400 lines (7 PMs × ~55 lines each)

**Note:** These tests are optional in CI (skip if tools not available).

---

## PHASE 3: BATTLE TESTING

**Goal:** Test on real-world projects to catch issues unit tests miss.

**Following:** docs/internal/chaos-testing.md Battle Testing section

### 3.1: Clone Real-World Projects

```bash
mkdir -p /tmp/goupdate-battle-test
cd /tmp/goupdate-battle-test

# JavaScript
git clone --depth 1 https://github.com/expressjs/express.git
git clone --depth 1 https://github.com/nuxt/nuxt.git
git clone --depth 1 https://github.com/facebook/react.git

# Go
git clone --depth 1 https://github.com/spf13/cobra.git

# PHP
git clone --depth 1 https://github.com/laravel/laravel.git

# Python
git clone --depth 1 https://github.com/django/django.git
git clone --depth 1 https://github.com/psf/requests.git
```

### 3.2: Test Matrix

For each project, run:

```bash
PROJECT=/tmp/goupdate-battle-test/express

# Build fresh binary with changes
go build -o /tmp/goupdate-test ./

# Non-destructive tests
/tmp/goupdate-test scan -d $PROJECT
/tmp/goupdate-test list -d $PROJECT
/tmp/goupdate-test list -d $PROJECT --type prod
/tmp/goupdate-test list -d $PROJECT --type dev
/tmp/goupdate-test outdated -d $PROJECT
/tmp/goupdate-test outdated -d $PROJECT --major
/tmp/goupdate-test outdated -d $PROJECT --minor
/tmp/goupdate-test outdated -d $PROJECT --patch
/tmp/goupdate-test update -d $PROJECT --dry-run
/tmp/goupdate-test update -d $PROJECT --dry-run --patch

# Destructive test (in test branch)
cd $PROJECT
git checkout -b goupdate-test
/tmp/goupdate-test update -d . --patch -y
git diff  # Verify changes
git checkout main
git branch -D goupdate-test
```

### 3.3: Battle Test Checklist

**What to verify:**

- [ ] Scan detects all manifest files correctly
- [ ] List shows all packages with correct types (prod/dev)
- [ ] Outdated detects updates without errors
- [ ] Outdated --major/--minor/--patch filtering works
- [ ] Update --dry-run doesn't modify files
- [ ] Update actually modifies manifest files correctly
- [ ] No table formatting issues (alignment, emoji rendering)
- [ ] No crashes or panics
- [ ] Status column values correct (🟢 UpToDate, 🟠 Outdated, etc.)
- [ ] Version numbers displayed correctly

### 3.4: Document Results

**File:** `docs/battle-testing-results.md` (NEW)

```markdown
# Battle Testing Results

## Test Date: 2025-12-12

| Project | PM | Commands Tested | Result | Issues Found |
|---------|-----|-----------------|--------|--------------|
| express | npm | scan, list, outdated, update --dry-run | ✅ Pass | None |
| nuxt | pnpm | scan, list, outdated, update --dry-run | ✅ Pass | None |
| ... | ... | ... | ... | ... |
```

---

## PHASE 4: CHAOS ENGINEERING

**Goal:** Validate that tests catch breakages when features are deliberately broken.

**Following:** docs/internal/chaos-testing.md methodology

### 4.1: Chaos Test Execution

For each critical feature:

```bash
# 1. Break feature
# 2. Run tests
# 3. Verify tests fail
# 4. Restore code

# Example: Break lock resolution
echo "Breaking ApplyInstalledVersions..."
# Edit pkg/lock/resolve.go: return empty instead of resolved versions

go test ./... 2>&1 | tee chaos-test-results.txt

if grep -q "FAIL" chaos-test-results.txt; then
    echo "✅ Tests caught the breakage"
else
    echo "❌ Tests did NOT catch - need to add tests"
fi

# Restore original code
git checkout pkg/lock/resolve.go
```

### 4.2: Features to Test

**High Priority (Critical Paths):**

| Feature | File | Break Pattern | Expected |
|---------|------|---------------|----------|
| ApplyInstalledVersions | pkg/lock/resolve.go | return nil | Integration tests fail |
| updateJSONVersion | pkg/update/json.go | return original content | Update tests fail |
| ListNewerVersions | pkg/outdated/core.go | return empty slice | Outdated tests fail |
| FilterVersionsByConstraint | pkg/outdated/core.go | return all versions | Constraint tests fail |
| ParsePackages | pkg/packages/parser.go | return empty result | Parser tests fail |

**Medium Priority:**

| Feature | File | Break Pattern | Expected |
|---------|------|---------------|----------|
| LoadConfig | pkg/config/load.go | return empty config | Config tests fail |
| ValidatePackages | pkg/preflight/preflight.go | always return nil | Preflight tests fail |
| matchesTypeFilter | cmd/list.go | always return true | Filter tests fail |

### 4.3: Document Results

**File:** `docs/chaos-testing-results.md` (UPDATE)

Add new section:

```markdown
## Chaos Test Results: 2025-12-12

| Test ID | Feature | File | Tests Caught? | Action |
|---------|---------|------|---------------|--------|
| L1 | ApplyInstalledVersions | pkg/lock/resolve.go | YES (15+ tests) | None |
| L2 | resolveInstalledVersions | pkg/lock/resolve.go | YES (integration) | None |
| U1 | updateJSONVersion | pkg/update/json.go | YES (5 tests) | None |
| U2 | DisplayValueSync | pkg/update/core.go | YES (new test) | None |
| O1 | ListNewerVersions | pkg/outdated/core.go | YES (3 tests) | None |
| ... | ... | ... | ... | ... |

### Summary
- X chaos tests executed
- Y tests caught breakages (Y/X%)
- Z gaps identified and fixed
```

---

## PHASE 5: VALIDATION & POLISH

**Goal:** Ensure CI will pass, coverage targets met, docs updated.

### 5.1: Pre-Commit Validation

Run all checks locally before pushing:

```bash
# 1. All tests pass
make test

# 2. Race detector clean
go test -race ./...

# 3. Static analysis
make vet

# 4. Coverage check (must be ≥95% total)
make coverage-func | grep "^total:"
# Output should be ≥95.0%

# 5. Function coverage check (all functions ≥75%)
make coverage-func | grep -v "^total:" | awk 'NF==3 && $NF < 75 && $1 !~ /main.go/'
# Output should be empty

# 6. Build verification
go build ./...

# 7. Formatting
make fmt
git diff --exit-code  # Should have no changes
```

### 5.2: Coverage Targets Verification

**After adding all tests, verify:**

| Package | Target | Expected After | Verification |
|---------|--------|----------------|--------------|
| pkg/outdated | 80% | ~82% | New integration tests |
| Total | 95% | ~96% | All new tests |

**If below target:**
- Run `make coverage-func` to identify low-coverage functions
- Add specific tests for those functions
- Prioritize functions in critical paths

### 5.3: Update Makefile

**File:** `Makefile` (UPDATE)

Add new test targets:

```makefile
# Run integration tests only
test-integration:
	go test -race -v -run Integration ./pkg/...

# Run E2E workflow tests
test-e2e-workflow:
	go test -race -v ./cmd/... -run E2E

# Run real PM tests (skip if tools not installed)
test-real-pm:
	go test -race -v ./tests/... -run RealPM

# Run battle tests (manual)
battle-test:
	@echo "Battle testing on real projects..."
	@./scripts/battle-test.sh

# Run chaos tests (manual)
chaos-test:
	@echo "Running chaos engineering tests..."
	@./scripts/chaos-test.sh
```

### 5.4: Update Documentation

**Files to update:**

1. **docs/developer/testing.md** (UPDATE)
   - Add section on integration test patterns
   - Document testdata_samples/ directory
   - Add real PM test guidelines

2. **pkg/testdata/README.md** (CREATE)
   ```markdown
   # Testdata Directory Structure

   This directory contains real package files for integration testing.

   ## Structure
   - npm/ - NPM package.json + package-lock.json
   - pnpm/ - PNPM package.json + pnpm-lock.yaml
   - yarn/ - Yarn package.json + yarn.lock
   - ...

   ## Offline Testing
   Each directory has .goupdate.yml that overrides commands with file-based parsing.
   ```

3. **pkg/testdata_samples/README.md** (CREATE)
   ```markdown
   # Test Data Samples

   Captured real command outputs for testing without running actual commands.

   ## Regenerating Samples
   ```bash
   cd pkg/testdata/pnpm
   pnpm ls --json --depth=0 > ../testdata_samples/lock-commands/pnpm-ls-standard.json
   ```
   ```

4. **docs/battle-testing-results.md** (CREATE) - From Phase 3

5. **docs/chaos-testing-results.md** (UPDATE) - From Phase 4

### 5.5: CI/CD Verification

**Verify GitHub Actions will pass:**

```bash
# Simulate CI environment
export CI=true

# Run tests as CI does
go test -race ./...

# Run coverage as CI does
make coverage-func

# Verify coverage meets PR workflow requirements:
# - Total ≥95%
# - All functions ≥75%
```

**GitHub PR workflow checks (.github/workflows/pr.yml):**
- ✅ Tests pass (make test)
- ✅ Coverage ≥95% total
- ✅ All functions ≥75%
- ✅ Lint passes (golangci-lint)

---

## DELIVERABLES SUMMARY

### Code Changes

| File | Type | Lines | Purpose |
|------|------|-------|---------|
| pkg/testdata/pnpm/* | NEW | ~100 | PNPM testdata with real lock file |
| pkg/testdata/yarn/* | NEW | ~100 | Yarn testdata with real lock file |
| pkg/testdata_samples/* | NEW | ~50 | Captured command outputs |
| pkg/lock/integration_test.go | UPDATE | +150 | 5 new integration tests |
| pkg/lock/command_output_test.go | NEW | ~200 | Command output parsing tests |
| pkg/update/display_sync_integration_test.go | NEW | ~150 | Display value sync tests |
| pkg/outdated/integration_test.go | NEW | ~100 | Registry parsing tests |
| cmd/e2e_workflow_test.go | NEW | ~300 | End-to-end workflow tests |
| tests/real_pm_test.go | NEW | ~400 | Real PM execution tests |
| **Total** | | **~1,550** | **New test code** |

### Documentation

| File | Type | Purpose |
|------|------|---------|
| pkg/testdata/README.md | NEW | Testdata structure guide |
| pkg/testdata_samples/README.md | NEW | Samples regeneration guide |
| docs/battle-testing-results.md | NEW | Battle test results |
| docs/chaos-testing-results.md | UPDATE | Chaos test results |
| docs/developer/testing.md | UPDATE | Integration test guidelines |
| Makefile | UPDATE | New test targets |

### Testing Artifacts

| Artifact | Purpose |
|----------|---------|
| Battle test on 7+ real projects | Real-world validation |
| Chaos test on 10+ critical features | Coverage validation |
| 95%+ total coverage | CI requirement |
| All functions ≥75% | CI requirement |

---

## RISK MITIGATION

### Risk 1: Tests reveal existing bugs

**Likelihood:** Medium
**Impact:** Medium
**Mitigation:** Fix bugs as discovered, add regression tests

### Risk 2: Coverage drops during implementation

**Likelihood:** Low (following TASK-FIRST)
**Impact:** High (blocks PR)
**Mitigation:** Phase 1 only adds testdata (no code changes), Phase 2 adds tests

### Risk 3: Real PM tests fail in CI

**Likelihood:** High (tools not installed)
**Impact:** Low
**Mitigation:** Skip tests when tools not available (t.Skip pattern)

### Risk 4: Battle testing finds config issues

**Likelihood:** Medium
**Impact:** Medium
**Mitigation:** Fix in default.yml, add regression tests

### Risk 5: Chaos tests reveal coverage gaps

**Likelihood:** Medium
**Impact:** Medium
**Mitigation:** Add missing tests immediately, re-run chaos tests

---

## SUCCESS CRITERIA

- [x] All 9 package managers have complete testdata with .goupdate.yml overrides
- [x] All 9 package managers have integration tests in pkg/lock/integration_test.go
- [x] Command output parsing tests prevent format breakage
- [x] Display value sync tests prevent cached value bugs
- [x] End-to-end workflow tests validate CLI integration
- [x] pkg/outdated coverage ≥80%
- [x] Total coverage ≥95%
- [x] All functions ≥75% coverage
- [x] Battle tested on 7+ real-world projects with no critical issues
- [x] Chaos testing validates test coverage catches breakages
- [x] All tests pass: `go test ./...`
- [x] Race detector clean: `go test -race ./...`
- [x] Lint passes: `golangci-lint run`
- [x] Documentation complete and accurate

---

## TIMELINE ESTIMATE

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Phase 1: Implementation | 2-3 hours | None |
| Phase 2: Testing | 4-5 hours | Phase 1 complete |
| Phase 3: Battle Testing | 2-3 hours | Phase 2 complete |
| Phase 4: Chaos Engineering | 2-3 hours | Phase 2 complete |
| Phase 5: Validation & Polish | 2 hours | All above complete |
| **Total** | **12-16 hours** | Sequential execution |

**Parallelization opportunities:**
- Phase 3 and 4 can run in parallel after Phase 2

---

## NEXT STEPS

**Awaiting approval to proceed with:**

1. ✅ Phase 1: Create pnpm/yarn testdata + testdata_samples directory
2. ✅ Phase 2: Add 1,550 lines of integration/e2e/real PM tests
3. ✅ Phase 3: Battle test on Express, React, Laravel, Django, Cobra, etc.
4. ✅ Phase 4: Chaos test 10+ critical features
5. ✅ Phase 5: Verify coverage, update docs, polish

**Ready to start?** Approval needed to proceed with implementation.

---

## NOTES

- Following AGENTS.md TASK-FIRST WORKFLOW strictly
- Real package files only (no mocks for testdata)
- All changes on branch: `claude/review-config-tests-LwFiQ`
- This plan designed to prevent bugs like pnpm --lockfile-only from reaching production
- Comprehensive testing = confidence in releases

