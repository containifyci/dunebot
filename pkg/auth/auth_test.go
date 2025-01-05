package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestSigning(t *testing.T) {
	privateKey, publicKey := generateECDSAKey()

	signer := NewSigningService(privateKey)

	token := signer.CreateToken(ServiceClaims{ServiceName: "service_test.go"})

	verifier := NewVerifyService(publicKey)

	claims, err := verifier.ValidateToken(token)
	assert.NoError(t, err)

	fmt.Println(jwt.NewNumericDate(time.Now()), claims.IssuedAt)

	assert.True(t, signer.IsEnabled())
	assert.Equal(t, "service:service_test.go", claims.Issuer)
	assert.Equal(t, "service:service_test.go", claims.Subject)
	assert.Equal(t, jwt.ClaimStrings{"service:service_test.go"}, claims.Audience)
	//TODO: ignore the time assertion for now find a good way to mock the time so that it can be tested.
	// assert.Equal(t, jwt.NewNumericDate(time.Now()), claims.IssuedAt)
	// assert.Equal(t, jwt.NewNumericDate(time.Now().Add(time.Hour*1)), claims.ExpiresAt)
	// assert.Equal(t, jwt.NewNumericDate(time.Now().Add(time.Minute*-5)), claims.NotBefore)
}

func TestSigningWithoutPrivateKey(t *testing.T) {
	signer := NewSigningService("")

	token := signer.CreateToken(ServiceClaims{ServiceName: "service_test.go"})

	verifier := NewVerifyService("")

	claims, err := verifier.ValidateToken(token)
	assert.NoError(t, err)

	assert.False(t, signer.IsEnabled())
	assert.Equal(t, "service:service_test.go", claims.Issuer)
	assert.Equal(t, "service:service_test.go", claims.Subject)
	assert.Equal(t, jwt.ClaimStrings{"service:service_test.go"}, claims.Audience)
	//TODO: ignore the time assertion for now find a good way to mock the time so that it can be tested.
	// assert.Equal(t, jwt.NewNumericDate(time.Now()), claims.IssuedAt)
	// assert.Equal(t, jwt.NewNumericDate(time.Now().Add(time.Hour*1)), claims.ExpiresAt)
	// assert.Equal(t, jwt.NewNumericDate(time.Now().Add(time.Minute*-5)), claims.NotBefore)
}

// utility functions

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
