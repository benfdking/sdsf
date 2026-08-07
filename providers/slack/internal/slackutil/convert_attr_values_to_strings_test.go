package slackutil_test

import (
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestConvertAttrValuesToStrings(t *testing.T) {
	tests := []struct {
		name        string
		input       []attr.Value
		expected    []string
		expectError bool
	}{
		{
			name:        "emptyInputList",
			input:       []attr.Value{},
			expected:    nil,
			expectError: true,
		},
		{
			name: "ValidInputList",
			input: []attr.Value{
				basetypes.NewStringValue("first"),
				basetypes.NewStringValue("second"),
			},
			expected:    []string{"first", "second"},
			expectError: false,
		},
		{
			name: "ListWithEmptyStringAndNullValue",
			input: []attr.Value{
				basetypes.NewStringValue("first"),
				basetypes.NewStringValue(""),
				basetypes.StringValue{},
				basetypes.NewStringValue("second"),
			},
			expected:    []string{"first", "second"},
			expectError: false,
		},
		{
			name: "ListWithOnlyNullAndEmptyValues",
			input: []attr.Value{
				basetypes.NewStringValue(""),
				basetypes.StringValue{},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "ListWithNonStringValues",
			input: []attr.Value{
				basetypes.NewBoolValue(true),
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := slackutil.ConvertAttrValuesToStrings(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
