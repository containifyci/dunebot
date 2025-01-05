package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

type DummyHandler struct {
	payload json.RawMessage
}

func (h *DummyHandler) Handles() []string {
	return []string{"dummy"}
}

func (h *DummyHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	h.payload = payload
	return nil
}

func TestRepositoryDispatchHandles(t *testing.T) {
	handler := &RepositoryDispatchHandler{}

	handles := handler.Handles()
	assert.Equal(t, []string{"repository_dispatch"}, handles)
}

func TestRepositoryDispatchHandle(t *testing.T) {
	dummyHandler := &DummyHandler{}
	handler := &RepositoryDispatchHandler{
		Handler: dummyHandler,
	}

	payload := RepositoryDispatchPayload{
		PullRequest: &github.PullRequest{
			Number: github.Int(2),
		},
		Owner:      "test-owner",
		Repository: "test-repo",
	}

	clientPayload, err := json.Marshal(payload)
	assert.NoError(t, err)

	event := github.RepositoryDispatchEvent{
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("test-owner"),
			},
			Name: github.String("test-repo"),
		},
		ClientPayload: clientPayload,
		Installation: &github.Installation{
			ID: github.Int64(43975733),
		},
	}

	b, err := json.Marshal(event)
	assert.NoError(t, err)

	err = handler.Handle(context.Background(), "repository_dispatch", "123", b)
	assert.NoError(t, err)

	expect := AssertHandlerPayload(t, event)

	assert.Equal(t, expect, dummyHandler.payload)
}

func AssertHandlerPayload(t *testing.T, event github.RepositoryDispatchEvent) json.RawMessage {

	var clientPayload RepositoryDispatchPayload
	err := json.Unmarshal(event.ClientPayload, &clientPayload)
	assert.NoError(t, err)

	prEvent := github.PullRequestEvent{
		Repo:         event.Repo,
		PullRequest:  clientPayload.PullRequest,
		Action:       github.String("opened"),
		Installation: event.Installation,
	}
	prEventBytes, err := json.Marshal(prEvent)
	assert.NoError(t, err)
	return json.RawMessage(prEventBytes)
}
