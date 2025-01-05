package validators

import (
	"errors"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

func TestValidationComments(t *testing.T) {
	tests := []struct {
		name             string
		commentsFunc     func() ([]*github.IssueComment, error)
		ccomments        []*config.Comment
		branch           *config.Branch
		failFast         bool
		expectedEnabled  bool
		expectedValid    bool
		expectedMessages string
	}{
		{
			name: "Validation disabled (no config comments)",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return []*github.IssueComment{}, nil
			},
			failFast:        false,
			expectedEnabled: false,
			expectedValid:   false,
		},
		{
			name: "Error fetching comments",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return nil, errors.New("fetch error")
			},
			ccomments: []*config.Comment{
				{Title: "Comment Title", Body: "Comment Body"},
			},
			failFast:         true,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting comments fetch error",
		},
		{
			name: "Error fetching comments branch",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return nil, errors.New("fetch error")
			},
			ccomments: []*config.Comment{
				{Title: "Comment Title", Body: "Comment Body"},
			},
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			failFast:         true,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "Error getting comments fetch error",
		},
		{
			name: "Comments do not match",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return []*github.IssueComment{
					{Body: strPtr("Unrelated comment")},
				}, nil
			},
			ccomments: []*config.Comment{
				{Title: "Expected Title", Body: "Expected Body"},
			},
			failFast:         true,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "type comments do not match",
		},
		{
			name: "Comments do not match branch",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return []*github.IssueComment{
					{Body: strPtr("Unrelated comment")},
				}, nil
			},
			ccomments: []*config.Comment{
				{Title: "Expected Title", Body: "Expected Body"},
			},
			branch:           &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			failFast:         true,
			expectedEnabled:  true,
			expectedValid:    true,
			expectedMessages: "type comments do not match",
		},
		{
			name: "Comments match",
			commentsFunc: func() ([]*github.IssueComment, error) {
				return []*github.IssueComment{
					{Body: strPtr("Expected Title - Expected Body")},
				}, nil
			},
			ccomments: []*config.Comment{
				{Title: "Expected Title", Body: "Expected Body"},
			},
			branch:          &config.Branch{Key: "branch-key", Prefix: "branch-prefix"},
			failFast:        false,
			expectedEnabled: true,
			expectedValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := ForComments(tt.commentsFunc, tt.ccomments, "type", tt.branch)

			// Check IsEnabled
			assert.Equal(t, tt.expectedEnabled, validation.IsEnabled(), "Unexpected IsEnabled result")

			// Validate
			results := &ValidationResults{FailFast: tt.failFast}
			valid := Validate(validation, results)

			assert.Equal(t, tt.expectedValid, valid, "Unexpected Validate result")
			if tt.expectedMessages != "" {
				assert.Equal(t, tt.expectedMessages, results.Results[0].Message, "Unexpected validation messages")
				if tt.branch != nil {
					assert.Equal(t, tt.branch.Prefix, results.Results[0].Branch, "Unexpected branch key")
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
