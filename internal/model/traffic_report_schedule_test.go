package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewTrafficReportScheduleValidatesInput(testingT *testing.T) {
	validInput := TrafficReportScheduleInput{
		SiteID:         "site-id",
		Enabled:        true,
		Frequency:      TrafficReportFrequencyDaily,
		RecipientEmail: "owner@example.com",
		Timezone:       DefaultTrafficReportTimezone,
		SendHour:       9,
		SendMinute:     0,
		Weekday:        DefaultTrafficReportWeekday,
		MonthDay:       DefaultTrafficReportMonthDay,
	}

	testCases := []struct {
		name   string
		mutate func(*TrafficReportScheduleInput)
	}{
		{name: "missing site", mutate: func(input *TrafficReportScheduleInput) { input.SiteID = " " }},
		{name: "invalid frequency", mutate: func(input *TrafficReportScheduleInput) { input.Frequency = "hourly" }},
		{name: "invalid email", mutate: func(input *TrafficReportScheduleInput) { input.RecipientEmail = "invalid" }},
		{name: "invalid timezone", mutate: func(input *TrafficReportScheduleInput) { input.Timezone = "Mars/Olympus" }},
		{name: "invalid hour", mutate: func(input *TrafficReportScheduleInput) { input.SendHour = 24 }},
		{name: "invalid minute", mutate: func(input *TrafficReportScheduleInput) { input.SendMinute = 60 }},
		{name: "invalid weekday", mutate: func(input *TrafficReportScheduleInput) { input.Weekday = 8 }},
		{name: "invalid month day", mutate: func(input *TrafficReportScheduleInput) { input.MonthDay = 29 }},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			input := validInput
			testCase.mutate(&input)
			_, err := NewTrafficReportSchedule(input)
			require.Error(testingT, err)
			require.True(testingT, errors.Is(err, ErrInvalidTrafficReportSchedule))
		})
	}
}

func TestTrafficReportScheduleNextAfter(testingT *testing.T) {
	reference := time.Date(2026, time.April, 20, 10, 30, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		frequency string
		hour      int
		minute    int
		weekday   int
		monthDay  int
		expected  time.Time
	}{
		{
			name:      "daily rolls to tomorrow",
			frequency: TrafficReportFrequencyDaily,
			hour:      9,
			minute:    0,
			weekday:   int(time.Monday),
			monthDay:  1,
			expected:  time.Date(2026, time.April, 21, 9, 0, 0, 0, time.UTC),
		},
		{
			name:      "weekly uses configured weekday",
			frequency: TrafficReportFrequencyWeekly,
			hour:      8,
			minute:    15,
			weekday:   int(time.Wednesday),
			monthDay:  1,
			expected:  time.Date(2026, time.April, 22, 8, 15, 0, 0, time.UTC),
		},
		{
			name:      "monthly rolls to next month",
			frequency: TrafficReportFrequencyMonthly,
			hour:      7,
			minute:    45,
			weekday:   int(time.Monday),
			monthDay:  14,
			expected:  time.Date(2026, time.May, 14, 7, 45, 0, 0, time.UTC),
		},
	}

	for _, testCase := range testCases {
		testingT.Run(testCase.name, func(testingT *testing.T) {
			schedule, err := NewTrafficReportSchedule(TrafficReportScheduleInput{
				SiteID:         "site-id",
				Enabled:        true,
				Frequency:      testCase.frequency,
				RecipientEmail: "owner@example.com",
				Timezone:       DefaultTrafficReportTimezone,
				SendHour:       testCase.hour,
				SendMinute:     testCase.minute,
				Weekday:        testCase.weekday,
				MonthDay:       testCase.monthDay,
				ReferenceTime:  reference,
			})
			require.NoError(testingT, err)
			require.Equal(testingT, testCase.expected, schedule.NextSendAt)
		})
	}
}

func TestTrafficReportScheduleReportWindowDays(testingT *testing.T) {
	require.Equal(testingT, 1, TrafficReportSchedule{Frequency: TrafficReportFrequencyDaily}.ReportWindowDays())
	require.Equal(testingT, 7, TrafficReportSchedule{Frequency: TrafficReportFrequencyWeekly}.ReportWindowDays())
	require.Equal(testingT, 30, TrafficReportSchedule{Frequency: TrafficReportFrequencyMonthly}.ReportWindowDays())
	require.Equal(testingT, 1, TrafficReportSchedule{Frequency: "unknown"}.ReportWindowDays())
}
