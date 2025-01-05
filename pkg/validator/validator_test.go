package validator

import (
	"fmt"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/config/testdata"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/validator/validators"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestValidateApproveConditions(t *testing.T) {
	validator := NewValidator(&config.AppConfig{})
	test := []struct {
		name         string
		config       string
		pr           string
		fnc          func(github.PullRequestResolver) *validators.ValidationResults
		expectedType string
	}{
		{
			name:         "simple pull request",
			fnc:          validator.ValidateApproveConditions,
			expectedType: "Approve",
		},
		{
			name:         "simple pull request",
			fnc:          validator.ValidateApproveCommentsAndChecks,
			expectedType: "Approve",
		},
		{
			name:         "simple pull request",
			fnc:          validator.ValidateMergeConditions,
			expectedType: "Merge",
		},
		{
			name:         "simple pull request",
			fnc:          validator.ValidateMergeCommentsAndChecks,
			expectedType: "Merge",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.fnc(nil)
			assert.Equal(t, false, res.Success)
			assert.Equal(t, tt.expectedType, res.Type)
		})
	}
}

func TestValidateMergeConditions(t *testing.T) {
	res := NewValidator(&config.AppConfig{}).ValidateMergeConditions(nil)
	assert.Equal(t, false, res.Success)
	assert.Equal(t, "Merge", res.Type)
}

func TestValidateSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		config_file       string
		pull_request_file string
		expected          validators.ValidationResults
	}{
		{
			name:              "simple pull request",
			config_file:       "valid_config.yaml",
			pull_request_file: "pull_request_develop_simple.yaml",
			expected:          validators.ValidationResults{Success: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cfg, err := Unmarshal(tt.config_file, &config.AppConfig{})
			assert.NoError(t, err)

			pr, err := Unmarshal(tt.pull_request_file, &testdata.MockPullRequest{})
			assert.NoError(t, err)

			validator := NewValidator(cfg)

			approvalValidation := validator.ValidateApproveConditions(pr)
			mergeValidation := validator.ValidateMergeConditions(pr)

			assert.Equal(t, tt.expected.Success, approvalValidation.Success)
			fmt.Println(approvalValidation.Results)

			approvalValidation = validator.ValidateApproveCommentsAndChecks(pr)

			assert.Equal(t, tt.expected.Success, approvalValidation.Success)

			assert.Equal(t, tt.expected.Success, mergeValidation.Success)
			fmt.Println(mergeValidation.Results)

			mergeValidation = validator.ValidateMergeCommentsAndChecks(pr)

			assert.Equal(t, tt.expected.Success, mergeValidation.Success)
			fmt.Println(mergeValidation.Results)
		})
	}
}

func TestValidateWithError(t *testing.T) {
	t.Parallel()

	expectedResults := []validators.ValidationResult{
		{Reason: "comments", Message: "Error getting comments mock error: comments"},
		{Reason: "comments", Message: "Approve comments do not match"},
		{Config: "dependabot", Branch: "dependabot", Reason: "comments", Message: "Error getting comments mock error: comments"},
		{Config: "dependabot", Branch: "dependabot", Reason: "comments", Message: "Approve comments do not match"},
		{Config: "dependabot", Branch: "dependabot", Reason: "statuses", Message: "Error getting success statuses mock error: checks"},
		{Config: "dependabot", Branch: "dependabot", Reason: "statuses", Message: "Required statuses not found: check"},
	}

	cfg, err := Unmarshal("valid_config.yaml", &config.AppConfig{})
	assert.NoError(t, err)

	pr := &testdata.MockErrorPullRequest{}

	pr2, err := Unmarshal("pull_request_develop_simple.yaml", &testdata.MockPullRequest{})
	assert.NoError(t, err)
	pr.MockPullRequest = *pr2

	validator := NewValidator(cfg)
	valid := validator.ValidateApproveCommentsAndChecks(pr)

	assert.Equal(t, false, valid.Success)
	assert.Equal(t, expectedResults, valid.Results)
	fmt.Println(valid.String())
}

func TestValidateDoneFailFast(t *testing.T) {
	t.Parallel()

	resultsApprove := []validators.ValidationResult{
		{Reason: "author", Message: "Author is not in the list of approved authors for Approve"},
		// Additional results can be uncommented or added as needed
	}

	resultsMerge := []validators.ValidationResult{
		{Reason: "author", Message: "Author is not in the list of approved authors for Merge"},
		// Additional results can be uncommented or added as needed
	}

	tests := []struct {
		name              string
		config_file       string
		pull_request_file string
		expectedApprove   validators.ValidationResults
		expectedMerge     validators.ValidationResults
	}{
		{
			name:              "nil pull request",
			config_file:       "valid_config.yaml",
			pull_request_file: "",
			expectedApprove:   validators.ValidationResults{Success: false},
			expectedMerge:     validators.ValidationResults{Success: false},
		},
		{
			name:              "simple pull request",
			config_file:       "valid_config.yaml",
			pull_request_file: "pull_request_develop_no_match.yaml",
			expectedApprove:   validators.ValidationResults{Success: false, Results: resultsApprove},
			expectedMerge:     validators.ValidationResults{Success: false, Results: resultsMerge},
		},
		{
			name:              "simple pull request no matching branch",
			config_file:       "valid_config.yaml",
			pull_request_file: "pull_request_develop_no_branch.yaml",
			expectedApprove: validators.ValidationResults{Success: false, Results: []validators.ValidationResult{
				{Reason: "branch", Message: "Branch not found"},
			}},
			expectedMerge: validators.ValidationResults{Success: false, Results: []validators.ValidationResult{
				{Reason: "branch", Message: "Branch not found"},
			}},
		},
		{
			name:              "valid multi branch pull request",
			config_file:       "valid_config_multi_branch_approve_and_merge.yaml",
			pull_request_file: "pull_request_develop_multi_branch.yaml",
			expectedApprove: validators.ValidationResults{
				Success: true,
				Results: []validators.ValidationResult{},
			},
			expectedMerge: validators.ValidationResults{
				Success: true,
				Results: []validators.ValidationResult{},
			},
		},
		{
			name:              "to young pull request",
			config_file:       "valid_config.yaml",
			pull_request_file: "pull_request_develop_createdAt.yaml",
			expectedApprove: validators.ValidationResults{
				Success: false,
				Results: []validators.ValidationResult{
					{Config: "dependabot", Branch: "dependabot", Reason: "age", Message: "Pull request age is less than required (2h30m0s) for Approve"},
				},
			},
			expectedMerge: validators.ValidationResults{
				Success: false,
				Results: []validators.ValidationResult{
					{Config: "dependabot", Branch: "dependabot", Reason: "age", Message: "Pull request age is less than required (2h30m0s) for Merge"},
				},
			},
		},
		{
			name:              "invalid multi branch pull request",
			config_file:       "valid_config_multi_branch_approve_and_merge.yaml",
			pull_request_file: "pull_request_invalid_multi_branch.yaml",
			expectedApprove: validators.ValidationResults{
				Success: false,
				Results: []validators.ValidationResult{
					{Config: "minor", Branch: "dependabot", Reason: "author", Message: "Author is not in the list of approved authors for Approve"},
					{Config: "patch", Branch: "dependabot", Reason: "labels", Message: "Labels do not match for Approve"},
				},
			},
			expectedMerge: validators.ValidationResults{
				Success: false,
				Results: []validators.ValidationResult{
					{Config: "minor", Branch: "dependabot", Reason: "author", Message: "Author is not in the list of approved authors for Merge"},
					{Config: "patch", Branch: "dependabot", Reason: "labels", Message: "Labels do not match for Merge"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Unmarshal(tt.config_file, &config.AppConfig{})
			assert.NoError(t, err)

			validator := NewValidator(cfg)
			validator.FailFast = true
			if tt.pull_request_file != "" {
				pr, err := Unmarshal(tt.pull_request_file, &testdata.MockPullRequest{})
				assert.NoError(t, err)

				// Validate approval conditions
				approvalResults := validator.ValidateApproveConditions(pr)
				assert.Equal(t, tt.expectedApprove.Success, approvalResults.Success)
				assert.ElementsMatch(t, tt.expectedApprove.Results, approvalResults.Results)

				// Validate merge conditions
				mergeResults := validator.ValidateMergeConditions(pr)
				assert.Equal(t, tt.expectedMerge.Success, mergeResults.Success)
				assert.ElementsMatch(t, tt.expectedMerge.Results, mergeResults.Results)

			} else {
				assert.Equal(t, tt.expectedApprove.Success, validator.ValidateApproveConditions(nil).Success)
				assert.Equal(t, tt.expectedMerge.Success, validator.ValidateMergeConditions(nil).Success)
			}

		})
	}
}

func TestValidateCommentsAndCheckDoneFailFast(t *testing.T) {
	t.Parallel()

	resultsComments := []validators.ValidationResult{
		{Reason: "comments", Message: "Approve comments do not match"},
	}

	mergeResultsComments := []validators.ValidationResult{
		{Reason: "comments", Message: "Merge comments do not match"},
	}

	tests := []struct {
		name                      string
		config_file               string
		pull_request_file         string
		expectedApproveValidation validators.ValidationResults
		expectedMergeValidation   validators.ValidationResults
	}{
		{
			name:                      "nil pull request",
			config_file:               "valid_config.yaml",
			pull_request_file:         "",
			expectedApproveValidation: validators.ValidationResults{Success: false},
			expectedMergeValidation:   validators.ValidationResults{Success: false},
		},
		{
			name:                      "simple pull request",
			config_file:               "valid_config.yaml",
			pull_request_file:         "pull_request_develop_no_match.yaml",
			expectedApproveValidation: validators.ValidationResults{Success: false, Results: resultsComments},
			expectedMergeValidation:   validators.ValidationResults{Success: false, Results: mergeResultsComments},
		},
		{
			name:                      "valid multi branch pull request",
			config_file:               "valid_config_multi_branch_approve_and_merge.yaml",
			pull_request_file:         "pull_request_develop_multi_branch.yaml",
			expectedApproveValidation: validators.ValidationResults{Success: true, Results: []validators.ValidationResult{}},
			expectedMergeValidation:   validators.ValidationResults{Success: true, Results: []validators.ValidationResult{}},
		},
		{
			name:              "invalid multi branch pull request",
			config_file:       "valid_config_multi_branch_approve_and_merge.yaml",
			pull_request_file: "pull_request_invalid_multi_branch.yaml",
			expectedApproveValidation: validators.ValidationResults{Success: false, Results: []validators.ValidationResult{
				{Config: "minor", Branch: "dependabot", Reason: "comments", Message: "Approve comments do not match"},
				{Config: "patch", Branch: "dependabot", Reason: "statuses", Message: "Required statuses not found: ci/circleci: ete-tests"},
			}},
			expectedMergeValidation: validators.ValidationResults{Success: false, Results: []validators.ValidationResult{
				{Config: "minor", Branch: "dependabot", Reason: "comments", Message: "Merge comments do not match"},
				{Config: "patch", Branch: "dependabot", Reason: "statuses", Message: "Required statuses not found: ci/circleci: ete-tests"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Unmarshal(tt.config_file, &config.AppConfig{})
			assert.NoError(t, err)

			validator := NewValidator(cfg)
			validator.FailFast = true
			if tt.pull_request_file != "" {
				pr, err := Unmarshal(tt.pull_request_file, &testdata.MockPullRequest{})
				assert.NoError(t, err)

				// Test comments and checks validation for approve
				approveCommentsResults := validator.ValidateApproveCommentsAndChecks(pr)

				assert.Equal(t, tt.expectedApproveValidation.Success, approveCommentsResults.Success)
				assert.ElementsMatch(t, tt.expectedApproveValidation.Results, approveCommentsResults.Results)

				// Test comments and checks validation for merge
				mergeCommentsResults := validator.ValidateMergeCommentsAndChecks(pr)

				assert.Equal(t, tt.expectedMergeValidation.Success, mergeCommentsResults.Success)
				assert.ElementsMatch(t, tt.expectedMergeValidation.Results, mergeCommentsResults.Results)

			} else {
				// If no pull request file is provided, use nil
				assert.Equal(t, tt.expectedApproveValidation.Success, validator.ValidateApproveCommentsAndChecks(nil).Success)
				assert.Equal(t, tt.expectedMergeValidation.Success, validator.ValidateMergeCommentsAndChecks(nil).Success)
			}
		})
	}
}

func TestMatchedBranchConfig(t *testing.T) {
	t.Parallel()

	cfg, err := Unmarshal("valid_config_no_branch_approve.yaml", &config.AppConfig{})
	assert.NoError(t, err)
	validator := NewValidator(cfg)

	branches := validator.MatchedBranchConfig(cfg.Approve.GetInclude(), "not_exist")
	assert.Equal(t, 0, len(branches))
}

func TestMatchedBranchConfigMultiple(t *testing.T) {
	t.Parallel()

	cfg, err := Unmarshal("valid_config_multi_branch.yaml", &config.AppConfig{})
	assert.NoError(t, err)
	validator := NewValidator(cfg)

	branches := validator.MatchedBranchConfig(cfg.Approve.GetInclude(), "dependabot")
	assert.Equal(t, 2, len(branches))
	assert.Equal(t, "dependabot", branches[0].Prefix)
	assert.Equal(t, "dependabot", branches[1].Prefix)
}

func TestMatchedBranchConfigSignleMultiple(t *testing.T) {
	t.Parallel()

	test := []struct {
		name     string
		branch   string
		expected []config.Branch
	}{
		{
			name:     "single branch",
			branch:   "dependabot/major",
			expected: []config.Branch{{Prefix: "dependabot/major"}},
		},
		{
			name:     "single branch",
			branch:   "dependabot/major/v1.14.6",
			expected: []config.Branch{{Prefix: "dependabot/major/v1.14.6"}},
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Unmarshal("valid_config_single_multi_branch.yaml", &config.AppConfig{})
			assert.NoError(t, err)
			validator := NewValidator(cfg)

			branches := validator.MatchedBranchConfig(cfg.Approve.GetInclude(), tt.branch)
			assert.Equal(t, len(tt.expected), len(branches))
			assert.Equal(t, tt.expected[0].Prefix, branches[0].Prefix)
		})
	}
}

func Unmarshal[T any](file string, typ *T) (*T, error) {
	var t T
	cnt, err := testdata.Data.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(cnt, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
