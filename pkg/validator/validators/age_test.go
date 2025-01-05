package validators

import (
	"testing"
	"time"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestValidationAge(t *testing.T) {
	oneDay := config.Duration{Duration: 24 * time.Hour}
	now := time.Now()

	tests := []struct {
		name             string
		age              *config.Duration
		branch           *config.Branch
		timestamp        func() *github.Timestamp
		expectedEnabled  bool
		expectedValid    bool
		expectedMessages string
	}{
		{
			name:             "Timestamp is nil",
			age:              &oneDay,
			timestamp:        func() *github.Timestamp { return nil },
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting pull request age for type",
		},
		{
			name:             "Timestamp is nil for branch",
			age:              &oneDay,
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			timestamp:        func() *github.Timestamp { return nil },
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting pull request age for type",
		},
		{
			name: "Pull request age is less than required",
			age:  &oneDay,
			timestamp: func() *github.Timestamp {
				return &github.Timestamp{Time: now.Add(-12 * time.Hour)}
			},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Pull request age is less than required (12h0m0s) for type",
		},
		{
			name: "Pull request age is less than required",
			age:  &oneDay,
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			timestamp: func() *github.Timestamp {
				return &github.Timestamp{Time: now.Add(-12 * time.Hour)}
			},
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Pull request age is less than required (12h0m0s) for type",
		},
		{
			name: "Pull request age is sufficient",
			age:  &oneDay,
			timestamp: func() *github.Timestamp {
				return &github.Timestamp{Time: now.Add(-48 * time.Hour)}
			},
			expectedEnabled: true,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := ForAge(tt.age, tt.timestamp, "type", tt.branch)
			assert.Equal(t, tt.expectedEnabled, validation.IsEnabled(), "Unexpected IsEnabled result")

			results := &ValidationResults{}
			valid := validation.Validate(results)

			assert.Equal(t, tt.expectedValid, valid, "Unexpected Validate result")
			if tt.expectedMessages != "" {
				assert.Equal(t, tt.expectedMessages, results.Results[0].Message, "Unexpected validation messages")
				if tt.branch != nil {
					assert.Equal(t, tt.branch.Prefix, results.Results[0].Branch, "Unexpected branch prefix")
				}
			}
		})
	}
}
