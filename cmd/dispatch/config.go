package dispatch

import "github.com/kelseyhightower/envconfig"

type dispatchConfig struct {
	GithubToken string `yaml:"token" json:"token" envconfig:"GITHUB_TOKEN" required:"true"`

	RepositoryEndpoint string `yaml:"repository_endpoint" json:"repository_endpoint" envconfig:"REPOSITORY_ENDPOINT"`
}

func LoadConfig() (*dispatchConfig, error) {
	var cfg dispatchConfig
	err := envconfig.Process("DUNEBOT", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, err
}
