package slackutil

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConvertTypesListToStrings converts a types.List into a slice of strings, returning an error for invalid input.
//
// It checks if the provided list is null or unknown and returns an error if so.
// For each non-null and non-unknown element in the list, it extracts the corresponding string value.
// If all elements are null or unknown, an error is returned.
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
//	output, err := ConvertTypesListToStrings(listVal)
//	// output will contain []string{"first", "second", "third"}.
//
// Returns:
//
//	A slice of strings extracted from the input list.
//	If the input list is null or unknown, an error is returned.
//	If all elements are null or unknown, an error is returned.
func ConvertTypesListToStrings(list types.List) ([]string, error) {
	// Check if the list is null or unknown.
	if list.IsNull() || list.IsUnknown() {
		return nil, fmt.Errorf("input list is null or unknown")
	}

	// Prepare a slice to store the string elements.
	strElems := make([]string, 0, len(list.Elements()))

	// Iterate over the list elements.
	for _, v := range list.Elements() {
		// Attempt to extract a string from the value.
		strVal, ok := v.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected string value, got %T", v)
		}

		// Append the actual string value to the result slice if valid.
		if !strVal.IsNull() && !strVal.IsUnknown() {
			strElems = append(strElems, strVal.ValueString())
		}
	}

	// If no valid strings were found, return an error.
	if len(strElems) == 0 {
		return nil, errors.New("no valid non-null strings found")
	}

	// Return the slice of valid strings.
	return strElems, nil
}
