package review

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/containifyci/dunebot/oauth2"
	"github.com/containifyci/dunebot/pkg/auth"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/containifyci/dunebot/pkg/queue"
	"github.com/containifyci/dunebot/pkg/validator"
	"github.com/palantir/go-githubapp/githubapp"
)

type (
	PullRequestReview struct {
		PullRequest *github.PullRequest
		Repository  *github.Repository
		Config      *config.AppConfig
		Event       *github.PullRequestEvent
	}

	Reviewer interface {
		PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) error
		// ss() string
	}

	reviewer struct {
		clientCreator githubapp.ClientCreator
		transport     http.RoundTripper
		config        config.Config
		logger        logger.Logger
	}

	Option func(*reviewer)
)

func WithTransport(transport http.RoundTripper) Option {
	return func(r *reviewer) {
		r.transport = transport
	}
}

func NewReviewer(logger logger.Logger, queue queue.Queue, clientCreator githubapp.ClientCreator, config config.Config, opts ...Option) Reviewer {
	typ := config.AppConfig.ReviewerConfig.Type
	if typ == "" {
		typ = "queue"
	}
	switch typ {
	case "temporal":
		return NewTemporalReviewer(logger, config.AppConfig)
	case "queue":
		r := &reviewer{
			clientCreator: clientCreator,
			config:        config,
			logger:        logger,
		}
		for _, opt := range opts {
			opt(r)
		}
		return NewQueueReviewer(logger, queue,
			r, clientCreator, config)
	case "direct":
		r := &reviewer{
			clientCreator: clientCreator,
			config:        config,
			logger:        logger,
		}
		for _, opt := range opts {
			opt(r)
		}
		return r
	default:
		panic("Unknown reviewer type")
	}
}

func (r *reviewer) PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) (err error) {
	pr := pullRequestReview.PullRequest
	cc := r.clientCreator
	appCfg := pullRequestReview.Config
	installationId := pullRequestReview.Event.GetInstallation().GetID()
	repo := pullRequestReview.Repository
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	prNum := pr.GetNumber()

	client, err := cc.NewInstallationClient(installationId)
	if err != nil {
		r.logger.Error("Failed to create github client", "Error", err)
		return err
	}

	gh := github.NewClient(github.WithLogger(r.logger), github.WithGithubClient(client), github.WithContext(ctx), github.WithConfig(github.Config{
		GitHubRepository: repoName,
		GitHubOwner:      repoOwner,
	}))
	pr, _, err = gh.Client.PullRequests.Get(ctx, repoOwner, repoName, prNum)
	if err != nil {
		r.logger.Error("Error getting PR", "error", err)
		return err
	}

	if pr.GetState() == "closed" {
		r.logger.Debug("Pull request already closed")
		return nil
	}

	prC := github.NewPullRequestClient(&gh, pr)
	validator := validator.NewValidator(appCfg)

	approveResult := validator.ValidateApproveConditions(prC)

	repo, _, err = gh.Client.Repositories.Get(ctx, repoOwner, repoName)
	if err != nil {
		r.logger.Error("Error getting repository", "error", err)
		return err
	}
	r.logger.Debug("Checking if PR is approvable..")
	if approveResult.Success {
		r.logger.Debug("PR is approvable.")
		//TODO: use private key for secure communication
		authSrv := auth.NewSigningService("")
		accessToken := authSrv.CreateTokenFnc(auth.ServiceClaims{ServiceName: "dunebot"})

		cfg := r.config

		oauth2cfg := oauth2.Config{
			InstallationId:  fmt.Sprintf("%d", installationId),
			User:            appCfg.Approve.Approver,
			Ctx:             ctx,
			OAuth2Config:    cfg.Config().ToOAuth2Config(),
			AuthInterceptor: *oauth2.NewAuthInterceptor(accessToken),
			Addr:            cfg.Config().JWT.Address,
		}
		// ghu := github.NewClient(github.WithConfig(cfg))
		opts := []github.Option{github.WithTokenSource(oauth2cfg.TokenSourceFrom(ctx)), github.WithContext(ctx), github.WithConfig(github.NewRepositoryConfig(repoOwner, repoName))}
		if r.transport != nil {
			cli := http.DefaultClient
			cli.Transport = r.transport
			opts = append(opts, github.WithHttpClient(cli))
		}
		ghu := github.NewClient(opts...)

		sha := pr.GetHead().GetSHA()

		if gh.IsOutOfDate(pr) {
			r.logger.Debug("PR is out-of-date. Update branch.")
			tmplErr := ghu.AddRebaseComment(pr)

			if tmplErr != nil {
				r.logger.Warn("Error update pull request branch", "error", tmplErr)
				gh.AddCommentSilent(repo, prNum, tmplErr)
				return tmplErr
			}

			sha, pr, tmplErr = gh.WaitForRebase(&github.Retry{MaxAttempts: -1, MaxWaitTime: 10 * time.Minute}, pr)
			if err != nil {
				r.logger.Warn("Error waiting for rebase", "error", tmplErr)
				gh.AddCommentSilent(repo, prNum, tmplErr)
				return err
			}

			r.logger.Debug("Rebase is done. Checking if PR is now mergeable.")
		}

		if pr.GetState() == "closed" {
			r.logger.Debug("Pull request already closed")
			return nil
		}

		r.logger.Debug("Waiting for required status checks to complete.")
		// Wait for PR to pass all checks
		passed, tmplErr := gh.WaitForStatusChecks(github.Retry{MaxAttempts: -1}, sha, pr)

		if tmplErr != nil {
			r.logger.Warn("Error waiting for status checks", "error", tmplErr)
			gh.AddCommentSilent(repo, prNum, tmplErr)
			return tmplErr
		}

		if !passed {
			r.logger.Debug("Status checks failed can't be merged.")
			_, _ = gh.AddComment(repo, prNum, "Status checks failed can't be merged.")
			return nil
		}

		approveResult = validator.ValidateApproveCommentsAndChecks(github.NewPullRequestClient(&gh, pr))
		if !approveResult.Success {
			r.logger.Debug("PR can not be approved.", "reason", approveResult.Results)
			_, _ = gh.AddComment(repo, prNum, approveResult.String())
			return nil
		}

		r.logger.Debug("All checks passed. Approving PR.")
		// Approve the PR
		tmplErr = ghu.ApprovePullRequest(prNum)
		if tmplErr != nil {
			r.logger.Warn("Error approving PR", "error", tmplErr)
			gh.AddCommentSilent(repo, prNum, tmplErr)
			return tmplErr
		}
		mergeResult := validator.ValidateMergeConditions(github.NewPullRequestClient(&gh, pr))
		if !mergeResult.Success {
			r.logger.Debug("PR is not mergeable.", "result", mergeResult.Results)
			_, _ = gh.AddComment(repo, prNum, mergeResult.String())
			return nil
		}

		if mergeResult.Success {
			r.logger.Debug("Merge result is successful. Checking if all checks are passed in the main branch.")

			// First, ensure the main branch doesn't have any failed CI checks
			passed, tmplErr := gh.WaitForPRChecksToPassMain(&github.Retry{MaxAttempts: 10, PauseTime: time.Minute}, pr.GetBase().GetRef(), pr.GetBase().GetSHA(), "success", "skipped")
			if tmplErr != nil {
				r.logger.Error("Error while waiting for main branch checks", "error", tmplErr)
				gh.AddCommentSilent(repo, prNum, tmplErr)
				return tmplErr
			}
			if !passed {
				r.logger.Debug("CI checks on main branch failed; skipping merge.")
				_, err := gh.AddComment(repo, prNum, "CI checks on the main branch failed; skipping merge.")
				if err != nil {
					return err
				}
				return nil
			}

			// Now ensure the PR checks are also okay
			passed, tmplErr = gh.WaitForPRChecksToPass(&github.Retry{MaxAttempts: 10, PauseTime: time.Minute}, pr.GetHead().GetRef(), pr.GetHead().GetSHA(), "pull_request", "success", "skipped")
			if tmplErr != nil {
				r.logger.Error("Error while waiting for PR checks", "error", tmplErr)
				gh.AddCommentSilent(repo, prNum, tmplErr)
				return tmplErr
			}
			if !passed {
				r.logger.Debug("CI checks on PR failed; skipping merge.")
				_, err = gh.AddComment(repo, prNum, "CI checks on the PR failed; skipping merge.")
				if err != nil {
					return err
				}
				return nil
			}

			mergeResult = validator.ValidateMergeCommentsAndChecks(prC)
			if !mergeResult.Success {
				r.logger.Debug("PR is not mergeable.", "result", mergeResult.Results)
				_, _ = gh.AddComment(repo, prNum, mergeResult.String())
				return nil
			}

			// If all checks are passed, merge the PR
			r.logger.Debug("All checks passed. Proceeding to merge PR.")
			_, tmplErr = gh.MergePullRequest(prNum, appCfg.GetMerge().Method, repo)
			if tmplErr != nil {
				r.logger.Warn("Error merging PR", "error", tmplErr)
				gh.AddCommentSilent(repo, prNum, tmplErr)
				return tmplErr
			}
			r.logger.Debug("PR merged successfully and CI checks passed.", "pr", prNum, "repository", repoName, "owner", repoOwner)
			return nil
		}
	}

	r.logger.Debug("PR is not mergeable.", "result", approveResult.Results)
	_, _ = gh.AddComment(repo, prNum, approveResult.String())

	return nil
}
