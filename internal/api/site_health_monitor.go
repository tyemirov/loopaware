package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/task"
)

const (
	errorValueInvalidHealthMonitor       = "invalid_health_monitor"
	errorValueHealthMonitorEmailDisabled = "health_monitor_email_disabled"
	errorValueHealthMonitorCheckFailed   = "health_monitor_check_failed"

	defaultSiteHealthScanInterval = time.Minute
	siteHealthNoRecipientsError   = "site_health_alert: no recipients"
)

type siteHealthMonitorRequest struct {
	Enabled          bool     `json:"enabled"`
	TargetURL        string   `json:"target_url"`
	IntervalSeconds  *int     `json:"interval_seconds"`
	TimeoutSeconds   *int     `json:"timeout_seconds"`
	FailureThreshold *int     `json:"failure_threshold"`
	RecipientMode    string   `json:"recipient_mode"`
	RecipientEmails  []string `json:"recipient_emails"`
}

type siteHealthMonitorResponse struct {
	SiteID              string   `json:"site_id"`
	Enabled             bool     `json:"enabled"`
	TargetURL           string   `json:"target_url"`
	IntervalSeconds     int      `json:"interval_seconds"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	FailureThreshold    int      `json:"failure_threshold"`
	RecipientEmail      string   `json:"recipient_email"`
	RecipientMode       string   `json:"recipient_mode"`
	RecipientEmails     []string `json:"recipient_emails"`
	Status              string   `json:"status"`
	ConsecutiveFailures int      `json:"consecutive_failures"`
	NextCheckAt         int64    `json:"next_check_at"`
	LastCheckedAt       int64    `json:"last_checked_at"`
	LastSuccessAt       int64    `json:"last_success_at"`
	LastFailureAt       int64    `json:"last_failure_at"`
	LastStatusCode      int      `json:"last_status_code"`
	LastErrorCode       string   `json:"last_error_code"`
	LastErrorMessage    string   `json:"last_error_message"`
	LastDurationMs      int      `json:"last_duration_ms"`
	LastAlertedAt       int64    `json:"last_alerted_at"`
	EmailEnabled        bool     `json:"email_enabled"`
	Persisted           bool     `json:"persisted"`
}

// SiteHealthHandlers owns authenticated site health monitor APIs.
type SiteHealthHandlers struct {
	database     *gorm.DB
	logger       *zap.Logger
	manager      *SiteHealthManager
	emailEnabled bool
	now          func() time.Time
}

// NewSiteHealthHandlers builds site health handlers.
func NewSiteHealthHandlers(database *gorm.DB, logger *zap.Logger, manager *SiteHealthManager, emailEnabled bool) *SiteHealthHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SiteHealthHandlers{
		database:     database,
		logger:       logger,
		manager:      manager,
		emailEnabled: emailEnabled,
		now:          time.Now,
	}
}

// GetMonitor returns the saved health monitor or site-specific defaults.
func (handlers *SiteHealthHandlers) GetMonitor(context *gin.Context) {
	site, currentUser, ok := handlers.resolveAuthorizedSite(context)
	if !ok {
		return
	}
	monitor, exists, findErr := handlers.findMonitor(context.Request.Context(), site.ID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		monitor = defaultSiteHealthMonitor(site, currentUser.normalizedEmail(), handlers.now())
	}
	context.JSON(http.StatusOK, handlers.toMonitorResponse(monitor, exists))
}

// SaveMonitor validates and persists a health monitor.
func (handlers *SiteHealthHandlers) SaveMonitor(context *gin.Context) {
	site, currentUser, ok := handlers.resolveManagedSite(context)
	if !ok {
		return
	}
	var payload siteHealthMonitorRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	monitor, monitorErr := buildSiteHealthMonitorFromRequest(site, currentUser.normalizedEmail(), payload, handlers.now().UTC())
	if monitorErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidHealthMonitor})
		return
	}
	if recipientErr := handlers.validateSiteHealthRecipients(context.Request.Context(), site, monitor); recipientErr != nil {
		if errors.Is(recipientErr, model.ErrInvalidSiteHealthMonitor) || errors.Is(recipientErr, model.ErrInvalidSiteRecipient) {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidHealthMonitor})
			return
		}
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	savedMonitor, saveErr := handlers.upsertMonitor(context.Request.Context(), monitor)
	if saveErr != nil {
		handlers.logger.Warn("site_health_monitor_save_failed", zap.Error(saveErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}
	context.JSON(http.StatusOK, handlers.toMonitorResponse(savedMonitor, true))
}

// RunCheck executes one health check immediately.
func (handlers *SiteHealthHandlers) RunCheck(context *gin.Context) {
	site, currentUser, ok := handlers.resolveManagedSite(context)
	if !ok {
		return
	}
	if handlers.manager == nil {
		context.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueHealthMonitorCheckFailed})
		return
	}
	monitor, exists, findErr := handlers.findMonitor(context.Request.Context(), site.ID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		monitor = defaultSiteHealthMonitor(site, currentUser.normalizedEmail(), handlers.now())
		if _, targetErr := model.NormalizeSiteHealthTargetURL(monitor.TargetURL); targetErr != nil {
			context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidHealthMonitor})
			return
		}
	}
	checkedMonitor, checkErr := handlers.manager.RunManualCheck(context.Request.Context(), site, monitor)
	if checkErr != nil {
		handlers.logger.Warn("site_health_manual_check_failed", zap.Error(checkErr), zap.String("site_id", site.ID))
		context.JSON(http.StatusBadGateway, gin.H{jsonKeyError: errorValueHealthMonitorCheckFailed})
		return
	}
	context.JSON(http.StatusOK, handlers.toMonitorResponse(checkedMonitor, true))
}

func (handlers *SiteHealthHandlers) resolveAuthorizedSite(context *gin.Context) (model.Site, *CurrentUser, bool) {
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
	if !currentUserCanViewSite(context.Request.Context(), handlers.database, currentUser, site) {
		context.JSON(http.StatusForbidden, gin.H{jsonKeyError: errorValueNotAuthorized})
		return model.Site{}, nil, false
	}
	return site, currentUser, true
}

func (handlers *SiteHealthHandlers) resolveManagedSite(context *gin.Context) (model.Site, *CurrentUser, bool) {
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

func (handlers *SiteHealthHandlers) findMonitor(ctx context.Context, siteID string) (model.SiteHealthMonitor, bool, error) {
	var monitor model.SiteHealthMonitor
	err := handlers.database.WithContext(ctx).First(&monitor, "site_id = ?", siteID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SiteHealthMonitor{}, false, nil
	}
	if err != nil {
		return model.SiteHealthMonitor{}, false, err
	}
	return monitor, true, nil
}

func (handlers *SiteHealthHandlers) upsertMonitor(ctx context.Context, monitor model.SiteHealthMonitor) (model.SiteHealthMonitor, error) {
	var existing model.SiteHealthMonitor
	findErr := handlers.database.WithContext(ctx).First(&existing, "site_id = ?", monitor.SiteID).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		if createErr := handlers.database.WithContext(ctx).Create(&monitor).Error; createErr != nil {
			return model.SiteHealthMonitor{}, createErr
		}
		return monitor, nil
	}
	if findErr != nil {
		return model.SiteHealthMonitor{}, findErr
	}

	targetChanged := !strings.EqualFold(strings.TrimSpace(existing.TargetURL), strings.TrimSpace(monitor.TargetURL))
	existing.Enabled = monitor.Enabled
	existing.TargetURL = monitor.TargetURL
	existing.IntervalSeconds = monitor.IntervalSeconds
	existing.TimeoutSeconds = monitor.TimeoutSeconds
	existing.FailureThreshold = monitor.FailureThreshold
	existing.RecipientEmail = monitor.RecipientEmail
	existing.RecipientMode = monitor.RecipientMode
	existing.RecipientEmails = monitor.RecipientEmails
	if monitor.Enabled {
		existing.NextCheckAt = monitor.NextCheckAt
	} else {
		existing.NextCheckAt = time.Time{}
	}
	if targetChanged {
		resetSiteHealthMonitorState(&existing)
	}
	if saveErr := handlers.database.WithContext(ctx).Save(&existing).Error; saveErr != nil {
		return model.SiteHealthMonitor{}, saveErr
	}
	return existing, nil
}

func (handlers *SiteHealthHandlers) validateSiteHealthRecipients(ctx context.Context, site model.Site, monitor model.SiteHealthMonitor) error {
	if monitor.RecipientModeValue() != model.SiteRecipientModeSelected {
		return nil
	}
	selectedRecipients := monitor.SelectedRecipientEmails()
	if len(selectedRecipients) == 0 {
		return fmt.Errorf("%w: missing recipient_emails", model.ErrInvalidSiteHealthMonitor)
	}
	teamRecipientSet, teamRecipientErr := siteTeamMemberRecipientSet(ctx, handlers.database, site.ID)
	if teamRecipientErr != nil {
		return teamRecipientErr
	}
	for _, recipientEmail := range selectedRecipients {
		if _, exists := teamRecipientSet[recipientEmail]; !exists {
			return fmt.Errorf("%w: unknown recipient_email", model.ErrInvalidSiteHealthMonitor)
		}
	}
	return nil
}

func (handlers *SiteHealthHandlers) toMonitorResponse(monitor model.SiteHealthMonitor, persisted bool) siteHealthMonitorResponse {
	status := strings.TrimSpace(monitor.Status)
	if status == "" {
		status = model.SiteHealthStatusUnknown
	}
	return siteHealthMonitorResponse{
		SiteID:              monitor.SiteID,
		Enabled:             monitor.Enabled,
		TargetURL:           monitor.TargetURL,
		IntervalSeconds:     monitor.IntervalSeconds,
		TimeoutSeconds:      monitor.TimeoutSeconds,
		FailureThreshold:    monitor.FailureThreshold,
		RecipientEmail:      monitor.RecipientEmail,
		RecipientMode:       monitor.RecipientModeValue(),
		RecipientEmails:     monitor.SelectedRecipientEmails(),
		Status:              status,
		ConsecutiveFailures: monitor.ConsecutiveFailures,
		NextCheckAt:         unixSeconds(monitor.NextCheckAt),
		LastCheckedAt:       unixSeconds(monitor.LastCheckedAt),
		LastSuccessAt:       unixSeconds(monitor.LastSuccessAt),
		LastFailureAt:       unixSeconds(monitor.LastFailureAt),
		LastStatusCode:      monitor.LastStatusCode,
		LastErrorCode:       monitor.LastErrorCode,
		LastErrorMessage:    monitor.LastErrorMessage,
		LastDurationMs:      monitor.LastDurationMs,
		LastAlertedAt:       unixSeconds(monitor.LastAlertedAt),
		EmailEnabled:        handlers.emailEnabled,
		Persisted:           persisted,
	}
}

func defaultSiteHealthMonitor(site model.Site, recipientEmail string, referenceTime time.Time) model.SiteHealthMonitor {
	targetURL := strings.TrimSpace(primaryAllowedOrigin(site.AllowedOrigin))
	monitor, monitorErr := model.NewSiteHealthMonitor(model.SiteHealthMonitorInput{
		SiteID:           site.ID,
		Enabled:          false,
		TargetURL:        targetURL,
		IntervalSeconds:  model.DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   model.DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: model.DefaultSiteHealthFailureThreshold,
		RecipientEmail:   recipientEmail,
		ReferenceTime:    referenceTime,
	})
	if monitorErr == nil {
		return monitor
	}
	recipient, recipientErr := model.NormalizeSiteRecipientEmail(recipientEmail)
	if recipientErr != nil {
		recipient = strings.ToLower(strings.TrimSpace(recipientEmail))
	}
	return model.SiteHealthMonitor{
		ID:               storage.NewID(),
		SiteID:           site.ID,
		Enabled:          false,
		TargetURL:        targetURL,
		IntervalSeconds:  model.DefaultSiteHealthIntervalSeconds,
		TimeoutSeconds:   model.DefaultSiteHealthTimeoutSeconds,
		FailureThreshold: model.DefaultSiteHealthFailureThreshold,
		RecipientEmail:   recipient,
		RecipientMode:    model.SiteRecipientModeManager,
		RecipientEmails:  "[]",
		Status:           model.SiteHealthStatusUnknown,
	}
}

func buildSiteHealthMonitorFromRequest(site model.Site, recipientEmail string, payload siteHealthMonitorRequest, referenceTime time.Time) (model.SiteHealthMonitor, error) {
	targetURL := strings.TrimSpace(payload.TargetURL)
	if targetURL == "" {
		targetURL = primaryAllowedOrigin(site.AllowedOrigin)
	}
	return model.NewSiteHealthMonitor(model.SiteHealthMonitorInput{
		SiteID:           site.ID,
		Enabled:          payload.Enabled,
		TargetURL:        targetURL,
		IntervalSeconds:  intValueOrDefault(payload.IntervalSeconds, model.DefaultSiteHealthIntervalSeconds),
		TimeoutSeconds:   intValueOrDefault(payload.TimeoutSeconds, model.DefaultSiteHealthTimeoutSeconds),
		FailureThreshold: intValueOrDefault(payload.FailureThreshold, model.DefaultSiteHealthFailureThreshold),
		RecipientEmail:   recipientEmail,
		RecipientMode:    payload.RecipientMode,
		RecipientEmails:  payload.RecipientEmails,
		ReferenceTime:    referenceTime,
	})
}

func resetSiteHealthMonitorState(monitor *model.SiteHealthMonitor) {
	monitor.Status = model.SiteHealthStatusUnknown
	monitor.ConsecutiveFailures = 0
	monitor.LastCheckedAt = time.Time{}
	monitor.LastSuccessAt = time.Time{}
	monitor.LastFailureAt = time.Time{}
	monitor.LastStatusCode = 0
	monitor.LastErrorCode = ""
	monitor.LastErrorMessage = ""
	monitor.LastDurationMs = 0
	monitor.LastAlertedStatus = ""
	monitor.LastAlertedAt = time.Time{}
}

// SiteHealthManager runs due checks and records state transitions.
type SiteHealthManager struct {
	database     *gorm.DB
	logger       *zap.Logger
	prober       SiteHealthProber
	emailSender  EmailSender
	emailEnabled bool
	scanInterval time.Duration
	now          func() time.Time
	inFlight     sync.Map
	scheduler    *task.Scheduler
	startOnce    sync.Once
	stopOnce     sync.Once
}

type SiteHealthManagerOption func(*SiteHealthManager)

// NewSiteHealthManager creates a health monitor manager.
func NewSiteHealthManager(database *gorm.DB, logger *zap.Logger, prober SiteHealthProber, emailSender EmailSender, emailEnabled bool, options ...SiteHealthManagerOption) *SiteHealthManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	if prober == nil {
		prober = NewHTTPHealthProber(nil)
	}
	manager := &SiteHealthManager{
		database:     database,
		logger:       logger,
		prober:       prober,
		emailSender:  emailSender,
		emailEnabled: emailEnabled,
		scanInterval: defaultSiteHealthScanInterval,
		now:          time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}
	manager.scheduler = task.NewScheduler(manager.scanInterval, manager.RunDueChecks)
	return manager
}

// WithSiteHealthScanInterval overrides the scheduler polling interval.
func WithSiteHealthScanInterval(interval time.Duration) SiteHealthManagerOption {
	return func(manager *SiteHealthManager) {
		if interval > 0 {
			manager.scanInterval = interval
		}
	}
}

// WithSiteHealthClock overrides manager time.
func WithSiteHealthClock(clock func() time.Time) SiteHealthManagerOption {
	return func(manager *SiteHealthManager) {
		if clock != nil {
			manager.now = clock
		}
	}
}

// Start runs the recurring health monitor scheduler.
func (manager *SiteHealthManager) Start(ctx context.Context) {
	if manager == nil || manager.scheduler == nil {
		return
	}
	manager.startOnce.Do(func() {
		manager.scheduler.Start(ctx)
	})
}

// Stop terminates the health monitor scheduler.
func (manager *SiteHealthManager) Stop() {
	if manager == nil || manager.scheduler == nil {
		return
	}
	manager.stopOnce.Do(func() {
		manager.scheduler.Stop()
	})
}

// TriggerScheduledChecks asks the scheduler to run one polling cycle soon.
func (manager *SiteHealthManager) TriggerScheduledChecks() {
	if manager == nil || manager.scheduler == nil {
		return
	}
	manager.scheduler.Trigger()
}

// RunDueChecks executes due monitor checks once.
func (manager *SiteHealthManager) RunDueChecks(ctx context.Context) {
	if manager == nil || manager.database == nil || manager.prober == nil {
		return
	}
	now := manager.now().UTC()
	var monitors []model.SiteHealthMonitor
	if queryErr := manager.database.WithContext(ctx).
		Where("enabled = ? AND next_check_at <= ?", true, now).
		Order("next_check_at asc").
		Find(&monitors).Error; queryErr != nil {
		manager.logger.Warn("site_health_due_query_failed", zap.Error(queryErr))
		return
	}
	for _, monitor := range monitors {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if _, inFlight := manager.inFlight.LoadOrStore(monitor.ID, struct{}{}); inFlight {
			continue
		}
		func() {
			defer manager.inFlight.Delete(monitor.ID)
			var site model.Site
			if siteErr := manager.database.WithContext(ctx).First(&site, "id = ?", monitor.SiteID).Error; siteErr != nil {
				if !errors.Is(siteErr, gorm.ErrRecordNotFound) {
					manager.logger.Warn("site_health_load_site_failed", zap.Error(siteErr), zap.String("site_id", monitor.SiteID))
				}
				return
			}
			if _, checkErr := manager.runCheck(ctx, site, monitor, true); checkErr != nil {
				manager.logger.Warn("site_health_scheduled_check_failed", zap.Error(checkErr), zap.String("site_id", monitor.SiteID))
			}
		}()
	}
}

// RunManualCheck executes one immediate health check and persists its result.
func (manager *SiteHealthManager) RunManualCheck(ctx context.Context, site model.Site, monitor model.SiteHealthMonitor) (model.SiteHealthMonitor, error) {
	if manager == nil {
		return model.SiteHealthMonitor{}, errors.New("site health manager is not configured")
	}
	return manager.runCheck(ctx, site, monitor, monitor.Enabled)
}

func (manager *SiteHealthManager) runCheck(ctx context.Context, site model.Site, monitor model.SiteHealthMonitor, allowNotifications bool) (model.SiteHealthMonitor, error) {
	if manager.database == nil || manager.prober == nil {
		return model.SiteHealthMonitor{}, errors.New("site health manager dependencies are missing")
	}
	result := manager.prober.Probe(ctx, monitor.TargetURL, monitor.CheckTimeout())
	updatedMonitor, eventKind := applySiteHealthProbeResult(monitor, result)
	if saveErr := manager.persistCheckResult(ctx, updatedMonitor, result, eventKind); saveErr != nil {
		return model.SiteHealthMonitor{}, saveErr
	}
	alertKind := pendingSiteHealthAlertKind(updatedMonitor)
	if alertKind == "" || !allowNotifications {
		return updatedMonitor, nil
	}
	alertedMonitor, notifyErr := manager.sendTransitionAlert(ctx, site, updatedMonitor, result, alertKind)
	if notifyErr != nil {
		return updatedMonitor, notifyErr
	}
	return alertedMonitor, nil
}

func pendingSiteHealthAlertKind(monitor model.SiteHealthMonitor) string {
	switch monitor.Status {
	case model.SiteHealthStatusDown:
		if monitor.LastAlertedStatus != model.SiteHealthStatusDown {
			return model.SiteHealthEventKindDown
		}
	case model.SiteHealthStatusUp:
		if monitor.LastAlertedStatus == model.SiteHealthStatusDown {
			return model.SiteHealthEventKindRecovered
		}
	}
	return ""
}

func applySiteHealthProbeResult(monitor model.SiteHealthMonitor, result SiteHealthProbeResult) (model.SiteHealthMonitor, string) {
	checkedAt := result.CheckedAt
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	if monitor.FailureThreshold <= 0 {
		monitor.FailureThreshold = model.DefaultSiteHealthFailureThreshold
	}
	previousStatus := strings.TrimSpace(monitor.Status)
	if previousStatus == "" {
		previousStatus = model.SiteHealthStatusUnknown
	}
	monitor.LastCheckedAt = checkedAt.UTC()
	monitor.LastStatusCode = result.StatusCode
	monitor.LastDurationMs = durationMilliseconds(result.Duration)
	if result.Success {
		monitor.ConsecutiveFailures = 0
		monitor.LastSuccessAt = checkedAt.UTC()
		monitor.LastErrorCode = ""
		monitor.LastErrorMessage = ""
		monitor.Status = model.SiteHealthStatusUp
		if previousStatus == model.SiteHealthStatusDown {
			if monitor.Enabled {
				monitor.NextCheckAt = checkedAt.UTC().Add(monitor.CheckInterval())
			} else {
				monitor.NextCheckAt = time.Time{}
			}
			return monitor, model.SiteHealthEventKindRecovered
		}
	} else {
		monitor.ConsecutiveFailures += 1
		monitor.LastFailureAt = checkedAt.UTC()
		monitor.LastErrorCode = model.TruncateSiteHealthErrorCode(result.ErrorCode)
		monitor.LastErrorMessage = model.TruncateSiteHealthErrorMessage(result.ErrorMessage)
		if monitor.ConsecutiveFailures >= monitor.FailureThreshold {
			monitor.Status = model.SiteHealthStatusDown
			if previousStatus != model.SiteHealthStatusDown {
				if monitor.Enabled {
					monitor.NextCheckAt = checkedAt.UTC().Add(monitor.CheckInterval())
				} else {
					monitor.NextCheckAt = time.Time{}
				}
				return monitor, model.SiteHealthEventKindDown
			}
		} else {
			monitor.Status = previousStatus
		}
	}
	if monitor.Enabled {
		monitor.NextCheckAt = checkedAt.UTC().Add(monitor.CheckInterval())
	} else {
		monitor.NextCheckAt = time.Time{}
	}
	return monitor, ""
}

func (manager *SiteHealthManager) persistCheckResult(ctx context.Context, monitor model.SiteHealthMonitor, result SiteHealthProbeResult, eventKind string) error {
	return manager.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if saveErr := transaction.Save(&monitor).Error; saveErr != nil {
			return saveErr
		}
		if eventKind == "" {
			return nil
		}
		event := model.SiteHealthEvent{
			ID:               storage.NewID(),
			SiteID:           monitor.SiteID,
			MonitorID:        monitor.ID,
			Kind:             eventKind,
			Status:           monitor.Status,
			TargetURL:        monitor.TargetURL,
			HTTPStatus:       result.StatusCode,
			ErrorCode:        monitor.LastErrorCode,
			ErrorMessage:     monitor.LastErrorMessage,
			DurationMs:       monitor.LastDurationMs,
			ConsecutiveFails: monitor.ConsecutiveFailures,
			CreatedAt:        monitor.LastCheckedAt,
		}
		return transaction.Create(&event).Error
	})
}

func (manager *SiteHealthManager) sendTransitionAlert(ctx context.Context, site model.Site, monitor model.SiteHealthMonitor, result SiteHealthProbeResult, eventKind string) (model.SiteHealthMonitor, error) {
	if !manager.emailEnabled || manager.emailSender == nil {
		return monitor, nil
	}
	if eventKind == model.SiteHealthEventKindDown && monitor.LastAlertedStatus == model.SiteHealthStatusDown {
		return monitor, nil
	}
	if eventKind == model.SiteHealthEventKindRecovered && monitor.LastAlertedStatus != model.SiteHealthStatusDown {
		return monitor, nil
	}
	recipients, recipientsErr := siteNotificationRecipients(ctx, manager.database, site, siteRecipientConfig{
		recipientEmail:   monitor.RecipientEmail,
		recipientMode:    monitor.RecipientModeValue(),
		recipientEmails:  monitor.SelectedRecipientEmails(),
		noRecipientError: siteHealthNoRecipientsError,
	})
	if recipientsErr != nil {
		return monitor, recipientsErr
	}
	subject, message := buildSiteHealthAlertEmail(site, monitor, result, eventKind)
	var sendErr error
	for _, recipient := range recipients {
		if err := manager.emailSender.SendEmail(ctx, recipient, subject, message); err != nil {
			sendErr = errors.Join(sendErr, err)
		}
	}
	if sendErr != nil {
		return monitor, sendErr
	}
	alertedStatus := model.SiteHealthStatusUp
	if eventKind == model.SiteHealthEventKindDown {
		alertedStatus = model.SiteHealthStatusDown
	}
	monitor.LastAlertedStatus = alertedStatus
	monitor.LastAlertedAt = monitor.LastCheckedAt
	updates := map[string]any{
		"last_alerted_status": monitor.LastAlertedStatus,
		"last_alerted_at":     monitor.LastAlertedAt,
	}
	if updateErr := manager.database.WithContext(ctx).Model(&model.SiteHealthMonitor{}).Where("id = ?", monitor.ID).Updates(updates).Error; updateErr != nil {
		return monitor, updateErr
	}
	return monitor, nil
}

func buildSiteHealthAlertEmail(site model.Site, monitor model.SiteHealthMonitor, result SiteHealthProbeResult, eventKind string) (string, string) {
	siteName := strings.TrimSpace(site.Name)
	if siteName == "" {
		siteName = strings.TrimSpace(site.ID)
	}
	checkedAt := monitor.LastCheckedAt.UTC().Format(time.RFC3339)
	if eventKind == model.SiteHealthEventKindRecovered {
		return "Site recovered: " + siteName, fmt.Sprintf(
			"%s recovered at %s.\n\nTarget: %s\nHTTP status: %d\nResponse time: %d ms",
			siteName,
			checkedAt,
			monitor.TargetURL,
			result.StatusCode,
			monitor.LastDurationMs,
		)
	}
	errorDetail := strings.TrimSpace(monitor.LastErrorMessage)
	if errorDetail == "" {
		errorDetail = strings.TrimSpace(monitor.LastErrorCode)
	}
	return "Site down: " + siteName, fmt.Sprintf(
		"%s is down as of %s.\n\nTarget: %s\nFailure threshold: %d consecutive checks\nConsecutive failures: %d\nHTTP status: %d\nError: %s",
		siteName,
		checkedAt,
		monitor.TargetURL,
		monitor.FailureThreshold,
		monitor.ConsecutiveFailures,
		result.StatusCode,
		errorDetail,
	)
}

func durationMilliseconds(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}
	return int(duration / time.Millisecond)
}
