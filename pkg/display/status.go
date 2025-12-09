package display

import (
	"fmt"
	"strings"

	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/lock"
)

// Status constants re-exported for convenience.
const (
	// StatusUpToDate indicates the package is already at the target version.
	StatusUpToDate = constants.StatusUpToDate

	// StatusUpdated indicates the package was successfully updated.
	StatusUpdated = constants.StatusUpdated

	// StatusPlanned indicates the package update is planned (dry-run mode).
	StatusPlanned = constants.StatusPlanned

	// StatusFailed indicates the update operation failed.
	StatusFailed = constants.StatusFailed

	// StatusConfigError indicates a configuration error prevented the update.
	StatusConfigError = constants.StatusConfigError

	// StatusOutdated indicates newer versions are available for the package.
	StatusOutdated = constants.StatusOutdated
)

// Icon constants re-exported for convenience.
const (
	// IconSuccess indicates a successful or positive state.
	IconSuccess = constants.IconSuccess

	// IconWarning indicates a warning or caution state.
	IconWarning = constants.IconWarning

	// IconError indicates an error or failed state.
	IconError = constants.IconError

	// IconInfo indicates informational or neutral state.
	IconInfo = constants.IconInfo

	// IconPending indicates a pending or planned state.
	IconPending = constants.IconPending

	// IconNotConfigured indicates unconfigured state.
	IconNotConfigured = constants.IconNotConfigured

	// IconBlocked indicates a blocked or unsupported state.
	IconBlocked = constants.IconBlocked

	// IconWarn is the warning prefix for messages.
	IconWarn = constants.IconWarn
)

// FormatStatus formats a status string with the appropriate icon.
//
// Parameters:
//   - status: The status string (e.g., "Updated", "Failed", "Planned")
//
// Returns:
//   - string: Formatted status with icon prefix (e.g., "🟢 Updated")
//
// Example:
//
//	display.FormatStatus("Updated")   // Returns "🟢 Updated"
//	display.FormatStatus("Failed")    // Returns "❌ Failed"
//	display.FormatStatus("Planned")   // Returns "🟡 Planned"
func FormatStatus(status string) string {
	switch status {
	case constants.StatusUpdated:
		return fmt.Sprintf("%s %s", constants.IconSuccess, constants.StatusUpdated)
	case constants.StatusPlanned:
		return fmt.Sprintf("%s %s", constants.IconPending, constants.StatusPlanned)
	case constants.StatusUpToDate:
		return fmt.Sprintf("%s %s", constants.IconSuccess, constants.StatusUpToDate)
	case constants.StatusFailed:
		return fmt.Sprintf("%s %s", constants.IconError, constants.StatusFailed)
	case constants.StatusOutdated:
		return fmt.Sprintf("%s %s", constants.IconWarning, constants.StatusOutdated)
	case lock.InstallStatusNotConfigured:
		return fmt.Sprintf("%s %s", constants.IconNotConfigured, lock.InstallStatusNotConfigured)
	case lock.InstallStatusFloating:
		return fmt.Sprintf("%s %s", constants.IconBlocked, lock.InstallStatusFloating)
	case constants.StatusConfigError:
		return fmt.Sprintf("%s %s", constants.IconError, constants.StatusConfigError)
	case constants.StatusSummarizeError:
		return fmt.Sprintf("%s %s", constants.IconError, constants.StatusSummarizeError)
	default:
		return status
	}
}

// StatusIcon returns the icon for a given status.
//
// Parameters:
//   - status: The status string
//
// Returns:
//   - string: The icon for this status, or empty string if unknown
//
// Example:
//
//	display.StatusIcon("Updated")  // Returns "🟢"
//	display.StatusIcon("Failed")   // Returns "❌"
func StatusIcon(status string) string {
	switch status {
	case constants.StatusUpdated, constants.StatusUpToDate:
		return constants.IconSuccess
	case constants.StatusPlanned:
		return constants.IconPending
	case constants.StatusFailed, constants.StatusConfigError, constants.StatusSummarizeError:
		return constants.IconError
	case constants.StatusOutdated:
		return constants.IconWarning
	case lock.InstallStatusNotConfigured:
		return constants.IconNotConfigured
	case lock.InstallStatusFloating:
		return constants.IconBlocked
	default:
		return ""
	}
}

// IsSuccessStatus returns true if the status indicates success.
//
// Parameters:
//   - status: The status string to check
//
// Returns:
//   - bool: true if status is Updated or UpToDate
func IsSuccessStatus(status string) bool {
	return status == constants.StatusUpdated || status == constants.StatusUpToDate
}

// IsFailureStatus returns true if the status indicates failure.
//
// Parameters:
//   - status: The status string to check
//
// Returns:
//   - bool: true if status is Failed, ConfigError, or SummarizeError
func IsFailureStatus(status string) bool {
	return status == constants.StatusFailed ||
		status == constants.StatusConfigError ||
		status == constants.StatusSummarizeError
}

// IsPendingStatus returns true if the status indicates a pending operation.
//
// Parameters:
//   - status: The status string to check
//
// Returns:
//   - bool: true if status is Planned
func IsPendingStatus(status string) bool {
	return status == constants.StatusPlanned
}

// FormatInstallStatus formats an installation status for display.
//
// Converts lock file status values to user-friendly display strings.
//
// Parameters:
//   - status: Installation status (e.g., "LockFound", "NotInLock", "Floating")
//
// Returns:
//   - string: Formatted status with icon
func FormatInstallStatus(status string) string {
	switch status {
	case lock.InstallStatusLockFound:
		return fmt.Sprintf("%s LockFound", constants.IconSuccess)
	case lock.InstallStatusNotInLock:
		return fmt.Sprintf("%s NotInLock", constants.IconInfo)
	case lock.InstallStatusLockMissing:
		return fmt.Sprintf("%s LockMissing", constants.IconWarning)
	case lock.InstallStatusFloating:
		return fmt.Sprintf("%s Floating", constants.IconBlocked)
	case lock.InstallStatusNotConfigured:
		return fmt.Sprintf("%s NotConfigured", constants.IconNotConfigured)
	case lock.InstallStatusVersionMissing:
		return fmt.Sprintf("%s VersionMissing", constants.IconError)
	case lock.InstallStatusSelfPinned:
		return fmt.Sprintf("%s SelfPinned", constants.IconPinned)
	default:
		return status
	}
}

// statusIconMap maps lowercase status prefixes to their icons.
var statusIconMap = map[string]string{
	strings.ToLower(constants.StatusOutdated):         constants.IconWarning,
	strings.ToLower(lock.InstallStatusNotConfigured):  constants.IconNotConfigured,
	strings.ToLower(lock.InstallStatusFloating):       constants.IconBlocked,
	strings.ToLower(constants.StatusUpToDate):         constants.IconSuccess,
	strings.ToLower(constants.StatusUpdated):          constants.IconSuccess,
	strings.ToLower(lock.InstallStatusLockFound):      constants.IconSuccess,
	strings.ToLower(lock.InstallStatusSelfPinned):     constants.IconPinned,
	strings.ToLower(lock.InstallStatusNotInLock):      constants.IconInfo,
	strings.ToLower(lock.InstallStatusLockMissing):    constants.IconWarning,
	strings.ToLower(lock.InstallStatusVersionMissing): constants.IconBlocked,
	strings.ToLower(constants.StatusFailed):           constants.IconError,
	strings.ToLower(constants.StatusPlanned):          constants.IconPending,
}

// FormatStatusWithIcon formats any status string with the appropriate icon prefix.
//
// This function handles both exact status matches and prefix matches (e.g., "Failed(1)").
// It uses case-insensitive matching and preserves the original status text.
//
// Parameters:
//   - status: The status string to format
//
// Returns:
//   - string: Formatted status with icon prefix (e.g., "🟢 Updated", "❌ Failed(1)")
//
// Example:
//
//	display.FormatStatusWithIcon("Updated")    // Returns "🟢 Updated"
//	display.FormatStatusWithIcon("Failed(1)")  // Returns "❌ Failed(1)"
//	display.FormatStatusWithIcon("LockFound")  // Returns "🟢 LockFound"
func FormatStatusWithIcon(status string) string {
	normalized := strings.ToLower(status)

	for key, icon := range statusIconMap {
		if normalized == key || strings.HasPrefix(normalized, key+"(") {
			return icon + " " + status
		}
	}

	return status
}
