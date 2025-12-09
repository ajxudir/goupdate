# Error Test Data (testdata_errors)

This directory contains test data that produces errors, bad exit codes, or exceptions.
Unlike `testdata/` which contains valid configurations for automated testing, these
files are used for manual testing of error handling.

## Test Cases

### invalid-command/
Tests error handling when configured commands don't exist.
- `package.json` - Valid npm manifest
- `.goupdate.yml` - Config with non-existent command `nonexistent-command-xyz-12345`

**Expected behavior:** `goupdate outdated` should report command execution failure.

### malformed-json/
Tests error handling for invalid JSON files.
- `package.json` - JSON with missing closing brace

**Expected behavior:** `goupdate list` should report JSON parse error.

### malformed-xml/
Tests error handling for invalid XML files.
- `TestProject.csproj` - XML with unclosed PropertyGroup tag

**Expected behavior:** `goupdate list` should report XML parse error.

### command-timeout/
Tests error handling when commands exceed timeout.
- `package.json` - Valid npm manifest
- `.goupdate.yml` - Config with `sleep 60` and 2 second timeout

**Expected behavior:** `goupdate outdated` should report timeout error.

### package-not-found/npm/
Tests error handling when a package doesn't exist in the registry.
- `package.json` - NPM manifest with non-existent `missing-package`
- `package-lock.json` - Lock file referencing the missing package

**Expected behavior:** `goupdate outdated` should report package not found error (404).

## Usage

```bash
# Test invalid command handling
goupdate outdated -d ./pkg/testdata_errors/invalid-command

# Test malformed JSON handling
goupdate list -d ./pkg/testdata_errors/malformed-json

# Test malformed XML handling
goupdate list -d ./pkg/testdata_errors/malformed-xml

# Test command timeout handling
goupdate outdated -d ./pkg/testdata_errors/command-timeout

# Test package not found handling
goupdate outdated -d ./pkg/testdata_errors/package-not-found/npm
```
