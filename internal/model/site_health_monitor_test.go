package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewSiteHealthMonitorValidatesInput(testingT *testing.T) {
	validInput := SiteHealthMonitorInput{
		SiteID:           "site-id",
		Enabled:          true,
		TargetURL:        "https://health.example.com/status",
		IntervalSeconds:  DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: DefaultSiteHealthFailureThreshold,
		RecipientEmail:   "owner@example.com",
		ReferenceTime:    time.Date(2026, time.June, 25, 19, 0, 0, 0, time.UTC),
	}

	testCases := []struct {
		name   string
		mutate func(*SiteHealthMonitorInput)
	}{
		{name: "missing site", mutate: func(input *SiteHealthMonitorInput) { input.SiteID = " " }},
		{name: "private target", mutate: func(input *SiteHealthMonitorInput) { input.TargetURL = "http://127.0.0.1/healthz" }},
		{name: "invalid scheme", mutate: func(input *SiteHealthMonitorInput) { input.TargetURL = "ftp://health.example.com/status" }},
		{name: "invalid interval", mutate: func(input *SiteHealthMonitorInput) { input.IntervalSeconds = MinSiteHealthIntervalSeconds - 1 }},
		{name: "invalid timeout", mutate: func(input *SiteHealthMonitorInput) { input.TimeoutSeconds = MaxSiteHealthTimeoutSeconds + 1 }},
		{name: "invalid threshold", mutate: func(input *SiteHealthMonitorInput) { input.FailureThreshold = MaxSiteHealthFailureThreshold + 1 }},
		{name: "invalid recipient", mutate: func(input *SiteHealthMonitorInput) { input.RecipientEmail = "invalid" }},
		{name: "invalid recipient mode", mutate: func(input *SiteHealthMonitorInput) { input.RecipientMode = "everyone" }},
		{name: "missing selected recipient", mutate: func(input *SiteHealthMonitorInput) { input.RecipientMode = SiteRecipientModeSelected }},
		{name: "invalid selected recipient", mutate: func(input *SiteHealthMonitorInput) {
			input.RecipientMode = SiteRecipientModeSelected
			input.RecipientEmails = []string{"invalid"}
		}},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			input := validInput
			testCase.mutate(&input)
			_, monitorErr := NewSiteHealthMonitor(input)
			require.Error(testingT, monitorErr)
			require.True(testingT, errors.Is(monitorErr, ErrInvalidSiteHealthMonitor))
		})
	}
}

func TestNewSiteHealthMonitorStoresNormalizedConfiguration(testingT *testing.T) {
	referenceAt := time.Date(2026, time.June, 25, 19, 30, 0, 0, time.UTC)
	monitor, monitorErr := NewSiteHealthMonitor(SiteHealthMonitorInput{
		SiteID:           " site-id ",
		Enabled:          true,
		TargetURL:        "https://health.example.com/status#ignored",
		IntervalSeconds:  600,
		TimeoutSeconds:   15,
		FailureThreshold: 3,
		RecipientEmail:   "OWNER@example.com",
		RecipientMode:    SiteRecipientModeSelected,
		RecipientEmails:  []string{"TEAM@example.com", "team@example.com"},
		ReferenceTime:    referenceAt,
	})
	require.NoError(testingT, monitorErr)
	require.NotEmpty(testingT, monitor.ID)
	require.Equal(testingT, "site-id", monitor.SiteID)
	require.True(testingT, monitor.Enabled)
	require.Equal(testingT, "https://health.example.com/status", monitor.TargetURL)
	require.Equal(testingT, 600, monitor.IntervalSeconds)
	require.Equal(testingT, 15, monitor.TimeoutSeconds)
	require.Equal(testingT, 3, monitor.FailureThreshold)
	require.Equal(testingT, "owner@example.com", monitor.RecipientEmail)
	require.Equal(testingT, SiteRecipientModeSelected, monitor.RecipientMode)
	require.Equal(testingT, []string{"team@example.com"}, monitor.SelectedRecipientEmails())
	require.Equal(testingT, SiteHealthStatusUnknown, monitor.Status)
	require.Equal(testingT, referenceAt, monitor.NextCheckAt)
	require.Equal(testingT, 10*time.Minute, monitor.CheckInterval())
	require.Equal(testingT, 15*time.Second, monitor.CheckTimeout())
}

func TestSiteHealthMonitorDefaultsAndTruncation(testingT *testing.T) {
	monitor, monitorErr := NewSiteHealthMonitor(SiteHealthMonitorInput{
		SiteID:         "site-id",
		TargetURL:      "https://health.example.com/status",
		RecipientEmail: "owner@example.com",
	})
	require.NoError(testingT, monitorErr)
	require.False(testingT, monitor.Enabled)
	require.Equal(testingT, DefaultSiteHealthIntervalSeconds, monitor.IntervalSeconds)
	require.Equal(testingT, DefaultSiteHealthTimeoutSeconds, monitor.TimeoutSeconds)
	require.Equal(testingT, DefaultSiteHealthFailureThreshold, monitor.FailureThreshold)
	require.Equal(testingT, SiteRecipientModeManager, monitor.RecipientModeValue())
	require.Empty(testingT, monitor.SelectedRecipientEmails())
	require.True(testingT, monitor.NextCheckAt.IsZero())

	require.Len(testingT, TruncateSiteHealthErrorCode(stringWithLength(siteHealthErrorCodeMaxLength+1)), siteHealthErrorCodeMaxLength)
	require.Len(testingT, TruncateSiteHealthErrorMessage(stringWithLength(siteHealthErrorMessageMaxLength+1)), siteHealthErrorMessageMaxLength)
}

func stringWithLength(length int) string {
	value := make([]byte, length)
	for index := range value {
		value[index] = 'x'
	}
	return string(value)
}
