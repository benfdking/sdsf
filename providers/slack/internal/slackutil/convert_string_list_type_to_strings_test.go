package slackutil

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStringListTypeToStrings(t *testing.T) {
	tests := []struct {
		name        string
		input       StringListType
		expected    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyList",
			input:       StringListType{Elements: []types.String{}},
			expected:    nil,
			expectError: true,
			errorMsg:    "no valid non-null strings found",
		},
		{
			name: "ListWithNullValues",
			input: StringListType{
				Elements: []types.String{
					basetypes.NewStringNull(),
					basetypes.NewStringNull(),
				},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "no valid non-null strings found",
		},
		{
			name: "ValidListWithUniqueStrings",
			input: StringListType{
				Elements: []types.String{
					basetypes.NewStringValue("userA"),
					basetypes.NewStringValue("userB"),
					basetypes.NewStringValue("userC"),
				},
			},
			expected:    []string{"userA", "userB", "userC"},
			expectError: false,
		},
		{
			name: "ListWithMixedNullAndValidValues",
			input: StringListType{
				Elements: []types.String{
					basetypes.NewStringValue("userA"),
					basetypes.NewStringNull(),
					basetypes.NewStringValue("userB"),
				},
			},
			expected:    []string{"userA", "userB"},
			expectError: false,
		},
		{
			name: "ListWithUnknownValues",
			input: StringListType{
				Elements: []types.String{
					basetypes.NewStringValue("userA"),
					basetypes.NewStringUnknown(),
					basetypes.NewStringValue("userB"),
				},
			},
			expected:    []string{"userA", "userB"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertStringListTypeToStrings(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
