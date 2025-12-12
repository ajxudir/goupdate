# Error Test Data (testdata_errors)

This directory contains test data with **real malformed files** that produce errors
naturally without requiring command mocking.

Unlike `testdata/` which contains valid configurations for automated testing, these
files contain intentionally broken or malformed data for testing error handling.

> **Note:** Mock-dependent error scenarios (invalid commands, timeouts, registry failures)
> have been moved to `mocksdata_errors/`. See `../mocksdata/README.md`.

## Directory Structure

### _config-errors/
Tests config validation errors.
- `invalid-yaml/` - Malformed YAML syntax
- `duplicate-groups/` - Duplicate group definitions
- `unknown-extends/` - Invalid extends references

### _invalid-syntax/
Tests manifest parsing errors for malformed syntax.
- `npm/`, `pnpm/`, `yarn/` - Invalid JSON syntax
- `composer/` - Invalid JSON syntax
- `mod/` - Invalid go.mod syntax
- `requirements/`, `pipfile/` - Invalid Python syntax
- `msbuild/`, `nuget/` - Invalid XML syntax

### _malformed/
Tests structurally broken manifest files.
- Contains valid syntax but invalid structure for each PM

### _lock-errors/
Tests lock file parsing errors.
- Broken lock files for npm, pnpm, yarn, composer, mod

### _lock-missing/
Tests behavior when lock file is completely missing.
- npm project with no package-lock.json

### _lock-not-found/
Tests behavior when lock file doesn't exist.
- npm and mod projects without lock files

### _lock-scenarios/
Tests multi-lock configuration scenarios.
- Config files that reference multiple or non-existent lock files

### malformed-json/
Tests JSON parse errors.
- `package.json` with missing closing brace

### malformed-xml/
Tests XML parse errors.
- `TestProject.csproj` with unclosed tags

## Usage

```bash
# Test malformed JSON handling
goupdate list -d ./pkg/testdata_errors/malformed-json

# Test malformed XML handling
goupdate list -d ./pkg/testdata_errors/malformed-xml

# Test lock file missing
goupdate list -d ./pkg/testdata_errors/_lock-missing/npm

# Test invalid syntax
goupdate list -d ./pkg/testdata_errors/_invalid-syntax/npm
```

## See Also

- `../testdata/` - Valid test data for integration tests
- `../mocksdata_errors/` - Mock-dependent error scenarios
