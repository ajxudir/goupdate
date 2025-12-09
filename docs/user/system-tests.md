# System Tests

System tests in goupdate validate application health before, during, and after dependency updates. They enable automated, test-driven dependency maintenance where updates only proceed if your application continues to work correctly.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Run Modes](#run-modes)
- [CLI Flags](#cli-flags)
- [Best Practices](#best-practices)
- [Automated Maintenance with TDD](#automated-maintenance-with-tdd)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Overview

System tests integrate your existing test suites (unit tests, integration tests, e2e tests, Playwright tests, etc.) with the update process. This provides:

- **Preflight validation**: Ensure your application works before making any changes
- **Post-update validation**: Verify updates don't break functionality
- **Automatic rollback**: Revert problematic updates when tests fail
- **Confidence in automation**: Enable fully automated dependency updates in CI/CD

### When to Use System Tests

| Scenario | Recommended |
|----------|-------------|
| Production applications | Yes - validates updates before deployment |
| CI/CD pipelines | Yes - enables automated dependency maintenance |
| Libraries/packages | Optional - useful for integration tests |
| Development environments | Optional - can slow down local updates |

## Configuration

Add `system_tests` to your `.goupdate.yml` configuration file.

> **Note:** Your config must include `extends: [default]` to inherit package manager rules, or define your own rules. See [Configuration Guide](./configuration.md) for details.

```yaml
extends: [default]  # Required for package manager rules

system_tests:
  run_preflight: true      # Run tests before any updates (default: true)
  run_mode: after_all      # When to run after updates (default: after_all)
  stop_on_fail: true       # Stop updates if tests fail (default: true)

  tests:
    - name: unit-tests
      commands: npm test
      timeout_seconds: 120

    - name: e2e-tests
      commands: |
        npm run build
        npx playwright test
      timeout_seconds: 300
```

### Configuration Reference

#### Root Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `run_preflight` | bool | `true` | Run tests before any updates begin |
| `run_mode` | string | `after_all` | When to run tests after updates |
| `stop_on_fail` | bool | `true` | Stop update process if tests fail |
| `tests` | list | `[]` | List of test configurations |

#### Test Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | required | Identifier for the test |
| `commands` | string | required | Commands to execute (multiline supported) |
| `env` | map | `{}` | Environment variables for test execution |
| `timeout_seconds` | int | `300` | Maximum execution time (5 minutes default) |
| `continue_on_fail` | bool | `false` | Continue updates even if this test fails |

## Run Modes

System tests support three run modes that control when tests execute after updates:

### `after_all` (Default)

Runs all tests once after all package updates complete.

```yaml
system_tests:
  run_mode: after_all
```

**Pros:**
- Fast for many packages (tests run once)
- Efficient for CI/CD pipelines
- Good for comprehensive test suites

**Cons:**
- If tests fail, harder to identify which package caused the issue
- All packages updated before validation

**Use when:** You have many packages to update and comprehensive tests.

### `after_each`

Runs all tests after each individual package update.

```yaml
system_tests:
  run_mode: after_each
```

**Pros:**
- Immediately identifies breaking packages
- Automatic rollback of problematic updates
- Maximum safety

**Cons:**
- Slow for many packages (tests run N times)
- Higher CI/CD resource usage

**Use when:** Safety is critical and you have fast tests or few packages.

### `none`

Only runs preflight tests (if enabled), skips post-update tests.

```yaml
system_tests:
  run_mode: none
  run_preflight: true  # Still validates before updates
```

**Use when:** You want quick updates with manual testing afterward.

### Performance Comparison

| Packages | Test Time | `after_each` | `after_all` |
|----------|-----------|--------------|-------------|
| 10 | 2 min | 20 min | 2 min |
| 50 | 2 min | 100 min | 2 min |
| 100 | 2 min | 200 min | 2 min |

## CLI Flags

Override system test behavior from the command line:

```bash
# Skip all system tests
goupdate update --skip-system-tests

# Override run mode
goupdate update --system-test-mode after_each
goupdate update --system-test-mode after_all
goupdate update --system-test-mode none

# Combine with other flags
goupdate update --dry-run  # System tests skipped in dry-run mode
```

## Best Practices

### 1. Start with Fast Tests

Configure quick smoke tests first, add comprehensive tests later:

```yaml
system_tests:
  tests:
    # Fast smoke test - catches obvious breaks
    - name: smoke
      commands: npm run test:smoke
      timeout_seconds: 60

    # Comprehensive test - thorough validation
    - name: full-suite
      commands: npm run test:all
      timeout_seconds: 600
```

### 2. Use `continue_on_fail` for Non-Critical Tests

Mark optional tests to avoid blocking updates:

```yaml
system_tests:
  tests:
    - name: critical-tests
      commands: npm test
      continue_on_fail: false  # Must pass (default)

    - name: performance-tests
      commands: npm run test:perf
      continue_on_fail: true   # Optional - warn but continue
```

### 3. Set Appropriate Timeouts

Avoid hanging tests blocking your pipeline:

```yaml
system_tests:
  tests:
    - name: unit-tests
      commands: npm test
      timeout_seconds: 120     # 2 minutes for unit tests

    - name: e2e-tests
      commands: npx playwright test
      timeout_seconds: 600     # 10 minutes for e2e tests
```

### 4. Use Environment Variables for CI/CD

Configure test behavior based on environment:

```yaml
system_tests:
  tests:
    - name: e2e-tests
      commands: npx playwright test
      env:
        CI: "true"
        PLAYWRIGHT_BROWSERS_PATH: "/cache/browsers"
        TEST_BASE_URL: "http://localhost:3000"
```

## Automated Maintenance with TDD

System tests enable fully automated dependency maintenance when combined with comprehensive test coverage. This approach eliminates manual intervention for routine updates while ensuring application stability.

### The Automation Pipeline

```
AUTOMATED UPDATE PIPELINE
═══════════════════════════════════════════════════════════════════

1. CI/CD triggers goupdate (daily/weekly cron)
            ↓
2. Preflight tests verify app health
            ↓
3. Updates applied with validation
            ↓
4. System tests validate all changes
            ↓
   Tests Pass          Tests Fail
       ↓                   ↓
   Auto-merge          Create Issue
   Deploy              Alert Team
```

### Benefits of TDD-Based Updates

| Benefit | Description |
|---------|-------------|
| **Zero manual intervention** | Updates proceed automatically when tests pass |
| **Immediate detection** | Breaking changes caught before deployment |
| **Automatic rollback** | Problematic updates reverted automatically |
| **Developer focus** | Team only intervenes when tests fail |
| **Continuous security** | Security patches applied promptly |
| **Audit trail** | All updates tracked through version control |

### CI/CD Integration Example

**GitHub Actions:**

```yaml
name: Automated Dependency Updates

on:
  schedule:
    - cron: '0 3 * * 1'  # Weekly on Monday 3am
  workflow_dispatch:      # Manual trigger

jobs:
  update-dependencies:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run goupdate
        run: |
          goupdate update --yes --output json > update-result.json

      - name: Create PR if updates
        if: success()
        run: |
          if [ -s update-result.json ]; then
            git checkout -b deps/automated-update-$(date +%Y%m%d)
            git add .
            git commit -m "chore(deps): automated dependency updates"
            gh pr create --title "Automated Dependency Updates" \
              --body "Updates validated by system tests"
          fi
```

### Test Coverage Requirements

For reliable automated updates, ensure your tests cover:

| Area | Coverage Required |
|------|-------------------|
| Core functionality | High - business logic must work |
| API endpoints | High - external interfaces validated |
| UI components | Medium - major user flows tested |
| Database operations | High - data integrity verified |
| External integrations | Medium - mocked or integration tests |

> **Building coverage over time:** You don't need perfect coverage on day one. When an issue slips through, add a test for it. Each regression test permanently prevents that class of problem from recurring. Over time, your suite grows to catch an ever-wider range of issues automatically - turning past problems into future protection instead of repeated firefighting.

### Incremental Adoption

Start with manual oversight and gradually increase automation:

1. **Phase 1: Preflight Only**
   ```yaml
   system_tests:
     run_preflight: true
     run_mode: none  # Manual testing after updates
   ```

2. **Phase 2: Validation Mode**
   ```yaml
   system_tests:
     run_preflight: true
     run_mode: after_all
     stop_on_fail: true
   ```

3. **Phase 3: Full Automation**
   - Comprehensive test suite
   - CI/CD pipeline integration
   - Automatic PR creation
   - Auto-merge on success

## Examples

### Node.js Project (npm)

```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: lint
      commands: npm run lint
      timeout_seconds: 60
      continue_on_fail: true  # Lint issues don't block updates

    - name: unit-tests
      commands: npm test
      timeout_seconds: 180

    - name: build
      commands: npm run build
      timeout_seconds: 300

    - name: e2e
      commands: |
        npm run start:test &
        sleep 5
        npx playwright test
      timeout_seconds: 600
      env:
        CI: "true"
```

### Python Project (pytest)

```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: unit-tests
      commands: pytest tests/unit -v
      timeout_seconds: 120

    - name: integration-tests
      commands: pytest tests/integration -v
      timeout_seconds: 300
      env:
        DATABASE_URL: "sqlite:///test.db"
```

### Go Project

```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: unit-tests
      commands: go test ./...
      timeout_seconds: 180

    - name: build
      commands: go build ./cmd/...
      timeout_seconds: 120
```

### Full-Stack Application

```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: backend-tests
      commands: |
        cd backend
        npm test
      timeout_seconds: 180

    - name: frontend-tests
      commands: |
        cd frontend
        npm test
      timeout_seconds: 180

    - name: e2e-tests
      commands: |
        docker-compose up -d
        sleep 10
        npm run test:e2e
        docker-compose down
      timeout_seconds: 600
      env:
        E2E_BASE_URL: "http://localhost:3000"
```

### Monorepo with Multiple Packages

```yaml
system_tests:
  run_preflight: true
  run_mode: after_all
  tests:
    - name: workspace-tests
      commands: npm test --workspaces
      timeout_seconds: 300

    - name: build-all
      commands: npm run build --workspaces
      timeout_seconds: 600
```

## Troubleshooting

### Tests Timeout

**Symptoms:** Tests exceed `timeout_seconds` and fail.

**Solutions:**
1. Increase timeout:
   ```yaml
   tests:
     - name: slow-tests
       timeout_seconds: 900  # 15 minutes
   ```
2. Use `--no-timeout` flag for debugging:
   ```bash
   goupdate update --no-timeout
   ```
3. Optimize test suite for faster execution

### Tests Fail in CI but Pass Locally

**Common causes:**
- Missing environment variables
- Different Node/Python/Go versions
- Browser dependencies not installed

**Solutions:**
1. Set required environment variables:
   ```yaml
   tests:
     - name: e2e
       env:
         CI: "true"
         DISPLAY: ":99"
   ```
2. Install browser dependencies in CI:
   ```bash
   npx playwright install --with-deps
   ```

### Cannot Identify Breaking Package

**When using `after_all` mode:**

Switch to `after_each` mode to identify the specific package:
```bash
goupdate update --system-test-mode after_each
```

Or update packages incrementally:
```bash
goupdate update --name specific-package
```

### Tests Pass But Application Broken

**Possible causes:**
- Test coverage gaps
- Tests not running in production-like environment
- Missing integration tests

**Solutions:**
1. Add more comprehensive tests
2. Use production-like test environment
3. Add smoke tests for critical paths

### Skip System Tests Temporarily

For emergency updates or debugging:
```bash
# Skip all system tests
goupdate update --skip-system-tests

# Or use dry-run to preview (skips tests automatically)
goupdate update --dry-run
```

## Related Documentation

- [CLI Reference](./cli.md) - All command flags
- [Configuration Guide](./configuration.md) - Full config options
- [Features Overview](./features.md) - All goupdate features
