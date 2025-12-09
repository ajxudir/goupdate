package display

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/user/goupdate/pkg/formats"
)

// TestFormatConstraintDisplay tests the FormatConstraintDisplay function.
//
// It verifies that:
//   - Known constraints (^, ~, etc.) return appropriate display strings
//   - Empty constraints default to "Major"
//   - Unmapped constraints trigger the warning branch and return "Exact (#N/A)"
func TestFormatConstraintDisplay(t *testing.T) {
	t.Run("known constraint returns display string", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Constraint: "^"}
		result := FormatConstraintDisplay(pkg)
		assert.Equal(t, "Compatible (^)", result)
	})

	t.Run("empty constraint returns Major", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Constraint: ""}
		result := FormatConstraintDisplay(pkg)
		assert.Equal(t, "Major", result)
	})

	t.Run("tilde constraint returns Patch", func(t *testing.T) {
		pkg := formats.Package{Name: "test", Constraint: "~"}
		result := FormatConstraintDisplay(pkg)
		assert.Equal(t, "Patch (~)", result)
	})

	t.Run("unmapped constraint returns Exact with N/A", func(t *testing.T) {
		// An unmapped constraint triggers the warning branch and returns "Exact (#N/A)"
		pkg := formats.Package{
			Name:        "test",
			PackageType: "prod",
			Rule:        "npm",
			Constraint:  "unknown_constraint_xyz",
		}
		result := FormatConstraintDisplay(pkg)
		assert.Equal(t, "Exact (#N/A)", result)
	})
}

// TestFormatConstraintDisplayWithFlags tests the FormatConstraintDisplayWithFlags function.
//
// It verifies that:
//   - Version flags (--major, --minor, --patch) override constraint display
//   - Major flag takes precedence over minor and patch flags
//   - No flags returns the default constraint display
func TestFormatConstraintDisplayWithFlags(t *testing.T) {
	pkg := formats.Package{Name: "test", Constraint: "^"}

	t.Run("major flag shows major with flag indicator", func(t *testing.T) {
		result := FormatConstraintDisplayWithFlags(pkg, true, false, false)
		assert.Equal(t, "Major (--major)", result)
	})

	t.Run("minor flag shows minor with flag indicator", func(t *testing.T) {
		result := FormatConstraintDisplayWithFlags(pkg, false, true, false)
		assert.Equal(t, "Minor (--minor)", result)
	})

	t.Run("patch flag shows patch with flag indicator", func(t *testing.T) {
		result := FormatConstraintDisplayWithFlags(pkg, false, false, true)
		assert.Equal(t, "Patch (--patch)", result)
	})

	t.Run("no flags returns constraint display", func(t *testing.T) {
		result := FormatConstraintDisplayWithFlags(pkg, false, false, false)
		expected := FormatConstraintDisplay(pkg)
		assert.Equal(t, expected, result)
	})

	t.Run("major flag takes precedence", func(t *testing.T) {
		result := FormatConstraintDisplayWithFlags(pkg, true, true, true)
		assert.Equal(t, "Major (--major)", result)
	})
}
