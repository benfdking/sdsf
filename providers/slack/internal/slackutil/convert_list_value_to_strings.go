package slackutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConvertListValueToStrings converts a types.List containing string values into a slice of strings.
//
// It checks if the provided list is null or unknown and returns nil if so.
// For each element in the list, it attempts to extract a string value.
// If the value is not of the expected type, an error is returned.
// Only non-null and non-unknown string values are included in the resulting slice.
//
// Sample Input:
//
//	listVal := types.List{
//	    Elements: []types.Value{
//	        types.String{Value: "first"},
//	        types.String{Value: "second"},
//	        types.String{Value: "third"},
//	    },
//	}
//
// Sample Output:
//
//	output, err := ConvertListValueToStrings(listVal)
//	// output will be []string{"first", "second", "third"}, err will be nil if successful.
//
// Returns:
//
//	A slice of strings extracted from the list elements or nil if the list is null/unknown.
//	An error if an unexpected value type is encountered.
func ConvertListValueToStrings(listVal types.List) ([]string, error) {
	// Check if the list is null or unknown.
	if listVal.IsNull() || listVal.IsUnknown() {
		return nil, nil
	}

	// If the list is empty, return an empty slice instead of nil.
	if len(listVal.Elements()) == 0 {
		return []string{}, nil
	}

	// Prepare a slice to store the string elements.
	var result []string

	// Iterate over the list elements.
	for _, v := range listVal.Elements() {
		// Attempt to extract a string from the value.
		strVal, ok := v.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected string value, got %T", v)
		}

		// Append the actual string value to the result slice if valid.
		if !strVal.IsNull() && !strVal.IsUnknown() {
			result = append(result, strVal.ValueString())
		}
	}

	return result, nil
}
