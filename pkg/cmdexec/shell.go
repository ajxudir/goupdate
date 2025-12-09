package cmdexec

// getDefaultShell returns the default shell for the system.
//
// This is the platform-specific fallback used when the SHELL environment
// variable is not set. On Unix systems, this returns "sh" with the "-c" flag.
//
// Returns:
//   - shell: The path to the default shell executable
//   - args: The shell arguments needed to execute a command string
func getDefaultShell() (shell string, args []string) {
	return "sh", []string{"-c"}
}
