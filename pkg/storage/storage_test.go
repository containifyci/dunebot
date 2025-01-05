package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type cxtTestKey struct{}

type TestContext struct {
	k8sStorage  K8sStorage
	fileStorage FileStorage
}

var ctx context.Context

/*
Setup the storages with dummy data for testing
*/
func setup() {
	k8sStorage := K8sStorage{
		namespace: "test",
		clientset: testclient.NewSimpleClientset(),
	}
	fileStorage := FileStorage{
		file: os.TempDir() + "/dunebot-token-storage.json",
	}

	ctx = context.WithValue(context.Background(), cxtTestKey{}, TestContext{k8sStorage: k8sStorage, fileStorage: fileStorage})
	installations := map[int64]*Installation{
		1: {
			InstallationId: 1,
			Tokens: []*CustomToken{{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				Expiry:       timestamppb.Now(),
				TokenType:    "token-type",
				User:         "user",
			},
			},
		}}

	err := k8sStorage.Save(installations)
	if err != nil {
		panic(err)
	}
	err = fileStorage.Save(installations)
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestNewK8sStorage(t *testing.T) {
	storage, err := NewK8sStorage("test", func() (*rest.Config, error) {
		return &rest.Config{}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "test", storage.namespace)
}

//TODO add error handling test cases

func TestK8sStorage_Load(t *testing.T) {
	ctx := ctx.Value(cxtTestKey{}).(TestContext)
	data, err := ctx.k8sStorage.Load()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(data))
	assert.Equal(t, int64(1), data[1].InstallationId)
}

func TestK8sStorage_Save(t *testing.T) {
	clientset := testclient.NewSimpleClientset()
	storage := K8sStorage{
		namespace: "test",
		clientset: clientset,
	}
	installations := map[int64]*Installation{
		2: {
			InstallationId: 2,
			Tokens: []*CustomToken{{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				Expiry:       timestamppb.Now(),
				TokenType:    "token-type",
				User:         "user",
			},
			},
		}}
	//First time save will create the k8s secret
	err := storage.Save(installations)
	assert.NoError(t, err)
	//Second time save will update the k8s secret
	err = storage.Save(installations)
	assert.NoError(t, err)
}

func TestFileStorage_Save(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "prefix")
	assert.NoError(t, err)

	storage := FileStorage{
		file: file.Name(),
	}
	installations := map[int64]*Installation{
		2: {
			InstallationId: 2,
			Tokens: []*CustomToken{{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				Expiry:       timestamppb.Now(),
				TokenType:    "token-type",
				User:         "user",
			},
			},
		}}
	err = storage.Save(installations)
	assert.NoError(t, err)
}

func TestFileStorage_Load(t *testing.T) {
	ctx := ctx.Value(cxtTestKey{}).(TestContext)
	data, err := ctx.fileStorage.Load()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(data))
	assert.Equal(t, int64(1), data[1].InstallationId)
}

func TestTokenStorage_Load(t *testing.T) {
	ctx := ctx.Value(cxtTestKey{}).(TestContext)
	storage := NewTokenService(Config{
		StorageFile: ctx.fileStorage.file,
	})
	err := storage.Load()

	data := storage.tokens

	assert.NoError(t, err)
	assert.Equal(t, 1, len(data))
	assert.Equal(t, int64(1), data[1].InstallationId)
}

func TestTokenStorage_Save(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "prefix")
	assert.NoError(t, err)

	storage := NewTokenService(Config{
		StorageFile: file.Name(),
	})

	storage.tokens = map[int64]*Installation{
		2: {
			InstallationId: 2,
			Tokens: []*CustomToken{{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				Expiry:       timestamppb.Now(),
				TokenType:    "token-type",
				User:         "user",
			},
			},
		}}
	err = storage.Save()
	assert.NoError(t, err)
}

func TestTokenStorage_SyncWithError(t *testing.T) {
	storage := NewTokenService(Config{
		StorageFile: "./test_/adsasdsadsa",
	})

	storage.tokens = map[int64]*Installation{
		2: {
			InstallationId: 2,
			Tokens: []*CustomToken{{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				Expiry:       timestamppb.Now(),
				TokenType:    "token-type",
				User:         "user",
			},
			},
		}}
	ctx2, cancel := context.WithCancel(ctx)
	// Create a buffered channel to communicate errors from the goroutine
	errCh := make(chan error, 1)
	storage.SyncWithError(ctx2, 1 * time.Second, errCh)
	time.Sleep(2 * time.Second)
	cancel()
	// Wait for the goroutine to finish
	select {
	case err := <-errCh:
		assert.ErrorContains(t, err, "open ./test_/adsasdsadsa: no such file or directory")
	case <-time.After(5 * time.Second):
		assert.Fail(t, "expected error")
	}
}
