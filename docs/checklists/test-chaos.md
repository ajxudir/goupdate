# Chaos Testing Checklist

Verify test coverage by deliberately breaking features.
**Parallel execution** - test multiple packages simultaneously.

---

## Quick Start

```bash
# Verify tests pass before chaos testing
go test ./... -count=1

# Run chaos test on specific package
./scripts/chaos-test.sh pkg/config/loader.go
```

---

## Methodology

1. **Select target** - Pick feature/function to break
2. **Break it** - Comment out, return early, force error
3. **Run tests** - `go test ./... -count=1`
4. **Evaluate** - Tests should FAIL
5. **Revert** - `git checkout -- <file>`

| Result | Meaning | Action |
|--------|---------|--------|
| Tests FAIL | Coverage exists | Revert, move to next target |
| Tests PASS | Coverage gap! | Add test, then revert |

---

## Phase 1: CLI Commands (Parallel by Command)

Test each command's critical path. **Run in parallel across different commands:**

### scan Command (`cmd/scan.go`)

| Target | Break Method | Test File | Status |
|--------|-------------|-----------|--------|
| File discovery | Return empty file list | `cmd/scan_test.go` | [ ] |
| Format detection | Skip format check | `cmd/scan_test.go` | [ ] |
| Output generation | Return malformed JSON | `cmd/scan_test.go` | [ ] |
| File filter (`-f`) | Ignore filter | `cmd/scan_test.go` | [ ] |
| Directory flag (`-d`) | Ignore directory | `cmd/scan_test.go` | [ ] |

### list Command (`cmd/list.go`)

| Target | Break Method | Test File | Status |
|--------|-------------|-----------|--------|
| Package listing | Return empty list | `cmd/list_test.go` | [ ] |
| Lock resolution | Skip lock file | `cmd/list_test.go` | [ ] |
| Type filter (`-t`) | Ignore type flag | `cmd/list_test.go` | [ ] |
| PM filter (`-p`) | Ignore PM flag | `cmd/list_test.go` | [ ] |
| Rule filter (`-r`) | Ignore rule flag | `cmd/list_test.go` | [ ] |
| Name filter (`-n`) | Ignore name flag | `cmd/list_test.go` | [ ] |
| Group filter (`-g`) | Ignore group flag | `cmd/list_test.go` | [ ] |

### outdated Command (`cmd/outdated.go`)

| Target | Break Method | Test File | Status |
|--------|-------------|-----------|--------|
| Version fetch | Return empty versions | `cmd/outdated_test.go` | [ ] |
| Version compare | Swap greater/less | `cmd/outdated_test.go` | [ ] |
| Major flag | Ignore `--major` | `cmd/outdated_test.go` | [ ] |
| Minor flag | Ignore `--minor` | `cmd/outdated_test.go` | [ ] |
| Patch flag | Ignore `--patch` | `cmd/outdated_test.go` | [ ] |
| Timeout | Ignore `--no-timeout` | `cmd/outdated_test.go` | [ ] |
| Preflight | Skip `--skip-preflight` | `cmd/outdated_test.go` | [ ] |

### update Command (`cmd/update.go`)

| Target | Break Method | Test File | Status |
|--------|-------------|-----------|--------|
| File modification | Skip write | `cmd/update_test.go` | [ ] |
| Dry run | Execute on dry-run | `cmd/update_test.go` | [ ] |
| Version update | Wrong version format | `cmd/update_test.go` | [ ] |
| Lock command | Skip lock execution | `cmd/update_test.go` | [ ] |
| Yes flag (`-y`) | Ignore confirmation | `cmd/update_test.go` | [ ] |
| Skip lock | Execute lock anyway | `cmd/update_test.go` | [ ] |
| Incremental | Skip incremental logic | `cmd/update_test.go` | [ ] |
| Continue on fail | Stop on first error | `cmd/update_test.go` | [ ] |

### config Command (`cmd/config.go`)

| Target | Break Method | Test File | Status |
|--------|-------------|-----------|--------|
| Show defaults | Return wrong defaults | `cmd/config_test.go` | [ ] |
| Show effective | Skip merge | `cmd/config_test.go` | [ ] |
| Init | Create wrong template | `cmd/config_test.go` | [ ] |
| Validate | Skip validation | `cmd/config_test.go` | [ ] |

---

## Phase 2: Configuration (pkg/config)

### Loader (`pkg/config/loader.go`)

| Target | Break Method | Status |
|--------|-------------|--------|
| `Load()` | Return empty config | [ ] |
| `LoadFile()` | Skip file read | [ ] |
| `Merge()` | Skip merge logic | [ ] |
| Extends resolution | Ignore extends | [ ] |
| YAML parsing | Return parse error | [ ] |

### Validation (`pkg/config/validation.go`)

| Target | Break Method | Status |
|--------|-------------|--------|
| `Validate()` | Return nil always | [ ] |
| Rule validation | Skip rule checks | [ ] |
| Field validation | Accept invalid fields | [ ] |
| Security checks | Skip path traversal check | [ ] |
| Regex complexity | Skip ReDoS check | [ ] |

### Defaults (`pkg/config/default.yml`)

| Target | Break Method | Status |
|--------|-------------|--------|
| NPM rule | Remove npm config | [ ] |
| Go mod rule | Remove mod config | [ ] |
| Composer rule | Remove composer config | [ ] |
| Default excludes | Clear exclude_versions | [ ] |

---

## Phase 3: Format Parsing (pkg/formats)

**Can test all parsers in parallel:**

### JSON Parser

| Target | Break Method | Status |
|--------|-------------|--------|
| Field extraction | Wrong JSON path | [ ] |
| Nested objects | Skip nested handling | [ ] |
| Array handling | Return single item | [ ] |

### XML Parser

| Target | Break Method | Status |
|--------|-------------|--------|
| XPath extraction | Wrong XPath | [ ] |
| Attribute extraction | Skip attributes | [ ] |
| Element extraction | Wrong element name | [ ] |
| Dev detection | Ignore PrivateAssets | [ ] |

### Raw Parser (Regex)

| Target | Break Method | Status |
|--------|-------------|--------|
| Pattern matching | Break regex | [ ] |
| Multi-pattern | Skip patterns array | [ ] |
| Named groups | Wrong group names | [ ] |
| Detection patterns | Skip detect condition | [ ] |

### YAML Parser

| Target | Break Method | Status |
|--------|-------------|--------|
| Map parsing | Return empty map | [ ] |
| List parsing | Return empty list | [ ] |
| Nested structures | Flatten structure | [ ] |

---

## Phase 4: Lock Resolution (pkg/lock)

| Target | Break Method | Status |
|--------|-------------|--------|
| `Resolve()` | Return empty versions | [ ] |
| File detection | Skip lock file search | [ ] |
| NPM lock v1 | Wrong version path | [ ] |
| NPM lock v2/v3 | Wrong package path | [ ] |
| Yarn lock | Break regex pattern | [ ] |
| Composer lock | Wrong JSON path | [ ] |
| Go sum | Break checksum parsing | [ ] |
| Command execution | Return wrong JSON | [ ] |

---

## Phase 5: Outdated Logic (pkg/outdated)

| Target | Break Method | Status |
|--------|-------------|--------|
| Version fetch | Return hardcoded version | [ ] |
| Version compare | Invert comparison | [ ] |
| Semver parsing | Skip parsing | [ ] |
| Pre-release filter | Allow pre-releases | [ ] |
| Scope filtering | Ignore major/minor/patch | [ ] |
| Command template | Wrong placeholders | [ ] |
| Timeout handling | Ignore timeout | [ ] |

---

## Phase 6: Update Logic (pkg/update)

| Target | Break Method | Status |
|--------|-------------|--------|
| Constraint update | Wrong format | [ ] |
| File write | Skip write | [ ] |
| Backup creation | Skip backup | [ ] |
| Rollback | Skip rollback on error | [ ] |
| Lock execution | Skip lock command | [ ] |
| Group handling | Ignore groups | [ ] |
| Incremental | Update all at once | [ ] |

---

## Phase 7: Filtering (pkg/filtering)

| Target | Break Method | Status |
|--------|-------------|--------|
| Type filter | Return all types | [ ] |
| PM filter | Return all PMs | [ ] |
| Rule filter | Return all rules | [ ] |
| Name filter | Case-sensitive when shouldn't | [ ] |
| Group filter | Ignore group membership | [ ] |
| File filter | Ignore glob pattern | [ ] |
| Combined filters | OR instead of AND | [ ] |

---

## Phase 8: Output (pkg/output)

| Target | Break Method | Status |
|--------|-------------|--------|
| JSON format | Invalid JSON | [ ] |
| CSV format | Wrong delimiter | [ ] |
| XML format | Invalid XML | [ ] |
| Table format | Missing columns | [ ] |
| Status icons | Wrong status codes | [ ] |
| Summary counts | Wrong totals | [ ] |

---

## Phase 9: Exit Codes

| Code | Meaning | Break Method | Status |
|------|---------|-------------|--------|
| 0 | Success | Return 1 on success | [ ] |
| 1 | Partial failure | Return 0 on partial | [ ] |
| 2 | Complete failure | Return 0 on failure | [ ] |
| 3 | Config error | Return 0 on config error | [ ] |

---

## Phase 10: Package Manager Rules

**Test each PM rule in parallel:**

| PM | Rule | Critical Path | Status |
|----|------|--------------|--------|
| npm | `npm` | package.json parsing | [ ] |
| pnpm | `pnpm` | pnpm-lock.yaml parsing | [ ] |
| yarn | `yarn` | yarn.lock parsing | [ ] |
| composer | `composer` | composer.json parsing | [ ] |
| pip | `requirements` | requirements.txt regex | [ ] |
| pipenv | `pipfile` | Pipfile parsing | [ ] |
| go | `mod` | go.mod parsing | [ ] |
| dotnet | `msbuild` | .csproj XML parsing | [ ] |
| dotnet | `nuget` | packages.config parsing | [ ] |

---

## Breakage Templates

### Comment Out Critical Code
```go
// result := doSomething()  // CHAOS: commented out
result := nil
```

### Return Early
```go
if true { // CHAOS: always return early
    return nil, nil
}
```

### Force Error
```go
return nil, fmt.Errorf("CHAOS: forced error")
```

### Invert Condition
```go
if !condition { // CHAOS: was "if condition"
```

### Skip Validation
```go
// if err := validate(); err != nil { // CHAOS: skip validation
//     return err
// }
```

### Return Empty
```go
return []Package{} // CHAOS: return empty instead of actual
```

---

## Chaos Test Script

```bash
#!/bin/bash
# chaos-test.sh <file>
set -e

FILE=$1
if [ -z "$FILE" ]; then
    echo "Usage: chaos-test.sh <file>"
    exit 1
fi

# Verify clean state
if ! git diff --quiet; then
    echo "ERROR: Uncommitted changes exist"
    exit 1
fi

# Save backup
BACKUP=$(mktemp)
cp "$FILE" "$BACKUP"

echo "=== CHAOS TEST: $FILE ==="
echo "1. Apply your breakage to $FILE"
echo "2. Press Enter to run tests"
read

# Verify change was made
if git diff --quiet "$FILE"; then
    echo "ERROR: No changes detected in $FILE"
    rm "$BACKUP"
    exit 1
fi

# Run tests
echo "Running tests..."
if go test ./... -count=1 2>&1 | grep -q "FAIL"; then
    echo ""
    echo "✅ PASS: Tests caught the breakage"
    RESULT="PASS"
else
    echo ""
    echo "❌ FAIL: Tests did NOT catch the breakage - COVERAGE GAP!"
    RESULT="FAIL"
fi

# Restore
echo ""
echo "Restoring original file..."
cp "$BACKUP" "$FILE"
rm "$BACKUP"

echo ""
echo "=== RESULT: $RESULT ==="
```

---

## Coverage Thresholds

```bash
make coverage-func
```

| Package | Target | After Chaos |
|---------|--------|-------------|
| pkg/config | 95% | [ ] |
| pkg/formats | 90% | [ ] |
| pkg/lock | 95% | [ ] |
| pkg/outdated | 90% | [ ] |
| pkg/update | 90% | [ ] |
| pkg/filtering | 95% | [ ] |
| pkg/output | 90% | [ ] |
| cmd/ | 85% | [ ] |

---

## Parallel Execution Plan

### Group A (Config & Validation)
```bash
# Terminal 1
./chaos-test.sh pkg/config/loader.go
./chaos-test.sh pkg/config/validation.go
```

### Group B (Parsers)
```bash
# Terminal 2
./chaos-test.sh pkg/formats/json.go
./chaos-test.sh pkg/formats/xml.go
./chaos-test.sh pkg/formats/raw.go
```

### Group C (Lock & Outdated)
```bash
# Terminal 3
./chaos-test.sh pkg/lock/resolver.go
./chaos-test.sh pkg/outdated/checker.go
```

### Group D (Update & Filtering)
```bash
# Terminal 4
./chaos-test.sh pkg/update/updater.go
./chaos-test.sh pkg/filtering/packages.go
```

---

## Session Log

| Date | Target | Package | Result | Gap? | Action |
|------|--------|---------|--------|------|--------|
| | | | | | |

---

## Reference

- `docs/internal/chaos-testing.md` - Detailed chaos engineering plan
- `docs/checklists/test-improvement.md` - Adding missing tests
- `docs/checklists/test-battle.md` - CLI testing
