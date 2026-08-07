package slackutil

// GetStringFromMap retrieves a string value from a map based on the specified key.
//
// Parameters:
//   - m: A map[string]interface{} from which to retrieve the string value.
//   - key: A string specifying the key for the desired value in the map.
//
// Returns:
//   - A string containing the value associated with the specified key if it exists
//     and is of type string; otherwise, an empty string is returned.
//
// Example usage:
//
//	m := map[string]interface{}{"example_key": "example_value"}
//	value := GetStringFromMap(m, "example_key")
//	fmt.Println(value)  // Output: "example_value"
func GetStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
