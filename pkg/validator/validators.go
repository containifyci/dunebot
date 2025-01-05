package validator

import (
	"sort"
	"strings"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/validator/validators"
	"golang.org/x/exp/maps"
)

type Validator struct {
	Config   *config.AppConfig
	FailFast bool
}

func NewValidator(cfg *config.AppConfig) *Validator {
	return &Validator{
		Config:   cfg,
		FailFast: false,
	}
}

func (r *Validator) ShouldFailFast() bool {
	return r.FailFast
}

func (v *Validator) MatchedBranchConfig(include *config.Include, branchName string) []*config.Branch {
	if len(include.Branches) <= 0 {
		return nil
	}
	var foundBranches []*config.Branch
	var onlybranch *config.Branch

	//ensures that the branches are always sorted
	keys := maps.Keys(include.Branches)
	sort.Strings(keys)
	strict := false
	maxKey := ""

	for _, key := range keys {
		branch := include.Branches[key]
		branch.Key = key
		if !strings.HasPrefix(branchName, branch.Prefix) {
			continue
		}
		strict = strict || branch.Strict
		if strict {
			if len(branch.Prefix) > len(maxKey) {
				onlybranch = branch
				maxKey = branch.Prefix
			}
		} else {
			foundBranches = append(foundBranches, branch)
		}
	}

	if strict {
		return []*config.Branch{onlybranch}
	}
	return foundBranches
}

func (v *Validator) ValidateApproveCommentsAndChecks(pr github.PullRequestResolver) *validators.ValidationResults {
	if v.Config.NoApprove() {
		return &validators.ValidationResults{Success: false, Type: "Approve"}
	}
	return v.ValidateCommentsAndChecks(pr, v.Config.Approve.GetInclude(), "Approve")
}

func (v *Validator) ValidateMergeCommentsAndChecks(pr github.PullRequestResolver) *validators.ValidationResults {
	if v.Config.NoMerge() {
		return &validators.ValidationResults{Success: false, Type: "Merge"}
	}
	return v.ValidateCommentsAndChecks(pr, v.Config.Merge.GetInclude(), "Merge")
}

func (v *Validator) ValidateCommentsAndChecks(pr github.PullRequestResolver, include *config.Include, _type string) *validators.ValidationResults {
	results := &validators.ValidationResults{
		Results:  make([]validators.ValidationResult, 0),
		Success:  false,
		Type:     _type,
		FailFast: v.ShouldFailFast(),
	}

	if pr == nil {
		results.AddResult("pr", "Pull request object is nil")
		return results
	}

	validateComments := validators.ForComments(pr.Comments, include.Comments, _type, nil)

	validations := []validators.Validation{validateComments}

	for _, val := range validations {
		if validators.Validate(val, results) {
			if v.ShouldFailFast() {
				return results
			}
		}
	}

	foundBranches := v.MatchedBranchConfig(include, pr.Head().GetRef())
	branchPassed := false
	branchResults := []*validators.ValidationResults{}

	if len(foundBranches) > 0 {
		for _, branch := range foundBranches {
			branchResult := &validators.ValidationResults{
				Results:  make([]validators.ValidationResult, 0),
				Success:  true,
				Type:     _type,
				FailFast: v.ShouldFailFast(),
			}

			validateComments := validators.ForComments(pr.Comments, branch.Comments, _type, branch)
			validateRequiredStatues := validators.ForRequiredStatuses(pr.Checks, branch.RequiredStatuses, branch)

			validations := []validators.Validation{validateComments, validateRequiredStatues}

			for _, val := range validations {
				if validators.Validate(val, branchResult) {
					if v.ShouldFailFast() {
						goto end
					}
				}
			}

			if branchResult.Success {
				// if we reached here then the branch is valid
				branchPassed = true
				break
			}
		end:
			branchResults = append(branchResults, branchResult)
			continue
		}
		if !branchPassed {
			for _, branchResult := range branchResults {
				results.Results = append(results.Results, branchResult.Results...)
			}
		}
	}
	return results.SuccessResult()
}

func (v *Validator) ValidateApproveConditions(pr github.PullRequestResolver) *validators.ValidationResults {
	if v.Config.NoApprove() {
		return &validators.ValidationResults{Success: false, Type: "Approve"}
	}
	return v.ValidateConditions(pr, v.Config.Approve.GetInclude(), "Approve")
}

func (v *Validator) ValidateMergeConditions(pr github.PullRequestResolver) *validators.ValidationResults {
	if v.Config.NoMerge() {
		return &validators.ValidationResults{Success: false, Type: "Merge"}
	}
	return v.ValidateConditions(pr, v.Config.Merge.GetInclude(), "Merge")
}

func (v *Validator) ValidateConditions(pr github.PullRequestResolver, include *config.Include, _type string) *validators.ValidationResults {
	results := &validators.ValidationResults{
		Results: make([]validators.ValidationResult, 0),
		Success: false,
		Type:    _type,
	}

	if pr == nil {
		return results
	}

	validateAuthor := validators.ForAuthors(pr.Owner().GetLogin(), include.Authors, _type, nil)
	validationLabels := validators.ForLabels(pr.Labels(), include.Labels, _type, nil)

	validations := []validators.Validation{validateAuthor, validationLabels}

	for _, val := range validations {
		if validators.Validate(val, results) {
			if v.ShouldFailFast() {
				return results
			}
		}
	}

	foundBranches := v.MatchedBranchConfig(include, pr.Head().GetRef())

	if len(foundBranches) <= 0 {
		results.AddResult("branch", "Branch not found")
		return results
	}

	branchPassed := false
	branchResults := []*validators.ValidationResults{}
	for _, branch := range foundBranches {
		branchResult := &validators.ValidationResults{
			Results: make([]validators.ValidationResult, 0),
			Success: true,
			Type:    _type,
		}

		validateAuthor := validators.ForAuthors(pr.Owner().GetLogin(), branch.Authors, _type, branch)
		validationLabels := validators.ForLabels(pr.Labels(), branch.Labels, _type, branch)
		validationAge := validators.ForAge(branch.Age, pr.CreatedAt, _type, branch)

		validations := []validators.Validation{validateAuthor, validationLabels, validationAge}

		for _, val := range validations {
			if validators.Validate(val, branchResult) {
				if v.ShouldFailFast() {
					goto end
				}
			}
		}

		if branchResult.Success {
			branchPassed = true
			break
		}
	end:
		branchResults = append(branchResults, branchResult)
		continue
	}
	if !branchPassed {
		for _, branchResult := range branchResults {
			results.Results = append(results.Results, branchResult.Results...)
		}
	}

	return results.SuccessResult()
}
