package slackutil

import (
	"errors"
)

// CompareMode defines the types of comparison modes available for CompareStrings.
type CompareMode int

const (
	Any        CompareMode = iota // Return true if any element from list2 is in list1
	Match                         // Return true if all elements from list2 are in list1
	NotMatch                      // Return true if none of the elements from list2 are in list1
	ExactMatch                    // Return true if both lists are exactly the same in terms of elements and order
	Subset                        // Return true if all elements of list2 are in list1 (order does not matter)
	Superset                      // Return true if list1 is a superset of list2 (list1 contains all elements of list2)
	None                          // Return true if both lists are empty
)

// CompareStrings compares two lists of strings based on the provided mode and returns a boolean result.
//
// This function accepts two slices of strings (list1 and list2) and a comparison mode to determine
// how the lists should be evaluated. Depending on the mode, the function can check if any element from
// list2 is in list1, if all elements match, if none match, or if the lists are exact matches or subsets.
//
// Sample Input:
//
//	list1 := []string{"apple", "banana", "cherry"}
//	list2 := []string{"banana", "apple"}
//
// Sample Output:
//
//	result := CompareStrings(list1, list2, Subset)
//	// result will be true because list2 is a subset of list1.
//
// Returns:
//
//	A boolean result based on the specified comparison mode. If either list is nil, an error is returned.
func CompareStrings(list1, list2 []string, mode CompareMode) (bool, error) {
	if list1 == nil || list2 == nil {
		return false, errors.New("one or both input lists are nil")
	}

	set := make(map[string]bool)

	// Add all elements of list1 to a map for fast lookup.
	for _, item := range list1 {
		set[item] = true
	}

	switch mode {
	case Any:
		for _, item := range list2 {
			if set[item] {
				return true, nil
			}
		}
		return false, nil

	case Match:
		for _, item := range list2 {
			if !set[item] {
				return false, nil
			}
		}
		return true, nil

	case NotMatch:
		for _, item := range list2 {
			if set[item] {
				return false, nil
			}
		}
		return true, nil

	case ExactMatch:
		if len(list1) != len(list2) {
			return false, nil
		}
		for i, item := range list1 {
			if list2[i] != item {
				return false, nil
			}
		}
		return true, nil

	case Subset:
		for _, item := range list2 {
			if !set[item] {
				return false, nil
			}
		}
		return true, nil

	case Superset:
		if len(list1) < len(list2) {
			return false, nil
		}
		for _, item := range list2 {
			if !set[item] {
				return false, nil
			}
		}
		return true, nil

	case None:
		return len(list1) == 0 && len(list2) == 0, nil

	default:
		return false, errors.New("invalid comparison mode")
	}
}
