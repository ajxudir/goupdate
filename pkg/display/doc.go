// Package display provides unified display and formatting for goupdate output.
//
// This package consolidates all display logic that was previously scattered
// across cmd/shared/display.go, cmd/shared/warning.go, and pkg/update/display.go.
//
// Value Formatting:
//
// Use formatting functions for consistent value display:
//
//	installed := display.SafeInstalledValue(pkg.InstalledVersion)  // Returns "#N/A" if empty
//	declared := display.SafeDeclaredValue(pkg.Version)             // Returns "*" if empty
//
// Status Formatting:
//
// Use status functions for consistent status display with icons:
//
//	status := display.FormatStatus("Updated")  // Returns "🟢 Updated"
//
// Messages:
//
// Use message functions for consistent user feedback:
//
//	display.PrintWarnings(os.Stderr, warnings)
//	display.PrintNoPackagesMessage(os.Stdout, "matching filters")
//
// For table output, use the pkg/output package directly.
package display
