# Troubleshooting Guide

This guide covers common issues and their solutions when using goupdate.

## Quick Diagnostics

Before diving into specific issues, try these diagnostic steps:

```bash
# Check goupdate version
goupdate version

# Validate configuration
goupdate config --validate

# Run with verbose output
goupdate list --verbose

# Test a dry run
goupdate update --dry-run
```

---

## Configuration Errors

### "Configuration validation failed"

**Symptom**: Error message about invalid configuration

**Common Causes**:
1. Invalid YAML syntax
2. Unknown fields in configuration
3. Missing required fields
4. Typos in field names

**Solutions**:

```bash
# Validate your configuration
goupdate config --validate

# Check for common typos (goupdate will suggest corrections)
goupdate config --validate --verbose
```

**Example Fixes**:
```yaml
# Wrong: 'excludes' is not a valid field
excludes:
  - node_modules

# Correct: use 'exclude'
exclude:
  - node_modules
```

### "Unknown rule: xyz"

**Symptom**: goupdate doesn't recognize your rule name

**Solutions**:
1. Check that the rule is defined in your `.goupdate.yml`
2. If using `extends`, ensure the parent config defines the rule
3. Check for typos in rule names

```yaml
# Ensure rule is defined
rules:
  npm:  # This is the rule name
    manager: npm
    include:
      - "**/package.json"
```

### "Invalid extends reference"

**Symptom**: Error when loading config that uses `extends`

**Solutions**:
1. Check that the referenced file exists
2. Use relative paths from the config file location
3. Ensure the parent config is valid

```yaml
# Relative to current config file
extends:
  - ./base-config.yml

# Or absolute path
extends:
  - /path/to/base-config.yml
```

---

## Command Execution Errors

### "Command validation failed"

**Symptom**: Preflight checks fail for update/outdated commands

**Common Causes**:
1. Command not installed
2. Command not in PATH
3. Missing shell configuration

**Solutions**:

```bash
# Check if command is available
which npm  # or composer, pip, etc.

# Test the command directly
npm --version

# Skip preflight checks (use with caution)
goupdate update --skip-preflight
```

### "Command timed out"

**Symptom**: Commands take too long and get killed

**Solutions**:
1. Increase timeout in configuration:
   ```yaml
   rules:
     npm:
       outdated:
         timeout_seconds: 120  # Default is 60
   ```
2. Use `--no-timeout` flag for debugging:
   ```bash
   goupdate outdated --no-timeout
   ```
3. Check network connectivity

### "Lock command failed"

**Symptom**: Update fails during lock file generation

**Common Causes**:
1. Missing dependencies
2. Version conflicts
3. Network issues
4. Insufficient permissions

**Solutions**:

```bash
# Try running the lock command manually
npm install  # or composer update, pip install, etc.

# Check for version conflicts
npm ls

# Skip lock file generation
goupdate update --skip-lock
```

---

## Lock File Errors

### "Lock file not found"

**Symptom**: Packages show as "Lock Missing" status

**Solutions**:
1. Generate the lock file:
   ```bash
   npm install   # Creates package-lock.json
   composer install  # Creates composer.lock
   ```
2. Check lock file patterns in config:
   ```yaml
   rules:
     npm:
       lock_files:
         - files:
             - "**/package-lock.json"
   ```

### "Failed to parse lock file"

**Symptom**: Error parsing lock file content

**Common Causes**:
1. Corrupted lock file
2. Incompatible lock file version
3. Wrong extraction pattern

**Solutions**:

```bash
# Regenerate lock file
rm package-lock.json
npm install

# Validate lock file syntax
cat package-lock.json | jq .  # For JSON lock files
```

### "Package not in lock file"

**Symptom**: Package shows "Not In Lock" status

**Common Causes**:
1. Package was added but `npm install` not run
2. Package is a peer dependency
3. Package name mismatch

**Solutions**:
```bash
# Install dependencies to update lock file
npm install

# Check if package is in lock file
grep "package-name" package-lock.json
```

---

## Version Detection Errors

### "No versions found"

**Symptom**: Outdated check returns no available versions

**Common Causes**:
1. Package doesn't exist in registry
2. Authentication required for private packages
3. Network issues
4. Registry is down

**Solutions**:

```bash
# Test with the native package manager
npm view package-name versions

# Check authentication
npm whoami

# Try a different registry
npm view package-name versions --registry https://registry.npmjs.org/
```

### "Version mismatch after update"

**Symptom**: Update reports success but version didn't change

**Common Causes**:
1. Lock file wasn't updated
2. Version constraint prevented update
3. Update command succeeded but didn't apply

**Solutions**:

```bash
# Run with verbose mode to see what happened
goupdate update package-name --verbose

# Check constraint type
goupdate list package-name  # Shows constraint column

# Force update by modifying constraint in manifest
```

### "Floating constraint cannot be updated"

**Symptom**: Package shows "Floating" status

**Cause**: Package has a floating constraint like `5.*`, `>=1.0.0`, or `[8.0,9.0)`

**Solutions**:
1. Update the constraint manually to an exact version
2. Use self-pinning for requirements.txt style files:
   ```yaml
   rules:
     pip:
       self_pinning: true
   ```

---

## Rollback Errors

### "Rollback failed"

**Symptom**: Update failed and rollback also failed

**Common Causes**:
1. File permissions changed
2. Disk full
3. File was modified externally

**Solutions**:

```bash
# Check git status for changes
git status

# Restore from git
git checkout -- package.json package-lock.json

# Check disk space
df -h
```

### "Partial update state"

**Symptom**: Some packages updated, some didn't

**Solutions**:
1. Check `--continue-on-fail` flag behavior
2. Review failures with verbose output
3. Consider using groups for atomic updates:
   ```yaml
   groups:
     core:
       packages:
         - lodash
         - express
   ```

---

## Output Format Errors

### "Invalid output format"

**Symptom**: Error when using `--output` flag

**Solutions**:
Use a valid format: `json`, `csv`, `xml`, or `table` (default)

```bash
goupdate list --output json
goupdate outdated --output csv
```

### "JSON parsing error"

**Symptom**: JSON output is invalid

**Solutions**:
1. Ensure no verbose output mixed with JSON:
   ```bash
   goupdate list --output json 2>/dev/null
   ```
2. Check for warning messages in stderr

---

## System Test Errors

### "System tests failed"

**Symptom**: Tests fail after update

**Common Causes**:
1. Breaking changes in updated package
2. Test timeout
3. Missing test dependencies

**Solutions**:

```bash
# Run tests manually
npm test

# Increase test timeout
# In .goupdate.yml:
system_tests:
  tests:
    - name: "Unit Tests"
      commands: npm test
      timeout_seconds: 300  # 5 minutes

# Mark non-critical tests
system_tests:
  tests:
    - name: "Lint"
      commands: npm run lint
      continue_on_fail: true  # Don't rollback on lint failure
```

### "Test timeout"

**Symptom**: System tests are killed before completion

**Solutions**:
1. Increase timeout in config
2. Use `--no-timeout` for debugging
3. Optimize slow tests

---

## Performance Issues

### "Slow outdated checks"

**Symptom**: `goupdate outdated` takes a long time

**Solutions**:
1. Use filters to check fewer packages:
   ```bash
   goupdate outdated -p npm --type prod
   ```
2. Increase parallelism (if supported by registry)
3. Use local caching when available

### "High memory usage"

**Symptom**: goupdate uses too much memory

**Solutions**:
1. Process fewer packages at once
2. Use structured output to avoid buffering:
   ```bash
   goupdate list --output json | head -100
   ```

---

## Platform-Specific Issues

### Windows: "Command not found"

**Symptom**: Commands fail on Windows

**Solutions**:
1. Use full path to executables
2. Configure shell in environment:
   ```yaml
   rules:
     npm:
       update:
         env:
           SHELL: cmd.exe
   ```

### macOS: "Permission denied"

**Symptom**: Cannot write to node_modules or similar

**Solutions**:
```bash
# Fix npm permissions
sudo chown -R $(whoami) ~/.npm
sudo chown -R $(whoami) /usr/local/lib/node_modules
```

### Linux: "EACCES: permission denied"

**Symptom**: Cannot install packages globally

**Solutions**:
1. Use local installation
2. Configure npm prefix
3. Use nvm for Node.js version management

---

## Getting Help

### Verbose Output

Always run with `--verbose` when troubleshooting:

```bash
goupdate update --verbose 2>&1 | tee goupdate.log
```

### Reporting Issues

When reporting issues, include:

1. goupdate version (`goupdate version`)
2. Operating system and version
3. Package manager versions
4. Configuration file (redact sensitive info)
5. Full error output with `--verbose`
6. Steps to reproduce

File issues at: https://github.com/anthropics/claude-code/issues

### Debug Mode

For deep debugging:

```bash
# Maximum verbosity
DEBUG=1 goupdate update --verbose --no-timeout
```

---

## Common Error Messages Reference

| Error | Cause | Solution |
|-------|-------|----------|
| `configuration validation failed` | Invalid YAML | Run `goupdate config --validate` |
| `command timed out` | Slow network/command | Increase `timeout_seconds` |
| `lock file not found` | Missing lock file | Run package manager install |
| `failed to parse lock file` | Corrupted/incompatible lock | Regenerate lock file |
| `no versions found` | Registry issues | Check network/auth |
| `rollback failed` | Permission/disk issues | Check permissions, disk space |
| `floating constraint` | Non-exact version | Update constraint manually |
| `package not in lock` | Not installed | Run package manager install |
| `unsupported format` | Missing config | Add format to rule config |

## See Also

- [Configuration Reference](./configuration.md)
- [Private Registries](./private-registries.md)
- [CLI Reference](./cli.md)
