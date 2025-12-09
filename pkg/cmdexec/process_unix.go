//go:build unix

package cmdexec

import (
	"os/exec"
	"syscall"
)

// setProcGroup configures the command to run in its own process group.
//
// On Unix systems, this sets the Setpgid flag which creates a new process group.
// This allows us to kill all child processes when the command times out by
// sending a signal to the entire process group.
//
// Parameters:
//   - cmd: The command to configure for process group execution
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killProcGroup kills the entire process group for the given process.
//
// On Unix systems, this sends SIGKILL to the entire process group (using negative PID).
// This ensures all child processes spawned by the command are terminated, preventing
// orphaned processes after a timeout.
//
// Parameters:
//   - cmd: The command whose process group should be killed
//
// Returns:
//   - error: Error if the kill operation fails, nil if successful or process is nil
func killProcGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Negative PID means kill the entire process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
