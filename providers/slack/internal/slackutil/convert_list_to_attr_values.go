package slackutil

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConvertListToAttrValues converts a types.List containing any values into a slice of attr.Value.
//
// It checks if the provided list is null or unknown and returns nil if so.
// For each element in the list, it attempts to extract an attr.Value.
// If the value cannot be converted, an error is returned.
// Only non-null and non-unknown values are included in the resulting slice.
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
//	output, err := ConvertListToAttrValues(listVal)
//	// output will be []attr.Value with converted values, err will be nil if successful.
//
// Returns:
//
//	A slice of attr.Value extracted from the list elements or nil if the list is null/unknown.
//	An error if an unexpected value type is encountered.
// func ConvertListToAttrValues(list types.List) ([]attr.Value, error) {
// 	var newList []attr.Value

// 	// Convert the list elements to attr.Value
// 	diags := list.ElementsAs(context.Background(), &newList, false)
// 	if diags.HasError() {
// 		return nil, fmt.Errorf("failed to convert list elements: %s", diags)
// 	}

// 	return newList, nil
// }

func ConvertListToAttrValues(list types.List) ([]string, error) {
	var newList []string

	// Convert the list elements to a slice of strings
	diags := list.ElementsAs(context.Background(), &newList, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert list elements: %s", diags)
	}

	return newList, nil
}
