package display

import (
	"fmt"

	"github.com/ajxudir/goupdate/pkg/constants"
	"github.com/ajxudir/goupdate/pkg/formats"
	"github.com/ajxudir/goupdate/pkg/utils"
	"github.com/ajxudir/goupdate/pkg/warnings"
)

// FormatConstraintDisplay formats a package constraint for display.
//
// It converts the internal constraint representation to a human-readable string.
// If the constraint is unmapped, it logs a warning and returns "Exact (#N/A)".
//
// Parameters:
//   - p: The package whose constraint to format
//
// Returns:
//   - string: Human-readable constraint display (e.g., "Minor", "Exact (^1.0.0)")
//
// Example:
//
//	display.FormatConstraintDisplay(pkg) // "Minor" or "Exact (^1.0.0)"
func FormatConstraintDisplay(p formats.Package) string {
	display, ok, warn := utils.GetConstraintDisplay(p.Constraint)
	if warn {
		warnings.Warnf("⚠️ %s (%s/%s): Unmapped constraint '%s', falling back to exact to be safe\n", p.Name, p.PackageType, p.Rule, p.Constraint)
	}

	if !ok {
		return fmt.Sprintf("Exact (%s)", constants.PlaceholderNA)
	}

	return display
}

// FormatConstraintDisplayWithFlags returns the constraint display with flag override indicator.
//
// When a flag overrides the package's constraint, the display shows the effective constraint
// with the flag that caused the override (e.g., "Major (--major)").
//
// Parameters:
//   - p: The package whose constraint to format
//   - majorFlag: Whether --major flag is set
//   - minorFlag: Whether --minor flag is set
//   - patchFlag: Whether --patch flag is set
//
// Returns:
//   - string: Constraint display with flag indicator if overridden
//
// Example:
//
//	display.FormatConstraintDisplayWithFlags(pkg, true, false, false)
//	// Returns "Major (--major)"
func FormatConstraintDisplayWithFlags(p formats.Package, majorFlag, minorFlag, patchFlag bool) string {
	// If a flag overrides the constraint, show the effective constraint with flag indicator
	switch {
	case majorFlag:
		return "Major (--major)"
	case minorFlag:
		return "Minor (--minor)"
	case patchFlag:
		return "Patch (--patch)"
	default:
		return FormatConstraintDisplay(p)
	}
}
