package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testSchedulerInterval = 10 * time.Millisecond
	testSchedulerTimeout  = 2 * time.Second
)

func TestNewSchedulerDefaultsInterval(testingT *testing.T) {
	scheduler := NewScheduler(0, func(context.Context) {})
	require.Equal(testingT, time.Minute, scheduler.interval)
}

func TestSchedulerRunsOnTrigger(testingT *testing.T) {
	var runCount int64
	runner := func(context.Context) {
		atomic.AddInt64(&runCount, 1)
	}
	scheduler := NewScheduler(testSchedulerInterval, runner)
	runtimeContext, cancel := context.WithCancel(context.Background())
	testingT.Cleanup(cancel)

	scheduler.Start(runtimeContext)
	scheduler.Trigger()

	require.Eventually(testingT, func() bool {
		return atomic.LoadInt64(&runCount) > 0
	}, testSchedulerTimeout, testSchedulerInterval)

	scheduler.Stop()
	require.Nil(testingT, scheduler.cancel)
}

func TestSchedulerHandlesNilReceiver(testingT *testing.T) {
	var scheduler *Scheduler
	scheduler.Start(context.Background())
	scheduler.Trigger()
	scheduler.Stop()
}

func TestSchedulerSkipsStartWhenRunnerMissing(testingT *testing.T) {
	scheduler := NewScheduler(testSchedulerInterval, nil)
	scheduler.Start(context.Background())
	require.Nil(testingT, scheduler.cancel)
}

func TestSchedulerStartIsIdempotent(testingT *testing.T) {
	scheduler := NewScheduler(testSchedulerInterval, func(context.Context) {})
	scheduler.Start(context.Background())
	doneAfterStart := scheduler.done
	require.NotNil(testingT, scheduler.cancel)
	scheduler.Start(context.Background())
	require.Equal(testingT, doneAfterStart, scheduler.done)
	scheduler.Stop()
}

func TestSchedulerRunNoopWithNilRunner(testingT *testing.T) {
	scheduler := &Scheduler{}
	scheduler.run(context.Background())
}
