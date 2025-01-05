package oauth2

import (
	"net/http"
	"net/http/httptest"

	"golang.org/x/oauth2"
)

type MockOAuth2Server struct {
	*httptest.Server
}

func NewMockOAuth2Server() *MockOAuth2Server {
	return (&MockOAuth2Server{}).setup()
}

func (m *MockOAuth2Server) setup() *MockOAuth2Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://localhost:8080/oauth2/callback?code=mockcode")
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("access_token=mocktoken&scope=user&token_type=mocktype&refresh_token=mockrefresh"))
	})

	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_code":"mockdevicecode","user_code":"mockusercode","verification_uri":"mockverificationuri","interval":1,"expires_in":600}`))
	})

	m.Server = httptest.NewServer(mux)
	return m
}

func (m *MockOAuth2Server) Close() {
	m.Server.Close()
}

func (m *MockOAuth2Server) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:       m.URL + "/auth",
		TokenURL:      m.URL + "/token",
		DeviceAuthURL: m.URL + "/device",
	}
}
