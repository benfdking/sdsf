package slackutil

import (
	"errors"
	"strings"
)

// ConvertStringsToListValue converts a slice of strings into a ListValue.
//
// It checks if the provided slice is empty and returns a null list if so.
// For each non-empty and non-whitespace string in the slice, it creates a corresponding StringValue.
// If all strings are empty or contain only whitespace, an error is returned.
//
// Sample Input:
//
//	input := []string{"first", "second", "third"}
//
// Sample Output:
//
//	listVal, err := ConvertStringsToListValue(input)
//	// listVal will contain ListValue with Elements corresponding to the non-empty input strings.
//
// Returns:
//
//	A ListValue containing StringValue elements extracted from the input slice.
//	If the input slice is empty, it returns a null list. If all strings are empty or whitespace, an error is returned.

// ConvertStringsToListValue converts a slice of strings into a ListValue, returning an error for invalid input.
func ConvertStringsToListValue(input []string) (ListValue, error) {
	// If the input slice is empty, return a null list.
	if len(input) == 0 {
		return ListValue{Null: true}, nil // Return null list if input is empty.
	}

	// Prepare a slice to store the string elements.
	listElems := make([]StringValue, 0, len(input))

	// Iterate over the input slice.
	for _, str := range input {
		// If the string is empty or contains only whitespace, skip it.
		if strings.TrimSpace(str) == "" {
			continue
		}

		// Create a StringValue from the string.
		listElems = append(listElems, StringValue{Content: str})
	}

	// If no valid strings were found, return an error.
	if len(listElems) == 0 {
		return ListValue{}, errors.New("no valid non-empty strings found")
	}

	// Create and return a ListValue containing the non-empty string elements.
	return ListValue{Elements: listElems}, nil
}
