package slackutil

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertStringsToBasetypesList(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		expected  basetypes.ListValue
		expectErr bool
	}{
		{
			name:  "ValidStrings",
			input: []string{"foo", "bar", "baz"},
			expected: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("foo"),
				types.StringValue("bar"),
				types.StringValue("baz"),
			}),
			expectErr: false,
		},
		{
			name:      "EmptyInput",
			input:     []string{},
			expected:  types.ListValueMust(types.StringType, []attr.Value{}),
			expectErr: false,
		},
		{
			name:  "SingleItem",
			input: []string{"single"},
			expected: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("single"),
			}),
			expectErr: false,
		},
		{
			name:      "NilInput",
			input:     nil,
			expected:  basetypes.ListValue{},
			expectErr: true,
		},
		{
			name:  "WhitespaceStrings",
			input: []string{" ", "\t", "\n"},
			expected: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue(" "),
				types.StringValue("\t"),
				types.StringValue("\n"),
			}),
			expectErr: false,
		},
		{
			name:  "NilElementInList",
			input: []string{"valid", "", "another"},
			expected: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("valid"),
				types.StringValue(""),
				types.StringValue("another"),
			}),
			expectErr: false,
		},
		{
			name:  "LongStrings",
			input: []string{"a very long string that exceeds typical lengths", "another long string"},
			expected: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("a very long string that exceeds typical lengths"),
				types.StringValue("another long string"),
			}),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ConvertStringsToBasetypesList(tt.input)

			if tt.expectErr {
				require.Error(t, err)
				assert.Empty(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}
