# Test Improvement Checklist

Use this checklist when adding or improving tests in goupdate.

---

## Phase 1: Assessment

- [ ] Current coverage checked: `make coverage-func`
- [ ] Gaps identified in coverage report
- [ ] Missing test scenarios documented
- [ ] Priority areas identified

### Coverage Targets
| Package | Minimum | Target |
|---------|---------|--------|
| cmd/ | 90% | 95%+ |
| pkg/config/ | 95% | 98%+ |
| pkg/formats/ | 90% | 95%+ |
| pkg/update/ | 90% | 95%+ |
| pkg/outdated/ | 90% | 95%+ |
| Overall | 95% | 97%+ |

---

## Phase 2: Test Types

### Unit Tests
- [ ] Happy path covered
- [ ] Error paths covered
- [ ] Boundary conditions tested
- [ ] Nil/empty inputs handled

### Integration Tests
- [ ] Real testdata used (no mocks)
- [ ] Test named `Test*Integration*`
- [ ] File parsing verified
- [ ] Lock resolution verified

### Chaos Tests
- [ ] Malformed input tested
- [ ] Unicode/special characters tested
- [ ] Large inputs tested
- [ ] Empty/null values tested

### Edge Case Tests
- [ ] `pkg/testdata/*/_edge-cases/` used
- [ ] No-lock scenarios tested
- [ ] Prerelease versions tested
- [ ] Platform-specific cases (if applicable)

---

## Phase 3: Test Quality

### Test Structure
```go
func TestFeatureName(t *testing.T) {
    // Arrange - setup

    // Act - execute

    // Assert - verify
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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Test Isolation
- [ ] Tests don't depend on order
- [ ] Package-level flags saved/restored with `t.Cleanup()`
- [ ] Temp files cleaned up
- [ ] No global state leakage

---

## Phase 4: Testdata

### Adding Testdata
- [ ] Real package files (not fabricated)
- [ ] Placed in `pkg/testdata/<ecosystem>/`
- [ ] Lock files included
- [ ] Edge cases in `_edge-cases/` subdirectory

### Testdata Structure
```
pkg/testdata/
├── npm/
│   ├── package.json
│   ├── package-lock.json
│   ├── .goupdate.yml
│   └── _edge-cases/
│       ├── no-lock/
│       └── prerelease/
├── mod/
│   ├── go.mod
│   ├── go.sum
│   └── _edge-cases/
└── README.md
```

---

## Phase 5: Test Pollution Prevention

### Flag Save/Restore Pattern
```go
func TestSomething(t *testing.T) {
    // Save original values
    originalFlag := someFlag
    originalOutput := outputFlag

    // Set test values
    someFlag = "test"
    outputFlag = "json"

    // Restore on cleanup
    t.Cleanup(func() {
        someFlag = originalFlag
        outputFlag = originalOutput
    })

    // Run test
}
```

### Common Flags to Save
- [ ] `updateOutputFlag`
- [ ] `updateRuleFlag`
- [ ] `scanOutputFlag`
- [ ] `updateDirFlag`
- [ ] `updateDryRunFlag`

---

## Phase 6: Verification

### Run Tests
- [ ] `go test ./...` - All pass
- [ ] `go test -race ./...` - No races
- [ ] `go test -count=10 ./...` - Consistent (no flakes)

### Coverage Check
- [ ] `make coverage-func` - Coverage increased
- [ ] New code covered
- [ ] No coverage regression

---

## Phase 7: Documentation

- [ ] Test purpose documented in function comment
- [ ] Complex test logic explained
- [ ] Testdata README updated if new files added

---

## Test Categories Reference

| Category | Location | Purpose |
|----------|----------|---------|
| Unit | `*_test.go` | Test individual functions |
| Integration | `*_integration_test.go` | Test with real files |
| Chaos | `chaos_*_test.go` | Test malformed inputs |
| Edge cases | Uses `_edge-cases/` testdata | Test boundary conditions |
| E2E | `cmd/e2e_test.go` | Test full CLI workflows |
| Benchmark | `*_benchmark_test.go` | Performance testing |
