package github

import (
	"github.com/containifyci/dunebot/pkg/logger"
)

type PullRequestResolver interface {
	Comments() ([]*IssueComment, error)
	Checks() (Statuses, error)
	Owner() *User
	Labels() []*Label
	Base() *PullRequestBranch
	Head() *PullRequestBranch
	CreatedAt() *Timestamp
	Logger() logger.Logger
}

type PullRequestClient struct {
	*PullRequest
	client *GithubClient
}

func NewPullRequestClient(client *GithubClient, pr *PullRequest) PullRequestResolver {
	p := &PullRequestClient{
		PullRequest: pr,
		client:      client,
	}
	return p
}

func (pr *PullRequestClient) Comments() ([]*IssueComment, error) {
	return pr.client.GetPRComments(pr.GetNumber())
}

func (pr *PullRequestClient) Owner() *User {
	return pr.GetUser()
}

func (pr *PullRequestClient) Labels() []*Label {
	return pr.PullRequest.Labels
}

func (pr *PullRequestClient) Base() *PullRequestBranch {
	return pr.GetBase()
}

func (pr *PullRequestClient) Head() *PullRequestBranch {
	return pr.GetHead()
}

func (pr *PullRequestClient) Checks() (Statuses, error) {
	return pr.client.GetStatuses(pr.PullRequest)
}

func (pr *PullRequestClient) CreatedAt() *Timestamp {
	return pr.PullRequest.CreatedAt
}

func (pr *PullRequestClient) Logger() logger.Logger {
	return pr.client.logger
}

// compile time check that the PullRequestClient implements the PullRequestResolver interface
var _ PullRequestResolver = &PullRequestClient{}
