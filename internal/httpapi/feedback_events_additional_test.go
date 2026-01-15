package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testFeedbackSiteIdentifier   = "feedback-site"
	testFeedbackOwnerAddress     = "owner@example.com"
	testFeedbackMessageBody      = "Hello from feedback"
	testFeedbackEventTimeout     = 2 * time.Second
	testFeedbackEventPoll        = 10 * time.Millisecond
	testFeedbackIdentifier       = "feedback-1"
	testFeedbackBroadcasterEmail = "creator@example.com"
)

func TestFeedbackEventBroadcasterSubscribeAndClose(testingT *testing.T) {
	broadcaster := NewFeedbackEventBroadcaster()
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	event := FeedbackEvent{
		SiteID:        testFeedbackSiteIdentifier,
		FeedbackID:    testFeedbackIdentifier,
		CreatedAt:     time.Now().UTC(),
		FeedbackCount: 1,
	}

	broadcaster.Broadcast(event)

	select {
	case receivedEvent := <-subscription.Events():
		require.Equal(testingT, event.SiteID, receivedEvent.SiteID)
		require.Equal(testingT, event.FeedbackID, receivedEvent.FeedbackID)
	case <-time.After(testFeedbackEventTimeout):
		testingT.Fatal("timeout waiting for feedback event")
	}

	subscription.Close()

	select {
	case _, open := <-subscription.Events():
		require.False(testingT, open)
	case <-time.After(testFeedbackEventTimeout):
		testingT.Fatal("timeout waiting for subscription close")
	}

	broadcaster.Close()
	require.Nil(testingT, broadcaster.Subscribe())

	var nilSubscription *FeedbackEventSubscription
	require.Nil(testingT, nilSubscription.Events())
	nilSubscription.Close()
}

func TestBroadcastFeedbackEventIncludesCount(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	site := model.Site{
		ID:            testFeedbackSiteIdentifier,
		Name:          "Feedback Site",
		OwnerEmail:    testFeedbackOwnerAddress,
		CreatorEmail:  testFeedbackBroadcasterEmail,
		AllowedOrigin: "https://example.com",
	}
	require.NoError(testingT, database.Create(&site).Error)

	feedback := model.Feedback{
		ID:      testFeedbackIdentifier,
		SiteID:  site.ID,
		Contact: testFeedbackOwnerAddress,
		Message: testFeedbackMessageBody,
	}
	require.NoError(testingT, database.Create(&feedback).Error)

	broadcaster := NewFeedbackEventBroadcaster()
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	broadcastFeedbackEvent(database, zap.NewNop(), broadcaster, context.Background(), model.Feedback{SiteID: site.ID})

	select {
	case receivedEvent := <-subscription.Events():
		require.Equal(testingT, site.ID, receivedEvent.SiteID)
		require.Equal(testingT, int64(1), receivedEvent.FeedbackCount)
		require.False(testingT, receivedEvent.CreatedAt.IsZero())
	case <-time.After(testFeedbackEventTimeout):
		testingT.Fatal("timeout waiting for feedback broadcast")
	}

	subscription.Close()
	broadcaster.Close()
}
