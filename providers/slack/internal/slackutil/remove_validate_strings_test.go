package slackutil

import (
	"testing"
)

func TestRemoveAndValidateStrings(t *testing.T) {
	tests := []struct {
		name      string
		target    []string
		toRemove  [][]string
		expected  []string
		expectErr bool
	}{
		{
			name:      "RemoveStringsSuccessfully",
			target:    []string{"userA", "userB", "userC", "userD"},
			toRemove:  [][]string{{"userB", "userD"}},
			expected:  []string{"userA", "userC"},
			expectErr: false,
		},
		{
			name:      "RemoveMultipleDuplicateStrings",
			target:    []string{"userA", "userB", "userC", "userD"},
			toRemove:  [][]string{{"userB"}, {"userD"}},
			expected:  []string{"userA", "userC"},
			expectErr: false,
		},
		{
			name:      "RemoveNoStrings",
			target:    []string{"userA", "userB", "userC", "userD"},
			toRemove:  [][]string{},
			expected:  []string{"userA", "userB", "userC", "userD"},
			expectErr: false,
		},
		{
			name:      "RemoveNonExistentStrings",
			target:    []string{"userA", "userB", "userC"},
			toRemove:  [][]string{{"userX", "userY"}},
			expected:  []string{"userA", "userB", "userC"},
			expectErr: false,
		},
		{
			name:      "IgnoreEmptyStringsInTarget",
			target:    []string{"userA", "userB", "", "userC"},
			toRemove:  [][]string{{"userB"}},
			expected:  []string{"userA", "userC"},
			expectErr: false,
		},
		{
			name:      "HandleNilTargetSlice",
			target:    nil,
			toRemove:  [][]string{{"userA", "userB"}},
			expected:  nil,
			expectErr: true,
		},
		{
			name:      "HandleNilRemoveList",
			target:    []string{"userA", "userB"},
			toRemove:  [][]string{nil},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remainingList, err := RemoveAndValidateStrings(tt.target, tt.toRemove...)

			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				if len(remainingList) != len(tt.expected) {
					t.Fatalf("expected length: %d, got: %d", len(tt.expected), len(remainingList))
				}

				for i, v := range tt.expected {
					if remainingList[i] != v {
						t.Errorf("expected: %s, got: %s", v, remainingList[i])
					}
				}
			}
		})
	}
}
