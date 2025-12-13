# Chaos Testing Checklist

Use this checklist to verify test coverage by deliberately breaking features.

---

## Purpose

Chaos testing validates that the test suite catches breakages by:
1. Deliberately breaking a feature
2. Running tests
3. Verifying tests fail
4. Reverting the break

If tests DON'T fail when you break something, you have a coverage gap.

---

## Phase 1: Select Target

- [ ] Target feature identified
- [ ] Feature location found (file, function)
- [ ] Current tests verified passing: `go test ./...`

### Target Categories

| Category | Example Breakage | Risk |
|----------|-----------------|------|
| Config parsing | Comment out field validation | Medium |
| File parsing | Break regex pattern | Medium |
| Lock resolution | Skip lock file check | High |
| Version checking | Return wrong version | High |
| Update logic | Skip file write | High |
| CLI flags | Disable flag binding | Low |

---

## Phase 2: Break the Feature

### Breakage Methods

```go
// Method 1: Comment out critical code
// result := doSomething()
result := nil  // CHAOS

// Method 2: Return early
if true { // CHAOS: skip logic
    return nil, nil
}

// Method 3: Force error path
return nil, fmt.Errorf("CHAOS: forced error")

// Method 4: Change condition
if enabled { // Change to: if !enabled
```

- [ ] Breakage applied
- [ ] Code still compiles: `go build ./...`

---

## Phase 3: Run Tests

```bash
# Run all tests
go test ./... -count=1

# Run specific package tests
go test -v ./pkg/config/...

# Run with race detection
go test -race ./...
```

- [ ] Tests executed
- [ ] Tests FAIL as expected

### Expected Behavior

| Result | Meaning | Action |
|--------|---------|--------|
| Tests fail | Coverage exists | Revert break |
| Tests pass | Coverage gap! | Add tests first, then revert |

---

## Phase 4: Document Gap (if found)

If tests passed when they shouldn't:

- [ ] Gap documented in progress report
- [ ] New test case written
- [ ] Test fails on broken code
- [ ] Test passes on working code

### Test Template
```go
func TestFeatureX_WhenBroken_ShouldFail(t *testing.T) {
    // Arrange: Set up scenario that uses the feature

    // Act: Execute the feature

    // Assert: Verify expected behavior
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

---

## Phase 5: Revert and Verify

- [ ] Breakage reverted: `git checkout -- <file>`
- [ ] Tests pass again: `go test ./...`
- [ ] No leftover changes: `git status`

---

## Quick Chaos Test Script

```bash
#!/bin/bash
# chaos-test.sh - Quick chaos verification

FILE=$1
BACKUP=$(mktemp)

# Save original
cp "$FILE" "$BACKUP"

echo "Apply your breakage to $FILE, then press Enter"
read

# Run tests
if go test ./... -count=1 2>&1 | grep -q "FAIL"; then
    echo "PASS: Tests caught the breakage"
else
    echo "FAIL: Tests did NOT catch the breakage - coverage gap!"
fi

# Restore
cp "$BACKUP" "$FILE"
rm "$BACKUP"
```

---

## Chaos Testing Targets by Package

### pkg/config
| Target | Break Method |
|--------|-------------|
| `Validate()` | Skip validation checks |
| `Load()` | Return empty config |
| Field defaults | Change default values |

### pkg/formats
| Target | Break Method |
|--------|-------------|
| JSON parser | Break field paths |
| YAML parser | Skip nested handling |
| Raw parser | Break regex |

### pkg/lock
| Target | Break Method |
|--------|-------------|
| `Resolve()` | Return empty versions |
| Lock file read | Skip file entirely |

### pkg/outdated
| Target | Break Method |
|--------|-------------|
| Version comparison | Swap greater/less |
| Command execution | Return wrong output |

### pkg/update
| Target | Break Method |
|--------|-------------|
| File modification | Skip write |
| Constraint update | Wrong format |

### cmd/
| Target | Break Method |
|--------|-------------|
| Flag binding | Disable flag |
| Output format | Wrong template |

---

## Coverage Thresholds

After chaos testing, verify coverage:

```bash
make coverage-func
```

| Package | Minimum Coverage |
|---------|-----------------|
| pkg/config | 95% |
| pkg/formats | 90% |
| pkg/lock | 95% |
| pkg/outdated | 90% |
| pkg/update | 90% |
| cmd/ | 85% |

---

## Session Log

| Date | Target | Result | Gap Found? |
|------|--------|--------|------------|
| | | | |

---

## Reference

For detailed chaos engineering methodology, see:
- `docs/internal/chaos-testing.md` - Full chaos testing plan
- `docs/checklists/test-improvement.md` - Adding missing tests
