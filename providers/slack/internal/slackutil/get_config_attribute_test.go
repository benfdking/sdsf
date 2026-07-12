package slackutil

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stretchr/testify/assert"
)

// MockGetter is a mock implementation of the interface used by GetConfigAttribute.
type MockGetter struct {
	Attribute any
	Diags     diag.Diagnostics
}

// GetAttribute simulates the retrieval of an attribute. It assigns the attribute value
// to the provided pointer based on its type and returns any associated diagnostics.
func (m *MockGetter) GetAttribute(ctx context.Context, p path.Path, value any) diag.Diagnostics {
	// Properly assert the value to the correct type and assign
	switch v := value.(type) {
	case *string:
		if str, ok := m.Attribute.(string); ok {
			*v = str
		}
	case *int:
		if num, ok := m.Attribute.(int); ok {
			*v = num
		}
	// Add other type cases as needed (e.g., *types.List, *bool, etc.)
	default:
		// Handle unexpected types if necessary
	}
	return m.Diags
}

// TestGetConfigAttributeSuccess tests the successful retrieval of a string attribute.
func TestGetConfigAttributeSuccess(t *testing.T) {
	ctx := context.TODO()
	var diags diag.Diagnostics

	// Mock success case
	mockGetter := &MockGetter{
		Attribute: "test-user",        // Simulate a string attribute
		Diags:     diag.Diagnostics{}, // No error diagnostics
	}

	result, ok := GetConfigAttribute[string](ctx, mockGetter, "users", &diags)
	assert.True(t, ok, "Expected retrieval to succeed")
	assert.Equal(t, "test-user", result, "Expected retrieved attribute to be 'test-user'")
	assert.False(t, diags.HasError(), "Expected no diagnostics error")
}

// TestGetConfigAttributeError tests the behavior when an error occurs during retrieval.
func TestGetConfigAttributeError(t *testing.T) {
	ctx := context.TODO()
	var diags diag.Diagnostics

	// Mock error case
	mockGetter := &MockGetter{
		Attribute: nil,
		Diags:     diag.Diagnostics{diag.NewErrorDiagnostic("TestError", "Simulated error")}, // Simulated error diagnostics
	}

	result, ok := GetConfigAttribute[string](ctx, mockGetter, "users", &diags)
	assert.False(t, ok, "Expected retrieval to fail")
	assert.Empty(t, result, "Expected result to be empty on failure")
	assert.True(t, diags.HasError(), "Expected diagnostics to have an error")
}

// TestGetConfigAttributeDifferentType tests the retrieval of an attribute with a different type (int).
func TestGetConfigAttributeDifferentType(t *testing.T) {
	ctx := context.TODO()
	var diags diag.Diagnostics

	// Mock success case with a different type (e.g., int)
	mockGetter := &MockGetter{
		Attribute: 42,
		Diags:     diag.Diagnostics{}, // No error diagnostics
	}

	result, ok := GetConfigAttribute[int](ctx, mockGetter, "some_number", &diags)
	assert.True(t, ok, "Expected retrieval to succeed")
	assert.Equal(t, 42, result, "Expected retrieved attribute to be 42")
	assert.False(t, diags.HasError(), "Expected no diagnostics error")
}
