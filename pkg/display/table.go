package display

import "github.com/user/goupdate/pkg/output"

// Alignment specifies text alignment within a column.
//
// Used by table column definitions to control how text is aligned.
type Alignment int

const (
	// AlignLeft aligns text to the left of the column.
	AlignLeft Alignment = iota

	// AlignRight aligns text to the right of the column.
	AlignRight

	// AlignCenter centers text within the column.
	AlignCenter
)

// ColumnDef defines a single table column's properties.
//
// Fields:
//   - Name: Column header text (displayed in uppercase)
//   - MinWidth: Minimum column width in characters
//   - Align: Text alignment within the column
//   - Optional: If true, column can be hidden via TableOptions
//
// Example:
//
//	col := ColumnDef{Name: "NAME", MinWidth: 10, Align: AlignLeft}
type ColumnDef struct {
	// Name is the column header text.
	Name string

	// MinWidth is the minimum width in characters.
	// Column will expand to fit content if content is wider.
	MinWidth int

	// Align specifies how text is aligned within the column.
	// Default is AlignLeft.
	Align Alignment

	// Optional indicates this column can be hidden.
	// Use TableOptions.ShowOptional to control visibility.
	Optional bool
}

// Schema defines a complete table structure.
//
// Fields:
//   - Columns: Ordered list of column definitions
//   - OptionalCols: Map of column names to show/hide state
//
// Example:
//
//	schema := Schema{
//	    Columns: []ColumnDef{
//	        {Name: "RULE", MinWidth: 4},
//	        {Name: "NAME", MinWidth: 10},
//	    },
//	}
type Schema struct {
	// Columns defines the table columns in display order.
	Columns []ColumnDef

	// OptionalCols controls which optional columns are shown.
	// Key is column name, value is whether to show.
	OptionalCols map[string]bool
}

// Predefined table schemas - SINGLE SOURCE OF TRUTH.
//
// These schemas define the exact column structure for each command's
// table output. All table creation should use these schemas.
var (
	// ListSchema defines columns for the 'list' command output.
	// Columns: RULE, PM, TYPE, CONSTRAINT, VERSION, INSTALLED, STATUS, GROUP*, NAME
	// * GROUP is optional
	ListSchema = Schema{
		Columns: []ColumnDef{
			{Name: "RULE", MinWidth: 4},
			{Name: "PM", MinWidth: 2},
			{Name: "TYPE", MinWidth: 4},
			{Name: "CONSTRAINT", MinWidth: 10},
			{Name: "VERSION", MinWidth: 7},
			{Name: "INSTALLED", MinWidth: 9},
			{Name: "STATUS", MinWidth: 6},
			{Name: "GROUP", MinWidth: 5, Optional: true},
			{Name: "NAME", MinWidth: 4},
		},
	}

	// OutdatedSchema defines columns for the 'outdated' command output.
	// Columns: RULE, PM, TYPE, CONSTRAINT, INSTALLED, MAJOR, MINOR, PATCH, STATUS, GROUP*, NAME
	// * GROUP is optional
	OutdatedSchema = Schema{
		Columns: []ColumnDef{
			{Name: "RULE", MinWidth: 4},
			{Name: "PM", MinWidth: 2},
			{Name: "TYPE", MinWidth: 4},
			{Name: "CONSTRAINT", MinWidth: 10},
			{Name: "INSTALLED", MinWidth: 9},
			{Name: "MAJOR", MinWidth: 5},
			{Name: "MINOR", MinWidth: 5},
			{Name: "PATCH", MinWidth: 5},
			{Name: "STATUS", MinWidth: 6},
			{Name: "GROUP", MinWidth: 5, Optional: true},
			{Name: "NAME", MinWidth: 4},
		},
	}

	// UpdateSchema defines columns for the 'update' command output.
	// Columns: RULE, PM, TYPE, CONSTRAINT, VERSION, INSTALLED, TARGET, STATUS, GROUP*, NAME
	// * GROUP is optional
	UpdateSchema = Schema{
		Columns: []ColumnDef{
			{Name: "RULE", MinWidth: 4},
			{Name: "PM", MinWidth: 2},
			{Name: "TYPE", MinWidth: 4},
			{Name: "CONSTRAINT", MinWidth: 10},
			{Name: "VERSION", MinWidth: 7},
			{Name: "INSTALLED", MinWidth: 9},
			{Name: "TARGET", MinWidth: 6},
			{Name: "STATUS", MinWidth: 6},
			{Name: "GROUP", MinWidth: 5, Optional: true},
			{Name: "NAME", MinWidth: 4},
		},
	}

	// ScanSchema defines columns for the 'scan' command output.
	// Columns: RULE, PM, FORMAT, FILE, STATUS
	ScanSchema = Schema{
		Columns: []ColumnDef{
			{Name: "RULE", MinWidth: 4},
			{Name: "PM", MinWidth: 2},
			{Name: "FORMAT", MinWidth: 6},
			{Name: "FILE", MinWidth: 4},
			{Name: "STATUS", MinWidth: 6},
		},
	}
)

// TableOptions configures table creation from a schema.
//
// Fields:
//   - ShowOptional: Map of optional column names to show
//   - NoHeader: If true, omits the header row
//   - NoSeparator: If true, omits the separator line after header
//
// Example:
//
//	opts := TableOptions{
//	    ShowOptional: map[string]bool{"GROUP": true},
//	}
type TableOptions struct {
	// ShowOptional controls which optional columns are displayed.
	// Key is column name (e.g., "GROUP"), value is whether to show.
	ShowOptional map[string]bool

	// NoHeader omits the header row if true.
	NoHeader bool

	// NoSeparator omits the separator line if true.
	NoSeparator bool
}

// NewTableFromSchema creates an output.Table from a schema and options.
//
// Parameters:
//   - schema: Table schema defining columns
//   - options: Configuration options
//
// Returns:
//   - *output.Table: New table ready for adding rows
//
// Example:
//
//	opts := TableOptions{ShowOptional: map[string]bool{"GROUP": true}}
//	table := display.NewTableFromSchema(display.ListSchema, opts)
func NewTableFromSchema(schema Schema, options TableOptions) *output.Table {
	table := output.NewTable()
	for _, col := range schema.Columns {
		if col.Optional {
			visible := options.ShowOptional[col.Name]
			table.AddConditionalColumn(col.Name, visible)
		} else if col.MinWidth > 0 {
			table.AddColumnWithMinWidth(col.Name, col.MinWidth)
		} else {
			table.AddColumn(col.Name)
		}
	}
	return table
}

// NewListTable creates a table for 'list' command output.
//
// Parameters:
//   - showGroup: If true, includes the GROUP column
//
// Returns:
//   - *output.Table: Table configured with ListSchema
//
// Example:
//
//	table := display.NewListTable(true)  // with GROUP column
//	table := display.NewListTable(false) // without GROUP column
func NewListTable(showGroup bool) *output.Table {
	return NewTableFromSchema(ListSchema, TableOptions{
		ShowOptional: map[string]bool{"GROUP": showGroup},
	})
}

// NewOutdatedTable creates a table for 'outdated' command output.
//
// Parameters:
//   - showGroup: If true, includes the GROUP column
//
// Returns:
//   - *output.Table: Table configured with OutdatedSchema
//
// Example:
//
//	table := display.NewOutdatedTable(true)
func NewOutdatedTable(showGroup bool) *output.Table {
	return NewTableFromSchema(OutdatedSchema, TableOptions{
		ShowOptional: map[string]bool{"GROUP": showGroup},
	})
}

// NewUpdateTable creates a table for 'update' command output.
//
// Parameters:
//   - showGroup: If true, includes the GROUP column
//
// Returns:
//   - *output.Table: Table configured with UpdateSchema
//
// Example:
//
//	table := display.NewUpdateTable(true)
func NewUpdateTable(showGroup bool) *output.Table {
	return NewTableFromSchema(UpdateSchema, TableOptions{
		ShowOptional: map[string]bool{"GROUP": showGroup},
	})
}

// NewScanTable creates a table for 'scan' command output.
//
// Returns:
//   - *output.Table: Table configured with ScanSchema
//
// Example:
//
//	table := display.NewScanTable()
func NewScanTable() *output.Table {
	return NewTableFromSchema(ScanSchema, TableOptions{})
}
