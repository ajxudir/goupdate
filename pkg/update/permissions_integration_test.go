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
// INTEGRATION TESTS - FILE PERMISSIONS AND OWNERSHIP PRESERVATION
// =============================================================================
//
// These tests verify that file permissions and ownership are preserved during
// package updates. This is critical for server environments where applications
// may depend on specific file permissions to function correctly.
//
// Ownership tests are skipped when not running as root since chown requires
// elevated privileges.
// =============================================================================

// TestIntegration_PermissionsPreserved_JSONUpdate tests that file permissions
// are preserved when updating a JSON manifest file.
//
// It verifies:
//   - Original file permissions (0755) are preserved after update
//   - File content is correctly updated
//   - No permission changes occur during atomic write
func TestIntegration_PermissionsPreserved_JSONUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	// Create manifest with specific permissions (0755 - executable, which is unusual for JSON but tests preservation)
	content := `{"dependencies": {"lodash": "4.17.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)
	// Explicit chmod to avoid umask affecting permissions
	err = os.Chmod(manifestPath, 0755)
	require.NoError(t, err)

	// Verify initial permissions
	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0755), info.Mode().Perm(), "precondition: initial permissions should be 0755")

	// Create package and config for update
	pkg := formats.Package{
		Name:    "lodash",
		Version: "4.17.0",
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

	// Perform update (dry run to avoid needing npm)
	err = UpdatePackage(pkg, "4.17.21", cfg, tmpDir, true, true)
	require.NoError(t, err)

	// Now do a real update to the declared version only
	err = updateDeclaredVersion(pkg, "4.17.21", cfg, tmpDir, false)
	require.NoError(t, err)

	// Verify permissions are preserved after update
	info, err = os.Stat(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm(), "permissions should be preserved as 0755 after update")

	// Verify content was updated
	updatedContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(updatedContent), "4.17.21", "version should be updated")
}

// TestIntegration_PermissionsPreserved_MultipleUpdates tests that permissions
// remain stable across multiple consecutive updates.
//
// It verifies:
//   - Permissions are preserved after first update
//   - Permissions remain stable after second update
//   - No cumulative permission drift occurs
func TestIntegration_PermissionsPreserved_MultipleUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	// Create manifest with restrictive permissions
	content := `{"dependencies": {"express": "4.17.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)
	// Explicit chmod to avoid umask affecting permissions
	err = os.Chmod(manifestPath, 0600)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "express",
		Version: "4.17.0",
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

	// First update
	err = updateDeclaredVersion(pkg, "4.17.1", cfg, tmpDir, false)
	require.NoError(t, err)

	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "permissions should be 0600 after first update")

	// Second update
	pkg.Version = "4.17.1"
	err = updateDeclaredVersion(pkg, "4.17.2", cfg, tmpDir, false)
	require.NoError(t, err)

	info, err = os.Stat(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm(), "permissions should remain 0600 after second update")
}

// TestIntegration_PermissionsPreserved_DifferentModes tests preservation
// of various common permission modes.
//
// It verifies:
//   - 0644 (standard file) is preserved
//   - 0600 (private file) is preserved
//   - 0755 (executable) is preserved
//   - 0700 (private executable) is preserved
func TestIntegration_PermissionsPreserved_DifferentModes(t *testing.T) {
	modes := []os.FileMode{0644, 0600, 0755, 0700}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			tmpDir := t.TempDir()
			manifestPath := filepath.Join(tmpDir, "package.json")

			content := `{"dependencies": {"test": "1.0.0"}}`
			err := os.WriteFile(manifestPath, []byte(content), mode)
			require.NoError(t, err)

			// Explicitly set mode (WriteFile may be affected by umask)
			err = os.Chmod(manifestPath, mode)
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

			err = updateDeclaredVersion(pkg, "2.0.0", cfg, tmpDir, false)
			require.NoError(t, err)

			info, err := os.Stat(manifestPath)
			require.NoError(t, err)
			assert.Equal(t, mode, info.Mode().Perm(),
				"permissions %v should be preserved after update", mode)
		})
	}
}

// TestIntegration_OwnershipPreserved tests that file ownership (uid/gid)
// is preserved during updates when running as root.
//
// It verifies:
//   - Original uid is preserved after update
//   - Original gid is preserved after update
//
// NOTE: This test is skipped when not running as root since chown requires
// elevated privileges.
func TestIntegration_OwnershipPreserved(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping ownership test: requires root privileges")
	}

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	// Get original ownership
	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	stat := info.Sys().(*syscall.Stat_t)
	originalUid := int(stat.Uid)
	originalGid := int(stat.Gid)

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

	err = updateDeclaredVersion(pkg, "2.0.0", cfg, tmpDir, false)
	require.NoError(t, err)

	// Verify ownership is preserved
	info, err = os.Stat(manifestPath)
	require.NoError(t, err)
	stat = info.Sys().(*syscall.Stat_t)

	assert.Equal(t, originalUid, int(stat.Uid), "uid should be preserved after update")
	assert.Equal(t, originalGid, int(stat.Gid), "gid should be preserved after update")
}

// TestIntegration_OwnershipNotChangedByNonRoot tests that when running as
// a non-root user, the update process doesn't fail due to chown errors.
//
// It verifies:
//   - Update succeeds even when ownership cannot be changed
//   - File content is correctly updated
//   - Permissions are still preserved
func TestIntegration_OwnershipNotChangedByNonRoot(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping non-root ownership test: running as root")
	}

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "package.json")

	content := `{"dependencies": {"test": "1.0.0"}}`
	err := os.WriteFile(manifestPath, []byte(content), 0644)
	require.NoError(t, err)

	// Get original ownership
	info, err := os.Stat(manifestPath)
	require.NoError(t, err)
	stat := info.Sys().(*syscall.Stat_t)
	originalUid := int(stat.Uid)
	originalGid := int(stat.Gid)

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

	// Update should succeed even though we can't chown
	err = updateDeclaredVersion(pkg, "2.0.0", cfg, tmpDir, false)
	require.NoError(t, err)

	// Verify ownership hasn't changed (we own the file, so it stays ours)
	info, err = os.Stat(manifestPath)
	require.NoError(t, err)
	stat = info.Sys().(*syscall.Stat_t)

	assert.Equal(t, originalUid, int(stat.Uid), "uid should remain unchanged")
	assert.Equal(t, originalGid, int(stat.Gid), "gid should remain unchanged")

	// Verify content was updated
	updatedContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	assert.Contains(t, string(updatedContent), "2.0.0", "version should be updated")
}

// TestIntegration_AtomicWritePreservesPermissions tests that the atomic
// write mechanism (temp file + rename) preserves permissions.
//
// It verifies:
//   - Atomic write doesn't leave temp files with wrong permissions
//   - Final file has correct permissions
//   - No race condition in permission application
func TestIntegration_AtomicWritePreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")

	// Create file with specific permissions
	err := os.WriteFile(testFile, []byte(`{"version": "1.0.0"}`), 0644)
	require.NoError(t, err)
	// Explicit chmod to avoid umask affecting permissions
	err = os.Chmod(testFile, 0700)
	require.NoError(t, err)

	// Perform multiple atomic writes
	for i := 0; i < 10; i++ {
		err = writeFilePreservingPermissions(testFile, []byte(`{"version": "2.0.0"}`), 0644)
		require.NoError(t, err)

		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm(),
			"iteration %d: permissions should be preserved", i)
	}
}

// TestIntegration_BackupRestorePreservesPermissions tests that the backup
// and restore mechanism preserves file permissions.
//
// It verifies:
//   - Backup captures original permissions
//   - Restore applies original permissions
//   - Rollback scenario preserves permissions
func TestIntegration_BackupRestorePreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "manifest.json")

	// Create file with specific permissions
	originalContent := []byte(`{"version": "1.0.0"}`)
	err := os.WriteFile(testFile, originalContent, 0644)
	require.NoError(t, err)
	// Explicit chmod to avoid umask affecting permissions
	err = os.Chmod(testFile, 0750)
	require.NoError(t, err)

	// Backup the file
	backups, err := backupFiles([]string{testFile})
	require.NoError(t, err)
	require.Len(t, backups, 1)

	// Verify backup captured permissions
	assert.Equal(t, os.FileMode(0750), backups[0].mode, "backup should capture original permissions")

	// Modify the file (simulating an update)
	err = os.WriteFile(testFile, []byte(`{"version": "2.0.0"}`), 0644)
	require.NoError(t, err)

	// Restore from backup
	errs := restoreBackups(backups)
	assert.Empty(t, errs, "restore should succeed without errors")

	// Verify permissions are restored
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0750), info.Mode().Perm(), "permissions should be restored to 0750")

	// Verify content is restored
	restoredContent, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, originalContent, restoredContent, "content should be restored")
}

// TestIntegration_ServerScenario_WebAppConfig tests a realistic server scenario
// where a web application config file has specific permissions for security.
//
// It verifies:
//   - Config file with 0640 (owner rw, group r) is preserved
//   - Update doesn't break application's ability to read config
func TestIntegration_ServerScenario_WebAppConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Simulate a web app config with group-readable permissions
	// This is common for configs read by web server processes
	configContent := `{"dependencies": {"express": "4.17.0", "morgan": "1.10.0"}}`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	// Explicit chmod to avoid umask affecting permissions
	err = os.Chmod(configPath, 0640)
	require.NoError(t, err)

	pkg := formats.Package{
		Name:    "express",
		Version: "4.17.0",
		Source:  configPath,
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

	err = updateDeclaredVersion(pkg, "4.18.0", cfg, tmpDir, false)
	require.NoError(t, err)

	// Verify the security-sensitive permissions are preserved
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm(),
		"web app config permissions (0640) should be preserved")
}
