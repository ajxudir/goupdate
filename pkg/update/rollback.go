package update

import (
	"errors"
	"os"
)

// rollbackOnFailure attempts to restore a file to its original content when errors occur.
//
// It performs the following operations:
//   - Step 1: Check if original content exists (skip if nil)
//   - Step 2: Retrieve original file permissions or use default 0644
//   - Step 3: Write original content back to the file
//   - Step 4: Append write errors to the error list if rollback fails
//
// Parameters:
//   - path: The file path to restore
//   - original: The original file content to restore, or nil to skip rollback
//   - errs: Existing errors to include in the combined result
//
// Returns:
//   - error: Combined error from all operations; returns joined errors if rollback fails, original errors if rollback succeeds
func rollbackOnFailure(path string, original []byte, errs []error) error {
	if original == nil {
		return errors.Join(errs...)
	}

	// Preserve original file permissions if possible, default to 0644 for new files
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	if writeErr := os.WriteFile(path, original, mode); writeErr != nil {
		errs = append(errs, writeErr)
	}

	return errors.Join(errs...)
}
