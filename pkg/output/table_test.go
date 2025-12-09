package output

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTable tests the behavior of NewTable.
//
// It verifies:
//   - Creates table with zero columns and default separator
func TestNewTable(t *testing.T) {
	table := NewTable()
	require.NotNil(t, table)
	assert.Equal(t, 0, table.ColumnCount())
	assert.Equal(t, "  ", table.separator)
}

// TestTableAddColumn tests the behavior of AddColumn.
//
// It verifies:
//   - Adds column with header width
//   - Adds multiple columns correctly
//   - Chain returns same table instance
func TestTableAddColumn(t *testing.T) {
	t.Run("adds column with header width", func(t *testing.T) {
		table := NewTable().AddColumn("NAME")
		assert.Equal(t, 1, table.ColumnCount())
		assert.Equal(t, 4, table.GetColumnWidth(0)) // "NAME" = 4 chars
	})

	t.Run("adds multiple columns", func(t *testing.T) {
		table := NewTable().
			AddColumn("RULE").
			AddColumn("PM").
			AddColumn("STATUS")
		assert.Equal(t, 3, table.ColumnCount())
		assert.Equal(t, 4, table.GetColumnWidth(0)) // RULE
		assert.Equal(t, 2, table.GetColumnWidth(1)) // PM
		assert.Equal(t, 6, table.GetColumnWidth(2)) // STATUS
	})

	t.Run("chain returns same table", func(t *testing.T) {
		table := NewTable()
		result := table.AddColumn("TEST")
		assert.Same(t, table, result)
	})
}

// TestTableAddColumnWithMinWidth tests the behavior of AddColumnWithMinWidth.
//
// It verifies:
//   - Uses minWidth when larger than header
//   - Uses header width when larger than minWidth
func TestTableAddColumnWithMinWidth(t *testing.T) {
	t.Run("uses minWidth when larger than header", func(t *testing.T) {
		table := NewTable().AddColumnWithMinWidth("PM", 10)
		assert.Equal(t, 10, table.GetColumnWidth(0))
	})

	t.Run("uses header width when larger than minWidth", func(t *testing.T) {
		table := NewTable().AddColumnWithMinWidth("CONSTRAINT", 5)
		assert.Equal(t, 10, table.GetColumnWidth(0)) // CONSTRAINT = 10 chars
	})
}

// TestTableAddConditionalColumn tests the behavior of AddConditionalColumn.
//
// It verifies:
//   - Visible column is not hidden
//   - Invisible column is hidden
func TestTableAddConditionalColumn(t *testing.T) {
	t.Run("visible column is not hidden", func(t *testing.T) {
		table := NewTable().AddConditionalColumn("GROUP", true)
		assert.False(t, table.IsColumnHidden(0))
	})

	t.Run("invisible column is hidden", func(t *testing.T) {
		table := NewTable().AddConditionalColumn("GROUP", false)
		assert.True(t, table.IsColumnHidden(0))
	})
}

// TestTableSetColumnVisible tests the behavior of SetColumnVisible.
//
// It verifies:
//   - Hides column when set to false
//   - Shows column when set to true
//   - Ignores invalid index
func TestTableSetColumnVisible(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddColumn("GROUP")

	t.Run("hides column", func(t *testing.T) {
		table.SetColumnVisible(1, false)
		assert.True(t, table.IsColumnHidden(1))
	})

	t.Run("shows column", func(t *testing.T) {
		table.SetColumnVisible(1, true)
		assert.False(t, table.IsColumnHidden(1))
	})

	t.Run("ignores invalid index", func(t *testing.T) {
		table.SetColumnVisible(99, false) // Should not panic
	})
}

// TestTableSetColumnVisibleByHeader tests the behavior of SetColumnVisibleByHeader.
//
// It verifies:
//   - Sets column visibility by header name
func TestTableSetColumnVisibleByHeader(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddColumn("GROUP")

	table.SetColumnVisibleByHeader("GROUP", false)
	assert.True(t, table.IsColumnHidden(1))
	assert.False(t, table.IsColumnHidden(0))
}

// TestTableUpdateWidths tests the behavior of UpdateWidths.
//
// It verifies:
//   - Updates widths from row data
//   - Keeps larger width
//   - Handles unicode correctly
//   - Handles emoji correctly
func TestTableUpdateWidths(t *testing.T) {
	t.Run("updates widths from row data", func(t *testing.T) {
		table := NewTable().
			AddColumn("NAME").
			AddColumn("VERSION")

		table.UpdateWidths("react", "18.2.0")

		assert.Equal(t, 5, table.GetColumnWidth(0)) // "react" > "NAME"
		assert.Equal(t, 7, table.GetColumnWidth(1)) // "VERSION" stays (longer than "18.2.0")
	})

	t.Run("keeps larger width", func(t *testing.T) {
		table := NewTable().AddColumn("NAME")
		table.UpdateWidths("a")                     // width 1 < 4
		assert.Equal(t, 4, table.GetColumnWidth(0)) // "NAME" = 4
	})

	t.Run("handles unicode correctly", func(t *testing.T) {
		table := NewTable().AddColumn("NAME")
		table.UpdateWidths("日本語") // 6 display width (3 chars * 2)
		assert.Equal(t, 6, table.GetColumnWidth(0))
	})

	t.Run("handles emoji correctly", func(t *testing.T) {
		table := NewTable().AddColumn("ST")
		table.UpdateWidths("🟢 OK")
		// Emoji typically has width 2, plus " OK" = 5 total
		assert.GreaterOrEqual(t, table.GetColumnWidth(0), 4)
	})
}

// TestTableUpdateWidth tests the behavior of UpdateWidth for a single column.
//
// It verifies:
//   - Updates single column width
func TestTableUpdateWidth(t *testing.T) {
	table := NewTable().
		AddColumn("A").
		AddColumn("B")

	table.UpdateWidth(1, "longer-value")
	assert.Equal(t, 1, table.GetColumnWidth(0))  // unchanged
	assert.Equal(t, 12, table.GetColumnWidth(1)) // "longer-value"
}

// TestTableHeaderRow tests the behavior of HeaderRow.
//
// It verifies:
//   - Formats header row
//   - Pads headers to column widths
//   - Excludes hidden columns
func TestTableHeaderRow(t *testing.T) {
	t.Run("formats header row", func(t *testing.T) {
		table := NewTable().
			AddColumn("RULE").
			AddColumn("PM").
			AddColumn("NAME")

		header := table.HeaderRow()
		assert.Equal(t, "RULE  PM  NAME", header)
	})

	t.Run("pads headers to column widths", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddColumn("B")

		table.UpdateWidths("LONGER", "X")

		header := table.HeaderRow()
		assert.Equal(t, "A       B", header) // A padded to 6, B stays at 1
	})

	t.Run("excludes hidden columns", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddConditionalColumn("HIDDEN", false).
			AddColumn("B")

		header := table.HeaderRow()
		assert.Equal(t, "A  B", header)
	})
}

// TestTableSeparatorRow tests the behavior of SeparatorRow.
//
// It verifies:
//   - Creates dashes matching widths
//   - Excludes hidden columns
func TestTableSeparatorRow(t *testing.T) {
	t.Run("creates dashes matching widths", func(t *testing.T) {
		table := NewTable().
			AddColumn("RULE"). // 4
			AddColumn("PM")    // 2

		sep := table.SeparatorRow()
		assert.Equal(t, "----  --", sep)
	})

	t.Run("excludes hidden columns", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddConditionalColumn("HIDDEN", false).
			AddColumn("B")

		sep := table.SeparatorRow()
		assert.Equal(t, "-  -", sep)
	})
}

// TestTableFormatRow tests the behavior of FormatRow.
//
// It verifies:
//   - Formats data row with padding
//   - Handles missing values
//   - Skips hidden columns in output
func TestTableFormatRow(t *testing.T) {
	t.Run("formats data row with padding", func(t *testing.T) {
		table := NewTable().
			AddColumn("NAME").
			AddColumn("VERSION")

		table.UpdateWidths("react", "18.2.0")

		row := table.FormatRow("vue", "3.0.0")
		assert.Equal(t, "vue    3.0.0  ", row) // NAME:5, VERSION:7
	})

	t.Run("handles missing values", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddColumn("B").
			AddColumn("C")

		row := table.FormatRow("x") // Only one value provided
		assert.Contains(t, row, "x")
	})

	t.Run("skips hidden columns in output", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddConditionalColumn("B", false).
			AddColumn("C")

		row := table.FormatRow("1", "2", "3")
		assert.Equal(t, "1  3", row) // B is hidden, but value "2" at index 1 is for B
	})
}

// TestTableFormatRowFiltered tests the behavior of FormatRowFiltered.
//
// It verifies:
//   - Formats row from pre-filtered values
func TestTableFormatRowFiltered(t *testing.T) {
	t.Run("formats row from pre-filtered values", func(t *testing.T) {
		table := NewTable().
			AddColumn("A").
			AddConditionalColumn("B", false).
			AddColumn("C")

		// Values already exclude hidden column
		row := table.FormatRowFiltered("1", "3")
		assert.Equal(t, "1  3", row)
	})
}

// TestTableVisibleColumnCount tests the behavior of VisibleColumnCount.
//
// It verifies:
//   - Counts total and visible columns correctly
func TestTableVisibleColumnCount(t *testing.T) {
	table := NewTable().
		AddColumn("A").
		AddConditionalColumn("B", false).
		AddColumn("C").
		AddConditionalColumn("D", false)

	assert.Equal(t, 4, table.ColumnCount())
	assert.Equal(t, 2, table.VisibleColumnCount())
}

// TestTableGetColumnWidthByHeader tests the behavior of GetColumnWidthByHeader.
//
// It verifies:
//   - Gets column width by header name
//   - Returns 0 for non-existent header
func TestTableGetColumnWidthByHeader(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddColumn("VERSION")

	table.UpdateWidths("react-native", "1.0.0")

	assert.Equal(t, 12, table.GetColumnWidthByHeader("NAME"))
	assert.Equal(t, 7, table.GetColumnWidthByHeader("VERSION"))
	assert.Equal(t, 0, table.GetColumnWidthByHeader("NONEXISTENT"))
}

// TestTableWithSeparator tests the behavior of WithSeparator.
//
// It verifies:
//   - Uses custom separator in output
func TestTableWithSeparator(t *testing.T) {
	table := NewTable().
		WithSeparator(" | ").
		AddColumn("A").
		AddColumn("B")

	header := table.HeaderRow()
	assert.Equal(t, "A | B", header)

	sep := table.SeparatorRow()
	assert.Equal(t, "- | -", sep)
}

// TestTableClone tests the behavior of Clone.
//
// It verifies:
//   - Creates independent copy of table
func TestTableClone(t *testing.T) {
	original := NewTable().
		AddColumn("NAME").
		AddColumn("VERSION")
	original.UpdateWidths("react", "1.0.0")

	clone := original.Clone()

	// Modify clone
	clone.UpdateWidths("very-long-package-name", "1.0.0")

	// Original should be unchanged
	assert.Equal(t, 5, original.GetColumnWidth(0))
	assert.Equal(t, 22, clone.GetColumnWidth(0))
}

// TestTableString tests the behavior of String.
//
// It verifies:
//   - Returns debug representation of table
func TestTableString(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddConditionalColumn("GROUP", false)

	str := table.String()
	assert.Contains(t, str, "NAME:4")
	assert.Contains(t, str, "GROUP:5 (hidden)")
}

// NOTE: DisplayWidth, ToWidth, and Max tests have been moved to pkg/utils/display_test.go
// to avoid duplication after consolidating utility functions.

// TestShouldShowGroupColumn tests the behavior of ShouldShowGroupColumn.
//
// It verifies:
//   - Returns false for empty groups
//   - Returns false for all unique groups
//   - Returns false for single item groups
//   - Returns true when group has 2+ items
//   - Returns true when multiple groups have 2+ items
//   - Ignores empty strings
//   - Trims whitespace
func TestShouldShowGroupColumn(t *testing.T) {
	t.Run("returns false for empty groups", func(t *testing.T) {
		assert.False(t, ShouldShowGroupColumn([]string{}))
	})

	t.Run("returns false for all unique groups", func(t *testing.T) {
		groups := []string{"a", "b", "c", "d"}
		assert.False(t, ShouldShowGroupColumn(groups))
	})

	t.Run("returns false for single item groups", func(t *testing.T) {
		groups := []string{"a", "", "b", "", "c"}
		assert.False(t, ShouldShowGroupColumn(groups))
	})

	t.Run("returns true when group has 2+ items", func(t *testing.T) {
		groups := []string{"a", "a", "b", "c"}
		assert.True(t, ShouldShowGroupColumn(groups))
	})

	t.Run("returns true when multiple groups have 2+ items", func(t *testing.T) {
		groups := []string{"a", "a", "b", "b", "b"}
		assert.True(t, ShouldShowGroupColumn(groups))
	})

	t.Run("ignores empty strings", func(t *testing.T) {
		groups := []string{"", "", "", ""}
		assert.False(t, ShouldShowGroupColumn(groups))
	})

	t.Run("trims whitespace", func(t *testing.T) {
		groups := []string{" a ", "a", "  a  "}
		assert.True(t, ShouldShowGroupColumn(groups))
	})
}

// TestTableIntegration tests the full workflow of table usage.
//
// It verifies:
//   - Full workflow in scan command style
//   - Full workflow with hidden group column
//   - Full workflow with visible group column
func TestTableIntegration(t *testing.T) {
	t.Run("full workflow - scan command style", func(t *testing.T) {
		table := NewTable().
			AddColumn("RULE").
			AddColumn("PM").
			AddColumn("FORMAT").
			AddColumn("FILE").
			AddColumn("STATUS")

		// Simulate adding rows
		rows := [][]string{
			{"npm", "npm", "json", "package.json", "✓ Valid"},
			{"golang", "go", "raw", "go.mod", "✓ Valid"},
			{"composer", "php", "json", "composer.json", "✗ Invalid"},
		}

		for _, row := range rows {
			table.UpdateWidths(row...)
		}

		// Check header
		header := table.HeaderRow()
		assert.Contains(t, header, "RULE")
		assert.Contains(t, header, "STATUS")

		// Check separator has correct width
		sep := table.SeparatorRow()
		assert.Contains(t, sep, "---") // At least some dashes

		// Check row formatting
		formatted := table.FormatRow(rows[0]...)
		assert.Contains(t, formatted, "npm")
		assert.Contains(t, formatted, "package.json")
	})

	t.Run("full workflow - with hidden group column", func(t *testing.T) {
		groups := []string{"core", "ui", "api"} // All unique
		showGroup := ShouldShowGroupColumn(groups)

		table := NewTable().
			AddColumn("NAME").
			AddConditionalColumn("GROUP", showGroup).
			AddColumn("VERSION")

		assert.False(t, showGroup)
		assert.Equal(t, 2, table.VisibleColumnCount())

		header := table.HeaderRow()
		assert.NotContains(t, header, "GROUP")
	})

	t.Run("full workflow - with visible group column", func(t *testing.T) {
		groups := []string{"core", "core", "ui"} // "core" has 2+
		showGroup := ShouldShowGroupColumn(groups)

		table := NewTable().
			AddColumn("NAME").
			AddConditionalColumn("GROUP", showGroup).
			AddColumn("VERSION")

		assert.True(t, showGroup)
		assert.Equal(t, 3, table.VisibleColumnCount())

		header := table.HeaderRow()
		assert.Contains(t, header, "GROUP")
	})
}

// TestTableFprint tests the behavior of Fprint.
//
// It verifies:
//   - Writes header and separator to writer
//   - Works with updated widths
func TestTableFprint(t *testing.T) {
	t.Run("writes header and separator to writer", func(t *testing.T) {
		table := NewTable().
			AddColumn("NAME").
			AddColumn("VERSION")

		var buf strings.Builder
		table.Fprint(&buf)

		output := buf.String()
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "VERSION")
		assert.Contains(t, output, "----") // separator contains dashes
	})

	t.Run("works with updated widths", func(t *testing.T) {
		table := NewTable().
			AddColumn("NAME").
			AddColumn("VERSION")
		table.UpdateWidths("react-native", "1.0.0")

		var buf strings.Builder
		table.Fprint(&buf)

		output := buf.String()
		// Header should be padded to match widths
		assert.Contains(t, output, "NAME")
		assert.Contains(t, output, "VERSION")
	})
}

// TestTablePrint tests the behavior of Print.
//
// It verifies:
//   - Does not panic with valid table
func TestTablePrint(t *testing.T) {
	// Print writes to stdout, so we just verify it doesn't panic
	t.Run("does not panic with valid table", func(t *testing.T) {
		table := NewTable().
			AddColumn("TEST").
			AddColumn("COL")

		// This should not panic
		assert.NotPanics(t, func() {
			// Note: This writes to stdout, which is expected
			// We're just testing it doesn't crash
			table.Print()
		})
	})
}

// TestGetColumnWidthEdgeCases tests edge cases for GetColumnWidth.
//
// It verifies:
//   - Returns zero for negative index
//   - Returns zero for index out of bounds
//   - Returns zero for index at boundary
func TestGetColumnWidthEdgeCases(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddColumn("VERSION")
	table.UpdateWidths("react", "1.0.0")

	t.Run("returns zero for negative index", func(t *testing.T) {
		assert.Equal(t, 0, table.GetColumnWidth(-1))
	})

	t.Run("returns zero for index out of bounds", func(t *testing.T) {
		assert.Equal(t, 0, table.GetColumnWidth(99))
	})

	t.Run("returns zero for index at boundary", func(t *testing.T) {
		assert.Equal(t, 0, table.GetColumnWidth(2)) // Only 2 columns (index 0,1)
	})
}

// TestIsColumnHiddenEdgeCases tests edge cases for IsColumnHidden.
//
// It verifies:
//   - Returns true for negative index
//   - Returns true for index out of bounds
//   - Returns true for index at boundary
func TestIsColumnHiddenEdgeCases(t *testing.T) {
	table := NewTable().
		AddColumn("NAME").
		AddConditionalColumn("GROUP", false)

	t.Run("returns true for negative index", func(t *testing.T) {
		assert.True(t, table.IsColumnHidden(-1))
	})

	t.Run("returns true for index out of bounds", func(t *testing.T) {
		assert.True(t, table.IsColumnHidden(99))
	})

	t.Run("returns true for index at boundary", func(t *testing.T) {
		assert.True(t, table.IsColumnHidden(2)) // Only 2 columns (index 0,1)
	})
}
