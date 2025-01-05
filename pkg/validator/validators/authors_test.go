package validators

import (
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestValidationAuthor(t *testing.T) {
	tests := []struct {
		name             string
		author           string
		authors          []string
		branch           *config.Branch
		expectedEnabled  bool
		expectedValid    bool
		expectedMessages string
	}{
		{
			name:            "Validation disabled (authors not enabled)",
			author:          "author1",
			branch:          nil,
			expectedEnabled: false,
			expectedValid:   false,
		},
		{
			name:            "Author is in the list (validation passes)",
			author:          "author1",
			authors:         []string{"author1", "author2"},
			branch:          nil,
			expectedEnabled: true,
			expectedValid:   false,
		},
		{
			name:             "Author is not in the list (no branch)",
			author:           "author3",
			authors:          []string{"author1", "author2"},
			branch:           nil,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Author is not in the list of approved authors for type",
		},
		{
			name:             "Author is not in the list (with branch)",
			author:           "author3",
			authors:          []string{"author1", "author2"},
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Author is not in the list of approved authors for type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := ForAuthors(tt.author, config.StringArray(tt.authors), "type", tt.branch)

			// Check IsEnabled
			assert.Equal(t, tt.expectedEnabled, validation.IsEnabled(), "Unexpected IsEnabled result")

			// Validate
			results := &ValidationResults{}
			valid := Validate(validation, results)
			assert.Equal(t, tt.expectedValid, valid, "Unexpected Validate result")
			if tt.expectedMessages != "" {
				assert.Equal(t, tt.expectedMessages, results.Results[0].Message, "Unexpected validation messages")
			}
		})
	}
}
