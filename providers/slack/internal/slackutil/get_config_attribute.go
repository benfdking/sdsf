package slackutil

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GetConfigAttribute retrieves a configuration attribute of type T from a getter interface.
// It checks for errors in obtaining the attribute and whether the attribute is explicitly set.
// If the attribute is not defined or there are errors, it returns the zero value for type T and a boolean indicating failure.
//
// Parameters:
//   - ctx: The context for request-scoped values and cancellation signals.
//   - getter: An interface that defines a method for retrieving an attribute by path.
//   - pathKey: A string key representing the path to the desired attribute.
//   - diagnostics: A pointer to a diag.Diagnostics object for collecting error messages.
//
// Returns:
//   - A value of type T representing the retrieved attribute. If the attribute is not defined or an error occurs,
//     this will be the zero value for type T (e.g., types.StringNull() for a string).
//   - A boolean indicating whether the retrieval was successful (true if successful, false if not).
//
// Example usage:
//
//	var myAttribute types.String
//	myAttribute, found := GetConfigAttribute[types.String](ctx, getter, "my_attribute_key", &diagnostics)
//	if !found {
//	    // Handle the case where the attribute was not defined or an error occurred.
//	}
func GetConfigAttribute[T any](ctx context.Context, getter interface {
	GetAttribute(context.Context, path.Path, any) diag.Diagnostics
}, pathKey string, diagnostics *diag.Diagnostics) (T, bool) {
	var attribute T
	diags := getter.GetAttribute(ctx, path.Root(pathKey), &attribute)
	diagnostics.Append(diags...)

	// Check if the attribute is null or unset (types.Null for single values, types.ListNull for lists).
	// For example, types.StringNull() represents a null `types.String`.
	if diags.HasError() || isNull(attribute) {
		fmt.Printf("Warning: The '%s' attribute is not defined.\n", pathKey)

		var zeroValue T
		return zeroValue, false
	}
	return attribute, true
}

// isNull checks if the given attribute is null or undefined based on its type.
// This function is useful to determine if the retrieved attribute has a valid value.
//
// Parameters:
//   - attribute: An interface{} representing the attribute to check.
//
// Returns:
//   - A boolean indicating whether the attribute is null or undefined.
func isNull(attribute any) bool {
	switch v := attribute.(type) {
	case types.String:
		return v.IsNull()
	case types.List:
		return v.IsNull()
	case types.Map:
		return v.IsNull()
	// Add cases for other types as needed
	default:
		return false
	}
}
