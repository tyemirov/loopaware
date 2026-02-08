package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testSubscriptionEventSiteID  = "subscription-site"
	testSubscriptionEventID      = "subscriber-id"
	testSubscriptionEventEmail   = "subscriber@example.com"
	testSubscriptionEventType    = "submission"
	testSubscriptionEventStatus  = "success"
	testSubscriptionEventTimeout = 2 * time.Second
)

func TestSubscriptionTestEventBroadcasterSubscribeAndClose(testingT *testing.T) {
	broadcaster := NewSubscriptionTestEventBroadcaster()
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	event := SubscriptionTestEvent{
		SiteID:       testSubscriptionEventSiteID,
		SubscriberID: testSubscriptionEventID,
		Email:        testSubscriptionEventEmail,
		EventType:    testSubscriptionEventType,
		Status:       testSubscriptionEventStatus,
		Timestamp:    time.Now().UTC(),
	}

	broadcaster.Broadcast(event)

	select {
	case receivedEvent := <-subscription.Events():
		require.Equal(testingT, event.SiteID, receivedEvent.SiteID)
		require.Equal(testingT, event.SubscriberID, receivedEvent.SubscriberID)
	case <-time.After(testSubscriptionEventTimeout):
		testingT.Fatal("timeout waiting for subscription event")
	}

	subscription.Close()

	select {
	case _, open := <-subscription.Events():
		require.False(testingT, open)
	case <-time.After(testSubscriptionEventTimeout):
		testingT.Fatal("timeout waiting for subscription close")
	}

	broadcaster.Close()
	require.Nil(testingT, broadcaster.Subscribe())

	var nilSubscription *SubscriptionTestEventSubscription
	require.Nil(testingT, nilSubscription.Events())
	nilSubscription.Close()
}

func TestSubscriptionTestEventBroadcasterBroadcastHandlesNil(testingT *testing.T) {
	var broadcaster *SubscriptionTestEventBroadcaster
	broadcaster.Broadcast(SubscriptionTestEvent{})
}
