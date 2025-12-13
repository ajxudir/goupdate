# Mock Test Data

This directory contains test data that requires **mocked commands** to function.
These tests cannot run with real package managers because they test error scenarios
that require controlled command failures.

## Why This Exists

Real integration tests use `pkg/testdata/` with real package files.
Mock-dependent error tests use this directory with controlled failures.

## Structure

```
mocksdata/
├── README.md                    # This file
└── (empty - no mock success cases yet)

mocksdata_errors/
├── invalid-command/             # Tests non-existent command handling
│   ├── package.json             # Valid npm manifest
│   └── .goupdate.yml            # Config with non-existent command
├── command-timeout/             # Tests command timeout handling
│   ├── package.json             # Valid npm manifest
│   └── .goupdate.yml            # Config with sleep command + short timeout
└── package-not-found/           # Tests registry 404 handling
    └── npm/
        ├── package.json         # Manifest with missing-package dependency
        ├── package-lock.json    # Lock file without missing-package
        └── .goupdate.yml        # Config for offline testing
```

## Running Tests

Tests using this data require mock injection or specific command configurations.
See the corresponding `*_test.go` files for how these are used.

### Manual Testing

```bash
# Test invalid command handling (will fail with "command not found")
goupdate outdated -d ./pkg/mocksdata_errors/invalid-command

# Test command timeout handling (will fail with "timeout")
goupdate outdated -d ./pkg/mocksdata_errors/command-timeout

# Test package not in lock handling
goupdate list -d ./pkg/mocksdata_errors/package-not-found/npm
```

## Difference from testdata_errors/

- `testdata_errors/` contains **real malformed files** that cause errors naturally
  - Invalid JSON/XML syntax
  - Missing lock files
  - Malformed package structures

- `mocksdata_errors/` contains **valid files** that require **mocked behaviors** to cause errors
  - Non-existent commands
  - Command timeouts
  - Registry failures
