package oauth2

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockOAuth2Server(t *testing.T) {
	// Create a new mock OAuth2 server
	server := NewMockOAuth2Server()
	defer server.Close()

	assert.NotNil(t, server)
	assert.NotEmpty(t, server.URL)

	// Test the /auth endpoint
	t.Run("Auth Endpoint", func(t *testing.T) {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		resp, err := client.Get(server.Endpoint().AuthURL)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// assert.Equal(t, http.StatusFound, resp.StatusCode)
		assert.Equal(t, "http://localhost:8080/oauth2/callback?code=mockcode", resp.Header.Get("Location"))
	})

	// Test the /token endpoint
	t.Run("Token Endpoint", func(t *testing.T) {
		resp, err := http.PostForm(server.Endpoint().TokenURL, url.Values{
			"client_id":     {"mockclient"},
			"client_secret": {"mocksecret"},
			"code":          {"mockcode"},
			"grant_type":    {"authorization_code"},
		})
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(body), "access_token=mocktoken")
		assert.Contains(t, string(body), "refresh_token=mockrefresh")
		assert.Contains(t, string(body), "token_type=mocktype")
		assert.Contains(t, string(body), "scope=user")
	})

	// Test the /device endpoint
	t.Run("Device Endpoint", func(t *testing.T) {
		resp, err := http.Post(server.Endpoint().DeviceAuthURL, "application/json", strings.NewReader(`{}`))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{
			"device_code": "mockdevicecode",
			"user_code": "mockusercode",
			"verification_uri": "mockverificationuri",
			"interval": 1,
			"expires_in": 600
		}`, string(body))
	})

	// Test the Endpoint() method
	t.Run("Endpoint Method", func(t *testing.T) {
		endpoint := server.Endpoint()
		assert.NotNil(t, endpoint)
		assert.Equal(t, server.URL+"/auth", endpoint.AuthURL)
		assert.Equal(t, server.URL+"/token", endpoint.TokenURL)
		assert.Equal(t, server.URL+"/device", endpoint.DeviceAuthURL)
	})

	// Ensure server shuts down properly
	t.Run("Server Close", func(t *testing.T) {
		server.Close()

		// Wait to ensure server is properly stopped
		time.Sleep(100 * time.Millisecond)
		_, err := http.Get(server.URL + "/auth")
		assert.Error(t, err)
	})
}
