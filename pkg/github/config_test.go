package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	c := NewConfig()
	assert.Equal(t, "api.github.com", c.GitHubAPIHost)
	assert.Equal(t, "https", c.GitHubAPIScheme)
}

func TestNewRepositoryConfig(t *testing.T) {
	t.Parallel()

	c := NewRepositoryConfig("containifyci", "test")
	assert.Equal(t, "api.github.com", c.GitHubAPIHost)
	assert.Equal(t, "https", c.GitHubAPIScheme)
	assert.Equal(t, "containifyci", c.GitHubOwner)
	assert.Equal(t, "test", c.GitHubRepository)
}

func TestNewStaticTokenConfig(t *testing.T) {
	t.Parallel()

	c := NewStaticTokenConfig("token")
	assert.Equal(t, "api.github.com", c.GitHubAPIHost)
	assert.Equal(t, "https", c.GitHubAPIScheme)
	assert.Equal(t, "token", c.GitHubToken)
}
