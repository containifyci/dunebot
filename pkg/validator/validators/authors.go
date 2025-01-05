package validators

import (
	"slices"

	"github.com/containifyci/dunebot/pkg/config"
)

type ValidationAuthor struct {
	Author  string
	Authors config.StringArray
	Type    string
	ValidationBranch
}

type ValidationBranch struct {
	Branch *config.Branch
}

func ForAuthors(author string, authors config.StringArray, _type string, branch *config.Branch) ValidationAuthor {
	return ValidationAuthor{
		Author: author,
		Authors: authors,
		Type: _type,
		ValidationBranch: ValidationBranch{Branch: branch},
	}
}
func (v ValidationAuthor) IsEnabled() bool {
	return v.Authors.Enabled()
}

func (v ValidationAuthor) Validate(results *ValidationResults) bool {
	if !slices.Contains(v.Authors, v.Author) {
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "author", "Author is not in the list of approved authors for %s", v.Type)
		} else {
			results.AddResult("author", "Author is not in the list of approved authors for %s", v.Type)
		}
		return true
	}
	return false
}

var _ Validation = ValidationAuthor{}
