package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/queue"

	"github.com/palantir/go-githubapp/githubapp"

	"github.com/stretchr/testify/assert"

	"github.com/containifyci/dunebot/pkg/github/testdata"
)

func TestHandles(t *testing.T) {
	h := &PRHandler{}
	assert.Equal(t, []string{"pull_request"}, h.Handles())
}

type TestQueue struct {
	entries map[string][]queue.EventEntry
}

func (mq *TestQueue) Entries() map[string][]queue.EventEntry {
	return mq.entries
}

func (mq *TestQueue) AddEvent(ctx context.Context, pr queue.EventEntry) {
	prefix := fmt.Sprintf("%s/%s", pr.EventRepo.Repository.GetOwner().GetLogin(), pr.EventRepo.Repository.GetName())
	mq.entries[prefix] = append(mq.entries[prefix], pr)
}

func newTestQueue() *TestQueue {
	return &TestQueue{
		entries: make(map[string][]queue.EventEntry),
	}
}

func TestHandle(t *testing.T) {
	// Generate a new RSA private key with 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)
	cfg := config.Config{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 1,
				PrivateKey:    string(privateKeyPEMBytes),
			},
		},
	}
	cc, err := githubapp.NewDefaultCachingClientCreator(
		//TODO maybe find an better way then to copy the config
		cfg.Github.ToGithubAppConfig(),
		githubapp.WithClientUserAgent(cfg.AppConfig.UserAgent),
		githubapp.WithClientTimeout(cfg.AppConfig.ClientTimeOutDuration()),
		githubapp.WithTransport(makeTestClient()),
	)
	assert.NoError(t, err)

	queue := newTestQueue()

	h := &PRHandler{
		ClientCreator:   cc,
		RepositoryQueue: queue,
		Config:          &cfg,
	}

	ctx := context.Background()

	event := github.PullRequestEvent{
		Action: github.String("opened"),
		Installation: &github.Installation{
			ID: github.Int64(43975733),
		},
		PullRequest: &github.PullRequest{
			Number: github.Int(2),
			Head: &github.PullRequestBranch{
				Ref: github.String("develop"),
			},
			User: &github.User{
				Login: github.String("dune"),
			},
		},
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("test-owner"),
			},
			Name: github.String("test-repo"),
		},
	}

	b, err := json.Marshal(event)
	assert.NoError(t, err)

	err = h.Handle(ctx, "pull_request", "", b)

	assert.NoError(t, err)

	assert.Equal(t, 1, len(queue.Entries()))

	entry := queue.Entries()["test-owner/test-repo"][0]
	assert.Equal(t, "test-owner", entry.EventRepo.Repository.GetOwner().GetLogin())
	assert.Equal(t, "test-repo", entry.EventRepo.Repository.GetName())

	event, ok := entry.Event.(github.PullRequestEvent)
	assert.True(t, ok)
	assert.Equal(t, "opened", event.GetAction())
	assert.Equal(t, int(2), event.GetPullRequest().GetNumber())
}

func makeTestClient() http.RoundTripper {
	return testdata.NewResponsePlayer("testdata")
}
