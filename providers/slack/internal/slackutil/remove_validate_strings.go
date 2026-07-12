package slackutil

import (
	"errors"
	"sort"
)

// RemoveAndValidateStrings removes specific strings from a list, ensures uniqueness of the remaining strings,
// and returns a sorted slice of unique strings.
//
// This function accepts a target slice from which to remove values, as well as one or more lists of strings
// that should be removed from the target. It removes all occurrences of the specified strings, skips empty strings,
// ensures the uniqueness of the remaining values, and returns a sorted slice.
//
// Sample Input:
//
//	targetList := []string{"userA", "userB", "userC", "userD"}
//	removeList := []string{"userB", "userD"} // Strings to be removed
//
// Sample Output:
//
//	remainingList, err := RemoveAndValidateStrings(targetList, removeList)
//	// remainingList will contain []string{"userA", "userC"}.
//
// Returns:
//
//	A sorted slice of unique strings from the target list with the specified strings removed.
//	If the target or any remove list is nil, an error is returned.
func RemoveAndValidateStrings(targetList []string, removeLists ...[]string) ([]string, error) {
	if targetList == nil {
		return nil, errors.New("target list is nil")
	}

	// Prepare a set to track unique remaining strings.
	remainingStrings := make(map[string]struct{})

	// Populate the remainingStrings set with all unique strings from targetList.
	for _, str := range targetList {
		if str != "" { // Skip empty strings
			remainingStrings[str] = struct{}{}
		}
	}

	// Iterate over all removeLists to delete specified strings from remainingStrings.
	for _, removeList := range removeLists {
		if removeList == nil {
			return nil, errors.New("remove list is nil")
		}

		// Remove strings found in removeList from the remainingStrings set.
		for _, str := range removeList {
			delete(remainingStrings, str)
		}
	}

	// Convert the remaining strings map keys to a sorted slice.
	finalList := make([]string, 0, len(remainingStrings))
	for item := range remainingStrings {
		finalList = append(finalList, item)
	}

	// Sort the final list alphabetically.
	sort.Strings(finalList)

	// Return the remaining and validated list.
	return finalList, nil
}
