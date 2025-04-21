package validators

import (
	"time"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
)

type ValidationAge struct {
	Age       *config.Duration
	Timestamp func() *github.Timestamp
	Type      string
	ValidationBranch
}

func ForAge(age *config.Duration, timestamp func() *github.Timestamp, _type string, branch *config.Branch) ValidationAge {
	return ValidationAge{
		Age:              age,
		Timestamp:        timestamp,
		Type:             _type,
		ValidationBranch: ValidationBranch{Branch: branch},
	}
}

func (v ValidationAge) IsEnabled() bool {
	return v.Age != nil
}

func (v ValidationAge) Validate(results *ValidationResults) bool {
	createdAt := v.Timestamp()
	if createdAt == nil {
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "age", "Error getting pull request age for %s", v.Type)
		} else {
			results.AddResult("age", "Error getting pull request age for %s", v.Type)
		}
		return true
	}
	now := time.Now().Add(-1 * v.Age.Duration).UTC()
	if now.Unix() <= createdAt.Unix() {
		diff := createdAt.Sub(now)
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "age", "Pull request age is less than required (%s) for %s", diff.Round(1*time.Minute), v.Type)
		} else {
			results.AddResult("age", "Pull request age is less than required (%s) for %s", diff.Round(1*time.Minute), v.Type)
		}
		return true
	}
	return false
}

var _ Validation = ValidationAge{}
