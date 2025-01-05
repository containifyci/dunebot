package cmd

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/alexedwards/scs"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
	"github.com/containifyci/dunebot/pkg/github/testdata"
	"github.com/containifyci/dunebot/pkg/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type (
	WebHandlerTest struct {
		sessions SessionHandler
		gh       *github.Client
		cfg      *config.Config

		tokenService *TokenServiceTest

		*testing.T
	}
	Option func(*WebHandlerTest)

	TokenServiceTest struct {
		server  *grpc.Server
		service *storage.TokenService
		config  storage.Config
	}
)

func WithSessionManager(mgr SessionHandler) Option {
	return func(w *WebHandlerTest) {
		if mgr != nil {
			w.sessions = mgr
		}
	}
}

func WithJWTAddress(address string) Option {
	return func(w *WebHandlerTest) {
		if address != "" {
			w.cfg.JWT.Address = address
		}
	}
}

func NewWebHandlerTest(t *testing.T, folder string, tokens string, opts ...Option) *WebHandlerTest {
	if tokens == "" {
		tokens = "{}"
	}
	tkCfg, tkSrv := setupTokenService(t, tokens)
	grpc := storage.StartGRPCServer(tkSrv)

	cfg := newConfigForTest(tkCfg)

	mockGitHubClient := makeGithubTestClient(folder)
	mockSessionManager := newMockSessionManaer()

	sessions := NewWebHandler(mockSessionManager)
	wb := &WebHandlerTest{sessions: sessions, gh: mockGitHubClient, cfg: cfg, T: t}
	wb.tokenService = &TokenServiceTest{server: grpc, service: tkSrv, config: tkCfg}

	for _, opt := range opts {
		opt(wb)
	}
	return wb
}

func (w *WebHandlerTest) ServeHTTP(handler http.Handler, url string) *httptest.ResponseRecorder {
	// Create a request to pass to the handler
	req, err := http.NewRequest("GET", url, nil)
	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "username",
	})
	assert.NoError(w.T, err)
	// Create a response recorder to capture the handler output
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func (w *WebHandlerTest) Close() {
	w.tokenService.server.GracefulStop()
}

func TestDashboardHandlerSuccess(t *testing.T) {
	t.Parallel()
	test := NewWebHandlerTest(t, "testdata", tokens)
	defer test.Close()
	handler := DashboardHandler(test.cfg, test.gh, &MockGithub{"testdata"}, test.sessions)
	rr := test.ServeHTTP(handler, "/dashboard")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "<!DOCTYPE html>\n<html>\n<head>\n\t\t<title>Authenticated Users</title>\n</head>\n<body>\n\t\t<h1>Authenticated Users</h1><div>\n\t\t\t\t\t<p>user</p>\n\t\t\t\t\t<hr class=\"horizontal_line_class\" />\n\t\t\t</div></body>\n</html>", strings.Trim(rr.Body.String(), "\n"))
}

type WebHandlerMock struct{}

func (w WebHandlerMock) GetString(req *http.Request, key string) (string, error) {
	return "", errors.New("forced-error")
}

func TestDashboardHandlerErrors(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name            string
		mockGitHubData  string
		mockGitHubData2 string
		mockSessionMgr  SessionHandler
		jwtAddress      string
		expectedCode    int
		expectedBody    string
		setup           func() func()
	}

	cases := []testCase{
		{
			name:           "SessionError",
			mockSessionMgr: &WebHandlerMock{},
			expectedCode:   http.StatusInternalServerError,
			expectedBody:   "Error reading session",
		},
		{
			name:           "InstallationError",
			mockGitHubData: "testdata/error",
			expectedCode:   http.StatusInternalServerError,
			expectedBody:   "Error reading installation",
		},
		{
			name:         "GRPCError",
			jwtAddress:   "\x7f",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Failed to connect to gRPC server",
		},
		{
			name:         "RetrieveInstallationError",
			jwtAddress:   "localhost:12345",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Failed to retrieve installation",
		},
		{
			name:            "UsersError",
			mockGitHubData2: "testdata/error2",
			expectedCode:    http.StatusInternalServerError,
			expectedBody:    "Error getting user from token source",
		},
		{
			name:            "UserOrgsError",
			mockGitHubData2: "testdata/error3",
			expectedCode:    http.StatusInternalServerError,
			expectedBody:    "Failed to retrieve user orgs",
		},
		{
			name:         "TemplateError",
			expectedCode: http.StatusInternalServerError,
			expectedBody: "Error parsing template",
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

			test := NewWebHandlerTest(
				t, c.mockGitHubData, tokens,
				WithSessionManager(c.mockSessionMgr), WithJWTAddress(c.jwtAddress),
			)
			defer test.Close()
			handler := DashboardHandler(test.cfg, test.gh, &MockGithub{c.mockGitHubData2}, test.sessions)
			rr := test.ServeHTTP(handler, "/dashboard")

			// Assertions
			assert.Equal(t, c.expectedCode, rr.Code)
			assert.Equal(t, c.expectedBody, strings.Trim(rr.Body.String(), "\n"))
		})
	}
}

// utility functions

func newConfigForTest(tkCfg storage.Config) *config.Config {
	privKey, _ := generateECDSAKey()
	// Mock data
	cfg := &config.Config{
		JWT: config.JWTConfig{
			PrivateKey: privKey,
			Address:    fmt.Sprintf(":%d", tkCfg.GRPCPort),
		},
	}
	return cfg
}

func makeGithubTestClient(folder string) *github.Client {
	rp := testdata.NewResponsePlayer(folder)
	return github.Newclient(&http.Client{Transport: rp})
}

func newMockSessionManaer() *scs.Manager {
	store := newMockStore()
	entry := mockEntry{map[string]interface{}{"username": "user"}, time.Now().Add(time.Hour).Unix()}
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(entry)
	_ = store.Save("username", b.Bytes(), time.Now().Add(time.Hour))
	mockSessionManager := scs.NewManager(store)
	mockSessionManager.AddToContext(context.Background(), &scs.Session{})
	return mockSessionManager
}

func generateECDSAKey() (privateKey string, publicKey string) {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// Extract public component.
	pub := key.Public()

	privKey, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}

	// Encode private key to PKCS#1 ASN.1 PEM.
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "ECDSA PRIVATE KEY",
			Bytes: privKey,
		},
	)

	privateKey = base64.StdEncoding.EncodeToString(keyPEM)

	pubKey, err := x509.MarshalPKIXPublicKey(pub.(*ecdsa.PublicKey))
	if err != nil {
		panic(err)
	}

	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "ECDSA PUBLIC KEY",
			Bytes: pubKey,
		},
	)
	publicKey = base64.StdEncoding.EncodeToString(pubPEM)
	return privateKey, publicKey
}

func setupTokenService(t *testing.T, tokens string) (storage.Config, *storage.TokenService) {
	file := t.TempDir() + "/tokens.json"
	fmt.Printf("token tmp %s", file)
	err := os.WriteFile(file, []byte(tokens), 0644)
	assert.NoError(t, err)

	cfg := storage.Config{
		TokenSyncPeriod: "1m",
		StorageFile:     file,
		GRPCPort:        getFreePort(),
	}

	ts := storage.NewTokenService(cfg)
	assert.NotNil(t, ts)
	return cfg, ts
}

const tokens = `{"1":{"installation_id":1,"tokens":[{"access_token":"access","refresh_token":"refresh","expiry":{"seconds":1715603314,"nanos":409109000},"token_type":"type","user":"user"}]}}`

type mockStore struct {
	m map[string][]byte
}

type mockEntry struct {
	Data     map[string]interface{} `json:"data"`
	Deadline int64                  `json:"deadline"`
}

func newMockStore() *mockStore {
	m := make(map[string][]byte)
	return &mockStore{m}
}

func (s *mockStore) Delete(token string) error {
	delete(s.m, token)
	return nil
}

func (s *mockStore) Find(token string) (b []byte, found bool, err error) {
	if token == "force-error" {
		return nil, false, errors.New("forced-error")
	}
	entry, exists := s.m[token]
	if !exists {
		return nil, false, nil
	}
	return entry, true, nil
}

func (s *mockStore) Save(token string, b []byte, expiry time.Time) error {
	s.m[token] = b
	return nil
}

func getFreePort() int {
	var a *net.TCPAddr
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	var l *net.TCPListener
	l, err = net.ListenTCP("tcp", a)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
