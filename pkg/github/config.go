package github

// Config is the main configuration for the github action watcher
type Config struct {
	GitHubToken      string `required:"true" envconfig:"GITHUB_TOKEN"`
	GitHubAPIHost    string `default:"api.github.com" envconfig:"GITHUB_API_HOST"`
	GitHubAPIScheme  string `default:"https" envconfig:"GITHUB_API_SCHEME"`
	GitHubOwner      string `default:"containifyci" envconfig:"GitHubOwner"`
	GitHubRepository string `required:"true" envconfig:"GITHUB_REPOSITORY"`
	GitHubAuthor     string `default:"dependabot[bot]" envconfig:"GITHUB_AUTHOR"`
}

func NewConfig() Config {
	return Config{
		GitHubAPIHost:   "api.github.com",
		GitHubAPIScheme: "https",
	}
}

func NewRepositoryConfig(owner, repo string) Config {
	return Config{
		GitHubAPIHost:    "api.github.com",
		GitHubAPIScheme:  "https",
		GitHubOwner:      owner,
		GitHubRepository: repo,
	}
}

// NewConfig creates a new config with default values
func NewStaticTokenConfig(token string) Config {
	cfg := NewConfig()
	cfg.GitHubToken = token
	return cfg
}
