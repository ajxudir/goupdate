//go:build windows

package cmdexec

import (
	"os/exec"
)

// setProcGroup is a no-op on Windows.
//
// Windows handles process groups differently through job objects, and the
// exec.CommandContext already handles process termination adequately on Windows.
// This function exists to maintain API compatibility with Unix systems.
//
// Parameters:
//   - cmd: The command to configure (no-op on Windows)
func setProcGroup(cmd *exec.Cmd) {
	// No-op on Windows - exec.CommandContext handles this
}

// killProcGroup kills the process on Windows.
//
// On Windows, killing the parent process typically terminates children, and
// exec.CommandContext handles cleanup adequately. This function calls Process.Kill()
// on the command's process.
//
// Parameters:
//   - cmd: The command whose process should be killed
//
// Returns:
//   - error: Error if the kill operation fails, nil if successful or process is nil
func killProcGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
