package review

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/logger"

	"github.com/containifyci/dunebot/pkg/github/testdata"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestReview(t *testing.T) {
	// Generate a new RSA private key with 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)
	cfg := config.Config{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 1,
				PrivateKey:    string(privateKeyPEMBytes),
			},
		},
		AppConfig: config.ApplicationConfig{
			ReviewerConfig: config.ReviewerConfig{
				Type: "direct",
			},
		},
	}
	cc, err := githubapp.NewDefaultCachingClientCreator(
		//TODO maybe find an better way then to copy the config
		cfg.Github.ToGithubAppConfig(),
		githubapp.WithClientUserAgent(cfg.AppConfig.UserAgent),
		githubapp.WithClientTimeout(cfg.AppConfig.ClientTimeOutDuration()),
		githubapp.WithTransport(makeTestClient()),
	)
	assert.NoError(t, err)

	ctx := context.Background()

	event := github.PullRequestEvent{
		Action: github.String("opened"),
		Installation: &github.Installation{
			ID: github.Int64(43975733),
		},
		PullRequest: &github.PullRequest{
			Number: github.Int(2),
			Head: &github.PullRequestBranch{
				Ref: github.String("develop"),
			},
			User: &github.User{
				Login: github.String("dune"),
			},
		},
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("test-owner"),
			},
			Name: github.String("test-repo"),
		},
	}

	reviewer := NewReviewer(logger.NewZeroLogger(), nil, cc, cfg, WithTransport(makeTestClient()))

	err = reviewer.PullRequestReview(ctx, PullRequestReview{
		PullRequest: event.GetPullRequest(),
		Repository:  event.GetRepo(),
		Event:       &event,
		Config: &config.AppConfig{
			Approve: &config.Approve{
				Approver: "unittest",
				Include: &config.Include{
					Authors: config.StringArray{"fr12k"},
					Branches: map[string]*config.Branch{
						"br1": {
							Key:    "br1",
							Prefix: "develop",
						},
					},
				},
			},
			Merge: &config.Merge{
				Include: &config.Include{
					Authors: config.StringArray{"fr12k"},
					Branches: map[string]*config.Branch{
						"br1": {
							Key:    "br1",
							Prefix: "develop",
						},
					},
				},
			},
		},
	})

	assert.NoError(t, err)
}

func TestPullRequestReviewError(t *testing.T) {
	// oauthCfg := oauth2.SetupGRPCClient(t, "user")

	// Generate a new RSA private key with 2048 bits
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)
	cfg := config.Config{
		Github: config.GithubConfig{
			App: struct {
				IntegrationID int64  `yaml:"integration_id" json:"integrationId" envconfig:"INTEGRATION_ID"`
				WebhookSecret string `yaml:"webhook_secret" json:"webhookSecret" envconfig:"WEBHOOK_SECRET"`
				PrivateKey    string `yaml:"private_key" json:"privateKey" envconfig:"PRIVATE_KEY"`
			}{
				IntegrationID: 1,
				PrivateKey:    string(privateKeyPEMBytes),
			},
		},
		AppConfig: config.ApplicationConfig{
			ReviewerConfig: config.ReviewerConfig{
				Type: "direct",
			},
		},
	}

	ctx := context.Background()

	event := github.PullRequestEvent{
		Action: github.String("opened"),
		Installation: &github.Installation{
			ID: github.Int64(43975733),
		},
		PullRequest: &github.PullRequest{
			Number: github.Int(2),
			Head: &github.PullRequestBranch{
				Ref: github.String("develop"),
			},
			User: &github.User{
				Login: github.String("dune"),
			},
		},
		Repo: &github.Repository{
			Owner: &github.User{
				Login: github.String("test-owner"),
			},
			Name: github.String("test-repo"),
		},
	}

	test := []struct {
		rule dynamicRule
	}{
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/pulls/2",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/pulls/2/update-branch",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/branches/main/protection",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/pulls/2/reviews",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/actions/runs",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
		{
			rule: dynamicRule{
				path:  "/repos/test-owner/test-repo/pulls/2/merge",
				query: "",
				handler: testdata.NewResponseHandler(func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
					}
				}),
			},
		},
	}

	for _, tt := range test {
		t.Run(tt.rule.path, func(t *testing.T) {
			testGithubClient := makeTestClient()
			testGithubClient.SetDynamicRule(tt.rule.path, tt.rule.query, tt.rule.handler)

			cc, err := githubapp.NewDefaultCachingClientCreator(
				//TODO maybe find an better way then to copy the config
				cfg.Github.ToGithubAppConfig(),
				githubapp.WithClientUserAgent(cfg.AppConfig.UserAgent),
				githubapp.WithClientTimeout(cfg.AppConfig.ClientTimeOutDuration()),
				githubapp.WithTransport(testGithubClient),
			)
			assert.NoError(t, err)
			reviewer := NewReviewer(logger.NewZeroLogger(), nil, cc, cfg, WithTransport(testGithubClient))

			err = reviewer.PullRequestReview(ctx, PullRequestReview{
				PullRequest: event.GetPullRequest(),
				Repository:  event.GetRepo(),
				Event:       &event,
				Config: &config.AppConfig{
					Approve: &config.Approve{
						Approver: "unittest",
						Include: &config.Include{
							Authors: config.StringArray{"fr12k"},
							Branches: map[string]*config.Branch{
								"br1": {
									Key:    "br1",
									Prefix: "develop",
								},
							},
						},
					},
					Merge: &config.Merge{
						Include: &config.Include{
							Authors: config.StringArray{"fr12k"},
							Branches: map[string]*config.Branch{
								"br1": {
									Key:    "br1",
									Prefix: "develop",
								},
							},
						},
					},
				},
			})
			assert.Error(t, err)
			assert.ErrorContains(t, err, "400  []")
		})
	}
}

// test utilities

type dynamicRule struct {
	path    string
	query   string
	handler testdata.ResponseHandler
}

func makeTestClient() *testdata.ResponsePlayer {
	return testdata.NewResponsePlayer("testdata")
}
