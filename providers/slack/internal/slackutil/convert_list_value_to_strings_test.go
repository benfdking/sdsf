package slackutil

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertListValueToStrings(t *testing.T) {
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
			expectError: false,
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
			name: "ListWithNullElements",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewStringNull(),
						basetypes.NewStringValue("third"),
					},
				)
				return listValue
			}(),
			expected:    []string{"first", "third"},
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
			expected:    []string{},
			expectError: false,
		},
		{
			name: "InvalidTypeInList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewBoolValue(true),
					},
				)
				return listValue
			}(),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertListValueToStrings(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.EqualError(t, err, tt.errorMsg)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
