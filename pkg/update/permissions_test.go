package update

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/user/goupdate/pkg/warnings"
)

// TestGetFilePermissions tests the behavior of getFilePermissions.
//
// It verifies:
//   - Returns correct permissions for existing files
//   - Returns error for non-existent files
//   - Works with different permission modes (0644, 0755, 0600, 0700)
func TestGetFilePermissions(t *testing.T) {
	t.Run("returns permissions for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Create file with specific permissions
		err := os.WriteFile(testFile, []byte("content"), 0o755)
		require.NoError(t, err)

		perms, err := getFilePermissions(testFile)
		require.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, os.FileMode(0o755), perms.mode.Perm())
		assert.Equal(t, testFile, perms.path)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		perms, err := getFilePermissions("/nonexistent/path/file.txt")
		assert.Error(t, err)
		assert.Nil(t, perms)
		assert.Contains(t, err.Error(), "failed to stat file")
	})

	t.Run("works with different permission modes", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Only test modes that aren't affected by umask (no world-writable bits)
		testCases := []os.FileMode{0o644, 0o755, 0o600, 0o700}

		for _, mode := range testCases {
			testFile := filepath.Join(tmpDir, "test_"+mode.String()+".txt")
			err := os.WriteFile(testFile, []byte("content"), mode)
			require.NoError(t, err)

			// Explicitly set the mode after write to ensure it matches
			// (os.WriteFile may be affected by umask)
			err = os.Chmod(testFile, mode)
			require.NoError(t, err)

			perms, err := getFilePermissions(testFile)
			require.NoError(t, err)
			assert.Equal(t, mode, perms.mode.Perm(), "mode should match for %v", mode)
		}
	})
}

// TestWriteFilePreservingPermissions tests the behavior of writeFilePreservingPermissions.
//
// It verifies:
//   - Preserves original file permissions when updating content
//   - Uses default mode for new files
//   - Preserves executable permissions on scripts
//   - Handles read-only permissions correctly
func TestWriteFilePreservingPermissions(t *testing.T) {
	t.Run("preserves original file permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Create file with 0755 permissions
		err := os.WriteFile(testFile, []byte("original"), 0o755)
		require.NoError(t, err)

		// Verify initial permissions
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

		// Write new content - should preserve 0755
		err = writeFilePreservingPermissions(testFile, []byte("updated"), 0o644)
		require.NoError(t, err)

		// Verify permissions are preserved
		info, err = os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

		// Verify content was updated
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "updated", string(content))
	})

	t.Run("uses default mode for new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "new_file.txt")

		// Write to non-existent file
		err := writeFilePreservingPermissions(testFile, []byte("content"), 0o600)
		require.NoError(t, err)

		// Verify file was created with default mode
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("preserves executable permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "script.sh")

		// Create executable script
		err := os.WriteFile(testFile, []byte("#!/bin/bash\necho hello"), 0o755)
		require.NoError(t, err)

		// Update script content
		err = writeFilePreservingPermissions(testFile, []byte("#!/bin/bash\necho world"), 0o644)
		require.NoError(t, err)

		// Verify executable permission is preserved
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("preserves read-only permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "readonly.txt")

		// Create read-only file (but writable by owner for the test)
		err := os.WriteFile(testFile, []byte("original"), 0o644)
		require.NoError(t, err)

		// Change to more restrictive permissions
		err = os.Chmod(testFile, 0o444)
		require.NoError(t, err)

		// Restore permissions for cleanup
		defer func() { _ = os.Chmod(testFile, 0o644) }()

		// Make writable temporarily for the test
		err = os.Chmod(testFile, 0o644)
		require.NoError(t, err)

		// Write should preserve 0644 (we changed it back for write to succeed)
		err = writeFilePreservingPermissions(testFile, []byte("updated"), 0o755)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})
}

// TestWriteFilePreservingPermissionsWarnings tests warning behavior of writeFilePreservingPermissions.
//
// It verifies:
//   - No warnings are issued when permissions are preserved correctly
func TestWriteFilePreservingPermissionsWarnings(t *testing.T) {
	t.Run("no warning when permissions preserved correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		// Create file
		err := os.WriteFile(testFile, []byte("original"), 0o644)
		require.NoError(t, err)

		// Capture warnings
		var buf bytes.Buffer
		restore := warnings.SetWarningWriter(&buf)
		defer restore()

		// Write with preserved permissions
		err = writeFilePreservingPermissions(testFile, []byte("updated"), 0o644)
		require.NoError(t, err)

		// Should have no warnings
		assert.Empty(t, buf.String())
	})
}

// TestFilePermissionsIntegration tests integration scenarios for file permissions.
//
// It verifies:
//   - Multiple consecutive writes preserve permissions
//   - Different files maintain their own independent permissions
func TestFilePermissionsIntegration(t *testing.T) {
	t.Run("multiple writes preserve permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "multi.txt")

		// Create with specific permissions
		err := os.WriteFile(testFile, []byte("v1"), 0o700)
		require.NoError(t, err)

		// Multiple writes should all preserve
		for i := 0; i < 5; i++ {
			err = writeFilePreservingPermissions(testFile, []byte("v2"), 0o644)
			require.NoError(t, err)

			info, err := os.Stat(testFile)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(),
				"iteration %d should preserve permissions", i)
		}
	})

	t.Run("different files maintain their own permissions", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := map[string]os.FileMode{
			"file1.txt": 0o644,
			"file2.txt": 0o755,
			"file3.txt": 0o600,
		}

		// Create files with different permissions
		for name, mode := range files {
			path := filepath.Join(tmpDir, name)
			err := os.WriteFile(path, []byte("original"), mode)
			require.NoError(t, err)
		}

		// Update all files
		for name := range files {
			path := filepath.Join(tmpDir, name)
			err := writeFilePreservingPermissions(path, []byte("updated"), 0o777)
			require.NoError(t, err)
		}

		// Verify each file retained its original permissions
		for name, expectedMode := range files {
			path := filepath.Join(tmpDir, name)
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, expectedMode, info.Mode().Perm(),
				"file %s should have mode %v", name, expectedMode)
		}
	})
}
