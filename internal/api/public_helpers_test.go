package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	testPublicAllowedOriginPrimary   = "https://primary.example"
	testPublicAllowedOriginSecondary = "https://secondary.example"
	testPublicAllowedOriginShared    = "https://shared.example"
	testPublicSourceURLAllowed       = "https://primary.example/path"
	testPublicSourceURLBlocked       = "https://blocked.example/page"
)

type stubSubscriptionNotifier struct {
	callCount int
	callError error
}

func (notifier *stubSubscriptionNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	notifier.callCount++
	return notifier.callError
}

func TestParseAllowedOriginsDeduplicatesAndTrims(testingT *testing.T) {
	origins := parseAllowedOrigins(" https://primary.example, https://secondary.example https://PRIMARY.example ; https://secondary.example ")
	require.Equal(testingT, []string{testPublicAllowedOriginPrimary, testPublicAllowedOriginSecondary}, origins)
}

func TestParseAllowedOriginsReturnsNilForEmpty(testingT *testing.T) {
	origins := parseAllowedOrigins("   ")
	require.Nil(testingT, origins)
}

func TestParseAllowedOriginsReturnsNilForSeparators(testingT *testing.T) {
	origins := parseAllowedOrigins(" , ;  , ")
	require.Nil(testingT, origins)
}

func TestMergedAllowedOriginsHandlesEmptyInputs(testingT *testing.T) {
	require.Empty(testingT, mergedAllowedOrigins("", ""))
	require.Equal(testingT, testPublicAllowedOriginPrimary, mergedAllowedOrigins("", testPublicAllowedOriginPrimary))
	require.Equal(testingT, testPublicAllowedOriginPrimary, mergedAllowedOrigins(testPublicAllowedOriginPrimary, ""))
}

func TestMergedAllowedOriginsCombinesAndDeduplicates(testingT *testing.T) {
	merged := mergedAllowedOrigins(testPublicAllowedOriginPrimary+" "+testPublicAllowedOriginShared, testPublicAllowedOriginSecondary+" "+testPublicAllowedOriginShared)
	require.Equal(testingT, testPublicAllowedOriginPrimary+" "+testPublicAllowedOriginShared+" "+testPublicAllowedOriginSecondary, merged)
}

func TestIsOriginAllowedMatchesHeadersAndURL(testingT *testing.T) {
	allowed := testPublicAllowedOriginPrimary + " " + testPublicAllowedOriginSecondary
	require.True(testingT, isOriginAllowed("", "", "", ""))
	require.True(testingT, isOriginAllowed(allowed, testPublicAllowedOriginPrimary, "", ""))
	require.True(testingT, isOriginAllowed(allowed, "", testPublicAllowedOriginSecondary+"/page", ""))
	require.True(testingT, isOriginAllowed(allowed, "", "", testPublicAllowedOriginPrimary+"/path"))
	require.False(testingT, isOriginAllowed(allowed, "", testPublicAllowedOriginPrimary+".evil/path", ""))
	require.False(testingT, isOriginAllowed(allowed, "", "", testPublicAllowedOriginPrimary+".evil/path"))
	require.False(testingT, isOriginAllowed(allowed, "", "", "https://untrusted.example"))
}

func TestNormalizeOriginValue(testingT *testing.T) {
	require.Equal(testingT, testPublicAllowedOriginPrimary, normalizeOriginValue(testPublicAllowedOriginPrimary+"/path?x=1"))
	require.Equal(testingT, testPublicAllowedOriginPrimary, normalizeOriginValue("HTTPS://PRIMARY.EXAMPLE/path"))
	require.Empty(testingT, normalizeOriginValue("not-a-url"))
}

func TestTruncatePreservesShortAndCutsLong(testingT *testing.T) {
	require.Equal(testingT, "short", truncate("short", 10))
	require.Equal(testingT, "tool", truncate("toolong", 4))
}

func TestNormalizeFeedbackContact(testingT *testing.T) {
	testCases := []struct {
		name          string
		rawInput      string
		expectedValue string
		expectError   bool
	}{
		{name: "email", rawInput: "User@example.com", expectedValue: "user@example.com"},
		{name: "phone_with_symbols", rawInput: "+1 (415) 555-1212", expectedValue: "+14155551212"},
		{name: "phone_with_spaces", rawInput: "415 555 1212", expectedValue: "4155551212"},
		{name: "display_name_email", rawInput: "User <user@example.com>", expectError: true},
		{name: "text", rawInput: "fuck you", expectError: true},
		{name: "short_phone", rawInput: "12345", expectError: true},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			normalizedValue, normalizeErr := normalizeFeedbackContact(testCase.rawInput)
			if testCase.expectError {
				require.Error(testingT, normalizeErr)
				return
			}
			require.NoError(testingT, normalizeErr)
			require.Equal(testingT, testCase.expectedValue, normalizedValue)
		})
	}
}

func TestSubscriptionConfirmationOpenURLPrefersSourceURL(testingT *testing.T) {
	site := model.Site{AllowedOrigin: testPublicAllowedOriginPrimary}
	subscriber := model.Subscriber{SourceURL: testPublicSourceURLAllowed}
	require.Equal(testingT, testPublicSourceURLAllowed, subscriptionConfirmationOpenURL(site, subscriber))
}

func TestSubscriptionConfirmationOpenURLFallsBackToAllowedOrigin(testingT *testing.T) {
	site := model.Site{AllowedOrigin: testPublicAllowedOriginPrimary}
	subscriber := model.Subscriber{SourceURL: testPublicSourceURLBlocked}
	require.Equal(testingT, testPublicAllowedOriginPrimary, subscriptionConfirmationOpenURL(site, subscriber))
}

func TestSubscriptionConfirmationOpenURLRejectsInvalidOrigin(testingT *testing.T) {
	site := model.Site{AllowedOrigin: "mailto:info@example.com"}
	subscriber := model.Subscriber{SourceURL: "not-a-url"}
	require.Empty(testingT, subscriptionConfirmationOpenURL(site, subscriber))
}

func TestPublicHandlersIsRateLimitedCountsRequests(testingT *testing.T) {
	handlers := &PublicHandlers{
		rateWindow:                time.Hour,
		maxRequestsPerIPPerWindow: 1,
		rateCountersByIP:          make(map[string]publicRateCounter),
	}
	require.False(testingT, handlers.isRateLimited("127.0.0.1"))
	require.True(testingT, handlers.isRateLimited("127.0.0.1"))
}

func TestPublicHandlersIsRateLimitedPrunesExpiredBuckets(testingT *testing.T) {
	handlers := &PublicHandlers{
		rateWindow:                time.Hour,
		maxRequestsPerIPPerWindow: 1,
		rateCountersByIP: map[string]publicRateCounter{
			"192.0.2.10": {windowBucket: 1, count: 1},
			"192.0.2.11": {windowBucket: 2, count: 1},
		},
	}

	handlers.pruneRateCounters(2)
	require.NotContains(testingT, handlers.rateCountersByIP, "192.0.2.10")
	require.Contains(testingT, handlers.rateCountersByIP, "192.0.2.11")
}

func TestPublicHandlersIsRateLimitedCapsNewClientBuckets(testingT *testing.T) {
	handlers := &PublicHandlers{
		rateWindow:                time.Hour,
		maxRequestsPerIPPerWindow: 1,
		rateCountersByIP:          make(map[string]publicRateCounter, publicMaxRateCounterEntries),
	}
	for index := 0; index < publicMaxRateCounterEntries; index++ {
		handlers.rateCountersByIP[fmt.Sprintf("192.0.2.%d", index)] = publicRateCounter{windowBucket: time.Now().Unix() / int64(time.Hour.Seconds()), count: 1}
	}

	require.True(testingT, handlers.isRateLimited("198.51.100.1"))
}

func TestRecordSubscriptionTestEventDefaults(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		subscriptionEvents: subscriptionEvents,
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Email: "User@Example.com"}
	handlers.recordSubscriptionTestEvent(site, subscriber, "", "", "")

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, site.ID, event.SiteID)
		require.Equal(testingT, subscriber.ID, event.SubscriberID)
		require.Equal(testingT, "user@example.com", event.Email)
		require.Equal(testingT, subscriptionEventTypeSubmission, event.EventType)
		require.Equal(testingT, subscriptionEventStatusSuccess, event.Status)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestRecordSubscriptionTestEventSkipsMissingIdentifiers(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		subscriptionEvents: subscriptionEvents,
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	handlers.recordSubscriptionTestEvent(model.Site{}, model.Subscriber{}, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")

	select {
	case <-subscription.Events():
		testingT.Fatal("unexpected subscription test event")
	default:
	}
}

func TestRecordSubscriptionTestEventSkipsNilHandler(testingT *testing.T) {
	var handlers *PublicHandlers
	handlers.recordSubscriptionTestEvent(model.Site{}, model.Subscriber{}, subscriptionEventTypeSubmission, subscriptionEventStatusSuccess, "")
}

func TestApplySubscriptionNotificationSkipsUnconfirmedSubscriber(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		logger:                    zap.NewNop(),
		subscriptionEvents:        subscriptionEvents,
		subscriptionNotifications: true,
		subscriptionNotifier:      &stubSubscriptionNotifier{},
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Status: model.SubscriberStatusPending}
	handlers.applySubscriptionNotification(context.Background(), site, subscriber)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, subscriptionEventStatusSkipped, event.Status)
		require.Equal(testingT, "subscriber not confirmed", event.Error)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestApplySubscriptionNotificationSkipsWhenDisabled(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		logger:                    zap.NewNop(),
		subscriptionEvents:        subscriptionEvents,
		subscriptionNotifications: false,
		subscriptionNotifier:      &stubSubscriptionNotifier{},
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Status: model.SubscriberStatusConfirmed}
	handlers.applySubscriptionNotification(context.Background(), site, subscriber)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, subscriptionEventStatusSkipped, event.Status)
		require.Equal(testingT, "subscription notifications disabled", event.Error)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestApplySubscriptionNotificationSkipsWhenNotifierMissing(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		logger:                    zap.NewNop(),
		subscriptionEvents:        subscriptionEvents,
		subscriptionNotifications: true,
		subscriptionNotifier:      nil,
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Status: model.SubscriberStatusConfirmed}
	handlers.applySubscriptionNotification(context.Background(), site, subscriber)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, subscriptionEventStatusSkipped, event.Status)
		require.Equal(testingT, "subscription notifier unavailable", event.Error)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestApplySubscriptionNotificationRecordsNotifierError(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		logger:                    zap.NewNop(),
		subscriptionEvents:        subscriptionEvents,
		subscriptionNotifications: true,
		subscriptionNotifier:      &stubSubscriptionNotifier{callError: errors.New("notify failed")},
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Status: model.SubscriberStatusConfirmed}
	handlers.applySubscriptionNotification(context.Background(), site, subscriber)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, subscriptionEventStatusError, event.Status)
		require.Equal(testingT, "notify failed", event.Error)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestApplySubscriptionNotificationRecordsSuccess(testingT *testing.T) {
	subscriptionEvents := NewSubscriptionTestEventBroadcaster()
	testingT.Cleanup(subscriptionEvents.Close)
	handlers := &PublicHandlers{
		logger:                    zap.NewNop(),
		subscriptionEvents:        subscriptionEvents,
		subscriptionNotifications: true,
		subscriptionNotifier:      &stubSubscriptionNotifier{},
	}
	subscription := subscriptionEvents.Subscribe()
	require.NotNil(testingT, subscription)
	testingT.Cleanup(subscription.Close)

	site := model.Site{ID: "site-1"}
	subscriber := model.Subscriber{ID: "subscriber-1", Status: model.SubscriberStatusConfirmed}
	handlers.applySubscriptionNotification(context.Background(), site, subscriber)

	select {
	case event := <-subscription.Events():
		require.Equal(testingT, subscriptionEventStatusSuccess, event.Status)
	case <-time.After(time.Second):
		testingT.Fatal("expected subscription test event")
	}
}

func TestSendSubscriptionConfirmationSkipsNilHandler(testingT *testing.T) {
	var handlers *PublicHandlers
	handlers.sendSubscriptionConfirmation(context.Background(), model.Site{}, model.Subscriber{})
}
