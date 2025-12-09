package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"ascii string", "hello", 5},
		{"with emoji", "test🟢", 6},
		{"unicode chars", "日本語", 6},
		{"mixed", "abc日本", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DisplayWidth(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToWidth(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		width    int
		expected string
	}{
		{"zero width", "test", 0, "test"},
		{"negative width", "test", -1, "test"},
		{"exact width", "test", 4, "test"},
		{"longer than width", "testing", 4, "testing"},
		{"needs padding", "test", 8, "test    "},
		{"empty string", "", 4, "    "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToWidth(tt.val, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		values   []int
		expected int
	}{
		{"empty", []int{}, 0},
		{"single value", []int{5}, 5},
		{"multiple values", []int{1, 5, 3}, 5},
		{"negative values", []int{-1, -5, -3}, -1},
		{"mixed", []int{-1, 0, 5, 3}, 5},
		{"first is max", []int{10, 5, 3}, 10},
		{"last is max", []int{1, 2, 10}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Max(tt.values...)
			assert.Equal(t, tt.expected, result)
		})
	}
}
