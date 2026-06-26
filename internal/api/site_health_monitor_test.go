package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	testSiteHealthSiteID          = "site-health-site-id"
	testSiteHealthSiteName        = "Health Monitor Site"
	testSiteHealthOwnerEmail      = "health-owner@example.com"
	testSiteHealthOtherEmail      = "health-other@example.com"
	testSiteHealthTeamEmail       = "health-team@example.com"
	testSiteHealthSecondTeamEmail = "health-team-two@example.com"
	testSiteHealthAllowedOrigin   = "https://site-health.example.com"
	testSiteHealthTargetURL       = testSiteHealthAllowedOrigin + "/healthz"
	testSiteHealthPath            = "/api/sites/" + testSiteHealthSiteID + "/health-monitor"
	testSiteHealthCheckPath       = "/api/sites/" + testSiteHealthSiteID + "/health-monitor/check"
)

type siteHealthHarness struct {
	handlers    *SiteHealthHandlers
	manager     *SiteHealthManager
	database    *gorm.DB
	prober      *recordingSiteHealthProber
	emailSender *recordingSiteHealthEmailSender
	referenceAt time.Time
}

type recordingSiteHealthProber struct {
	results []SiteHealthProbeResult
	calls   []siteHealthProbeCall
}

type siteHealthProbeCall struct {
	targetURL string
	timeout   time.Duration
}

func (prober *recordingSiteHealthProber) Probe(_ context.Context, targetURL string, timeout time.Duration) SiteHealthProbeResult {
	prober.calls = append(prober.calls, siteHealthProbeCall{targetURL: targetURL, timeout: timeout})
	if len(prober.results) == 0 {
		return SiteHealthProbeResult{
			TargetURL:  targetURL,
			Success:    true,
			StatusCode: http.StatusNoContent,
			CheckedAt:  time.Now().UTC(),
		}
	}
	result := prober.results[0]
	prober.results = prober.results[1:]
	if result.TargetURL == "" {
		result.TargetURL = targetURL
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	return result
}

type recordingSiteHealthEmailSender struct {
	calls []siteHealthEmailCall
	err   error
}

type siteHealthEmailCall struct {
	recipient string
	subject   string
	message   string
}

func (sender *recordingSiteHealthEmailSender) SendEmail(_ context.Context, recipient string, subject string, message string) error {
	sender.calls = append(sender.calls, siteHealthEmailCall{recipient: recipient, subject: subject, message: message})
	return sender.err
}

func buildSiteHealthHarness(testingT *testing.T, emailEnabled bool, results []SiteHealthProbeResult) siteHealthHarness {
	testingT.Helper()
	gin.SetMode(gin.TestMode)
	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	database, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	database = testutil.ConfigureDatabaseLogger(testingT, database)
	require.NoError(testingT, storage.AutoMigrate(database))

	referenceAt := time.Date(2026, time.June, 25, 15, 0, 0, 0, time.UTC)
	prober := &recordingSiteHealthProber{results: results}
	emailSender := &recordingSiteHealthEmailSender{}
	manager := NewSiteHealthManager(
		database,
		zap.NewNop(),
		prober,
		emailSender,
		emailEnabled,
		WithSiteHealthClock(func() time.Time {
			return referenceAt
		}),
	)
	handlers := NewSiteHealthHandlers(database, zap.NewNop(), manager, emailEnabled)
	handlers.now = func() time.Time {
		return referenceAt
	}
	return siteHealthHarness{
		handlers:    handlers,
		manager:     manager,
		database:    database,
		prober:      prober,
		emailSender: emailSender,
		referenceAt: referenceAt,
	}
}

func buildSiteHealthContext(method string, path string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		encodedBody, _ := json.Marshal(body)
		requestBody = bytes.NewReader(encodedBody)
	}
	request := httptest.NewRequest(method, path, requestBody)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	return context, recorder
}

func insertSiteHealthSite(testingT *testing.T, database *gorm.DB) model.Site {
	testingT.Helper()
	site := model.Site{
		ID:            testSiteHealthSiteID,
		Name:          testSiteHealthSiteName,
		AllowedOrigin: testSiteHealthAllowedOrigin,
		OwnerEmail:    testSiteHealthOwnerEmail,
		CreatorEmail:  testSiteHealthOwnerEmail,
	}
	require.NoError(testingT, database.Create(&site).Error)
	return site
}

func insertSiteHealthTeamMember(testingT *testing.T, database *gorm.DB, email string) model.SiteTeamMember {
	testingT.Helper()
	teamMember, teamMemberErr := model.NewSiteTeamMember(model.SiteTeamMemberInput{
		SiteID:       testSiteHealthSiteID,
		Email:        email,
		AddedByEmail: testSiteHealthOwnerEmail,
	})
	require.NoError(testingT, teamMemberErr)
	require.NoError(testingT, database.Create(&teamMember).Error)
	return teamMember
}

func setSiteHealthUser(context *gin.Context, email string) {
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: email, Role: RoleUser})
	context.Params = gin.Params{{Key: "id", Value: testSiteHealthSiteID}}
}

func decodeSiteHealthMonitor(testingT *testing.T, recorder *httptest.ResponseRecorder) siteHealthMonitorResponse {
	testingT.Helper()
	var response siteHealthMonitorResponse
	require.NoError(testingT, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func siteHealthIntPointer(value int) *int {
	return &value
}

func saveSiteHealthMonitor(testingT *testing.T, harness siteHealthHarness, payload siteHealthMonitorRequest) siteHealthMonitorResponse {
	testingT.Helper()
	context, recorder := buildSiteHealthContext(http.MethodPut, testSiteHealthPath, payload)
	setSiteHealthUser(context, testSiteHealthOwnerEmail)
	harness.handlers.SaveMonitor(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	return decodeSiteHealthMonitor(testingT, recorder)
}

func runSiteHealthManualCheck(testingT *testing.T, harness siteHealthHarness) siteHealthMonitorResponse {
	testingT.Helper()
	context, recorder := buildSiteHealthContext(http.MethodPost, testSiteHealthCheckPath, nil)
	setSiteHealthUser(context, testSiteHealthOwnerEmail)
	harness.handlers.RunCheck(context)
	require.Equal(testingT, http.StatusOK, recorder.Code)
	return decodeSiteHealthMonitor(testingT, recorder)
}

func TestSiteHealthMonitorManualCheckTransitionsAndAlerts(testingT *testing.T) {
	firstFailureAt := time.Date(2026, time.June, 25, 15, 1, 0, 0, time.UTC)
	secondFailureAt := time.Date(2026, time.June, 25, 15, 2, 0, 0, time.UTC)
	recoveredAt := time.Date(2026, time.June, 25, 15, 3, 0, 0, time.UTC)
	harness := buildSiteHealthHarness(testingT, true, []SiteHealthProbeResult{
		{
			TargetURL:    testSiteHealthTargetURL,
			Success:      false,
			StatusCode:   http.StatusServiceUnavailable,
			ErrorCode:    model.SiteHealthErrorHTTP5xx,
			ErrorMessage: "HTTP 503",
			Duration:     120 * time.Millisecond,
			CheckedAt:    firstFailureAt,
		},
		{
			TargetURL:    testSiteHealthTargetURL,
			Success:      false,
			StatusCode:   http.StatusServiceUnavailable,
			ErrorCode:    model.SiteHealthErrorHTTP5xx,
			ErrorMessage: "HTTP 503",
			Duration:     150 * time.Millisecond,
			CheckedAt:    secondFailureAt,
		},
		{
			TargetURL:  testSiteHealthTargetURL,
			Success:    true,
			StatusCode: http.StatusNoContent,
			Duration:   90 * time.Millisecond,
			CheckedAt:  recoveredAt,
		},
	})
	insertSiteHealthSite(testingT, harness.database)

	savedMonitor := saveSiteHealthMonitor(testingT, harness, siteHealthMonitorRequest{
		Enabled:          true,
		TargetURL:        testSiteHealthTargetURL,
		IntervalSeconds:  siteHealthIntPointer(model.DefaultSiteHealthIntervalSeconds),
		TimeoutSeconds:   siteHealthIntPointer(5),
		FailureThreshold: siteHealthIntPointer(2),
		RecipientMode:    model.SiteRecipientModeManager,
		RecipientEmails:  []string{},
	})
	require.True(testingT, savedMonitor.Persisted)
	require.Equal(testingT, model.SiteHealthStatusUnknown, savedMonitor.Status)

	firstCheck := runSiteHealthManualCheck(testingT, harness)
	require.Equal(testingT, model.SiteHealthStatusUnknown, firstCheck.Status)
	require.Equal(testingT, 1, firstCheck.ConsecutiveFailures)
	require.Equal(testingT, firstFailureAt.Unix(), firstCheck.LastCheckedAt)
	require.Empty(testingT, harness.emailSender.calls)

	var events []model.SiteHealthEvent
	require.NoError(testingT, harness.database.Where("site_id = ?", testSiteHealthSiteID).Find(&events).Error)
	require.Empty(testingT, events)

	secondCheck := runSiteHealthManualCheck(testingT, harness)
	require.Equal(testingT, model.SiteHealthStatusDown, secondCheck.Status)
	require.Equal(testingT, 2, secondCheck.ConsecutiveFailures)
	require.Equal(testingT, http.StatusServiceUnavailable, secondCheck.LastStatusCode)
	require.Equal(testingT, model.SiteHealthErrorHTTP5xx, secondCheck.LastErrorCode)
	require.Equal(testingT, 150, secondCheck.LastDurationMs)
	require.Len(testingT, harness.emailSender.calls, 1)
	require.Equal(testingT, testSiteHealthOwnerEmail, harness.emailSender.calls[0].recipient)
	require.Equal(testingT, "Site down: "+testSiteHealthSiteName, harness.emailSender.calls[0].subject)
	require.Contains(testingT, harness.emailSender.calls[0].message, "Failure threshold: 2 consecutive checks")
	require.Contains(testingT, harness.emailSender.calls[0].message, "HTTP 503")

	require.NoError(testingT, harness.database.Order("created_at asc").Where("site_id = ?", testSiteHealthSiteID).Find(&events).Error)
	require.Len(testingT, events, 1)
	require.Equal(testingT, model.SiteHealthEventKindDown, events[0].Kind)
	require.Equal(testingT, model.SiteHealthStatusDown, events[0].Status)

	recoveredCheck := runSiteHealthManualCheck(testingT, harness)
	require.Equal(testingT, model.SiteHealthStatusUp, recoveredCheck.Status)
	require.Equal(testingT, 0, recoveredCheck.ConsecutiveFailures)
	require.Equal(testingT, recoveredAt.Unix(), recoveredCheck.LastCheckedAt)
	require.Equal(testingT, recoveredAt.Add(5*time.Minute).Unix(), recoveredCheck.NextCheckAt)
	require.Len(testingT, harness.emailSender.calls, 2)
	require.Equal(testingT, "Site recovered: "+testSiteHealthSiteName, harness.emailSender.calls[1].subject)
	require.Contains(testingT, harness.emailSender.calls[1].message, "HTTP status: 204")

	require.NoError(testingT, harness.database.Order("created_at asc").Where("site_id = ?", testSiteHealthSiteID).Find(&events).Error)
	require.Len(testingT, events, 2)
	require.Equal(testingT, model.SiteHealthEventKindRecovered, events[1].Kind)

	var storedMonitor model.SiteHealthMonitor
	require.NoError(testingT, harness.database.First(&storedMonitor, "site_id = ?", testSiteHealthSiteID).Error)
	require.Equal(testingT, model.SiteHealthStatusUp, storedMonitor.LastAlertedStatus)
	require.WithinDuration(testingT, recoveredAt, storedMonitor.LastAlertedAt, time.Second)

	require.Len(testingT, harness.prober.calls, 3)
	require.Equal(testingT, testSiteHealthTargetURL, harness.prober.calls[0].targetURL)
	require.Equal(testingT, 5*time.Second, harness.prober.calls[0].timeout)
}

func TestSiteHealthMonitorManualCheckUsesSelectedTeamRecipients(testingT *testing.T) {
	checkedAt := time.Date(2026, time.June, 25, 16, 0, 0, 0, time.UTC)
	harness := buildSiteHealthHarness(testingT, true, []SiteHealthProbeResult{
		{
			TargetURL:    testSiteHealthTargetURL,
			Success:      false,
			StatusCode:   http.StatusInternalServerError,
			ErrorCode:    model.SiteHealthErrorHTTP5xx,
			ErrorMessage: "HTTP 500",
			Duration:     75 * time.Millisecond,
			CheckedAt:    checkedAt,
		},
	})
	insertSiteHealthSite(testingT, harness.database)
	insertSiteHealthTeamMember(testingT, harness.database, testSiteHealthTeamEmail)
	insertSiteHealthTeamMember(testingT, harness.database, testSiteHealthSecondTeamEmail)

	savedMonitor := saveSiteHealthMonitor(testingT, harness, siteHealthMonitorRequest{
		Enabled:          true,
		TargetURL:        testSiteHealthTargetURL,
		IntervalSeconds:  siteHealthIntPointer(model.DefaultSiteHealthIntervalSeconds),
		TimeoutSeconds:   siteHealthIntPointer(model.DefaultSiteHealthTimeoutSeconds),
		FailureThreshold: siteHealthIntPointer(1),
		RecipientMode:    model.SiteRecipientModeSelected,
		RecipientEmails:  []string{testSiteHealthSecondTeamEmail, testSiteHealthSecondTeamEmail},
	})
	require.Equal(testingT, model.SiteRecipientModeSelected, savedMonitor.RecipientMode)
	require.Equal(testingT, []string{testSiteHealthSecondTeamEmail}, savedMonitor.RecipientEmails)

	checkedMonitor := runSiteHealthManualCheck(testingT, harness)
	require.Equal(testingT, model.SiteHealthStatusDown, checkedMonitor.Status)
	require.Len(testingT, harness.emailSender.calls, 1)
	require.Equal(testingT, testSiteHealthSecondTeamEmail, harness.emailSender.calls[0].recipient)
	require.Contains(testingT, harness.emailSender.calls[0].message, testSiteHealthTargetURL)
}

func TestSiteHealthMonitorSaveRejectsForeignSelectedRecipient(testingT *testing.T) {
	harness := buildSiteHealthHarness(testingT, true, nil)
	insertSiteHealthSite(testingT, harness.database)

	payload := siteHealthMonitorRequest{
		Enabled:          true,
		TargetURL:        testSiteHealthTargetURL,
		IntervalSeconds:  siteHealthIntPointer(model.DefaultSiteHealthIntervalSeconds),
		TimeoutSeconds:   siteHealthIntPointer(model.DefaultSiteHealthTimeoutSeconds),
		FailureThreshold: siteHealthIntPointer(model.DefaultSiteHealthFailureThreshold),
		RecipientMode:    model.SiteRecipientModeSelected,
		RecipientEmails:  []string{testSiteHealthOtherEmail},
	}
	context, recorder := buildSiteHealthContext(http.MethodPut, testSiteHealthPath, payload)
	setSiteHealthUser(context, testSiteHealthOwnerEmail)

	harness.handlers.SaveMonitor(context)
	require.Equal(testingT, http.StatusBadRequest, recorder.Code)
	require.Contains(testingT, recorder.Body.String(), errorValueInvalidHealthMonitor)
}

func TestSiteHealthManagerRunsOnlyDueEnabledMonitors(testingT *testing.T) {
	checkedAt := time.Date(2026, time.June, 25, 17, 0, 0, 0, time.UTC)
	harness := buildSiteHealthHarness(testingT, true, []SiteHealthProbeResult{
		{
			TargetURL:  testSiteHealthTargetURL,
			Success:    true,
			StatusCode: http.StatusNoContent,
			Duration:   25 * time.Millisecond,
			CheckedAt:  checkedAt,
		},
	})
	dueSite := insertSiteHealthSite(testingT, harness.database)
	futureSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Future Health Site",
		AllowedOrigin: "https://future-health.example.com",
		OwnerEmail:    testSiteHealthOwnerEmail,
		CreatorEmail:  testSiteHealthOwnerEmail,
	}
	require.NoError(testingT, harness.database.Create(&futureSite).Error)
	disabledSite := model.Site{
		ID:            storage.NewID(),
		Name:          "Disabled Health Site",
		AllowedOrigin: "https://disabled-health.example.com",
		OwnerEmail:    testSiteHealthOwnerEmail,
		CreatorEmail:  testSiteHealthOwnerEmail,
	}
	require.NoError(testingT, harness.database.Create(&disabledSite).Error)

	dueMonitor, dueMonitorErr := model.NewSiteHealthMonitor(model.SiteHealthMonitorInput{
		SiteID:           dueSite.ID,
		Enabled:          true,
		TargetURL:        testSiteHealthTargetURL,
		IntervalSeconds:  model.DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   model.DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: model.DefaultSiteHealthFailureThreshold,
		RecipientEmail:   testSiteHealthOwnerEmail,
		ReferenceTime:    harness.referenceAt.Add(-time.Minute),
	})
	require.NoError(testingT, dueMonitorErr)
	require.NoError(testingT, harness.database.Create(&dueMonitor).Error)
	futureMonitor, futureMonitorErr := model.NewSiteHealthMonitor(model.SiteHealthMonitorInput{
		SiteID:           futureSite.ID,
		Enabled:          true,
		TargetURL:        futureSite.AllowedOrigin,
		IntervalSeconds:  model.DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   model.DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: model.DefaultSiteHealthFailureThreshold,
		RecipientEmail:   testSiteHealthOwnerEmail,
		ReferenceTime:    harness.referenceAt.Add(time.Hour),
	})
	require.NoError(testingT, futureMonitorErr)
	require.NoError(testingT, harness.database.Create(&futureMonitor).Error)
	disabledMonitor, disabledMonitorErr := model.NewSiteHealthMonitor(model.SiteHealthMonitorInput{
		SiteID:           disabledSite.ID,
		Enabled:          false,
		TargetURL:        disabledSite.AllowedOrigin,
		IntervalSeconds:  model.DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   model.DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: model.DefaultSiteHealthFailureThreshold,
		RecipientEmail:   testSiteHealthOwnerEmail,
		ReferenceTime:    harness.referenceAt.Add(-time.Hour),
	})
	require.NoError(testingT, disabledMonitorErr)
	require.NoError(testingT, harness.database.Create(&disabledMonitor).Error)

	harness.manager.RunDueChecks(context.Background())

	require.Len(testingT, harness.prober.calls, 1)
	require.Equal(testingT, testSiteHealthTargetURL, harness.prober.calls[0].targetURL)

	var storedDueMonitor model.SiteHealthMonitor
	require.NoError(testingT, harness.database.First(&storedDueMonitor, "id = ?", dueMonitor.ID).Error)
	require.Equal(testingT, model.SiteHealthStatusUp, storedDueMonitor.Status)
	require.WithinDuration(testingT, checkedAt, storedDueMonitor.LastCheckedAt, time.Second)

	var storedFutureMonitor model.SiteHealthMonitor
	require.NoError(testingT, harness.database.First(&storedFutureMonitor, "id = ?", futureMonitor.ID).Error)
	require.True(testingT, storedFutureMonitor.LastCheckedAt.IsZero())

	var storedDisabledMonitor model.SiteHealthMonitor
	require.NoError(testingT, harness.database.First(&storedDisabledMonitor, "id = ?", disabledMonitor.ID).Error)
	require.True(testingT, storedDisabledMonitor.LastCheckedAt.IsZero())
}
