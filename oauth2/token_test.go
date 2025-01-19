package oauth2

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

type DeRef struct {
	Validate *bool
}

func TestDeRef(t *testing.T) {
	t.Parallel()

	deRef := DeRef{}
	assert.Nil(t, deRef.Validate)

	deRef.Validate = new(bool)
	*deRef.Validate = true
}

func TestRetrieveToken(t *testing.T) {
	t.Parallel()

	config := SetupGRPCClient(t, "user", tokens)

	token, err := config.RetrieveToken()
	assert.NoError(t, err)

	assert.NotNil(t, token)
	assert.Equal(t, "access", token.AccessToken)
	assert.Equal(t, "refresh", token.RefreshToken)
	assert.Equal(t, "type", token.TokenType)
}

func TestStoreToken(t *testing.T) {
	t.Parallel()

	config := SetupGRPCClient(t, "user2", tokens)

	token := &oauth2.Token{
		AccessToken:  "new_access",
		TokenType:    "new_type",
		RefreshToken: "new_refresh",
		Expiry:       time.Now(),
	}

	err := config.StoreToken(token)
	assert.NoError(t, err)

	token2, err := config.RetrieveToken()
	assert.NoError(t, err)

	assert.NotNil(t, token)
	assert.Equal(t, token.AccessToken, token2.AccessToken)
	assert.Equal(t, token.RefreshToken, token2.RefreshToken)
	assert.Equal(t, token.TokenType, token2.TokenType)
}

func TestRevokeToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		user           string
		installationId string
		err            error
	}{
		{name: "revoke token", user: "user", installationId: "1"},
		{name: "revoke token no token found", user: "user1", installationId: "1", err: fmt.Errorf("user user1 has no token")},
		{name: "revoke token no installation found", user: "user", installationId: "2", err: fmt.Errorf("requested token for installation 2 not found")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SetupGRPCClient(t, tt.user, tokens)
			config.InstallationId = tt.installationId
			err := config.RevokeToken()
			if tt.err != nil {
				assert.ErrorContains(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenSourceFrom(t *testing.T) {
	config := SetupGRPCClient(t, "user", tokens)

	srv := NewMockOAuth2Server()
	defer srv.Close()
	config.OAuth2Config.Endpoint = srv.Endpoint()

	ctx := context.Background()
	tokenSource := config.TokenSourceFrom(ctx)

	token, err := tokenSource.Token()
	assert.NoError(t, err)

	assert.NotNil(t, token)
	assert.Equal(t, "mocktoken", token.AccessToken)
	assert.Equal(t, "mockrefresh", token.RefreshToken)
	assert.Equal(t, "mocktype", token.TokenType)
}

// utility functions
const tokens = `{"1":{"installation_id":1,"tokens":[{"access_token":"access","refresh_token":"refresh","expiry":{"seconds":1715603314,"nanos":409109000},"token_type":"type","user":"user"}]}}`
