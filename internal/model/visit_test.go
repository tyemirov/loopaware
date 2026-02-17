package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testVisitBaseURL = "https://example.com"
)

func TestNewSiteVisitValidatesAndNormalizes(testingT *testing.T) {
	now := time.Now().UTC()
	visit, err := NewSiteVisit(SiteVisitInput{
		SiteID:    "site-1",
		URL:       "https://example.com/welcome?utm=1#hash",
		VisitorID: "12345678-1234-1234-1234-123456789abc",
		IP:        "127.0.0.1",
		UserAgent: "ua",
		Referrer:  "https://ref.example.com/page",
		Occurred:  now,
	})
	require.NoError(testingT, err)
	require.Equal(testingT, "site-1", visit.SiteID)
	require.Equal(testingT, "https://example.com/welcome?utm=1", visit.URL)
	require.Equal(testingT, "/welcome", visit.Path)
	require.Equal(testingT, "12345678-1234-1234-1234-123456789abc", visit.VisitorID)
	require.Equal(testingT, "127.0.0.1", visit.IP)
	require.Equal(testingT, "ua", visit.UserAgent)
	require.Equal(testingT, "https://ref.example.com/page", visit.Referrer)
	require.False(testingT, visit.IsBot)
	require.Equal(testingT, VisitStatusRecorded, visit.Status)
	require.Equal(testingT, now, visit.OccurredAt)
}

func TestNewSiteVisitStoresBotFlag(testingT *testing.T) {
	visit, err := NewSiteVisit(SiteVisitInput{
		SiteID: "site-1",
		URL:    "https://example.com/welcome",
		IsBot:  true,
	})
	require.NoError(testingT, err)
	require.True(testingT, visit.IsBot)
}

func TestNewSiteVisitRequiresValidInputs(testingT *testing.T) {
	_, err := NewSiteVisit(SiteVisitInput{})
	require.ErrorIs(testingT, err, ErrInvalidVisitSiteID)

	_, err = NewSiteVisit(SiteVisitInput{SiteID: "s", URL: "not a url"})
	require.ErrorIs(testingT, err, ErrInvalidVisitURL)

	_, err = NewSiteVisit(SiteVisitInput{
		SiteID:    "s",
		URL:       "https://example.com",
		VisitorID: "invalid",
	})
	require.ErrorIs(testingT, err, ErrInvalidVisitID)
}

func TestNormalizeVisitURLRejectsBlank(testingT *testing.T) {
	normalizedURL, normalizedPath, normalizeErr := normalizeVisitURL("   ")
	require.ErrorIs(testingT, normalizeErr, ErrInvalidVisitURL)
	require.Empty(testingT, normalizedURL)
	require.Empty(testingT, normalizedPath)
}

func TestNormalizeVisitURLTruncatesLongFields(testingT *testing.T) {
	longPath := "/" + strings.Repeat("a", visitPathMaxLength+20)
	queryString := strings.Repeat("b", visitURLMaxLength)
	rawURL := testVisitBaseURL + longPath + "?" + queryString

	normalizedURL, normalizedPath, normalizeErr := normalizeVisitURL(rawURL)
	require.NoError(testingT, normalizeErr)
	require.Len(testingT, normalizedURL, visitURLMaxLength)
	require.Len(testingT, normalizedPath, visitPathMaxLength)
}

func TestTruncateStringHandlesLimits(testingT *testing.T) {
	shortValue := "short"
	require.Equal(testingT, shortValue, truncateString(shortValue, len(shortValue)))

	longValue := strings.Repeat("c", 10)
	truncatedValue := truncateString(longValue, 3)
	require.Equal(testingT, longValue[:3], truncatedValue)
}
