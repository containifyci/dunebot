package backoff

import (
	"log/slog"
	"time"

	"golang.org/x/exp/rand"
)

type Waiter interface {
	Wait()
}

type SleepWaiter struct {
	count    int
	duration time.Duration
}

type DummyWaiter struct{}

func New(count int) Waiter {
	if count%50 != 0 {
		return &DummyWaiter{}
	}

	maxBackoff := 2 * time.Minute
	baseDelay := 10 * time.Second

	backoffDuration := time.Duration(rand.Int63n(int64(maxBackoff-baseDelay))) + baseDelay
	return &SleepWaiter{
		count:    count,
		duration: backoffDuration,
	}
}

func (w *SleepWaiter) Wait() {
	slog.Debug("Backoff to relax processing", "prs", w.count, "waitfor", w.duration)
	time.Sleep(w.duration)
}

func (w *DummyWaiter) Wait() { /*do nothing*/ }
