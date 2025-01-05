package validators

import (
	"strings"

	"github.com/containifyci/dunebot/pkg/compare"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
)

type ValidationRequiredStatuses struct {
	Statuses  func() (github.Statuses, error)
	CStatuses config.StringArray
	ValidationBranch
}

func ForRequiredStatuses(statuses func() (github.Statuses, error), cstatuses config.StringArray, branch *config.Branch) ValidationRequiredStatuses {
	return ValidationRequiredStatuses{
		Statuses:         statuses,
		CStatuses:        cstatuses,
		ValidationBranch: ValidationBranch{Branch: branch},
	}
}

func (v ValidationRequiredStatuses) IsEnabled() bool {
	return v.CStatuses.Enabled()
}

func (v ValidationRequiredStatuses) Validate(results *ValidationResults) bool {
	successStatuses, err := v.Statuses()
	if err != nil {
		// pr.Logger().Error().Err(err).Msgf("Error getting success statuses: %v", err)
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "statuses", "Error getting success statuses %s", err)
		} else {
			results.AddResult("statuses", "Error getting success statuses %s", err)
		}
		if results.FailFast {
			return true
		}
	}

	unsatisfiedStatuses := compare.Complement(successStatuses.ToString(), v.CStatuses)
	// pr.Logger().Debug().Msg("Checking statuses...")

	if len(unsatisfiedStatuses) > 0 {
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "statuses", "Required statuses not found: %s", strings.Join(unsatisfiedStatuses, ", "))
		} else {
			results.AddResult("statuses", "Required statuses not found: %s", strings.Join(unsatisfiedStatuses, ", "))
		}
		return true
	}
	return false
}

var _ Validation = ValidationRequiredStatuses{}
