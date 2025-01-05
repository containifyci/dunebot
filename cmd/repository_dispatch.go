package cmd

import (
	"context"
	"encoding/json"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/containifyci/dunebot/pkg/github"
)

type RepositoryDispatchPayload struct {
	PullRequest *github.PullRequest `json:"pull_request"`
	Owner       string              `json:"owner"`
	Repository  string              `json:"repository"`
}

type RepositoryDispatchHandler struct {
	Handler githubapp.EventHandler
}

func (h *RepositoryDispatchHandler) Handles() []string {
	return []string{"repository_dispatch"}
}

func (h *RepositoryDispatchHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	if eventType != "repository_dispatch" {
		return errors.Errorf("unsupported event type: %s", eventType)
	}

	var event github.RepositoryDispatchEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse repository dispatch payload")
	}

	repo := event.GetRepo()

	var clientPayload RepositoryDispatchPayload
	if err := json.Unmarshal(event.ClientPayload, &clientPayload); err != nil {
		log.Error().Err(err).Msg("failed to parse dispatch client payload ")
		return errors.Wrap(err, "failed to parse dispatch client payload")
	}

	log.Debug().Msgf("Processing Repository Dispatch event for PR %d", *clientPayload.PullRequest.Number)

	openedEventAction := "opened"

	prEvent := github.PullRequestEvent{
		Repo: repo,
		//Be careful the PullRequest only contains required fields like Number, State, User.Login, Head.Ref because repository_dispacth event has a strict payload size limit
		//The Pull Request will be loaded in the PRHandle anyway so we don't need to load all the fields here
		//https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#repository_dispatch
		PullRequest:  clientPayload.PullRequest,
		Action:       &openedEventAction,
		Installation: event.Installation,
	}

	prEventBytes, err := json.Marshal(prEvent)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal PR event")
		return errors.Wrap(err, "failed to marshal PR event")
	}

	return h.Handler.Handle(ctx, "pull_request", deliveryID, prEventBytes)
}
