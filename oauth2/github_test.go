package oauth2

import (
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGetConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Github: config.GithubConfig{
			OAuth: config.GithubOAuthConfig{
				ClientID:     "client_id",
				ClientSecret: "client_secret",
				Scopes:       []string{"repo", "user"},
				RedirectURL:  "http://localhost:8080/oauth2/callback",
			},
		},
	}
	config := GetConfig(&cfg)
	assert.Equal(t, "client_id", config.ClientID)
	assert.Equal(t, "client_secret", config.ClientSecret)
	assert.Equal(t, "http://localhost:8080/oauth2/callback", config.RedirectURL)
	assert.Equal(t, []string{"repo", "user"}, config.Scopes)
}
