# Test Data

This directory contains valid configuration files and package manifests for testing goupdate.

## Directory Structure

```
testdata/
├── composer/          # PHP Composer configs with lock files
├── groups/            # Package grouping feature tests
├── incremental/       # Incremental update feature tests
├── mod/               # Go modules with go.mod and go.sum
├── msbuild/           # C# projects with packages.lock.json
├── npm/               # Node.js manifests with package-lock.json
├── nuget/             # NuGet configs with lock files
├── pipfile/           # Python Pipfile with Pipfile.lock
└── requirements/      # Python requirements.txt
```

## Usage

### Manual Testing

You can test goupdate against any of these directories:

```bash
# List all packages in npm testdata
goupdate list -d ./pkg/testdata/npm

# Check for outdated packages in Go modules
goupdate outdated -d ./pkg/testdata/mod

# Run scan on entire testdata directory
goupdate scan -d ./pkg/testdata
```

### Automated Tests

These files are used by unit and integration tests throughout the codebase.

## Error Test Data

For intentionally invalid configurations (used for error handling tests), see:
- `../_testdata/` - Contains malformed configs, invalid syntax, and error scenarios

The separation allows you to:
1. Test the entire `testdata/` directory without hitting errors
2. Target specific error scenarios in `_testdata/` for error handling tests
