package api

import (
	"sync"
	"testing"
	"time"
)

func TestFeedbackEventBroadcasterBroadcastWithConcurrentCloseDoesNotPanic(t *testing.T) {
	t.Parallel()
	broadcaster := NewFeedbackEventBroadcaster()
	subscription := broadcaster.Subscribe()
	if subscription == nil {
		t.Fatalf("expected subscription")
	}

	panicSignal := make(chan interface{}, 1)
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		defer func() {
			recoveredValue := recover()
			if recoveredValue != nil {
				panicSignal <- recoveredValue
			}
		}()
		broadcaster.Broadcast(FeedbackEvent{})
	}()

	mutexReleaseCompleted := make(chan struct{})
	go func() {
		broadcaster.mutex.Lock()
		channel, exists := broadcaster.subscribers[subscription.identifier]
		if exists {
			delete(broadcaster.subscribers, subscription.identifier)
			close(channel)
		}
		broadcaster.mutex.Unlock()
		close(mutexReleaseCompleted)
	}()

	select {
	case <-mutexReleaseCompleted:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcaster mutex")
	}

	waitGroup.Wait()

	select {
	case recoveredValue := <-panicSignal:
		t.Fatalf("broadcast panicked: %v", recoveredValue)
	default:
	}
}
