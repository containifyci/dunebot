package github

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"net/http"
	"strings"
	"time"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v79/github"
	"github.com/rs/zerolog/log"

	"golang.org/x/oauth2"

	"github.com/containifyci/dunebot/pkg/compare"
	"github.com/containifyci/dunebot/pkg/logger"
)

type PullRequestEvent = github.PullRequestEvent

var Int = github.Ptr[int]
var Int64 = github.Ptr[int64]
var String = github.Ptr[string]
var Bool = github.Ptr[bool]

type Client = github.Client

type Response = github.Response

type CheckRun = github.CheckRun
type DeployKeyEvent = github.DeployKeyEvent
type GitHubAppAuthorizationEvent = github.GitHubAppAuthorizationEvent
type Installation = github.Installation
type IssueComment = github.IssueComment
type Label = github.Label
type ListCursorOptions = github.ListCursorOptions
type ListOptions = github.ListOptions
type RepositoryListByAuthenticatedUserOptions = github.RepositoryListByAuthenticatedUserOptions

type PullRequest = github.PullRequest
type PullRequestListOptions = github.PullRequestListOptions
type PullRequestBranch = github.PullRequestBranch
type PullRequestReviewDismissalRequest = github.PullRequestReviewDismissalRequest
type PullRequestReviewEvent = github.PullRequestReviewEvent
type Repository = github.Repository
type DispatchRequestOptions = github.DispatchRequestOptions
type RepositoryDispatchEvent = github.RepositoryDispatchEvent
type Timestamp = github.Timestamp
type Organization = github.Organization
type User = github.User
type WorkflowRun = github.WorkflowRun

var Newclient = github.NewClient

type Retry struct {
	Attempts  int
	PauseTime time.Duration
	startTime time.Time

	MaxWaitTime time.Duration
	MaxAttempts int
	Name        string
}

type GithubClient struct {
	Client      *github.Client
	logger      logger.Logger
	cfg         Config
	ctx         context.Context
	tokenSource oauth2.TokenSource
}

type GithubClientCreator interface {
	NewClient(opts ...Option) GithubClient
}

type Option func(*GithubClient)

func WithContext(ctx context.Context) Option {
	return func(g *GithubClient) {
		g.ctx = ctx
	}
}

func WithConfig(cfg Config) Option {
	return func(g *GithubClient) {
		g.cfg = cfg
	}
}

func WithLogger(logger logger.Logger) Option {
	return func(g *GithubClient) {
		g.logger = logger
	}
}

func WithHttpClient(cli *http.Client) Option {
	return func(g *GithubClient) {
		g.Client = github.NewClient(cli)
	}
}

func WithGithubClient(cli *github.Client) Option {
	return func(g *GithubClient) {
		g.Client = cli
	}
}

func WithTokenSource(tokenSource oauth2.TokenSource) Option {
	return func(g *GithubClient) {
		g.tokenSource = tokenSource
	}
}

func NewGithubClient(cli *http.Client) *github.Client {
	return github.NewClient(cli)
}

func (gc *GithubClient) NewClient(opts ...Option) GithubClient {
	return NewClient(opts...)
}

func NewClient(opts ...Option) GithubClient {

	gc := GithubClient{}

	for _, opt := range opts {
		opt(&gc)
	}

	if gc.ctx == nil {
		gc.ctx = context.Background()
	}

	if gc.logger == nil {
		gc.logger = logger.NewZeroLogger()
	}

	if gc.Client == nil {
		var tokenSource oauth2.TokenSource
		if gc.tokenSource == nil {
			tokenSource = oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: gc.cfg.GitHubToken},
			)
		} else {
			tokenSource = gc.tokenSource
		}
		tc := oauth2.NewClient(gc.ctx, tokenSource)

		rateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(tc.Transport)
		if err != nil {
			panic(err)
		}

		gc.Client = github.NewClient(rateLimiter)
		gc.Client.BaseURL.Host = gc.cfg.GitHubAPIHost
		gc.Client.BaseURL.Scheme = gc.cfg.GitHubAPIScheme
	}
	return gc
}

func (cli *GithubClient) WaitForPRChecksToPassMain(retry *Retry, branch string, sha string, requiredStates ...string) (bool, TemplatedError) {
	return cli.WaitForPRChecksToPass(retry, branch, sha, "push", requiredStates...)
}

func (cli *GithubClient) WaitForPRChecksToPass(retry *Retry, branch string, sha string, event string, requiredStates ...string) (bool, TemplatedError) {
	retry.init()
	retry.start("wait for PR checks")
	log.Debug().Msgf("Starting wait for CI checks on %s with SHA %s", branch, sha)

	for {
		statuses, tmplErr := cli.AllWorkflowsPassed(&Retry{MaxAttempts: 5, PauseTime: time.Minute}, &github.ListWorkflowRunsOptions{
			Event:               event,
			Branch:              branch,
			HeadSHA:             sha,
			ExcludePullRequests: true,
		})
		if tmplErr != nil {
			log.Error().Err(tmplErr).Msgf("Error fetching CI statuses: %v", tmplErr)
			return false, tmplErr
		}

		if len(statuses) == 0 {
			log.Debug().Msg("No workflows are found, proceeding without CI checks.")
			return true, nil
		}

		allChecksPassed := true
		for _, status := range statuses {
			if !slices.Contains(requiredStates, status.State) {
				allChecksPassed = false
				break
			}
		}

		if allChecksPassed {
			log.Debug().Msg("All required CI checks have passed.")
			return true, nil
		}

		if retry.retryExceeded() {
			log.Warn().Msg("Timed out waiting for PR checks to pass")
			return false, NewRetryExceedError(cli.ctx, *retry)
		}

		log.Debug().Msgf("Not all CI checks have passed, retrying after %s...", retry.PauseTime)
		time.Sleep(retry.PauseTime)
		retry.increment()
	}
}

func prettyJSON(logger logger.Logger, data any) {
	// Marshal the JSON data with an indent to make it pretty
	prettyJSON, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Error("Error marshalling JSON", "error", err)
		return
	}
	logger.Debug("Pretty JSON", "json", string(prettyJSON))
}

func (cli GithubClient) IsOutOfDate(pr *github.PullRequest) bool {
	// Check if the PR is out-of-date
	commitComparison, _, err := cli.Client.Repositories.CompareCommits(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetBase().GetRepo().GetDefaultBranch(), pr.GetHead().GetSHA(), nil)
	if err != nil {
		cli.logger.Error("Error comparing commits", "error", err)
		return true
	}

	prettyJSON(cli.logger, commitComparison)

	//TODO missing check for "diverged"
	return commitComparison.GetStatus() == "behind" || commitComparison.GetStatus() == "diverged"
}

// Is returns whether the provided error equals this error.
func IsAcceptedError(target error) bool {
	_, ok := target.(*github.AcceptedError)
	return ok
}

func (cli GithubClient) AddRebaseComment(pr *github.PullRequest) TemplatedError {
	update, _, err := cli.Client.PullRequests.UpdateBranch(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetNumber(), nil)
	if err != nil && !IsAcceptedError(err) {
		cli.logger.Error("Error updating branch", "error", err)
		return NewAPIError(cli.ctx, err)
	}
	cli.logger.Debug("Update Branch response", "response", update)
	return nil
}

func (cli GithubClient) AddComment(repo *github.Repository, prNumber int, body string) (*github.IssueComment, error) {
	prComment := github.IssueComment{
		Body: &body,
	}

	comment, _, err := cli.Client.Issues.CreateComment(cli.ctx, repo.GetOwner().GetLogin(), repo.GetName(), prNumber, &prComment)
	if err != nil {
		cli.logger.Error("Error writing comment", "error", err)
		return nil, NewAPIError(cli.ctx, err)
	}
	return comment, nil
}

// AddCommentSilent try to add comment to a pull request without error handling
func (cli GithubClient) AddCommentSilent(repo *github.Repository, prNumber int, err TemplatedError) {
	_, _ = cli.AddComment(repo, prNumber, err.Template())
}

func (cli GithubClient) WaitForRebase(retry *Retry, pr *github.PullRequest) (string, *github.PullRequest, TemplatedError) {
	retry.init()
	retry.start("rebase")

	oldSha := pr.GetHead().GetSHA()
	for {
		updatedPr, _, err := cli.Client.PullRequests.Get(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetNumber())
		if err != nil {
			cli.logger.Error("Error getting PR", "error", err)
		}

		cli.logger.Debug("Compare sha and pr sha", "sha", oldSha, "pr_sha", updatedPr.GetHead().GetSHA())
		if oldSha != updatedPr.GetHead().GetSHA() {
			cli.logger.Debug("PR has been rebased")
			return oldSha, updatedPr, nil
		}

		if retry.retryExceeded() {
			return oldSha, updatedPr, NewRetryExceedError(cli.ctx, *retry)
		}
		retry.increment()
		time.Sleep(retry.PauseTime)
	}
}

func (cli GithubClient) GetPRComments(prNumber int) ([]*github.IssueComment, error) {
	comments, _, err := cli.Client.Issues.ListComments(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, prNumber, nil)
	if err != nil {
		cli.logger.Error("Error listing comments", "error", err)
		return nil, err
	}
	return comments, nil
}

func (cli GithubClient) ApprovePullRequest(prNumber int) TemplatedError {
	_, _, err := cli.Client.PullRequests.CreateReview(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, prNumber, &github.PullRequestReviewRequest{
		Event: String("APPROVE"),
	})
	if err != nil {
		cli.logger.Error("Error merging PR", "error", err)
		return NewNotMergableError(cli.ctx, err)
	}
	return nil
}

func SafeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func GetMergeMethod(method string, repo *github.Repository) string {
	if method == "merge" && SafeBool(repo.AllowMergeCommit) {
		return "merge"
	} else if method == "squash" && SafeBool(repo.AllowSquashMerge) {
		return "squash"
	} else if method == "rebase" && SafeBool(repo.AllowRebaseMerge) {
		return "rebase"
	}
	switch {
	case SafeBool(repo.AllowMergeCommit):
		return "merge"
	case SafeBool(repo.AllowSquashMerge):
		return "squash"
	case SafeBool(repo.AllowRebaseMerge):
		return "rebase"
	default:
		return ""
	}
}

func (cli GithubClient) MergePullRequest(prNumber int, method string, repo *github.Repository) (*github.PullRequestMergeResult, TemplatedError) {
	mergeMethod := GetMergeMethod(method, repo)
	cli.logger.Debug("Merging PR", "method", mergeMethod, "conf_method", method, "merge", SafeBool(repo.AllowMergeCommit), "squash", SafeBool(repo.AllowSquashMerge), "rebase", SafeBool(repo.AllowRebaseMerge))
	commit, _, err := cli.Client.PullRequests.Merge(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, prNumber, "", &github.PullRequestOptions{MergeMethod: mergeMethod, DontDefaultIfBlank: false})
	if err != nil {
		cli.logger.Warn("Error merging PR", "method", mergeMethod, "conf_method", method, "merge", SafeBool(repo.AllowMergeCommit), "squash", SafeBool(repo.AllowSquashMerge), "rebase", SafeBool(repo.AllowRebaseMerge), "error", err)
		return nil, NewNotMergableError(cli.ctx, err)
	}
	return commit, nil
}

func isNotFound(err error) bool {
	rerr, ok := err.(*github.ErrorResponse)
	return ok && rerr.Response.StatusCode == http.StatusNotFound
}

type Statuses []*Status

func (stats Statuses) ToString() []string {
	finishedStatuses := []string{}
	//TODO make it possible to define required checks in config (not GitHub branch protection checks)
	//Please implement code the fullfill the following requirement. Iterate over statuses and log a debug messager and return false if an entries state is not success, neutral or skipped
	for _, s := range stats {
		if s.State == "success" || s.State == "neutral" || s.State == "skipped" || s.State == "completed" {
			finishedStatuses = append(finishedStatuses, s.Name)
		}
	}
	return finishedStatuses
}

type Status struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// TODO add documentation for that function
func (cli GithubClient) GetStatuses(pr *github.PullRequest) (Statuses, error) {
	opts := &github.ListOptions{PerPage: 100}
	var statuses []*Status

	for {
		combinedStatus, res, err := cli.Client.Repositories.GetCombinedStatus(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetHead().GetSHA(), opts)
		if err != nil {
			//TODO check if it should panic
			cli.logger.Error("Error getting combined status", "error", err)
		}
		if combinedStatus == nil {
			combinedStatus = &github.CombinedStatus{}
		}

		for _, s := range combinedStatus.Statuses {
			statuses = append(statuses, &Status{Name: s.GetContext(), State: s.GetState()})
		}

		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}

	checkOpts := &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 1000}}
	for {
		checkRuns, res, err := cli.Client.Checks.ListCheckRunsForRef(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetHead().GetSHA(), checkOpts)
		if err != nil {
			cli.logger.Error("Error getting check runs", "error", err)
		}

		for _, s := range checkRuns.CheckRuns {
			// map check run conclusion to status to simplify the validation
			if s.GetConclusion() != "" && s.GetConclusion() != "success" && s.GetConclusion() != "neutral" && s.GetConclusion() != "skipped" {
				cli.logger.Debug("Check run %s has conclusion %s.", s.GetName(), s.GetConclusion())
				statuses = append(statuses, &Status{Name: s.GetName(), State: s.GetConclusion()})
				continue
			}
			statuses = append(statuses, &Status{Name: s.GetName(), State: s.GetStatus()})
		}

		if res.NextPage == 0 {
			break
		}
		checkOpts.Page = res.NextPage
	}
	return statuses, nil
}

func (cli GithubClient) GetStatusChecksForPullRequest(ref string) ([]*github.CheckRun, error) {
	opts := &github.ListCheckRunsOptions{}
	checkRuns, _, err := cli.Client.Checks.ListCheckRunsForRef(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, ref, opts)
	if err != nil {
		return nil, err
	}
	return checkRuns.CheckRuns, nil
}

func (cli GithubClient) GetLastMergedPullRequest() (*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 20,
		},
	}
	prs, _, err := cli.Client.PullRequests.List(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, opts)
	if err != nil {
		return nil, err
	}

	for _, pr := range prs {
		cli.logger.Debug("merged PR", "number", pr.GetNumber(), "timestamp", pr.GetMergedAt())
		if (pr.GetMergedAt() != github.Timestamp{}) {
			cli.logger.Debug("found PR", "number", pr.GetNumber())
			return pr, nil
		}
	}
	return nil, fmt.Errorf("no merged pull request found")
}

func (cli GithubClient) GetLatestWorkflowRuns(opts *github.ListWorkflowRunsOptions) (map[string]*github.WorkflowRun, error) {
	runs, _, err := cli.Client.Actions.ListRepositoryWorkflowRuns(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, opts)
	if err != nil {
		return nil, err
	}

	latestRuns := make(map[string]*github.WorkflowRun)
	for _, run := range runs.WorkflowRuns {
		if existingRun, exists := latestRuns[run.GetName()]; !exists || run.GetRunNumber() > existingRun.GetRunNumber() {
			latestRuns[run.GetName()] = run
		}
	}
	return latestRuns, nil
}

func (cli GithubClient) AllWorkflowsPassed(retry *Retry, opts *github.ListWorkflowRunsOptions) ([]*Status, TemplatedError) {
	//find the last merged PR
	lastMergedPR, err := cli.GetLastMergedPullRequest()
	if err != nil {
		cli.logger.Error("Error fetching last merged PR", "error", err)
	}

	if lastMergedPR != nil {
		//get the last executed workflow run from the last merged PR as a baseline
		cli.logger.Debug("Found last merged PR", "lastPRNum", lastMergedPR.Number)
		lastOpts := &github.ListWorkflowRunsOptions{
			Event:               "pull_request",
			HeadSHA:             lastMergedPR.GetHead().GetSHA(),
			Branch:              lastMergedPR.GetHead().GetRef(),
			ExcludePullRequests: true,
		}
		runs, err := cli.GetLatestWorkflowRuns(lastOpts)
		if err != nil {
			cli.logger.Error("Error fetching latest workflow runs", "error", err)
			return nil, NewAPIError(cli.ctx, err)
		}
		expectedCheckNames := []string{}
		for key := range runs {
			expectedCheckNames = append(expectedCheckNames, key)
		}

		var statuses []*Status
		retry.init()
		retry.start("check status")
		for {
			statuses = []*Status{}
			workflowRuns, err := cli.GetLatestWorkflowRuns(opts)
			if err != nil {
				cli.logger.Error("Error listing workflow runs", "error", err)
				return nil, NewAPIError(cli.ctx, err)
			}

			for _, run := range workflowRuns {
				state := run.GetStatus()
				if run.GetConclusion() != "" {
					state = run.GetConclusion()
				}
				statuses = append(statuses, &Status{Name: run.GetName(), State: state})
			}

			if opts.Event == "push" && len(statuses) == 0 {
				//if there were no workflow runs in the main branch consider this as a success
				return statuses, nil
			}

			//if the current PR has the same amount of workflow runs or more then the last merged PR then we can consider all checks are at least queued
			if len(statuses) >= len(expectedCheckNames) {
				return statuses, nil
			}
			if retry.retryExceeded() {
				break
			}
			retry.increment()
			time.Sleep(retry.PauseTime)
		}
		cli.logger.Debug("Stop waiting for status check to be queued.")
		return statuses, nil
	}
	cli.logger.Debug("No last merged PR found, just ")
	var statuses []*Status

	log.Debug().Msgf("repos/%s/%s/actions/runs", cli.cfg.GitHubOwner, cli.cfg.GitHubRepository)
	workflowRuns, _, err := cli.Client.Actions.ListRepositoryWorkflowRuns(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, opts)
	log.Debug().Msgf("Workflow runs are listed for repo")
	//TODO make it configurable how long to retry to wait for the workflow to run or finish
	for i := 0; i < 3; i++ {
		statuses = []*Status{}
		if err != nil {
			cli.logger.Error("Error listing workflow runs", "error", err)
			return nil, NewAPIError(cli.ctx, err)
		}
		log.Debug().Msgf("Number of Workflow runs %d", len(workflowRuns.WorkflowRuns))
		for _, run := range workflowRuns.WorkflowRuns {
			//if workflow run is not failed or was cancelled then consider status checks failed or even missing.
			log.Debug().Msgf("Workflow runs states conclusion: %s", run.GetConclusion())

			if run.GetConclusion() == "failure" || run.GetConclusion() == "cancelled" {
				cli.logger.Debug("Workflow run", "name", run.GetName(), "conclusion", run.GetConclusion())

				return nil, NewWorkflowRunError(run)
			}
			statuses = append(statuses, &Status{Name: run.GetName(), State: run.GetConclusion()})
		}
	}
	return statuses, nil
}

func (cli GithubClient) WaitForStatusChecks(retry Retry, sha string, pr *github.PullRequest) (bool, TemplatedError) {
	retry.init()
	retry.start("wait for status checks of pull request")

	for {
		pr, _, err := cli.Client.PullRequests.Get(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetNumber())
		if err != nil {
			cli.logger.Error("Error getting PR", "error", err)
			return false, NewAPIError(cli.ctx, err)
		}

		cli.logger.Debug("Check status", "sha", sha, "pr_sha", pr.GetHead().GetSHA())

		protection, _, err := cli.Client.Repositories.GetBranchProtection(cli.ctx, cli.cfg.GitHubOwner, cli.cfg.GitHubRepository, pr.GetBase().GetRef())
		if err != nil {
			if isNotFound(err) || err == github.ErrBranchNotProtected {
				protection = &github.Protection{}
			} else {
				cli.logger.Error("Error getting branch protection", "error", err)
				return false, NewAPIError(cli.ctx, err)
			}
		}

		requiredChecks := []string{}
		if checks := protection.GetRequiredStatusChecks(); checks != nil {
			if checks.Contexts != nil {
				requiredChecks = *checks.Contexts
			}
		}

		_, tmplErr := cli.AllWorkflowsPassed(&Retry{MaxAttempts: 6, PauseTime: 20 * time.Second}, &github.ListWorkflowRunsOptions{
			Event:               "pull_request",
			HeadSHA:             pr.GetHead().GetSHA(),
			ExcludePullRequests: true,
		})
		if tmplErr != nil {
			return false, tmplErr
		}

		statuses, err := cli.GetStatuses(pr)
		if err != nil {
			cli.logger.Error("Error getting success statuses", "error", err)
			return false, NewAPIError(cli.ctx, err)
		}

		finishedStatuses := []string{}
		inProgres := false
		//TODO make it possible to define required checks in config (not GitHub branch protection checks)
		//Please implement code the fullfill the following requirement. Iterate over statuses and log a debug messager and return false if an entries state is not success, neutral or skipped
		for _, s := range statuses {
			if s.State == "" {
				cli.logger.Debug("Status has no state", "name", s.Name)
				inProgres = true
				break
			}
			if s.State != "success" && s.State != "neutral" && s.State != "skipped" && s.State != "completed" && s.State != "in_progress" && s.State != "queued" {
				cli.logger.Debug("Status has a different state then is not success, neutral, skipped, completed, in_progress or queued", "name", s.Name, "state", s.State)
				return false, NewStatusCheckError(fmt.Sprintf("Status '%s' has a different state '%s' then is not success, neutral, skipped, completed, in_progress or queued", s.Name, s.State))
			}
			if s.State == "success" || s.State == "neutral" || s.State == "skipped" || s.State == "completed" {
				finishedStatuses = append(finishedStatuses, s.Name)
			}
			if s.State == "in_progress" || s.State == "queued" {
				cli.logger.Debug("Status has a state which is not finished", "name", s.Name, "state", s.State)
				inProgres = true
				break
			}
		}
		if inProgres {
			retry.increment()
			time.Sleep(retry.PauseTime)
			continue
		}

		unsatisfiedStatuses := compare.Complement(finishedStatuses, requiredChecks)

		cli.logger.Debug("Checking statuses...")
		prettyJSON(cli.logger, requiredChecks)
		prettyJSON(cli.logger, statuses)
		prettyJSON(cli.logger, unsatisfiedStatuses)

		if len(unsatisfiedStatuses) <= 0 {
			return true, nil
		}
		cli.logger.Debug("is deemed not mergeable because of unfulfilled status checks", "title", pr.GetTitle(), "missing_checks", strings.Join(unsatisfiedStatuses, ","))
		if retry.retryExceeded() {
			cli.logger.Debug("Timed out waiting for PR checks to pass")
			return false, NewRetryExceedError(cli.ctx, retry)
		}
		retry.increment()
		time.Sleep(retry.PauseTime)
	}
}

func (r Retry) retryExceeded() bool {
	return time.Since(r.startTime) > r.MaxWaitTime ||
		r.Attempts >= r.MaxAttempts
}

func (r *Retry) start(name string) {
	r.Name = name
	r.startTime = time.Now()
}

func (r *Retry) increment() {
	r.Attempts += 1
}

func (r *Retry) init() {
	var zero time.Duration

	if r.MaxAttempts < 0 {
		r.MaxAttempts = math.MaxInt32
	}
	if r.MaxWaitTime == zero {
		r.MaxWaitTime = 10 * time.Minute
	} else if r.MaxWaitTime < zero {
		r.MaxWaitTime = 0
	}
	if r.PauseTime == zero {
		r.PauseTime = 20 * time.Second
	} else if r.PauseTime < zero {
		r.PauseTime = 0
	}
}
