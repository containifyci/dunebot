package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/containifyci/dunebot/pkg/github"
	"github.com/stretchr/testify/assert"
)

type HandlerPullRequest struct {
	prs   []EventEntry
	mutex *sync.Mutex
}

func (h *HandlerPullRequest) HandleEvent(eventEntry EventEntry) error {
	defer h.mutex.Unlock()
	h.mutex.Lock()
	h.prs = append(h.prs, eventEntry)
	return nil
}

func TestNewRepositoryQueue(t *testing.T) {
	t.Parallel()

	handler := &HandlerPullRequest{prs: []EventEntry{}, mutex: &sync.Mutex{}}
	queue := NewRepositoryQueue()
	ctx := context.Background()

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(1),
			},
			Repository: github.Repository{
				Name: github.String("test"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(1),
			},
			Repository: github.Repository{
				Name: github.String("test2"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(2),
			},
			Repository: github.Repository{
				Name: github.String("test"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(2),
			},
			Repository: github.Repository{
				Name: github.String("test2"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(3),
			},
			Repository: github.Repository{
				Name: github.String("test"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	queue.AddEvent(ctx, EventEntry{
		EventRepo: EventRepo{
			PullRequest: github.PullRequest{
				Number: github.Int(3),
			},
			Repository: github.Repository{
				Name: github.String("test2"),
				Owner: &github.User{
					Login: github.String("containifyci"),
				},
			},
		},
		Event:        github.PullRequestEvent{},
		EventHandler: handler,
	})

	fmt.Println("Waiting for graceful shutdown")
	queue.GraceFullShutdown()
	fmt.Println("Shutdown complete")
	assert.Equal(t, 6, len(handler.prs))
	assert.Equal(t, 0, len(queue.Entries()))
}
