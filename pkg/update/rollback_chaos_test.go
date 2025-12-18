//go:build unix

package update

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/formats"
)

// =============================================================================
// CHAOS TESTS - ROLLBACK AND BACKUP/RESTORE
// =============================================================================
//
// These tests verify that rollback operations correctly restore both content
// AND permissions, and that the tests themselves don't produce false positives.
// =============================================================================

// TestChaos_RollbackRestoresPermissions verifies that when an update fails
// and rollback occurs, the file's permissions are restored along with content.
//
// This is critical because a permission change during failed update could:
// - Break application's ability to read/write the file
// - Create security vulnerabilities (e.g., world-readable secrets)
func TestChaos_RollbackRestoresPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	// Create manifest with specific permissions
	originalContent := `{"dependencies":{"demo":"^1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(originalContent), 0644)
	require.NoError(t, err)
	err = os.Chmod(manifestPath, 0750) // Unusual permissions to verify preservation
	require.NoError(t, err)

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"r": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "echo {{package}}"},
		},
	}}

	pkg := formats.Package{
		Name:        "demo",
		Rule:        "r",
		PackageType: "js",
		Type:        "prod",
		Constraint:  "^",
		Version:     "1.0.0",
		Source:      manifestPath,
	}

	// Make lock command fail to trigger rollback
	originalExec := execCommandFunc
	execCommandFunc = func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
		return nil, errors.New("lock failed - triggering rollback")
	}
	t.Cleanup(func() { execCommandFunc = originalExec })

	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, false)
	require.Error(t, err, "update should fail due to lock failure")

	// CRITICAL: Verify BOTH content AND permissions are restored
	restoredContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, string(restoredContent),
		"content should be restored after rollback")

	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), info.Mode().Perm(),
		"ROLLBACK BUG: permissions should be restored to 0750 after rollback")
}

// TestChaos_BackupCapturesPermissions verifies that backupFiles actually
// captures the file's permissions, not a default value.
func TestChaos_BackupCapturesPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different permissions
	testCases := []struct {
		name string
		mode os.FileMode
	}{
		{"standard", 0644},
		{"restrictive", 0600},
		{"executable", 0755},
		{"group_read", 0640},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(tmpDir, tc.name+".txt")
			err := os.WriteFile(file, []byte("content"), 0644)
			require.NoError(t, err)
			err = os.Chmod(file, tc.mode)
			require.NoError(t, err)

			backups, err := backupFiles([]string{file})
			require.NoError(t, err)
			require.Len(t, backups, 1)

			assert.Equal(t, tc.mode, backups[0].mode,
				"backup should capture actual mode %v, not default", tc.mode)
		})
	}
}

// TestChaos_RestoreChangesPermissions verifies that restoreBackups actually
// CHANGES permissions back to the backed-up value, not just preserves current.
//
// This catches the bug where restore might preserve current permissions.
func TestChaos_RestoreChangesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")

	// Create file with 0750 permissions
	err := os.WriteFile(file, []byte("original"), 0644)
	require.NoError(t, err)
	err = os.Chmod(file, 0750)
	require.NoError(t, err)

	// Backup
	backups, err := backupFiles([]string{file})
	require.NoError(t, err)

	// Modify file with DIFFERENT permissions
	err = os.WriteFile(file, []byte("modified"), 0644)
	require.NoError(t, err)
	err = os.Chmod(file, 0600) // Different from original
	require.NoError(t, err)

	// Verify permissions changed
	info, err := os.Stat(file)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm(),
		"precondition: permissions should be 0600 after modification")

	// Restore
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// CRITICAL: Verify permissions are CHANGED back to 0750
	info, err = os.Stat(file)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), info.Mode().Perm(),
		"restore should CHANGE permissions back to 0750, not keep current 0600")
}

// TestChaos_ConcurrentBackupRestore tests that backup/restore is safe for
// concurrent access to different files.
func TestChaos_ConcurrentBackupRestore(t *testing.T) {
	tmpDir := t.TempDir()
	numFiles := 10

	// Create test files
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('a'+i))+".txt")
		err := os.WriteFile(file, []byte("content "+string(rune('a'+i))), 0644)
		require.NoError(t, err)
		files[i] = file
	}

	// Concurrent backup
	var wg sync.WaitGroup
	backupsChan := make(chan []fileBackup, numFiles)
	errsChan := make(chan error, numFiles)

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			b, err := backupFiles([]string{f})
			if err != nil {
				errsChan <- err
				return
			}
			backupsChan <- b
		}(file)
	}

	wg.Wait()
	close(backupsChan)
	close(errsChan)

	// Check for errors
	for err := range errsChan {
		t.Fatalf("concurrent backup failed: %v", err)
	}

	// Collect all backups
	var allBackups []fileBackup
	for b := range backupsChan {
		allBackups = append(allBackups, b...)
	}

	assert.Len(t, allBackups, numFiles, "should have backed up all files")

	// Modify all files
	for _, file := range files {
		err := os.WriteFile(file, []byte("modified"), 0644)
		require.NoError(t, err)
	}

	// Concurrent restore
	var restoreWg sync.WaitGroup
	restoreErrsChan := make(chan []error, len(allBackups))

	for _, backup := range allBackups {
		restoreWg.Add(1)
		go func(b fileBackup) {
			defer restoreWg.Done()
			errs := restoreBackups([]fileBackup{b})
			if len(errs) > 0 {
				restoreErrsChan <- errs
			}
		}(backup)
	}

	restoreWg.Wait()
	close(restoreErrsChan)

	// Check for restore errors
	for errs := range restoreErrsChan {
		t.Fatalf("concurrent restore failed: %v", errs)
	}

	// Verify all files restored
	for i, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		expectedContent := "content " + string(rune('a'+i))
		assert.Equal(t, expectedContent, string(content),
			"file %s should be restored to original content", file)
	}
}

// TestChaos_PartialUpdateRecovery tests recovery when update succeeds for
// some packages but fails for others in a batch.
func TestChaos_PartialUpdateRecovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two manifest files
	manifest1 := filepath.Join(tmpDir, "package1.json")
	manifest2 := filepath.Join(tmpDir, "package2.json")

	content1 := `{"dependencies":{"pkg1":"1.0.0"}}`
	content2 := `{"dependencies":{"pkg2":"1.0.0"}}`

	err := os.WriteFile(manifest1, []byte(content1), 0644)
	require.NoError(t, err)
	err = os.WriteFile(manifest2, []byte(content2), 0644)
	require.NoError(t, err)

	// Backup both files
	backups, err := backupFiles([]string{manifest1, manifest2})
	require.NoError(t, err)
	require.Len(t, backups, 2)

	// Modify first file (simulating successful update)
	err = os.WriteFile(manifest1, []byte(`{"dependencies":{"pkg1":"2.0.0"}}`), 0644)
	require.NoError(t, err)

	// Second file update "fails" - we need to restore

	// Restore both files
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Both files should be restored to original
	restored1, _ := os.ReadFile(manifest1)
	restored2, _ := os.ReadFile(manifest2)

	assert.Equal(t, content1, string(restored1), "manifest1 should be restored")
	assert.Equal(t, content2, string(restored2), "manifest2 should be restored")
}

// TestChaos_LockFileBackupRestore tests that lock files are also properly
// backed up and restored during rollback.
func TestChaos_LockFileBackupRestore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manifest and lock file
	manifest := filepath.Join(tmpDir, "package.json")
	lockFile := filepath.Join(tmpDir, "package-lock.json")

	manifestContent := `{"dependencies":{"test":"1.0.0"}}`
	lockContent := `{"lockfileVersion":2,"packages":{"test":{"version":"1.0.0"}}}`

	err := os.WriteFile(manifest, []byte(manifestContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(lockFile, []byte(lockContent), 0644)
	require.NoError(t, err)

	// Backup both
	backups, err := backupFiles([]string{manifest, lockFile})
	require.NoError(t, err)
	require.Len(t, backups, 2)

	// Modify both (simulating update)
	err = os.WriteFile(manifest, []byte(`{"dependencies":{"test":"2.0.0"}}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(lockFile, []byte(`{"lockfileVersion":2,"packages":{"test":{"version":"2.0.0"}}}`), 0644)
	require.NoError(t, err)

	// Restore
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Verify both restored
	restoredManifest, _ := os.ReadFile(manifest)
	restoredLock, _ := os.ReadFile(lockFile)

	assert.Equal(t, manifestContent, string(restoredManifest))
	assert.Equal(t, lockContent, string(restoredLock))
}

// TestChaos_EmptyFileBackupRestore tests backup/restore of empty files.
func TestChaos_EmptyFileBackupRestore(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "empty.txt")

	// Create empty file
	err := os.WriteFile(file, []byte{}, 0644)
	require.NoError(t, err)

	// Backup
	backups, err := backupFiles([]string{file})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.Empty(t, backups[0].content, "backup of empty file should have empty content")

	// Modify
	err = os.WriteFile(file, []byte("not empty anymore"), 0644)
	require.NoError(t, err)

	// Restore
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Verify empty
	content, _ := os.ReadFile(file)
	assert.Empty(t, content, "restored file should be empty")
}

// TestChaos_LargeFileBackupRestore tests backup/restore of large files.
func TestChaos_LargeFileBackupRestore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "large.txt")

	// Create a 1MB file
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}

	err := os.WriteFile(file, largeContent, 0644)
	require.NoError(t, err)

	// Backup
	backups, err := backupFiles([]string{file})
	require.NoError(t, err)
	require.Len(t, backups, 1)
	assert.Equal(t, len(largeContent), len(backups[0].content))

	// Modify
	err = os.WriteFile(file, []byte("small now"), 0644)
	require.NoError(t, err)

	// Restore
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Verify
	restored, _ := os.ReadFile(file)
	assert.Equal(t, largeContent, restored)
}

// TestChaos_BackupNonExistentFileGraceful tests that backup handles missing
// files gracefully without failing the entire operation.
func TestChaos_BackupNonExistentFileGraceful(t *testing.T) {
	tmpDir := t.TempDir()

	existingFile := filepath.Join(tmpDir, "exists.txt")
	missingFile := filepath.Join(tmpDir, "missing.txt")

	err := os.WriteFile(existingFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Backup should succeed for existing file and skip missing
	backups, err := backupFiles([]string{existingFile, missingFile})
	require.NoError(t, err)
	assert.Len(t, backups, 1, "should backup only existing file")
	assert.Equal(t, existingFile, backups[0].path)
}

// TestChaos_RestoreToDeletedFile tests restore behavior when target file
// was deleted between backup and restore.
func TestChaos_RestoreToDeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "willbedeleted.txt")

	err := os.WriteFile(file, []byte("original"), 0644)
	require.NoError(t, err)

	backups, err := backupFiles([]string{file})
	require.NoError(t, err)

	// Delete the file
	err = os.Remove(file)
	require.NoError(t, err)

	// Restore should recreate the file
	errs := restoreBackups(backups)
	assert.Empty(t, errs)

	// Verify file exists with correct content
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, "original", string(content))
}

// TestChaos_UpdateFailurePreservesOriginal verifies that if an update fails
// at any point, the original file state is preserved.
func TestChaos_UpdateFailurePreservesOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	originalContent := `{"dependencies":{"test":"1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(originalContent), 0644)
	require.NoError(t, err)
	err = os.Chmod(manifestPath, 0755)
	require.NoError(t, err)

	cfg := &config.Config{Rules: map[string]config.PackageManagerCfg{
		"npm": {
			Format: "json",
			Fields: map[string]string{"dependencies": "prod"},
			Update: &config.UpdateCfg{Commands: "false"}, // Will fail
		},
	}}

	pkg := formats.Package{
		Name:    "test",
		Version: "1.0.0",
		Source:  manifestPath,
		Rule:    "npm",
		Type:    "prod",
	}

	// This should fail
	err = UpdatePackage(pkg, "2.0.0", cfg, tmpDir, false, false)
	require.Error(t, err)

	// Original should be preserved
	content, _ := os.ReadFile(manifestPath)
	assert.Equal(t, originalContent, string(content),
		"content should be preserved after failed update")

	info, _ := os.Stat(manifestPath)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm(),
		"permissions should be preserved after failed update")
}
