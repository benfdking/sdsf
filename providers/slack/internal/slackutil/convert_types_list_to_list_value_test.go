package slackutil

import (
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTypesListToListValue(t *testing.T) {
	tests := []struct {
		name        string
		input       types.List
		expected    ListValue
		expectError bool
		errorMsg    string
	}{
		{
			name:        "NullInput",
			input:       types.ListNull(basetypes.StringType{}),
			expected:    ListValue{},
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
			expected: ListValue{
				Elements: []StringValue{
					{Content: "first"},
					{Content: "second"},
					{Content: "third"},
				},
			},
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
			expected:    ListValue{},
			expectError: true,
			errorMsg:    "no valid non-null strings found",
		},
		{
			name: "MixedTypesList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.DynamicType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewBoolValue(true),
						basetypes.NewNumberValue(big.NewFloat(123)),
					},
				)
				return listValue
			}(),
			expected:    ListValue{},
			expectError: true,
			errorMsg:    "input list is null or unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertTypesListToListValue(tt.input)

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
