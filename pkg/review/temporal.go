package review

import (
	"context"
	"fmt"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/logger"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
)

const (
	//TODO make this configurable
	WorkflowName = "PullRequestReviewWorkflow"
	TaskQueue    = "hello-world"
)

type (
	TemporalReviewer struct {
		logger logger.Logger
		config config.ApplicationConfig
		opts []grpc.DialOption
	}
)

func NewTemporalReviewer(logger logger.Logger, config config.ApplicationConfig, opts ...grpc.DialOption) Reviewer {
	return &TemporalReviewer{
		logger: logger,
		config: config,
		opts: opts,
	}
}

func (r *TemporalReviewer) PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) error {
	c, err := client.Dial(client.Options{
		HostPort: r.config.ReviewerConfig.Address,
		ConnectionOptions: client.ConnectionOptions{
			DialOptions: r.opts,
		},
	})
	if err != nil {
		r.logger.Error("Unable to create client", "Error", err)
		return err
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:                    fmt.Sprintf("pull_request_review_%s_%s", pullRequestReview.Repository.Owner, *pullRequestReview.Repository.Name),
		TaskQueue:             TaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, WorkflowName, pullRequestReview)
	if err != nil {
		r.logger.Error("Unable to execute workflow", "error", err)
		return err
	}

	r.logger.Debug("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())
	return nil
}
