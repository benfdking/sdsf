package slackutil

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConvertTypesListToListValue converts a types.List into a ListValue, returning an error for invalid input.
//
// It checks if the provided list is null or unknown and returns an error if so.
// For each non-null and non-unknown element in the list, it creates a corresponding StringValue.
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
//	output, err := ConvertTypesListToListValue(listVal)
//	// output will contain ListValue with Elements corresponding to the non-null input strings.
//
// Returns:
//
//	A ListValue containing StringValue elements extracted from the input list.
//	If the input list is null or unknown, an error is returned.
//	If all elements are null or unknown, an error is returned.

// ConvertTypesListToListValue converts a types.List into a ListValue, returning an error for invalid input.
func ConvertTypesListToListValue(list types.List) (ListValue, error) {
	// Check if the list is null or unknown.
	if list.IsNull() || list.IsUnknown() {
		return ListValue{}, fmt.Errorf("input list is null or unknown")
	}

	// Prepare a slice to store the string elements.
	listElems := make([]StringValue, 0, len(list.Elements()))

	// Iterate over the list elements.
	for _, v := range list.Elements() {
		// Attempt to extract a string from the value.
		strVal, ok := v.(types.String)
		if !ok {
			return ListValue{}, fmt.Errorf("expected string value, got %T", v)
		}

		// Append the actual string value to the result slice if valid.
		if !strVal.IsNull() && !strVal.IsUnknown() {
			listElems = append(listElems, StringValue{Content: strVal.ValueString()})
		}
	}

	// If no valid strings were found, return an error.
	if len(listElems) == 0 {
		return ListValue{}, errors.New("no valid non-null strings found")
	}

	// Create and return a ListValue containing the valid string elements.
	return ListValue{Elements: listElems}, nil
}
