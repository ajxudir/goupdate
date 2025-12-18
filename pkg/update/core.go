// Package update provides functionality for updating package versions in manifest files.
// It supports atomic file writes with rollback on failure, file permission preservation,
// and format-specific version update strategies (JSON, YAML, XML, raw text).
package update

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/verbose"
	"github.com/ajxudir/goupdate/pkg/warnings"
)

var (
	readFileFunc              = os.ReadFile
	writeFileFunc             = writeFilePreservingPermissions
	updateDeclaredVersionFunc = updateDeclaredVersion
	statFileFunc              = os.Stat
)

// filePermissions stores file metadata for preservation
type filePermissions struct {
	mode os.FileMode
	path string
	uid  int
	gid  int
}

// getFilePermissions retrieves the current permissions and ownership of a file
func getFilePermissions(path string) (*filePermissions, error) {
	info, err := statFileFunc(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}
	uid, gid := getFileOwnership(info)
	return &filePermissions{
		mode: info.Mode(),
		path: path,
		uid:  uid,
		gid:  gid,
	}, nil
}

// generateTempSuffix creates a random suffix for temporary files
func generateTempSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple suffix if random fails
		return ".tmp"
	}
	return "." + hex.EncodeToString(b) + ".tmp"
}

// writeFileAtomic writes content to a file atomically using a temporary file and rename.
// This prevents corruption if the process is interrupted during write.
// NOTE: This function checks if the target file is writable before attempting the atomic
// write, because rename() is a directory operation and may bypass file permissions on
// some operating systems.
func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	// Check if target file exists and is writable before attempting atomic write
	// This catches read-only files early, since rename() may bypass file permissions
	if info, err := statFileFunc(path); err == nil {
		// File exists - check if it's writable
		if info.Mode().Perm()&0200 == 0 {
			// File is not writable by owner
			return fmt.Errorf("file is read-only: %s", path)
		}
	}

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Create temp file in the same directory to ensure atomic rename works
	tempPath := filepath.Join(dir, base+generateTempSuffix())

	// Write to temp file
	if err := os.WriteFile(tempPath, content, mode); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temp file on failure, log if cleanup fails
		if removeErr := os.Remove(tempPath); removeErr != nil {
			warnings.Warnf("Warning: failed to clean up temp file %s: %v\n", tempPath, removeErr)
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// writeFilePreservingPermissions writes content to a file while preserving its original permissions and ownership.
// If the file exists, its permissions and ownership are preserved. If it doesn't exist, defaultMode is used.
// Uses atomic write (temp file + rename) to prevent corruption on interruption.
// Warns the user if permissions or ownership change unexpectedly after the write operation.
func writeFilePreservingPermissions(path string, content []byte, defaultMode os.FileMode) error {
	// Get original permissions and ownership if file exists
	origPerms, err := getFilePermissions(path)
	mode := defaultMode

	if err == nil {
		mode = origPerms.mode.Perm() // Use only permission bits
	}

	// Use atomic write for safety
	if writeErr := writeFileAtomic(path, content, mode); writeErr != nil {
		return writeErr
	}

	// Restore ownership if we had the original info
	if origPerms != nil && origPerms.uid >= 0 && origPerms.gid >= 0 {
		if chownErr := chownFile(path, origPerms.uid, origPerms.gid); chownErr != nil {
			// Only warn, don't fail - chown may fail if not running as root
			verbose.Printf("Unable to preserve file ownership for %s: %v\n", path, chownErr)
		}
	}

	// Verify permissions after write
	newPerms, statErr := getFilePermissions(path)
	if statErr != nil {
		warnings.Warnf("Warning: unable to verify file permissions after write for %s: %v\n", path, statErr)
		return nil
	}

	// Check if permissions changed unexpectedly
	if origPerms != nil && newPerms.mode.Perm() != origPerms.mode.Perm() {
		warnings.Warnf("Warning: file permissions changed for %s: %v -> %v\n",
			path, origPerms.mode.Perm(), newPerms.mode.Perm())
	}

	// Check if ownership changed unexpectedly (only warn if we had valid original ownership)
	if origPerms != nil && origPerms.uid >= 0 && origPerms.gid >= 0 {
		if newPerms.uid != origPerms.uid || newPerms.gid != origPerms.gid {
			warnings.Warnf("Warning: file ownership changed for %s: %d:%d -> %d:%d\n",
				path, origPerms.uid, origPerms.gid, newPerms.uid, newPerms.gid)
		}
	}

	return nil
}

// RunGroupLockCommand runs the lock command once for a group of packages.
// This is used when packages are in a named group to run the lock command after all
// packages in the group have their declared versions updated.
// The withAllDeps parameter enables the -W flag for composer (or equivalent for other managers).
func RunGroupLockCommand(cfg *config.UpdateCfg, workDir string, withAllDeps bool) error {
	if cfg == nil {
		return fmt.Errorf("update configuration is required")
	}

	if strings.TrimSpace(cfg.Commands) == "" {
		return &errors.UnsupportedError{Reason: "no lock command configured"}
	}

	verbose.Printf("Lock command: running group lock (withAllDeps=%v)\n", withAllDeps)

	// Run lock command without package-specific replacements (group-level)
	_, err := execCommandFunc(cfg, "", "", "", workDir, withAllDeps)
	if err != nil {
		verbose.Printf("Lock command FAILED: %v\n", err)
	} else {
		verbose.Printf("Lock command completed successfully\n")
	}
	return err
}

// fileBackup stores the original content of a file for rollback
type fileBackup struct {
	path    string
	content []byte
	mode    os.FileMode
}

// backupFiles reads and stores the content of files for potential rollback.
// Uses atomic semantics: either all files are backed up successfully, or none are.
// This prevents partial backup state that could lead to inconsistent rollbacks.
func backupFiles(paths []string) ([]fileBackup, error) {
	// Pre-allocate with expected capacity for efficiency
	backups := make([]fileBackup, 0, len(paths))

	// Phase 1: Read all files into memory (validation phase)
	// If any file fails to read, we return early without partial state
	for _, path := range paths {
		content, err := readFileFunc(path)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist, skip backup but note it for potential cleanup
				continue
			}
			// Return error before any state is modified - atomic semantics
			return nil, fmt.Errorf("failed to backup %s (atomic backup aborted): %w", path, err)
		}

		// Get file permissions, use default if stat fails
		info, statErr := statFileFunc(path)
		mode := os.FileMode(0o644)
		if statErr != nil {
			warnings.Warnf("Warning: unable to stat %s for backup, using default permissions: %v\n", path, statErr)
		} else if info != nil {
			mode = info.Mode().Perm()
		}

		backups = append(backups, fileBackup{path: path, content: content, mode: mode})
	}

	return backups, nil
}

// writeFileWithBackupMode writes content to a file and forcefully sets the specified mode.
// Unlike writeFilePreservingPermissions, this function uses the provided mode instead of
// preserving the current file's permissions. Used for restore operations where we want
// to restore the original backed-up permissions.
func writeFileWithBackupMode(path string, content []byte, mode os.FileMode) error {
	// Get original ownership if file exists (to restore it after write)
	origPerms, _ := getFilePermissions(path)

	// Use atomic write
	if writeErr := writeFileAtomic(path, content, mode); writeErr != nil {
		return writeErr
	}

	// Restore ownership if we had the original info
	if origPerms != nil && origPerms.uid >= 0 && origPerms.gid >= 0 {
		if chownErr := chownFile(path, origPerms.uid, origPerms.gid); chownErr != nil {
			verbose.Printf("Unable to preserve file ownership for %s: %v\n", path, chownErr)
		}
	}

	return nil
}

// restoreBackups restores files from their backups after a failed update operation.
//
// It performs the following operations:
//   - Step 1: Iterate through all backups
//   - Step 2: Write each backup's content back to its original path with backed-up permissions
//   - Step 3: Collect any errors that occur during restoration
//   - Step 4: Log successful restorations
//
// Parameters:
//   - backups: Slice of file backups containing path, content, and permissions
//
// Returns:
//   - []error: Slice of errors encountered during restoration; returns empty slice if all restorations succeed
func restoreBackups(backups []fileBackup) []error {
	var errs []error
	for _, backup := range backups {
		// Use writeFileWithBackupMode to forcefully restore the backed-up permissions
		if err := writeFileWithBackupMode(backup.path, backup.content, backup.mode); err != nil {
			errs = append(errs, fmt.Errorf("failed to restore %s: %w", backup.path, err))
		} else {
			verbose.Printf("Restored %s from backup\n", backup.path)
		}
	}
	return errs
}

// getLockFilePaths returns the lock file paths for a package manager rule configuration.
//
// It performs the following operations:
//   - Step 1: Iterate through all lock file configurations in the rule
//   - Step 2: For each pattern, use glob matching to find matching files
//   - Step 3: Collect all matched file paths
//
// Parameters:
//   - ruleCfg: Package manager configuration containing lock file patterns
//   - scopeDir: Base directory to resolve lock file patterns from
//
// Returns:
//   - []string: Slice of absolute paths to lock files; returns empty slice if no lock files are configured or found
func getLockFilePaths(ruleCfg config.PackageManagerCfg, scopeDir string) []string {
	var paths []string
	for _, lockCfg := range ruleCfg.LockFiles {
		for _, pattern := range lockCfg.Files {
			// Try to find matching files
			matches, err := filepath.Glob(filepath.Join(scopeDir, pattern))
			if err == nil && len(matches) > 0 {
				paths = append(paths, matches...)
			}
		}
	}
	return paths
}

// UpdatePackage attempts to update a package to the provided target version.
// When dryRun is true, no files or lock commands are executed.
// Flow: 1) Backup manifest and lock files 2) Update declared version 3) Run lock command 4) Rollback on failure
func UpdatePackage(p formats.Package, target string, cfg *config.Config, workDir string, dryRun bool, skipLock bool) error {
	if cfg == nil {
		return fmt.Errorf("configuration is required")
	}

	effectiveCfg, err := ResolveUpdateCfg(p, cfg)
	if err != nil {
		return err
	}

	ruleCfg, ruleOk := cfg.Rules[p.Rule]
	if !ruleOk {
		return fmt.Errorf("rule configuration missing for %s", p.Rule)
	}

	scopeDir := workDir
	if p.Source != "" {
		scopeDir = filepath.Dir(p.Source)
	}
	if scopeDir == "" {
		scopeDir = cfg.WorkingDir
	}
	if scopeDir == "" {
		scopeDir = "."
	}

	// Read original manifest content for rollback if needed
	originalContent, readErr := readFileFunc(p.Source)
	if readErr != nil {
		return fmt.Errorf("failed to read %s: %w", p.Source, readErr)
	}

	// Backup lock files before update (for consistent rollback)
	var lockFileBackups []fileBackup
	if !dryRun && !skipLock {
		lockFilePaths := getLockFilePaths(ruleCfg, scopeDir)
		if len(lockFilePaths) > 0 {
			lockFileBackups, err = backupFiles(lockFilePaths)
			if err != nil {
				verbose.Printf("Warning: failed to backup lock files: %v\n", err)
				// Continue anyway - we'll still have the manifest backup
			} else {
				verbose.Printf("Backed up %d lock file(s) for rollback\n", len(lockFileBackups))
			}
		}
	}

	// Check if this package needs -W flag (with all dependencies)
	withAllDeps := ruleCfg.ShouldUpdateWithAllDependencies(p.Name)
	if withAllDeps {
		verbose.Printf("Package %s configured with with_all_dependencies\n", p.Name)
	}

	runLockCommand := func(version string) error {
		if strings.TrimSpace(effectiveCfg.Commands) == "" {
			return &errors.UnsupportedError{Reason: fmt.Sprintf("lock update missing for %s", p.Rule)}
		}

		verbose.Printf("Running lock command for %s in %s\n", p.Name, scopeDir)
		if _, err := execCommandFunc(effectiveCfg, p.Name, version, p.Constraint, scopeDir, withAllDeps); err != nil {
			verbose.Printf("Lock command failed for %s: %v\n", p.Name, err)
			return err
		}
		verbose.Printf("Lock command completed for %s\n", p.Name)

		return nil
	}

	// performRollback restores both manifest and lock files to their original state
	performRollback := func(originalErr error) error {
		verbose.Printf("Rolling back %s due to failure\n", p.Name)
		var rollbackErrs []error

		// Restore manifest file
		if restoreErr := writeFileFunc(p.Source, originalContent, 0o644); restoreErr != nil {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("manifest restore failed: %w", restoreErr))
		} else {
			verbose.Printf("Restored manifest %s\n", p.Source)
		}

		// Restore lock files from backup (if we have backups)
		if len(lockFileBackups) > 0 {
			restoreErrs := restoreBackups(lockFileBackups)
			rollbackErrs = append(rollbackErrs, restoreErrs...)
		}

		// Log any rollback errors but return the original error
		if len(rollbackErrs) > 0 {
			for _, re := range rollbackErrs {
				warnings.Warnf("Rollback warning: %v\n", re)
			}
		}

		return originalErr
	}

	verbose.Printf("Updating %s: %s -> %s (source: %s)\n", p.Name, p.Version, target, p.Source)

	// Step 1: Update declared version in manifest file
	applyErr := updateDeclaredVersionFunc(p, target, cfg, scopeDir, dryRun)
	if applyErr != nil {
		verbose.Printf("Failed to update declared version for %s: %v\n", p.Name, applyErr)
		return applyErr
	}
	verbose.Printf("Updated declared version for %s in manifest\n", p.Name)

	if dryRun || skipLock {
		verbose.Printf("Skipping lock command for %s (dryRun=%v, skipLock=%v)\n", p.Name, dryRun, skipLock)
		return nil
	}

	// Step 2: Run lock command to regenerate lock file
	if err := runLockCommand(target); err != nil {
		return performRollback(err)
	}

	verbose.Printf("Successfully updated %s to %s\n", p.Name, target)
	return nil
}

// updateDeclaredVersion updates the declared version of a package in its manifest file.
//
// It performs the following operations:
//   - Step 1: Validate rule configuration exists
//   - Step 2: Read current manifest file content
//   - Step 3: Get format-specific updater from registry
//   - Step 4: Apply version update using the updater
//   - Step 5: Write updated content back to file (unless dry run)
//
// Parameters:
//   - p: The package to update with source file and version information
//   - target: The target version to update to
//   - cfg: Global configuration containing rule definitions
//   - scopeDir: Base directory for scope-based updates (reserved for future use)
//   - dryRun: When true, skips writing changes to disk
//
// Returns:
//   - error: Returns error if rule configuration is missing, file read/write fails, or update fails; returns nil on success
func updateDeclaredVersion(p formats.Package, target string, cfg *config.Config, scopeDir string, dryRun bool) error {
	ruleCfg, ok := cfg.Rules[p.Rule]
	if !ok {
		return fmt.Errorf("rule configuration missing for %s", p.Rule)
	}

	// Capture file modification time before read for drift detection
	var readModTime int64
	if info, statErr := statFileFunc(p.Source); statErr == nil {
		readModTime = info.ModTime().UnixNano()
	}

	content, err := readFileFunc(p.Source)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", p.Source, err)
	}

	// Get updater from registry (supports extensibility for new formats)
	updater, err := getUpdaterForFormat(ruleCfg.Format)
	if err != nil {
		return err
	}

	updated, err := updater.UpdateVersion(content, p, ruleCfg, target)
	if err != nil {
		return err
	}

	// Preserve trailing newline from original file
	if len(content) > 0 && content[len(content)-1] == '\n' {
		if len(updated) == 0 || updated[len(updated)-1] != '\n' {
			updated = append(updated, '\n')
		}
	}

	if dryRun {
		return nil
	}

	// Check for file drift - another process may have modified the file
	if readModTime > 0 {
		if info, statErr := statFileFunc(p.Source); statErr == nil {
			if info.ModTime().UnixNano() != readModTime {
				warnings.Warnf("Warning: %s was modified by another process during update\n", p.Source)
				// Continue anyway - the atomic write will still work, but warn the user
			}
		}
	}

	if writeErr := writeFileFunc(p.Source, updated, 0o644); writeErr != nil {
		return fmt.Errorf("failed to write %s: %w", p.Source, writeErr)
	}

	_ = scopeDir // reserved for future scope-based updates
	return nil
}
