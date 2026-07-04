package dispatch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/containifyci/dunebot/pkg/config"
)

func TestLoadRequiredConfig(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_TOKEN", "top_secret")
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.Equal(t, "top_secret", cfg.GithubToken)
}

func TestLoadConfigError(t *testing.T) {
	os.Clearenv()

	_, err := LoadConfig()
	assert.ErrorContains(t, err, "GITHUB_TOKEN is required")
}

func TestLoadConfigWithAppMode(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_APP_INTEGRATION_ID", "12345")
	t.Setenv("DUNEBOT_GITHUB_APP_PRIVATE_KEY", "dGVzdC1rZXk=")
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.IsAppMode())
	assert.Equal(t, int64(12345), cfg.Github.App.IntegrationID)
	assert.Equal(t, "test-key", cfg.Github.App.PrivateKey)
}

func TestLoadConfigWithAppModeBase64(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_APP_INTEGRATION_ID", "12345")
	t.Setenv("DUNEBOT_GITHUB_APP_PRIVATE_KEY", "dGVzdC1rZXk=") // base64 of "test-key"
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.True(t, cfg.IsAppMode())
	pk, err := cfg.GetPrivateKey()
	assert.NoError(t, err)
	assert.Equal(t, "test-key", pk)
}

func TestGetPrivateKeyDirect(t *testing.T) {
	cfg := &dispatchConfig{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				PrivateKey: "direct-key",
			},
		},
	}
	pk, err := cfg.GetPrivateKey()
	assert.NoError(t, err)
	assert.Equal(t, "direct-key", pk)
}

func TestGetPrivateKeyEmpty(t *testing.T) {
	cfg := &dispatchConfig{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{},
		},
	}
	_, err := cfg.GetPrivateKey()
	assert.Error(t, err)
}

func TestLoadConfigBase64Invalid(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_APP_INTEGRATION_ID", "12345")
	t.Setenv("DUNEBOT_GITHUB_APP_PRIVATE_KEY", "!!!invalid!!!")
	_, err := LoadConfig()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "failed to decode base64 private key")
}

func TestIsAppMode(t *testing.T) {
	cfg := &dispatchConfig{}
	assert.False(t, cfg.IsAppMode())

	cfg = &dispatchConfig{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 12345,
			},
		},
	}
	assert.True(t, cfg.IsAppMode())
}
