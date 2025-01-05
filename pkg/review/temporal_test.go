package review

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// Mock gRPC interceptor to simulate Temporal behavior
func mockTemporalInterceptor(methods ...string) func (
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	return func (
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		for _, m := range methods {
			if method == fmt.Sprintf("/temporal.api.workflowservice.v1.WorkflowService/%s", m) {
				return nil
			}
		}
		return errors.New("mock workflow execution error")
	}
}

func TestPullRequestReviewSuccess(t *testing.T) {
	t.Parallel()
	mockLogger := logger.NewZeroLogger()
	mockConfig := config.ApplicationConfig{
		ReviewerConfig: config.ReviewerConfig{Address: "localhost:7233"},
	}

	reviewer := NewTemporalReviewer(mockLogger, mockConfig, grpc.WithUnaryInterceptor(mockTemporalInterceptor("GetSystemInfo", "StartWorkflowExecution")))

	prReview := PullRequestReview{
		PullRequest: &github.PullRequest{Number: github.Int(1)},
		Repository:  &github.Repository{Name: github.String("repo")},
		Event:       &github.PullRequestEvent{Action: github.String("opened")},
		Config:      &config.AppConfig{},
	}

	err := reviewer.PullRequestReview(context.Background(), prReview)
	assert.NoError(t, err)
}

func TestPullRequestReviewUnKnownWorkflow(t *testing.T) {
	t.Parallel()
	mockLogger := logger.NewZeroLogger()
	mockConfig := config.ApplicationConfig{
		ReviewerConfig: config.ReviewerConfig{Address: "localhost:7233"},
	}

	reviewer := NewTemporalReviewer(mockLogger, mockConfig, grpc.WithUnaryInterceptor(mockTemporalInterceptor("GetSystemInfo")))

	prReview := PullRequestReview{
		PullRequest: &github.PullRequest{Number: github.Int(1)},
		Repository:  &github.Repository{Name: github.String("repo")},
		Event:       &github.PullRequestEvent{Action: github.String("opened")},
		Config:      &config.AppConfig{},
	}

	err := reviewer.PullRequestReview(context.Background(), prReview)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock workflow execution error")
}

func TestPullRequestReviewClientError(t *testing.T) {
	t.Parallel()
	mockLogger := logger.NewZeroLogger()
	mockConfig := config.ApplicationConfig{
		ReviewerConfig: config.ReviewerConfig{Address: "invalid-host"},
	}

	reviewer := NewTemporalReviewer(mockLogger, mockConfig)

	prReview := PullRequestReview{
		PullRequest: &github.PullRequest{Number: github.Int(1)},
		Repository:  &github.Repository{Name: github.String("repo")},
		Event:       &github.PullRequestEvent{Action: github.String("opened")},
		Config:      &config.AppConfig{},
	}

	err := reviewer.PullRequestReview(context.Background(), prReview)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed reaching server")
}
