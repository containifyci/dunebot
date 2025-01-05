package cmd

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"

	"github.com/containifyci/dunebot/pkg/auth"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	log "github.com/containifyci/dunebot/pkg/logger"
	"github.com/containifyci/dunebot/pkg/queue"
	"github.com/containifyci/dunebot/pkg/review"
)

type PRHandler struct {
	githubapp.ClientCreator
	RepositoryQueue queue.Queue
	*config.Config
	AuthSrv *auth.AuthService
}

func (h *PRHandler) Handles() []string {
	return []string{"pull_request"}
}

// func preparePRContext(ctx context.Context, installationID int64, repo *github.Repository, number int) (context.Context, zerolog.Logger) {
// 	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, number)
// 	ctx = context.WithValue(ctx, githubapp.LogKeyRepositoryName, repo.GetName()) //nolint:all
// 	ctx = context.WithValue(ctx, githubapp.LogKeyPRNum, number)                  //nolint:all
// 	return ctx, logger
// }

// // HandleEvent is to handle the pull request event implementing the HandleEvent interface
// func (h *PRHandler) HandleEvent(prEntry queue.EventEntry) (err error) {
// 	ctx := context.Background()
// 	pr := &prEntry.EventRepo.PullRequest

// 	//TODO check type before casting
// 	event := prEntry.Event.(github.PullRequestEvent)
// 	repo := event.GetRepo()
// 	prNum := pr.GetNumber()
// 	installationID := githubapp.GetInstallationIDFromEvent(&event)

// 	ctx, logger := preparePRContext(ctx, installationID, repo, prNum)
// 	logger.Debug().Msgf("Event action is %s", event.GetAction())

// 	if event.GetAction() != "opened" && event.GetAction() != "reopened" && event.GetAction() != "labeled" {

// 		logger.Debug().Msgf("Skip event action %s", event.GetAction())
// 		return nil
// 	}

// 	appCfg := prEntry.EventRepo.AppConfig

// 	repoOwner := repo.GetOwner().GetLogin()
// 	repoName := repo.GetName()
// 	// author := pr.GetUser().GetLogin()

// 	reviewer := review.NewReviewer(log.ZeroLogger(logger), h.ClientCreator, *h.Config)

// 	err = reviewer.PullRequestReview(ctx, review.PullRequestReview{
// 		Number:         prNum,
// 		Repository:     repoName,
// 		Owner:          repoOwner,
// 		InstallationId: installationID,
// 		Config:         *appCfg,
// 	})
// 	if err != nil {
// 		logger.Error().Err(err).Msgf("Error while processing PR %s/%s#%d", repoOwner, repoName, prNum)
// 		return err
// 	}
// 	return nil
// }

func attachEventAction(logctx zerolog.Context, action string) zerolog.Context {
	if action != "" {
		return logctx.Str("github_event_action", action)
	}
	return logctx
}

// PreparePRContext adds information about a pull request to the logger in a
// context and returns the modified context and logger.
func AddEventActionToContext(ctx context.Context, _ *zerolog.Logger, action string) (context.Context, zerolog.Logger) {
	logctx := zerolog.Ctx(ctx).With()

	logctx = attachEventAction(logctx, action)

	logger := logctx.Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	return logger.WithContext(ctx), logger
}

func (h *PRHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request payload")
	}

	repo := event.GetRepo()
	repoOwner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	installationID := githubapp.GetInstallationIDFromEvent(&event)

	client, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	appCfg, cnt, err := config.LoadConfig(ctx, client, repoOwner, repoName, event.Repo.GetDefaultBranch())
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msgf("Try to Load App Config %s", string(cnt))
		return err
	}
	if appCfg == nil {
		appCfg = &config.AppConfig{}
	}

	prNum := event.GetPullRequest().GetNumber()
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, prNum)
	ctx, logger = AddEventActionToContext(ctx, &logger, event.GetAction())

	if event.GetAction() != "review_requested" && event.GetAction() != "synchronize" && event.GetAction() != "opened" && event.GetAction() != "reopened" {
		zerolog.Ctx(ctx).Debug().Msgf("Skip event action %s", event.GetAction())
		return nil
	}

	if appCfg.NoApprove() {
		logger.Debug().Msg("Skip because Approve config is not defined or disabled")
		return nil
	}

	if len(appCfg.Approve.GetInclude().Branches) > 0 {
		var foundBranch *config.Branch
		for _, branch := range appCfg.Approve.GetInclude().Branches {
			if strings.HasPrefix(event.PullRequest.GetHead().GetRef(), branch.Prefix) {
				foundBranch = branch
				break
			}
		}

		if foundBranch == nil {
			logger.Debug().Msgf("Skip because branch %s is not configured to process %s.", event.PullRequest.GetHead().GetRef(), maps.Keys(appCfg.Approve.GetInclude().Branches))
			return nil
		}
	}

	if event.GetPullRequest().GetState() == "closed" {
		logger.Debug().Msg("Pull request already closed")
		return nil
	}

	logger.Debug().Msg("Add Pull Request action to the queue")

	// // //TODO only add to queue if review type is not temporal
	// h.RepositoryQueue.AddEvent(ctx, queue.EventEntry{
	// 	EventRepo: queue.EventRepo{
	// 		PullRequest: *event.PullRequest,
	// 		Repository:  *event.Repo,
	// 		AppConfig:   appCfg,
	// 	},
	// 	Event:        event,
	// 	EventHandler: h,
	// })

	reviewer := review.NewReviewer(log.ZeroLogger(logger), h.RepositoryQueue, h.ClientCreator, *h.Config)

	err = reviewer.PullRequestReview(ctx, review.PullRequestReview{
		PullRequest: event.PullRequest,
		Repository:  event.Repo,
		Config:      appCfg,
		Event:       &event,
	})
	if err != nil {
		logger.Error().Err(err).Msgf("Error while processing PR %s/%s#%d", repoOwner, repoName, prNum)
		return err
	}
	return nil
}
