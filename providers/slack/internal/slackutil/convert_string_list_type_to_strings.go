package slackutil

import (
	"fmt"
)

// ConvertStringListTypeToStrings converts a StringListType into a slice of strings.
//
// It iterates over the elements of the StringListType and extracts the string values,
// returning a slice of non-null and non-unknown strings. If all elements are null or
// unknown, an error is returned.
//
// Sample Input:
//
//	stringList := StringListType{
//	    Elements: []types.String{
//	        basetypes.NewStringValue("userA"),
//	        basetypes.NewStringValue("userB"),
//	        basetypes.NewStringValue("userC"),
//	    },
//	}
//
// Sample Output:
//
//	output := ConvertStringListTypeToStrings(stringList)
//	// output will contain []string{"userA", "userB", "userC"}.
//
// Returns:
//
//	A slice of strings extracted from the input StringListType.
//	If all elements are null or unknown, an error is returned.
func ConvertStringListTypeToStrings(stringList StringListType) ([]string, error) {
	// Prepare a slice to store the string values.
	strElems := make([]string, 0, len(stringList.Elements))

	// Iterate over the elements of the StringListType.
	for _, strValue := range stringList.Elements {
		// Check if the string value is not null or unknown.
		if !strValue.IsNull() && !strValue.IsUnknown() {
			strElems = append(strElems, strValue.ValueString())
		}
	}

	// If no valid strings were found, return an error.
	if len(strElems) == 0 {
		return nil, fmt.Errorf("no valid non-null strings found")
	}

	// Return the slice of valid strings.
	return strElems, nil
}
