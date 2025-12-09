# System Tests Architecture

> System tests provide automated validation of application health before, during, and after dependency updates. This ensures that updates don't break critical functionality.

## Overview

System tests are shell commands configured in `.goupdate.yml` that verify the application works correctly. They can run at three points during the update process:

1. **Preflight** - Before any updates begin (validates environment)
2. **After Each** - After each package update (validates incremental changes)
3. **After All** - After all updates complete (final validation)

## Architecture Diagram

```
UPDATE FLOW WITH SYSTEM TESTS
═══════════════════════════════════════════════════════════════════════════

                    ┌─────────────────┐
                    │  Configuration  │
                    │   Loading       │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Preflight     │──────► Stop if critical failure
                    │   Tests         │
                    └────────┬────────┘
                             │
                             ▼
              ┌──────────────────────────────┐
              │    For Each Package Update   │
              │  ┌─────────────────────────┐ │
              │  │ 1. Backup files         │ │
              │  │ 2. Update manifest      │ │
              │  │ 3. Run lock command     │ │
              │  │ 4. Run after_each tests │──────► Rollback if failed
              │  │ 5. Continue or stop     │ │
              │  └─────────────────────────┘ │
              └──────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   After All     │
                    │   Validation    │
                    └─────────────────┘
```

## Key Components

### Runner (`pkg/systemtest/systemtest.go`)

The `Runner` struct is the main orchestrator for system test execution:

```go
type Runner struct {
    cfg       *config.SystemTestsCfg  // Test configuration
    workDir   string                   // Working directory for commands
    noTimeout bool                     // Disable timeouts (for debugging)
    verbose   bool                     // Enable verbose output
}
```

**Key Methods:**
- `NewRunner()` - Creates a runner with configuration
- `HasTests()` - Checks if tests are configured
- `ShouldRunPreflight()` - Checks if preflight is enabled
- `ShouldRunAfterEach()` - Checks run_mode setting
- `RunPreflight()` - Executes preflight tests
- `RunAfterUpdate()` - Executes after-each tests
- `RunValidation()` - Executes after-all tests

### Result (`pkg/systemtest/result.go`)

Test results are captured in structured types:

```go
type TestResult struct {
    Name           string        // Test identifier
    Passed         bool          // Pass/fail status
    Duration       time.Duration // Execution time
    Error          error         // Error message if failed
    Output         string        // Command output (stdout/stderr)
    ContinueOnFail bool          // Whether to continue on failure
}

type Result struct {
    Tests         []TestResult   // Individual test results
    Phase         string         // When tests ran (Preflight/After Update/Validation)
    TotalDuration time.Duration  // Total execution time
}
```

**Key Methods:**
- `Passed()` - Returns true if all tests passed
- `HasCriticalFailure()` - Checks for failures that should stop updates
- `CriticalFailures()` - Returns tests that failed and are critical
- `Summary()` - Returns human-readable summary
- `FormatResults()` - Returns formatted output for display

### Configuration (`pkg/config/model.go`)

System tests are configured in the `system_tests` section:

```go
type SystemTestsCfg struct {
    RunPreflight *bool            `yaml:"run_preflight,omitempty"`
    RunMode      string           `yaml:"run_mode,omitempty"`
    StopOnFail   *bool            `yaml:"stop_on_fail,omitempty"`
    Tests        []SystemTestCfg  `yaml:"tests"`
}

type SystemTestCfg struct {
    Name           string            `yaml:"name"`
    Commands       string            `yaml:"commands"`
    Env            map[string]string `yaml:"env,omitempty"`
    TimeoutSeconds int               `yaml:"timeout_seconds,omitempty"`
    ContinueOnFail bool              `yaml:"continue_on_fail,omitempty"`
}
```

## Configuration Options

### Run Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `after_each` | Run tests after each package update | Catch issues early, safer for critical systems |
| `after_all` | Run tests once after all updates | Faster, suitable for comprehensive test suites |

### Behavior Flags

| Flag | Default | Description |
|------|---------|-------------|
| `run_preflight` | `true` | Run tests before starting updates |
| `stop_on_fail` | `true` | Stop all updates if a test fails |
| `continue_on_fail` | `false` | Per-test flag to continue despite failure |

### Timeouts

- Default timeout: 300 seconds (5 minutes)
- Per-test timeout: Configurable via `timeout_seconds`
- Global disable: `--no-timeout` CLI flag

## Data Flow

### 1. Configuration Loading

```
.goupdate.yml
     │
     ▼
config.LoadConfig()
     │
     ▼
config.SystemTestsCfg parsed
     │
     ▼
systemtest.NewRunner() created
```

### 2. Test Execution

```
Runner.runTests(phase)
     │
     ▼
For each test in cfg.Tests:
     │
     ├──► runSingleTest(test)
     │         │
     │         ▼
     │    cmdexec.Execute(commands, env, workDir, timeout, nil)
     │         │
     │         ▼
     │    TestResult created with output/error/duration
     │
     ▼
Result aggregated with all TestResults
```

### 3. Integration with Update Command

```go
// In cmd/update.go - simplified flow:

// 1. Create runner
testRunner := systemtest.NewRunner(cfg.SystemTests, workDir, noTimeout, verbose)

// 2. Run preflight if configured
if testRunner.ShouldRunPreflight() {
    result := testRunner.RunPreflight()
    if result.HasCriticalFailure() {
        return ErrPreflightFailed
    }
}

// 3. After each package update (if run_mode == "after_each")
for _, pkg := range packages {
    err := update.UpdatePackage(pkg, target, cfg, workDir, dryRun, false)
    if err != nil {
        rollback()
        continue
    }

    if testRunner.ShouldRunAfterEach() {
        result := testRunner.RunAfterUpdate()
        if result.HasCriticalFailure() {
            rollback()
            if testRunner.StopOnFail() {
                return ErrTestsFailed
            }
        }
    }
}

// 4. Final validation (if run_mode == "after_all")
if testRunner.ShouldRunAfterAll() {
    result := testRunner.RunValidation()
    // Report results
}
```

## Error Handling

### Test Failures

When a test fails:

1. **Critical failure** (`continue_on_fail: false`):
   - Current package update is rolled back
   - If `stop_on_fail: true`, entire update process stops
   - Exit code indicates failure

2. **Non-critical failure** (`continue_on_fail: true`):
   - Warning is logged
   - Update process continues
   - Final summary shows failure count

### Rollback on Failure

```
Test Failure Detected
        │
        ▼
Is ContinueOnFail?
        │
   No ──┴── Yes
   │         │
   ▼         ▼
Rollback   Log warning
manifest   and continue
and lock
files
```

### Timeout Handling

Commands that exceed their timeout are terminated:

1. Context deadline exceeded
2. Process group killed (prevents orphaned children)
3. Timeout error returned
4. Test marked as failed

## Example Configuration

```yaml
system_tests:
  run_preflight: true
  run_mode: after_each
  stop_on_fail: true
  tests:
    - name: "Unit Tests"
      commands: |
        npm test
      timeout_seconds: 300

    - name: "Type Check"
      commands: |
        npm run typecheck
      timeout_seconds: 60

    - name: "Lint (non-critical)"
      commands: |
        npm run lint
      continue_on_fail: true
      timeout_seconds: 60

    - name: "Build"
      commands: |
        npm run build
      timeout_seconds: 180
```

## Best Practices

### Test Selection

1. **Preflight tests**: Fast checks that validate environment
   - Compiler/interpreter available
   - Required services running
   - Permissions correct

2. **After-each tests**: Core functionality tests
   - Unit tests
   - Type checking
   - Critical integration tests

3. **After-all tests**: Comprehensive validation
   - Full test suite
   - E2E tests
   - Build verification

### Performance Considerations

- **Timeout appropriately**: Set realistic timeouts per test
- **Use `after_all` for slow suites**: Full test runs can be slow
- **Mark non-critical tests**: Use `continue_on_fail` for linting, etc.
- **Parallelize where possible**: Structure commands to run in parallel

### Debugging

1. **Enable verbose mode**: `--verbose` shows command output
2. **Disable timeouts**: `--no-timeout` for debugging hangs
3. **Run tests manually**: Copy commands and run outside goupdate
4. **Check working directory**: Commands run in `working_dir` context

## Testing the System Tests Feature

### Unit Tests

Located in `pkg/systemtest/systemtest_test.go`:

- Test runner creation
- Test execution flow
- Result aggregation
- Phase handling

### Integration Tests

Test system tests alongside update command:

```go
func TestUpdateWithSystemTests(t *testing.T) {
    cfg := &config.Config{
        SystemTests: &config.SystemTestsCfg{
            Tests: []config.SystemTestCfg{
                {Name: "test", Commands: "echo 'pass'"},
            },
        },
    }
    // ... verify tests run and results captured
}
```

## Related Documentation

- [Update Command](./update.md) - How updates trigger system tests
- [Command Execution](./command-execution.md) - How test commands are executed
- [Configuration](./configuration.md) - Full configuration reference

---

*Last updated: 2025-12-07*
