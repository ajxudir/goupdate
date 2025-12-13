# Task: Chaos Testing for Config Validation
**Agent:** Claude
**Date:** 2025-12-13
**Status:** Completed

## Objective
Create comprehensive chaos tests for configuration validation to ensure the config system handles malicious, malformed, and edge-case inputs gracefully without panics, memory exhaustion, or security bypasses.

## Progress
- [x] Analyze codebase for chaos testing opportunities
- [x] Create chaos tests for version tag processing (`pkg/outdated/chaos_versioning_test.go`)
- [x] Create chaos tests for config validation (`pkg/config/chaos_config_test.go`)
- [x] Fix false positives in integration tests
- [x] Fix test pollution from updateOutputFlag in cmd tests
- [x] Fix test pollution in XML/CSV integration tests (updateRuleFlag persistence)
- [x] Review coverage gaps in pkg/packages (94.2%) and pkg/update (92.9%)
- [x] Commit and push all changes

## Files Modified
- `pkg/config/chaos_config_test.go` (NEW - 841 lines)
- `cmd/scan_test.go` (flag save/restore added)
- `cmd/update_constraint_test.go` (flag save/restore added)
- `cmd/update_sorting_test.go` (flag save/restore added)
- `cmd/update_test.go` (flag save/restore added)
- `cmd/output_format_integration_test.go` (flag save/restore for JSON, XML, CSV tests)

## Test Coverage

### YAML Parsing Edge Cases
| Test | Description | Status |
|------|-------------|--------|
| empty | Empty YAML | PASS |
| whitespace_only | Whitespace with tabs (Go YAML rejects) | PASS |
| unclosed_bracket | `rules: [npm` | PASS |
| duplicate_keys | YAML 1.1 duplicate key handling | PASS |
| anchor_reference | YAML anchors `*anchor` | PASS |
| deep_nesting_100_levels | 100+ nested structures | PASS |
| wide_map_1000_keys | 1000+ rules | PASS |

### Security Policy Tests
| Test | Description | Status |
|------|-------------|--------|
| single_dotdot | `../parent.yml` blocked | PASS |
| double_dotdot | `../../parent.yml` blocked | PASS |
| unicode_dotdot | Unicode periods `\u002e\u002e/` blocked | PASS |
| allowed_with_flag | Path traversal allowed when enabled | PASS |
| absolute_path | Absolute paths blocked by default | PASS |
| cyclic_A_extends_A | Self-reference cycle detected | PASS |
| cyclic_A_B_A | Two-file cycle detected | PASS |
| cyclic_A_B_C_A | Three-file cycle detected | PASS |
| file_over_limit | Files over 10MB rejected | PASS |

### Regex Pattern Tests
| Test | Description | Status |
|------|-------------|--------|
| unclosed_bracket | `[a-z` rejected | PASS |
| lookbehind | `(?<=foo)bar` rejected (RE2) | PASS |
| lookahead | `foo(?=bar)` rejected (RE2) | PASS |
| backreference | `(a)\1` rejected (RE2) | PASS |
| ReDoS_vulnerable | `(a+)+b` completes quickly (RE2 safe) | PASS |

### Field Value Tests
| Test | Description | Status |
|------|-------------|--------|
| null_byte | `\x00` in string rejected | PASS |
| shell_injection | `$(whoami)` stored as literal | PASS |
| command_substitution | `` `whoami` `` stored as literal | PASS |
| unicode_rtl | RTL override stored as literal | PASS |

### Group Configuration Tests
| Test | Description | Status |
|------|-------------|--------|
| empty_group_name | Empty string key allowed | PASS |
| numeric_group_name | Numeric key (123) allowed | PASS |
| unicode_group_name | Unicode key allowed | PASS |
| empty_package_list | Empty array allowed | PASS |

### Validation Tests
| Test | Description | Status |
|------|-------------|--------|
| unknown_root_field | Detected with error | PASS |
| typo_extends | `extend` suggests `extends` | PASS |
| type_coercion | YAML `yes`/`on` as boolean | PASS |

## Notes

### Test Organization
- **Unit tests** in `pkg/config/chaos_config_test.go` test the config layer directly
- **Integration tests** in `cmd/edge_cases_test.go` test via CLI commands
- Both test similar scenarios but at different abstraction levels (good practice)

### Design Decisions
1. **Inline YAML strings** used for chaos tests because:
   - Each test case documents the exact malformed input
   - Chaos tests deliberately create broken inputs
   - Having 100+ tiny fixture files would be unwieldy

2. **RE2 regex engine** provides built-in ReDoS protection - tests verify this

3. **YAML 1.1 behavior** documented in test comments (tabs rejected, duplicates rejected)

### Related Files
- `pkg/testdata_errors/_config-errors/` - Testdata fixtures used by `cmd/edge_cases_test.go`
- `pkg/config/load_test.go` - Additional config loading tests
- `pkg/outdated/chaos_versioning_test.go` - Chaos tests for version parsing

## Test Pollution Fixes

Integration tests were polluting package-level flag variables, causing subsequent tests to fail:

### Root Cause
- Tests set flags like `updateOutputFlag`, `updateRuleFlag`, `scanOutputFlag` in `setupFunc()` but never restored them
- When tests ran in alphabetical order, later tests would inherit unexpected flag values

### Tests Fixed
1. `TestRunScanNoMatches` - added `scanOutputFlag` save/restore
2. `TestFloatingConstraintInGroupShowsFloating` - added `updateOutputFlag` save/restore
3. `TestFloatingConstraintShowsUnsupported` - added `updateOutputFlag` save/restore
4. `TestRunUpdateSortingComparators` - added `updateOutputFlag` save/restore
5. `TestRunUpdateSortingDifferentPackageTypes` - added `updateOutputFlag` save/restore
6. `TestRunUpdateSortingDifferentGroups` - added `updateOutputFlag` save/restore
7. `TestRunUpdateNoPackages` - added `updateOutputFlag` save/restore
8. `TestRunUpdateWithMockedVersions` - added `updateOutputFlag` save/restore
9. `TestRunUpdateAfterAllValidationFailure` - added `updateOutputFlag` save/restore
10. `TestIntegration_AllCommands_JSONFileOutput` - comprehensive flag save/restore
11. `TestIntegration_AllCommands_XMLFileOutput` - comprehensive flag save/restore
12. `TestIntegration_AllCommands_CSVFileOutput` - comprehensive flag save/restore

## Coverage Analysis

Final coverage after all fixes:
| Package | Coverage |
|---------|----------|
| cmd | 96.2% |
| pkg/packages | 94.2% |
| pkg/update | 92.9% |
| pkg/config | 98.3% |
| All others | 97-100% |

Remaining uncovered code in pkg/packages and pkg/update is primarily:
- Error handling for crypto/rand failure (requires mocking internals)
- os.Rename failure during atomic write (requires complex mock)
- Permission-denied during file walks (system-dependent)
- Edge cases in symlink handling during directory traversal

These are acceptable coverage gaps as they would require complex test fixtures with diminishing returns.

## Battle Testing Results

Battle tested CLI on real-world projects per AGENTS.md:
- **express** (npm/js) - scan, list, outdated with all output formats
- **cobra** (golang/mod) - scan, list, outdated, **actual update** (go.mod modified)

All commands working correctly:
- Table, JSON, CSV, XML output formats
- Filter flags (--type, --rule, -p)
- Actual updates modify manifest files correctly
- Error handling for invalid paths, no-match filters

## Commits
- `38601b8` - Add comprehensive chaos tests for config validation
- `10ea825` - Fix test pollution from updateOutputFlag in multiple test files
- `349ba7b` - Fix test pollution in XML and CSV integration tests
- `1dd2093` - Update progress report with test pollution fixes and coverage analysis
- `be3118b` - Enhance AGENTS.md battle testing guidelines
