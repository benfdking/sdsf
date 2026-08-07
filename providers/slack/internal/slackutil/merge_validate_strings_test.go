package slackutil

import (
	"testing"
)

func TestMergeAndValidateStrings(t *testing.T) {
	tests := []struct {
		name      string
		input     [][]string
		expected  []string
		expectErr bool
	}{
		{
			name:      "MergeNonDuplicateStrings",
			input:     [][]string{{"userA", "userB"}, {"userC", "userD"}},
			expected:  []string{"userA", "userB", "userC", "userD"},
			expectErr: false,
		},
		{
			name:      "MergeWithDuplicates",
			input:     [][]string{{"userA", "userB"}, {"userB", "userC"}},
			expected:  []string{"userA", "userB", "userC"},
			expectErr: false,
		},
		{
			name:      "IgnoreEmptyStrings",
			input:     [][]string{{"userA", "", "userB"}, {"", "userC"}},
			expected:  []string{"userA", "userB", "userC"},
			expectErr: false,
		},
		{
			name:      "HandleNilSlice",
			input:     [][]string{{"userA", "userB"}, nil},
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "MultipleSlicesWithEmptyStrings",
			input:     [][]string{{"userA", "userB"}, {"userB", "", "userC"}, {"userD", "userE"}},
			expected:  []string{"userA", "userB", "userC", "userD", "userE"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedList, err := MergeAndValidateStrings(tt.input...)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				if len(mergedList) != len(tt.expected) {
					t.Fatalf("expected length: %d, got: %d", len(tt.expected), len(mergedList))
				}

				for i, v := range tt.expected {
					if mergedList[i] != v {
						t.Errorf("expected: %s, got: %s", v, mergedList[i])
					}
				}
			}
		})
	}
}
