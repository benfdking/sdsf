package slackutil

import "testing"

// TestCompareStrings tests the CompareStrings function with different comparison modes.
func TestCompareStrings(t *testing.T) {
	tests := []struct {
		name     string
		list1    []string
		list2    []string
		mode     CompareMode
		expected bool
		err      bool
	}{
		{
			name:     "AnyTrue",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"banana", "grape"},
			mode:     Any,
			expected: true,
			err:      false,
		},
		{
			name:     "AnyFalse",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"grape", "orange"},
			mode:     Any,
			expected: false,
			err:      false,
		},
		{
			name:     "MatchTrue",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"banana", "apple"},
			mode:     Match,
			expected: true,
			err:      false,
		},
		{
			name:     "MatchFalse",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"banana", "grape"},
			mode:     Match,
			expected: false,
			err:      false,
		},
		{
			name:     "NotMatchTrue",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"grape", "orange"},
			mode:     NotMatch,
			expected: true,
			err:      false,
		},
		{
			name:     "NotMatchFalse",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"banana", "grape"},
			mode:     NotMatch,
			expected: false,
			err:      false,
		},
		{
			name:     "ExactMatchTrue",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"apple", "banana", "cherry"},
			mode:     ExactMatch,
			expected: true,
			err:      false,
		},
		{
			name:     "ExactMatchFalse",
			list1:    []string{"apple", "banana", "cherry"},
			list2:    []string{"banana", "apple"},
			mode:     ExactMatch,
			expected: false,
			err:      false,
		},
		{
			name:     "SubsetTrue",
			list1:    []string{"apple", "banana", "cherry", "orange"},
			list2:    []string{"banana", "apple"},
			mode:     Subset,
			expected: true,
			err:      false,
		},
		{
			name:     "SubsetFalse",
			list1:    []string{"apple", "banana"},
			list2:    []string{"grape", "apple"},
			mode:     Subset,
			expected: false,
			err:      false,
		},
		{
			name:     "SupersetTrue",
			list1:    []string{"apple", "banana", "cherry", "orange"},
			list2:    []string{"banana", "apple"},
			mode:     Superset,
			expected: true,
			err:      false,
		},
		{
			name:     "SupersetFalse",
			list1:    []string{"apple", "banana"},
			list2:    []string{"grape", "apple"},
			mode:     Superset,
			expected: false,
			err:      false,
		},
		{
			name:     "NoneTrue",
			list1:    []string{},
			list2:    []string{},
			mode:     None,
			expected: true,
			err:      false,
		},
		{
			name:     "NoneFalse",
			list1:    []string{"apple"},
			list2:    []string{},
			mode:     None,
			expected: false,
			err:      false,
		},
		{
			name:     "NilListError",
			list1:    nil,
			list2:    []string{"apple"},
			mode:     Any,
			expected: false,
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareStrings(tt.list1, tt.list2, tt.mode)
			if (err != nil) != tt.err {
				t.Fatalf("Expected error: %v, got: %v", tt.err, err)
			}
			if result != tt.expected {
				t.Errorf("Expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}
