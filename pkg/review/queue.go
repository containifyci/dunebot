package review

import (
	"context"
	"fmt"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/containifyci/dunebot/pkg/queue"
	"github.com/palantir/go-githubapp/githubapp"
)

type (
	QueueReviewer struct {
		logger        logger.Logger
		clientCreator githubapp.ClientCreator
		config        config.Config
		queue         queue.Queue
		reviewer      Reviewer
	}
)

func NewQueueReviewer(logger logger.Logger, queue queue.Queue, reviewer Reviewer, clientCreator githubapp.ClientCreator, config config.Config) Reviewer {
	return &QueueReviewer{
		logger:        logger,
		clientCreator: clientCreator,
		config:        config,
		queue:         queue,
		reviewer:      reviewer,
	}
}

func (r *QueueReviewer) PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) error {
	r.queue.AddEvent(ctx, queue.EventEntry{
		EventRepo: queue.EventRepo{
			PullRequest: *pullRequestReview.PullRequest,
			Repository:  *pullRequestReview.Repository,
			AppConfig:   pullRequestReview.Config,
		},
		Event:        *pullRequestReview.Event,
		EventHandler: r,
	})
	return nil
}

func (r *QueueReviewer) HandleEvent(prEntry queue.EventEntry) error {
	event, ok := prEntry.Event.(github.PullRequestEvent)
	if !ok {
		r.logger.Error("invalid event type expect github.PullRequestEvent", "event", fmt.Sprintf("%T", prEntry.Event))
		return fmt.Errorf("invalid event type: %T", prEntry.Event)
	}

	err := r.reviewer.PullRequestReview(context.Background(), PullRequestReview{
		PullRequest: &prEntry.EventRepo.PullRequest,
		Repository:  &prEntry.EventRepo.Repository,
		Config:      prEntry.EventRepo.AppConfig,
		Event:       &event,
	})
	if err != nil {
		r.logger.Error("error processing queue event", "error", err)
		return fmt.Errorf("error processing queue event: %w", err)
	}
	return nil
}
