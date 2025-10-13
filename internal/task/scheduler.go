package task

import (
	"context"
	"sync"
	"time"
)

type RunnerFunc func(context.Context)

type Scheduler struct {
	interval     time.Duration
	runner       RunnerFunc
	trigger      chan struct{}
	controlMutex sync.Mutex
	cancel       context.CancelFunc
	done         chan struct{}
}

func NewScheduler(interval time.Duration, runner RunnerFunc) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{
		interval: interval,
		runner:   runner,
		trigger:  make(chan struct{}, 1),
	}
}

func (scheduler *Scheduler) Start(ctx context.Context) {
	if scheduler == nil || scheduler.runner == nil {
		return
	}
	scheduler.controlMutex.Lock()
	if scheduler.cancel != nil {
		scheduler.controlMutex.Unlock()
		return
	}
	runtimeCtx, cancel := context.WithCancel(ctx)
	scheduler.cancel = cancel
	scheduler.done = make(chan struct{})
	scheduler.controlMutex.Unlock()

	go scheduler.loop(runtimeCtx)
}

func (scheduler *Scheduler) Trigger() {
	if scheduler == nil {
		return
	}
	select {
	case scheduler.trigger <- struct{}{}:
	default:
	}
}

func (scheduler *Scheduler) Stop() {
	if scheduler == nil {
		return
	}
	scheduler.controlMutex.Lock()
	cancel := scheduler.cancel
	done := scheduler.done
	scheduler.cancel = nil
	scheduler.done = nil
	scheduler.controlMutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (scheduler *Scheduler) loop(ctx context.Context) {
	timer := time.NewTimer(scheduler.interval)
	done := scheduler.done
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}()
	defer func() {
		scheduler.controlMutex.Lock()
		if done != nil {
			close(done)
		}
		scheduler.controlMutex.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-scheduler.trigger:
			scheduler.run(ctx)
		case <-timer.C:
			scheduler.run(ctx)
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(scheduler.interval)
	}
}

func (scheduler *Scheduler) run(ctx context.Context) {
	if scheduler.runner == nil {
		return
	}
	scheduler.runner(ctx)
}
