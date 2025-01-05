package queue

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/containifyci/dunebot/pkg/config"
	"github.com/containifyci/dunebot/pkg/github"
)

type EventEntry struct {
	EventRepo    EventRepo
	Event        any
	EventHandler HandleEvent
}

type Queue interface {
	Entries() map[string][]EventEntry
	AddEvent(ctx context.Context, eventEntry EventEntry)
}

type RepositoryQueue struct {
	queues   map[string][]EventEntry
	mutex    *sync.Mutex
	shutdown chan struct{}
	wg       *sync.WaitGroup
}

type HandleEvent interface {
	HandleEvent(eventEntry EventEntry) error
}

type EventRepo struct {
	PullRequest github.PullRequest
	Repository  github.Repository
	AppConfig   *config.AppConfig
}

// NewMessageQueue creates a new MessageQueue instance.
func NewRepositoryQueue() *RepositoryQueue {
	return &RepositoryQueue{
		mutex:    &sync.Mutex{},
		wg:       &sync.WaitGroup{},
		queues:   make(map[string][]EventEntry),
		shutdown: make(chan struct{}),
	}
}

func getPrefix(eventEntry EventEntry) string {
	return fmt.Sprintf("%s/%s", eventEntry.EventRepo.Repository.GetOwner().GetLogin(), eventEntry.EventRepo.Repository.GetName())
}

func (mq *RepositoryQueue) Entries() map[string][]EventEntry {
	mq.mutex.Lock()
	defer mq.mutex.Unlock()
	return mq.queues
}

func (mq *RepositoryQueue) AddEvent(ctx context.Context, eventEntry EventEntry) {
	mq.mutex.Lock()
	prefix := getPrefix(eventEntry)
	log.Debug().Msgf("Adding event to queue: %s\n", prefix)

	defer mq.mutex.Unlock()

	if _, ok := mq.queues[getPrefix(eventEntry)]; !ok {
		mq.wg.Add(1)
		go mq.ReceiveEvent(getPrefix(eventEntry), ctx)
	}

	mq.queues[getPrefix(eventEntry)] = append(mq.queues[getPrefix(eventEntry)], eventEntry)
	log.Debug().Msgf("Event added to queue: %s, Total events in queue: %d\n", prefix, len(mq.queues[prefix]))

}

func (mq *RepositoryQueue) ReceiveEvent(prefix string, ctx context.Context) {
	defer func() {
		mq.wg.Done()
	}()

	for {
		select {
		case <-mq.shutdown:
			log.Debug().Msgf("Shutdown signal received for prefix: %s\n", prefix)

			return
		default:
			mq.mutex.Lock()
			events, ok := mq.queues[prefix]

			if ok && len(events) > 0 {
				eventEntry := events[0]
				h := eventEntry.EventHandler
				mq.queues[prefix] = events[1:]
				log.Debug().Msgf("Processing event for prefix: %s, Remaining events: %d\n", prefix, len(mq.queues[prefix]))
				log.Debug().Msgf("Handler type: %s\n", reflect.TypeOf(h).String())

				mq.mutex.Unlock()

				err := h.HandleEvent(eventEntry)
				if err != nil {
					fmt.Printf("Error processing event: %s\n", err)
				}
			} else {
				if !ok {
					log.Debug().Msgf("No events found for prefix: %s\n", prefix)
				} else if len(events) <= 0 {
					log.Debug().Msgf("No more events to process for prefix: %s\n", prefix)
				}
				delete(mq.queues, prefix)
				mq.mutex.Unlock()
				return
			}
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func (mq *RepositoryQueue) GraceFullShutdown() {
	mq.WaitUntilFinished()
	close(mq.shutdown)
}

func (mq *RepositoryQueue) WaitUntilFinished() {
	for {
		mq.mutex.Lock()
		if len(mq.queues) > 0 {
			mq.mutex.Unlock()
			time.Sleep(time.Millisecond * 50)
		} else {
			mq.mutex.Unlock()
			break
		}
	}
	mq.wg.Wait()
}
