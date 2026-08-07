package slackutil

// GetBoolFromMap retrieves a boolean value from a map based on the specified key.
//
// Parameters:
//   - m: A map[string]interface{} from which to retrieve the boolean value.
//   - key: A string specifying the key for the desired value in the map.
//
// Returns:
//   - A boolean containing the value associated with the specified key if it exists
//     and is of type bool; otherwise, false is returned.
//
// Example usage:
//
//	m := map[string]interface{}{"example_key": true}
//	value := GetBoolFromMap(m, "example_key")
//	fmt.Println(value)  // Output: true
func GetBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
