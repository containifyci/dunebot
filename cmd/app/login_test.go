package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/palantir/go-githubapp/oauth2"
	"github.com/stretchr/testify/assert"
)

func TestRequireLoginWithLoggedInUser(t *testing.T) {
	t.Parallel()
	cfg := config.Config{}
	sessions := newTestSessions()
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Define a handler that requires a use that is already logged in to Github
	handler := RequireLogin(sessions, ts.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("User is logged in OK"))
		assert.NoError(t, err)
	}))

	mux.Handle("/login/protected", handler)

	// Create a new login handler with mocked Github client that returns already an authenticated user
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Login(cfg.Github, sessions)(w, r, &oauth2.Login{
			Client: makeTestGithubClient(makeTestClient()).Client(),
		})
	})
	mux.Handle(oauth2.DefaultRoute, handler)

	cli := newTestClient(ts, true)

	resp, err := cli.Get(ts.URL + "/login/protected")
	assert.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, string(body), "User is logged in OK")
}

func TestRequireLoginWithUnauthenticatedUser(t *testing.T) {
	t.Parallel()
	cfg := config.Config{}
	sessions := newTestSessions()
	mux := setupHttpServeMux(&cfg)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Define a handler that requires a use that is already logged in to Github
	handler := RequireLogin(sessions, ts.URL)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Should not be called because user is not logged in.")
	}))

	mux.Handle("/login/protected", handler)

	cli := newTestClient(ts, false)

	resp, err := cli.Get(ts.URL + "/login/protected")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("%s/api/github/auth", ts.URL), resp.Header.Get("Location"))
}

// test helper functions

func newTestSessions() *scs.Manager {
	sessions := scs.NewCookieManager("session-key")
	sessions.Name("dunebot")
	sessions.Lifetime(24 * time.Hour)
	sessions.Persist(true)
	sessions.HttpOnly(true)
	sessions.Secure(false)
	return sessions
}

func newTestClient(ts *httptest.Server, followRedirect bool) *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	cli := ts.Client()

	cli.Jar = jar
	if !followRedirect {
		cli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return cli
}
