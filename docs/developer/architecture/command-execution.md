# Command Execution Architecture

> The `cmdexec` package provides shell command execution with multiline support, piping, environment variables, and template placeholders.

## Table of Contents

- [Key Files](#key-files)
- [Execution Flow](#execution-flow)
- [Main Function](#main-function)
- [Template Replacements](#template-replacements)
- [Multiline Command Parsing](#multiline-command-parsing)
- [Shell Detection](#shell-detection)
- [Command Execution](#command-execution)
- [Pipe Parsing](#pipe-parsing)
- [Usage in Outdated](#usage-in-outdated)
- [Usage in Update](#usage-in-update)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Related Documentation](#related-documentation)

---

## Key Files

| File | Purpose |
|------|---------|
| `pkg/cmdexec/cmdexec.go` | Main execution logic |
| `pkg/outdated/exec.go` | Outdated command execution |
| `pkg/update/exec.go` | Update command execution |

## Execution Flow

```
Apply Replacements ──► Parse Command Groups ──► Execute Through Shell
```

## Main Function

**Location:** `pkg/cmdexec/cmdexec.go:43-72`

```go
var Execute ExecuteFunc = executeCommands

func executeCommands(commands string, env map[string]string, dir string, timeoutSeconds int, replacements map[string]string) ([]byte, error)
```

**Parameters:**

| Parameter | Description |
|-----------|-------------|
| `commands` | Multiline command string |
| `env` | Environment variables to set |
| `dir` | Working directory |
| `timeoutSeconds` | Command timeout (0 = no timeout) |
| `replacements` | Template placeholder values |

## Template Replacements

**Location:** `pkg/cmdexec/cmdexec.go:74-82`

```go
func applyReplacements(commands string, replacements map[string]string) string {
    result := commands
    for key, value := range replacements {
        placeholder := "{{" + key + "}}"
        result = strings.ReplaceAll(result, placeholder, value)
    }
    return result
}
```

**Standard Placeholders:**

| Placeholder | Value |
|-------------|-------|
| `{{package}}` | Package name |
| `{{version}}` | Target version |
| `{{constraint}}` | Version constraint |

**Building Replacements:**

```go
func BuildReplacements(pkg, version, constraint string) map[string]string {
    return map[string]string{
        "package":    pkg,
        "version":    version,
        "constraint": constraint,
    }
}
```

## Multiline Command Parsing

**Location:** `pkg/cmdexec/cmdexec.go:84-154`

### Supported Syntax

| Syntax | Behavior |
|--------|----------|
| Newline | Sequential execution |
| `\` at end | Line continuation |
| `\|` at end | Pipe to next line |
| Inline `\|` | Inline piping |

### Examples

**Sequential commands:**
```yaml
commands: |
  npm cache clean --force
  npm install
```

**Piped commands:**
```yaml
commands: |
  curl -s https://api.example.com |
  jq '.versions'
```

**Line continuation:**
```yaml
commands: |
  npm install --package-lock-only \
    --ignore-scripts \
    --legacy-peer-deps
```

## Shell Detection

**Location:** `pkg/cmdexec/cmdexec.go:20-37`

```go
func getShell() (shell string, args []string) {
    // Check SHELL environment variable (Unix)
    if sh := os.Getenv("SHELL"); sh != "" {
        return sh, []string{"-l", "-c"}
    }

    // Windows: use cmd.exe
    if runtime.GOOS == "windows" {
        if ps := os.Getenv("ComSpec"); ps != "" {
            return ps, []string{"/c"}
        }
        return "cmd.exe", []string{"/c"}
    }

    // Fallback
    return "sh", []string{"-c"}
}
```

**Shell Selection:**

| Platform | Shell | Args |
|----------|-------|------|
| Unix (SHELL set) | `$SHELL` | `-l -c` |
| Windows | `%ComSpec%` or `cmd.exe` | `/c` |
| Unix (fallback) | `sh` | `-c` |

The `-l` flag enables login shell mode, making aliases and shell configs available.

## Command Execution

**Location:** `pkg/cmdexec/cmdexec.go:217-287`

```go
func executePipedCommands(commands []string, env map[string]string, dir string, timeoutSeconds int) ([]byte, error)
```

**Process:**
1. Create context with timeout (if specified)
2. Build environment (inherit + custom)
3. For single command: execute directly
4. For piped commands: join with ` | ` and execute

### Environment Handling

```go
environ := os.Environ()  // Inherit current environment
for key, value := range env {
    expandedValue := os.ExpandEnv(value)  // Expand $VAR references
    environ = append(environ, fmt.Sprintf("%s=%s", key, expandedValue))
}
```

### Timeout Handling

```go
ctx := context.Background()
if timeoutSeconds > 0 {
    ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
    defer cancel()
}

// On timeout:
if ctx.Err() == context.DeadlineExceeded {
    return nil, fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
}
```

## Pipe Parsing

**Location:** `pkg/cmdexec/cmdexec.go:156-199`

```go
func splitByPipe(line string) []string
```

**Features:**
- Respects quoted strings (won't split inside quotes)
- Handles escaped quotes
- Returns clean command parts

## Usage in Outdated

**Location:** `pkg/outdated/exec.go`

```go
func ExecuteOutdatedCommand(cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
    replacements := cmdexec.BuildReplacements(pkg, version, constraint)
    return cmdexec.Execute(cfg.Commands, cfg.Env, dir, cfg.TimeoutSeconds, replacements)
}
```

## Usage in Update

**Location:** `pkg/update/exec.go`

```go
func executeUpdateCommand(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
    replacements := cmdexec.BuildReplacements(pkg, version, constraint)
    return cmdexec.Execute(cfg.Commands, cfg.Env, dir, cfg.TimeoutSeconds, replacements)
}
```

## Error Handling

```go
if err := cmd.Run(); err != nil {
    errMsg := strings.TrimSpace(stderr.String())
    if errMsg == "" {
        errMsg = strings.TrimSpace(stdout.String())
    }
    if errMsg != "" {
        return nil, fmt.Errorf("%w: %s", err, errMsg)
    }
    return nil, err
}
```

**Error priority:**
1. stderr content (if any)
2. stdout content (if stderr empty)
3. Raw error

## Testing

**Mocking:**

```go
var Execute ExecuteFunc = executeCommands

// In tests:
cmdexec.Execute = func(commands string, env map[string]string, dir string, timeout int, replacements map[string]string) ([]byte, error) {
    return []byte(`["1.0.0", "2.0.0"]`), nil
}
```

## Related Documentation

- [update.md](./update.md) - Update command execution
- [outdated.md](./outdated.md) - Outdated command execution
- [configuration.md](./configuration.md) - Command configuration
