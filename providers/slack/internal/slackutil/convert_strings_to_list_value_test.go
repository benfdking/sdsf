package slackutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertStringsToListValue(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    ListValue
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyInput",
			input:       []string{},
			expected:    ListValue{Null: true},
			expectError: false,
		},
		{
			name: "ValidInput",
			input: []string{
				"first", "second", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "first"},
					{Content: "second"},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name: "InputWithEmptyString",
			input: []string{
				"first", "", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "first"},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name: "InputWithWhitespace",
			input: []string{
				"  first", "second  ", "  ", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "  first"},
					{Content: "second  "},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name: "MixedValidAndInvalidStrings",
			input: []string{
				"first", "\n", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "first"},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name: "SpecialCharacters",
			input: []string{
				"!@#$%^", "second", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "!@#$%^"},
					{Content: "second"},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name: "StringsWithNumbers",
			input: []string{
				"123", "456", "789",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "123"},
					{Content: "456"},
					{Content: "789"},
				},
			},
			expectError: false,
		},
		{
			name: "AllEmptyStrings",
			input: []string{
				"", "", "",
			},
			expected: ListValue{
				Null:     false,
				Elements: nil,
			},
			expectError: true,
			errorMsg:    "no valid non-empty strings found",
		},
		{
			name: "ValidAndWhitespaceStrings",
			input: []string{
				"first", " ", "", "second", "\t", "third",
			},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: "first"},
					{Content: "second"},
					{Content: "third"},
				},
			},
			expectError: false,
		},
		{
			name:        "WhitespaceOnly",
			input:       []string{" ", "\t", "\n"},
			expectError: true,
			errorMsg:    "no valid non-empty strings found",
		},
		{
			name:  "LongStringInput",
			input: []string{strings.Repeat("a", 1000)},
			expected: ListValue{
				Null: false,
				Elements: []StringValue{
					{Content: strings.Repeat("a", 1000)},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringsToListValue(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Null, result.Null)
				assert.Equal(t, len(tt.expected.Elements), len(result.Elements))

				for i, expectedElement := range tt.expected.Elements {
					assert.Equal(t, expectedElement.Content, result.Elements[i].Content)
				}
			}
		})
	}
}
