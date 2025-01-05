package cmd

import (
	"context"
	"encoding/json"

	"github.com/palantir/go-githubapp/githubapp"
	appoauth "github.com/containifyci/dunebot/oauth2"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type AppHandler struct {
	githubapp.ClientCreator
	*config.Config
}

func (h *AppHandler) Handles() []string {
	return []string{"github_app_authorization"}
}

func (h *AppHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.GitHubAppAuthorizationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request payload")
	}

	if event.GetAction() != "revoked" {
		zerolog.Ctx(ctx).Debug().Msgf("Skip event action %s", event.GetAction())
	}

	event.GetSender().GetLogin()
	installationID := event.GetInstallation().GetID()

	oauth2cfg := appoauth.Config{
			InstallationId: installationID,
			//TODO make the account name configurable
			User:           "fr12k",
			Ctx:            ctx,
			OAuth2Config:         appoauth.GetConfig(h.Config),
	}

	oauth2cfg.TokenSourceFrom(ctx)

	zerolog.Ctx(ctx).Debug().Msgf("Process event %v+", event)

	ctx, logger := AddEventActionToContext(ctx, nil, event.GetAction())

	gh := github.NewClient(github.WithTokenSource(oauth2cfg.TokenSourceFrom(ctx)), github.WithContext(ctx))

	installs, _, err := gh.Client.Apps.ListUserInstallations(ctx, nil)
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to list installations for %s", oauth2cfg.User)
	}

	for _, install := range installs {
		logger.Debug().Msgf("Retrieved following installations %v+", install)
	}


	// oauth2cfg := appoauth.Config{
	// 	InstallationId: installationID,
	// 	//TODO make the account name configurable
	// 	User:           event.GetSender().GetLogin(),
	// 	Ctx:            ctx,
	// 	Config:         appoauth.GetConfig(h.Config),
	// }

	//TODO revoke token can not be based on installationId because its not provided in the event
	err = oauth2cfg.RevokeToken()
	if err != nil {
		logger.Error().Err(err).Msgf("Failed to revoke token for user %s", oauth2cfg.User)
		return err
	}

	return nil
}
