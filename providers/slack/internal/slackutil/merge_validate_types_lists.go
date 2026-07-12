package slackutil

import (
	"context"
	"errors"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// MergeAndValidateStringLists merges multiple types.List inputs into a single StringListType, removing duplicates and sorting the result.
//
// It checks if any of the input lists are null or unknown, returning an error if so. Additionally, it ensures that each list contains only string elements.
// Non-string elements will result in an error. After validation, the function removes duplicate string values and sorts the merged list in alphabetical order.
//
// Sample Input:
//
//	list1 := types.List{
//	    Elems: []types.Value{
//	        basetypes.NewStringValue("userA"),
//	        basetypes.NewStringValue("userB"),
//	    },
//	    ElementType: types.StringType,
//	}
//
//	list2 := types.List{
//	    Elems: []types.Value{
//	        basetypes.NewStringValue("userB"), // Duplicate value
//	        basetypes.NewStringValue("userC"),
//	    },
//	    ElementType: types.StringType,
//	}
//
// Sample Output:
//
//	mergedList, err := MergeAndValidateStringLists(list1, list2)
//	// mergedList.Elements will contain []types.String{"userA", "userB", "userC"}.
//
// Returns:
//
//	A StringListType struct containing a sorted slice of unique strings extracted from the input lists.
//	If any input list is null, unknown, or contains non-string elements, an error is returned.
func MergeAndValidateStringLists(lists ...types.List) (StringListType, error) {
	// Prepare a map to track unique string elements.
	uniqueStrings := make(map[string]struct{})

	for _, list := range lists {
		// Check if the list is null or unknown.
		if list.IsNull() || list.IsUnknown() {
			return StringListType{}, errors.New("input list is null or unknown")
		}

		// Check for element type.
		if list.ElementType(context.Background()) != types.StringType {
			return StringListType{}, errors.New("expected list of strings")
		}

		// Extract the elements from the list and add them to the map for uniqueness.
		for _, v := range list.Elements() {
			strVal, ok := v.(types.String)
			if !ok {
				return StringListType{}, errors.New("expected string value")
			}

			if !strVal.IsNull() && !strVal.IsUnknown() {
				uniqueStrings[strVal.ValueString()] = struct{}{}
			}
		}
	}

	// Convert the map keys (unique strings) into a sorted slice.
	finalList := make([]types.String, 0, len(uniqueStrings))
	for item := range uniqueStrings {
		finalList = append(finalList, basetypes.NewStringValue(item))
	}

	// Sort the list alphabetically.
	sort.Slice(finalList, func(i, j int) bool {
		return finalList[i].ValueString() < finalList[j].ValueString()
	})

	// Return the merged and validated list.
	return StringListType{Elements: finalList}, nil
}
