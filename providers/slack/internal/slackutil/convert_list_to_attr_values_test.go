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

func TestConvertListToAttrValues(t *testing.T) {
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
			name: "ValidStringList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.StringType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewStringValue("second"),
					},
				)
				return listValue
			}(),
			expected:    []string{"first", "second"},
			expectError: false,
		},
		{
			name: "MixedTypesinList",
			input: func() types.List {
				listValue, _ := basetypes.NewListValue(
					basetypes.DynamicType{},
					[]attr.Value{
						basetypes.NewStringValue("first"),
						basetypes.NewNumberValue(big.NewFloat(42)),
					},
				)
				return listValue
			}(),
			expected:    nil,
			expectError: true,
			errorMsg:    "failed to convert list elements",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertListToAttrValues(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
