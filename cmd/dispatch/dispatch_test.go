package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/containifyci/dunebot/cmd"
	"github.com/stretchr/testify/assert"

	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/github/testdata"
)

func TestExecute(t *testing.T) {
	// Get the root command
	rootCmd := cmd.RootCmd

	// Capture the output
	output := new(bytes.Buffer)
	rootCmd.SetOut(output)

	t.Setenv("DUNEBOT_GITHUB_TOKEN", "top_secret")
	t.Setenv("DUNEBOT_GITHUB_APPLICATION_ID", "123")

	server := RepositoryServer("[\"test_owner/test_repository\"]")
	defer server.Close() // Ensure the server is closed after the test

	t.Setenv("DUNEBOT_REPOSITORY_ENDPOINT", server.URL)

	// Execute the greet command
	rootCmd.SetArgs([]string{"dispatch"})

	err := rootCmd.Execute()
	assert.ErrorContains(t, err, "401 Bad credentials")
}

func TestRun(t *testing.T) {
	server := RepositoryServer("[\"test-owner/test-repo\"]")
	defer server.Close() // Ensure the server is closed after the test

	cfg := NewDispatchConfig(server)

	tc := makeTestClient()

	tc.AddDynamicRule("/repos/test-owner/test-repo/dispatches", "",
		testdata.NewResponseHandler(AssertDispatchRequest(t, 1)), testdata.NewResponseHandler(AssertDispatchRequest(t, 2)))

	gh := github.NewClient(github.WithGithubClient(tc.Client))
	err := run(cfg, gh)
	assert.NoError(t, err)
}

func TestRunWithDryRun(t *testing.T) {
	server := RepositoryServer("[\"test-owner/test-repo\"]")
	defer server.Close() // Ensure the server is closed after the test

	cfg := NewDispatchConfig(server)
	dispatchArgs.dryRun = true

	tc := makeTestClient()

	tc.AddDynamicRule("/repos/test-owner/test-repo/dispatches", "", testdata.NewResponseHandler(func(req *http.Request) *http.Response {
		assert.FailNowf(t, "unexpected request", "unexpected request: %v", req)
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
		}
	}))

	gh := github.NewClient(github.WithGithubClient(tc.Client))
	err := run(cfg, gh)
	assert.NoError(t, err)
}

func TestRunWithRepoFilter(t *testing.T) {
	server := RepositoryServer("[\"test-owner/test-repo\"]")
	defer server.Close() // Ensure the server is closed after the test

	cfg := NewDispatchConfig(server)
	// dispatchArgs.dryRun = true
	dispatchArgs.repo = "test-owner/test-repo"

	tc := makeTestClient()

	h1 := testdata.NewResponseHandler(AssertDispatchRequest(t, 1))

	h2 := testdata.NewResponseHandler(AssertDispatchRequest(t, 2))

	tc.AddDynamicRule("/repos/test-owner/test-repo/dispatches", "", h1, h2)

	gh := github.NewClient(github.WithGithubClient(tc.Client))

	err := run(cfg, gh)
	assert.NoError(t, err)

	assert.True(t, h1.Called, "Expected handler to be called")
	assert.True(t, h2.Called, "Expected handler to be called")
}

func TestRunDispatchError(t *testing.T) {
	server := RepositoryServer("[\"test-owner/test-repo\"]")
	defer server.Close() // Ensure the server is closed after the test

	cfg := NewDispatchConfig(server)

	tc := makeTestClient()

	tc.AddDynamicRule("/repos/test-owner/test-repo/dispatches", "", testdata.NewResponseHandler(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
		}
	}))

	gh := github.NewClient(github.WithGithubClient(tc.Client))
	err := run(cfg, gh)
	assert.ErrorContains(t, err, "500  []")
}

func TestListDuneBotRepositoriesError(t *testing.T) {
	// Create a new instance of the test HTTP server
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Your handler logic here
		w.WriteHeader(status)
		_, err := w.Write(nil)
		assert.NoError(t, err)
	}))
	defer server.Close() // Ensure the server is closed after the test

	cfg := NewDispatchConfig(nil)

	test := []struct {
		name   string
		ctx    context.Context
		pre    func()
		errMsg string
	}{
		{
			name:   "fail to create http request",
			ctx:    nil,
			errMsg: "net/http: nil Context",
		},
		{
			name:   "fail to make http request",
			ctx:    context.Background(),
			errMsg: "Get \"\": unsupported protocol scheme \"\"",
		},
		{
			name:   "fail to unmarshal json",
			ctx:    context.Background(),
			pre:    func() { cfg.RepositoryEndpoint = server.URL },
			errMsg: "unexpected end of JSON input",
		},
		{
			name:   "received non 200 status code",
			ctx:    context.Background(),
			pre:    func() { cfg.RepositoryEndpoint = server.URL; status = http.StatusUnauthorized },
			errMsg: "received status code 401",
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pre != nil {
				tt.pre()
			}
			_, err := listDuneBotRepositories(tt.ctx, cfg)
			assert.ErrorContains(t, err, tt.errMsg)
		})
	}
}

// test helper funcions

func makeTestClient() TestGithubClient {
	rp := testdata.NewResponsePlayer("../testdata")
	return TestGithubClient{
		rp,
		github.Newclient(&http.Client{Transport: rp}),
	}
}

type TestGithubClient struct {
	rp *testdata.ResponsePlayer
	*github.Client
}

func (c *TestGithubClient) AddDynamicRule(path, query string, h ...*testdata.ResponseHandlerFuncWrapper) {
	rule := c.rp.AddDynamicRule("/repos/test-owner/test-repo/dispatches", "", h[0])
	handlers := h[1:]
	for _, handler := range handlers {
		rule.AddResponseHandler(handler)
	}
}

// Create a new instance of the test HTTP server
func RepositoryServer(response string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		if err != nil {
			panic(err)
		}
	}))
	return server
}

func AssertDispatchRequest(t *testing.T, prNumber int) testdata.ResponseHandlerFunc {
	return func(req *http.Request) *http.Response {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("{\"event_type\":\"pull_request\",\"client_payload\":{\"pull_request\":{\"number\":%d,\"state\":\"open\",\"title\":\"Test PR %d\",\"user\":{\"login\":\"fr12k\"},\"head\":{\"ref\":\"develop\"}},\"owner\":\"test-owner\",\"repository\":\"test-repo\"}}\n", prNumber, prNumber), string(body))
		return &http.Response{
			StatusCode: http.StatusNoContent,
		}
	}
}

func NewDispatchConfig(server *httptest.Server) *dispatchConfig {
	dispatchArgs.dryRun = false
	if server == nil {
		return &dispatchConfig{
			GithubToken: "top_secret",
		}
	}
	return &dispatchConfig{
		GithubToken:        "top_secret",
		RepositoryEndpoint: server.URL,
	}
}
