package slackutil_test

import (
	"github.com/benfdking/sdsf/providers/slack/internal/slackutil"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestConvertBasetypesListToStrings(t *testing.T) {
	tests := []struct {
		name      string
		input     []basetypes.StringValue
		expected  []string
		expectErr bool
	}{
		{
			name: "ValidInputWithMultipleStrings",
			input: []basetypes.StringValue{
				basetypes.NewStringValue("first"),
				basetypes.NewStringValue("second"),
				basetypes.NewStringValue("third"),
			},
			expected:  []string{"first", "second", "third"},
			expectErr: false,
		},
		{
			name:      "NilinputSlice",
			input:     nil,
			expected:  nil,
			expectErr: true,
		},
		{
			name: "InputWithEmptyStrings",
			input: []basetypes.StringValue{
				basetypes.NewStringValue(""),
				basetypes.NewStringValue(""),
			},
			expected:  nil,
			expectErr: true,
		},
		{
			name: "MixedValidAndEmptyStrings",
			input: []basetypes.StringValue{
				basetypes.NewStringValue("first"),
				basetypes.NewStringValue(""),
				basetypes.NewStringValue("third"),
			},
			expected:  []string{"first", "third"},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := slackutil.ConvertBasetypesListToStrings(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, output)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, output)
			}
		})
	}
}
