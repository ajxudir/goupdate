// Package output provides utilities for formatting command output.
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/ajxudir/goupdate/pkg/utils"
)

// Column represents a single table column with its header and current width.
//
// Fields:
//   - Header: The display text for this column's header
//   - Width: The current display width for this column in characters
//   - hidden: Whether this column should be excluded from output
type Column struct {
	Header string
	Width  int
	hidden bool
}

// Table provides a flexible table formatter with dynamic column widths.
// It handles Unicode-aware width calculations and consistent formatting.
//
// Fields:
//   - columns: List of columns with their headers, widths, and visibility state
//   - separator: String used to separate columns in formatted output (default: "  ")
type Table struct {
	columns   []Column
	separator string
}

// NewTable creates a new table formatter and returns a pointer to it.
//
// The table is initialized with an empty column list and a default separator
// of two spaces ("  ").
//
// Returns:
//   - *Table: A new table instance ready for column configuration
func NewTable() *Table {
	return &Table{
		columns:   make([]Column, 0),
		separator: "  ",
	}
}

// WithSeparator sets a custom column separator and returns the table.
//
// Parameters:
//   - sep: The string to use between columns (e.g., " | " for pipe-separated output)
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) WithSeparator(sep string) *Table {
	t.separator = sep
	return t
}

// AddColumn adds a column with the given header and returns the table.
//
// The initial width is set to the display width of the header using
// Unicode-aware width calculation.
//
// Parameters:
//   - header: The text to display in the column header
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) AddColumn(header string) *Table {
	t.columns = append(t.columns, Column{
		Header: header,
		Width:  utils.DisplayWidth(header),
		hidden: false,
	})
	return t
}

// AddColumnWithMinWidth adds a column with a minimum width guarantee and returns the table.
//
// The column width will be set to the larger of minWidth or the display width
// of the header. This is useful for ensuring columns don't become too narrow.
//
// Parameters:
//   - header: The text to display in the column header
//   - minWidth: Minimum width in characters for this column
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) AddColumnWithMinWidth(header string, minWidth int) *Table {
	width := utils.DisplayWidth(header)
	if minWidth > width {
		width = minWidth
	}
	t.columns = append(t.columns, Column{
		Header: header,
		Width:  width,
		hidden: false,
	})
	return t
}

// AddConditionalColumn adds a column with configurable visibility and returns the table.
//
// This is useful for columns that should only appear when certain data exists,
// such as a GROUP column that's hidden when no items have group assignments.
//
// Parameters:
//   - header: The text to display in the column header
//   - visible: Whether the column should be initially visible
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) AddConditionalColumn(header string, visible bool) *Table {
	t.columns = append(t.columns, Column{
		Header: header,
		Width:  utils.DisplayWidth(header),
		hidden: !visible,
	})
	return t
}

// SetColumnVisible sets the visibility of a column by index and returns the table.
//
// Parameters:
//   - index: Zero-based index of the column to modify
//   - visible: Whether the column should be visible (true) or hidden (false)
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) SetColumnVisible(index int, visible bool) *Table {
	if index >= 0 && index < len(t.columns) {
		t.columns[index].hidden = !visible
	}
	return t
}

// SetColumnVisibleByHeader sets the visibility of a column by header name and returns the table.
//
// If multiple columns have the same header, only the first match is affected.
//
// Parameters:
//   - header: The header text of the column to modify
//   - visible: Whether the column should be visible (true) or hidden (false)
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) SetColumnVisibleByHeader(header string, visible bool) *Table {
	for i := range t.columns {
		if t.columns[i].Header == header {
			t.columns[i].hidden = !visible
			break
		}
	}
	return t
}

// UpdateWidths updates column widths based on a row of values and returns the table.
//
// It performs the following operations:
//   - Step 1: Calculates display width for each value using Unicode-aware measurement
//   - Step 2: Compares each value's width with the current column width
//   - Step 3: Keeps the larger width to ensure all content fits
//
// Parameters:
//   - values: Variable number of strings representing a data row
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) UpdateWidths(values ...string) *Table {
	for i, val := range values {
		if i < len(t.columns) {
			width := utils.DisplayWidth(val)
			if width > t.columns[i].Width {
				t.columns[i].Width = width
			}
		}
	}
	return t
}

// UpdateWidth updates a single column's width by index and returns the table.
//
// The column width is only increased if the value's display width is larger
// than the current width.
//
// Parameters:
//   - index: Zero-based index of the column to update
//   - value: The string value to measure and potentially expand the column width
//
// Returns:
//   - *Table: The table instance for method chaining
func (t *Table) UpdateWidth(index int, value string) *Table {
	if index >= 0 && index < len(t.columns) {
		width := utils.DisplayWidth(value)
		if width > t.columns[index].Width {
			t.columns[index].Width = width
		}
	}
	return t
}

// HeaderRow returns the formatted header row string.
//
// Hidden columns are excluded from the output. Each header is padded to match
// its column's width.
//
// Returns:
//   - string: Formatted header row with columns separated by the separator
func (t *Table) HeaderRow() string {
	var parts []string
	for _, col := range t.columns {
		if !col.hidden {
			parts = append(parts, utils.ToWidth(col.Header, col.Width))
		}
	}
	return strings.Join(parts, t.separator)
}

// SeparatorRow returns a separator row with dashes matching column widths.
//
// Hidden columns are excluded. Each separator contains as many dashes as the
// column's width to create a visual divider between header and data rows.
//
// Returns:
//   - string: Formatted separator row with dash sequences separated by the separator
func (t *Table) SeparatorRow() string {
	var parts []string
	for _, col := range t.columns {
		if !col.hidden {
			parts = append(parts, strings.Repeat("-", col.Width))
		}
	}
	return strings.Join(parts, t.separator)
}

// FormatRow formats a data row with proper padding for each column and returns the formatted string.
//
// Values are padded to match their respective column widths. Hidden columns are
// skipped, but their corresponding values should still be included in the input.
// Missing values (when fewer values than columns are provided) are treated as empty strings.
//
// Parameters:
//   - values: Variable number of strings representing the row data, one per column
//
// Returns:
//   - string: Formatted row with values separated by the separator
func (t *Table) FormatRow(values ...string) string {
	var parts []string
	visibleIdx := 0
	for i, col := range t.columns {
		if !col.hidden {
			val := ""
			if i < len(values) {
				val = values[i]
			}
			parts = append(parts, utils.ToWidth(val, col.Width))
			visibleIdx++
		}
	}
	return strings.Join(parts, t.separator)
}

// FormatRowFiltered formats a row using only visible column values and returns the formatted string.
//
// This method expects that you've already filtered out values for hidden columns.
// It maps values sequentially to visible columns only.
//
// Parameters:
//   - values: Variable number of strings, one for each visible column only
//
// Returns:
//   - string: Formatted row with values separated by the separator
func (t *Table) FormatRowFiltered(values ...string) string {
	var parts []string
	valIdx := 0
	for _, col := range t.columns {
		if !col.hidden {
			val := ""
			if valIdx < len(values) {
				val = values[valIdx]
			}
			parts = append(parts, utils.ToWidth(val, col.Width))
			valIdx++
		}
	}
	return strings.Join(parts, t.separator)
}

// ColumnCount returns the total number of columns including hidden ones.
//
// Returns:
//   - int: Total count of all columns (both visible and hidden)
func (t *Table) ColumnCount() int {
	return len(t.columns)
}

// VisibleColumnCount returns the number of visible columns.
//
// Returns:
//   - int: Count of columns that are not hidden
func (t *Table) VisibleColumnCount() int {
	count := 0
	for _, col := range t.columns {
		if !col.hidden {
			count++
		}
	}
	return count
}

// GetColumnWidth returns the width of a column by index.
//
// Parameters:
//   - index: Zero-based index of the column
//
// Returns:
//   - int: The column's width in characters; returns 0 if index is out of bounds
func (t *Table) GetColumnWidth(index int) int {
	if index >= 0 && index < len(t.columns) {
		return t.columns[index].Width
	}
	return 0
}

// GetColumnWidthByHeader returns the width of a column by header name.
//
// Parameters:
//   - header: The header text of the column to query
//
// Returns:
//   - int: The column's width in characters; returns 0 if no matching column is found
func (t *Table) GetColumnWidthByHeader(header string) int {
	for _, col := range t.columns {
		if col.Header == header {
			return col.Width
		}
	}
	return 0
}

// IsColumnHidden returns whether a column is hidden by index.
//
// Parameters:
//   - index: Zero-based index of the column to check
//
// Returns:
//   - bool: true if the column is hidden or index is out of bounds; false otherwise
func (t *Table) IsColumnHidden(index int) bool {
	if index >= 0 && index < len(t.columns) {
		return t.columns[index].hidden
	}
	return true
}

// Clone creates a deep copy of the table and returns it.
//
// The cloned table has independent column state, so modifications to the clone
// do not affect the original table.
//
// Returns:
//   - *Table: A new table instance with copied column configuration
func (t *Table) Clone() *Table {
	clone := &Table{
		columns:   make([]Column, len(t.columns)),
		separator: t.separator,
	}
	copy(clone.columns, t.columns)
	return clone
}

// ShouldShowGroupColumn determines if a GROUP column should be displayed.
//
// It performs the following operations:
//   - Step 1: Counts occurrences of each unique group (ignoring empty/whitespace-only groups)
//   - Step 2: Checks if any group appears 2 or more times
//
// This is a common pattern used across list, outdated, and update commands to
// avoid showing an empty or single-item group column.
//
// Parameters:
//   - groups: Slice of group names, may contain duplicates, empty strings, or whitespace
//
// Returns:
//   - bool: true if at least one group has 2 or more items; false otherwise
func ShouldShowGroupColumn(groups []string) bool {
	groupCounts := make(map[string]int)
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group != "" {
			groupCounts[group]++
		}
	}

	for _, count := range groupCounts {
		if count >= 2 {
			return true
		}
	}

	return false
}

// Print outputs the table header and separator to stdout.
//
// This is a convenience method for displaying table headers before printing
// data rows in a loop.
func (t *Table) Print() {
	fmt.Println(t.HeaderRow())
	fmt.Println(t.SeparatorRow())
}

// Fprint outputs the table header and separator to the given writer.
//
// Parameters:
//   - w: The writer to output to (e.g., os.Stdout, os.Stderr, or a buffer)
func (t *Table) Fprint(w io.Writer) {
	_, _ = fmt.Fprintln(w, t.HeaderRow())
	_, _ = fmt.Fprintln(w, t.SeparatorRow())
}

// String returns a string representation of the table structure for debugging.
//
// The output shows all columns with their headers, widths, and visibility state
// in the format: "Table{columns: [Header1:Width1, Header2:Width2 (hidden), ...]}".
//
// Returns:
//   - string: A human-readable representation of the table configuration
func (t *Table) String() string {
	var sb strings.Builder
	sb.WriteString("Table{columns: [")
	for i, col := range t.columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		hidden := ""
		if col.hidden {
			hidden = " (hidden)"
		}
		sb.WriteString(fmt.Sprintf("%s:%d%s", col.Header, col.Width, hidden))
	}
	sb.WriteString("]}")
	return sb.String()
}
