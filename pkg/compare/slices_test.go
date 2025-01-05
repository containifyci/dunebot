package compare

import (
	"testing"
)

func TestComplement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		// name string
		actual   []string
		required []string
		expected []string
	}{
		{
			// name: ""
			actual:   []string{"a", "b", "c"},
			required: []string{"a", "b", "c"},
			expected: []string{},
		},
		{
			actual:   []string{"a", "b", "c", "d"},
			required: []string{"a", "b", "c"},
			expected: []string{},
		},
		{
			actual:   []string{"a", "b", "c"},
			required: []string{"a", "b", "c", "d"},
			expected: []string{"d"},
		},
		{
			actual:   []string{},
			required: []string{"a", "b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			actual:   []string{"a", "b", "c", "d", "e"},
			required: []string{},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := Complement(tc.actual, tc.required)
			if len(actual) != len(tc.expected) {
				t.Errorf("expected %v, got %v len %d got %d", tc.expected, actual, len(tc.expected), len(actual))
			}
		})
	}
}
