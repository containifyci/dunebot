package oauth2

import (
	"github.com/containifyci/dunebot/pkg/config"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"
)

func GetConfig(cfg *config.Config) *oauth2.Config {
	return &oauth2.Config{
		RedirectURL:  cfg.Github.OAuth.RedirectURL,
		ClientSecret: cfg.Github.OAuth.ClientSecret,
		ClientID:     cfg.Github.OAuth.ClientID,
		Scopes:       cfg.Github.OAuth.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:       oauth2github.Endpoint.AuthURL,
			TokenURL:      oauth2github.Endpoint.TokenURL,
			DeviceAuthURL: oauth2github.Endpoint.DeviceAuthURL,
		},
	}
}
