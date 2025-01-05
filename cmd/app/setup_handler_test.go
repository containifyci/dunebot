package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/containifyci/dunebot/oauth2"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRegisterSetup(t *testing.T) {
	test := NewWebHandlerTest(t, "testdata", "")
	t.Cleanup(test.Close)

	srv := oauth2.NewMockOAuth2Server()
	t.Cleanup(srv.Close)

	cfg := &TestConfig{
		endpoint: srv.Endpoint(),
		cfg:      test.cfg,
	}

	mux := http.NewServeMux()
	registerSetup(cfg, test.gh, &MockGithub{"testdata"}, mux)

	rr := test.ServeHTTP(mux, "/app/github/setup?installation_id=1")

	fmt.Println("Active goroutines:", runtime.NumGoroutine())

	err := waitUntilGoroutines(runtime.NumGoroutine()-1, 30*time.Second)
	assert.NoError(t, err)
	fmt.Println("Active goroutines:", runtime.NumGoroutine())

	err = test.tokenService.service.Save()
	assert.NoError(t, err)

	fileStorage := storage.NewFileStorage(test.tokenService.config.StorageFile)
	data, err := fileStorage.Load()
	assert.NoError(t, err)

	assert.Equal(t, 1, len(data))
	assert.Equal(t, int64(1), data[1].InstallationId)
	assert.Equal(t, 1, len(data[1].Tokens))
	assert.Equal(t, "mocktoken", data[1].Tokens[0].AccessToken)
	assert.Equal(t, "mockrefresh", data[1].Tokens[0].RefreshToken)
	assert.Equal(t, "user", data[1].Tokens[0].User)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "<!DOCTYPE html>\n<html>\n<head>\n    <title>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</title>\n</head>\n<body>\n    <h1>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</h1>\n    <a href=\"mockverificationuri\" target=\"_blank\">Please open</a>\n    <p>mockusercode</p>\n</body>\n</html>", strings.Trim(rr.Body.String(), "\n"))
}

func TestRegisterSetupErrors(t *testing.T) {
	origLogError := logError
	defer func() {
		logError = origLogError
	}()
	type testCase struct {
		installationId     string
		name               string
		mockGitHubData     string
		mockGitHubData2    string
		mockSessionMgr     SessionHandler
		jwtAddress         string
		expectedCode       int
		expectedBody       string
		expectedErr        error
		waitForGoroutine   bool
		noMockOAuth2Server bool
		setup              func() func()
	}

	cases := []testCase{
		{
			name:           "Invalid Installation Id",
			installationId: "ab",
			mockGitHubData: "testdata/error5",
			expectedCode:   http.StatusBadRequest,
			expectedBody:   "Invalid installation ID",
		},
		{
			name:           "InstallationError",
			mockGitHubData: "testdata/error5",
			expectedCode:   http.StatusInternalServerError,
			expectedBody:   "Error requesting installation",
		},
		{
			name:               "DeviceFlowError",
			expectedCode:       http.StatusInternalServerError,
			expectedBody:       "Error requesting device flow",
			noMockOAuth2Server: true,
		},
		{
			name:             "GRPCError",
			jwtAddress:       "\x7f",
			expectedCode:     http.StatusOK,
			expectedBody:     "<!DOCTYPE html>\n<html>\n<head>\n    <title>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</title>\n</head>\n<body>\n    <h1>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</h1>\n    <a href=\"mockverificationuri\" target=\"_blank\">Please open</a>\n    <p>mockusercode</p>\n</body>\n</html>",
			expectedErr:      errors.New("parse \"dns:///\\x7f\": net/url: invalid control character in URL"),
			waitForGoroutine: true,
		},
		{
			name:             "UsersError",
			mockGitHubData2:  "testdata/error2",
			expectedCode:     http.StatusOK,
			expectedBody:     "<!DOCTYPE html>\n<html>\n<head>\n    <title>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</title>\n</head>\n<body>\n    <h1>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</h1>\n    <a href=\"mockverificationuri\" target=\"_blank\">Please open</a>\n    <p>mockusercode</p>\n</body>\n</html>",
			expectedErr:      errors.New("GET https://api.github.com/user: 500 Internal server error []"),
			waitForGoroutine: true,
		},
		{
			name:             "RetrieveUserInstallationsError",
			mockGitHubData2:  "testdata/error4",
			expectedCode:     http.StatusOK,
			expectedBody:     "<!DOCTYPE html>\n<html>\n<head>\n    <title>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</title>\n</head>\n<body>\n    <h1>In order for DuneBot to approve PRs, you need to connect it to an Github account with CodeOwnership permissions.</h1>\n    <a href=\"mockverificationuri\" target=\"_blank\">Please open</a>\n    <p>mockusercode</p>\n</body>\n</html>",
			expectedErr:      errors.New("GET https://api.github.com/user/installations: 500 Internal server error []"),
			waitForGoroutine: true,
		},
		{
			name:             "TemplateError",
			expectedCode:     http.StatusInternalServerError,
			expectedBody:     "Error rendering template",
			waitForGoroutine: true,
			setup: func() func() {
				originalTemplates := templateFnc
				templateFnc = func(tmpl string, name string, data any) (*string, *template.Template, error) {
					return nil, nil, assert.AnError
				}
				return func() {
					templateFnc = originalTemplates
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.installationId == "" {
				c.installationId = "1"
			}
			if c.mockGitHubData == "" {
				c.mockGitHubData = "testdata"
			}
			if c.mockGitHubData2 == "" {
				c.mockGitHubData2 = c.mockGitHubData
			}
			if c.setup != nil {
				f := c.setup()
				defer f()
			}

			testError := &TestErrorReporter{t: t, expectedErr: c.expectedErr}
			logError = testError

			fmt.Println("Active goroutines:", runtime.NumGoroutine())

			test := NewWebHandlerTest(
				t, c.mockGitHubData, "",
				WithSessionManager(c.mockSessionMgr), WithJWTAddress(c.jwtAddress),
			)
			defer test.Close()
			cfg := &TestConfig{
				cfg: test.cfg,
			}
			if !c.noMockOAuth2Server {
				srv := oauth2.NewMockOAuth2Server()
				t.Cleanup(srv.Close)
				cfg.endpoint = srv.Endpoint()
			}

			mux := http.NewServeMux()
			registerSetup(cfg, test.gh, &MockGithub{c.mockGitHubData2}, mux)
			rr := test.ServeHTTP(mux, fmt.Sprintf("/app/github/setup?installation_id=%s", c.installationId))
			if c.waitForGoroutine {
				err := waitUntilGoroutines(runtime.NumGoroutine()-1, 2*time.Second)
				assert.NoError(t, err)
			}
			assert.Equal(t, c.expectedCode, rr.Code)
			assert.Equal(t, c.expectedBody, strings.Trim(rr.Body.String(), "\n"))
			if c.expectedErr != nil {
				testError.Assert(10 * time.Second)
			}
		})
	}
}

func TestLogErrorReporter(t *testing.T) {
	logs := &logSink{}
	logErrorReporter{logger: zerolog.New(logs)}.Errorf(errors.New("test error"), "test %s", "message")

	assert.Equal(t, 1, len(logs.logs))
	assert.Contains(t, logs.Index(0), "test message")
}

// Test utility

type logSink struct {
	logs []string
}

func (l *logSink) Write(p []byte) (n int, err error) {
	l.logs = append(l.logs, string(p))
	return len(p), nil
}

func (l *logSink) Index(i int) string {
	return l.logs[i]
}

type TestErrorReporter struct {
	mutex       sync.RWMutex
	t           *testing.T
	expectedErr error
	wasCalled   bool
}

func (t *TestErrorReporter) Errorf(err error, format string, args interface{}) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.wasCalled = true
	fmt.Printf(format, args)
	if t.expectedErr != nil {
		assert.Equal(t.t, t.expectedErr.Error(), err.Error())
	}
}

func (t *TestErrorReporter) Assert(timeout time.Duration) {
	timeoutCh := time.After(timeout)
	for {
		select {
		case <-timeoutCh:
			// Timeout reached
			fmt.Println("Timeout reached")
			assert.Fail(t.t, "Expected an error message to be logged")
		default:
			t.mutex.RLock()
			if t.wasCalled {
				t.mutex.RUnlock()
				return
			}
			t.mutex.RUnlock()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

type TestConfig struct {
	endpoint oauth2.Endpoint
	cfg      *config.Config
}

func (c *TestConfig) Config() *config.Config {
	return c.cfg
}

func (c *TestConfig) ToOAuth2Config() *oauth2.OAuth2Config {
	return &oauth2.OAuth2Config{
		ClientID:     "client_id",
		ClientSecret: "client_secret",
		Scopes:       []string{"user"},
		Endpoint:     c.endpoint,
	}
}

type MockGithub struct {
	folder string
}

func (m *MockGithub) NewClient(opts ...github.Option) github.GithubClient {
	return github.NewClient(
		github.WithContext(context.Background()),
		github.WithGithubClient(makeGithubTestClient(m.folder)),
	)
}

// waitUntilGoroutines waits until the number of goroutines reaches targetGoroutines or a timeout occurs.
func waitUntilGoroutines(targetGoroutines int, timeout time.Duration) error {
	timeoutCh := time.After(timeout)

	for {
		select {
		case <-timeoutCh:
			return fmt.Errorf("timeout reached while waiting for goroutines")
		default:
			if runtime.NumGoroutine() <= targetGoroutines {
				return nil
			}
			time.Sleep(100 * time.Millisecond) // Polling interval
		}
	}
}
