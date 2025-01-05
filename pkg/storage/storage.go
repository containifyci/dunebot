package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/containifyci/dunebot/pkg/proto"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type (
	Installation = proto.Installation
	CustomToken = proto.CustomToken
	SingleToken = proto.SingleToken
	RevokeMessage = proto.RevokeMessage
	RevokeMessage_Error = proto.RevokeMessage_Error
	UnimplementedTokenServer = proto.UnimplementedTokenServer
)
var RegisterTokenServer = proto.RegisterTokenServer

const (
	DefaultK8sSecret = "dunebot-token-storage"
)

type K8sStorage struct {
	clientset kubernetes.Interface
	namespace string
}

type FileStorage struct {
	file string
}

type K8sConfigReadder func() (*rest.Config, error)

type Storage interface {
	Load() (map[int64]*Installation, error)
	Save(tokens map[int64]*Installation) error
}

func (s *TokenService) SyncWithError(ctx context.Context, period time.Duration, errCh chan<- error) {
	if period == 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(period)
		for {
			select {
			case <-ticker.C:
				err := s.Save()
				if err != nil {
					//TODO find a better way to log this
					fmt.Printf("error saving tokens %s\n", err)
					if errCh != nil {
						select {
						case errCh <- err:
						default:
							fmt.Printf("error channel is blocked cant receive %s\n", err)
							// If the channel is blocked, just continue without sending the error
						}
					}
				}
			case <-ctx.Done():
				close(errCh)
				return
			}
		}
	}()
}

func (s *TokenService) Save() error {
	if s.storage == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.storage.Save(s.tokens)
}

func (s *TokenService) Load() error {
	if s.storage == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.storage.Load()
	if err != nil {
		return err
	}
	s.tokens = data
	return nil
}

func NewFileStorage(file string) *FileStorage {
	return &FileStorage{
		file: file,
	}
}

func (s *FileStorage) Load() (map[int64]*Installation, error) {
	data, err := os.ReadFile(s.file)
	if err != nil {
		return nil, err
	}

	var tokens map[int64]*Installation
	err = json.Unmarshal(data, &tokens)
	return tokens, err
}

func (s *FileStorage) Save(tokens map[int64]*Installation) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0644)
}

func InClusterConfig() K8sConfigReadder {
	return func() (*rest.Config, error) {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		return config, nil
	}
}

func NewK8sStorage(namespace string, k8sConfig K8sConfigReadder) (*K8sStorage, error) {
	config, err := k8sConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sStorage{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

func (s *K8sStorage) GetSecret() (*v1.Secret, error) {
	ctx := context.Background()
	api := s.clientset.CoreV1()
	secret, err := api.Secrets(s.namespace).Get(ctx, DefaultK8sSecret, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	} else if errors.IsNotFound(err) {
		return nil, nil
	}
	return secret, nil
}

func (s *K8sStorage) Load() (map[int64]*Installation, error) {
	secret, err := s.GetSecret()
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return make(map[int64]*Installation, 0), nil
	}

	secretData := secret.Data["tokens"]
	if len(secretData) > 0 {
		var tokens map[int64]*Installation
		err = json.Unmarshal(secretData, &tokens)
		return tokens, err
	}
	return make(map[int64]*Installation, 0), nil
}

func (s *K8sStorage) Save(tokens map[int64]*Installation) error {
	data, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	secret, err := s.GetSecret()
	if err != nil {
		return err
	}

	ctx := context.Background()
	api := s.clientset.CoreV1()

	//create if the secret does not exist
	if secret == nil {
		_, err = api.Secrets(s.namespace).Create(ctx, &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: DefaultK8sSecret,
			},
			Data: map[string][]byte{
				"tokens": []byte(data),
			},
		}, metav1.CreateOptions{})

		if err != nil {
			return err
		}
	} else { //update the secret if it exists already
		retryErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			currentSecret, err := api.Secrets(s.namespace).Get(context.TODO(), DefaultK8sSecret, metav1.GetOptions{})
			if err != nil {
				return err
			}

			currentSecret.Data["date"] = []byte(time.Now().Format(time.RFC3339))
			currentSecret.Data["tokens"] = []byte(data)

			_, updateErr := api.Secrets(s.namespace).Update(context.TODO(), currentSecret, metav1.UpdateOptions{})
			return updateErr
		})

		if retryErr != nil {
			return err
		}
	}

	return nil
}
