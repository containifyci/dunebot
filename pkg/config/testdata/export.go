package testdata

import (
	"embed"
	"fmt"
	"time"

	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"
	"github.com/rs/zerolog/log"
)

//go:embed *.yaml
var Data embed.FS

type MockErrorPullRequest struct {
	MockPullRequest
}

type MockPullRequest struct {
	LabelsValue    []*github.Label           `yaml:"labels,omitempty"`
	BaseValue      *github.PullRequestBranch `yaml:"base,omitempty"`
	HeadValue      *github.PullRequestBranch `yaml:"head,omitempty"`
	CommentsValue  []*github.IssueComment    `yaml:"comments,omitempty"`
	ChecksValue    github.Statuses           `yaml:"checks,omitempty"`
	OwnerValue     *github.User              `yaml:"owner,omitempty"`
	CreatedAtValue string                    `yaml:"createdAt,omitempty"`
}

func (pr *MockPullRequest) Labels() []*github.Label                   { return pr.LabelsValue }
func (pr *MockPullRequest) Base() *github.PullRequestBranch           { return pr.BaseValue }
func (pr *MockPullRequest) Head() *github.PullRequestBranch           { return pr.HeadValue }
func (pr *MockPullRequest) Comments() ([]*github.IssueComment, error) { return pr.CommentsValue, nil }
func (pr *MockPullRequest) Checks() (github.Statuses, error)          { return pr.ChecksValue, nil }
func (pr *MockPullRequest) Logger() logger.Logger                     { return logger.ZeroLogger(log.Logger) }
func (pr *MockPullRequest) Owner() *github.User                       { return pr.OwnerValue }
func (pr *MockPullRequest) CreatedAt() *github.Timestamp {
	if pr.CreatedAtValue == "now" {
		return &github.Timestamp{time.Now()}
	}
	t, err := time.Parse(time.RFC3339, pr.CreatedAtValue)
	if err != nil {
		panic(err)
	}
	return &github.Timestamp{t}
}

func (pr *MockErrorPullRequest) Comments() ([]*github.IssueComment, error) {
	return nil, fmt.Errorf("mock error: comments")
}
func (pr *MockErrorPullRequest) Checks() (github.Statuses, error) {
	return nil, fmt.Errorf("mock error: checks")
}

// type assertion
var _ github.PullRequestResolver = &MockPullRequest{}
var _ github.PullRequestResolver = &MockErrorPullRequest{}
