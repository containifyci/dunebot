package config

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/containifyci/dunebot/pkg/config/testdata"
)

// TestGetConfigWithValidPath tests the GetConfig function with a valid file path.
func TestGetConfigWithValidPath(t *testing.T) {
	t.Parallel()

	cnt, err := testdata.Data.ReadFile("valid_appconfig.yaml")
	assert.NoError(t, err, "Unexpected error in reading test config file")

	// Mock a valid config file path
	configFilePath := createTempConfigFile(cnt)
	defer os.Remove(configFilePath)

	// Call the GetConfig function
	config, err := GetConfig(configFilePath)

	// Assertions
	assert.NoError(t, err, "Unexpected error in GetConfig")
	assert.NotNil(t, config, "Config should not be nil")
	assert.Equal(t, "localhost", config.Server.Address, "Unexpected server address")
	assert.Equal(t, "/dunebot", config.Server.BasePath, "Unexpected base path")

	assert.Equal(t, 8080, config.Server.Port, "Unexpected server port")
	assert.Equal(t, "https://github.com", config.Github.WebURL, "Unexpected GitHub web URL")

	//AppConfig
	assert.Equal(t, "your-user-agent", config.AppConfig.UserAgent, "Unexpected UserAgent")
	assert.Equal(t, "5s", config.AppConfig.ClientTimeOut, "Unexpected ClientTimeout")
	assert.Equal(t, int64(123), config.AppConfig.InstallationId, "Unexpected Installation Id")
	// Add more assertions based on your specific requirements
}

// TestGetConfigWithInvalidPath tests the GetConfig function with an invalid file path.
func TestGetConfigWithInvalidPath(t *testing.T) {
	t.Parallel()
	// Mock an invalid config file path
	invalidConfigFilePath := "invalid/path/to/config.yaml"

	// Call the GetConfig function
	config, err := GetConfig(invalidConfigFilePath)

	// Assertions
	assert.Error(t, err, "Expected an error in GetConfig")
	assert.Nil(t, config, "Config should be nil")
}

// TestReadConfigWithInvalidContent tests the ReadConfig function with invalid YAML content.
func TestReadConfigWithInvalidContent(t *testing.T) {
	t.Parallel()
	// Mock invalid YAML content
	invalidConfigContent := []byte("invalid_yaml_content")

	// Create a temporary file to write the invalid config content
	tmpFile, err := os.CreateTemp("", "invalid-config-test-*.yaml")
	assert.NoError(t, err, "Error creating temporary file")
	defer os.Remove(tmpFile.Name())

	// Write the invalid config content to the temporary file
	_, err = tmpFile.Write(invalidConfigContent)
	assert.NoError(t, err, "Error writing to temporary file")

	// Call the ReadConfig function with the temporary file path
	config, err := ReadConfig(tmpFile.Name())

	// Assertions
	assert.Error(t, err, "Expected an error in ReadConfig")
	assert.Nil(t, config, "Config should be nil")
}

// TestToGithubAppConfig tests the ToGithubAppConfig function.
func TestToGithubAppConfig(t *testing.T) {
	// Mock a GithubConfig
	githubConfig := GithubConfig{
		WebURL:   "https://github.com",
		V3APIURL: "https://api.github.com",
		V4APIURL: "https://api.github.com/graphql",
		App: struct {
			IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
			WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
			PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
		}{
			IntegrationID: 123,
			WebhookSecret: "your-webhook-secret",
			PrivateKey:    "your-private-key",
		},
		OAuth: struct {
			ClientID     string   `yaml:"client_id" json:"clientId" envconfig:"CLIENT_ID"`
			ClientSecret string   `yaml:"client_secret" json:"clientSecret" envconfig:"CLIENT_SECRET"`
			Scopes       []string `yaml:"scopes" json:"scopes" envconfig:"SCOPES"`
			RedirectURL  string   `yaml:"redirect_url" json:"redirectUrl" envconfig:"REDIRECT_URL"`
		}{
			ClientID:     "your-client-id",
			ClientSecret: "your-client-secret",
			Scopes:       []string{"scope1", "scope2"},
			RedirectURL:  "https://redirect.url",
		},
	}

	// Call the ToGithubAppConfig function
	githubAppConfig := githubConfig.ToGithubAppConfig()

	// Assertions
	assert.Equal(t, "https://github.com", githubAppConfig.WebURL, "Unexpected WebURL")
	assert.Equal(t, "https://api.github.com", githubAppConfig.V3APIURL, "Unexpected V3APIURL")
	assert.Equal(t, "https://api.github.com/graphql", githubAppConfig.V4APIURL, "Unexpected V4APIURL")
	assert.Equal(t, int64(123), githubAppConfig.App.IntegrationID, "Unexpected IntegrationID")
	assert.Equal(t, "your-webhook-secret", githubAppConfig.App.WebhookSecret, "Unexpected WebhookSecret")
	assert.Equal(t, "your-private-key", githubAppConfig.App.PrivateKey, "Unexpected PrivateKey")
	assert.Equal(t, "your-client-id", githubAppConfig.OAuth.ClientID, "Unexpected ClientID")
	assert.Equal(t, "your-client-secret", githubAppConfig.OAuth.ClientSecret, "Unexpected ClientSecret")
}

func TestClientTimeOutDuration(t *testing.T) {
	t.Parallel()
	// Mock a MyApplicationConfig
	appConfig := ApplicationConfig{
		ClientTimeOut: "10adad",
	}

	// Call the ClientTimeOutDuration function
	clientTimeOutDuration := appConfig.ClientTimeOutDuration()

	// Assertions
	assert.Equal(t, "5s", clientTimeOutDuration.String(), "Unexpected ClientTimeOutDuration")
}

// TestLoad tests the Load function.
func TestLoad(t *testing.T) {
	t.Parallel()
	// Mock environment variables
	setEnv(t, "DUNEBOT_APP_CONFIGURATION_CLIENT_TIMEOUT", "5s")
	setEnv(t, "DUNEBOT_APP_CONFIGURATION_USER_AGENT", "DuneBot/1.0.0")
	setEnv(t, "DUNEBOT_APP_CONFIGURATION_INSTALLATION_ID", "123")

	setEnv(t, "DUNEBOT_GITHUB_APP_INTEGRATION_ID", "123")
	setEnv(t, "DUNEBOT_GITHUB_APP_WEBHOOK_SECRET", "your-webhook-secret")
	setEnv(t, "DUNEBOT_GITHUB_APP_PRIVATE_KEY", base64.StdEncoding.EncodeToString([]byte("your-private-key")))
	setEnv(t, "DUNEBOT_GITHUB_OAUTH_CLIENT_ID", "your-client-id")
	setEnv(t, "DUNEBOT_GITHUB_OAUTH_CLIENT_SECRET", "your-client-secret")
	setEnv(t, "DUNEBOT_GITHUB_OAUTH_SCOPES", "scope1,scope2")
	setEnv(t, "DUNEBOT_GITHUB_OAUTH_REDIRECT_URL", "https://redirect.url")
	setEnv(t, "DUNEBOT_GITHUB_V3_API_URL", "https://api.github.com")
	setEnv(t, "DUNEBOT_GITHUB_V4_API_URL", "https://api.github.com/graphql")
	setEnv(t, "DUNEBOT_GITHUB_WEB_URL", "https://github.com")

	setEnv(t, "DUNEBOT_JWT_PRIVATE_KEY", "ecdsa private key for JWT signing base64 encoded")
	setEnv(t, "DUNEBOT_JWT_PUBLIC_KEY", "ecdsa public key for JWT verification base64 encoded")
	setEnv(t, "DUNEBOT_JWT_SERVER_ADDRESS", "token:50051")

	setEnv(t, "DUNEBOT_SERVER_ADDRESS", "0.0.0.0")
	setEnv(t, "DUNEBOT_SERVER_BASE_PATH", "/dunebot")
	setEnv(t, "DUNEBOT_SERVER_PORT", "80")

	// Call the Load function
	config, err := GetConfig("")

	// Assertions
	assert.NoError(t, err, "Unexpected error in Load")
	assert.NotNil(t, config, "Config should not be nil")
	assert.Equal(t, "5s", config.AppConfig.ClientTimeOutDuration().String(), "Unexpected ClientTimeOutDuration")
	assert.Equal(t, "DuneBot/1.0.0", config.AppConfig.UserAgent, "Unexpected UserAgent")
	assert.Equal(t, int64(123), config.AppConfig.InstallationId, "Unexpected InstallationId")

	assert.Equal(t, int64(123), config.Github.App.IntegrationID, "Unexpected IntegrationID")
	assert.Equal(t, "your-webhook-secret", config.Github.App.WebhookSecret, "Unexpected WebhookSecret")
	assert.Equal(t, "your-private-key", config.Github.App.PrivateKey, "Unexpected PrivateKey")
	assert.Equal(t, "your-client-id", config.Github.OAuth.ClientID, "Unexpected ClientID")
	assert.Equal(t, "your-client-secret", config.Github.OAuth.ClientSecret, "Unexpected ClientSecret")
	assert.Equal(t, []string{"scope1", "scope2"}, config.Github.OAuth.Scopes, "Unexpected Scopes")
	assert.Equal(t, "https://redirect.url", config.Github.OAuth.RedirectURL, "Unexpected RedirectURL")
	assert.Equal(t, "https://api.github.com", config.Github.V3APIURL, "Unexpected V3APIURL")
	assert.Equal(t, "https://api.github.com/graphql", config.Github.V4APIURL, "Unexpected V4APIURL")
	assert.Equal(t, "https://github.com", config.Github.WebURL, "Unexpected WebURL")

	assert.Equal(t, "token:50051", config.JWT.Address, "Unexpected Address")
	assert.Equal(t, "ecdsa private key for JWT signing base64 encoded", config.JWT.PrivateKey, "Unexpected PrivateKey")
	assert.Equal(t, "ecdsa public key for JWT verification base64 encoded", config.JWT.PublicKey, "Unexpected PublicKey")
	assert.Equal(t, "0.0.0.0", config.Server.Address, "Unexpected Server Address")
	assert.Equal(t, "/dunebot", config.Server.BasePath, "Unexpected Server Base Path")

	assert.Equal(t, 80, config.Server.Port, "Unexpected Server Port")
}

func TestConfigTransformer(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Github: GithubConfig{
			OAuth: GithubOAuthConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Scopes:       []string{"scope1", "scope2"},
				RedirectURL:  "https://redirect.url",
			},
		},
	}

	oauth2Cfg := cfg.ToOAuth2Config()
	assert.NotNil(t, cfg)
	assert.Equal(t, "client-id", oauth2Cfg.ClientID, "Unexpected ClientID")
	assert.Equal(t, "client-secret", oauth2Cfg.ClientSecret, "Unexpected ClientSecret")
	assert.Equal(t, []string{"scope1", "scope2"}, oauth2Cfg.Scopes, "Unexpected Scopes")
	assert.Equal(t, "https://redirect.url", oauth2Cfg.RedirectURL, "Unexpected RedirectURL")

	assert.Equal(t, oauth2Cfg.Endpoint.AuthURL, oauth2Cfg.Endpoint.AuthURL, "Unexpected AuthURL")
	assert.Equal(t, oauth2Cfg.Endpoint.TokenURL, oauth2Cfg.Endpoint.TokenURL, "Unexpected TokenURL")
	assert.Equal(t, oauth2Cfg.Endpoint.DeviceAuthURL, oauth2Cfg.Endpoint.DeviceAuthURL, "Unexpected DeviceAuthURL")
}

// test utility

// Additional tests can be added based on specific functions and requirements.

func setEnv(t *testing.T, key, value string) {
	err := os.Setenv(key, value)
	assert.NoError(t, err, "Error setting environment variable")
}

// Helper function to create a temporary config file with given content.
func createTempConfigFile(content []byte) string {
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	if err != nil {
		panic(err)
	}
	defer tmpFile.Close()

	_, err = tmpFile.Write(content)
	if err != nil {
		panic(err)
	}

	return tmpFile.Name()
}
