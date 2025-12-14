# Test Improvement Checklist

Use this checklist when adding or improving tests in goupdate.
**Parallel execution** - run independent test suites simultaneously.

---

## Quick Reference

### Coverage Targets
| Package | Minimum | Target | Critical Functions |
|---------|---------|--------|-------------------|
| cmd/ | 90% | 95%+ | All command handlers |
| pkg/config/ | 95% | 98%+ | Load, Validate, Merge |
| pkg/formats/ | 90% | 95%+ | All parsers |
| pkg/lock/ | 95% | 98%+ | Resolve, all lock formats |
| pkg/outdated/ | 90% | 95%+ | Check, Compare |
| pkg/update/ | 90% | 95%+ | Update, Write |
| pkg/filtering/ | 95% | 98%+ | All filter functions |
| pkg/output/ | 90% | 95%+ | All render functions |
| **Overall** | 95% | 97%+ | |

### Test File Locations
| Component | Test File |
|-----------|-----------|
| scan command | `cmd/scan_test.go` |
| list command | `cmd/list_test.go` |
| outdated command | `cmd/outdated_test.go` |
| update command | `cmd/update_test.go` |
| config command | `cmd/config_test.go` |
| Config loading | `pkg/config/loader_test.go` |
| Config validation | `pkg/config/validation_test.go` |
| JSON parsing | `pkg/formats/json_test.go` |
| XML parsing | `pkg/formats/xml_test.go` |
| Raw/Regex parsing | `pkg/formats/raw_test.go` |
| Lock resolution | `pkg/lock/*_test.go` |
| Version checking | `pkg/outdated/*_test.go` |
| Update logic | `pkg/update/*_test.go` |
| Filtering | `pkg/filtering/*_test.go` |
| Output | `pkg/output/*_test.go` |

---

## Phase 1: Assessment (Parallel)

Run simultaneously:

```bash
# Terminal 1: Current coverage
make coverage-func

# Terminal 2: Find uncovered code
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # Opens in browser

# Terminal 3: List test files
find . -name "*_test.go" | wc -l
```

| Assessment | Status |
|------------|--------|
| Current coverage checked | [ ] |
| Gaps identified | [ ] |
| Missing scenarios documented | [ ] |
| Priority areas identified | [ ] |

### Coverage Gap Analysis
| Package | Current | Target | Gap | Priority |
|---------|---------|--------|-----|----------|
| cmd/ | % | 95% | % | |
| pkg/config/ | % | 98% | % | |
| pkg/formats/ | % | 95% | % | |
| pkg/lock/ | % | 98% | % | |
| pkg/outdated/ | % | 95% | % | |
| pkg/update/ | % | 95% | % | |

---

## Phase 2: Test Types

### Unit Tests
| Scenario | Status |
|----------|--------|
| Happy path covered | [ ] |
| Error paths covered | [ ] |
| Boundary conditions | [ ] |
| Nil/empty inputs | [ ] |
| Invalid inputs | [ ] |

### Integration Tests
| Requirement | Status |
|-------------|--------|
| Real testdata used | [ ] |
| Named `Test*Integration*` | [ ] |
| File parsing verified | [ ] |
| Lock resolution verified | [ ] |
| All PMs covered | [ ] |

### Edge Case Tests
| Scenario | Status |
|----------|--------|
| Empty files | [ ] |
| Malformed input | [ ] |
| Unicode/special chars | [ ] |
| Large inputs | [ ] |
| Missing dependencies | [ ] |
| Network failures | [ ] |

### Package Manager Coverage
| PM | scan | list | outdated | update | lock |
|----|------|------|----------|--------|------|
| npm | [ ] | [ ] | [ ] | [ ] | [ ] |
| pnpm | [ ] | [ ] | [ ] | [ ] | [ ] |
| yarn | [ ] | [ ] | [ ] | [ ] | [ ] |
| composer | [ ] | [ ] | [ ] | [ ] | [ ] |
| requirements | [ ] | [ ] | [ ] | [ ] | [ ] |
| pipfile | [ ] | [ ] | [ ] | [ ] | [ ] |
| mod | [ ] | [ ] | [ ] | [ ] | [ ] |
| msbuild | [ ] | [ ] | [ ] | [ ] | [ ] |
| nuget | [ ] | [ ] | [ ] | [ ] | [ ] |

### Output Format Coverage
| Command | table | json | csv | xml |
|---------|-------|------|-----|-----|
| scan | [ ] | [ ] | [ ] | [ ] |
| list | [ ] | [ ] | [ ] | [ ] |
| outdated | [ ] | [ ] | [ ] | [ ] |
| update | [ ] | [ ] | [ ] | [ ] |

### Filter Coverage
| Filter | list | outdated | update |
|--------|------|----------|--------|
| `-t prod` | [ ] | [ ] | [ ] |
| `-t dev` | [ ] | [ ] | [ ] |
| `-p js` | [ ] | [ ] | [ ] |
| `-p golang` | [ ] | [ ] | [ ] |
| `-r npm` | [ ] | [ ] | [ ] |
| `-n name` | [ ] | [ ] | [ ] |
| `-g group` | [ ] | [ ] | [ ] |
| combined | [ ] | [ ] | [ ] |

---

## Phase 3: Test Structure

### Standard Test Template
```go
func TestFeatureName(t *testing.T) {
    // Arrange - setup
    input := "test input"
    expected := "expected output"

    // Act - execute
    result := FunctionUnderTest(input)

    // Assert - verify
    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}
```

### Table-Driven Tests
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "foo", "bar", false},
        {"empty input", "", "", true},
        {"special chars", "foo@1.0", "bar@1.0", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionUnderTest(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Subtests for Organization
```go
func TestCommand(t *testing.T) {
    t.Run("scan", func(t *testing.T) {
        // scan tests
    })
    t.Run("list", func(t *testing.T) {
        // list tests
    })
    t.Run("outdated", func(t *testing.T) {
        // outdated tests
    })
}
```

---

## Phase 4: Test Isolation (CRITICAL)

### Flag Save/Restore Pattern
```go
func TestCommand(t *testing.T) {
    // Save ALL affected flags
    origOutput := outputFlag
    origDir := dirFlag
    origRule := ruleFlag
    origType := typeFlag
    origPM := pmFlag
    origDryRun := dryRunFlag

    // Restore on cleanup (runs even if test fails)
    t.Cleanup(func() {
        outputFlag = origOutput
        dirFlag = origDir
        ruleFlag = origRule
        typeFlag = origType
        pmFlag = origPM
        dryRunFlag = origDryRun
    })

    // Set test values
    outputFlag = "json"
    dirFlag = "/test/path"

    // Run test
}
```

### Flags to Save/Restore by Command
| Command | Flags |
|---------|-------|
| scan | `scanOutputFlag`, `scanDirFlag`, `scanFileFlag` |
| list | `listOutputFlag`, `listDirFlag`, `listTypeFlag`, `listPMFlag`, `listRuleFlag`, `listNameFlag`, `listGroupFlag` |
| outdated | All list flags + `majorFlag`, `minorFlag`, `patchFlag`, `noTimeoutFlag`, `skipPreflightFlag` |
| update | All outdated flags + `dryRunFlag`, `skipLockFlag`, `yesFlag`, `incrementalFlag` |
| config | `configShowDefaults`, `configShowEffective`, `configInit`, `configValidate` |

### Temp File Cleanup
```go
func TestWithTempFiles(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "test-*")
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        os.RemoveAll(tmpDir)
    })

    // Use tmpDir for test files
}
```

---

## Phase 5: Testdata Management

### Structure
```
pkg/testdata/
├── npm/
│   ├── package.json          # Real package file
│   ├── package-lock.json     # Real lock file
│   └── _edge-cases/
│       ├── no-lock/          # No lock file scenario
│       ├── prerelease/       # Prerelease versions
│       └── unicode/          # Unicode package names
├── mod/
│   ├── go.mod
│   ├── go.sum
│   └── _edge-cases/
├── composer/
│   ├── composer.json
│   ├── composer.lock
│   └── _edge-cases/
└── README.md                  # Document testdata sources
```

### Adding Testdata
| Requirement | Status |
|-------------|--------|
| Real files (not fabricated) | [ ] |
| From actual projects | [ ] |
| Source documented | [ ] |
| Lock files included | [ ] |
| Edge cases in `_edge-cases/` | [ ] |
| No node_modules/vendor | [ ] |

---

## Phase 6: Running Tests (Parallel)

Run all test types simultaneously:

```bash
# Terminal 1: Unit tests
go test ./... -count=1

# Terminal 2: Race detection
go test -race ./...

# Terminal 3: Coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Terminal 4: Flaky test detection
go test -count=10 ./...
```

| Test Type | Command | Status |
|-----------|---------|--------|
| Unit tests | `go test ./...` | [ ] |
| Race detection | `go test -race ./...` | [ ] |
| Coverage report | `make coverage-func` | [ ] |
| Flaky detection | `go test -count=10 ./...` | [ ] |
| Integration | `make test-integration` | [ ] |

---

## Phase 7: Verification

### Coverage Verification
```bash
# Check coverage increased
make coverage-func

# Visual coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

| Check | Status |
|-------|--------|
| Coverage increased | [ ] |
| New code covered | [ ] |
| No regression | [ ] |
| Target met | [ ] |

### Test Quality
| Check | Status |
|-------|--------|
| Tests are deterministic | [ ] |
| Tests are independent | [ ] |
| Tests are fast (<1s each) | [ ] |
| Tests document behavior | [ ] |
| Test names are descriptive | [ ] |

---

## Phase 8: Documentation

| Task | Status |
|------|--------|
| Test purpose in comments | [ ] |
| Complex logic explained | [ ] |
| Testdata README updated | [ ] |
| Coverage targets updated | [ ] |

---

## Test Categories Reference

| Category | File Pattern | Purpose |
|----------|--------------|---------|
| Unit | `*_test.go` | Test individual functions |
| Integration | `*_integration_test.go` | Test with real files |
| Edge case | Uses `_edge-cases/` | Boundary conditions |
| E2E | `cmd/e2e_test.go` | Full CLI workflows |
| Benchmark | `*_benchmark_test.go` | Performance testing |
| Chaos | `*chaos*_test.go` | Deliberate failure injection |

---

## Existing Test Files Reference

### Chaos Tests (Error Injection)

| File | Lines | Purpose |
|------|-------|---------|
| `pkg/update/chaos_test.go` | 811 | Filesystem errors, rollback, concurrent |
| `pkg/outdated/chaos_versioning_test.go` | 849 | Version parsing, edge cases |
| `pkg/config/chaos_config_test.go` | 841 | Config loading/validation errors |

### Edge Case Tests

| File | Purpose |
|------|---------|
| `cmd/edge_cases_test.go` | Security, network, output edge cases |
| `cmd/context_cancellation_test.go` | Context cancellation handling |

### Integration Tests

| File | Purpose | Requires |
|------|---------|----------|
| `cmd/update_integration_test.go` | Real PM execution | go, npm |
| `cmd/output_format_integration_test.go` | All output formats | - |

### Testdata Structure

```
pkg/testdata/                           # Valid test fixtures
├── npm/, npm_v1/, npm_v2/, npm_v3/    # NPM lockfile versions
├── pnpm/, pnpm_v6-v9/                  # PNPM versions
├── yarn/, yarn_berry/                   # Yarn versions
├── composer/                            # PHP Composer
├── mod/                                 # Go modules
├── pipfile/, requirements/              # Python
├── msbuild/, nuget/                     # .NET
├── groups/, incremental/                # Special scenarios
└── */_edge-cases/                       # Edge case subdirs
    ├── no-lock/                         # Missing lock files
    └── prerelease/                      # Prerelease versions

pkg/testdata_errors/                     # Error test fixtures
├── _config-errors/                      # Config validation errors
│   ├── invalid-yaml/                    # Malformed YAML
│   ├── cyclic-extends-*/                # Circular extends
│   ├── duplicate-groups/                # Duplicate definitions
│   ├── empty-extends/, empty-rules/     # Empty values
│   ├── unknown-extends/                 # Invalid references
│   ├── type-mismatch-*/                 # Type errors
│   └── path-traversal/                  # Security test
├── _invalid-syntax/                     # Malformed manifest files
├── _malformed/                          # Structural errors
├── _lock-errors/                        # Lock file parse errors
├── _lock-missing/, _lock-not-found/     # Missing lock files
├── _lock-scenarios/                     # Multi-lock configs
├── malformed-json/                      # JSON parse errors
└── malformed-xml/                       # XML parse errors

pkg/mocksdata_errors/                    # Mock-dependent errors
├── command-timeout/                     # Timeout scenarios
├── invalid-command/                     # Bad command execution
└── package-not-found/                   # Missing packages
```

### Example Projects (`examples/`)

| Project | PM | Use Case |
|---------|-----|----------|
| go-cli | mod | Go CLI application |
| react-app | npm | React frontend |
| django-app | pip | Django backend |
| laravel-app | composer | Laravel PHP |
| kpas-api | npm | Node.js API |
| kpas-frontend | npm | Frontend SPA |
| ruby-api | bundler | Ruby API |

---

## Parallel Execution Summary

### Can Run in Parallel
- Phase 1: All assessment tasks
- Phase 6: Unit, race, coverage, flaky tests
- Testing different packages
- Testing different commands

### Must Run Sequentially
- Writing tests (one feature at a time)
- Phase 7: Verification (after tests written)

---

## Common Test Patterns

### Testing Error Paths
```go
func TestFunction_Error(t *testing.T) {
    _, err := FunctionUnderTest(invalidInput)
    if err == nil {
        t.Error("expected error, got nil")
    }
    if !strings.Contains(err.Error(), "expected message") {
        t.Errorf("unexpected error: %v", err)
    }
}
```

### Testing CLI Commands
```go
func TestScanCommand(t *testing.T) {
    // Save flags
    orig := scanDirFlag
    t.Cleanup(func() { scanDirFlag = orig })

    // Set test directory
    scanDirFlag = "testdata/npm"

    // Capture output
    var buf bytes.Buffer
    cmd := NewScanCmd()
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{})

    // Execute
    err := cmd.Execute()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    // Verify output
    output := buf.String()
    if !strings.Contains(output, "package.json") {
        t.Errorf("expected package.json in output: %s", output)
    }
}
```

### Testing JSON Output
```go
func TestListJSON(t *testing.T) {
    output := runCommand(t, "list", "-d", "testdata/npm", "-o", "json")

    var result ListResult
    if err := json.Unmarshal([]byte(output), &result); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }

    if len(result.Packages) == 0 {
        t.Error("expected packages in result")
    }
}
```
