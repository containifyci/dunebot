package validators

import (
	"embed"
	"fmt"

	"github.com/containifyci/dunebot/pkg/template"
)

const (
	ValidationReasonAuthor   = "author"
	ValidationReasonLabels   = "labels"
	ValidationReasonComments = "comments"
	ValidationReasonBranch   = "branch"
	ValidationReasonStatuses = "statuses"
)

//go:embed templates/*.tmpl
var templates embed.FS

var templateFnc = template.NewTemplate

type ValidationResults struct {
	Results []ValidationResult
	Success bool
	Type    string

	FailFast bool
}

type ValidationResult struct {
	Config  string
	Branch  string
	Reason  string
	Message string
}

func (vr *ValidationResults) String() string {
	cnt, _, err := templateFnc(templates, "templates/NotPassed.tmpl", vr)
	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}
	return *cnt
}

func (r *ValidationResults) SuccessResult() *ValidationResults {
	if len(r.Results) > 0 {
		r.Success = false
	} else {
		r.Success = true
	}
	return r
}

func (r *ValidationResults) AddBranchResult(key string, branch string, reason string, message string, args ...any) *ValidationResults {
	msg := message
	if len(args) > 0 {
		msg = fmt.Sprintf(message, args...)
	}
	r.Results = append(r.Results, ValidationResult{
		Config:  key,
		Branch:  branch,
		Reason:  reason,
		Message: msg,
	})
	r.Success = false
	return r
}

func (r *ValidationResults) AddResult(reason string, message string, args ...any) *ValidationResults {
	msg := message
	if len(args) > 0 {
		msg = fmt.Sprintf(message, args...)
	}
	r.Results = append(r.Results, ValidationResult{
		Reason:  reason,
		Message: msg,
	})

	return r
}

type Validation interface {
	IsEnabled() bool
	Validate(result *ValidationResults) bool
}

func Validate(v Validation, result *ValidationResults) bool {
	if v.IsEnabled() {
		return v.Validate(result)
	}
	return false
}

func isSubSet(subset []string, superset []string) bool {
	checkset := make(map[string]bool)
	for _, element := range superset {
		checkset[element] = true
	}
	for _, value := range subset {
		if !checkset[value] {
			return false //return false if set is not subset of superset
		}
	}
	return true
}
