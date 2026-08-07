package slackutil

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAndValidateStringLists(t *testing.T) {
	tests := []struct {
		name        string
		input       []types.List
		expected    StringListType
		expectError bool
		errorMsg    string
	}{
		{
			name: "NullInput",
			input: []types.List{
				types.ListNull(basetypes.StringType{}),
			},
			expected:    StringListType{},
			expectError: true,
			errorMsg:    "input list is null or unknown",
		},
		{
			name: "ValidLists",
			input: []types.List{
				func() types.List {
					listValue, _ := basetypes.NewListValue(
						basetypes.StringType{},
						[]attr.Value{
							basetypes.NewStringValue("userA"),
							basetypes.NewStringValue("userB"),
						},
					)
					return listValue
				}(),
				func() types.List {
					listValue, _ := basetypes.NewListValue(
						basetypes.StringType{},
						[]attr.Value{
							basetypes.NewStringValue("userB"), // Duplicate value
							basetypes.NewStringValue("userC"),
						},
					)
					return listValue
				}(),
			},
			expected: StringListType{
				Elements: []types.String{
					basetypes.NewStringValue("userA"),
					basetypes.NewStringValue("userB"),
					basetypes.NewStringValue("userC"),
				},
			},
			expectError: false,
		},
		{
			name: "ListWithInvalidElementType",
			input: []types.List{
				func() types.List {
					listValue, _ := basetypes.NewListValue(
						basetypes.StringType{},
						[]attr.Value{
							basetypes.NewInt64Value(123), // Invalid element type
						},
					)
					return listValue
				}(),
			},
			expected:    StringListType{},
			expectError: true,
			errorMsg:    "input list is null or unknown",
		},
		{
			name: "ListWithNullAndValidValues",
			input: []types.List{
				func() types.List {
					listValue, _ := basetypes.NewListValue(
						basetypes.StringType{},
						[]attr.Value{
							basetypes.NewStringValue("valid"),
							basetypes.NewStringNull(),
						},
					)
					return listValue
				}(),
			},
			expected: StringListType{
				Elements: []types.String{
					basetypes.NewStringValue("valid"),
				},
			},
			expectError: false,
		},
		{
			name: "EmptyList",
			input: []types.List{
				func() types.List {
					listValue, _ := basetypes.NewListValue(
						basetypes.StringType{},
						[]attr.Value{},
					)
					return listValue
				}(),
			},
			expected: StringListType{},
			errorMsg: "input list is null or unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeAndValidateStringLists(tt.input...)

			if tt.expectError {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tt.expected.Elements, got.Elements)
			}
		})
	}
}
