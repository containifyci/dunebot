package github

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWorkflowRunError(t *testing.T) {
	t.Parallel()

	run := &WorkflowRun{
		ID:         Int64(1),
		Name:       String("test"),
		Conclusion: String("failure"),
		HTMLURL:    String("http://example.com"),
	}

	err := NewWorkflowRunError(run)
	if err == nil {
		t.Errorf("NewWorkflowRunError(%v) = nil, want error", run)
	}

	assert.Equal(t, "workflow run failed", err.Error())
	assert.Equal(t, workflowRunErrorMsg, err.Template())
}

func TestNewCheckRunError(t *testing.T) {
	t.Parallel()

	run := &CheckRun{
		ID:         Int64(1),
		Name:       String("test"),
		Conclusion: String("failure"),
		HTMLURL:    String("http://example.com"),
	}

	err := NewCheckRunError(run)
	if err == nil {
		t.Errorf("TestNewCheckRunError(%v) = nil, want error", run)
	}

	assert.Equal(t, "check run failed", err.Error())
	assert.Equal(t, workflowRunErrorMsg, err.Template())
}

func TestNewStatusCheckError(t *testing.T) {
	t.Parallel()

	err := NewStatusCheckError("Status 'Build Go' has a different state 'failure' then is not success, neutral, skipped, completed, in_progress or queued")
	if err == nil {
		t.Errorf("TestNewStatusCheckError = nil, want error")
	}

	assert.Equal(t, "status check failed", err.Error())
	assert.Equal(t, statusCheckErrMsg, err.Template())
}

func TestNewAPIError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = context.WithValue(ctx, logKeyPRNum, 23)                       //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here
	ctx = context.WithValue(ctx, logKeyRepositoryName, "dunebot-infra") //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here

	e := ErrorHandler{now: time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC)}
	stackTrace, err := e.NewAPIError(ctx, fmt.Errorf("test error"))
	if err == nil {
		t.Errorf("TestNewAPIError() = nil, want error")
	}

	errMsg := strings.ReplaceAll(generalErrorMsg, "{{.Stacktrace}}", stackTrace)

	assert.Equal(t, "test error", err.Error())
	assert.Equal(t, errMsg, err.Template())
}

func TestNewNotMergableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = context.WithValue(ctx, logKeyPRNum, 23)                       //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here
	ctx = context.WithValue(ctx, logKeyRepositoryName, "dunebot-infra") //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here

	e := ErrorHandler{now: time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC)}
	stackTrace, err := e.NewNotMergableError(ctx, fmt.Errorf("test error"))
	if err == nil {
		t.Errorf("TestNewAPIError() = nil, want error")
	}

	errMsg := strings.ReplaceAll(notMergableErrMsg, "{{.Stacktrace}}", stackTrace)

	assert.Equal(t, "test error", err.Error())
	assert.Equal(t, errMsg, err.Template())
}

func TestNewRetryError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = context.WithValue(ctx, logKeyPRNum, 23)                       //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here
	ctx = context.WithValue(ctx, logKeyRepositoryName, "dunebot-infra") //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here

	e := ErrorHandler{now: time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC)}
	stackTrace, err := e.NewRetryExceedError(ctx, Retry{Name: "test retry", Attempts: 3, MaxAttempts: 2})
	if err == nil {
		t.Errorf("TestNewRetryError() = nil, want error")
	}

	errMsg := strings.ReplaceAll(retryErrMsg, "{{.Stacktrace}}", stackTrace)

	assert.Equal(t, "retry exceeded", err.Error())
	assert.Equal(t, errMsg, err.Template())
}

func TestDataDogQueryParam(t *testing.T) {
	t.Parallel()

	q := make(dataDogQueryParam)
	q.Add("service", "dunebot")
	q.Add("env", "shared")
	q.Add("@github_repository_name", "dunebot-infra")
	q.Add("@github_pr_num", "23")

	s := q.Encode()
	assert.Equal(t, "%40github_pr_num%3A23%20%40github_repository_name%3Adunebot-infra%20env%3Ashared%20service%3Adunebot", s)
}

func TestDataDogLogUrl(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = context.WithValue(ctx, logKeyPRNum, 23)                       //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here
	ctx = context.WithValue(ctx, logKeyRepositoryName, "dunebot-infra") //nolint:staticcheck //the context value is defined in the githubapp package cant be resolved here

	then := time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	url := dataDogLogUrl(ctx, then)
	assert.Equal(t, "https://app.datadoghq.eu/logs?from_ts=1258482898&to_ts=1258497298&query=%40github_pr_num%3A23%20%40github_repository_name%3Adunebot-infra%20env%3Ashared%20service%3Adunebot", url)
}

func TestNewErrorTemplate(t *testing.T) {
	t.Parallel()

	err := newErrorTemplate("not_existing_template", nil)

	assert.Error(t, err)
	assert.Equal(t, "> [!CAUTION]\n> Failed to load the template for the error message. Please check the DuneBot logs for more details.", err.template)
}

const (
	workflowRunErrorMsg = `> [!NOTE]
> This Pull Request can't be approved by **DuneBot** because the workflow run failed.

<details><summary>Workflow test run has conclusion failure</summary>

<p>

<a href="http://example.com">Check the workflow run</a>

</p>
</details>

---

❓ Got questions?
Check out DuneBot's [Docs](https://containifyci.atlassian.net/wiki/spaces/PLATFORM/pages/780337242/DuneBot)
For further assistance reach out to [#ask-platform](https://containifyci.slack.com/archives/C01M4MM051A).

---

_You can trigger DuneBot reprocessing by closing and reopen the Pull Request._

This PR comment was generated by **DuneBot**
`
	generalErrorMsg = `> [!CAUTION]
> test error

<details><summary>Error Stacktrace</summary>
<p>

<a href="https://app.datadoghq.eu/logs?from_ts=1258482898&to_ts=1258497298&query=%40github_pr_num%3A23%20%40github_repository_name%3Adunebot-infra%20env%3Ashared%20service%3Adunebot">DuneBot Logs in DataDog</a>

` + "```" + `
{{.Stacktrace}}
` + "```" + `
</p>
</details>

---

❓ Got questions?
Check out DuneBot's [Docs](https://containifyci.atlassian.net/wiki/spaces/PLATFORM/pages/780337242/DuneBot)
For further assistance reach out to [#ask-platform](https://containifyci.slack.com/archives/C01M4MM051A).

---

_You can trigger DuneBot reprocessing by closing and reopen the Pull Request._

This PR comment was generated by **DuneBot**
`
	notMergableErrMsg = `> [!NOTE]
> This Pull Request is not mergeable by **DuneBot**.

<details><summary>Details</summary>
<p>

<a href="https://app.datadoghq.eu/logs?from_ts=1258482898&to_ts=1258497298&query=%40github_pr_num%3A23%20%40github_repository_name%3Adunebot-infra%20env%3Ashared%20service%3Adunebot">DuneBot Logs in DataDog</a>

test error
` + "```" + `
{{.Stacktrace}}
` + "```" + `
</p>
</details>

---

❓ Got questions?
Check out DuneBot's [Docs](https://containifyci.atlassian.net/wiki/spaces/PLATFORM/pages/780337242/DuneBot)
For further assistance reach out to [#ask-platform](https://containifyci.slack.com/archives/C01M4MM051A).

---

_You can trigger DuneBot reprocessing by closing and reopen the Pull Request._

This PR comment was generated by **DuneBot**
`
	retryErrMsg = `> [!WARNING]
> The retry of test retry exceeded its limit before this Pull Request could be approved by **DuneBot**.

<details><summary>Retry Details</summary>
<p>

<a href="https://app.datadoghq.eu/logs?from_ts=1258482898&to_ts=1258497298&query=%40github_pr_num%3A23%20%40github_repository_name%3Adunebot-infra%20env%3Ashared%20service%3Adunebot">DuneBot Logs in DataDog</a>

` + "```" + `
{{.Stacktrace}}
` + "```" + `
</p>
</details>

---

❓ Got questions?
Check out DuneBot's [Docs](https://containifyci.atlassian.net/wiki/spaces/PLATFORM/pages/780337242/DuneBot)
For further assistance reach out to [#ask-platform](https://containifyci.slack.com/archives/C01M4MM051A).

---

_You can trigger DuneBot reprocessing by closing and reopen the Pull Request._

This PR comment was generated by **DuneBot**
`
	statusCheckErrMsg = `> [!NOTE]
> This Pull Request can't be approved by **DuneBot** because the following status check failed.

Status 'Build Go' has a different state 'failure' then is not success, neutral, skipped, completed, in_progress or queued

---

❓ Got questions?
Check out DuneBot's [Docs](https://containifyci.atlassian.net/wiki/spaces/PLATFORM/pages/780337242/DuneBot)
For further assistance reach out to [#ask-platform](https://containifyci.slack.com/archives/C01M4MM051A).

---

_You can trigger DuneBot reprocessing by closing and reopen the Pull Request._

This PR comment was generated by **DuneBot**
`
)
