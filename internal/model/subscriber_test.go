package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testSubscriberEmail     = "USER@example.com"
	testSubscriberSiteID    = "site-123"
	testSubscriberName      = "Ada"
	testSubscriberSource    = "https://example.com/welcome"
	testSubscriberIP        = "127.0.0.1"
	testSubscriberUserAgent = "test-agent"
)

func TestNewSubscriberValidatesAndNormalizes(t *testing.T) {
	consentTime := time.Now().UTC()
	subscriber, err := NewSubscriber(SubscriberInput{
		SiteID:    "  " + testSubscriberSiteID + " ",
		Email:     testSubscriberEmail,
		Name:      testSubscriberName,
		SourceURL: testSubscriberSource,
		IP:        testSubscriberIP,
		UserAgent: testSubscriberUserAgent,
		Status:    SubscriberStatusConfirmed,
		ConsentAt: consentTime,
	})
	require.NoError(t, err)

	require.NotEmpty(t, subscriber.ID)
	require.Equal(t, testSubscriberSiteID, subscriber.SiteID)
	require.Equal(t, strings.ToLower(testSubscriberEmail), subscriber.Email)
	require.Equal(t, testSubscriberName, subscriber.Name)
	require.Equal(t, testSubscriberSource, subscriber.SourceURL)
	require.Equal(t, testSubscriberIP, subscriber.IP)
	require.Equal(t, testSubscriberUserAgent, subscriber.UserAgent)
	require.Equal(t, SubscriberStatusConfirmed, subscriber.Status)
	require.Equal(t, consentTime, subscriber.ConsentAt)
}

func TestNewSubscriberDefaultsStatusToPending(t *testing.T) {
	subscriber, err := NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  testSubscriberEmail,
	})
	require.NoError(t, err)
	require.Equal(t, SubscriberStatusPending, subscriber.Status)
}

func TestNewSubscriberRejectsInvalidSiteID(t *testing.T) {
	_, err := NewSubscriber(SubscriberInput{
		SiteID: "   ",
		Email:  testSubscriberEmail,
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberSiteID)
}

func TestNewSubscriberRejectsInvalidEmail(t *testing.T) {
	_, err := NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  "not-an-email",
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberEmail)

	longEmail := strings.Repeat("a", subscriberEmailMaxLength+1)
	_, err = NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  longEmail,
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberEmail)
}

func TestNewSubscriberRejectsInvalidStatus(t *testing.T) {
	_, err := NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  testSubscriberEmail,
		Status: "paused",
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberStatus)
}

func TestNewSubscriberRejectsOversizedFields(t *testing.T) {
	_, err := NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  testSubscriberEmail,
		Name:   strings.Repeat("n", subscriberNameMaxLength+1),
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberContact)

	_, err = NewSubscriber(SubscriberInput{
		SiteID:    testSubscriberSiteID,
		Email:     testSubscriberEmail,
		SourceURL: "https://example.com/" + strings.Repeat("s", subscriberSourceURLMaxLength),
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberContact)

	_, err = NewSubscriber(SubscriberInput{
		SiteID: testSubscriberSiteID,
		Email:  testSubscriberEmail,
		IP:     strings.Repeat("1", subscriberIPMaxLength+1),
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberContact)

	_, err = NewSubscriber(SubscriberInput{
		SiteID:    testSubscriberSiteID,
		Email:     testSubscriberEmail,
		UserAgent: strings.Repeat("u", subscriberUserAgentMaxLength+1),
	})
	require.ErrorIs(t, err, ErrInvalidSubscriberContact)
}
