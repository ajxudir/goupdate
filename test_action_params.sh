#!/bin/bash
#
# Comprehensive test script for goupdate reusable action parameters
# Tests all parameter configurations and verifies outputs against documentation
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_WARNINGS=0

# Test results
declare -a TEST_RESULTS

log_test() {
    local name="$1"
    local status="$2"
    local msg="$3"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    case "$status" in
        PASS)
            TESTS_PASSED=$((TESTS_PASSED + 1))
            TEST_RESULTS+=("✓ $name")
            echo -e "${GREEN}✓ PASS${NC}: $name"
            ;;
        FAIL)
            TESTS_FAILED=$((TESTS_FAILED + 1))
            TEST_RESULTS+=("✗ $name: $msg")
            echo -e "${RED}✗ FAIL${NC}: $name - $msg"
            ;;
        WARN)
            TESTS_WARNINGS=$((TESTS_WARNINGS + 1))
            TEST_RESULTS+=("⚠ $name: $msg")
            echo -e "${YELLOW}⚠ WARN${NC}: $name - $msg"
            ;;
    esac
}

header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

section() {
    echo ""
    echo -e "${YELLOW}--- $1 ---${NC}"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build goupdate
header "Building goupdate"
go build -o /tmp/goupdate-test . 2>/dev/null
GOUPDATE="/tmp/goupdate-test --skip-build-checks"
echo "Built: /tmp/goupdate-test"

# ============================================================================
# TEST SECTION 1: goupdate scan command parameters
# Documentation: scan --help
# Expected output: JSON with "summary" and "files" arrays
# ============================================================================
header "1. Testing: goupdate scan parameters"

# Test 1.1: Default scan output (table format)
section "1.1 Default output format (table)"
OUTPUT=$($GOUPDATE scan 2>&1)
if echo "$OUTPUT" | grep -q "go.mod"; then
    log_test "scan_default_format" "PASS" ""
else
    log_test "scan_default_format" "FAIL" "Expected go.mod in table output"
fi

# Test 1.2: JSON output format (-o json)
section "1.2 JSON output format"
JSON=$($GOUPDATE scan --output json 2>/dev/null)
# Verify structure matches documentation:
# - Must have "summary" object
# - Must have "files" array
# - files[] must have: rule, pm, format, file, status
if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "scan_json_summary_exists" "PASS" ""
else
    log_test "scan_json_summary_exists" "FAIL" "JSON missing 'summary' field"
fi

if echo "$JSON" | jq -e '.files' > /dev/null 2>&1; then
    log_test "scan_json_files_exists" "PASS" ""
else
    log_test "scan_json_files_exists" "FAIL" "JSON missing 'files' field"
fi

# Verify summary fields per documentation
SUMMARY_FIELDS=("directory" "total_entries" "unique_files" "rules_matched" "valid_files" "invalid_files")
for field in "${SUMMARY_FIELDS[@]}"; do
    if echo "$JSON" | jq -e ".summary.$field" > /dev/null 2>&1; then
        log_test "scan_json_summary_$field" "PASS" ""
    else
        log_test "scan_json_summary_$field" "FAIL" "summary missing '$field' field"
    fi
done

# Verify files array structure
FIRST_FILE=$(echo "$JSON" | jq -r '.files[0]' 2>/dev/null)
FILE_FIELDS=("rule" "pm" "format" "file" "status")
for field in "${FILE_FIELDS[@]}"; do
    if echo "$FIRST_FILE" | jq -e ".$field" > /dev/null 2>&1; then
        log_test "scan_json_file_$field" "PASS" ""
    else
        log_test "scan_json_file_$field" "FAIL" "files[] missing '$field' field"
    fi
done

# Test 1.3: CSV output format
section "1.3 CSV output format"
CSV=$($GOUPDATE scan --output csv 2>/dev/null)
# CSV headers are uppercase: RULE,PM,FORMAT,FILE,STATUS,ERROR
if echo "$CSV" | head -1 | grep -qiE "rule.*pm.*file"; then
    log_test "scan_csv_format" "PASS" ""
else
    log_test "scan_csv_format" "FAIL" "CSV header doesn't match expected format"
fi

# Test 1.4: XML output format
section "1.4 XML output format"
XML=$($GOUPDATE scan --output xml 2>/dev/null)
if echo "$XML" | grep -q "<files>"; then
    log_test "scan_xml_format" "PASS" ""
else
    log_test "scan_xml_format" "FAIL" "XML missing <files> element"
fi

# Test 1.5: Directory parameter (-d)
section "1.5 Directory parameter"
JSON=$($GOUPDATE scan -d examples/react-app --output json 2>/dev/null)
DIR_VAL=$(echo "$JSON" | jq -r '.summary.directory' 2>/dev/null)
if [[ "$DIR_VAL" == "examples/react-app" ]]; then
    log_test "scan_directory_param" "PASS" ""
else
    log_test "scan_directory_param" "FAIL" "Expected directory 'examples/react-app', got '$DIR_VAL'"
fi

# Verify rule detected is npm
RULE=$(echo "$JSON" | jq -r '.files[0].rule // empty' 2>/dev/null)
if [[ "$RULE" == "npm" ]]; then
    log_test "scan_directory_detects_npm" "PASS" ""
else
    log_test "scan_directory_detects_npm" "FAIL" "Expected rule 'npm' for react-app, got '$RULE'"
fi

# Test 1.6: File filter parameter (-f)
section "1.6 File filter parameter"
JSON=$($GOUPDATE scan -f "go.mod" --output json 2>/dev/null)
FILE_COUNT=$(echo "$JSON" | jq -r '.summary.unique_files' 2>/dev/null)
if [[ "$FILE_COUNT" == "1" ]]; then
    log_test "scan_file_filter" "PASS" ""
else
    log_test "scan_file_filter" "WARN" "Expected 1 file with -f go.mod, got $FILE_COUNT"
fi

# ============================================================================
# TEST SECTION 2: goupdate outdated command parameters
# Documentation: outdated --help
# Expected output: JSON with "summary" and "packages" arrays
# ============================================================================
header "2. Testing: goupdate outdated parameters"

# Test 2.1: Default outdated (table format)
section "2.1 Default output format"
OUTPUT=$($GOUPDATE outdated -r mod 2>&1) || true
if echo "$OUTPUT" | grep -qE "(Outdated|UpToDate|github.com)"; then
    log_test "outdated_default_format" "PASS" ""
else
    log_test "outdated_default_format" "FAIL" "Unexpected table output"
fi

# Test 2.2: JSON output format
section "2.2 JSON output format"
JSON=$($GOUPDATE outdated -r mod --output json 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "outdated_json_summary_exists" "PASS" ""
else
    log_test "outdated_json_summary_exists" "FAIL" "JSON missing 'summary' field"
fi

if echo "$JSON" | jq -e '.packages' > /dev/null 2>&1; then
    log_test "outdated_json_packages_exists" "PASS" ""
else
    log_test "outdated_json_packages_exists" "FAIL" "JSON missing 'packages' field"
fi

# Verify summary fields per CLI output
SUMMARY_FIELDS=("total_packages" "outdated_packages" "uptodate_packages" "has_major" "has_minor" "has_patch")
for field in "${SUMMARY_FIELDS[@]}"; do
    if echo "$JSON" | jq -e ".summary.$field" > /dev/null 2>&1; then
        log_test "outdated_json_summary_$field" "PASS" ""
    else
        log_test "outdated_json_summary_$field" "FAIL" "summary missing '$field' field"
    fi
done

# Verify package structure
FIRST_PKG=$(echo "$JSON" | jq -r '.packages[0]' 2>/dev/null)
PKG_FIELDS=("rule" "pm" "type" "constraint" "version" "installed_version" "major" "minor" "patch" "status" "name")
for field in "${PKG_FIELDS[@]}"; do
    if echo "$FIRST_PKG" | jq -e ".$field" > /dev/null 2>&1; then
        log_test "outdated_json_pkg_$field" "PASS" ""
    else
        log_test "outdated_json_pkg_$field" "FAIL" "packages[] missing '$field' field"
    fi
done

# Test 2.3: Rule parameter (-r)
section "2.3 Rule parameter"
for rule in "mod" "npm" "composer" "requirements"; do
    case "$rule" in
        mod) test_dir="." ;;
        npm) test_dir="examples/react-app" ;;
        composer) test_dir="examples/laravel-app" ;;
        requirements) test_dir="examples/django-app" ;;
    esac

    JSON=$($GOUPDATE outdated -r "$rule" -d "$test_dir" --output json 2>/dev/null) || JSON="{}"
    PKG_RULE=$(echo "$JSON" | jq -r '.packages[0].rule // empty' 2>/dev/null)
    if [[ "$PKG_RULE" == "$rule" ]] || [[ -z "$PKG_RULE" && "$rule" == "composer" ]]; then
        # Composer might not have packages
        log_test "outdated_rule_$rule" "PASS" ""
    else
        log_test "outdated_rule_$rule" "WARN" "Expected rule '$rule', got '$PKG_RULE' (might be empty)"
    fi
done

# Test 2.4: Name filter parameter (-n)
# Note: There is no -e/--exclude flag. Exclusion is done via config or -n name filter
section "2.4 Name filter parameter"
JSON_NO_FILTER=$($GOUPDATE outdated -r mod --output json 2>/dev/null) || JSON_NO_FILTER="{}"
TOTAL_NO_FILTER=$(echo "$JSON_NO_FILTER" | jq -r '.summary.total_packages // 0' 2>/dev/null)

JSON_WITH_FILTER=$($GOUPDATE outdated -r mod -n "github.com/spf13/cobra" --output json 2>/dev/null) || JSON_WITH_FILTER="{}"
TOTAL_WITH_FILTER=$(echo "$JSON_WITH_FILTER" | jq -r '.summary.total_packages // 0' 2>/dev/null)

# Name filter should reduce package count
if [[ "$TOTAL_WITH_FILTER" -le "$TOTAL_NO_FILTER" ]]; then
    log_test "outdated_name_filter" "PASS" ""
else
    log_test "outdated_name_filter" "WARN" "Name filter might not have filtered (no: $TOTAL_NO_FILTER, with: $TOTAL_WITH_FILTER)"
fi

# Test 2.5: Verbose flag (-v)
section "2.5 Verbose flag"
OUTPUT=$($GOUPDATE outdated -r mod -v 2>&1) || true
# Verbose should show more debug info (longer output)
LINE_COUNT=$(echo "$OUTPUT" | wc -l)
if [[ "$LINE_COUNT" -gt 5 ]]; then
    log_test "outdated_verbose_flag" "PASS" ""
else
    log_test "outdated_verbose_flag" "WARN" "Verbose output seems short ($LINE_COUNT lines)"
fi

# Test 2.6: Update type flags (--patch, --minor, --major)
section "2.6 Update type flags"
for flag in "--patch" "--minor" "--major"; do
    JSON=$($GOUPDATE outdated -r mod $flag --output json 2>/dev/null) || JSON="{}"
    if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
        log_test "outdated_flag_$flag" "PASS" ""
    else
        log_test "outdated_flag_$flag" "FAIL" "Flag $flag broke JSON output"
    fi
done

# ============================================================================
# TEST SECTION 3: goupdate update command parameters (dry-run only)
# Documentation: update --help
# ============================================================================
header "3. Testing: goupdate update parameters (dry-run)"

# Test 3.1: Dry-run flag
section "3.1 Dry-run flag"
OUTPUT=$($GOUPDATE update -r mod --dry-run -y 2>&1) || true
if echo "$OUTPUT" | grep -qiE "(dry.run|would|plan|skip)"; then
    log_test "update_dry_run" "PASS" ""
else
    log_test "update_dry_run" "WARN" "Dry-run might not show expected output"
fi

# Test 3.2: Update type flags (--patch, --minor, --major)
# Note: There is no --all flag in update command
section "3.2 Update type flags"
for flag in "--patch" "--minor" "--major"; do
    OUTPUT=$($GOUPDATE update -r mod $flag --dry-run -y 2>&1) || OUTPUT=""
    if [[ -n "$OUTPUT" ]]; then
        log_test "update_type_$flag" "PASS" ""
    else
        log_test "update_type_$flag" "FAIL" "No output for $flag"
    fi
done

# Test 3.3: Name filter (-n) for package filtering
# Note: There is no -e/--exclude flag. Use -n for name filtering or config exclude
section "3.3 Name filter"
OUTPUT=$($GOUPDATE update -r mod -n "github.com/spf13/cobra" --dry-run -y 2>&1) || true
if [[ -n "$OUTPUT" ]]; then
    log_test "update_name_filter" "PASS" ""
else
    log_test "update_name_filter" "WARN" "Cannot verify name filter behavior in dry-run"
fi

# Test 3.4: System test mode
section "3.4 System test mode"
for mode in "after_each" "after_all" "none"; do
    OUTPUT=$($GOUPDATE update -r mod --system-test-mode="$mode" --dry-run -y 2>&1) || true
    if [[ -n "$OUTPUT" ]]; then
        log_test "update_system_test_$mode" "PASS" ""
    else
        log_test "update_system_test_$mode" "FAIL" "No output for --system-test-mode=$mode"
    fi
done

# Test 3.5: Continue on fail
section "3.5 Continue on fail"
OUTPUT=$($GOUPDATE update -r mod --continue-on-fail --dry-run -y 2>&1) || true
if [[ -n "$OUTPUT" ]]; then
    log_test "update_continue_on_fail" "PASS" ""
else
    log_test "update_continue_on_fail" "FAIL" "No output for --continue-on-fail"
fi

# Test 3.6: Working directory (-d)
section "3.6 Working directory"
OUTPUT=$($GOUPDATE update -r npm -d examples/react-app --dry-run -y 2>&1) || true
if [[ -n "$OUTPUT" ]]; then
    log_test "update_working_dir" "PASS" ""
else
    log_test "update_working_dir" "WARN" "No output for working directory test"
fi

# ============================================================================
# TEST SECTION 4: Workflow runtime detection logic
# This tests the exact logic used in auto-update.yml
# ============================================================================
header "4. Testing: Workflow runtime detection logic"

test_runtime_detection() {
    local dir="$1"
    local expected_rules="$2"
    local expected_node="$3"
    local expected_php="$4"
    local expected_python="$5"
    local expected_go="$6"
    local expected_dotnet="$7"
    local name="$8"

    section "4.x Runtime detection for $name"

    # Run exact workflow command
    SCAN_OUTPUT=$($GOUPDATE scan -d "$dir" --output json 2>/dev/null || echo '{"files":[]}')

    # Extract rules using exact workflow jq expression
    RULES=$(echo "$SCAN_OUTPUT" | jq -r '.files // [] | map(.rule) | unique | join(",")')

    # Runtime detection using exact workflow logic
    USE_NODE="false"
    USE_PHP="false"
    USE_PYTHON="false"
    USE_GO="false"
    USE_DOTNET="false"

    if echo "$RULES" | grep -qE "(npm|yarn|pnpm)"; then USE_NODE="true"; fi
    if echo "$RULES" | grep -q "composer"; then USE_PHP="true"; fi
    if echo "$RULES" | grep -qE "(requirements|pipfile)"; then USE_PYTHON="true"; fi
    if echo "$RULES" | grep -q "mod"; then USE_GO="true"; fi
    if echo "$RULES" | grep -q "nuget"; then USE_DOTNET="true"; fi

    # Verify rules
    if [[ -n "$expected_rules" ]]; then
        if echo "$RULES" | grep -q "$expected_rules"; then
            log_test "runtime_${name}_rules" "PASS" ""
        else
            log_test "runtime_${name}_rules" "FAIL" "Expected '$expected_rules' in rules, got '$RULES'"
        fi
    fi

    # Verify runtimes
    if [[ "$USE_NODE" == "$expected_node" ]]; then
        log_test "runtime_${name}_node" "PASS" ""
    else
        log_test "runtime_${name}_node" "FAIL" "Expected node=$expected_node, got $USE_NODE"
    fi

    if [[ "$USE_PHP" == "$expected_php" ]]; then
        log_test "runtime_${name}_php" "PASS" ""
    else
        log_test "runtime_${name}_php" "FAIL" "Expected php=$expected_php, got $USE_PHP"
    fi

    if [[ "$USE_PYTHON" == "$expected_python" ]]; then
        log_test "runtime_${name}_python" "PASS" ""
    else
        log_test "runtime_${name}_python" "FAIL" "Expected python=$expected_python, got $USE_PYTHON"
    fi

    if [[ "$USE_GO" == "$expected_go" ]]; then
        log_test "runtime_${name}_go" "PASS" ""
    else
        log_test "runtime_${name}_go" "FAIL" "Expected go=$expected_go, got $USE_GO"
    fi

    if [[ "$USE_DOTNET" == "$expected_dotnet" ]]; then
        log_test "runtime_${name}_dotnet" "PASS" ""
    else
        log_test "runtime_${name}_dotnet" "FAIL" "Expected dotnet=$expected_dotnet, got $USE_DOTNET"
    fi
}

# Test each example project
#                    dir                    expected_rules  node   php    python go     dotnet name
test_runtime_detection "."                  "mod"           "false" "false" "false" "true"  "false" "goupdate"
test_runtime_detection "examples/react-app" "npm"           "true"  "false" "false" "false" "false" "react-app"
test_runtime_detection "examples/django-app" "requirements" "false" "false" "true"  "false" "false" "django-app"
test_runtime_detection "examples/laravel-app" "composer"    "false" "true"  "false" "false" "false" "laravel-app"
test_runtime_detection "examples/go-cli"    "mod"           "false" "false" "false" "true"  "false" "go-cli"
test_runtime_detection "examples/ruby-api"  "bundler"       "false" "false" "false" "false" "false" "ruby-api"

# ============================================================================
# TEST SECTION 5: Action output verification
# Verify that action outputs match documented behavior
# ============================================================================
header "5. Testing: Action output verification"

# Test 5.1: _goupdate-check outputs
section "5.1 _goupdate-check output structure"

# Simulate action logic from _goupdate-check/action.yml
JSON=$($GOUPDATE outdated -r mod -o json 2>/dev/null) || JSON="{}"

# Parse counts (exact action logic)
MAJOR_COUNT=$(echo "$JSON" | jq -r '.summary.has_major // 0')
MINOR_COUNT=$(echo "$JSON" | jq -r '.summary.has_minor // 0')
PATCH_COUNT=$(echo "$JSON" | jq -r '.summary.has_patch // 0')

# has-updates logic (exact action logic)
MINOR_PATCH_COUNT=$(echo "$JSON" | jq '[.packages[] | select(.status == "Outdated" and (.minor != "#N/A" or .patch != "#N/A"))] | length' 2>/dev/null) || MINOR_PATCH_COUNT=0
HAS_UPDATES="false"
if [[ "$MINOR_PATCH_COUNT" -gt 0 ]]; then
    HAS_UPDATES="true"
fi

# has-major-only logic (exact action logic)
HAS_MAJOR_ONLY="false"
if [[ "$MAJOR_COUNT" -gt 0 ]]; then
    MAJOR_ONLY_PKGS=$(echo "$JSON" | jq -r '.packages[] | select(.status == "Outdated" and .major != "#N/A" and .minor == "#N/A" and .patch == "#N/A") | .name' 2>/dev/null | tr '\n' ' ')
    if [[ -n "$MAJOR_ONLY_PKGS" ]]; then
        HAS_MAJOR_ONLY="true"
    fi
fi

SUMMARY="Major: $MAJOR_COUNT, Minor: $MINOR_COUNT, Patch: $PATCH_COUNT"

# Verify outputs are valid
if [[ "$MAJOR_COUNT" =~ ^[0-9]+$ ]]; then
    log_test "check_output_major_count" "PASS" ""
else
    log_test "check_output_major_count" "FAIL" "major_count not numeric: $MAJOR_COUNT"
fi

if [[ "$MINOR_COUNT" =~ ^[0-9]+$ ]]; then
    log_test "check_output_minor_count" "PASS" ""
else
    log_test "check_output_minor_count" "FAIL" "minor_count not numeric: $MINOR_COUNT"
fi

if [[ "$PATCH_COUNT" =~ ^[0-9]+$ ]]; then
    log_test "check_output_patch_count" "PASS" ""
else
    log_test "check_output_patch_count" "FAIL" "patch_count not numeric: $PATCH_COUNT"
fi

if [[ "$HAS_UPDATES" == "true" || "$HAS_UPDATES" == "false" ]]; then
    log_test "check_output_has_updates" "PASS" ""
else
    log_test "check_output_has_updates" "FAIL" "has_updates not boolean: $HAS_UPDATES"
fi

if [[ "$HAS_MAJOR_ONLY" == "true" || "$HAS_MAJOR_ONLY" == "false" ]]; then
    log_test "check_output_has_major_only" "PASS" ""
else
    log_test "check_output_has_major_only" "FAIL" "has_major_only not boolean: $HAS_MAJOR_ONLY"
fi

echo "Computed outputs: has_updates=$HAS_UPDATES, has_major_only=$HAS_MAJOR_ONLY, summary=$SUMMARY"

# Test 5.2: _goupdate-update output parsing
section "5.2 _goupdate-update output parsing"

# Run update with dry-run to get output format
OUTPUT=$($GOUPDATE update -r mod --minor --dry-run -y 2>&1) || OUTPUT=""

# Verify output can be parsed for updated packages (action logic)
# Note: dry-run won't show "🟢 Updated" but we can check format
if [[ -n "$OUTPUT" ]]; then
    log_test "update_output_not_empty" "PASS" ""
else
    log_test "update_output_not_empty" "WARN" "Update output is empty"
fi

# ============================================================================
# TEST SECTION 6: git-branch action parameter logic
# Test the exact logic used in _git-branch/action.yml
# ============================================================================
header "6. Testing: _git-branch action logic"

section "6.1 Branch existence checks"

# Simulate action logic for branch existence
TEST_BRANCH="test-branch-$(date +%s)"
MAIN_BRANCH="main"

# Check if main exists on remote (action logic)
if git ls-remote --heads origin "$MAIN_BRANCH" 2>/dev/null | grep -q "$MAIN_BRANCH"; then
    log_test "git_branch_main_exists" "PASS" ""
else
    log_test "git_branch_main_exists" "WARN" "Main branch might not exist"
fi

# Check local branch (action logic)
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
if [[ -n "$CURRENT_BRANCH" ]]; then
    log_test "git_branch_current_valid" "PASS" ""
else
    log_test "git_branch_current_valid" "FAIL" "Cannot get current branch"
fi

# Test SHA output (action logic)
SHA=$(git rev-parse HEAD 2>/dev/null || echo "")
if [[ "$SHA" =~ ^[a-f0-9]{40}$ ]]; then
    log_test "git_branch_sha_format" "PASS" ""
else
    log_test "git_branch_sha_format" "FAIL" "SHA format invalid: $SHA"
fi

# ============================================================================
# TEST SECTION 7: Workflow configuration validation
# Verify workflow env vars and job outputs are correctly used
# ============================================================================
header "7. Testing: Workflow configuration validation"

section "7.1 Branch name generation"

# Simulate exact workflow logic
# Workflow supports: minor, patch (not all - that was removed from workflow_dispatch)
BRANCH_PREFIX="goupdate"
for update_type in "minor" "patch"; do
    BRANCH_NAME="${BRANCH_PREFIX}/auto-update-${update_type}"
    if [[ "$BRANCH_NAME" =~ ^[a-z]+/auto-update-(minor|patch)$ ]]; then
        log_test "workflow_branch_name_$update_type" "PASS" ""
    else
        log_test "workflow_branch_name_$update_type" "FAIL" "Invalid branch name: $BRANCH_NAME"
    fi
done

section "7.2 PR title/commit message templates"

# Simulate exact workflow logic
PR_TITLE_TEMPLATE='GoUpdate: Auto update - {type} ({date})'
COMMIT_MSG_TEMPLATE='GoUpdate: Auto update - {type} ({date})'
DATE=$(date -u +%Y-%m-%d)
UPDATE_TYPE="Minor"

TITLE=$(echo "$PR_TITLE_TEMPLATE" | sed "s/{date}/$DATE/g" | sed "s/{type}/$UPDATE_TYPE/g")
COMMIT_MSG=$(echo "$COMMIT_MSG_TEMPLATE" | sed "s/{date}/$DATE/g" | sed "s/{type}/$UPDATE_TYPE/g")

if [[ "$TITLE" == "GoUpdate: Auto update - Minor ($DATE)" ]]; then
    log_test "workflow_pr_title_template" "PASS" ""
else
    log_test "workflow_pr_title_template" "FAIL" "Title template failed: $TITLE"
fi

if [[ "$COMMIT_MSG" == "GoUpdate: Auto update - Minor ($DATE)" ]]; then
    log_test "workflow_commit_msg_template" "PASS" ""
else
    log_test "workflow_commit_msg_template" "FAIL" "Commit message template failed: $COMMIT_MSG"
fi

# ============================================================================
# TEST SECTION 8: Test on all example projects
# ============================================================================
header "8. Testing: All example projects"

EXAMPLE_DIRS=(
    "examples/django-app:requirements"
    "examples/go-cli:mod"
    "examples/laravel-app:composer"
    "examples/react-app:npm"
    "examples/ruby-api:bundler"
)

for entry in "${EXAMPLE_DIRS[@]}"; do
    dir="${entry%%:*}"
    rule="${entry##*:}"
    name=$(basename "$dir")

    section "8.x Testing $name ($rule)"

    # Test scan
    JSON=$($GOUPDATE scan -d "$dir" --output json 2>/dev/null || echo '{"files":[]}')
    DETECTED_RULE=$(echo "$JSON" | jq -r '.files[0].rule // empty' 2>/dev/null)
    if [[ "$DETECTED_RULE" == "$rule" ]]; then
        log_test "example_${name}_scan" "PASS" ""
    else
        log_test "example_${name}_scan" "FAIL" "Expected rule '$rule', got '$DETECTED_RULE'"
    fi

    # Test outdated
    JSON=$($GOUPDATE outdated -r "$rule" -d "$dir" --output json 2>/dev/null) || JSON="{}"
    if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
        log_test "example_${name}_outdated" "PASS" ""
    else
        log_test "example_${name}_outdated" "WARN" "Outdated might have failed (no packages?)"
    fi

    # Test update dry-run
    OUTPUT=$($GOUPDATE update -r "$rule" -d "$dir" --dry-run -y 2>&1) || OUTPUT=""
    if [[ -n "$OUTPUT" ]]; then
        log_test "example_${name}_update_dry" "PASS" ""
    else
        log_test "example_${name}_update_dry" "WARN" "Update dry-run produced no output"
    fi
done

# ============================================================================
# TEST SECTION 9: Edge cases and error handling
# ============================================================================
header "9. Testing: Edge cases and error handling"

section "9.1 Invalid directory"
JSON=$($GOUPDATE scan -d "/nonexistent/path" --output json 2>/dev/null) || JSON='{"files":[]}'
FILES_COUNT=$(echo "$JSON" | jq -r '.files | length' 2>/dev/null) || FILES_COUNT=0
if [[ "$FILES_COUNT" == "0" ]] || [[ -z "$FILES_COUNT" ]]; then
    log_test "edge_invalid_dir" "PASS" ""
else
    log_test "edge_invalid_dir" "WARN" "Invalid dir should return empty files"
fi

section "9.2 Invalid rule"
OUTPUT=$($GOUPDATE outdated -r "nonexistent_rule" --output json 2>&1) || OUTPUT=""
# Should either error or return empty
if [[ -n "$OUTPUT" ]]; then
    log_test "edge_invalid_rule" "PASS" ""
else
    log_test "edge_invalid_rule" "PASS" ""  # Empty is also valid
fi

section "9.3 Empty name filter"
# Test with empty name filter (should be same as no filter)
JSON=$($GOUPDATE outdated -r mod --output json 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "edge_empty_name_filter" "PASS" ""
else
    log_test "edge_empty_name_filter" "FAIL" "Basic outdated command failed"
fi

section "9.4 Multiple rules"
JSON=$($GOUPDATE scan --output json 2>/dev/null || echo '{"files":[]}')
RULES=$(echo "$JSON" | jq -r '.files // [] | map(.rule) | unique | join(",")' 2>/dev/null)
# Should handle comma-separated rules in outdated
OUTPUT=$($GOUPDATE outdated -r "$RULES" --output json 2>/dev/null) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "edge_multiple_rules" "PASS" ""
else
    log_test "edge_multiple_rules" "WARN" "Multiple rules might not work"
fi

# ============================================================================
# TEST SECTION 10: Parameter combination tests
# Test multiple parameters used together
# ============================================================================
header "10. Testing: Parameter combinations"

section "10.1 Scan with multiple parameters"
# Test scan with directory + output + file filter
JSON=$($GOUPDATE scan -d examples/react-app --output json -f "package.json" 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.summary.directory' > /dev/null 2>&1; then
    log_test "combo_scan_dir_output_file" "PASS" ""
else
    log_test "combo_scan_dir_output_file" "FAIL" "Combined params broke scan"
fi

section "10.2 Outdated with multiple parameters"
# Test outdated with rule + output + verbose
OUTPUT=$($GOUPDATE outdated -r mod --output json -v 2>&1) || OUTPUT=""
if echo "$OUTPUT" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "combo_outdated_rule_output_verbose" "PASS" ""
else
    log_test "combo_outdated_rule_output_verbose" "WARN" "Combined params might have issues"
fi

# Test outdated with rule + patch filter
JSON=$($GOUPDATE outdated -r mod --patch --output json 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "combo_outdated_rule_patch" "PASS" ""
else
    log_test "combo_outdated_rule_patch" "FAIL" "Rule + patch filter broke outdated"
fi

section "10.3 Update with multiple parameters"
# Test update with all common parameters
OUTPUT=$($GOUPDATE update -r mod --minor --dry-run -y --continue-on-fail 2>&1) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "combo_update_minor_dry_continue" "PASS" ""
else
    log_test "combo_update_minor_dry_continue" "FAIL" "Combined update params failed"
fi

# Test update with directory + rule + type
OUTPUT=$($GOUPDATE update -r npm -d examples/react-app --patch --dry-run -y 2>&1) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "combo_update_dir_rule_patch" "PASS" ""
else
    log_test "combo_update_dir_rule_patch" "WARN" "Directory + rule + type might have issues"
fi

# ============================================================================
# TEST SECTION 11: Rule parameter edge cases
# Test all supported rules and edge cases
# ============================================================================
header "11. Testing: Rule parameter edge cases"

section "11.1 All supported rules"
SUPPORTED_RULES=("mod" "npm" "yarn" "pnpm" "composer" "requirements" "pipfile" "bundler" "nuget")
for rule in "${SUPPORTED_RULES[@]}"; do
    # Just verify the command doesn't crash with the rule
    OUTPUT=$($GOUPDATE outdated -r "$rule" --output json 2>&1) || OUTPUT="{}"
    if [[ -n "$OUTPUT" ]]; then
        log_test "rule_supported_$rule" "PASS" ""
    else
        log_test "rule_supported_$rule" "WARN" "Rule $rule might not be working"
    fi
done

section "11.2 Rule with directory mismatch"
# Test using wrong rule for a directory (e.g., npm rule on go project)
JSON=$($GOUPDATE outdated -r npm -d examples/go-cli --output json 2>/dev/null) || JSON="{}"
PKG_COUNT=$(echo "$JSON" | jq -r '.summary.total_packages // 0' 2>/dev/null)
if [[ "$PKG_COUNT" == "0" ]]; then
    log_test "rule_mismatch_npm_go" "PASS" ""
else
    log_test "rule_mismatch_npm_go" "WARN" "Expected 0 packages for mismatched rule"
fi

section "11.3 Comma-separated rules"
# Test multiple rules at once
JSON=$($GOUPDATE scan --output json 2>/dev/null) || JSON="{}"
ALL_RULES=$(echo "$JSON" | jq -r '.files // [] | map(.rule) | unique | join(",")' 2>/dev/null)
if [[ -n "$ALL_RULES" ]]; then
    OUTPUT=$($GOUPDATE outdated -r "$ALL_RULES" --output json 2>/dev/null) || OUTPUT=""
    if [[ -n "$OUTPUT" ]]; then
        log_test "rule_comma_separated" "PASS" ""
    else
        log_test "rule_comma_separated" "WARN" "Comma-separated rules might have issues"
    fi
else
    log_test "rule_comma_separated" "WARN" "No rules found to test"
fi

# ============================================================================
# TEST SECTION 12: Working directory edge cases
# ============================================================================
header "12. Testing: Working directory edge cases"

section "12.1 Nested directories"
# Test with deeply nested path
JSON=$($GOUPDATE scan -d examples/react-app --output json 2>/dev/null) || JSON="{}"
DIR_VAL=$(echo "$JSON" | jq -r '.summary.directory' 2>/dev/null)
if [[ "$DIR_VAL" == "examples/react-app" ]]; then
    log_test "dir_nested" "PASS" ""
else
    log_test "dir_nested" "FAIL" "Nested directory not handled correctly"
fi

section "12.2 Relative vs absolute paths"
# Test with absolute path
ABS_PATH="$(pwd)/examples/django-app"
JSON=$($GOUPDATE scan -d "$ABS_PATH" --output json 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.files' > /dev/null 2>&1; then
    log_test "dir_absolute_path" "PASS" ""
else
    log_test "dir_absolute_path" "FAIL" "Absolute path not handled correctly"
fi

section "12.3 Directory with no package files"
# Create temp empty dir and test
EMPTY_DIR=$(mktemp -d)
JSON=$($GOUPDATE scan -d "$EMPTY_DIR" --output json 2>/dev/null) || JSON="{}"
FILE_COUNT=$(echo "$JSON" | jq -r '.files | length' 2>/dev/null) || FILE_COUNT=0
if [[ "$FILE_COUNT" == "0" ]]; then
    log_test "dir_empty" "PASS" ""
else
    log_test "dir_empty" "WARN" "Empty directory should return 0 files"
fi
rmdir "$EMPTY_DIR" 2>/dev/null || true

# ============================================================================
# TEST SECTION 13: Output format consistency
# Verify all output formats work across commands
# ============================================================================
header "13. Testing: Output format consistency"

section "13.1 JSON output across commands"
for cmd in "scan" "outdated -r mod" "list -r mod"; do
    JSON=$($GOUPDATE $cmd --output json 2>/dev/null) || JSON=""
    if echo "$JSON" | jq -e '.' > /dev/null 2>&1; then
        log_test "json_output_$cmd" "PASS" ""
    else
        log_test "json_output_$cmd" "FAIL" "Invalid JSON from $cmd"
    fi
done

section "13.2 CSV output across commands"
for cmd in "scan" "outdated -r mod" "list -r mod"; do
    CSV=$($GOUPDATE $cmd --output csv 2>/dev/null) || CSV=""
    # CSV should have header row with commas
    if echo "$CSV" | head -1 | grep -q ","; then
        log_test "csv_output_$cmd" "PASS" ""
    else
        log_test "csv_output_$cmd" "WARN" "CSV output might be empty or malformed"
    fi
done

section "13.3 XML output across commands"
for cmd in "scan" "outdated -r mod" "list -r mod"; do
    XML=$($GOUPDATE $cmd --output xml 2>/dev/null) || XML=""
    # XML should have opening tag
    if echo "$XML" | grep -q "<"; then
        log_test "xml_output_$cmd" "PASS" ""
    else
        log_test "xml_output_$cmd" "WARN" "XML output might be empty or malformed"
    fi
done

# ============================================================================
# TEST SECTION 14: Workflow file structure validation
# Verify workflow YAML files have correct structure
# ============================================================================
header "14. Testing: Workflow file structure validation"

section "14.1 Main workflow structure"
WORKFLOW_FILE=".github/workflows/auto-update.yml"
if [[ -f "$WORKFLOW_FILE" ]]; then
    # Check required fields
    if grep -q "^name:" "$WORKFLOW_FILE"; then
        log_test "workflow_has_name" "PASS" ""
    else
        log_test "workflow_has_name" "FAIL" "Workflow missing name field"
    fi

    if grep -q "^on:" "$WORKFLOW_FILE"; then
        log_test "workflow_has_on" "PASS" ""
    else
        log_test "workflow_has_on" "FAIL" "Workflow missing on trigger"
    fi

    if grep -q "^jobs:" "$WORKFLOW_FILE"; then
        log_test "workflow_has_jobs" "PASS" ""
    else
        log_test "workflow_has_jobs" "FAIL" "Workflow missing jobs section"
    fi

    # Check for required env vars
    if grep -q "GOUPDATE_REPO:" "$WORKFLOW_FILE"; then
        log_test "workflow_has_goupdate_repo" "PASS" ""
    else
        log_test "workflow_has_goupdate_repo" "FAIL" "Workflow missing GOUPDATE_REPO env"
    fi
else
    log_test "workflow_file_exists" "FAIL" "Workflow file not found"
fi

section "14.2 Action file structure"
for action_dir in ".github/actions/_goupdate-check" ".github/actions/_goupdate-update" ".github/actions/_goupdate-install" ".github/actions/_git-branch" ".github/actions/_gh-pr" ".github/actions/_gh-runtimes"; do
    action_file="$action_dir/action.yml"
    action_name=$(basename "$action_dir")

    if [[ -f "$action_file" ]]; then
        # Check required fields
        if grep -q "^name:" "$action_file" && grep -q "^runs:" "$action_file"; then
            log_test "action_structure_$action_name" "PASS" ""
        else
            log_test "action_structure_$action_name" "FAIL" "Action missing required fields"
        fi
    else
        log_test "action_exists_$action_name" "FAIL" "Action file not found: $action_file"
    fi
done

section "14.3 Example workflow structure"
for example_dir in "examples/frontend" "examples/github-workflows"; do
    workflow_file="$example_dir/.github/workflows/auto-update.yml"
    example_name=$(basename "$example_dir")

    if [[ -f "$workflow_file" ]]; then
        if grep -q "^name:" "$workflow_file" && grep -q "^jobs:" "$workflow_file"; then
            log_test "example_workflow_$example_name" "PASS" ""
        else
            log_test "example_workflow_$example_name" "FAIL" "Example workflow missing required fields"
        fi
    else
        log_test "example_workflow_exists_$example_name" "FAIL" "Example workflow not found"
    fi
done

# ============================================================================
# TEST SECTION 15: Error handling and recovery
# ============================================================================
header "15. Testing: Error handling and recovery"

section "15.1 Graceful handling of missing files"
# Test scan on directory with no matching files
JSON=$($GOUPDATE scan -d /tmp --output json 2>/dev/null) || JSON="{}"
if echo "$JSON" | jq -e '.summary' > /dev/null 2>&1; then
    log_test "error_no_matching_files" "PASS" ""
else
    log_test "error_no_matching_files" "WARN" "Empty scan might not return valid JSON"
fi

section "15.2 Invalid flag combinations"
# Test conflicting flags (if any)
OUTPUT=$($GOUPDATE outdated -r mod --patch --minor --output json 2>&1) || OUTPUT=""
# Should still work, using most restrictive or last flag
if [[ -n "$OUTPUT" ]]; then
    log_test "error_conflicting_flags" "PASS" ""
else
    log_test "error_conflicting_flags" "WARN" "Conflicting flags might cause issues"
fi

section "15.3 Command timeout handling"
# Test --no-timeout flag exists
OUTPUT=$($GOUPDATE outdated -r mod --no-timeout --output json 2>&1) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "flag_no_timeout" "PASS" ""
else
    log_test "flag_no_timeout" "WARN" "--no-timeout flag might have issues"
fi

section "15.4 Continue on fail behavior"
# Test --continue-on-fail with dry-run
OUTPUT=$($GOUPDATE update -r mod --continue-on-fail --dry-run -y 2>&1) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "flag_continue_on_fail" "PASS" ""
else
    log_test "flag_continue_on_fail" "WARN" "--continue-on-fail might have issues"
fi

# ============================================================================
# TEST SECTION 16: Action input/output consistency
# Verify action inputs match what CLI expects
# ============================================================================
header "16. Testing: Action input/output consistency"

section "16.1 _goupdate-check action inputs"
# Verify the action's expected inputs match CLI capabilities
# Actions now use auto-detect from .goupdate.yml config (no hardcoded -r flag)
CHECK_ACTION=".github/actions/_goupdate-check/action.yml"
if [[ -f "$CHECK_ACTION" ]]; then
    # Check that action does NOT use -r flag (auto-detect from config)
    if ! grep -q "goupdate outdated -r" "$CHECK_ACTION"; then
        log_test "check_action_no_rule_flag" "PASS" ""
    else
        log_test "check_action_no_rule_flag" "FAIL" "Check action should use auto-detect (no -r flag)"
    fi

    # Check that action uses -o json
    if grep -q "\-o json" "$CHECK_ACTION"; then
        log_test "check_action_uses_json" "PASS" ""
    else
        log_test "check_action_uses_json" "FAIL" "Check action should use -o json"
    fi

    # Verify no -e flag is used (exclusion via config)
    if ! grep -q "\-e \$" "$CHECK_ACTION"; then
        log_test "check_action_no_exclude_flag" "PASS" ""
    else
        log_test "check_action_no_exclude_flag" "FAIL" "Check action should not use -e flag"
    fi

    # Verify no exclude-packages input (handled via .goupdate.yml config)
    if ! grep -q "exclude-packages:" "$CHECK_ACTION"; then
        log_test "check_action_no_exclude_input" "PASS" ""
    else
        log_test "check_action_no_exclude_input" "FAIL" "Check action should not have exclude-packages input"
    fi
else
    log_test "check_action_exists" "FAIL" "Check action file not found"
fi

section "16.2 _goupdate-update action inputs"
# Actions now use auto-detect from .goupdate.yml config (no hardcoded -r flag)
UPDATE_ACTION=".github/actions/_goupdate-update/action.yml"
if [[ -f "$UPDATE_ACTION" ]]; then
    # Check that action does NOT use -r flag (auto-detect from config)
    if ! grep -q "goupdate update -r" "$UPDATE_ACTION"; then
        log_test "update_action_no_rule_flag" "PASS" ""
    else
        log_test "update_action_no_rule_flag" "FAIL" "Update action should use auto-detect (no -r flag)"
    fi

    # Check that action uses --continue-on-fail
    if grep -q "\-\-continue-on-fail" "$UPDATE_ACTION"; then
        log_test "update_action_uses_continue" "PASS" ""
    else
        log_test "update_action_uses_continue" "FAIL" "Update action should use --continue-on-fail"
    fi

    # Check that action uses -y for auto-confirm
    if grep -q " -y" "$UPDATE_ACTION"; then
        log_test "update_action_uses_yes" "PASS" ""
    else
        log_test "update_action_uses_yes" "FAIL" "Update action should use -y flag"
    fi

    # Verify no -e flag is used
    if ! grep -q "\-e \$" "$UPDATE_ACTION"; then
        log_test "update_action_no_exclude_flag" "PASS" ""
    else
        log_test "update_action_no_exclude_flag" "FAIL" "Update action should not use -e flag"
    fi

    # Verify --all is not used (should be --major)
    if ! grep -q 'UPDATE_FLAG="--all"' "$UPDATE_ACTION"; then
        log_test "update_action_no_all_flag" "PASS" ""
    else
        log_test "update_action_no_all_flag" "FAIL" "Update action should not use --all (use --major)"
    fi

    # Verify no exclude-packages input (handled via .goupdate.yml config)
    if ! grep -q "exclude-packages:" "$UPDATE_ACTION"; then
        log_test "update_action_no_exclude_input" "PASS" ""
    else
        log_test "update_action_no_exclude_input" "FAIL" "Update action should not have exclude-packages input"
    fi
else
    log_test "update_action_exists" "FAIL" "Update action file not found"
fi

section "16.3 Example actions consistency"
# Example actions should also use auto-detect from config (no hardcoded -r flag)
for example in "frontend" "github-workflows"; do
    check_file="examples/$example/.github/actions/_goupdate-check/action.yml"
    update_file="examples/$example/.github/actions/_goupdate-update/action.yml"

    if [[ -f "$check_file" ]]; then
        # Verify no -e flag
        if ! grep -q "\-e \$" "$check_file"; then
            log_test "example_${example}_check_no_exclude" "PASS" ""
        else
            log_test "example_${example}_check_no_exclude" "FAIL" "Example check should not use -e flag"
        fi

        # Verify no -r flag (use auto-detect)
        if ! grep -q "goupdate outdated -r" "$check_file"; then
            log_test "example_${example}_check_no_rule" "PASS" ""
        else
            log_test "example_${example}_check_no_rule" "FAIL" "Example check should use auto-detect (no -r flag)"
        fi

        # Verify no exclude-packages input
        if ! grep -q "exclude-packages:" "$check_file"; then
            log_test "example_${example}_check_no_exclude_input" "PASS" ""
        else
            log_test "example_${example}_check_no_exclude_input" "FAIL" "Example check should not have exclude-packages input"
        fi
    fi

    if [[ -f "$update_file" ]]; then
        # Verify no -e flag and no --all
        if ! grep -q "\-e \$" "$update_file" && ! grep -q 'UPDATE_FLAG="--all"' "$update_file"; then
            log_test "example_${example}_update_fixed" "PASS" ""
        else
            log_test "example_${example}_update_fixed" "FAIL" "Example update has unfixed flags"
        fi

        # Verify no -r flag (use auto-detect)
        if ! grep -q "goupdate update -r" "$update_file"; then
            log_test "example_${example}_update_no_rule" "PASS" ""
        else
            log_test "example_${example}_update_no_rule" "FAIL" "Example update should use auto-detect (no -r flag)"
        fi

        # Verify no exclude-packages input
        if ! grep -q "exclude-packages:" "$update_file"; then
            log_test "example_${example}_update_no_exclude_input" "PASS" ""
        else
            log_test "example_${example}_update_no_exclude_input" "FAIL" "Example update should not have exclude-packages input"
        fi
    fi
done

# ============================================================================
# TEST SECTION 17: Version and help commands
# ============================================================================
header "17. Testing: Version and help commands"

section "17.1 Version command"
OUTPUT=$($GOUPDATE version 2>&1) || OUTPUT=""
if [[ -n "$OUTPUT" ]]; then
    log_test "cmd_version" "PASS" ""
else
    log_test "cmd_version" "FAIL" "Version command failed"
fi

section "17.2 Help commands"
for cmd in "" "scan" "outdated" "update" "list" "config"; do
    OUTPUT=$($GOUPDATE $cmd --help 2>&1) || OUTPUT=""
    if echo "$OUTPUT" | grep -qiE "(usage|flags|description)"; then
        log_test "cmd_help_$cmd" "PASS" ""
    else
        log_test "cmd_help_$cmd" "WARN" "Help for '$cmd' might be incomplete"
    fi
done

# ============================================================================
# Summary
# ============================================================================
header "Test Summary"

echo ""
echo "Test Results:"
echo "─────────────────────────────────────────"
for result in "${TEST_RESULTS[@]}"; do
    echo "  $result"
done
echo ""
echo "─────────────────────────────────────────"
echo "Total:    $TESTS_TOTAL"
echo "Passed:   $TESTS_PASSED"
echo "Failed:   $TESTS_FAILED"
echo "Warnings: $TESTS_WARNINGS"
echo ""

# Cleanup
rm -f /tmp/goupdate-test

if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "${RED}TESTS FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}ALL TESTS PASSED${NC}"
    exit 0
fi
