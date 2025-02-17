package dispatch

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadRequiredConfig(t *testing.T) {
	t.Setenv("DUNEBOT_GITHUB_TOKEN", "top_secret")
	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.Equal(t, "top_secret", cfg.GithubToken)
}

func TestLoadConfigError(t *testing.T) {
	os.Unsetenv("DUNEBOT_GITHUB_TOKEN")
	fmt.Printf("DUNEBOT_GITHUB_TOKEN '%s'", os.Getenv("DUNEBOT_GITHUB_TOKEN"))

	_, err := LoadConfig()
	assert.ErrorContains(t, err, "required key GITHUB_TOKEN missing value")
}
