package slackutil

import (
	"errors"
	"sort"
)

// MergeAndValidateStrings merges multiple slices of strings, ensures uniqueness of the strings,
// and returns a sorted slice of unique strings.
//
// It checks if any of the input slices are nil, returning an error if so. The function also skips
// empty strings when determining uniqueness. The final result is a sorted slice of unique strings.
//
// Sample Input:
//
//	list1 := []string{"userA", "userB"}
//	list2 := []string{"userB", "userC"} // Duplicate value
//	list3 := []string{"userD", "", "userE"} // Empty string will be ignored
//
// Sample Output:
//
//	mergedList, err := MergeAndValidateStrings(list1, list2, list3)
//	// mergedList will contain []string{"userA", "userB", "userC", "userD", "userE"}.
//
// Returns:
//
//	A sorted slice of unique strings extracted from the input slices.
//	If any input slice is nil, an error is returned.
func MergeAndValidateStrings(lists ...[]string) ([]string, error) {
	// Prepare a map to track unique string elements.
	uniqueStrings := make(map[string]struct{})

	for _, list := range lists {
		// Check if the list is nil.
		if list == nil {
			return nil, errors.New("input list is nil")
		}

		// Extract the elements from the list and add them to the map for uniqueness.
		for _, str := range list {
			if str != "" { // Skip empty strings
				uniqueStrings[str] = struct{}{}
			}
		}
	}

	// Convert the map keys (unique strings) into a sorted slice.
	finalList := make([]string, 0, len(uniqueStrings))
	for item := range uniqueStrings {
		finalList = append(finalList, item)
	}

	// Sort the list alphabetically.
	sort.Strings(finalList)

	// Return the merged and validated list.
	return finalList, nil
}
