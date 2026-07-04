package dispatch

import (
	"encoding/base64"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/containifyci/dunebot/pkg/config"
)

type dispatchConfig struct {
	Github config.GithubConfig `yaml:"github" json:"github" envconfig:"GITHUB"`

	GithubToken string `yaml:"token" json:"token" envconfig:"GITHUB_TOKEN"`

	RepositoryEndpoint string `yaml:"repository_endpoint" json:"repository_endpoint" envconfig:"REPOSITORY_ENDPOINT"`
}

func (c *dispatchConfig) IsAppMode() bool {
	return c.Github.App.IntegrationID != 0
}

func (c *dispatchConfig) GetPrivateKey() (string, error) {
	if c.Github.App.PrivateKey != "" {
		return c.Github.App.PrivateKey, nil
	}
	return "", errors.New("private key is not configured")
}

func LoadConfig() (*dispatchConfig, error) {
	var cfg dispatchConfig
	err := envconfig.Process("DUNEBOT", &cfg)
	if err != nil {
		return nil, err
	}

	// Decode base64 private key from AppPrivateKeyBase64 into Github.App.PrivateKey
	if cfg.Github.App.PrivateKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(cfg.Github.App.PrivateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64 private key")
		}
		cfg.Github.App.PrivateKey = string(decoded)
	}

	if cfg.GithubToken == "" && !cfg.IsAppMode() {
		return nil, errors.New("GITHUB_TOKEN is required when not using GitHub App authentication (GITHUB_APP_INTEGRATION_ID)")
	}

	return &cfg, nil
}
