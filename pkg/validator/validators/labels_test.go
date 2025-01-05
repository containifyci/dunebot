package validators

import (
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestValidationLabels(t *testing.T) {
	tests := []struct {
		name             string
		labels           []*github.Label
		clabels          config.StringArray
		branch           *config.Branch
		expectedEnabled  bool
		expectedValid    bool
		expectedMessages string
	}{
		{
			name: "Validation disabled (no configured labels)",
			labels: []*github.Label{
				{Name: strPtr("bug")},
			},
			expectedEnabled: false,
			expectedValid:   false,
		},
		{
			name: "Labels do not match",
			labels: []*github.Label{
				{Name: strPtr("enhancement")},
			},
			clabels: config.StringArray{
				"bug",
			},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Labels do not match for type",
		},
		{
			name: "Labels do not match branch",
			labels: []*github.Label{
				{Name: strPtr("enhancement")},
			},
			clabels: config.StringArray{
				"bug",
			},
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Labels do not match for type",
		},
		{
			name: "Labels match",
			labels: []*github.Label{
				{Name: strPtr("bug")},
				{Name: strPtr("enhancement")},
			},
			clabels: config.StringArray{
				"bug",
			},
			expectedEnabled: true,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &ValidationResults{}

			validation := ForLabels(tt.labels, tt.clabels, "type", tt.branch)

			// Check IsEnabled
			assert.Equal(t, tt.expectedEnabled, validation.IsEnabled(), "Unexpected IsEnabled result")

			// Validate
			valid := Validate(validation, results)

			// Check the validation result
			assert.Equal(t, tt.expectedValid, valid, "Unexpected Validate result")

			if tt.expectedMessages != "" {
				assert.Equal(t, tt.expectedMessages, results.Results[0].Message, "Unexpected validation messages")
			}
		})
	}
}
