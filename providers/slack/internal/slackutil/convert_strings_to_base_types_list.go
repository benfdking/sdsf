package slackutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConvertStringsToBasetypesList converts a slice of strings to a basetypes.ListValue.
//
// This function takes a slice of strings as input and converts each string into an
// `attr.Value` of type `String`. If the input slice is empty, it returns an empty `ListValue`
// without an error. If the input is nil, an error is returned.
//
// Parameters:
//   - slice: A slice of strings to be converted into a ListValue.
//
// Returns:
//   - A basetypes.ListValue that contains the converted string values as attr.Values.
//   - An error if the input slice is empty.
//
// Example usage:
//
//	strings := []string{"foo", "bar", "baz"}
//	listValue, err := ConvertStringsToBasetypesList(strings)
//	if err != nil {
//	    fmt.Println("Error:", err)
//	    return
//	}
//	fmt.Println(listValue) // Output: [foo bar baz]
func ConvertStringsToBasetypesList(slice []string) (basetypes.ListValue, error) {
	// Check if the input slice is nil
	if slice == nil {
		return basetypes.ListValue{}, fmt.Errorf("input slice cannot be nil")
	}

	// Return an empty ListValue if the slice is empty
	if len(slice) == 0 {
		return types.ListValueMust(types.StringType, []attr.Value{}), nil
	}

	// Convert each string to an attr.Value and build the result ListValue
	result := make([]attr.Value, len(slice))
	for i, v := range slice {
		result[i] = types.StringValue(v)
	}

	return types.ListValueMust(types.StringType, result), nil
}
