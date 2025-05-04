package github

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/containifyci/dunebot/pkg/github/testdata"
	"github.com/containifyci/dunebot/pkg/logger"
)

type GivenGithubRepositoryCommitModes int64

const (
	All GivenGithubRepositoryCommitModes = iota
	None
	Merge
	RebaseMerge
	OnlyRebaseMerge
	SquashMerge
	RebaseAndSquashMerge
	MergeAndSquashMerge
	MergeAndRebaseMerge
)

func GivenGithubRepository(mode GivenGithubRepositoryCommitModes) Repository {
	switch mode {
	case Merge:
		return Repository{AllowMergeCommit: Bool(true)}
	case RebaseMerge:
		return Repository{AllowRebaseMerge: Bool(true)}
	case OnlyRebaseMerge:
		return Repository{AllowMergeCommit: Bool(false), AllowRebaseMerge: Bool(true)}
	case SquashMerge:
		return Repository{AllowSquashMerge: Bool(true)}
	case RebaseAndSquashMerge:
		return Repository{AllowRebaseMerge: Bool(true), AllowSquashMerge: Bool(true)}
	case MergeAndSquashMerge:
		return Repository{AllowMergeCommit: Bool(true), AllowSquashMerge: Bool(true)}
	case MergeAndRebaseMerge:
		return Repository{AllowMergeCommit: Bool(true), AllowRebaseMerge: Bool(true)}
	case None:
		return Repository{AllowMergeCommit: Bool(false), AllowRebaseMerge: Bool(false), AllowSquashMerge: Bool(false)}
	case All:
		fallthrough
	default:
		return Repository{AllowMergeCommit: Bool(true), AllowRebaseMerge: Bool(true), AllowSquashMerge: Bool(true)}
	}
}

func GivenTestMergePullRequest(name, method string, mode GivenGithubRepositoryCommitModes, expectedMethod string) struct {
	name           string
	method         string
	repo           Repository
	expectedMethod string
} {
	return struct {
		name           string
		method         string
		repo           Repository
		expectedMethod string
	}{
		name:           name,
		method:         method,
		repo:           GivenGithubRepository(mode),
		expectedMethod: expectedMethod,
	}
}
func TestGetMergeMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		repo           Repository
		expectedMethod string
	}{
		GivenTestMergePullRequest("merge commit", "merge", Merge, "merge"),
		GivenTestMergePullRequest("merge commit", "rebase", Merge, "merge"),
		GivenTestMergePullRequest("rebase commit", "merge", RebaseMerge, "rebase"),
		GivenTestMergePullRequest("rebase commit", "merge", OnlyRebaseMerge, "rebase"),
		GivenTestMergePullRequest("squash commit", "merge", SquashMerge, "squash"),
		GivenTestMergePullRequest("no commit method allowed", "squash", None, ""),
		GivenTestMergePullRequest("merge commit all allowed", "merge", All, "merge"),
		GivenTestMergePullRequest("rebase commit all allowed", "rebase", All, "rebase"),
		GivenTestMergePullRequest("squash commit all allowed", "squash", All, "squash"),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			method := GetMergeMethod(tt.method, &tt.repo)

			assert.Equal(t, tt.expectedMethod, method)
		})
	}
}

func TestWaitForPRChecksToPass(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithContext(context.Background()),
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	tests := []struct {
		name     string
		states   []string
		passed   bool
		attempts int
	}{
		{name: "skipped", states: []string{"skipped"}, passed: false, attempts: 2},
		{name: "failure", states: []string{"failure"}, passed: false, attempts: 2},
		{name: "success", states: []string{"success"}, passed: true, attempts: 0},
		{name: "mixed success", states: []string{"success", "skipped", "failure"}, passed: true, attempts: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry := Retry{PauseTime: 10 * time.Millisecond, MaxWaitTime: 2 * time.Second, MaxAttempts: 2}
			passed, err := client.WaitForPRChecksToPass(&retry, "main", "sha", "push", tt.states...)

			if tt.passed {
				require.NoError(t, err, "Should not have error when checks are expected to pass")
			} else {
				assert.Error(t, err, "Should have error when checks are expected to fail")
			}

			assert.Equal(t, tt.passed, passed, "Check pass status should match expected")
			assert.LessOrEqual(t, retry.Attempts, tt.attempts, "Number of attempts should not exceed expected")
		})
	}
}

func TestIsOutOfDate(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithContext(context.Background()),
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)
	tests := []struct {
		name      string
		pr        *PullRequest
		outOfDate bool
	}{
		{name: "up to date", pr: &PullRequest{Base: &PullRequestBranch{Repo: &Repository{DefaultBranch: String("main")}}, Head: &PullRequestBranch{SHA: String("sha1")}}, outOfDate: false},
		{name: "behind", pr: &PullRequest{Base: &PullRequestBranch{Repo: &Repository{DefaultBranch: String("main")}}, Head: &PullRequestBranch{SHA: String("sha2")}}, outOfDate: true},
		{name: "diverged", pr: &PullRequest{Base: &PullRequestBranch{Repo: &Repository{DefaultBranch: String("main")}}, Head: &PullRequestBranch{SHA: String("sha3")}}, outOfDate: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outOfDate := client.IsOutOfDate(tt.pr)
			assert.Equal(t, tt.outOfDate, outOfDate)
		})
	}
}

func TestAddRebaseComment(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithContext(context.Background()),
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	tests := []struct {
		name string
		pr   *PullRequest
		err  error
	}{
		{name: "add comment", pr: &PullRequest{Number: Int(1)}},
		{name: "add comment failed", pr: &PullRequest{Number: Int(2)}, err: fmt.Errorf("PUT https://api.github.com/repos/test-owner/test-repo/pulls/2/update-branch: 422  []")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.AddRebaseComment(tt.pr)
			if tt.err != nil {
				assert.ErrorContains(t, err, tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithContext(context.Background()),
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	repo := &Repository{
		Owner: &User{Login: String("test-owner")},
		Name:  String("test-repo"),
	}

	tests := []struct {
		name  string
		prNum int
		err   error
	}{
		{name: "add comment", prNum: 1},
		{name: "add comment failed", prNum: 2, err: fmt.Errorf("POST https://api.github.com/repos/test-owner/test-repo/issues/2/comments: 410  []")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.AddComment(repo, tt.prNum, "test comment")
			if tt.err != nil {
				assert.ErrorContains(t, err, tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWaitForRebase(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	tests := []struct {
		number   int
		oldsha   string
		newsha   string
		attempts int
	}{
		{number: 1, oldsha: "1234567890", newsha: "5432109876", attempts: 1},
		{number: 2, oldsha: "1234567890", newsha: "1234567890", attempts: 10},
	}
	for _, tt := range tests {
		t.Run(tt.oldsha, func(t *testing.T) {
			pr := &PullRequest{
				Head: &PullRequestBranch{
					SHA: String(tt.oldsha),
				},
				Base: &PullRequestBranch{
					SHA: String(tt.oldsha),
				},
				Number: Int(tt.number),
			}
			retry := Retry{PauseTime: 1 * time.Millisecond, MaxAttempts: tt.attempts}
			sha, updatedPr, err := client.WaitForRebase(&retry, pr)
			if tt.attempts < 10 {
				require.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.oldsha, sha)
			assert.Equal(t, tt.newsha, updatedPr.Base.GetSHA())
			assert.Equal(t, tt.attempts, retry.Attempts)
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	tests := []struct {
		name   string
		prNum  int
		method string
		repo   Repository
		err    error
	}{
		{name: "merge", prNum: 1, method: "merge", repo: GivenGithubRepository(OnlyRebaseMerge)},
		{name: "rebase", prNum: 2, method: "rebase", err: fmt.Errorf("PUT https://api.github.com/repos/test-owner/test-repo/pulls/2/merge: 410  []")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.MergePullRequest(tt.prNum, tt.method, &tt.repo)
			if tt.err != nil {
				assert.ErrorContains(t, err, tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetStatuses(t *testing.T) {
	t.Parallel()

	client := NewClient(
		WithGithubClient(makeTestClient()),
		WithConfig(NewRepositoryConfig("test-owner", "test-repo")),
	)

	tests := []struct {
		name  string
		state string
		sha   string
	}{
		{name: "lint", state: "failure", sha: "sha1"},
		{name: "build", state: "completed", sha: "sha2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &PullRequest{
				Head: &PullRequestBranch{
					SHA: String(tt.sha),
				},
			}
			res, err := client.GetStatuses(pr)
			require.NoError(t, err)
			assert.Equal(t, tt.state, res[0].State)
		})
	}
}

func TestPrettyJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{name: "empty", input: "", expected: "{\"level\":\"debug\",\"json\":\"\\\"\\\"\",\"message\":\"Pretty JSON\"}\n"},
		{name: "valid json", input: struct {
			Key string `json:"key"`
		}{Key: "value"}, expected: "{\"level\":\"debug\",\"json\":\"{\\n    \\\"key\\\": \\\"value\\\"\\n}\",\"message\":\"Pretty JSON\"}\n"},
		{name: "nil", input: nil, expected: "{\"level\":\"debug\",\"json\":\"null\",\"message\":\"Pretty JSON\"}\n"},
		{name: "invalid json", input: "invalid", expected: "{\"level\":\"debug\",\"json\":\"\\\"invalid\\\"\",\"message\":\"Pretty JSON\"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := &logSink{}
			logger := logger.ZeroLogger(zerolog.New(logs))

			prettyJSON(logger, tt.input)

			assert.Equal(t, 1, len(logs.logs))
			assert.Equal(t, tt.expected, logs.logs[0])
		})
	}
}

func TestStatusesToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		statuses Statuses
		expected []string
	}{
		{name: "empty", statuses: Statuses{}, expected: []string{}},
		{name: "single", statuses: Statuses{{Name: "check", State: "success"}}, expected: []string{"check"}},
		{name: "multiple", statuses: Statuses{{Name: "success check", State: "success"}, {Name: "neutral check", State: "neutral"}, {Name: "skipped check", State: "skipped"}, {Name: "completed check", State: "completed"}, {Name: "failed check", State: "failed"}}, expected: []string{"success check", "neutral check", "skipped check", "completed check"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.statuses.ToString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		pauseTime        *time.Duration
		maxWaitTime      *time.Duration
		maxAttempts      int
		expectedAttempts int
	}{
		{name: "default retry", pauseTime: newDuration(1 * time.Millisecond), maxAttempts: 5, expectedAttempts: 5},
		{name: "retry exceed wait time", pauseTime: newDuration(1 * time.Second), maxWaitTime: newDuration(1 * time.Millisecond), maxAttempts: -1, expectedAttempts: 1},
		{name: "retry exceed attempts", pauseTime: newDuration(1 * time.Millisecond), maxWaitTime: newDuration(5 * time.Second), maxAttempts: 2, expectedAttempts: 2},
		{name: "without retry", pauseTime: newDuration(-1 * time.Millisecond), maxWaitTime: newDuration(-5 * time.Second), maxAttempts: 0, expectedAttempts: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry := newRetry(tt.pauseTime, tt.maxWaitTime, tt.maxAttempts)
			retry.init()
			retry.start(tt.name)
			for !retry.retryExceeded() {
				retry.increment()
				time.Sleep(retry.PauseTime)
			}
			fmt.Printf("Retry: %+v", retry)
			assert.Equal(t, tt.expectedAttempts, retry.Attempts)
		})
	}
}

func TestRetryInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pauseTime     *time.Duration
		maxWaitTime   *time.Duration
		maxAttempts   int
		expectedRetry Retry
	}{
		{name: "default retry", expectedRetry: Retry{PauseTime: 20 * time.Second, MaxWaitTime: 10 * time.Minute, MaxAttempts: 0}},
		{name: "zero retry",
			pauseTime: newDuration(-1 * time.Millisecond), maxWaitTime: newDuration(-5 * time.Second), maxAttempts: -1,
			expectedRetry: Retry{PauseTime: 0 * time.Millisecond, MaxWaitTime: 0 * time.Second, MaxAttempts: math.MaxInt32}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry := newRetry(tt.pauseTime, tt.maxWaitTime, tt.maxAttempts)
			retry.init()
			assert.Equal(t, tt.expectedRetry, retry)
		})
	}
}

func newDuration(dur time.Duration) *time.Duration {
	return &dur
}

func newRetry(pauseTime *time.Duration, maxWaitTime *time.Duration, maxAttempts int) Retry {
	retry := Retry{
		Attempts: 0,
	}
	if pauseTime != nil {
		retry.PauseTime = *pauseTime
	}
	if maxWaitTime != nil {
		retry.MaxWaitTime = *maxWaitTime
	}
	retry.MaxAttempts = maxAttempts
	return retry
}

// utility functions

func makeTestClient() *Client {
	rp := testdata.NewResponsePlayer("testdata")
	return Newclient(&http.Client{Transport: rp})
}

type logSink struct{ logs []string }

func (l *logSink) Index(i int) string { return l.logs[i] }

func (l *logSink) Write(p []byte) (n int, err error) {
	l.logs = append(l.logs, string(p))
	return len(p), nil
}
