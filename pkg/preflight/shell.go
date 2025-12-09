package preflight

import (
	"fmt"
	"os"
)

// getShellCommandCheck returns the shell and args for checking if a command exists.
//
// It performs the following operations:
//   - Reads the SHELL environment variable to determine the user's preferred shell
//   - Falls back to "sh" if SHELL is not set
//   - Constructs command arguments for a login shell (-l) executing 'command -v'
//
// The 'command -v' built-in is used because it detects executables, aliases, shell functions,
// and built-ins, providing comprehensive command detection across shell environments.
//
// Parameters:
//   - cmd: The command name to check for existence
//
// Returns:
//   - shell: The shell executable to use (from $SHELL env var or "sh" as fallback)
//   - args: Command arguments for checking command existence using 'command -v'
func getShellCommandCheck(cmd string) (shell string, args []string) {
	shell = os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	// Use 'command -v' which finds commands, aliases, and functions
	return shell, []string{"-l", "-c", fmt.Sprintf("command -v %s", cmd)}
}
