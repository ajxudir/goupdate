//go:build unix

package update

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// =============================================================================
// CHAOS TESTS - PERMISSIONS AND OWNERSHIP
// =============================================================================
//
// These tests verify that the permission/ownership tests themselves are not
// producing false positives. They do this by:
// 1. Testing with explicit chmod to avoid umask issues
// 2. Testing negative scenarios to ensure failures are caught
// 3. Verifying content actually changes (not just permissions)
// 4. Testing edge cases that could mask bugs
// =============================================================================

// TestChaos_PermissionsActuallyDifferent verifies that different permission modes
// are actually distinguishable by the test infrastructure.
//
// This chaos test prevents false positives by ensuring:
//   - Different modes produce different stat results
//   - Umask doesn't mask the differences
func TestChaos_PermissionsActuallyDifferent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with different permissions
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	err := os.WriteFile(file1, []byte("content"), 0644)
	require.NoError(t, err)
	err = os.Chmod(file1, 0644)
	require.NoError(t, err)

	err = os.WriteFile(file2, []byte("content"), 0755)
	require.NoError(t, err)
	err = os.Chmod(file2, 0755)
	require.NoError(t, err)

	// Verify they are actually different
	info1, err := os.Stat(file1)
	require.NoError(t, err)
	info2, err := os.Stat(file2)
	require.NoError(t, err)

	assert.NotEqual(t, info1.Mode().Perm(), info2.Mode().Perm(),
		"CHAOS TEST FAILED: Different permission modes should be distinguishable")
}

// TestChaos_UmaskAffectsWriteFile demonstrates that os.WriteFile is affected by umask.
//
// This test proves that explicit chmod is necessary after WriteFile.
func TestChaos_UmaskAffectsWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write file requesting 0777 permissions
	err := os.WriteFile(testFile, []byte("content"), 0777)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)

	// The actual permissions may differ from 0777 due to umask
	// This test documents the behavior - it's not a failure
	if info.Mode().Perm() != 0777 {
		t.Logf("INFO: os.WriteFile(path, content, 0777) created file with mode %v due to umask",
			info.Mode().Perm())
	}

	// Now set exact permissions with Chmod
	err = os.Chmod(testFile, 0777)
	require.NoError(t, err)

	info, err = os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0777), info.Mode().Perm(),
		"os.Chmod should set exact permissions regardless of umask")
}

// TestChaos_VerifyContentActuallyChanges ensures the update actually modifies file content.
//
// Prevents false positive where permissions are "preserved" because nothing happened.
func TestChaos_VerifyContentActuallyChanges(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	originalContent := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(originalContent), 0644)
	require.NoError(t, err)
	err = os.Chmod(manifestPath, 0755) // Explicit chmod
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
		Type:    "prod",
	}

	cfg := &config.Config{
		Rules: map[string]config.PackageManagerCfg{
			"npm": {
				Format: "json",
				Fields: map[string]string{"dependencies": "prod"},
				Update: &config.UpdateCfg{},
			},
		},
	}

	// Perform update
	err = updateDeclaredVersion(pkg, "2.0.0", cfg, tmpDir, false)
	require.NoError(t, err)

	// CRITICAL: Verify content actually changed
	updatedContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	assert.NotEqual(t, originalContent, string(updatedContent),
		"CHAOS TEST FAILED: Content should have changed after update")
	assert.Contains(t, string(updatedContent), "2.0.0",
		"CHAOS TEST FAILED: Updated content should contain new version")
	assert.NotContains(t, string(updatedContent), `"test": "1.0.0"`,
		"CHAOS TEST FAILED: Updated content should NOT contain old version")
}

// TestChaos_PermissionsNotPreservedWithoutFeature tests that if we delete and
// recreate a file (simulating a naive implementation), permissions DO change.
//
// This validates that our test would catch a bug if the feature were broken.
// NOTE: os.WriteFile only sets permissions when CREATING a new file, not when
// truncating an existing one. So we must delete first to simulate what happens
// with atomic write (temp file + rename) if we don't restore permissions.
// NOTE: This test is skipped when running as root because root can bypass
// permission restrictions and the behavior differs.
func TestChaos_PermissionsNotPreservedWithoutFeature(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: root user has different permission behavior")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with 0700 permissions
	err := os.WriteFile(testFile, []byte("original"), 0644)
	require.NoError(t, err)
	err = os.Chmod(testFile, 0700)
	require.NoError(t, err)

	// Verify initial permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	// Simulate atomic write WITHOUT permission preservation:
	// Delete and recreate (this is what would happen with temp+rename
	// if we didn't explicitly restore the mode after rename)
	err = os.Remove(testFile)
	require.NoError(t, err)
	err = os.WriteFile(testFile, []byte("updated"), 0644)
	require.NoError(t, err)

	// Permissions should have changed to 0644 (or umask-modified version)
	info, err = os.Stat(testFile)
	require.NoError(t, err)

	// The key assertion: permissions should NOT be 0700 anymore
	// (they will be 0644 minus umask, but definitely not 0700)
	assert.NotEqual(t, os.FileMode(0700), info.Mode().Perm(),
		"CHAOS TEST FAILED: Delete+recreate should change permissions, proving our test can detect the difference")
}

// TestChaos_PreservationFeatureWorks tests that writeFilePreservingPermissions
// actually preserves permissions (positive control).
func TestChaos_PreservationFeatureWorks(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with 0700 permissions
	err := os.WriteFile(testFile, []byte("original"), 0644)
	require.NoError(t, err)
	err = os.Chmod(testFile, 0700)
	require.NoError(t, err)

	// Verify initial permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0700), info.Mode().Perm(), "precondition: initial mode should be 0700")

	// Write using preservation function
	err = writeFilePreservingPermissions(testFile, []byte("updated"), 0644)
	require.NoError(t, err)

	// Permissions should be preserved as 0700
	info, err = os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm(),
		"writeFilePreservingPermissions should preserve original permissions")

	// Verify content was updated
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "updated", string(content), "content should be updated")
}

// TestChaos_OwnershipActuallyExtracted verifies that getFileOwnership returns
// valid uid/gid values.
func TestChaos_OwnershipActuallyExtracted(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	info, err := os.Stat(testFile)
	require.NoError(t, err)

	uid, gid := getFileOwnership(info)

	// uid and gid should be valid (non-negative)
	assert.GreaterOrEqual(t, uid, 0, "uid should be valid")
	assert.GreaterOrEqual(t, gid, 0, "gid should be valid")

	// Verify against syscall directly
	stat := info.Sys().(*syscall.Stat_t)
	assert.Equal(t, int(stat.Uid), uid, "uid should match syscall")
	assert.Equal(t, int(stat.Gid), gid, "gid should match syscall")
}

// TestChaos_OwnershipPreservationWithDifferentOwner tests ownership preservation
// when the file is owned by a different user (requires root).
func TestChaos_OwnershipPreservationWithDifferentOwner(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping ownership change test: requires root privileges")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file
	err := os.WriteFile(testFile, []byte("original"), 0644)
	require.NoError(t, err)

	// Change to a different owner (nobody user is typically uid 65534)
	// Use uid 1000 as a more common non-root user
	testUid := 1000
	testGid := 1000

	err = os.Chown(testFile, testUid, testGid)
	if err != nil {
		t.Skipf("cannot change ownership to %d:%d: %v", testUid, testGid, err)
	}

	// Verify ownership changed
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	stat := info.Sys().(*syscall.Stat_t)
	require.Equal(t, uint32(testUid), stat.Uid, "precondition: uid should be changed")
	require.Equal(t, uint32(testGid), stat.Gid, "precondition: gid should be changed")

	// Write using preservation function
	err = writeFilePreservingPermissions(testFile, []byte("updated"), 0644)
	require.NoError(t, err)

	// Verify ownership is preserved
	info, err = os.Stat(testFile)
	require.NoError(t, err)
	stat = info.Sys().(*syscall.Stat_t)

	assert.Equal(t, uint32(testUid), stat.Uid, "uid should be preserved after update")
	assert.Equal(t, uint32(testGid), stat.Gid, "gid should be preserved after update")

	// Verify content was updated
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "updated", string(content))
}

// TestChaos_BackupCapturesCorrectMode verifies that backup captures the actual
// mode, not a default.
func TestChaos_BackupCapturesCorrectMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with unusual permissions
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)
	err = os.Chmod(testFile, 0751) // Unusual mode
	require.NoError(t, err)

	// Backup
	backups, err := backupFiles([]string{testFile})
	require.NoError(t, err)
	require.Len(t, backups, 1)

	// Verify backup captured the unusual mode, not a default
	assert.Equal(t, os.FileMode(0751), backups[0].mode,
		"backup should capture actual mode (0751), not default")
	assert.NotEqual(t, os.FileMode(0644), backups[0].mode,
		"backup should NOT have default mode 0644")
}

// TestChaos_RestoreActuallyRestoresPermissions verifies that restore changes
// permissions back from a different state.
func TestChaos_RestoreActuallyRestoresPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create file with 0750 permissions
	err := os.WriteFile(testFile, []byte("original"), 0644)
	require.NoError(t, err)
	err = os.Chmod(testFile, 0750)
	require.NoError(t, err)

	// Backup
	backups, err := backupFiles([]string{testFile})
	require.NoError(t, err)

	// Change the file completely (different content AND permissions)
	err = os.WriteFile(testFile, []byte("modified"), 0644)
	require.NoError(t, err)
	err = os.Chmod(testFile, 0600)
	require.NoError(t, err)

	// Verify the file changed
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(), "precondition: mode should be 0600")

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	require.Equal(t, "modified", string(content), "precondition: content should be modified")

	// Restore
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Verify restore worked - BOTH content AND permissions should be restored
	info, err = os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), info.Mode().Perm(),
		"restore should bring back original permissions 0750")

	content, err = os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "original", string(content),
		"restore should bring back original content")
}

// TestChaos_AtomicWriteDoesntLeavePartialFile tests that atomic write doesn't
// leave temp files on error.
// NOTE: This test is skipped when running as root because root can write
// to read-only directories.
func TestChaos_AtomicWriteDoesntLeavePartialFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: root user can write to read-only directories")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create original file
	err := os.WriteFile(testFile, []byte("original"), 0644)
	require.NoError(t, err)

	// Try to write to a read-only directory (should fail)
	roDir := filepath.Join(tmpDir, "readonly")
	err = os.Mkdir(roDir, 0755)
	require.NoError(t, err)

	roFile := filepath.Join(roDir, "test.txt")
	err = os.WriteFile(roFile, []byte("original"), 0644)
	require.NoError(t, err)

	// Make directory read-only
	err = os.Chmod(roDir, 0555)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	// Attempt write should fail
	err = writeFileAtomic(roFile, []byte("updated"), 0644)
	assert.Error(t, err, "write to read-only dir should fail")

	// Verify no temp files left behind
	entries, err := os.ReadDir(roDir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp",
			"no temp files should be left in directory")
	}
}

// TestChaos_PermissionsNotHardcoded verifies that different initial permissions
// result in different preserved permissions.
func TestChaos_PermissionsNotHardcoded(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		mode     os.FileMode
		expected os.FileMode
	}{
		{0600, 0600},
		{0644, 0644},
		{0700, 0700},
		{0755, 0755},
		{0640, 0640},
	}

	for _, tc := range testCases {
		t.Run(tc.mode.String(), func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test_"+tc.mode.String()+".txt")

			// Create with specific permissions
			err := os.WriteFile(testFile, []byte("original"), 0644)
			require.NoError(t, err)
			err = os.Chmod(testFile, tc.mode)
			require.NoError(t, err)

			// Write with preservation
			err = writeFilePreservingPermissions(testFile, []byte("updated"), 0777)
			require.NoError(t, err)

			// Verify preserved mode matches expected
			info, err := os.Stat(testFile)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, info.Mode().Perm(),
				"mode %v should be preserved, not hardcoded", tc.mode)
		})
	}
}
