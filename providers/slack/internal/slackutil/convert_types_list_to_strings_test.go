package slackutil

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTypesListToStrings(t *testing.T) {
	tests := []struct {
		name        string
		input       types.List
		expected    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "NullInput",
			input:       types.ListNull(basetypes.StringType{}),
			expected:    nil,
			expectError: true,
			errorMsg:    "input list is null or unknown",
		},
		{
			name: "ValidList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewStringValue("second"),
						basetypes.NewStringValue("third"),
					},
				)
				return listValue
			}(),
			expected:    []string{"first", "second", "third"},
			expectError: false,
		},
		{
			name: "EmptyList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{},
				)
				return listValue
			}(),
			expected:    nil,
			expectError: true,
			errorMsg:    "no valid non-null strings found",
		},
		{
			name: "ListWithMixedTypes",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.DynamicType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewBoolValue(true),
					},
				)
				return listValue
			}(),
			expected:    nil,
			expectError: true,
			errorMsg:    "input list is null or unknown",
		},
		{
			name: "ListWithNullAndValidValues",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{
						basetypes.NewStringValue("valid"),
						basetypes.NewStringNull(),
					},
				)
				return listValue
			}(),
			expected:    []string{"valid"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertTypesListToStrings(tt.input)

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
