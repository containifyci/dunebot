package github

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/containifyci/dunebot/pkg/template"
	"github.com/palantir/go-githubapp/githubapp"
)

const (
	logKeyPRNum          = githubapp.LogKeyPRNum
	logKeyRepositoryName = githubapp.LogKeyRepositoryName
	templateLoadingError = `> [!CAUTION]
> Failed to load the template for the error message. Please check the DuneBot logs for more details.`

	DataDogLogUrl = "https://app.datadoghq.eu/logs?"
)

//go:embed templates/*.tmpl
var templates embed.FS

type ErrorHandler struct {
	now time.Time
}

type DuneBotError struct {
	template string
	err      error
}

type TemplatedError interface {
	error
	Template() string
}

func (e *DuneBotError) AddError(err error) *DuneBotError {
	e.err = err
	return e
}

func (e DuneBotError) Error() string {
	return e.err.Error()
}

func (e DuneBotError) Template() string {
	return e.template
}

func NewAPIError(ctx context.Context, err error) TemplatedError {
	e := ErrorHandler{now: time.Now()}
	_, err2 := e.NewAPIError(ctx, err)
	return err2
}

func (e ErrorHandler) NewAPIError(ctx context.Context, err error) (string, TemplatedError) {
	buf := make([]byte, 8096)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	errDetails := struct {
		Error      string
		StackTrace string
		DataDogUrl string
	}{
		Error:      err.Error(),
		StackTrace: string(buf),
		DataDogUrl: dataDogLogUrl(ctx, e.now),
	}
	tmplErr := newErrorTemplate("templates/GeneralApiError.tmpl", errDetails)
	return string(buf), tmplErr.AddError(err)
}

func NewCheckRunError(run *CheckRun) TemplatedError {
	err := newErrorTemplate("templates/WorkflowRunError.tmpl", run)
	return err.AddError(errors.New("check run failed"))
}

func NewWorkflowRunError(run *WorkflowRun) TemplatedError {
	err := newErrorTemplate("templates/WorkflowRunError.tmpl", run)
	return err.AddError(errors.New("workflow run failed"))
}

func NewStatusCheckError(msg string) TemplatedError {
	err := newErrorTemplate("templates/StatusCheckError.tmpl", msg)
	return err.AddError(errors.New("status check failed"))
}

func NewNotMergableError(ctx context.Context, err error) TemplatedError {
	e := ErrorHandler{now: time.Now()}
	_, err2 := e.NewNotMergableError(ctx, err)
	return err2
}

func (e ErrorHandler) NewNotMergableError(ctx context.Context, err error) (string, TemplatedError) {
	buf := make([]byte, 8096)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	mergableDetails := struct {
		StackTrace string
		DataDogUrl string
		Error      string
	}{
		StackTrace: string(buf),
		DataDogUrl: dataDogLogUrl(ctx, e.now),
		Error:      err.Error(),
	}
	tmplErr := newErrorTemplate("templates/NotMergable.tmpl", mergableDetails)
	return string(buf), tmplErr.AddError(err)
}

func NewRetryExceedError(ctx context.Context, retry Retry) TemplatedError {
	e := ErrorHandler{now: time.Now()}
	_, err2 := e.NewRetryExceedError(ctx, retry)
	return err2
}

func (e ErrorHandler) NewRetryExceedError(ctx context.Context, retry Retry) (string, TemplatedError) {
	buf := make([]byte, 8096)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	retryDetails := struct {
		StackTrace string
		DataDogUrl string
		Retry
	}{
		StackTrace: string(buf),
		DataDogUrl: dataDogLogUrl(ctx, e.now),
		Retry:      retry,
	}
	err := newErrorTemplate("templates/RetryExceedError.tmpl", retryDetails)
	return string(buf), err.AddError(errors.New("retry exceeded"))
}

func newErrorTemplate(templatePath string, data interface{}) *DuneBotError {
	cnt, _, err := template.NewTemplate(templates, templatePath, data)

	if err != nil {
		return &DuneBotError{err: errors.New("failed to load template"), template: templateLoadingError}
	}

	return &DuneBotError{template: *cnt, err: nil}
}

func dataDogLogUrl(ctx context.Context, now time.Time) string {
	//TODO add proper logging
	repo, ok := ctx.Value(logKeyRepositoryName).(string)
	if !ok {
		repo = "UndefinedRepo"
	}
	prNum, ok := ctx.Value(logKeyPRNum).(int)
	if !ok {
		prNum = -1
	}

	query := make(dataDogQueryParam)
	query.Add("service", "dunebot")
	query.Add("env", "shared")
	query.Add("@github_repository_name", repo)
	query.Add("@github_pr_num", fmt.Sprintf("%d", prNum))

	// Prepare Query Parameters
	params := url.Values{}
	from := now.Add(-2 * time.Hour).Unix()
	to := now.Add(+2 * time.Hour).Unix()
	params.Add("from_ts", fmt.Sprintf("%d", from)) //unix timestamp
	params.Add("to_ts", fmt.Sprintf("%d", to))     //unix timestamp

	return DataDogLogUrl + params.Encode() + "&query=" + query.Encode()
}

type dataDogQueryParam map[string]string

func (d dataDogQueryParam) Add(key, value string) {
	d[key] = value
}

func (d dataDogQueryParam) Encode() string {
	var pairs []string

	// Create a slice to store keys.
	keys := make([]string, 0, len(d))
	for key := range d {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	// Iterate over the map and construct the key-value pairs with colon delimiter
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s:%s", key, d[key]))
	}

	q := url.QueryEscape(strings.Join(pairs, " "))
	q = strings.ReplaceAll(q, "+", "%20")
	return q
}
