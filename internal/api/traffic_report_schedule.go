package api

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tyemirov/utils/scheduler"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	errorValueInvalidTrafficReportSchedule = "invalid_traffic_report_schedule"
	errorValueTrafficReportEmailDisabled   = "traffic_report_email_disabled"
	errorValueTrafficReportSendFailed      = "traffic_report_send_failed"

	defaultTrafficReportSchedulerInterval = 15 * time.Minute
	defaultTrafficReportSchedulerRetries  = 5
	trafficReportTopPagesLimit            = 10
	trafficReportStatusDispatchFailed     = "dispatch failed"
)

//go:embed templates/traffic_report_email.txt
var trafficReportEmailTemplateText string

var trafficReportEmailTemplate = template.Must(template.New("traffic_report_email").Option("missingkey=error").Parse(trafficReportEmailTemplateText))

type trafficReportScheduleRequest struct {
	Enabled         bool     `json:"enabled"`
	Frequency       string   `json:"frequency"`
	RecipientEmail  string   `json:"recipient_email"`
	RecipientMode   string   `json:"recipient_mode"`
	RecipientEmails []string `json:"recipient_emails"`
	Timezone        string   `json:"timezone"`
	SendHour        *int     `json:"send_hour"`
	SendMinute      *int     `json:"send_minute"`
	Weekday         *int     `json:"weekday"`
	MonthDay        *int     `json:"month_day"`
}

type trafficReportScheduleResponse struct {
	SiteID          string   `json:"site_id"`
	ReportID        string   `json:"report_id"`
	Enabled         bool     `json:"enabled"`
	Frequency       string   `json:"frequency"`
	RecipientEmail  string   `json:"recipient_email"`
	RecipientMode   string   `json:"recipient_mode"`
	RecipientEmails []string `json:"recipient_emails"`
	Timezone        string   `json:"timezone"`
	SendHour        int      `json:"send_hour"`
	SendMinute      int      `json:"send_minute"`
	Weekday         int      `json:"weekday"`
	MonthDay        int      `json:"month_day"`
	NextSendAt      int64    `json:"next_send_at"`
	LastSentAt      int64    `json:"last_sent_at"`
	LastStatus      string   `json:"last_status"`
	LastError       string   `json:"last_error"`
	EmailEnabled    bool     `json:"email_enabled"`
	Persisted       bool     `json:"persisted"`
}

// TrafficReportHandlers owns authenticated traffic report schedule APIs.
type TrafficReportHandlers struct {
	database      *gorm.DB
	logger        *zap.Logger
	statsProvider SiteStatisticsProvider
	emailSender   EmailSender
	emailEnabled  bool
	now           func() time.Time
}

// NewTrafficReportHandlers builds traffic report schedule handlers.
func NewTrafficReportHandlers(database *gorm.DB, logger *zap.Logger, statsProvider SiteStatisticsProvider, emailSender EmailSender, emailEnabled bool) *TrafficReportHandlers {
	if statsProvider == nil {
		statsProvider = NewDatabaseSiteStatisticsProvider(database)
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TrafficReportHandlers{
		database:      database,
		logger:        logger,
		statsProvider: statsProvider,
		emailSender:   emailSender,
		emailEnabled:  emailEnabled,
		now:           time.Now,
	}
}

// GetSchedule returns the saved traffic report schedule or site-specific defaults.
func (handlers *TrafficReportHandlers) GetSchedule(context *gin.Context) {
	site, currentUser, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	schedule, exists, findErr := handlers.findSchedule(context.Request.Context(), site.ID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		context.JSON(http.StatusOK, handlers.toScheduleResponse(defaultTrafficReportSchedule(site, currentUser.normalizedEmail()), site.ID, false))
		return
	}

	context.JSON(http.StatusOK, handlers.toScheduleResponse(schedule, site.ID, true))
}

// SaveSchedule validates and persists a traffic report schedule.
func (handlers *TrafficReportHandlers) SaveSchedule(context *gin.Context) {
	site, currentUser, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}

	var payload trafficReportScheduleRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	schedule, scheduleErr := buildTrafficReportScheduleFromRequest(site, currentUser.normalizedEmail(), payload, handlers.now().UTC())
	if scheduleErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidTrafficReportSchedule})
		return
	}
	if recipientErr := handlers.validateTrafficReportScheduleRecipients(context.Request.Context(), site, schedule); recipientErr != nil {
		if errors.Is(recipientErr, model.ErrInvalidTrafficReportSchedule) {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidTrafficReportSchedule})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	savedSchedule, saveErr := handlers.upsertSchedule(context.Request.Context(), schedule)
	if saveErr != nil {
		handlers.logger.Warn("traffic_report_schedule_save_failed", zap.Error(saveErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, handlers.toScheduleResponse(savedSchedule, site.ID, true))
}

// SendTestReport sends the saved traffic report immediately without changing the recurring cadence.
func (handlers *TrafficReportHandlers) SendTestReport(context *gin.Context) {
	site, currentUser, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}
	if !handlers.emailEnabled || handlers.emailSender == nil {
		context.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueTrafficReportEmailDisabled})
		return
	}

	schedule, exists, findErr := handlers.findSchedule(context.Request.Context(), site.ID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		schedule = defaultTrafficReportSchedule(site, currentUser.normalizedEmail())
	}

	dispatcher := trafficReportDispatcher{
		database:      handlers.database,
		statsProvider: handlers.statsProvider,
		emailSender:   handlers.emailSender,
	}
	if sendErr := dispatcher.sendSchedule(context.Request.Context(), site, schedule); sendErr != nil {
		handlers.logger.Warn("traffic_report_test_send_failed", zap.Error(sendErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusBadGateway, gin.H{jsonKeyError: errorValueTrafficReportSendFailed})
		return
	}

	context.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (handlers *TrafficReportHandlers) resolveAuthorizedSite(context *gin.Context) (model.Site, *CurrentUser, bool) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueMissingSite})
		return model.Site{}, nil, false
	}
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return model.Site{}, nil, false
	}
	var site model.Site
	if err := handlers.database.First(&site, "id = ?", siteIdentifier).Error; err != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownSite})
		return model.Site{}, nil, false
	}
	if !currentUser.canManageSite(site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return model.Site{}, nil, false
	}
	return site, currentUser, true
}

func (handlers *TrafficReportHandlers) findSchedule(ctx context.Context, siteID string) (model.TrafficReportSchedule, bool, error) {
	var schedule model.TrafficReportSchedule
	err := handlers.database.WithContext(ctx).First(&schedule, "site_id = ?", siteID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.TrafficReportSchedule{}, false, nil
	}
	if err != nil {
		return model.TrafficReportSchedule{}, false, err
	}
	return schedule, true, nil
}

func (handlers *TrafficReportHandlers) upsertSchedule(ctx context.Context, schedule model.TrafficReportSchedule) (model.TrafficReportSchedule, error) {
	var existing model.TrafficReportSchedule
	findErr := handlers.database.WithContext(ctx).First(&existing, "site_id = ?", schedule.SiteID).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		if createErr := handlers.database.WithContext(ctx).Create(&schedule).Error; createErr != nil {
			return model.TrafficReportSchedule{}, createErr
		}
		return schedule, nil
	}
	if findErr != nil {
		return model.TrafficReportSchedule{}, findErr
	}

	existing.Enabled = schedule.Enabled
	existing.Frequency = schedule.Frequency
	existing.RecipientEmail = schedule.RecipientEmail
	existing.RecipientMode = schedule.RecipientMode
	existing.RecipientEmails = schedule.RecipientEmails
	existing.Timezone = schedule.Timezone
	existing.SendHour = schedule.SendHour
	existing.SendMinute = schedule.SendMinute
	existing.Weekday = schedule.Weekday
	existing.MonthDay = schedule.MonthDay
	existing.NextSendAt = schedule.NextSendAt
	existing.LastAttemptedAt = time.Time{}
	existing.RetryCount = 0
	existing.LastStatus = model.TrafficReportStatusPending
	existing.LastError = ""
	existing.ProviderMessageID = ""
	if saveErr := handlers.database.WithContext(ctx).Save(&existing).Error; saveErr != nil {
		return model.TrafficReportSchedule{}, saveErr
	}
	return existing, nil
}

func (handlers *TrafficReportHandlers) toScheduleResponse(schedule model.TrafficReportSchedule, siteID string, persisted bool) trafficReportScheduleResponse {
	lastStatus := strings.TrimSpace(schedule.LastStatus)
	if lastStatus == "" {
		lastStatus = model.TrafficReportStatusPending
	}
	return trafficReportScheduleResponse{
		SiteID:          siteID,
		Enabled:         schedule.Enabled,
		Frequency:       schedule.Frequency,
		RecipientEmail:  schedule.RecipientEmail,
		RecipientMode:   schedule.RecipientModeValue(),
		RecipientEmails: schedule.SelectedRecipientEmails(),
		Timezone:        schedule.Timezone,
		SendHour:        schedule.SendHour,
		SendMinute:      schedule.SendMinute,
		Weekday:         schedule.Weekday,
		MonthDay:        schedule.MonthDay,
		NextSendAt:      unixSeconds(schedule.NextSendAt),
		LastSentAt:      unixSeconds(schedule.LastSentAt),
		LastStatus:      lastStatus,
		LastError:       schedule.LastError,
		EmailEnabled:    handlers.emailEnabled,
		Persisted:       persisted,
	}
}

func defaultTrafficReportSchedule(site model.Site, recipientEmail string) model.TrafficReportSchedule {
	recipientEmail = strings.TrimSpace(recipientEmail)
	schedule, scheduleErr := model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:         site.ID,
		Enabled:        false,
		Frequency:      model.TrafficReportFrequencyDaily,
		RecipientEmail: recipientEmail,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       model.DefaultTrafficReportSendHour,
		SendMinute:     model.DefaultTrafficReportSendMinute,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
	})
	if scheduleErr != nil {
		return model.TrafficReportSchedule{
			SiteID:          site.ID,
			Enabled:         false,
			Frequency:       model.TrafficReportFrequencyDaily,
			RecipientEmail:  strings.ToLower(strings.TrimSpace(recipientEmail)),
			RecipientMode:   model.TrafficReportRecipientModeManager,
			RecipientEmails: "[]",
			Timezone:        model.DefaultTrafficReportTimezone,
			SendHour:        model.DefaultTrafficReportSendHour,
			SendMinute:      model.DefaultTrafficReportSendMinute,
			Weekday:         model.DefaultTrafficReportWeekday,
			MonthDay:        model.DefaultTrafficReportMonthDay,
			LastStatus:      model.TrafficReportStatusPending,
		}
	}
	return schedule
}

func buildTrafficReportScheduleFromRequest(site model.Site, recipientEmail string, payload trafficReportScheduleRequest, referenceTime time.Time) (model.TrafficReportSchedule, error) {
	frequency := strings.TrimSpace(payload.Frequency)
	if frequency == "" {
		frequency = model.TrafficReportFrequencyDaily
	}
	timezoneName := strings.TrimSpace(payload.Timezone)
	if timezoneName == "" {
		timezoneName = model.DefaultTrafficReportTimezone
	}
	return model.NewTrafficReportSchedule(model.TrafficReportScheduleInput{
		SiteID:          site.ID,
		Enabled:         payload.Enabled,
		Frequency:       frequency,
		RecipientEmail:  recipientEmail,
		RecipientMode:   payload.RecipientMode,
		RecipientEmails: payload.RecipientEmails,
		Timezone:        timezoneName,
		SendHour:        intValueOrDefault(payload.SendHour, model.DefaultTrafficReportSendHour),
		SendMinute:      intValueOrDefault(payload.SendMinute, model.DefaultTrafficReportSendMinute),
		Weekday:         intValueOrDefault(payload.Weekday, model.DefaultTrafficReportWeekday),
		MonthDay:        intValueOrDefault(payload.MonthDay, model.DefaultTrafficReportMonthDay),
		ReferenceTime:   referenceTime,
	})
}

func (handlers *TrafficReportHandlers) validateTrafficReportScheduleRecipients(ctx context.Context, site model.Site, schedule model.TrafficReportSchedule) error {
	if schedule.RecipientModeValue() != model.TrafficReportRecipientModeSelected {
		return nil
	}
	selectedRecipients := schedule.SelectedRecipientEmails()
	if len(selectedRecipients) == 0 {
		return fmt.Errorf("%w: missing recipient_emails", model.ErrInvalidTrafficReportSchedule)
	}
	teamRecipientSet, teamRecipientErr := siteTeamMemberRecipientSet(ctx, handlers.database, site.ID)
	if teamRecipientErr != nil {
		return teamRecipientErr
	}
	for _, recipientEmail := range selectedRecipients {
		if _, exists := teamRecipientSet[recipientEmail]; !exists {
			return fmt.Errorf("%w: unknown recipient_email", model.ErrInvalidTrafficReportSchedule)
		}
	}
	return nil
}

func intValueOrDefault(value *int, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	return *value
}

func unixSeconds(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UTC().Unix()
}

type trafficReportRepository struct {
	database *gorm.DB
}

func (repository trafficReportRepository) PendingJobs(ctx context.Context, maxRetries int, now time.Time) ([]scheduler.Job, error) {
	var schedules []model.TrafficReportSchedule
	err := repository.database.WithContext(ctx).
		Where("enabled = ? AND next_send_at <= ? AND retry_count < ?", true, now.UTC(), maxRetries).
		Order("next_send_at asc").
		Find(&schedules).Error
	if err != nil {
		return nil, err
	}
	jobs := make([]scheduler.Job, 0, len(schedules))
	for _, schedule := range schedules {
		nextSendAt := schedule.NextSendAt.UTC()
		scheduleSnapshot := schedule
		jobs = append(jobs, scheduler.Job{
			ID:              schedule.ID,
			ScheduledFor:    &nextSendAt,
			RetryCount:      schedule.RetryCount,
			LastAttemptedAt: schedule.LastAttemptedAt,
			Payload:         scheduleSnapshot,
		})
	}
	var portfolioSchedules []model.PortfolioTrafficReportSchedule
	err = repository.database.WithContext(ctx).
		Where("enabled = ? AND next_send_at <= ? AND retry_count < ?", true, now.UTC(), maxRetries).
		Order("next_send_at asc").
		Find(&portfolioSchedules).Error
	if err != nil {
		return nil, err
	}
	for _, schedule := range portfolioSchedules {
		nextSendAt := schedule.NextSendAt.UTC()
		scheduleSnapshot := schedule
		jobs = append(jobs, scheduler.Job{
			ID:              schedule.ID,
			ScheduledFor:    &nextSendAt,
			RetryCount:      schedule.RetryCount,
			LastAttemptedAt: schedule.LastAttemptedAt,
			Payload:         scheduleSnapshot,
		})
	}
	return jobs, nil
}

func (repository trafficReportRepository) ClaimJobForAttempt(ctx context.Context, job scheduler.Job, attemptedAt time.Time) (bool, error) {
	switch schedule := job.Payload.(type) {
	case model.TrafficReportSchedule:
		result := repository.database.WithContext(ctx).
			Model(&model.TrafficReportSchedule{}).
			Where("id = ? AND retry_count = ? AND next_send_at = ?", job.ID, schedule.RetryCount, schedule.NextSendAt).
			Update("last_attempted_at", attemptedAt.UTC())
		if result.Error != nil {
			return false, result.Error
		}
		return result.RowsAffected > 0, nil
	case model.PortfolioTrafficReportSchedule:
		result := repository.database.WithContext(ctx).
			Model(&model.PortfolioTrafficReportSchedule{}).
			Where("id = ? AND retry_count = ? AND next_send_at = ?", job.ID, schedule.RetryCount, schedule.NextSendAt).
			Update("last_attempted_at", attemptedAt.UTC())
		if result.Error != nil {
			return false, result.Error
		}
		return result.RowsAffected > 0, nil
	default:
		return false, nil
	}
}

func (repository trafficReportRepository) ApplyAttemptResult(ctx context.Context, job scheduler.Job, update scheduler.AttemptUpdate) error {
	switch job.Payload.(type) {
	case model.TrafficReportSchedule:
		var schedule model.TrafficReportSchedule
		if err := repository.database.WithContext(ctx).First(&schedule, "id = ?", job.ID).Error; err != nil {
			return err
		}

		updates, updatesErr := trafficReportAttemptUpdates(schedule, update)
		if updatesErr != nil {
			return updatesErr
		}
		return repository.database.WithContext(ctx).Model(&model.TrafficReportSchedule{}).Where("id = ?", job.ID).Updates(updates).Error
	case model.PortfolioTrafficReportSchedule:
		var schedule model.PortfolioTrafficReportSchedule
		if err := repository.database.WithContext(ctx).First(&schedule, "id = ?", job.ID).Error; err != nil {
			return err
		}
		updates, updatesErr := portfolioTrafficReportAttemptUpdates(schedule, update)
		if updatesErr != nil {
			return updatesErr
		}
		return repository.database.WithContext(ctx).Model(&model.PortfolioTrafficReportSchedule{}).Where("id = ?", job.ID).Updates(updates).Error
	default:
		return fmt.Errorf("traffic_report_dispatch: invalid payload")
	}
}

type trafficReportDispatcher struct {
	database      *gorm.DB
	statsProvider SiteStatisticsProvider
	emailSender   EmailSender
}

type trafficReportDeliveryError struct {
	deliveredCount int
	failedCount    int
	deliveryErr    error
}

func (deliveryErr trafficReportDeliveryError) Error() string {
	return fmt.Sprintf("traffic_report_dispatch: %d delivered, %d failed: %v", deliveryErr.deliveredCount, deliveryErr.failedCount, deliveryErr.deliveryErr)
}

func (deliveryErr trafficReportDeliveryError) Unwrap() error {
	return deliveryErr.deliveryErr
}

func (dispatcher trafficReportDispatcher) Attempt(ctx context.Context, job scheduler.Job) (scheduler.DispatchResult, error) {
	switch schedule := job.Payload.(type) {
	case model.TrafficReportSchedule:
		var site model.Site
		if err := dispatcher.database.WithContext(ctx).First(&site, "id = ?", schedule.SiteID).Error; err != nil {
			return scheduler.DispatchResult{Status: model.TrafficReportStatusFailed}, err
		}
		if sendErr := dispatcher.sendSchedule(ctx, site, schedule); sendErr != nil {
			var deliveryErr trafficReportDeliveryError
			if errors.As(sendErr, &deliveryErr) && deliveryErr.deliveredCount > 0 {
				return scheduler.DispatchResult{Status: model.TrafficReportStatusSent}, sendErr
			}
			return scheduler.DispatchResult{Status: model.TrafficReportStatusFailed}, sendErr
		}
		return scheduler.DispatchResult{Status: model.TrafficReportStatusSent}, nil
	case model.PortfolioTrafficReportSchedule:
		if sendErr := dispatcher.sendPortfolioSchedule(ctx, schedule); sendErr != nil {
			return scheduler.DispatchResult{Status: model.TrafficReportStatusFailed}, sendErr
		}
		return scheduler.DispatchResult{Status: model.TrafficReportStatusSent}, nil
	default:
		return scheduler.DispatchResult{Status: model.TrafficReportStatusFailed}, fmt.Errorf("traffic_report_dispatch: invalid payload")
	}
}

func (dispatcher trafficReportDispatcher) sendSchedule(ctx context.Context, site model.Site, schedule model.TrafficReportSchedule) error {
	if dispatcher.emailSender == nil {
		return fmt.Errorf("traffic_report_dispatch: email sender is not configured")
	}
	statsProvider := dispatcher.statsProvider
	if statsProvider == nil {
		statsProvider = NewDatabaseSiteStatisticsProvider(dispatcher.database)
	}
	report, reportErr := buildTrafficReportEmail(ctx, statsProvider, site, schedule)
	if reportErr != nil {
		return reportErr
	}
	recipients, recipientsErr := trafficReportScheduleRecipients(ctx, dispatcher.database, site, schedule)
	if recipientsErr != nil {
		return recipientsErr
	}
	var deliveryErr error
	deliveredCount := 0
	failedCount := 0
	for _, recipient := range recipients {
		if sendErr := dispatcher.emailSender.SendEmail(ctx, recipient, report.subject, report.message); sendErr != nil {
			failedCount += 1
			deliveryErr = errors.Join(deliveryErr, sendErr)
			continue
		}
		deliveredCount += 1
	}
	if deliveryErr != nil {
		if deliveredCount > 0 {
			return trafficReportDeliveryError{
				deliveredCount: deliveredCount,
				failedCount:    failedCount,
				deliveryErr:    deliveryErr,
			}
		}
		return deliveryErr
	}
	return nil
}

func trafficReportScheduleRecipients(ctx context.Context, database *gorm.DB, site model.Site, schedule model.TrafficReportSchedule) ([]string, error) {
	return siteNotificationRecipients(ctx, database, site, siteRecipientConfig{
		recipientEmail:   schedule.RecipientEmail,
		recipientMode:    schedule.RecipientModeValue(),
		recipientEmails:  schedule.SelectedRecipientEmails(),
		noRecipientError: "traffic_report_dispatch: no recipients",
	})
}

func (dispatcher trafficReportDispatcher) sendPortfolioSchedule(ctx context.Context, schedule model.PortfolioTrafficReportSchedule) error {
	if dispatcher.emailSender == nil {
		return fmt.Errorf("traffic_report_dispatch: email sender is not configured")
	}
	report, reportErr := buildPortfolioTrafficReportEmail(ctx, dispatcher.database, schedule)
	if reportErr != nil {
		return reportErr
	}
	return dispatcher.emailSender.SendEmail(ctx, schedule.RecipientEmail, report.subject, report.message)
}

func trafficReportAttemptUpdates(schedule model.TrafficReportSchedule, update scheduler.AttemptUpdate) (map[string]any, error) {
	updates := map[string]any{
		"last_status":         update.Status,
		"last_attempted_at":   update.LastAttemptedAt.UTC(),
		"provider_message_id": update.ProviderMessageID,
	}
	if update.Status == model.TrafficReportStatusSent {
		nextSendAt, nextErr := schedule.NextAfter(update.LastAttemptedAt.UTC())
		if nextErr != nil {
			return nil, nextErr
		}
		updates["last_sent_at"] = update.LastAttemptedAt.UTC()
		updates["next_send_at"] = nextSendAt
		updates["retry_count"] = 0
		updates["last_error"] = ""
	} else {
		updates["retry_count"] = update.RetryCount
		updates["last_error"] = trafficReportStatusDispatchFailed
	}
	return updates, nil
}

func portfolioTrafficReportAttemptUpdates(schedule model.PortfolioTrafficReportSchedule, update scheduler.AttemptUpdate) (map[string]any, error) {
	updates := map[string]any{
		"last_status":         update.Status,
		"last_attempted_at":   update.LastAttemptedAt.UTC(),
		"provider_message_id": update.ProviderMessageID,
	}
	if update.Status == model.TrafficReportStatusSent {
		nextSendAt, nextErr := schedule.NextAfter(update.LastAttemptedAt.UTC())
		if nextErr != nil {
			return nil, nextErr
		}
		updates["last_sent_at"] = update.LastAttemptedAt.UTC()
		updates["next_send_at"] = nextSendAt
		updates["retry_count"] = 0
		updates["last_error"] = ""
	} else {
		updates["retry_count"] = update.RetryCount
		updates["last_error"] = trafficReportStatusDispatchFailed
	}
	return updates, nil
}

type trafficReportEmail struct {
	subject string
	message string
}

type trafficReportEmailTemplateData struct {
	FrequencyLabel string
	SiteName       string
	WindowDays     int
	PageViews      int64
	UniqueVisitors int64
	TopPages       []TopPageStat
	Devices        []DeviceTypeStat
	Locations      []LocationDistributionStat
}

func buildTrafficReportEmail(ctx context.Context, statsProvider SiteStatisticsProvider, site model.Site, schedule model.TrafficReportSchedule) (trafficReportEmail, error) {
	windowDays := schedule.ReportWindowDays()
	pageViews, pageViewsErr := statsProvider.VisitCountForDays(ctx, site.ID, windowDays)
	if pageViewsErr != nil {
		return trafficReportEmail{}, fmt.Errorf("traffic_report_email page_views: %w", pageViewsErr)
	}
	uniqueVisitors, uniqueVisitorsErr := statsProvider.UniqueVisitorCountForDays(ctx, site.ID, windowDays)
	if uniqueVisitorsErr != nil {
		return trafficReportEmail{}, fmt.Errorf("traffic_report_email unique_visitors: %w", uniqueVisitorsErr)
	}
	topPages, topPagesErr := statsProvider.TopPagesForDays(ctx, site.ID, windowDays, trafficReportTopPagesLimit)
	if topPagesErr != nil {
		return trafficReportEmail{}, fmt.Errorf("traffic_report_email top_pages: %w", topPagesErr)
	}
	devices, devicesErr := statsProvider.DeviceBreakdownForDays(ctx, site.ID, windowDays, trafficReportTopPagesLimit)
	if devicesErr != nil {
		return trafficReportEmail{}, fmt.Errorf("traffic_report_email devices: %w", devicesErr)
	}
	locations, locationsErr := statsProvider.LocationDistributionForDays(ctx, site.ID, windowDays, trafficReportTopPagesLimit)
	if locationsErr != nil {
		return trafficReportEmail{}, fmt.Errorf("traffic_report_email locations: %w", locationsErr)
	}

	templateData := trafficReportEmailTemplateData{
		FrequencyLabel: trafficReportFrequencyLabel(schedule.Frequency),
		SiteName:       strings.TrimSpace(site.Name),
		WindowDays:     windowDays,
		PageViews:      pageViews,
		UniqueVisitors: uniqueVisitors,
		TopPages:       topPages,
		Devices:        devices.DeviceTypes,
		Locations:      locations,
	}

	subject, subjectErr := renderTrafficReportEmailTemplate("subject", templateData)
	if subjectErr != nil {
		return trafficReportEmail{}, subjectErr
	}
	message, messageErr := renderTrafficReportEmailTemplate("body", templateData)
	if messageErr != nil {
		return trafficReportEmail{}, messageErr
	}

	return trafficReportEmail{subject: subject, message: message}, nil
}

func trafficReportFrequencyLabel(frequency string) string {
	switch frequency {
	case model.TrafficReportFrequencyDaily:
		return "Daily"
	case model.TrafficReportFrequencyWeekly:
		return "Weekly"
	case model.TrafficReportFrequencyMonthly:
		return "Monthly"
	default:
		return strings.TrimSpace(frequency)
	}
}

func renderTrafficReportEmailTemplate(templateName string, data trafficReportEmailTemplateData) (string, error) {
	var buffer bytes.Buffer
	if templateErr := trafficReportEmailTemplate.ExecuteTemplate(&buffer, templateName, data); templateErr != nil {
		return "", fmt.Errorf("traffic_report_email_template %s: %w", templateName, templateErr)
	}
	return strings.TrimSpace(buffer.String()), nil
}

// TrafficReportScheduler wraps the shared persisted scheduler worker.
type TrafficReportScheduler struct {
	worker *scheduler.Worker
	cancel context.CancelFunc
	group  sync.WaitGroup
}

// NewTrafficReportScheduler creates the persisted traffic report scheduler.
func NewTrafficReportScheduler(database *gorm.DB, logger *zap.Logger, statsProvider SiteStatisticsProvider, emailSender EmailSender, interval time.Duration, maxRetries int) (*TrafficReportScheduler, error) {
	if interval <= 0 {
		interval = defaultTrafficReportSchedulerInterval
	}
	if maxRetries <= 0 {
		maxRetries = defaultTrafficReportSchedulerRetries
	}
	if statsProvider == nil {
		statsProvider = NewDatabaseSiteStatisticsProvider(database)
	}
	worker, workerErr := scheduler.NewWorker(scheduler.Config{
		Repository:    trafficReportRepository{database: database},
		Dispatcher:    trafficReportDispatcher{database: database, statsProvider: statsProvider, emailSender: emailSender},
		Logger:        slog.Default(),
		Interval:      interval,
		MaxRetries:    maxRetries,
		SuccessStatus: model.TrafficReportStatusSent,
		FailureStatus: model.TrafficReportStatusFailed,
	})
	if workerErr != nil {
		return nil, workerErr
	}
	return &TrafficReportScheduler{worker: worker}, nil
}

// Start runs the scheduler until Stop is called or the parent context is canceled.
func (trafficScheduler *TrafficReportScheduler) Start(ctx context.Context) {
	if trafficScheduler == nil || trafficScheduler.worker == nil || trafficScheduler.cancel != nil {
		return
	}
	runtimeContext, cancel := context.WithCancel(ctx)
	trafficScheduler.cancel = cancel
	trafficScheduler.group.Add(1)
	go func() {
		defer trafficScheduler.group.Done()
		trafficScheduler.worker.Run(runtimeContext)
	}()
}

// Stop terminates the scheduler worker.
func (trafficScheduler *TrafficReportScheduler) Stop() {
	if trafficScheduler == nil {
		return
	}
	if trafficScheduler.cancel != nil {
		trafficScheduler.cancel()
	}
	trafficScheduler.group.Wait()
	trafficScheduler.cancel = nil
}

// RunOnce executes one scheduler polling cycle.
func (trafficScheduler *TrafficReportScheduler) RunOnce(ctx context.Context) {
	if trafficScheduler == nil || trafficScheduler.worker == nil {
		return
	}
	trafficScheduler.worker.RunOnce(ctx)
}
