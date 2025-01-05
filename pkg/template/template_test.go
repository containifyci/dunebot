package template

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func init() {
	funcMap["panic"] = func(input any) string {
		panic("panic function called")
	}
}

// Test Data
var mockTemplates = &fstest.MapFS{
	"valid_template.txt":        &fstest.MapFile{Data: []byte("Hello, {{.Name}}!")},
	"escape_template.txt":       &fstest.MapFile{Data: []byte("Escaped: {{escape .Markdown}}")},
	"invalid_template.txt":      &fstest.MapFile{Data: []byte("Hello {{.Name")},         // invalid syntax
	"missing_func_template.txt": &fstest.MapFile{Data: []byte("Hello {{panic .Name}}")}, // invalid syntax
}

// Test functions

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single special character",
			input:    "hello *world*",
			expected: "hello \\*world\\*",
		},
		{
			name:     "multiple special characters",
			input:    "hello *world*!",
			expected: "hello \\*world\\*\\!",
		},
		{
			name:     "already escaped characters",
			input:    "hello \\*world\\*",
			expected: "hello \\\\\\*world\\\\\\*",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := EscapeMarkdown(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewTemplate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		templateName string
		data         any
		expected     string
		expectError  bool
	}{
		{
			name:         "valid template with data",
			templateName: "valid_template.txt",
			data:         map[string]string{"Name": "world"},
			expected:     "Hello, world!",
		},
		{
			name:         "template with escaping",
			templateName: "escape_template.txt",
			data:         map[string]string{"Markdown": "*bold*"},
			expected:     "Escaped: \\*bold\\*",
		},
		{
			name:         "non-existent template",
			templateName: "missing_template.txt",
			data:         nil,
			expected:     "",
			expectError:  true,
		},
		{
			name:         "invalid template syntax",
			templateName: "invalid_template.txt",
			data:         nil,
			expected:     "",
			expectError:  true,
		},
		{
			name:         "missing template data syntax",
			templateName: "missing_func_template.txt",
			data:         map[string]string{"name": "world"},
			expected:     "",
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, tmpl, err := NewTemplate(mockTemplates, tc.templateName, tc.data)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Nil(t, tmpl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, tmpl)
				assert.Equal(t, tc.expected, *result)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	tmpl, _, err := New("Hello, {{.Name}}!", "valid_template.txt", map[string]string{"Name": "world"})
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)
	assert.Equal(t, "Hello, world!", *tmpl)
}
