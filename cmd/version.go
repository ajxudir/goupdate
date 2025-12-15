package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/spf13/cobra"
)

// Version information set at build time via ldflags.
// Example: go build -ldflags="-X github.com/ajxudir/goupdate/cmd.Version=1.0.0"
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// BuildTime is the timestamp of the build.
	BuildTime = ""
	// GitCommit is the git commit hash of the build.
	GitCommit = ""
	// BuildOS is the target OS the binary was built for.
	BuildOS = ""
	// BuildArch is the target architecture the binary was built for.
	BuildArch = ""
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Long:  `Show version, build date, and system information.`,
	Run:   runVersion,
}

// Note: versionCmd is added to rootCmd in root.go's init() to control command order

// runVersion executes the version command to display build and version information.
//
// Outputs the build target platform, runtime platform (if different), Go version,
// build date, git commit hash, and semantic version to stdout.
func runVersion(cmd *cobra.Command, args []string) {
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

// GetVersion returns the current version string.
//
// Returns the semantic version set at build time, or "dev" for development builds.
//
// Returns:
//   - string: Version string (e.g., "1.0.0", "dev", "_stage-20240101-rc1")
func GetVersion() string {
	return Version
}

// getBuildTarget returns the OS and architecture the binary was built for.
//
// Falls back to runtime values if build-time values weren't set (dev builds).
// This allows the application to gracefully handle both release and dev builds.
//
// Returns:
//   - string: Target operating system (e.g., "linux", "darwin", "windows")
//   - string: Target architecture (e.g., "amd64", "arm64")
func getBuildTarget() (string, string) {
	buildOS := BuildOS
	buildArch := BuildArch

	// Fall back to runtime values for dev builds where ldflags weren't set
	if buildOS == "" {
		buildOS = runtime.GOOS
	}
	if buildArch == "" {
		buildArch = runtime.GOARCH
	}

	return buildOS, buildArch
}

// HasArchMismatch returns true if the binary was built for a different
// OS or architecture than what it's running on.
//
// This detects cross-compilation scenarios where a user might be running
// a binary intended for a different platform, which could cause issues.
//
// Returns:
//   - bool: true if build target differs from runtime platform; false otherwise
func HasArchMismatch() bool {
	// If build values aren't set (dev build), no mismatch
	if BuildOS == "" && BuildArch == "" {
		return false
	}

	buildOS, buildArch := getBuildTarget()
	return buildOS != runtime.GOOS || buildArch != runtime.GOARCH
}

// GetArchMismatchWarning returns a warning message if there's an architecture
// mismatch, or an empty string if everything matches.
//
// The warning advises users to download the correct binary for their platform
// to avoid unexpected behavior.
//
// Returns:
//   - string: Warning message if mismatch exists; empty string if platforms match
func GetArchMismatchWarning() string {
	if !HasArchMismatch() {
		return ""
	}

	buildOS, buildArch := getBuildTarget()
	return fmt.Sprintf("%s  Architecture mismatch: binary built for %s/%s but running on %s/%s\n"+
		"   This may cause unexpected behavior. Please download the correct binary.\n",
		constants.IconWarn, buildOS, buildArch, runtime.GOOS, runtime.GOARCH)
}

// IsDevBuild returns true if this is a development build (no release tag).
//
// Development builds have the default "dev" version string, indicating
// they were built without release ldflags.
//
// Returns:
//   - bool: true if Version equals "dev"; false for tagged releases
func IsDevBuild() bool {
	return Version == "dev"
}

// IsPrerelease returns true if this is a prerelease/release candidate version.
//
// Prerelease versions follow the format: _stage-YYYYMMDD-rcN and are built
// from the stage branch for testing before stable releases.
//
// Returns:
//   - bool: true if Version starts with "_stage-"; false otherwise
func IsPrerelease() bool {
	return strings.HasPrefix(Version, "_stage-")
}

// GetDevBuildWarning returns a warning message if running a dev build,
// or an empty string if running a released version.
//
// The warning advises users that dev builds are unreleased versions
// and recommends installing a stable release for production use.
//
// Returns:
//   - string: Warning message for dev builds; empty string for releases
func GetDevBuildWarning() string {
	if !IsDevBuild() {
		return ""
	}

	return constants.IconWarn + "  Development build: this is an unreleased version without a version tag.\n" +
		"   For production use, please install a released version.\n"
}

// GetPrereleaseWarning returns a warning message if running a prerelease version,
// or an empty string if running a stable release.
//
// The warning advises users that stage builds are release candidates
// not intended for production use.
//
// Returns:
//   - string: Warning message for prereleases; empty string for stable releases
func GetPrereleaseWarning() string {
	if !IsPrerelease() {
		return ""
	}

	return constants.IconWarn + "  Staging build: " + Version + "\n" +
		"   This is a release candidate from the stage branch.\n" +
		"   Not intended for production. Install a stable release (vX.Y.Z) instead.\n"
}

// GetBuildWarnings returns all build-related warnings combined.
//
// Aggregates warnings from architecture mismatch, dev builds, and
// prerelease versions into a single string.
//
// Returns:
//   - string: Combined warning messages; empty string if no warnings
func GetBuildWarnings() string {
	var warnings string

	if w := GetArchMismatchWarning(); w != "" {
		warnings += w
	}

	if w := GetDevBuildWarning(); w != "" {
		warnings += w
	}

	if w := GetPrereleaseWarning(); w != "" {
		warnings += w
	}

	return warnings
}
