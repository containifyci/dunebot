package validators

import (
	"errors"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestValidationRequiredStatuses(t *testing.T) {
	tests := []struct {
		name             string
		statusesFunc     func() (github.Statuses, error)
		cstatuses        config.StringArray
		branch           *config.Branch
		failFast         bool
		expectedEnabled  bool
		expectedValid    bool
		expectedMessages string
	}{
		{
			name: "Validation disabled",
			statusesFunc: func() (github.Statuses, error) {
				return github.Statuses{}, nil
			},
			branch:          nil,
			expectedEnabled: false,
			expectedValid:   false,
		},
		{
			name: "Statuses match",
			statusesFunc: func() (github.Statuses, error) {
				return github.Statuses{{State: "success", Name: "ci/test"}}, nil
			},
			cstatuses:       config.StringArray{"ci/test"},
			expectedEnabled: true,
			expectedValid:   false,
		},
		{
			name: "Statuses do not match",
			statusesFunc: func() (github.Statuses, error) {
				return github.Statuses{{State: "success", Name: "ci/build"}}, nil
			},
			cstatuses:        config.StringArray{"ci/test", "ci/lint"},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Required statuses not found: ci/test, ci/lint",
		},
		{
			name: "Error fetching statuses",
			statusesFunc: func() (github.Statuses, error) {
				return nil, errors.New("API error")
			},
			cstatuses:        config.StringArray{"ci/test"},
			failFast:         true,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting success statuses API error",
		},
		{
			name: "Error fetching statuses branch",
			statusesFunc: func() (github.Statuses, error) {
				return nil, errors.New("API error")
			},
			cstatuses:        config.StringArray{"ci/test"},
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting success statuses API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &ValidationResults{}
			results.FailFast = tt.failFast

			validation := ForRequiredStatuses(tt.statusesFunc, tt.cstatuses, tt.branch)

			// Check IsEnabled
			assert.Equal(t, tt.expectedEnabled, validation.IsEnabled(), "Unexpected IsEnabled result")

			// Validate
			valid := Validate(validation, results)

			// Check the validation result
			assert.Equal(t, tt.expectedValid, valid, "Unexpected Validate result")

			// Check the validation messages
			if tt.expectedMessages != "" {
				assert.Equal(t, tt.expectedMessages, results.Results[0].Message, "Unexpected validation messages")
			}
		})
	}
}
