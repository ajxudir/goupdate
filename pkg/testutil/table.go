package testutil

import (
	"github.com/user/goupdate/pkg/output"
)

// CreateUpdateTable creates a table for update tests with all standard columns.
//
// This mirrors the table structure used in the update command, including
// columns for rule, package manager, type, constraint, version information,
// and package name.
//
// Returns:
//   - *output.Table: Pre-configured table with standard update columns
func CreateUpdateTable() *output.Table {
	return output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("TARGET").
		AddColumn("STATUS").
		AddConditionalColumn("GROUP", false).
		AddColumn("NAME")
}

// CreateUpdateTableWithGroup creates a table for update tests with the GROUP column enabled.
//
// This mirrors the table structure used in the update command when packages
// have group assignments, including an additional GROUP column for organization.
//
// Returns:
//   - *output.Table: Pre-configured table with standard update columns and GROUP column
func CreateUpdateTableWithGroup() *output.Table {
	return output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("TARGET").
		AddColumn("STATUS").
		AddConditionalColumn("GROUP", true).
		AddColumn("NAME")
}

// CreateOutdatedTable creates a table for outdated tests with standard columns.
//
// This mirrors the table structure used in the outdated command, including
// columns for rule, package manager, type, version information, and latest
// available version.
//
// Returns:
//   - *output.Table: Pre-configured table with standard outdated columns
func CreateOutdatedTable() *output.Table {
	return output.NewTable().
		AddColumn("RULE").
		AddColumn("PM").
		AddColumn("TYPE").
		AddColumn("CONSTRAINT").
		AddColumn("VERSION").
		AddColumn("INSTALLED").
		AddColumn("LATEST").
		AddConditionalColumn("GROUP", false).
		AddColumn("NAME")
}
