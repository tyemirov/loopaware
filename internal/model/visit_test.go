package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewSiteVisitValidatesAndNormalizes(t *testing.T) {
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
	require.NoError(t, err)
	require.Equal(t, "site-1", visit.SiteID)
	require.Equal(t, "https://example.com/welcome?utm=1", visit.URL)
	require.Equal(t, "/welcome", visit.Path)
	require.Equal(t, "12345678-1234-1234-1234-123456789abc", visit.VisitorID)
	require.Equal(t, "127.0.0.1", visit.IP)
	require.Equal(t, "ua", visit.UserAgent)
	require.Equal(t, "https://ref.example.com/page", visit.Referrer)
	require.Equal(t, VisitStatusRecorded, visit.Status)
	require.Equal(t, now, visit.OccurredAt)
}

func TestNewSiteVisitRequiresValidInputs(t *testing.T) {
	_, err := NewSiteVisit(SiteVisitInput{})
	require.ErrorIs(t, err, ErrInvalidVisitSiteID)

	_, err = NewSiteVisit(SiteVisitInput{SiteID: "s", URL: "not a url"})
	require.ErrorIs(t, err, ErrInvalidVisitURL)

	_, err = NewSiteVisit(SiteVisitInput{
		SiteID:    "s",
		URL:       "https://example.com",
		VisitorID: "invalid",
	})
	require.ErrorIs(t, err, ErrInvalidVisitID)
}
