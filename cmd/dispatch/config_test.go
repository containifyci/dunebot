package dispatch

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_TOKEN", "top_secret")
	t.Setenv("DUNEBOT_REPOSITORY_ENDPOINT", "dunebot_endpoint")
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.Equal(t, "top_secret", cfg.GithubToken)
	assert.Equal(t, "dunebot_endpoint", cfg.RepositoryEndpoint)
}

func TestLoadConfigError(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_TOKEN", "top_secret")
	os.Unsetenv("DUNEBOT_REPOSITORY_ENDPOINT")

	fmt.Printf("DUNEBOT_REPOSITORY_ENDPOINT '%s'", os.Getenv("DUNEBOT_REPOSITORY_ENDPOINT"))

	_, err := LoadConfig()
	assert.ErrorContains(t, err, "required key REPOSITORY_ENDPOINT missing value")
}
