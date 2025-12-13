// Package constants provides centralized string constants used throughout the application.
// This eliminates magic strings and provides a single source of truth for status values.
package constants

// Update status constants represent the state of a package during update operations.
const (
	// StatusUpToDate indicates the package is already at the target version.
	StatusUpToDate = "UpToDate"

	// StatusUpdated indicates the package was successfully updated.
	StatusUpdated = "Updated"

	// StatusPlanned indicates the package update is planned (dry-run mode).
	StatusPlanned = "Planned"

	// StatusFailed indicates the update operation failed.
	StatusFailed = "Failed"

	// StatusConfigError indicates a configuration error prevented the update.
	StatusConfigError = "ConfigError"

	// StatusSummarizeError indicates an error occurred while summarizing available versions.
	StatusSummarizeError = "SummarizeError"

	// StatusOutdated indicates newer versions are available for the package.
	StatusOutdated = "Outdated"
)

// Placeholder values for display when data is not available.
const (
	// PlaceholderNA is used when a value is not available.
	PlaceholderNA = "#N/A"

	// PlaceholderWildcard is used when a version is unconstrained.
	PlaceholderWildcard = "*"
)

// Output format constants.
const (
	// FilterAll is the default filter value that matches all items.
	FilterAll = "all"
)

// Icon constants for status display.
// These provide visual indicators for package states in CLI output.
const (
	// IconSuccess indicates a successful or positive state (green circle).
	IconSuccess = "🟢"

	// IconWarning indicates a warning or caution state (orange circle).
	IconWarning = "🟠"

	// IconError indicates an error or failed state (red X).
	IconError = "❌"

	// IconInfo indicates informational or neutral state (blue circle).
	IconInfo = "🔵"

	// IconNotConfigured indicates unconfigured state (white circle).
	IconNotConfigured = "⚪"

	// IconBlocked indicates a blocked or unsupported state (stop sign).
	IconBlocked = "⛔"

	// IconPinned indicates a pinned/self-pinning state (pin emoji).
	IconPinned = "📌"

	// IconPending indicates a pending or planned state (yellow circle).
	IconPending = "🟡"

	// IconIgnored indicates a package is excluded from processing (no entry).
	IconIgnored = "🚫"

	// IconCheckmark indicates a passed check (checkmark).
	IconCheckmark = "✓"

	// IconCross indicates a failed check (cross).
	IconCross = "✗"

	// IconWarn is the warning prefix for messages.
	IconWarn = "⚠️"

	// IconCheckmarkBox indicates successful validation (checkmark in box).
	IconCheckmarkBox = "✅"

	// IconLightbulb indicates a hint or suggestion.
	IconLightbulb = "💡"
)

// Validation status constants for file validation.
const (
	// ValidationValid indicates a valid file.
	ValidationValid = "🟢 valid"

	// ValidationInvalid indicates an invalid file.
	ValidationInvalid = "❌ invalid"
)
