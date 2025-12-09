package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// TestPersistentPreRunVerbose tests the behavior of PersistentPreRun with verbose flag.
//
// It verifies:
//   - Verbose mode is enabled when verboseFlag is set to true
func TestPersistentPreRunVerbose(t *testing.T) {
	// Save and restore globals
	oldVerbose := verboseFlag
	oldArgs := os.Args
	defer func() {
		verboseFlag = oldVerbose
		os.Args = oldArgs
		verbose.Disable()
	}()

	// Set verbose flag to true to cover lines 19-21
	verboseFlag = true

	// Manually call PersistentPreRun to cover the verbose enable path
	rootCmd.PersistentPreRun(rootCmd, []string{})

	// Verify verbose was enabled
	assert.True(t, verbose.IsEnabled())
}

// TestPersistentPreRunNotVerbose tests the behavior of PersistentPreRun without verbose flag.
//
// It verifies:
//   - Verbose mode is not enabled when verboseFlag is false
func TestPersistentPreRunNotVerbose(t *testing.T) {
	// Save and restore globals
	oldVerbose := verboseFlag
	defer func() {
		verboseFlag = oldVerbose
		verbose.Disable()
	}()

	// Set verbose flag to false
	verboseFlag = false

	// Manually call PersistentPreRun
	rootCmd.PersistentPreRun(rootCmd, []string{})

	// Verify verbose was not enabled
	assert.False(t, verbose.IsEnabled())
}

// TestPersistentPreRunBuildWarnings tests the behavior of PersistentPreRun with build warnings.
//
// It verifies:
//   - Build warnings are shown when skipBuildChecksFlag is false
//   - Build warnings are skipped when skipBuildChecksFlag is true
func TestPersistentPreRunBuildWarnings(t *testing.T) {
	// Save and restore globals
	oldVersion := Version
	oldBuildOS := BuildOS
	oldBuildArch := BuildArch
	oldSkip := skipBuildChecksFlag
	defer func() {
		Version = oldVersion
		BuildOS = oldBuildOS
		BuildArch = oldBuildArch
		skipBuildChecksFlag = oldSkip
	}()

	t.Run("shows warnings when not skipped", func(t *testing.T) {
		Version = "dev"
		BuildOS = ""
		BuildArch = ""
		skipBuildChecksFlag = false

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		rootCmd.PersistentPreRun(rootCmd, []string{})

		_ = w.Close()
		os.Stderr = oldStderr

		var buf [1024]byte
		n, _ := r.Read(buf[:])
		output := string(buf[:n])

		assert.Contains(t, output, "Development build")
	})

	t.Run("skips warnings when flag set", func(t *testing.T) {
		Version = "dev"
		BuildOS = ""
		BuildArch = ""
		skipBuildChecksFlag = true

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		rootCmd.PersistentPreRun(rootCmd, []string{})

		_ = w.Close()
		os.Stderr = oldStderr

		var buf [1024]byte
		n, _ := r.Read(buf[:])
		output := string(buf[:n])

		assert.Empty(t, output)
	})
}

// TestPrintVersionOutput tests the behavior of printVersionOutput.
//
// It verifies:
//   - Version output displays all build information
//   - Runtime information is shown when build architecture differs
//   - Optional fields are omitted when empty
func TestPrintVersionOutput(t *testing.T) {
	// Save and restore globals
	oldVersion := Version
	oldBuildTime := BuildTime
	oldGitCommit := GitCommit
	oldBuildOS := BuildOS
	oldBuildArch := BuildArch
	defer func() {
		Version = oldVersion
		BuildTime = oldBuildTime
		GitCommit = oldGitCommit
		BuildOS = oldBuildOS
		BuildArch = oldBuildArch
	}()

	t.Run("outputs version info", func(t *testing.T) {
		Version = "1.2.3"
		BuildTime = "2025-01-01T00:00:00Z"
		GitCommit = "abc123"
		BuildOS = ""
		BuildArch = ""

		output := captureStdout(t, func() {
			printVersionOutput()
		})

		assert.Contains(t, output, "Version: 1.2.3")
		assert.Contains(t, output, "Date:    2025-01-01T00:00:00Z")
		assert.Contains(t, output, "Git:     abc123")
		assert.Contains(t, output, "Build:")
		assert.Contains(t, output, "Go:")
	})

	t.Run("shows runtime when arch differs", func(t *testing.T) {
		Version = "1.0.0"
		BuildTime = ""
		GitCommit = ""
		BuildOS = "impossible_os"
		BuildArch = "impossible_arch"

		output := captureStdout(t, func() {
			printVersionOutput()
		})

		assert.Contains(t, output, "Build:   impossible_os/impossible_arch")
		assert.Contains(t, output, "Runtime:")
	})

	t.Run("omits optional fields when empty", func(t *testing.T) {
		Version = "1.0.0"
		BuildTime = ""
		GitCommit = ""
		BuildOS = ""
		BuildArch = ""

		output := captureStdout(t, func() {
			printVersionOutput()
		})

		assert.NotContains(t, output, "Date:")
		assert.NotContains(t, output, "Git:")
	})
}
