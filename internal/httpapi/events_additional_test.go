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

func TestFeedbackEventBroadcasterCloseClosesSubscribers(testingT *testing.T) {
	broadcaster := NewFeedbackEventBroadcaster()
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	broadcaster.Close()

	_, open := <-subscription.Events()
	require.False(testingT, open)

	broadcaster.Close()
}

func TestSubscriptionTestEventBroadcasterCloseClosesSubscribers(testingT *testing.T) {
	broadcaster := NewSubscriptionTestEventBroadcaster()
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	broadcaster.Close()

	_, open := <-subscription.Events()
	require.False(testingT, open)

	broadcaster.Close()
}

func TestBroadcastFeedbackEventUsesNowOnMissingTimestamp(testingT *testing.T) {
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	sqlDatabase, sqlErr := database.DB()
	require.NoError(testingT, sqlErr)
	require.NoError(testingT, sqlDatabase.Close())

	broadcaster := NewFeedbackEventBroadcaster()
	testingT.Cleanup(broadcaster.Close)
	subscription := broadcaster.Subscribe()
	require.NotNil(testingT, subscription)

	feedback := model.Feedback{ID: "feedback-id", SiteID: "site-id"}
	broadcastFeedbackEvent(database, zap.NewNop(), broadcaster, context.Background(), feedback)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, feedback.SiteID, event.SiteID)
		require.Equal(testingT, feedback.ID, event.FeedbackID)
		require.Equal(testingT, int64(0), event.FeedbackCount)
		require.False(testingT, event.CreatedAt.IsZero())
	case <-time.After(time.Second):
		testingT.Fatalf("expected feedback event")
	}
}
