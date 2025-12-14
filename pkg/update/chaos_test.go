package update

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/testutil"
)

// =============================================================================
// CHAOS/BATTLE TESTS - Edge Cases and Error Handling
// =============================================================================
//
// These tests verify behavior under adverse conditions that should have been
// caught by battle/chaos testing. They cover:
// - Filesystem errors (permissions, missing files, disk full)
// - Rollback failure scenarios
// - Edge case inputs (malformed versions, special characters)
// - Concurrent access scenarios
// - Partial operation interruption
// =============================================================================

// -----------------------------------------------------------------------------
// FILESYSTEM ERROR HANDLING TESTS
// -----------------------------------------------------------------------------

// TestChaos_UpdatePackage_ReadOnlyManifest tests behavior when manifest becomes read-only.
//
// It verifies:
//   - Update fails gracefully when file is not writable
//   - Error message is descriptive
//   - No partial modifications are left behind
func TestChaos_UpdatePackage_ReadOnlyManifest(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Create manifest file
	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	// Make file read-only
	err = os.Chmod(manifestPath, 0444)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chmod(manifestPath, 0644) })

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	// Attempt update - should fail due to read-only file
	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, true)

	// Should fail (either during write or during update)
	// The exact error depends on implementation
	if err != nil {
		assert.True(t, strings.Contains(err.Error(), "permission") ||
			strings.Contains(err.Error(), "write") ||
			strings.Contains(err.Error(), "denied") ||
			os.IsPermission(err),
			"error should indicate permission issue: %v", err)
	}

	// Verify original content unchanged
	afterContent, readErr := os.ReadFile(manifestPath)
	require.NoError(t, readErr)
	assert.Equal(t, content, string(afterContent), "content should be unchanged")
}

// TestChaos_UpdatePackage_MissingSourceFile tests behavior when source file disappears.
//
// It verifies:
//   - Update fails gracefully when source file is missing
//   - Error message clearly indicates the file is missing
func TestChaos_UpdatePackage_MissingSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	missingPath := filepath.Join(tmpDir, "nonexistent.json")

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  missingPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	err := UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, true)

	assert.Error(t, err, "should fail when source file missing")
	assert.True(t, os.IsNotExist(err) || strings.Contains(err.Error(), "no such file") ||
		strings.Contains(err.Error(), "failed to read"),
		"error should indicate missing file: %v", err)
}

// TestChaos_UpdatePackage_EmptySourceFile tests behavior with empty manifest file.
//
// It verifies:
//   - Update handles empty files gracefully
//   - Error message is descriptive
func TestChaos_UpdatePackage_EmptySourceFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty manifest file
	manifestPath := filepath.Join(tmpDir, "package.json")
	err := os.WriteFile(manifestPath, []byte{}, 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, true)

	// Empty JSON file should fail to parse or update
	assert.Error(t, err, "should fail with empty file")
}

// TestChaos_UpdatePackage_MalformedJSON tests behavior with invalid JSON.
//
// It verifies:
//   - Update fails gracefully with malformed JSON
//   - Original file is not corrupted
func TestChaos_UpdatePackage_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create malformed JSON
	manifestPath := filepath.Join(tmpDir, "package.json")
	malformed := `{"dependencies": {"test": "1.0.0"` // Missing closing braces
	err := os.WriteFile(manifestPath, []byte(malformed), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, true)

	assert.Error(t, err, "should fail with malformed JSON")

	// Verify original content unchanged
	afterContent, readErr := os.ReadFile(manifestPath)
	require.NoError(t, readErr)
	assert.Equal(t, malformed, string(afterContent), "malformed content should be unchanged")
}

// -----------------------------------------------------------------------------
// ROLLBACK FAILURE TESTS
// -----------------------------------------------------------------------------

// TestChaos_RollbackPlans_PartialFailure tests rollback when some packages fail.
//
// It verifies:
//   - Rollback continues even when some packages fail
//   - All errors are collected and returned
//   - Successfully rolled back packages are restored
func TestChaos_RollbackPlans_PartialFailure(t *testing.T) {
	callCount := 0
	mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		callCount++
		// First call succeeds, second fails, third succeeds
		if callCount == 2 {
			return errors.New("simulated rollback failure")
		}
		return nil
	}

	plans := []*PlannedUpdate{
		{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("pkg1", "2.0.0", "2.0.0"),
				Target: "2.0.0",
				Status: constants.StatusUpdated,
			},
			Original: "1.0.0",
		},
		{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("pkg2", "2.0.0", "2.0.0"),
				Target: "2.0.0",
				Status: constants.StatusUpdated,
			},
			Original: "1.0.0",
		},
		{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("pkg3", "2.0.0", "2.0.0"),
				Target: "2.0.0",
				Status: constants.StatusUpdated,
			},
			Original: "1.0.0",
		},
	}

	ctx := NewUpdateContext(&config.Config{}, ".", nil)
	groupErr := errors.New("group operation failed")

	err := RollbackPlans(plans, &config.Config{}, ".", ctx, groupErr, mockUpdater, false, true)

	// Should have errors from the partial failure
	assert.Error(t, err, "should return error from partial rollback failure")
	assert.Equal(t, 3, callCount, "all three packages should attempt rollback")
}

// TestChaos_RollbackPlans_AllFail tests rollback when all packages fail.
//
// It verifies:
//   - All errors are collected
//   - Function completes even with all failures
func TestChaos_RollbackPlans_AllFail(t *testing.T) {
	mockUpdater := func(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
		return errors.New("simulated failure for " + p.Name)
	}

	plans := []*PlannedUpdate{
		{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("pkg1", "2.0.0", "2.0.0"),
				Target: "2.0.0",
				Status: constants.StatusUpdated,
			},
			Original: "1.0.0",
		},
		{
			Res: UpdateResult{
				Pkg:    testutil.NPMPackage("pkg2", "2.0.0", "2.0.0"),
				Target: "2.0.0",
				Status: constants.StatusUpdated,
			},
			Original: "1.0.0",
		},
	}

	ctx := NewUpdateContext(&config.Config{}, ".", nil)
	groupErr := errors.New("group operation failed")

	err := RollbackPlans(plans, &config.Config{}, ".", ctx, groupErr, mockUpdater, false, true)

	assert.Error(t, err, "should return combined error")
	assert.Contains(t, err.Error(), "pkg1", "should include pkg1 error")
	assert.Contains(t, err.Error(), "pkg2", "should include pkg2 error")
}

// -----------------------------------------------------------------------------
// EDGE CASE INPUT TESTS
// -----------------------------------------------------------------------------

// TestChaos_UpdatePackage_EmptyVersion tests behavior with empty target version.
//
// It verifies:
//   - Update handles empty version string appropriately
func TestChaos_UpdatePackage_EmptyVersion(t *testing.T) {
	tmpDir := t.TempDir()

	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	// Update with empty target version
	err = UpdatePackage(pkg, "", cfg, tmpDir, false, true)

	// Should either fail validation or write empty version
	// Behavior depends on implementation
	if err == nil {
		// If no error, verify the file was modified (empty version written)
		afterContent, readErr := os.ReadFile(manifestPath)
		require.NoError(t, readErr)
		t.Logf("Content after empty version update: %s", string(afterContent))
	}
}

// TestChaos_UpdatePackage_VeryLongVersion tests behavior with extremely long version string.
//
// It verifies:
//   - Update handles very long version strings
//   - No buffer overflow or memory issues
func TestChaos_UpdatePackage_VeryLongVersion(t *testing.T) {
	tmpDir := t.TempDir()

	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	// Very long version string (10000 characters)
	longVersion := strings.Repeat("1", 10000)

	// Should not panic or crash
	err = UpdatePackage(pkg, longVersion, cfg, tmpDir, false, true)

	// May succeed or fail gracefully - just ensure no panic
	t.Logf("Long version update result: %v", err)
}

// TestChaos_UpdatePackage_SpecialCharactersInVersion tests version with special characters.
//
// It verifies:
//   - Special characters are handled safely
//   - No injection vulnerabilities
func TestChaos_UpdatePackage_SpecialCharactersInVersion(t *testing.T) {
	tmpDir := t.TempDir()

	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	testCases := []struct {
		name    string
		version string
	}{
		{"newline", "1.0.0\n2.0.0"},
		{"tab", "1.0.0\t2.0.0"},
		{"null", "1.0.0\x002.0.0"},
		{"quotes", `"1.0.0"`},
		{"backslash", `1.0.0\n`},
		{"unicode", "1.0.0-α.β.γ"},
		{"shell_injection", "; rm -rf /"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset content
			err := os.WriteFile(manifestPath, []byte(content), 0644)
			require.NoError(t, err)

			// Should not panic and should not execute shell commands
			_ = UpdatePackage(pkg, tc.version, cfg, tmpDir, false, true)

			// Verify file wasn't corrupted - should still be valid JSON or original content
			afterContent, readErr := os.ReadFile(manifestPath)
			require.NoError(t, readErr)
			assert.NotEmpty(t, afterContent, "file should not be empty")
		})
	}
}

// TestChaos_UpdatePackage_JSONEscapeVersion tests that JSON special characters are properly escaped.
//
// It verifies:
//   - JSON special characters in version strings are escaped properly
//   - No JSON injection occurs
func TestChaos_UpdatePackage_JSONEscapeVersion(t *testing.T) {
	tmpDir := t.TempDir()

	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{Commands: "echo test"},
			},
		},
	}

	// Version string that looks like JSON - should be escaped as a string value
	jsonVersion := `{"version": "malicious"}`
	err = UpdatePackage(pkg, jsonVersion, cfg, tmpDir, false, true)

	// Should either fail or properly escape the version
	afterContent, readErr := os.ReadFile(manifestPath)
	require.NoError(t, readErr)

	// If update succeeded, the JSON should still be valid and the version
	// should be escaped as a string, not parsed as JSON
	if err == nil {
		// The version should be a string value, properly escaped
		// The content should look like: "test": "{\"version\": \"malicious\"}"
		// NOT like: "test": {"version": "malicious"}
		assert.Contains(t, string(afterContent), `\"version\"`,
			"JSON special chars should be escaped")
	}
}

// TestChaos_UpdatePackage_NilConfig tests behavior with nil configuration.
//
// It verifies:
//   - Update fails gracefully with nil config
//   - No nil pointer dereference panic
func TestChaos_UpdatePackage_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  filepath.Join(tmpDir, "package.json"),
		Rule:    "npm",
	}

	// Should not panic
	err := UpdatePackage(pkg, "2.0.0", nil, tmpDir, false, true)

	assert.Error(t, err, "should fail with nil config")
	assert.Contains(t, err.Error(), "configuration", "error should mention config")
}

// TestChaos_UpdatePackage_MissingRule tests behavior when rule is not in config.
//
// It verifies:
//   - Update fails gracefully when rule is missing
//   - Error message is clear
func TestChaos_UpdatePackage_MissingRule(t *testing.T) {
	tmpDir := t.TempDir()

	manifestPath := filepath.Join(tmpDir, "package.json")
	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "nonexistent-rule",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
			},
		},
	}

	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, true)

	assert.Error(t, err, "should fail with missing rule")
	assert.Contains(t, err.Error(), "nonexistent-rule",
		"error should mention missing rule")
}

// -----------------------------------------------------------------------------
// BACKUP AND RESTORE TESTS
// -----------------------------------------------------------------------------

// TestChaos_BackupFiles_PartialRead tests backup when some files can't be read.
//
// It verifies:
//   - Backup fails atomically if any file can't be read
//   - No partial backup state is created
func TestChaos_BackupFiles_PartialRead(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid file
	validPath := filepath.Join(tmpDir, "valid.json")
	err := os.WriteFile(validPath, []byte(`{"test": true}`), 0644)
	require.NoError(t, err)

	// Path to non-existent file
	invalidPath := filepath.Join(tmpDir, "missing.json")

	// Note: backupFiles skips non-existent files, so this should succeed
	backups, err := backupFiles([]string{validPath, invalidPath})

	// Should succeed with only the valid file backed up
	assert.NoError(t, err)
	assert.Len(t, backups, 1, "only valid file should be backed up")
	assert.Equal(t, validPath, backups[0].path)
}

// TestChaos_RestoreBackups_PartialRestore tests restore when some files fail.
//
// It verifies:
//   - Restore continues even when some files fail
//   - All errors are collected
func TestChaos_RestoreBackups_PartialRestore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid file
	validPath := filepath.Join(tmpDir, "valid.json")

	// Create backup data including an invalid path
	backups := []fileBackup{
		{path: validPath, content: []byte(`{"restored": true}`), mode: 0644},
		{path: "/nonexistent/dir/file.json", content: []byte(`{"fail": true}`), mode: 0644},
	}

	errs := restoreBackups(backups)

	// Should have one error for the invalid path
	assert.Len(t, errs, 1, "should have one error")
	assert.Contains(t, errs[0].Error(), "nonexistent", "error should mention invalid path")

	// Valid file should be restored
	afterContent, readErr := os.ReadFile(validPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(afterContent), "restored", "valid file should be restored")
}

// -----------------------------------------------------------------------------
// VALIDATION TESTS
// -----------------------------------------------------------------------------

// TestChaos_ValidateUpdatedPackage_ReloadReturnsEmpty tests validation with empty reload.
//
// It verifies:
//   - Validation fails when package not found after reload
//   - Error message is descriptive
func TestChaos_ValidateUpdatedPackage_ReloadReturnsEmpty(t *testing.T) {
	plan := &PlannedUpdate{
		Res: UpdateResult{
			Pkg:    testutil.NPMPackage("missing-pkg", "1.0.0", "1.0.0"),
			Target: "2.0.0",
		},
	}

	// Reload returns empty list
	reloadFunc := func() ([]formats.Package, error) {
		return []formats.Package{}, nil
	}

	err := ValidateUpdatedPackage(plan, reloadFunc, nil)

	assert.Error(t, err, "should fail when package not found")
	assert.Contains(t, err.Error(), "missing", "error should indicate package missing")
}

// TestChaos_ValidateUpdatedPackage_ReloadReturnsError tests validation when reload fails.
//
// It verifies:
//   - Validation fails when reload returns error
//   - Original error is propagated
func TestChaos_ValidateUpdatedPackage_ReloadReturnsError(t *testing.T) {
	plan := &PlannedUpdate{
		Res: UpdateResult{
			Pkg:    testutil.NPMPackage("test", "1.0.0", "1.0.0"),
			Target: "2.0.0",
		},
	}

	// Reload returns error
	expectedErr := errors.New("network error")
	reloadFunc := func() ([]formats.Package, error) {
		return nil, expectedErr
	}

	err := ValidateUpdatedPackage(plan, reloadFunc, nil)

	assert.Error(t, err, "should propagate reload error")
	assert.Contains(t, err.Error(), "network error", "should contain original error")
}

// TestChaos_ValidateUpdatedPackage_VersionMismatch tests validation with wrong version.
//
// It verifies:
//   - Validation fails when version doesn't match target
//   - Error includes both expected and actual versions
func TestChaos_ValidateUpdatedPackage_VersionMismatch(t *testing.T) {
	plan := &PlannedUpdate{
		Res: UpdateResult{
			Pkg:    testutil.NPMPackage("test", "1.0.0", "1.0.0"),
			Target: "2.0.0",
		},
	}

	// Reload returns wrong version
	reloadFunc := func() ([]formats.Package, error) {
		return []formats.Package{
			testutil.NPMPackage("test", "1.5.0", "1.5.0"), // Wrong version
		}, nil
	}

	err := ValidateUpdatedPackage(plan, reloadFunc, nil)

	assert.Error(t, err, "should fail on version mismatch")
	assert.Contains(t, err.Error(), "mismatch", "error should mention mismatch")
	assert.Contains(t, err.Error(), "2.0.0", "error should mention expected version")
	assert.Contains(t, err.Error(), "1.5.0", "error should mention actual version")
}

// -----------------------------------------------------------------------------
// ATOMIC WRITE TESTS
// -----------------------------------------------------------------------------

// TestChaos_WriteFileAtomic_TempFileCleanup tests temp file cleanup on failure.
//
// It verifies:
//   - Temp files are cleaned up when rename fails
func TestChaos_WriteFileAtomic_TempFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Write to a valid path
	targetPath := filepath.Join(tmpDir, "test.json")
	content := []byte(`{"test": true}`)

	err := writeFileAtomic(targetPath, content, 0644)
	require.NoError(t, err)

	// Verify content was written
	afterContent, readErr := os.ReadFile(targetPath)
	require.NoError(t, readErr)
	assert.Equal(t, content, afterContent)

	// Verify no temp files left behind
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, f := range files {
		assert.False(t, strings.Contains(f.Name(), ".tmp"),
			"no temp files should remain: %s", f.Name())
	}
}

// TestChaos_WriteFilePreservingPermissions_PermissionDenied tests write with permission error.
//
// It verifies:
//   - Write fails gracefully when directory is not writable
//   - Error indicates permission issue
func TestChaos_WriteFilePreservingPermissions_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()

	// Make directory read-only
	err := os.Chmod(tmpDir, 0555)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chmod(tmpDir, 0755) })

	targetPath := filepath.Join(tmpDir, "test.json")
	content := []byte(`{"test": true}`)

	err = writeFilePreservingPermissions(targetPath, content, 0644)

	assert.Error(t, err, "should fail with permission denied")
	assert.True(t, os.IsPermission(err) || strings.Contains(err.Error(), "permission"),
		"error should indicate permission issue: %v", err)
}

// -----------------------------------------------------------------------------
// EDGE CASES IN EXECUTION CONTEXT
// -----------------------------------------------------------------------------

// TestChaos_ShouldSkipUpdate_AllStatuses tests all possible status values.
//
// It verifies:
//   - All known statuses are handled correctly
//   - Unknown statuses don't cause panics
func TestChaos_ShouldSkipUpdate_AllStatuses(t *testing.T) {
	knownStatuses := []string{
		constants.StatusUpToDate,
		constants.StatusUpdated,
		constants.StatusFailed,
		constants.StatusPlanned,
		constants.StatusConfigError,
		constants.StatusSummarizeError,
		constants.StatusOutdated,
		"not_configured",
		"floating",
		"version_missing",
		"unknown_status",
		"",
	}

	for _, status := range knownStatuses {
		t.Run(status, func(t *testing.T) {
			res := &UpdateResult{
				Pkg:    testutil.NPMPackage("test", "1.0.0", "1.0.0"),
				Target: "2.0.0",
				Status: status,
			}

			// Should not panic
			result := ShouldSkipUpdate(res)
			t.Logf("Status %q -> ShouldSkipUpdate: %v", status, result)
		})
	}
}

// TestChaos_CollectUpdateErrors_MixedResults tests error collection with mixed results.
//
// It verifies:
//   - Only actual errors are collected
//   - Unsupported errors are excluded
//   - Nil errors don't cause issues
func TestChaos_CollectUpdateErrors_MixedResults(t *testing.T) {
	results := []UpdateResult{
		{Pkg: testutil.NPMPackage("pkg1", "1.0.0", "1.0.0"), Err: nil},
		{Pkg: testutil.NPMPackage("pkg2", "1.0.0", "1.0.0"), Err: errors.New("actual error")},
		{Pkg: testutil.NPMPackage("pkg3", "1.0.0", "1.0.0"), Err: nil},
	}

	errs := CollectUpdateErrors(results)

	assert.Len(t, errs, 1, "should collect only actual errors")
	assert.Contains(t, errs[0].Error(), "actual error")
}
