package github

import (
	"strings"
	"testing"
	"time"

	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestClient(t *testing.T) {
	t.Parallel()

	var s = new(strings.Builder)
	logger := logger.ZeroLogger(zerolog.New(s))
	cli := NewClient(WithLogger(logger))

	timestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	assert.NoError(t, err)

	pr := NewPullRequestClient(&cli, &PullRequest{
		User: &User{
			Login: String("test_login"),
		},
		Labels: []*Label{
			{Name: String("test_label")},
		},
		Base: &PullRequestBranch{
			Ref: String("test_base_ref"),
		},
		Head: &PullRequestBranch{
			Ref: String("test_head_ref"),
		},
		CreatedAt: &Timestamp{
			Time: timestamp,
		},
	})
	assert.Equal(t, "test_base_ref", *pr.Base().Ref)
	assert.Equal(t, "test_head_ref", *pr.Head().Ref)
	assert.Equal(t, "test_label", *pr.Labels()[0].Name)
	assert.Equal(t, "test_login", *pr.Owner().Login)
	assert.Equal(t, "2021-01-01 00:00:00 +0000 UTC", pr.CreatedAt().UTC().String())

	//test logger
	pr.Logger().Debug("test")
	assert.Contains(t, s.String(), "test")
}

func TestPullRequestClientComments(t *testing.T) {
	t.Parallel()

	cli := NewClient(WithConfig(NewRepositoryConfig("org_test", "test_repo")), WithGithubClient(makeTestClient()))
	pr := NewPullRequestClient(&cli, &PullRequest{
		Number: Int(1),
	})
	comments, err := pr.Comments()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments))
	assert.Equal(t, "Me too", *comments[0].Body)
}

func TestPullRequestClientChecks(t *testing.T) {
	t.Parallel()

	cli := NewClient(WithConfig(NewRepositoryConfig("org_test", "test_repo")), WithGithubClient(makeTestClient()))
	pr := NewPullRequestClient(&cli, &PullRequest{
		Number: Int(1),
		Head: &PullRequestBranch{
			SHA: String("test_sha"),
		},
	})
	checks, err := pr.Checks()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(checks))
	assert.Equal(t, "continuous-integration/jenkins", checks[0].Name)
	assert.Equal(t, "success", checks[0].State)
	assert.Equal(t, "security/brakeman", checks[1].Name)
	assert.Equal(t, "success", checks[1].State)
	assert.Equal(t, "mighty_readme", checks[2].Name)
	assert.Equal(t, "completed", checks[2].State)
}
