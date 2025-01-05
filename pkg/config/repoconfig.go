package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/rs/zerolog/log"

	"github.com/containifyci/dunebot/pkg/github"
	"github.com/palantir/go-githubapp/appconfig"
	"gopkg.in/yaml.v3"
)

type Comment struct {
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
}

type Branch struct {
	Key              string
	Prefix           string      `yaml:"prefix"`
	RequiredStatuses StringArray `yaml:"required_statuses,omitempty"`
	Authors          StringArray `yaml:"authors,omitempty"`
	Age              *Duration   `yaml:"age,omitempty"`
	Labels           StringArray `yaml:"labels,omitempty"`
	Comments         []*Comment  `yaml:"comments,omitempty"`
	Strict           bool        `yaml:"strict,omitempty"`
}

type Include struct {
	Labels   StringArray `yaml:"labels,omitempty"`
	Authors  StringArray `yaml:"authors,omitempty"`
	Comments []*Comment  `yaml:"comments,omitempty"`
	Branches BranchMap   `yaml:"branches,omitempty"`
}

type BranchMap map[string]*Branch

type Approve struct {
	Approver         string      `yaml:"approver,omitempty"`
	Exclude          *Include    `yaml:"exclude,omitempty"`
	Include          *Include    `yaml:"include,omitempty"`
	RequiredStatuses StringArray `yaml:"required_statuses,omitempty"`
}

type Merge struct {
	Method           string      `yaml:"method,omitempty"`
	Exclude          *Include    `yaml:"exclude,omitempty"`
	Include          *Include    `yaml:"include,omitempty"`
	RequiredStatuses StringArray `yaml:"required_statuses,omitempty"`
}

type Error struct {
	EnablePrComment string `yaml:"level,omitempty"`
}

type AppConfig struct {
	Version string `yaml:"version"`

	GlobalConfig string   `yaml:"_config,omitempty"`
	Approve      *Approve `yaml:"approve,omitempty"`
	Merge        *Merge   `yaml:"merge,omitempty"`
	Error        *Error   `yaml:"error,omitempty"`
}

func (a *AppConfig) Approver() string {
	if a.Approve == nil {
		return ""
	}
	return a.Approve.Approver
}

func (a *AppConfig) NoApprove() bool {
	return a.Approve == nil || a.Approve.Include == nil
}

func (a *Approve) GetInclude() *Include {
	if a.Include == nil {
		return &Include{}
	}
	return a.Include
}

func (a *AppConfig) NoMerge() bool {
	return a.Merge == nil || a.Merge.Include == nil
}

func (a *Merge) GetInclude() *Include {
	if a.Include == nil {
		return &Include{}
	}
	return a.Include
}

func (a *AppConfig) GetMerge() *Merge {
	if a.Merge == nil {
		return &Merge{Method: "merge"}
	}
	if a.Merge.Method == "" {
		a.Merge.Method = "merge"
	}
	return a.Merge
}

type StringArray []string

func (a StringArray) Enabled() bool {
	return len(a) > 0
}

func (a StringArray) Contains(s string) bool {
	return slices.Contains(a, s)
}

func LoadConfig(ctx context.Context, client *github.Client, owner, repo, ref string) (*AppConfig, []byte, error) {
	// Get the path for local configuration from the environment variable
	localConfigPath := os.Getenv("LOCAL_CONFIG_PATH")
	log.Debug().Msgf("LOCAL_CONFIG_PATH: %s", localConfigPath)
	if localConfigPath != "" {
		// If LOCAL_CONFIG_PATH is set, load configuration from the specified local path
		paths := []string{
			filepath.Join(localConfigPath, "dunebot-example.yml"),
			filepath.Join(localConfigPath, "dunebot.yml"),
		}
		content, err := loadLocalConfig(paths)
		log.Debug().Msgf("Loading local config from %s", localConfigPath)
		if err != nil {
			return nil, nil, err
		}
		return Unmarshal(content)
	}

	// Code for loading from GitHub remains unchanged
	paths := []string{".github/dunebot.yml", ".github/dunebot.yaml"}
	loader := appconfig.NewLoader(paths, appconfig.WithOwnerDefault(".github", paths))

	c, err := loader.LoadConfig(ctx, client, owner, repo, ref)
	if err != nil {
		return nil, nil, err
	}
	if c.IsUndefined() {
		return nil, nil, nil
	}

	cfg, _c, err := Unmarshal(c.Content)

	if err != nil {
		return nil, nil, err
	}

	//TODO return also the source of the config its stored in the
	//c and c2 variable c.Source and c2.Source
	switch cfg.GlobalConfig {
	case "original":
		c2, err := loader.LoadConfig(ctx, client, owner, ".github", ref)
		if err != nil {
			return nil, nil, err
		}
		if c.IsUndefined() {
			return nil, nil, nil
		}
		return Unmarshal(c2.Content)
	case "merge":
		c2, err := loader.LoadConfig(ctx, client, owner, ".github", ref)
		if err != nil {
			return nil, nil, err
		}
		if c.IsUndefined() {
			return nil, nil, nil
		}
		cnt, err := merge(c2.Content, c.Content)
		if err != nil {
			return nil, nil, err
		}
		return Unmarshal(cnt)
	case "override":
		return cfg, _c, err
	default:
		return cfg, _c, err
	}
}

func loadLocalConfig(paths []string) ([]byte, error) {
	// Assuming the local file is present in one of the paths
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}

		// Check if file exists
		if _, err := os.Stat(absPath); err == nil {
			// File exists, read its content
			content, err := os.ReadFile(absPath)
			if err != nil {
				return nil, err
			}
			return content, nil
		}
	}

	return nil, fmt.Errorf("no local config file found")
}

func Unmarshal(b []byte) (*AppConfig, []byte, error) {
	var appConfig AppConfig
	if err := yaml.Unmarshal(b, &appConfig); err != nil {
		return nil, nil, err
	}
	return &appConfig, b, nil
}
