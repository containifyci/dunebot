package validators

import (
	"strings"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
)

type ValidationComments struct {
	Comments  func() ([]*github.IssueComment, error)
	Ccomments []*config.Comment
	Type      string
	ValidationBranch
}

func ForComments(comments func() ([]*github.IssueComment, error), ccomments []*config.Comment, _type string, branch *config.Branch) ValidationComments {
	return ValidationComments{
		Comments:         comments,
		Ccomments:        ccomments,
		Type:             _type,
		ValidationBranch: ValidationBranch{Branch: branch},
	}
}

func (v ValidationComments) IsEnabled() bool {
	return len(v.Ccomments) > 0
}

func (v ValidationComments) Validate(results *ValidationResults) bool {
	prComments, err := v.Comments()
	if err != nil {
		// pr.Logger().Err(err).Msg("Error getting comments")
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, ValidationReasonComments, "Error getting comments %s", err)
		} else {
			results.AddResult(ValidationReasonComments, "Error getting comments %s", err)
		}
		if results.FailFast {
			return true
		}
	}

	for _, comment := range v.Ccomments {
		found := false
		for _, prComment := range prComments {
			if strings.HasPrefix(*prComment.Body, comment.Title) && strings.Contains(*prComment.Body, comment.Body) {
				found = true
			}
		}
		if !found {
			if v.Branch != nil {
				results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, ValidationReasonComments, "%s comments do not match", v.Type)
			} else {
				results.AddResult(ValidationReasonComments, "%s comments do not match", v.Type)
			}
			if results.FailFast {
				return true
			}
		}
	}
	return false
}

var _ Validation = ValidationComments{}
