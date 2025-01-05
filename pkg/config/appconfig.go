package config

import (
	"encoding/base64"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"
	"gopkg.in/yaml.v2"

	"github.com/kelseyhightower/envconfig"

	"github.com/palantir/go-githubapp/githubapp"
)

type GithubOAuthConfig struct {
	ClientID     string   `yaml:"client_id" json:"clientId" envconfig:"CLIENT_ID"`
	ClientSecret string   `yaml:"client_secret" json:"clientSecret" envconfig:"CLIENT_SECRET"`
	Scopes       []string `yaml:"scopes" json:"scopes" envconfig:"SCOPES"`
	RedirectURL  string   `yaml:"redirect_url" json:"redirectUrl" envconfig:"REDIRECT_URL"`
}

// TODO setup proper default values for envconfig
type GithubConfig struct {
	WebURL   string `yaml:"web_url" json:"webUrl" envconfig:"WEB_URL"`
	V3APIURL string `yaml:"v3_api_url" json:"v3ApiUrl" envconfig:"V3_API_URL"`
	V4APIURL string `yaml:"v4_api_url" json:"v4ApiUrl" envconfig:"V4_API_URL"`

	App struct {
		IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
		WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
		PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
	} `yaml:"app" json:"app" envconfig:"APP"`

	OAuth GithubOAuthConfig `yaml:"oauth" json:"oauth" envconfig:"OAUTH"`
}

type HTTPConfig struct {
	Address  string `yaml:"address" json:"address" envconfig:"ADDRESS"`
	BasePath string `yaml:"base_path" json:"basePath" envconfig:"BASE_PATH"`
	Port     int    `yaml:"port" json:"port" envconfig:"PORT"`
}

type JWTConfig struct {
	PrivateKey string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
	PublicKey  string `yaml:"public_key" json:"publicKey" envconfig:"PUBLIC_KEY"`
	Address    string `yaml:"address" json:"address" envconfig:"SERVER_ADDRESS"`
}

type ApplicationConfig struct {
	UserAgent      string         `yaml:"user_agent" json:"userAgent" envconfig:"USER_AGENT"`
	ClientTimeOut  string         `yaml:"client_timeout" json:"clientTimeout" envconfig:"CLIENT_TIMEOUT"`
	InstallationId int64          `yaml:"installation_id" json:"installationId" envconfig:"INSTALLATION_ID"`
	Debug          bool           `yaml:"debug" json:"debug" envconfig:"DEBUG_LOGGING"`
	ReviewerConfig ReviewerConfig `yaml:"reviewer" json:"reviewer" envconfig:"REVIEWER"`
}

type ReviewerConfig struct {
	Type    string `yaml:"type" json:"type" envconfig:"TYPE"`
	Address string `yaml:"address" json:"address" envconfig:"SERVER_ADDRESS"`
}

type Config struct {
	Server HTTPConfig   `yaml:"server" json:"server" envconfig:"SERVER"`
	Github GithubConfig `yaml:"github" json:"github" envconfig:"GITHUB"`
	JWT    JWTConfig    `yaml:"jwt" json:"jwt" envconfig:"JWT"`

	AppConfig ApplicationConfig `yaml:"app_configuration" json:"app_configuration" envconfig:"APP_CONFIGURATION"`

	ConfigTransformer
}

type ConfigTransformer interface {
	ToOAuth2Config() *oauth2.Config
	Config() *Config
}

func (c *Config) Config() *Config {
	return c
}

func (c *Config) ToOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		RedirectURL:  c.Github.OAuth.RedirectURL,
		ClientSecret: c.Github.OAuth.ClientSecret,
		ClientID:     c.Github.OAuth.ClientID,
		Scopes:       c.Github.OAuth.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:       oauth2github.Endpoint.AuthURL,
			TokenURL:      oauth2github.Endpoint.TokenURL,
			DeviceAuthURL: oauth2github.Endpoint.DeviceAuthURL,
		},
	}
}

func (c *GithubConfig) ToGithubAppConfig() githubapp.Config {
	return githubapp.Config{
		WebURL:   c.WebURL,
		V3APIURL: c.V3APIURL,
		V4APIURL: c.V4APIURL,
		App: struct {
			IntegrationID int64  "yaml:\"integration_id\" json:\"integrationId\""
			WebhookSecret string "yaml:\"webhook_secret\" json:\"webhookSecret\""
			PrivateKey    string "yaml:\"private_key\" json:\"privateKey\""
		}{
			IntegrationID: c.App.IntegrationID,
			WebhookSecret: c.App.WebhookSecret,
			PrivateKey:    c.App.PrivateKey,
		},
		OAuth: struct {
			ClientID     string "yaml:\"client_id\" json:\"clientId\""
			ClientSecret string "yaml:\"client_secret\" json:\"clientSecret\""
		}{
			ClientID:     c.OAuth.ClientID,
			ClientSecret: c.OAuth.ClientSecret,
		},
	}
}

func (myAppCfg ApplicationConfig) ClientTimeOutDuration() time.Duration {
	duration, err := time.ParseDuration(myAppCfg.ClientTimeOut)
	if err != nil {
		//TODO add proper logging
		log.Warn().Err(err).Msgf("error %s parsing client timeout '%s' use default", err, myAppCfg.ClientTimeOut)
		duration = 5 * time.Second
	}
	return duration
}

func GetConfig(path string) (*Config, error) {
	if path != "" {
		return ReadConfig(path)
	}
	return Load()
}

func ReadConfig(path string) (*Config, error) {
	var c Config

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading server config file: %s", path)
	}

	if err := yaml.UnmarshalStrict(bytes, &c); err != nil {
		return nil, errors.Wrap(err, "failed parsing configuration file")
	}

	return &c, nil
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("DUNEBOT", &cfg)
	if err != nil {
		return nil, err
	}
	b, err := base64.StdEncoding.DecodeString(cfg.Github.App.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding Github Private Key, check if it's base64 encoded")
	}
	cfg.Github.App.PrivateKey = string(b)
	return &cfg, err
}

var _ ConfigTransformer = &Config{}
