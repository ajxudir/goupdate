// Package cmd implements the command-line interface for goupdate.
// It provides commands for scanning, listing, checking outdated packages,
// and performing updates across multiple package managers.
package cmd

import (
	stderrors "errors"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

var exitFunc = os.Exit
var verboseFlag bool
var versionFlag bool
var skipBuildChecksFlag bool

var rootCmd = &cobra.Command{
	Use:   "goupdate",
	Short: "Multi-package manager dependency scanner and updater",
	Long:  `Scan, analyze, and update dependencies across multiple package managers.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verboseFlag {
			verbose.Enable()
		}
		// Show build warnings (arch mismatch, dev build) at the top of every command
		if !skipBuildChecksFlag {
			if warnings := GetBuildWarnings(); warnings != "" {
				fmt.Fprint(os.Stderr, warnings)
				fmt.Fprintln(os.Stderr) // Blank line to separate from command output
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			printVersionOutput()
			return
		}
		_ = cmd.Help()
	},
}

// Execute runs the root command and exits with appropriate code:
//   - 0: Success
//   - 1: Partial failure (some packages failed, use --continue-on-fail)
//   - 2: Complete failure
//   - 3: Configuration or validation error
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := errors.GetExitCode(err)

		// Check for partial success
		var partialErr *errors.PartialSuccessError
		if stderrors.As(err, &partialErr) {
			code = errors.ExitPartialFailure
			verbose.Infof("Exit code %d: partial success - %d succeeded, %d failed", code, partialErr.Succeeded, partialErr.Failed)
		} else {
			verbose.Infof("Exit code %d: %v", code, err)
		}

		exitFunc(code)
	}
}

// ExecuteTest runs the root command for testing (returns error instead of exiting).
//
// Unlike Execute(), this function returns the error directly without calling
// os.Exit, making it suitable for use in test suites.
//
// Returns:
//   - error: Command execution error, or nil on success
func ExecuteTest() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Enable verbose debug output")
	rootCmd.PersistentFlags().BoolVar(&skipBuildChecksFlag, "skip-build-checks", false, "Skip build validation warnings (dev build, arch mismatch)")

	// Add -v/--version as a LOCAL flag (not persistent) so it only works on root command
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Show version information")

	// Commands ordered logically: info → config → workflow (scan → list → outdated → update)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(outdatedCmd)
	rootCmd.AddCommand(updateCmd)
}

// printVersionOutput prints version, build, and runtime information to stdout.
//
// Output includes build target platform, runtime platform (if different),
// Go version, build date, git commit, and version string.
func printVersionOutput() {
	// Show build architecture (what binary was compiled for)
	buildOS, buildArch := getBuildTarget()
	fmt.Printf("  Build:   %s/%s\n", buildOS, buildArch)

	// Show runtime (what user is running on) only if different
	if buildOS != runtime.GOOS || buildArch != runtime.GOARCH {
		fmt.Printf("  Runtime: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("  Go:      %s\n", runtime.Version())
	if BuildTime != "" {
		fmt.Printf("  Date:    %s\n", BuildTime)
	}
	fmt.Println()
	if GitCommit != "" {
		fmt.Printf("  Git:     %s\n", GitCommit)
	}
	fmt.Printf("  Version: %s\n", Version)
}
