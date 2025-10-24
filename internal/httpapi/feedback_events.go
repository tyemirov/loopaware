package httpapi

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

// FeedbackEvent represents a feedback creation notification for SSE clients.
type FeedbackEvent struct {
	SiteID        string
	FeedbackID    string
	CreatedAt     time.Time
	FeedbackCount int64
}

// FeedbackEventBroadcaster fan-outs feedback events to subscribed clients.
type FeedbackEventBroadcaster struct {
	mutex        sync.Mutex
	nextID       int64
	subscribers  map[int64]chan FeedbackEvent
	closed       bool
	bufferLength int
}

const feedbackEventDefaultBuffer = 8

// NewFeedbackEventBroadcaster constructs a broadcaster for feedback events.
func NewFeedbackEventBroadcaster() *FeedbackEventBroadcaster {
	return &FeedbackEventBroadcaster{
		subscribers:  make(map[int64]chan FeedbackEvent),
		bufferLength: feedbackEventDefaultBuffer,
	}
}

// Subscribe returns a subscription that streams feedback events.
func (broadcaster *FeedbackEventBroadcaster) Subscribe() *FeedbackEventSubscription {
	broadcaster.mutex.Lock()
	defer broadcaster.mutex.Unlock()
	if broadcaster.closed {
		return nil
	}
	subscriptionID := broadcaster.nextID
	broadcaster.nextID++
	eventChannel := make(chan FeedbackEvent, broadcaster.bufferLength)
	broadcaster.subscribers[subscriptionID] = eventChannel
	return &FeedbackEventSubscription{
		broadcaster: broadcaster,
		identifier:  subscriptionID,
		events:      eventChannel,
		once:        sync.Once{},
	}
}

// Broadcast delivers the event to all active subscribers.
func (broadcaster *FeedbackEventBroadcaster) Broadcast(event FeedbackEvent) {
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

// Close stops the broadcaster and closes all subscriber channels.
func (broadcaster *FeedbackEventBroadcaster) Close() {
	broadcaster.mutex.Lock()
	if broadcaster.closed {
		broadcaster.mutex.Unlock()
		return
	}
	broadcaster.closed = true
	for identifier, channel := range broadcaster.subscribers {
		close(channel)
		delete(broadcaster.subscribers, identifier)
	}
	broadcaster.mutex.Unlock()
}

func (broadcaster *FeedbackEventBroadcaster) remove(identifier int64) {
	broadcaster.mutex.Lock()
	channel, exists := broadcaster.subscribers[identifier]
	if exists {
		delete(broadcaster.subscribers, identifier)
		close(channel)
	}
	broadcaster.mutex.Unlock()
}

// FeedbackEventSubscription represents a single subscriber to feedback events.
type FeedbackEventSubscription struct {
	broadcaster *FeedbackEventBroadcaster
	identifier  int64
	events      chan FeedbackEvent
	once        sync.Once
}

// Events exposes the receive-only event channel.
func (subscription *FeedbackEventSubscription) Events() <-chan FeedbackEvent {
	if subscription == nil {
		return nil
	}
	return subscription.events
}

// Close unregisters the subscription and closes its channel.
func (subscription *FeedbackEventSubscription) Close() {
	if subscription == nil {
		return
	}
	subscription.once.Do(func() {
		if subscription.broadcaster != nil {
			subscription.broadcaster.remove(subscription.identifier)
		}
	})
}

func broadcastFeedbackEvent(database *gorm.DB, logger *zap.Logger, broadcaster *FeedbackEventBroadcaster, ctx context.Context, feedback model.Feedback) {
	if broadcaster == nil {
		return
	}
	timestamp := feedback.CreatedAt
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}
	var totalCount int64
	if database != nil {
		queryErr := database.WithContext(ctx).Model(&model.Feedback{}).
			Where("site_id = ?", feedback.SiteID).
			Count(&totalCount).Error
		if queryErr != nil {
			if logger != nil {
				logger.Debug("count_feedback_event_failed", zap.Error(queryErr))
			}
			totalCount = 0
		}
	}
	broadcaster.Broadcast(FeedbackEvent{
		SiteID:        feedback.SiteID,
		FeedbackID:    feedback.ID,
		CreatedAt:     timestamp,
		FeedbackCount: totalCount,
	})
}
