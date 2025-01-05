package validators

import (
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
)


type ValidationLabels struct {
	Labels  []*github.Label
	Clabels config.StringArray
	Type    string
	ValidationBranch
}

func ForLabels(labels []*github.Label, clabels config.StringArray, _type string, branch *config.Branch) ValidationLabels {
	return ValidationLabels{
		Labels:           labels,
		Clabels:          clabels,
		Type:             _type,
		ValidationBranch: ValidationBranch{Branch: branch},
	}
}

func (v ValidationLabels) IsEnabled() bool {
	return v.Clabels.Enabled()
}

func (v ValidationLabels) Validate(results *ValidationResults) bool {
	labels := make([]string, 0)
	for _, label := range v.Labels {
		labels = append(labels, label.GetName())
	}
	if !isSubSet(v.Clabels, labels) {
		if v.Branch != nil {
			results.AddBranchResult(v.Branch.Key, v.Branch.Prefix, "labels", "Labels do not match for %s", v.Type)
		} else {
			results.AddResult("labels", "Labels do not match for %s", v.Type)
		}
		return true
	}
	return false
}

var _ Validation = ValidationLabels{}
