package utils

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// DisplayWidth returns the display width of a string, accounting for unicode characters.
//
// It calculates the visual width of a string as it would appear in a terminal,
// correctly handling wide characters (e.g., CJK characters, emojis) that occupy
// more than one character cell.
//
// Parameters:
//   - val: The string to measure
//
// Returns:
//   - int: The display width in character cells (wide characters count as 2)
func DisplayWidth(val string) int {
	return runewidth.StringWidth(val)
}

// ToWidth pads a string to a specific display width.
//
// It performs the following operations:
//   - Step 1: Returns original string if width is <= 0
//   - Step 2: Calculates current display width (accounting for unicode)
//   - Step 3: Returns original string if already at or exceeds target width
//   - Step 4: Pads with spaces to reach target width
//
// Parameters:
//   - val: The string to pad
//   - width: The target display width in character cells (must be > 0 to have effect)
//
// Returns:
//   - string: The padded string, or original if already wide enough or width <= 0
func ToWidth(val string, width int) string {
	if width <= 0 {
		return val
	}
	current := DisplayWidth(val)
	if current >= width {
		return val
	}
	return val + strings.Repeat(" ", width-current)
}

// Max returns the maximum value from a list of integers.
//
// If the slice is empty, returns 0. Otherwise returns the largest integer
// from the provided values.
//
// Parameters:
//   - values: Variable number of integers to compare
//
// Returns:
//   - int: The maximum value from the input, or 0 if no values provided
func Max(values ...int) int {
	m := 0
	for i, v := range values {
		if i == 0 || v > m {
			m = v
		}
	}
	return m
}
