package validators

import (
	"io/fs"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestValidationResults(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(*ValidationResults)
		expectedResults []ValidationResult
		expectedSuccess bool
		expectedOutput  string
	}{
		{
			name: "AddBranchResult adds results and updates success",
			setup: func(results *ValidationResults) {
				results.AddBranchResult("key1", "branch1", ValidationReasonLabels, "Labels are missing for %s", "branch1")
				results.AddBranchResult("key2", "branch2", ValidationReasonAuthor, "Author validation failed")
			},
			expectedResults: []ValidationResult{
				{
					Config:  "key1",
					Branch:  "branch1",
					Reason:  ValidationReasonLabels,
					Message: "Labels are missing for branch1",
				},
				{
					Config:  "key2",
					Branch:  "branch2",
					Reason:  ValidationReasonAuthor,
					Message: "Author validation failed",
				},
			},
			expectedSuccess: false,
			expectedOutput:  "Labels are missing for branch1", // Expects the string representation to contain this text
		},
		{
			name: "AddResult adds results without branch and updates success",
			setup: func(results *ValidationResults) {
				results.AddResult(ValidationReasonComments, "Comment mismatch for %s", "branch1")
				results.AddResult(ValidationReasonStatuses, "Missing required status")
			},
			expectedResults: []ValidationResult{
				{
					Reason:  ValidationReasonComments,
					Message: "Comment mismatch for branch1",
				},
				{
					Reason:  ValidationReasonStatuses,
					Message: "Missing required status",
				},
			},
			expectedSuccess: false,
			expectedOutput:  "Comment mismatch for branch1",
		},
		{
			name: "SuccessResult updates success based on results",
			setup: func(results *ValidationResults) {
				// Initially no results, then add one to test transition
				results.SuccessResult() // Should initially mark as success=true
				results.AddResult(ValidationReasonBranch, "Branch mismatch")
				results.SuccessResult() // Should now mark success=false
			},
			expectedResults: []ValidationResult{
				{
					Reason:  ValidationReasonBranch,
					Message: "Branch mismatch",
				},
			},
			expectedSuccess: false,
			expectedOutput:  "Branch mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &ValidationResults{}

			// Run setup
			tt.setup(results)

			// Assertions for results
			assert.Equal(t, tt.expectedResults, results.Results, "Unexpected results")
			assert.Equal(t, tt.expectedSuccess, results.Success, "Unexpected success status")

			// String output validation (if applicable)
			if tt.expectedOutput != "" {
				assert.Contains(t, results.String(), tt.expectedOutput, "Unexpected string output")
			}
		})
	}
}
func TestValidationResultsStringError(t *testing.T) {
	// Override the `templates` variable with invalid data for this test
	originalTemplates := templateFnc
	defer func() { templateFnc = originalTemplates }() // Restore after the test
	templateFnc = func(t fs.ReadFileFS, name string, data any) (*string, *template.Template, error) {
		return nil, nil, assert.AnError
	}

	results := &ValidationResults{}
	results.AddResult(ValidationReasonLabels, "Some label mismatch")

	output := results.String()

	assert.Equal(t, output, "Error: assert.AnError general error for testing")
}
