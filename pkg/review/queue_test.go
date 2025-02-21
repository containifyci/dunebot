package review

import (
	"context"
	"fmt"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/containifyci/dunebot/pkg/queue"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"
)

func TestQueueReviewerPullRequestReview(t *testing.T) {
	t.Parallel()
	testCase := newReviewTestCase()

	reviewer := NewQueueReviewer(testCase.logger, testCase.queue, nil, testCase.cc, testCase.cfg)
	prReview := newPullRequestReview()

	err := reviewer.PullRequestReview(context.Background(), prReview)

	assert.NoError(t, err)
	assert.Len(t, testCase.queue.Entries(), 1)
}

func TestQueueReviewerHandleEvent(t *testing.T) {
	t.Parallel()
	testCase := newReviewTestCase()

	testReviewer := &testReviewer{}
	queueReviewer := testCase.newQueueReviewer(testReviewer)

	err := queueReviewer.HandleEvent(newEventEntry())

	assert.NoError(t, err)
	assert.Equal(t, 1, testReviewer.pullRequestReview.PullRequest.GetNumber())
	assert.Equal(t, "repo", testReviewer.pullRequestReview.Repository.GetName())
}

func TestQueueReviewerHandleInvalidEventError(t *testing.T) {
	t.Parallel()
	testCase := newReviewTestCase()

	queueReviewer := testCase.newQueueReviewer(nil)
	event := newEventEntry()
	event.Event = github.DeployKeyEvent{Action: github.String("opened")}

	err := queueReviewer.HandleEvent(event)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid event type: github.DeployKeyEvent")
}

func TestQueueReviewerHandleReviewError(t *testing.T) {
	t.Parallel()
	testCase := newReviewTestCase()

	testReviewer := &testErrorReviewer{err: assert.AnError}
	queueReviewer := &QueueReviewer{
		logger:        testCase.logger,
		clientCreator: testCase.cc,
		config:        testCase.cfg,
		queue:         testCase.queue,
		reviewer:      testReviewer,
	}
	err := queueReviewer.HandleEvent(queue.EventEntry{
		EventRepo: queue.EventRepo{
			PullRequest: github.PullRequest{Number: github.Int(1)},
			Repository:  github.Repository{Name: github.String("repo")},
			AppConfig:   &config.AppConfig{},
		},
		Event: github.PullRequestEvent{Action: github.String("opened")},
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

// test utilities

type reviewTestCase struct {
	logger logger.Logger
	queue  queue.Queue
	cc     githubapp.ClientCreator
	cfg    config.Config
}

func newReviewTestCase() *reviewTestCase {
	return &reviewTestCase{
		logger: logger.NewZeroLogger(),
		queue:  newTestQueue(),
		cc:     githubapp.NewClientCreator("", "", -1, nil),
		cfg: config.Config{
			Github: config.GithubConfig{
				App: struct {
					IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
					WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
					PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
				}{
					IntegrationID: 1,
				},
			},
		},
	}
}

func (r *reviewTestCase) newQueueReviewer(reviewer Reviewer) *QueueReviewer {
	return &QueueReviewer{
		logger:        r.logger,
		clientCreator: r.cc,
		config:        r.cfg,
		queue:         r.queue,
		reviewer:      reviewer,
	}
}

func newPullRequestReview() PullRequestReview {
	return PullRequestReview{
		PullRequest: &github.PullRequest{Number: github.Int(1)},
		Repository:  &github.Repository{Name: github.String("repo")},
		Event:       &github.PullRequestEvent{Action: github.String("opened")},
		Config:      &config.AppConfig{},
	}
}

func newEventEntry() queue.EventEntry {
	prReview := newPullRequestReview()
	return queue.EventEntry{
		EventRepo: queue.EventRepo{
			PullRequest: *prReview.PullRequest,
			Repository:  *prReview.Repository,
			AppConfig:   prReview.Config,
		},
		Event: *prReview.Event,
	}
}

type mockQueue struct {
	entries map[string][]queue.EventEntry
}

func newTestQueue() *mockQueue {
	return &mockQueue{
		entries: make(map[string][]queue.EventEntry),
	}
}

func (m *mockQueue) Entries() map[string][]queue.EventEntry {
	return m.entries
}

func (m *mockQueue) AddEvent(ctx context.Context, eventEntry queue.EventEntry) {
	prefix := fmt.Sprintf("%s/%s", eventEntry.EventRepo.Repository.GetOwner().GetLogin(), eventEntry.EventRepo.Repository.GetName())
	m.entries[prefix] = append(m.entries[prefix], eventEntry)
}

type testReviewer struct {
	pullRequestReview PullRequestReview
}

func (r *testReviewer) PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) error {
	r.pullRequestReview = pullRequestReview
	return nil
}

type testErrorReviewer struct {
	err error
}

func (r *testErrorReviewer) PullRequestReview(ctx context.Context, pullRequestReview PullRequestReview) error {
	return r.err
}
