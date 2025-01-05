package oauth2

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/containifyci/dunebot/pkg/auth"
	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)


func SetupGRPCClient(t *testing.T, user string, tokens ...string) Config {
	if len(tokens) == 0 {
		tokens = []string{""}
	}
	privateKey, publicKey := GenerateECDSAKey()
	cfg := SetupTokenService(t, tokens[0])

	go func() {
		cfg.PublicKey = publicKey
		err := storage.StartServers(cfg)
		assert.NoError(t, err)
	}()
	// wait for the server to start
	// time.Sleep(5 * time.Second)

	signer := auth.NewSigningService(privateKey)

	tokenFnc := signer.CreateTokenFnc(auth.ServiceClaims{ServiceName: "dunebot"})

	cfg2 := config.Config{
		Github: config.GithubConfig{
			OAuth: config.GithubOAuthConfig{
				ClientID:     "client_id",
				ClientSecret: "client_secret",
				Scopes:       []string{"repo", "user"},
				RedirectURL:  "http://localhost:8080/oauth2/callback",
			},
		},
	}

	config := Config{
		AuthInterceptor: *NewAuthInterceptor(tokenFnc),
		Addr:            fmt.Sprintf(":%d", cfg.GRPCPort),
		Ctx:             context.Background(),
		InstallationId:  1,
		User:            user,
		OAuth2Config:    GetConfig(&cfg2),
	}
	config.OAuth2Config.Endpoint.AuthURL = fmt.Sprintf("http://localhost:%d", cfg.GRPCPort)
	return config
}

func SetupTokenService(t *testing.T, tokens string) storage.Config {
	s := storage.Installation{
		InstallationId: 1,
		Tokens: []*storage.CustomToken{
			{
				AccessToken:  "access",
				RefreshToken: "refresh",
				TokenType:    "type",
				User:         "user",
				Expiry:       timestamppb.New(time.Now()),
			},
		},
	}

	m := make(map[int64]*storage.Installation)
	m[1] = &s
	b, err := json.Marshal(m)
	assert.NoError(t, err)

	file := t.TempDir() + "/tokens.json"

	err = os.WriteFile(file, b, 0644)
	assert.NoError(t, err)

	cfg := storage.Config{
		TokenSyncPeriod: "1m",
		StorageFile:     file,
		GRPCPort:        GetFreePort(),
	}

	return cfg
}

func GenerateECDSAKey() (privateKey string, publicKey string) {
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

func GetFreePort() int {
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
