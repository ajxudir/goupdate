# Testing and quality guide

This guide explains how to set up your environment, run the test suites, and contribute new coverage so contributors follow the same standards.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Test-driven development (TDD)](#test-driven-development-tdd)
- [Test suites](#test-suites)
- [Coverage requirements](#coverage-requirements)
- [Release regression prevention](#release-regression-prevention)
- [Chaos engineering](#chaos-engineering)
- [Adding tests](#adding-tests)
- [Static analysis and formatting](#static-analysis-and-formatting)
- [Testdata diversity rules](#testdata-diversity-rules)
- [Common troubleshooting](#common-troubleshooting)

---

## Prerequisites

- **Go:** 1.21 or later (see `go.mod` for the authoritative version).
- **Tooling:** `make`, `git`, and a POSIX shell.
- **OS support:** Commands are tested on macOS and Linux; Windows users should run inside WSL2 or a container.

Initialize dependencies once per checkout:

```bash
make init
```

## Test-driven development (TDD)

This project follows TDD principles to ensure reliability across releases:

### TDD workflow

1. **Write test first**: Before implementing a feature, write a failing test
2. **Run and see it fail**: Confirm the test fails for the right reason
3. **Implement minimum code**: Write just enough code to pass the test
4. **Run and see it pass**: Verify the implementation works
5. **Refactor**: Clean up while keeping tests green

### TDD by feature type

| Feature Type | Test Location | Example |
|--------------|---------------|---------|
| CLI command | `cmd/*_test.go` | `TestListCommand`, `TestUpdateDryRun` |
| Parser | `pkg/formats/*_test.go` | `TestJSONParser`, `TestYAMLParser` |
| Config option | `pkg/config/*_test.go` | `TestGroupCfg`, `TestExtends` |
| Lock resolution | `pkg/lock/*_test.go` | `TestApplyInstalledVersions` |
| Version logic | `pkg/outdated/*_test.go` | `TestFilterVersionsByConstraint` |
| Update logic | `pkg/update/*_test.go` | `TestUpdateJSONVersion` |

### Writing effective tests

```go
// Good: Tests specific behavior with clear assertion
func TestFilterVersionsByConstraint_PatchOnly(t *testing.T) {
    p := formats.Package{Version: "1.2.3", Constraint: "~"}
    versions := []string{"1.2.4", "1.3.0", "2.0.0"}
    flags := UpdateSelectionFlags{Patch: true}

    result := FilterVersionsByConstraint(p, versions, flags)

    assert.Equal(t, []string{"1.2.4"}, result)
}

// Good: Table-driven for multiple scenarios
func TestParseVersion(t *testing.T) {
    tests := []struct {
        name       string
        input      string
        wantVer    string
        wantConstr string
    }{
        {"caret", "^1.2.3", "1.2.3", "^"},
        {"tilde", "~1.2.3", "1.2.3", "~"},
        {"exact", "1.2.3", "1.2.3", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ParseVersion(tt.input)
            assert.Equal(t, tt.wantVer, result.Version)
            assert.Equal(t, tt.wantConstr, result.Constraint)
        })
    }
}
```

## Test suites

All test targets run with the race detector enabled.

- **Full suite:**
  ```bash
  make test
  ```
  Runs every package recursively and is the default for CI.

- **Unit tests only:**
  ```bash
  make test-unit
  ```
  Focuses on `cmd` and `pkg` packages for fast local iterations.

- **End-to-end tests only:**
  ```bash
  make test-e2e
  ```
  Executes tests tagged with `EndToEnd` to exercise CLI behavior.

To run a specific test or package directly, use Go tooling:

```bash
go test -race ./pkg/... -run TestConfigLoader
```

## Coverage requirements

### Current coverage status

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| pkg/config | 100% | 80% | ✅ Exceeds |
| pkg/formats | 100% | 85% | ✅ Exceeds |
| pkg/lock | 93.4% | 80% | ✅ Exceeds |
| pkg/outdated | 76.6% | 80% | ⚠️ Below target |
| pkg/update | 86.7% | 80% | ✅ Exceeds |
| pkg/preflight | 84.2% | 75% | ✅ Exceeds |
| pkg/utils | 97.7% | 80% | ✅ Exceeds |
| pkg/packages | 100% | 80% | ✅ Exceeds |
| pkg/warnings | 100% | 80% | ✅ Exceeds |
| pkg/cmdexec | 88.0% | 75% | ✅ Exceeds |
| cmd | 77.6% | 70% | ✅ Exceeds |

### Minimum coverage targets

| Package | Required | Critical Functions |
|---------|----------|-------------------|
| pkg/config | 80% | LoadConfig, mergeConfigs, validateGroupMembership |
| pkg/formats | 85% | All Parse methods |
| pkg/lock | 80% | ApplyInstalledVersions, resolveInstalledVersions |
| pkg/outdated | 80% | ListNewerVersions, SummarizeAvailableVersions, FilterVersionsByConstraint |
| pkg/update | 80% | UpdatePackage, updateJSONVersion, updateYAMLVersion |
| pkg/preflight | 75% | ValidatePackages, ValidateRules |
| cmd | 70% | All run* functions |

### Coverage commands

Generate and inspect coverage artifacts before sending changes:

```bash
make coverage       # Writes coverage.out with atomic mode
make coverage-func  # Prints per-function coverage summary
make coverage-html  # Opens an HTML report at coverage.html
```

### Checking coverage by package

```bash
# Overall coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# Per-package coverage
go test -cover ./pkg/config/...
go test -cover ./pkg/formats/...
go test -cover ./pkg/lock/...
```

The repository uses `coverage.out` for automation and review. Avoid committing generated artifacts.

## Release regression prevention

### Pre-release checklist

Before tagging a release, verify:

1. **All tests pass**: `go test ./...`
2. **Coverage meets targets**: Check per-package coverage
3. **No race conditions**: `go test -race ./...`
4. **Static analysis clean**: `go vet ./...`
5. **Chaos tests validated**: See [chaos-testing.md](./chaos-testing.md)

### Critical paths to test

These features must have passing tests before any release:

| Feature | Test | Why Critical |
|---------|------|--------------|
| Config loading | `TestLoadConfig*` | All commands depend on config |
| Package parsing | `TestJSONParser`, `TestYAMLParser` | Core data extraction |
| Lock resolution | `TestApplyInstalledVersions*` | Version comparison accuracy |
| Version filtering | `TestFilterVersionsByConstraint` | Correct update targeting |
| Preflight validation | `TestValidatePackages*` | Prevents runtime errors |
| Dry-run mode | `TestUpdateDryRun` | Safety feature |
| Post-update validation | `TestValidateUpdatedPackage*` | Verifies update succeeded and display values updated |
| Display value sync | Integration test (see below) | VERSION/INSTALLED must match actual file values |

### Integration test scenarios

Ensure these scenarios work end-to-end:

```bash
# Scenario 1: Basic scan and list
goupdate scan -d examples/react-app
goupdate list -d examples/react-app

# Scenario 2: Outdated with filters
goupdate outdated -d examples/react-app --type prod
goupdate outdated -d examples/react-app --patch

# Scenario 3: Dry-run update
goupdate update -d examples/react-app --dry-run

# Scenario 4: Config validation
goupdate config --show-effective -d examples/react-app
```

### Real update integration testing (CRITICAL)

**Unit tests are not sufficient for update functionality.** Always perform real update tests in a temporary directory before releasing changes to update logic.

Why this matters:
- Unit tests mock the file system and don't catch display/state synchronization bugs
- Validation functions may pass but display may show stale cached values
- Lock file regeneration behavior varies by package manager

**Required test procedure for update changes:**

```bash
# 1. Build the binary with your changes
go build -o /tmp/goupdate-bin .

# 2. Create isolated test environment (don't pollute git state)
cd /tmp && rm -rf goupdate-integration-test
mkdir goupdate-integration-test && cd goupdate-integration-test

# 3. Set up Go module test case
go mod init testproject
cat > main.go << 'EOF'
package main
import (
    "fmt"
    "github.com/spf13/cobra"
)
func main() {
    cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) { fmt.Println("test") }}
    cmd.Execute()
}
EOF
go get github.com/spf13/cobra@v1.7.0  # Use old version
go mod tidy

# 4. Verify outdated detection
/tmp/goupdate-bin outdated -r mod --skip-build-checks

# 5. Run REAL update (not dry-run) and verify output
/tmp/goupdate-bin update -r mod --minor -y --skip-build-checks

# 6. CRITICAL: Verify display values match actual file values
# The VERSION and INSTALLED columns must show the NEW version after update
# NOT the old cached version

# Check go.mod was actually updated
cat go.mod | grep cobra

# 7. Clean up
cd /tmp && rm -rf goupdate-integration-test
```

**What to verify in output:**

After a successful update, the table output must show:
- VERSION column: The new version from go.mod (e.g., `v1.10.2`)
- INSTALLED column: The new version from go.sum (e.g., `v1.10.2`)
- TARGET column: The target version (e.g., `v1.10.2`)
- STATUS column: `🟢 Updated`

**Common bug pattern to avoid:**
```
# BAD: Display shows old values despite successful update
VERSION   INSTALLED  TARGET    STATUS
v1.7.0    v1.7.0     v1.10.2   🟢 Updated  ← VERSION/INSTALLED should be v1.10.2!

# GOOD: Display shows new values after update
VERSION   INSTALLED  TARGET    STATUS
v1.10.2   v1.10.2    v1.10.2   🟢 Updated  ← Correct!
```

**NPM test case:**

```bash
cd /tmp && rm -rf npm-test && mkdir npm-test && cd npm-test
cat > package.json << 'EOF'
{
  "name": "test",
  "dependencies": {
    "lodash": "^4.17.0"
  }
}
EOF
npm install
/tmp/goupdate-bin update -r npm --minor -y --skip-build-checks
cat package.json | grep lodash
cat package-lock.json | grep -A1 '"lodash"'
```

### Regression test for bug fixes

When fixing a bug:

1. **Write a failing test** that reproduces the bug
2. **Fix the bug** to make the test pass
3. **Document the test** with issue reference if applicable

```go
// TestParseVersion_HandlesLeadingV regression test for issue #123
func TestParseVersion_HandlesLeadingV(t *testing.T) {
    // Bug: versions like "v1.2.3" were not parsed correctly
    result := ParseVersion("v1.2.3")
    assert.Equal(t, "1.2.3", result.Version)
}
```

## Chaos engineering

The project uses chaos engineering to validate test coverage. The methodology:

1. **Break** a feature by modifying its code (e.g., return empty, skip logic)
2. **Test** by running `go test ./...`
3. **Verify** that tests catch the breakage
4. **Fix** by adding tests if the breakage wasn't caught
5. **Restore** the original code

See [chaos-testing.md](./chaos-testing.md) for:
- Complete feature inventory
- Step-by-step test methodology
- Execution results and coverage validation

### Chaos test results

| Package | Features Validated | Coverage |
|---------|-------------------|----------|
| pkg/config | loadDefaultConfig, validateGroupMembership | ✅ |
| pkg/formats | JSONParser.Parse | ✅ |
| pkg/lock | ApplyInstalledVersions | ✅ |
| pkg/outdated | SummarizeAvailableVersions, FilterVersionsByConstraint | ✅ |
| pkg/update | updateJSONVersion | ✅ |
| pkg/preflight | ValidatePackages, ValidateRules | ✅ |
| pkg/utils | IsLatestIndicator | ✅ |
| cmd | matchesTypeFilter, validateUpdatedPackage | ✅ |

## Adding tests

- Place reusable fixtures under `testdata/` and mirror the directory structure of the packages under test.
- Keep `_test.go` files next to the code they cover (for example `config.go` and `config_test.go`).
- Put integration, smoke, and end-to-end style coverage that spans multiple packages under `tests/` and rely on exported APIs from the `pkg` module.
- Prefer table-driven tests for parser and discovery logic to keep assertions concise.
- Validate lock-file scenarios by combining manifests and lock files within the same fixture folder.
- When adding new package manager support, include discovery, parsing, and lock-file cases to keep regression coverage high.

## Static analysis and formatting

Run lightweight checks locally before opening a PR:

```bash
make vet   # go vet static analysis
make fmt   # gofmt -s -w across the repo
```

CI expects clean `gofmt` output and no `go vet` findings. Tests should pass without relying on network access.

## Testdata diversity rules

The project uses real package files for testing. Test fixtures must follow specific diversity rules to ensure comprehensive coverage of all scenarios.

### Directory structure

```
pkg/testdata/           # Valid test fixtures for automated testing
├── composer/           # PHP Composer (composer.json, composer.lock)
├── mod/                # Go modules (go.mod, go.sum)
├── msbuild/            # .NET MSBuild (.csproj, packages.lock.json)
├── npm/                # npm (package.json, package-lock.json)
├── nuget/              # NuGet (packages.config, packages.lock.json)
├── pipfile/            # Pipenv (Pipfile, Pipfile.lock)
├── requirements/       # pip (requirements.txt)
└── */_edge-cases/      # Edge cases (no-lock, prerelease, etc.)

pkg/_testdata/          # Error/failure scenarios for manual testing
├── _invalid-syntax/    # Malformed files (invalid JSON, XML, etc.)
├── _lock-errors/       # Packages not found in registry
├── _lock-missing/      # Lock files with missing packages
├── _malformed/         # Syntactically invalid files
├── _config-errors/     # Invalid configuration files
├── command-timeout/    # Commands that exceed timeout
├── invalid-command/    # Non-existent commands
├── malformed-json/     # Invalid JSON format
├── malformed-xml/      # Invalid XML format
└── package-not-found/  # Non-existent packages
```

### STATUS diversity

Testdata must include packages that produce different STATUS values:

| Status | Location | Description |
|--------|----------|-------------|
| 🟢 UpToDate | `testdata/` | Package at latest compatible version (target: 20-40%) |
| 🟠 Outdated | `testdata/` | Newer versions available (target: 60-80%) |
| 🔵 NotInLock | `testdata/_edge-cases/` | Package in manifest but not in lock file |
| 🟠 LockMissing | `testdata/_edge-cases/no-lock/` | Lock file doesn't exist |
| 🔴 VersionMissing | `testdata/requirements/` | No version specified in manifest |
| ⛔ Floating | `testdata/` | Floating constraint like `*` (limited use) |
| ❌ Failed | `_testdata/` | Command or network errors |

### CONSTRAINT diversity

Each package manager testdata must include multiple constraint types:

| Manager | Required Constraints | Example |
|---------|---------------------|---------|
| npm | `^`, `~`, `>=`, `<`, `=`, `*`, range, `x` notation | `^4.17.0`, `~4.18.2`, `>=1.0.0 <3.0.0`, `1.x` |
| composer | `^`, `~`, `>=,<`, `\|`, `*`, `x` notation | `^6.0`, `~3.40.0`, `3.7.*`, `^2.0\|^3.0` |
| pipfile | `>=`, `~=`, `==`, `*`, range | `>=4.0,<5.0`, `~=3.0.0`, `==2.31.0`, `*` |
| requirements | `>=`, `~=`, `==`, `*`, no-version | `>=1.24.0`, `~=3.0.0`, `==2.31.0`, `*`, `redis` |
| mod | exact only | `v1.9.1` (Go modules use exact versions) |
| nuget/msbuild | exact only | `13.0.3` (.NET uses exact versions) |

### TYPE diversity

Testdata should include both production and development dependencies:

| Manager | prod | dev |
|---------|------|-----|
| npm | `dependencies` | `devDependencies` |
| composer | `require` | `require-dev` |
| pipfile | `packages` | `dev-packages` |
| nuget | default | `developmentDependency="true"` attribute |
| msbuild | default PackageReference | `PrivateAssets="all"` attribute or `<PrivateAssets>all</PrivateAssets>` element |
| mod | all prod | N/A (no dev distinction) |
| requirements | all prod | N/A (no dev distinction) |

### MAJOR/MINOR/PATCH availability diversity

Testdata must include packages with varied update availability:

| Scenario | Example | Purpose |
|----------|---------|---------|
| All three available | MAJOR=2.0.0, MINOR=1.5.0, PATCH=1.2.4 | Test update selection flags |
| Only MAJOR available | MAJOR=2.0.0, MINOR=#N/A, PATCH=#N/A | Test `--major` flag |
| Only MINOR available | MAJOR=#N/A, MINOR=1.5.0, PATCH=#N/A | Test `--minor` flag |
| Only PATCH available | MAJOR=#N/A, MINOR=#N/A, PATCH=1.2.4 | Test `--patch` flag |
| None available (UpToDate) | MAJOR=#N/A, MINOR=#N/A, PATCH=#N/A | Test UpToDate detection |
| MAJOR+MINOR only | MAJOR=2.0.0, MINOR=1.5.0, PATCH=#N/A | Test combined flags |
| MAJOR+PATCH only | MAJOR=2.0.0, MINOR=#N/A, PATCH=1.2.4 | Test combined flags |
| MINOR+PATCH only | MAJOR=#N/A, MINOR=1.5.0, PATCH=1.2.4 | Test combined flags |

**Target distribution per manager:**
- 2-4 packages with all three (MAJOR/MINOR/PATCH)
- 2-3 packages with only MAJOR
- 2-3 packages with only MINOR
- 2-3 packages with only PATCH
- 2-4 packages UpToDate (no updates)
- Mix of MAJOR+MINOR, MAJOR+PATCH, MINOR+PATCH

### Real packages requirement

All testdata must use:
- **Real package names** from actual registries (npm, PyPI, NuGet, etc.)
- **Real version numbers** that exist in the registry
- **Realistic version ranges** that reflect common usage patterns

Never use:
- Synthetic/fake package names
- Non-existent version numbers
- Mock version catalogs or fake registries

### Fatal/error scenarios

All fatal or error-producing scenarios must be placed in `pkg/_testdata/`:

| Scenario | Location | Purpose |
|----------|----------|---------|
| Invalid JSON/YAML | `_testdata/_invalid-syntax/` | Test parse error handling |
| Invalid XML | `_testdata/_invalid-syntax/` | Test XML parse errors |
| Malformed files | `_testdata/_malformed/` | Test corrupted file handling |
| Non-existent packages | `_testdata/package-not-found/` | Test 404 error handling |
| Lock file parse errors | `_testdata/_lock-errors/` | Test lock parse failures |
| Command timeouts | `_testdata/command-timeout/` | Test timeout handling |
| Invalid commands | `_testdata/invalid-command/` | Test command not found |
| Config errors | `_testdata/_config-errors/` | Test config validation |

This separation ensures:
1. Automated tests (`pkg/testdata/`) run without critical errors
2. Manual error testing (`pkg/_testdata/`) can be done per-scenario
3. CI/CD pipelines don't fail due to intentional error fixtures

### Validating testdata diversity

Run the outdated command to verify diversity:

```bash
# Check overall diversity
./goupdate outdated -d pkg/testdata

# Verify STATUS distribution
./goupdate outdated -d pkg/testdata | grep -c "UpToDate"    # Target: 20-40%
./goupdate outdated -d pkg/testdata | grep -c "Outdated"    # Target: 60-80%

# Verify CONSTRAINT diversity (should show multiple types)
./goupdate outdated -d pkg/testdata | awk '{print $4}' | sort | uniq -c

# Verify TYPE diversity
./goupdate outdated -d pkg/testdata | awk '{print $3}' | sort | uniq -c
```

## Common troubleshooting

- **Missing dependencies:** Re-run `make init` after pulling new modules or when Go reports missing packages.
- **Race detector failures:** Investigate data races locally; do not disable `-race` as it is part of the contract for all targets.
- **Flaky paths:** Favor deterministic fixtures and avoid time-based assertions. If a flake is observed, document it in the test description and stabilize the behavior.
