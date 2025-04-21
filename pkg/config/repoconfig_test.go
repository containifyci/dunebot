package config

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containifyci/dunebot/pkg/config/testdata"
	github_testdata "github.com/containifyci/dunebot/pkg/github/testdata"

	"github.com/containifyci/dunebot/pkg/github"

	"github.com/stretchr/testify/assert"
)

func TestApproveOnlyRepoConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{
			name: "Approve-only valid config",
			file: "valid_config_approve_only.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cnt, err := testdata.Data.ReadFile(tt.file)
			assert.NoError(t, err)

			cfg, _, err := Unmarshal(cnt)
			assert.NoError(t, err)

			assert.Equal(t, "1.0.1", cfg.Version)
			assert.Equal(t, "fr12k", cfg.Approve.Approver)

			assert.Equal(t, StringArray{"ci/circleci: ete-tests"}, cfg.Approve.RequiredStatuses)
			assert.Equal(t, StringArray{"approve"}, cfg.Approve.Include.Labels)
			assert.Equal(t, StringArray{"dune"}, cfg.Approve.Include.Authors)
			assert.Equal(t, []*Comment{{Title: "## terraform plan for changed", Body: "No changes. Your infrastructure matches the configuration."}}, cfg.Approve.Include.Comments)

			devBranch := cfg.Approve.Include.Branches["develop"]
			assert.Equal(t, "develop", devBranch.Prefix)
			assert.Equal(t, StringArray{"ci/circleci: ete-tests"}, devBranch.RequiredStatuses)
			assert.Equal(t, StringArray{"approve"}, devBranch.Labels)
			assert.Equal(t, []*Comment{{Title: "Merge this Pull Request", Body: "Safe to merge"}}, devBranch.Comments)
			assert.Nil(t, devBranch.Age)

			dependabot := cfg.Approve.Include.Branches["dependabot"]
			assert.Equal(t, "dependabot", dependabot.Prefix)
			assert.Equal(t, StringArray{"check"}, dependabot.RequiredStatuses)
			assert.Equal(t, StringArray{"dependencies"}, dependabot.Labels)
			assert.Equal(t, []*Comment{{Title: "## terraform plan for changed", Body: "No changes. Your infrastructure matches the configuration."}}, dependabot.Comments)
			assert.NotNil(t, dependabot.Age)
			assert.Equal(t, 2*time.Hour+30*time.Minute, dependabot.Age.Duration)
		})
	}
}

func TestLoadingMergeConfig(t *testing.T) {
	// Loading config file that has _config: merge
	// Testing if it's overwriting global config properly
	cfg, _, err := LoadConfig(context.Background(), makeTestClient(), "test", "merge-file", "develop")
	assert.NoError(t, err)

	assert.Equal(t, "1.0.2", cfg.Version)
	assert.Equal(t, "frank", cfg.Approve.Approver)
	assert.Equal(t, StringArray{"ci/circleci: ete-tests"}, cfg.Approve.RequiredStatuses)
	assert.Equal(t, 2, len(cfg.Approve.Include.Branches))
}

func TestLoadingOriginalConfig(t *testing.T) {
	t.Parallel()

	// Loading config file that has _config: original
	cfg, _, err := LoadConfig(context.Background(), makeTestClient(), "test", "original-file", "develop")
	assert.NoError(t, err)

	assert.Equal(t, "1.0.1", cfg.Version)
	assert.Equal(t, "fr12k", cfg.Approve.Approver)
}

func TestLoadingOverrideConfig(t *testing.T) {
	t.Parallel()

	// Loading config file that has _config: override
	cfg, _, err := LoadConfig(context.Background(), makeTestClient(), "test", "override-file", "develop")
	assert.NoError(t, err)

	assert.Equal(t, "1.0.2", cfg.Version)
	assert.Equal(t, "frank", cfg.Approve.Approver)
}

func TestCompleteRepoConfig_Unmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
	}{
		{
			name: "Complete valid config",
			file: "valid_config.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cnt, err := testdata.Data.ReadFile(tt.file)
			assert.NoError(t, err)

			cfg, _, err := Unmarshal(cnt)
			assert.NoError(t, err)

			// Testing approve part
			assert.Equal(t, "1.0.1", cfg.Version)
			assert.Equal(t, "fr12k", cfg.Approve.Approver)
			assert.Equal(t, StringArray{"ci/circleci: ete-tests"}, cfg.Approve.RequiredStatuses)
			assert.Equal(t, StringArray{"approve"}, cfg.Approve.Include.Labels)
			assert.Equal(t, StringArray{"dune"}, cfg.Approve.Include.Authors)
			assert.Equal(t, []*Comment{
				{Title: "## terraform plan for changed", Body: "No changes. Your infrastructure matches the configuration."},
			}, cfg.Approve.Include.Comments)

			// Testing merge part
			assert.Equal(t, "merge", cfg.GetMerge().Method)
			assert.Equal(t, StringArray{"approve"}, cfg.GetMerge().Include.Labels)
			assert.Equal(t, StringArray{"dune"}, cfg.GetMerge().Include.Authors)
			assert.Equal(t, []*Comment{
				{Title: "## terraform plan for changed", Body: "No changes. Your infrastructure matches the configuration."},
			}, cfg.GetMerge().Include.Comments)

			// Testing specific branches in merge part
			developBranch := cfg.GetMerge().Include.Branches["develop"]
			assert.Equal(t, "develop", developBranch.Prefix)
			assert.Equal(t, StringArray{"ci/circleci: ete-tests"}, developBranch.RequiredStatuses)
			assert.Equal(t, StringArray{"approve"}, developBranch.Labels)
			assert.Equal(t, []*Comment{
				{Title: "Merge this Pull Request", Body: "Safe to merge"},
			}, developBranch.Comments)

			dependabotBranch := cfg.GetMerge().Include.Branches["dependabot"]
			assert.Equal(t, "dependabot", dependabotBranch.Prefix)
			assert.Equal(t, StringArray{"check"}, dependabotBranch.RequiredStatuses)
			assert.Equal(t, StringArray{"dependencies"}, dependabotBranch.Labels)
			assert.Equal(t, []*Comment{
				{Title: "## terraform plan for changed", Body: "No changes. Your infrastructure matches the configuration."},
			}, dependabotBranch.Comments)
		})
	}
}

func TestApprover(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *AppConfig
		expected string
	}{
		{
			name:     "Approve is nil",
			config:   &AppConfig{},
			expected: "",
		},
		{
			name: "Approve is not nil",
			config: &AppConfig{
				Approve: &Approve{
					Approver: "testuser",
				},
			},
			expected: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.config.Approver()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNoApprove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		approve *Approve
		expect  bool
	}{
		{approve: nil, expect: true},
		{approve: &Approve{}, expect: true},
		{approve: &Approve{Include: &Include{}}, expect: false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("NoApprove is %t", tt.expect), func(t *testing.T) {
			cfg := AppConfig{Approve: tt.approve}
			assert.Equal(t, tt.expect, cfg.NoApprove())
		})
	}
}

func TestNoMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		merge  *Merge
		expect bool
	}{
		{merge: nil, expect: true},
		{merge: &Merge{}, expect: true},
		{merge: &Merge{Include: &Include{}}, expect: false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("NoApprove is %t", tt.expect), func(t *testing.T) {
			cfg := AppConfig{Merge: tt.merge}
			assert.Equal(t, tt.expect, cfg.NoMerge())
		})
	}
}

func TestApproveInclude(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{Approve: &Approve{}}
	assert.NotNil(t, cfg.Approve.GetInclude())
	cfg = AppConfig{Approve: &Approve{Include: &Include{Authors: StringArray{"dune"}}}}
	assert.True(t, cfg.Approve.GetInclude().Authors.Enabled())
}

func TestMergeInclude(t *testing.T) {
	t.Parallel()

	cfg := AppConfig{Merge: &Merge{}}
	assert.NotNil(t, cfg.Merge.GetInclude())
	cfg = AppConfig{Merge: &Merge{Include: &Include{Authors: StringArray{"dune"}}}}
	assert.True(t, cfg.Merge.GetInclude().Authors.Enabled())
}

func makeTestClient() *github.Client {
	rp := github_testdata.NewResponsePlayer("testdata")
	return github.NewClient(github.WithHttpClient(&http.Client{Transport: rp})).Client
}

func TestLoadLocalConfig(t *testing.T) {
	t.Parallel()

	fs, err := testdata.Data.Open("valid_config.yaml")

	assert.NoError(t, err)
	stat, err := fs.Stat()
	assert.NoError(t, err)

	// TODO: use dynamic reference for testdata
	cfg, err := loadLocalConfig([]string{"non_existing_file.yaml", filepath.Join("testdata", stat.Name())})

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestLoadConfigFromTestData(t *testing.T) {
	testDataPath, err := filepath.Abs("./testdata")
	if err != nil {
		t.Fatalf("Failed to resolve test data path: %v", err)
	}
	t.Setenv("LOCAL_CONFIG_PATH", testDataPath)
	defer os.Clearenv() // Clean up after this test
	appConfig, _, err := LoadConfig(context.Background(), nil, "owner", "repo", "ref")

	assert.NoError(t, err)
	assert.NotNil(t, appConfig)

	assert.Equal(t, "containifyci-bot", appConfig.Approve.Approver)
}

func TestStringArrayContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		arr      StringArray
		contains string
		expected bool
	}{
		{
			name:     "Empty array",
			arr:      StringArray{},
			contains: "test",
			expected: false,
		},
		{
			name:     "Array contains",
			arr:      StringArray{"test"},
			contains: "test",
			expected: true,
		},
		{
			name:     "Array does not contain",
			arr:      StringArray{"test"},
			contains: "test2",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.arr.Contains(tt.contains)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
