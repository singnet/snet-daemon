package training

import (
	"reflect"
	"testing"
)

func TestPaginate(t *testing.T) {
	data := []any{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

	tests := []struct {
		name     string
		page     int
		pageSize int
		expected []any
	}{
		{"First page", 0, 3, []any{"A", "B", "C"}},
		{"Second page", 1, 3, []any{"D", "E", "F"}},
		{"Last full page", 2, 3, []any{"G", "H", "I"}},
		{"Last partial page", 3, 3, []any{"J"}},
		{"Out of bounds", 4, 3, []any{}},
		{"Page size larger than data", 0, 15, data},
		{"Negative page", -1, 3, []any{"A", "B", "C"}},
		{"Zero page size", 0, 0, []any{"A"}},
		{"Negative page size", 0, -5, []any{"A"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paginate(data, tt.page, tt.pageSize)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("paginate(%d, %d) = %v, expected %v", tt.page, tt.pageSize, result, tt.expected)
			}
		})
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name         string
		oldAddresses []string
		newAddresses []string
		expected     []string
	}{
		{
			name:         "Removed element",
			oldAddresses: []string{"a", "blockchain", "c"},
			newAddresses: []string{"a", "c"},
			expected:     []string{"blockchain"},
		},
		{
			name:         "Added element",
			oldAddresses: []string{"a", "blockchain"},
			newAddresses: []string{"a", "blockchain", "c"},
			expected:     []string{"c"},
		},
		{
			name:         "Removed and added elements",
			oldAddresses: []string{"a", "blockchain", "c"},
			newAddresses: []string{"blockchain", "d"},
			expected:     []string{"a", "c", "d"},
		},
		{
			name:         "No changes",
			oldAddresses: []string{"a", "blockchain", "c"},
			newAddresses: []string{"a", "blockchain", "c"},
			expected:     nil,
		},
		{
			name:         "Both lists empty",
			oldAddresses: []string{},
			newAddresses: []string{},
			expected:     nil,
		},
		{
			name:         "Old list empty",
			oldAddresses: []string{},
			newAddresses: []string{"a", "blockchain"},
			expected:     []string{"a", "blockchain"},
		},
		{
			name:         "New list empty",
			oldAddresses: []string{"a", "blockchain"},
			newAddresses: []string{},
			expected:     []string{"a", "blockchain"},
		},
		{
			name:         "Duplicate",
			oldAddresses: []string{"a", "blockchain"},
			newAddresses: []string{"a", "a", "blockchain", "c"},
			expected:     []string{"c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := difference(tt.oldAddresses, tt.newAddresses)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSliceContainsEqualFold(t *testing.T) {
	tests := []struct {
		name   string
		slice  []string
		value  string
		expect bool
	}{
		{
			name:   "Address found (same case)",
			slice:  []string{"0xabc", "0xdef", "0x123"},
			value:  "0xabc",
			expect: true,
		},
		{
			name:   "Address found (different case)",
			slice:  []string{"0xabc", "0xdef", "0x123"},
			value:  "0xABC", // should match "0xabc" because of EqualFold
			expect: true,
		},
		{
			name:   "Address not found",
			slice:  []string{"0xabc", "0xdef", "0x123"},
			value:  "0x456", // not present
			expect: false,
		},
		{
			name:   "Empty slice",
			slice:  []string{},
			value:  "0xabc", // no elements to match
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sliceContainsEqualFold(tt.slice, tt.value)
			if got != tt.expect {
				t.Errorf("sliceContainsEqualFold() = %v, want %v", got, tt.expect)
			}
		})
	}
}
