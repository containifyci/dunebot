package backoff

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoff(t *testing.T) {
	tests := []struct {
		count               int
		expectedMaxDuration *time.Duration
	}{{
		count: 1,
	}, {
		count: 2,
	}, {
		count: 10,
	}, {
		count:               50,
		expectedMaxDuration: durationPtr(2 * time.Minute),
	}, {
		count:               100,
		expectedMaxDuration: durationPtr(2 * time.Minute),
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("Backoff After %d", tt.count), func(t *testing.T) {
			waiter := New(tt.count)
			if tt.expectedMaxDuration != nil {
				v, _ := waiter.(*SleepWaiter)
				assert.IsType(t, &SleepWaiter{}, waiter)
				assert.LessOrEqual(t, v.duration, *tt.expectedMaxDuration)
				assert.Equal(t, tt.count, v.count)
			} else {
				assert.IsType(t, &DummyWaiter{}, waiter)
			}
		})
	}
}

func TestSleepWaiter(t *testing.T) {
	expectedDuration := 1 * time.Second
	marginOfError := 50 * time.Millisecond // Allowable error margin for timing

	waiter := &SleepWaiter{
		count:    1,
		duration: expectedDuration,
	}

	start := time.Now()
	waiter.Wait()
	elapsed := time.Since(start)
	assert.Equal(t, 1, waiter.count)
	// Assert that the elapsed time is within the expected range
	assert.GreaterOrEqual(t, elapsed, expectedDuration, "Elapsed time should be at least the expected duration")
	assert.Less(t, elapsed, expectedDuration+marginOfError, "Elapsed time should not exceed the expected duration by much")
}

func TestDummyWaiter(t *testing.T) {
	waiter := &DummyWaiter{}
	waiter.Wait()
	assert.True(t, true, "DummyWaiter.Wait() should not block")
}

func durationPtr(dur time.Duration) *time.Duration {
	return &dur
}
