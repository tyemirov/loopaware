package api

import (
	"sync"
	"time"
)

// SubscriptionTestEvent captures subscribe widget submission + notifier updates for dashboard previews.
type SubscriptionTestEvent struct {
	SiteID       string    `json:"site_id"`
	SubscriberID string    `json:"subscriber_id"`
	Email        string    `json:"email"`
	EventType    string    `json:"event_type"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// SubscriptionTestEventBroadcaster fans out subscription test events to SSE clients.
type SubscriptionTestEventBroadcaster struct {
	mutex        sync.Mutex
	nextID       int64
	subscribers  map[int64]chan SubscriptionTestEvent
	closed       bool
	bufferLength int
}

const subscriptionTestEventDefaultBuffer = 8

// NewSubscriptionTestEventBroadcaster constructs a broadcaster for subscribe widget test events.
func NewSubscriptionTestEventBroadcaster() *SubscriptionTestEventBroadcaster {
	return &SubscriptionTestEventBroadcaster{
		subscribers:  make(map[int64]chan SubscriptionTestEvent),
		bufferLength: subscriptionTestEventDefaultBuffer,
	}
}

// Subscribe registers a consumer for subscription test events.
func (broadcaster *SubscriptionTestEventBroadcaster) Subscribe() *SubscriptionTestEventSubscription {
	broadcaster.mutex.Lock()
	defer broadcaster.mutex.Unlock()
	if broadcaster.closed {
		return nil
	}
	subscriptionID := broadcaster.nextID
	broadcaster.nextID++
	channel := make(chan SubscriptionTestEvent, broadcaster.bufferLength)
	broadcaster.subscribers[subscriptionID] = channel
	return &SubscriptionTestEventSubscription{
		broadcaster: broadcaster,
		identifier:  subscriptionID,
		events:      channel,
		once:        sync.Once{},
	}
}

// Broadcast delivers an event to all active subscribers.
func (broadcaster *SubscriptionTestEventBroadcaster) Broadcast(event SubscriptionTestEvent) {
	if broadcaster == nil {
		return
	}
	broadcaster.mutex.Lock()
	defer broadcaster.mutex.Unlock()
	if broadcaster.closed || len(broadcaster.subscribers) == 0 {
		return
	}
	for _, channel := range broadcaster.subscribers {
		select {
		case channel <- event:
		default:
		}
	}
}

// Close shuts down the broadcaster and closes subscriber channels.
func (broadcaster *SubscriptionTestEventBroadcaster) Close() {
	broadcaster.mutex.Lock()
	if broadcaster.closed {
		broadcaster.mutex.Unlock()
		return
	}
	broadcaster.closed = true
	for id, channel := range broadcaster.subscribers {
		close(channel)
		delete(broadcaster.subscribers, id)
	}
	broadcaster.mutex.Unlock()
}

func (broadcaster *SubscriptionTestEventBroadcaster) remove(identifier int64) {
	broadcaster.mutex.Lock()
	channel, exists := broadcaster.subscribers[identifier]
	if exists {
		delete(broadcaster.subscribers, identifier)
		close(channel)
	}
	broadcaster.mutex.Unlock()
}

// SubscriptionTestEventSubscription represents one consumer of subscription test events.
type SubscriptionTestEventSubscription struct {
	broadcaster *SubscriptionTestEventBroadcaster
	identifier  int64
	events      chan SubscriptionTestEvent
	once        sync.Once
}

// Events exposes the event channel for a subscription.
func (subscription *SubscriptionTestEventSubscription) Events() <-chan SubscriptionTestEvent {
	if subscription == nil {
		return nil
	}
	return subscription.events
}

// Close unregisters the subscription.
func (subscription *SubscriptionTestEventSubscription) Close() {
	if subscription == nil {
		return
	}
	subscription.once.Do(func() {
		if subscription.broadcaster != nil {
			subscription.broadcaster.remove(subscription.identifier)
		}
	})
}
