package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
)

const (
	testTrafficReportSiteID        = "traffic-report-site-id"
	testTrafficReportSiteName      = "Traffic Report Site"
	testTrafficReportOwnerEmail    = "traffic-owner@example.com"
	testTrafficReportOtherEmail    = "traffic-other@example.com"
	testTrafficReportRecipient     = "reports@example.com"
	testTrafficReportAllowedOrigin = "https://traffic-report.example.com"
	testTrafficReportPath          = "/api/sites/" + testTrafficReportSiteID + "/traffic-report-schedule"
	testTrafficReportTestPath      = "/api/sites/" + testTrafficReportSiteID + "/traffic-report-schedule/test"
)

type trafficReportHarness struct {
	handlers    *TrafficReportHandlers
	database    *gorm.DB
	stats       *recordingTrafficReportStatsProvider
	emailSender *recordingTrafficReportEmailSender
}

type recordingTrafficReportEmailSender struct {
	calls []trafficReportEmailCall
	err   error
}

type trafficReportEmailCall struct {
	recipient string
	subject   string
	message   string
}

func (sender *recordingTrafficReportEmailSender) SendEmail(_ context.Context, recipient string, subject string, message string) error {
	sender.calls = append(sender.calls, trafficReportEmailCall{recipient: recipient, subject: subject, message: message})
	return sender.err
}

type recordingTrafficReportStatsProvider struct {
	trend     []DailyVisitTrendStat
	topPages  []TopPageStat
	devices   DeviceBreakdownStat
	timezones []TimezoneDistributionStat
}

func (provider *recordingTrafficReportStatsProvider) FeedbackCount(context.Context, string) (int64, error) {
	return 0, nil
}

func (provider *recordingTrafficReportStatsProvider) SubscriberCount(context.Context, string) (int64, error) {
	return 0, nil
}

func (provider *recordingTrafficReportStatsProvider) VisitCount(context.Context, string) (int64, error) {
	return 0, nil
}

func (provider *recordingTrafficReportStatsProvider) UniqueVisitorCount(context.Context, string) (int64, error) {
	return 0, nil
}

func (provider *recordingTrafficReportStatsProvider) TopPages(context.Context, string, int) ([]TopPageStat, error) {
	return provider.topPages, nil
}

func (provider *recordingTrafficReportStatsProvider) VisitTrend(context.Context, string, int) ([]DailyVisitTrendStat, error) {
	return provider.trend, nil
}

func (provider *recordingTrafficReportStatsProvider) VisitAttribution(context.Context, string, int) (VisitAttributionBreakdown, error) {
	return VisitAttributionBreakdown{}, nil
}

func (provider *recordingTrafficReportStatsProvider) VisitEngagement(context.Context, string, int) (VisitEngagementStat, error) {
	return VisitEngagementStat{}, nil
}

func (provider *recordingTrafficReportStatsProvider) DeviceBreakdown(context.Context, string, int) (DeviceBreakdownStat, error) {
	return provider.devices, nil
}

func (provider *recordingTrafficReportStatsProvider) TimezoneDistribution(context.Context, string, int) ([]TimezoneDistributionStat, error) {
	return provider.timezones, nil
}

func buildTrafficReportHarness(testingT *testing.T, emailEnabled bool) trafficReportHarness {
	testingT.Helper()
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	stats := &recordingTrafficReportStatsProvider{
		trend: []DailyVisitTrendStat{
			{Date: time.Date(2026, time.April, 22, 0, 0, 0, 0, time.UTC), PageViews: 5, UniqueVisitors: 3},
			{Date: time.Date(2026, time.April, 23, 0, 0, 0, 0, time.UTC), PageViews: 7, UniqueVisitors: 4},
		},
		topPages: []TopPageStat{
			{Path: "/pricing", VisitCount: 6},
			{Path: "/docs", VisitCount: 3},
		},
		devices: DeviceBreakdownStat{
			DeviceTypes: []DeviceTypeStat{{DeviceType: "desktop", VisitCount: 8}},
		},
		timezones: []TimezoneDistributionStat{{Timezone: "America/Los_Angeles", VisitCount: 9}},
	}
	emailSender := &recordingTrafficReportEmailSender{}
	handlers := NewTrafficReportHandlers(database, zap.NewNop(), stats, emailSender, emailEnabled)
	handlers.now = func() time.Time {
		return time.Date(2026, time.April, 23, 16, 0, 0, 0, time.UTC)
	}
	return trafficReportHarness{handlers: handlers, database: database, stats: stats, emailSender: emailSender}
}

func buildTrafficReportContext(method string, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	context.Request = request
	return context, recorder
}

func insertTrafficReportSite(testingT *testing.T, database *gorm.DB) {
	testingT.Helper()
	site := model.Site{
		ID:            testTrafficReportSiteID,
		Name:          testTrafficReportSiteName,
		AllowedOrigin: testTrafficReportAllowedOrigin,
		OwnerEmail:    testTrafficReportOwnerEmail,
		CreatorEmail:  testTrafficReportOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)
}

func setTrafficReportUser(context *gin.Context, email string) {
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: email, Role: RoleUser})
	context.Params = gin.Params{{Key: "id", Value: testTrafficReportSiteID}}
}

func decodeTrafficReportSchedule(testingT *testing.T, recorder *httptest.ResponseRecorder) trafficReportScheduleResponse {
	testingT.Helper()
	var response trafficReportScheduleResponse
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func marshalTrafficReportSchedule(testingT *testing.T, payload trafficReportScheduleRequest) []byte {
	testingT.Helper()
	body, marshalErr := json.Marshal(payload)
	require.NoError(testingT, marshalErr)
	return body
}

func TestGetTrafficReportScheduleReturnsDefaults(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	context, recorder := buildTrafficReportContext(http.MethodGet, testTrafficReportPath, nil)
	setTrafficReportUser(context, testTrafficReportOwnerEmail)

	harness.handlers.GetSchedule(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	response := decodeTrafficReportSchedule(testingT, recorder)
	require.Equal(testingT, testTrafficReportSiteID, response.SiteID)
	require.False(testingT, response.Enabled)
	require.Equal(testingT, model.TrafficReportFrequencyDaily, response.Frequency)
	require.Equal(testingT, testTrafficReportOwnerEmail, response.RecipientEmail)
	require.Equal(testingT, model.DefaultTrafficReportTimezone, response.Timezone)
	require.True(testingT, response.EmailEnabled)
	require.Positive(testingT, response.NextSendAt)
}

func TestSaveTrafficReportSchedulePersistsValues(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	hour := 14
	minute := 30
	weekday := int(time.Friday)
	monthDay := 12
	payload := trafficReportScheduleRequest{
		Enabled:        true,
		Frequency:      model.TrafficReportFrequencyWeekly,
		RecipientEmail: testTrafficReportRecipient,
		Timezone:       "America/New_York",
		SendHour:       &hour,
		SendMinute:     &minute,
		Weekday:        &weekday,
		MonthDay:       &monthDay,
	}
	context, recorder := buildTrafficReportContext(http.MethodPut, testTrafficReportPath, marshalTrafficReportSchedule(testingT, payload))
	setTrafficReportUser(context, testTrafficReportOwnerEmail)

	harness.handlers.SaveSchedule(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	response := decodeTrafficReportSchedule(testingT, recorder)
	require.True(testingT, response.Enabled)
	require.Equal(testingT, model.TrafficReportFrequencyWeekly, response.Frequency)
	require.Equal(testingT, testTrafficReportRecipient, response.RecipientEmail)
	require.Equal(testingT, "America/New_York", response.Timezone)
	require.Equal(testingT, hour, response.SendHour)
	require.Equal(testingT, minute, response.SendMinute)
	require.Equal(testingT, weekday, response.Weekday)
	require.Equal(testingT, monthDay, response.MonthDay)

	var stored model.TrafficReportSchedule
	require.NoError(testingT, harness.database.First(&stored, "site_id = ?", testTrafficReportSiteID).Error)
	require.True(testingT, stored.Enabled)
	require.Equal(testingT, testTrafficReportRecipient, stored.RecipientEmail)
	require.False(testingT, stored.NextSendAt.IsZero())
}

func TestSaveTrafficReportScheduleRejectsInvalidValues(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	hour := 9
	payload := trafficReportScheduleRequest{
		Enabled:        true,
		Frequency:      "hourly",
		RecipientEmail: "not-an-email",
		Timezone:       "Mars/Olympus",
		SendHour:       &hour,
	}
	context, recorder := buildTrafficReportContext(http.MethodPut, testTrafficReportPath, marshalTrafficReportSchedule(testingT, payload))
	setTrafficReportUser(context, testTrafficReportOwnerEmail)

	harness.handlers.SaveSchedule(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), errorValueInvalidTrafficReportSchedule)
}

func TestSaveTrafficReportScheduleRejectsForbiddenUser(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	context, recorder := buildTrafficReportContext(http.MethodPut, testTrafficReportPath, []byte(`{}`))
	setTrafficReportUser(context, testTrafficReportOtherEmail)

	harness.handlers.SaveSchedule(context)
	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestSendTrafficReportTestUsesPinguinSender(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	schedule, scheduleErr := model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:         testTrafficReportSiteID,
		Enabled:        true,
		Frequency:      model.TrafficReportFrequencyDaily,
		RecipientEmail: testTrafficReportRecipient,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       model.DefaultTrafficReportSendHour,
		SendMinute:     model.DefaultTrafficReportSendMinute,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
	})
	require.NoError(testingT, scheduleErr)
	require.NoError(testingT, harness.database.Create(&schedule).Error)
	context, recorder := buildTrafficReportContext(http.MethodPost, testTrafficReportTestPath, nil)
	setTrafficReportUser(context, testTrafficReportOwnerEmail)

	harness.handlers.SendTestReport(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	require.Len(testingT, harness.emailSender.calls, 1)
	call := harness.emailSender.calls[0]
	require.Equal(testingT, testTrafficReportRecipient, call.recipient)
	require.Contains(testingT, call.subject, "Daily traffic report")
	require.Contains(testingT, call.message, "Page views: 12")
	require.Contains(testingT, call.message, "/pricing: 6 views")
	require.Contains(testingT, call.message, "desktop: 8 visits")
}

func TestSendTrafficReportTestRequiresEmailEnabled(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, false)
	insertTrafficReportSite(testingT, harness.database)
	context, recorder := buildTrafficReportContext(http.MethodPost, testTrafficReportTestPath, nil)
	setTrafficReportUser(context, testTrafficReportOwnerEmail)

	harness.handlers.SendTestReport(context)
	require.Equal(testingT, http.StatusServiceUnavailable, recorder.Code)
	require.Empty(testingT, harness.emailSender.calls)
	require.Contains(testingT, recorder.Body.String(), errorValueTrafficReportEmailDisabled)
}

func TestBuildTrafficReportEmailRendersTemplateFallbackSections(testingT *testing.T) {
	schedule, scheduleErr := model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:         testTrafficReportSiteID,
		Enabled:        true,
		Frequency:      model.TrafficReportFrequencyWeekly,
		RecipientEmail: testTrafficReportRecipient,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       model.DefaultTrafficReportSendHour,
		SendMinute:     model.DefaultTrafficReportSendMinute,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
	})
	require.NoError(testingT, scheduleErr)

	report, reportErr := buildTrafficReportEmail(context.Background(), &recordingTrafficReportStatsProvider{}, model.Site{
		ID:   testTrafficReportSiteID,
		Name: testTrafficReportSiteName,
	}, schedule)
	require.NoError(testingT, reportErr)
	require.Equal(testingT, "Weekly traffic report for "+testTrafficReportSiteName, report.subject)
	require.Contains(testingT, report.message, "Window: last 7 days")
	require.Contains(testingT, report.message, "Page views: 0")
	require.Contains(testingT, report.message, "- No page views recorded.")
	require.Contains(testingT, report.message, "- No device data recorded.")
	require.Contains(testingT, report.message, "- No timezone data recorded.")
}

func TestTrafficReportSchedulerSendsDueSchedule(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	schedule, scheduleErr := model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:         testTrafficReportSiteID,
		Enabled:        true,
		Frequency:      model.TrafficReportFrequencyDaily,
		RecipientEmail: testTrafficReportRecipient,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       9,
		SendMinute:     0,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
		ReferenceTime:  time.Now().UTC().Add(-48 * time.Hour),
	})
	require.NoError(testingT, scheduleErr)
	schedule.NextSendAt = time.Now().UTC().Add(-time.Hour)
	require.NoError(testingT, harness.database.Create(&schedule).Error)

	trafficScheduler, schedulerErr := NewTrafficReportScheduler(harness.database, zap.NewNop(), harness.stats, harness.emailSender, time.Millisecond, 5)
	require.NoError(testingT, schedulerErr)
	trafficScheduler.RunOnce(context.Background())

	require.Len(testingT, harness.emailSender.calls, 1)
	var stored model.TrafficReportSchedule
	require.NoError(testingT, harness.database.First(&stored, "id = ?", schedule.ID).Error)
	require.Equal(testingT, model.TrafficReportStatusSent, stored.LastStatus)
	require.Equal(testingT, 0, stored.RetryCount)
	require.Empty(testingT, stored.LastError)
	require.False(testingT, stored.LastSentAt.IsZero())
	require.True(testingT, stored.NextSendAt.After(time.Now().UTC()))
}

func TestTrafficReportSchedulerRecordsSendFailure(testingT *testing.T) {
	harness := buildTrafficReportHarness(testingT, true)
	insertTrafficReportSite(testingT, harness.database)
	harness.emailSender.err = errors.New("smtp failed")
	schedule, scheduleErr := model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:         testTrafficReportSiteID,
		Enabled:        true,
		Frequency:      model.TrafficReportFrequencyDaily,
		RecipientEmail: testTrafficReportRecipient,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       9,
		SendMinute:     0,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
		ReferenceTime:  time.Now().UTC().Add(-48 * time.Hour),
	})
	require.NoError(testingT, scheduleErr)
	schedule.NextSendAt = time.Now().UTC().Add(-time.Hour)
	require.NoError(testingT, harness.database.Create(&schedule).Error)

	trafficScheduler, schedulerErr := NewTrafficReportScheduler(harness.database, zap.NewNop(), harness.stats, harness.emailSender, time.Millisecond, 5)
	require.NoError(testingT, schedulerErr)
	trafficScheduler.RunOnce(context.Background())

	require.Len(testingT, harness.emailSender.calls, 1)
	var stored model.TrafficReportSchedule
	require.NoError(testingT, harness.database.First(&stored, "id = ?", schedule.ID).Error)
	require.Equal(testingT, model.TrafficReportStatusFailed, stored.LastStatus)
	require.Equal(testingT, 1, stored.RetryCount)
	require.True(testingT, strings.Contains(stored.LastError, trafficReportStatusDispatchFailed))
	require.True(testingT, stored.LastSentAt.IsZero())
}
