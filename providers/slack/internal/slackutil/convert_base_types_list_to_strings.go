package slackutil

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConvertBasetypesListToStrings converts a slice of basetypes.StringValue into a slice of strings.
//
// This function takes a slice of basetypes.StringValue and extracts the non-empty string content
// from each element. It returns a slice of strings containing the valid values.
//
// If the input list is nil, it returns an error indicating that the input list is invalid.
// If all elements in the input list are either empty or invalid, an error is returned,
// indicating that no valid strings were found.
//
// Parameters:
//   - list: A slice of basetypes.StringValue containing the string values to be converted.
//
// Returns:
//   - A slice of strings extracted from the input slice.
//   - An error if the input list is nil or if no valid strings are found.
func ConvertBasetypesListToStrings(list []basetypes.StringValue) ([]string, error) {
	// Check if the list is nil.
	if list == nil {
		return nil, errors.New("input list is nil")
	}

	// Prepare a slice to store the string elements.
	strElems := make([]string, 0, len(list))

	// Iterate over the slice elements.
	for _, v := range list {
		// Append the actual string value to the result slice if valid (non-empty).
		if v.ValueString() != "" {
			strElems = append(strElems, v.ValueString())
		}
	}

	// If no valid strings were found, return an error.
	if len(strElems) == 0 {
		return nil, errors.New("no valid non-empty strings found")
	}

	// Return the slice of valid strings.
	return strElems, nil
}
