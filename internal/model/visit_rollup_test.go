package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewSiteVisitRollupValidatesInput(t *testing.T) {
	_, err := NewSiteVisitRollup("", time.Now(), 10, 5)
	require.Error(t, err)

	_, err = NewSiteVisitRollup("site", time.Time{}, 10, 5)
	require.Error(t, err)

	_, err = NewSiteVisitRollup("site", time.Now(), -1, 5)
	require.Error(t, err)
}

func TestNewSiteVisitRollupNormalizesDate(t *testing.T) {
	date := time.Date(2024, 1, 2, 15, 30, 0, 0, time.UTC)
	rollup, err := NewSiteVisitRollup("site", date, 10, 5)
	require.NoError(t, err)
	require.Equal(t, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), rollup.Date)
	require.Equal(t, int64(10), rollup.PageViews)
	require.Equal(t, int64(5), rollup.UniqueVisitors)
	require.Equal(t, "site", rollup.SiteID)
	require.NotEmpty(t, rollup.ID)
}
