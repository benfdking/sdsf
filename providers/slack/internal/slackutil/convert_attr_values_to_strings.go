package slackutil

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ConvertAttrValuesToStrings extracts non-empty string values from a slice of attr.Value elements.
//
// This function processes a slice of attr.Value elements and expects each element to be of
// the type basetypes.StringValue. If all elements are of the expected type, it converts them
// into a slice of strings, excluding any null, unknown, or empty values. This utility is
// especially useful for handling Terraform attribute lists that are intended to represent
// lists of strings.
//
// Parameters:
//   - listAttr: A slice of attr.Value containing elements expected to be of type basetypes.StringValue.
//
// Returns:
//   - []string: A slice of strings with all valid, non-empty values from the input slice.
//   - error: Returns an error if any of the following conditions are met:
//     1. listAttr is empty or nil.
//     2. An element within listAttr is not of type basetypes.StringValue.
//     3. All elements in listAttr are null, unknown, or empty strings.
//
// Usage example:
//
//	strValues, err := slackutil.ConvertAttrValuesToStrings(state.Types.Elements())
//	if err != nil {
//	    log.Println("Error:", err)
//	} else {
//	    log.Println("Converted string values:", strValues)
//	}
func ConvertAttrValuesToStrings(listAttr []attr.Value) ([]string, error) {
	// Check if the input slice is empty.
	if len(listAttr) == 0 {
		return nil, errors.New("input slice is nil or empty")
	}

	// Prepare a slice to store the string elements.
	strElems := make([]string, 0, len(listAttr))

	// Iterate over the slice elements.
	for _, v := range listAttr {
		// Use a type assertion to ensure each element is of type basetypes.StringValue.
		strVal, ok := v.(basetypes.StringValue)
		if !ok {
			return nil, fmt.Errorf("expected basetypes.StringValue, got %T", v)
		}

		// Append valid, non-null strings to the result slice.
		if !strVal.IsNull() && !strVal.IsUnknown() && strVal.ValueString() != "" {
			strElems = append(strElems, strVal.ValueString())
		}
	}

	// If no valid strings were found, return an error.
	if len(strElems) == 0 {
		return nil, errors.New("no valid non-empty strings found")
	}

	// Return the slice of valid strings.
	return strElems, nil
}
