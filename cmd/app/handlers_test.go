package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"

	"github.com/containifyci/dunebot/pkg/github/testdata"
)

// TODO add unit test to check Authentication and error handlin
func TestDuneBotRepositoriesHandler(t *testing.T) {
	t.Parallel()
	privateKey, err := createPrivaseRSAKey()
	assert.NoError(t, err)

	// Create a new request
	req, err := http.NewRequest("GET", "/repositories", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer top_secret")

	// Create a new response recorder
	rr := httptest.NewRecorder()

	cfg := config.Config{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 1,
				PrivateKey:    privateKey,
			},
		},
		AppConfig: config.ApplicationConfig{
			UserAgent:      "test",
			InstallationId: 43975733,
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

	handlerOpts := HandlerOptions{
		cfg:           &cfg,
		clientOpts:    []github.Option{github.WithGithubClient(makeTestGithubClient(makeTestClient()))},
		ClientCreator: cc,
	}

	// Create a new handler
	handler := http.HandlerFunc(DuneBotRepositoriesHandler(handlerOpts))

	// Serve the HTTP request to the response recorder
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "[\"palantir/test-repo-1\",\"test-owner/test-repo-2\"]\n", rr.Body.String())
}

func TestDuneBotRepositoriesHandlerAuthentication(t *testing.T) {
	t.Parallel()
	privateKey, err := createPrivaseRSAKey()
	assert.NoError(t, err)

	// Create a new request
	req, err := http.NewRequest("GET", "/repositories", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer top_secret")

	// Create a new response recorder
	rr := httptest.NewRecorder()

	cfg := config.Config{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 1,
				PrivateKey:    privateKey,
			},
		},
		AppConfig: config.ApplicationConfig{
			UserAgent:      "test",
			InstallationId: 43975733,
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

	rp := makeTestClient()
	rp.ReplaceRule("/user/orgs", "", filepath.Join("../testdata", "remote_user_orgs_non_containifyci.yaml"))

	handlerOpts := HandlerOptions{
		cfg:           &cfg,
		clientOpts:    []github.Option{github.WithGithubClient(makeTestGithubClient(rp))},
		ClientCreator: cc,
	}

	// Create a new handler
	handler := http.HandlerFunc(DuneBotRepositoriesHandler(handlerOpts))

	// Serve the HTTP request to the response recorder
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusUnauthorized)
	assert.Equal(t, rr.Body.String(), "User is not a member of the organisation\n")

	req.Header.Del("Authorization")
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusUnauthorized)
	assert.Equal(t, rr.Body.String(), "Missing Authorization header bearer token\n")
}

func makeTestClient() *testdata.ResponsePlayer {
	rp := testdata.NewResponsePlayer("../testdata")
	return rp
}

func makeTestGithubClient(rp *testdata.ResponsePlayer) *github.Client {
	return github.NewGithubClient(&http.Client{Transport: rp})
}

func createPrivaseRSAKey() (string, error) {
	// Generate a new RSA private key with 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)
	return string(privateKeyPEMBytes), nil
}
