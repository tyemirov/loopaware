package api

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	portfolioTrafficReportScopeOwned  = "owned"
	portfolioTrafficReportDefaultDays = 30
	portfolioTrafficReportMaxDays     = 30
	portfolioTrafficReportDefaultName = "All sites traffic"

	errorValueInvalidPortfolioTrafficReport = "invalid_portfolio_traffic_report"
	errorValueUnknownPortfolioTrafficReport = "unknown_portfolio_traffic_report"
)

//go:embed templates/portfolio_traffic_report_email.txt
var portfolioTrafficReportEmailTemplateText string

var portfolioTrafficReportEmailTemplate = template.Must(template.New("portfolio_traffic_report_email").Option("missingkey=error").Parse(portfolioTrafficReportEmailTemplateText))

type PortfolioTrafficReportResponse struct {
	ReportID           string                       `json:"report_id"`
	ReportName         string                       `json:"report_name"`
	Scope              string                       `json:"scope"`
	Days               int                          `json:"days"`
	SiteCount          int                          `json:"site_count"`
	VisitCount         int64                        `json:"visit_count"`
	UniqueVisitorCount int64                        `json:"unique_visitor_count"`
	Trend              []VisitTrendPoint            `json:"trend"`
	TopPages           []TopPageEntry               `json:"top_pages"`
	Sites              []PortfolioTrafficSiteRecord `json:"sites"`
}

type PortfolioTrafficSiteRecord struct {
	SiteID             string `json:"site_id"`
	SiteName           string `json:"site_name"`
	VisitCount         int64  `json:"visit_count"`
	UniqueVisitorCount int64  `json:"unique_visitor_count"`
}

type PortfolioTrafficReportDefinitionListResponse struct {
	Reports        []PortfolioTrafficReportDefinitionResponse `json:"reports"`
	AvailableSites []PortfolioTrafficSiteRecord               `json:"available_sites"`
}

type PortfolioTrafficReportDefinitionResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	SiteIDs   []string `json:"site_ids"`
	IsDefault bool     `json:"is_default"`
}

type portfolioTrafficReportDefinitionRequest struct {
	Name    string   `json:"name"`
	SiteIDs []string `json:"site_ids"`
}

type portfolioTrafficReportData struct {
	WindowDays     int
	SiteCount      int
	PageViews      int64
	UniqueVisitors int64
	Trend          []VisitTrendPoint
	TopPages       []TopPageEntry
	Sites          []PortfolioTrafficSiteRecord
}

type portfolioTrafficReportEmailTemplateData struct {
	FrequencyLabel string
	ReportName     string
	WindowDays     int
	SiteCount      int
	PageViews      int64
	UniqueVisitors int64
	TopPages       []TopPageEntry
	Sites          []PortfolioTrafficSiteRecord
}

type portfolioTrendRow struct {
	Day            string
	PageViews      int64
	UniqueVisitors int64
}

type portfolioCountRow struct {
	PageViews      int64
	UniqueVisitors int64
}

type portfolioSiteCountRow struct {
	SiteID         string
	PageViews      int64
	UniqueVisitors int64
}

func (handlers *TrafficReportHandlers) GetPortfolioReport(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}

	days, parseErr := parsePortfolioTrafficReportDays(context.Query("days"))
	if parseErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidDays})
		return
	}

	reportID := portfolioTrafficReportIDFromQuery(context)
	reportDefinition, sites, reportErr := handlers.resolvePortfolioReportDefinitionSites(context.Request.Context(), currentUser, reportID)
	if reportErr != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownPortfolioTrafficReport})
		return
	}
	report, buildErr := buildPortfolioTrafficReportData(context.Request.Context(), handlers.database, sites, days)
	if buildErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}

	context.JSON(http.StatusOK, PortfolioTrafficReportResponse{
		ReportID:           reportDefinition.ID,
		ReportName:         reportDefinition.Name,
		Scope:              portfolioTrafficReportScopeOwned,
		Days:               report.WindowDays,
		SiteCount:          report.SiteCount,
		VisitCount:         report.PageViews,
		UniqueVisitorCount: report.UniqueVisitors,
		Trend:              report.Trend,
		TopPages:           report.TopPages,
		Sites:              report.Sites,
	})
}

func (handlers *TrafficReportHandlers) ListPortfolioReports(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	availableSites, sitesErr := handlers.portfolioReportSites(context.Request.Context(), currentUser)
	if sitesErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	availableSiteRecords, recordsErr := portfolioSiteRows(context.Request.Context(), handlers.database, availableSites, portfolioTrafficReportDefaultDays)
	if recordsErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	definitions, definitionErr := handlers.portfolioReportDefinitions(context.Request.Context(), currentUser.normalizedEmail())
	if definitionErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	reports, reportsErr := handlers.toPortfolioReportDefinitionResponses(context.Request.Context(), availableSites, definitions)
	if reportsErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	context.JSON(http.StatusOK, PortfolioTrafficReportDefinitionListResponse{
		Reports:        reports,
		AvailableSites: availableSiteRecords,
	})
}

func (handlers *TrafficReportHandlers) CreatePortfolioReport(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	var payload portfolioTrafficReportDefinitionRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	definition, createErr := handlers.createPortfolioReportDefinition(context.Request.Context(), currentUser, payload)
	if createErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidPortfolioTrafficReport})
		return
	}
	context.JSON(http.StatusCreated, definition)
}

func (handlers *TrafficReportHandlers) UpdatePortfolioReport(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	reportID := strings.TrimSpace(context.Param("report_id"))
	if reportID == "" || reportID == model.PortfolioTrafficReportDefaultID {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidPortfolioTrafficReport})
		return
	}
	var payload portfolioTrafficReportDefinitionRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}
	definition, updateErr := handlers.updatePortfolioReportDefinition(context.Request.Context(), currentUser, reportID, payload)
	if errors.Is(updateErr, gorm.ErrRecordNotFound) {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownPortfolioTrafficReport})
		return
	}
	if updateErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidPortfolioTrafficReport})
		return
	}
	context.JSON(http.StatusOK, definition)
}

func (handlers *TrafficReportHandlers) GetPortfolioSchedule(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	reportID := portfolioTrafficReportIDFromQuery(context)
	if _, _, reportErr := handlers.resolvePortfolioReportDefinitionSites(context.Request.Context(), currentUser, reportID); reportErr != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownPortfolioTrafficReport})
		return
	}

	schedule, exists, findErr := handlers.findPortfolioSchedule(context.Request.Context(), currentUser.normalizedEmail(), reportID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		context.JSON(http.StatusOK, handlers.toPortfolioScheduleResponse(defaultPortfolioTrafficReportSchedule(currentUser.normalizedEmail(), reportID), false))
		return
	}
	schedule.RecipientEmail = currentUser.normalizedEmail()

	context.JSON(http.StatusOK, handlers.toPortfolioScheduleResponse(schedule, true))
}

func (handlers *TrafficReportHandlers) SavePortfolioSchedule(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	reportID := portfolioTrafficReportIDFromQuery(context)
	if _, _, reportErr := handlers.resolvePortfolioReportDefinitionSites(context.Request.Context(), currentUser, reportID); reportErr != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownPortfolioTrafficReport})
		return
	}

	var payload trafficReportScheduleRequest
	if bindErr := context.BindJSON(&payload); bindErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidJSON})
		return
	}

	schedule, scheduleErr := buildPortfolioTrafficReportScheduleFromRequest(currentUser.normalizedEmail(), reportID, payload, handlers.now().UTC())
	if scheduleErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{jsonKeyError: errorValueInvalidTrafficReportSchedule})
		return
	}

	savedSchedule, saveErr := handlers.upsertPortfolioSchedule(context.Request.Context(), schedule)
	if saveErr != nil {
		handlers.logger.Warn("portfolio_traffic_report_schedule_save_failed", zap.Error(saveErr))
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueSaveFailed})
		return
	}

	context.JSON(http.StatusOK, handlers.toPortfolioScheduleResponse(savedSchedule, true))
}

func (handlers *TrafficReportHandlers) SendPortfolioTestReport(context *gin.Context) {
	currentUser, ok := handlers.resolvePortfolioUser(context)
	if !ok {
		return
	}
	if !handlers.emailEnabled || handlers.emailSender == nil {
		context.JSON(http.StatusServiceUnavailable, gin.H{jsonKeyError: errorValueTrafficReportEmailDisabled})
		return
	}
	reportID := portfolioTrafficReportIDFromQuery(context)
	if _, _, reportErr := handlers.resolvePortfolioReportDefinitionSites(context.Request.Context(), currentUser, reportID); reportErr != nil {
		context.JSON(http.StatusNotFound, gin.H{jsonKeyError: errorValueUnknownPortfolioTrafficReport})
		return
	}

	schedule, exists, findErr := handlers.findPortfolioSchedule(context.Request.Context(), currentUser.normalizedEmail(), reportID)
	if findErr != nil {
		context.JSON(http.StatusInternalServerError, gin.H{jsonKeyError: errorValueQueryFailed})
		return
	}
	if !exists {
		schedule = defaultPortfolioTrafficReportSchedule(currentUser.normalizedEmail(), reportID)
	}

	dispatcher := trafficReportDispatcher{
		database:    handlers.database,
		emailSender: handlers.emailSender,
	}
	if sendErr := dispatcher.sendPortfolioSchedule(context.Request.Context(), schedule); sendErr != nil {
		handlers.logger.Warn("portfolio_traffic_report_test_send_failed", zap.Error(sendErr))
		context.JSON(http.StatusBadGateway, gin.H{jsonKeyError: errorValueTrafficReportSendFailed})
		return
	}

	context.JSON(http.StatusOK, gin.H{"status": "sent"})
}

func (handlers *TrafficReportHandlers) resolvePortfolioUser(context *gin.Context) (*CurrentUser, bool) {
	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return nil, false
	}
	if currentUser.normalizedEmail() == "" {
		context.JSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
		return nil, false
	}
	return currentUser, true
}

func portfolioTrafficReportIDFromQuery(context *gin.Context) string {
	reportID := strings.TrimSpace(context.Query("report_id"))
	if reportID == "" {
		return model.PortfolioTrafficReportDefaultID
	}
	return reportID
}

func (handlers *TrafficReportHandlers) portfolioReportSites(ctx context.Context, currentUser *CurrentUser) ([]model.Site, error) {
	normalizedEmail := currentUser.normalizedEmail()
	if normalizedEmail == "" {
		return nil, nil
	}
	var sites []model.Site
	err := handlers.database.WithContext(ctx).
		Where("(LOWER(owner_email) = ? OR LOWER(creator_email) = ?)", normalizedEmail, normalizedEmail).
		Order("created_at desc").
		Find(&sites).Error
	if err != nil {
		return nil, err
	}
	return sites, nil
}

func (handlers *TrafficReportHandlers) portfolioReportDefinitions(ctx context.Context, userEmail string) ([]model.PortfolioTrafficReportDefinition, error) {
	var definitions []model.PortfolioTrafficReportDefinition
	err := handlers.database.WithContext(ctx).
		Where("user_email = ?", strings.ToLower(strings.TrimSpace(userEmail))).
		Order("created_at asc").
		Find(&definitions).Error
	return definitions, err
}

func (handlers *TrafficReportHandlers) toPortfolioReportDefinitionResponses(ctx context.Context, availableSites []model.Site, definitions []model.PortfolioTrafficReportDefinition) ([]PortfolioTrafficReportDefinitionResponse, error) {
	availableSiteIDs := portfolioSiteIDs(availableSites)
	responses := []PortfolioTrafficReportDefinitionResponse{{
		ID:        model.PortfolioTrafficReportDefaultID,
		Name:      portfolioTrafficReportDefaultName,
		SiteIDs:   availableSiteIDs,
		IsDefault: true,
	}}
	for _, definition := range definitions {
		siteIDs, siteErr := handlers.portfolioReportDefinitionSiteIDs(ctx, definition.ID)
		if siteErr != nil {
			return nil, siteErr
		}
		responses = append(responses, PortfolioTrafficReportDefinitionResponse{
			ID:        definition.ID,
			Name:      strings.TrimSpace(definition.Name),
			SiteIDs:   filterPortfolioReportSiteIDs(siteIDs, availableSiteIDs),
			IsDefault: false,
		})
	}
	return responses, nil
}

func (handlers *TrafficReportHandlers) portfolioReportDefinitionSiteIDs(ctx context.Context, reportID string) ([]string, error) {
	var memberships []model.PortfolioTrafficReportDefinitionSite
	if err := handlers.database.WithContext(ctx).
		Where("report_id = ?", strings.TrimSpace(reportID)).
		Order("created_at asc").
		Find(&memberships).Error; err != nil {
		return nil, err
	}
	siteIDs := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		if strings.TrimSpace(membership.SiteID) != "" {
			siteIDs = append(siteIDs, membership.SiteID)
		}
	}
	return siteIDs, nil
}

func (handlers *TrafficReportHandlers) createPortfolioReportDefinition(ctx context.Context, currentUser *CurrentUser, payload portfolioTrafficReportDefinitionRequest) (PortfolioTrafficReportDefinitionResponse, error) {
	definition, definitionErr := model.NewPortfolioTrafficReportDefinition(model.PortfolioTrafficReportDefinitionInput{
		UserEmail: currentUser.normalizedEmail(),
		Name:      payload.Name,
	})
	if definitionErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, definitionErr
	}
	availableSites, sitesErr := handlers.portfolioReportSites(ctx, currentUser)
	if sitesErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, sitesErr
	}
	siteIDs, siteIDsErr := normalizePortfolioReportSiteIDs(payload.SiteIDs, portfolioSiteIDs(availableSites))
	if siteIDsErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, siteIDsErr
	}
	txErr := handlers.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&definition).Error; createErr != nil {
			return createErr
		}
		return createPortfolioReportDefinitionSites(tx, definition.ID, siteIDs)
	})
	if txErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, txErr
	}
	return PortfolioTrafficReportDefinitionResponse{
		ID:        definition.ID,
		Name:      definition.Name,
		SiteIDs:   siteIDs,
		IsDefault: false,
	}, nil
}

func (handlers *TrafficReportHandlers) updatePortfolioReportDefinition(ctx context.Context, currentUser *CurrentUser, reportID string, payload portfolioTrafficReportDefinitionRequest) (PortfolioTrafficReportDefinitionResponse, error) {
	var definition model.PortfolioTrafficReportDefinition
	if err := handlers.database.WithContext(ctx).First(&definition, "id = ? AND user_email = ?", strings.TrimSpace(reportID), currentUser.normalizedEmail()).Error; err != nil {
		return PortfolioTrafficReportDefinitionResponse{}, err
	}
	updatedDefinition, definitionErr := model.NewPortfolioTrafficReportDefinition(model.PortfolioTrafficReportDefinitionInput{
		UserEmail: currentUser.normalizedEmail(),
		Name:      payload.Name,
	})
	if definitionErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, definitionErr
	}
	availableSites, sitesErr := handlers.portfolioReportSites(ctx, currentUser)
	if sitesErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, sitesErr
	}
	siteIDs, siteIDsErr := normalizePortfolioReportSiteIDs(payload.SiteIDs, portfolioSiteIDs(availableSites))
	if siteIDsErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, siteIDsErr
	}
	txErr := handlers.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		definition.Name = updatedDefinition.Name
		if saveErr := tx.Save(&definition).Error; saveErr != nil {
			return saveErr
		}
		if deleteErr := tx.Where("report_id = ?", definition.ID).Delete(&model.PortfolioTrafficReportDefinitionSite{}).Error; deleteErr != nil {
			return deleteErr
		}
		return createPortfolioReportDefinitionSites(tx, definition.ID, siteIDs)
	})
	if txErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, txErr
	}
	return PortfolioTrafficReportDefinitionResponse{
		ID:        definition.ID,
		Name:      definition.Name,
		SiteIDs:   siteIDs,
		IsDefault: false,
	}, nil
}

func createPortfolioReportDefinitionSites(database *gorm.DB, reportID string, siteIDs []string) error {
	for _, siteID := range siteIDs {
		membership, membershipErr := model.NewPortfolioTrafficReportDefinitionSite(reportID, siteID)
		if membershipErr != nil {
			return membershipErr
		}
		if createErr := database.Create(&membership).Error; createErr != nil {
			return createErr
		}
	}
	return nil
}

func normalizePortfolioReportSiteIDs(siteIDs []string, availableSiteIDs []string) ([]string, error) {
	available := make(map[string]bool, len(availableSiteIDs))
	for _, siteID := range availableSiteIDs {
		trimmedSiteID := strings.TrimSpace(siteID)
		if trimmedSiteID != "" {
			available[trimmedSiteID] = true
		}
	}
	selected := make([]string, 0, len(siteIDs))
	for _, siteID := range siteIDs {
		trimmedSiteID := strings.TrimSpace(siteID)
		if trimmedSiteID == "" || !available[trimmedSiteID] {
			return nil, fmt.Errorf("%w: unknown site_id", model.ErrInvalidPortfolioTrafficReportDefinition)
		}
		if !stringSliceContains(selected, trimmedSiteID) {
			selected = append(selected, trimmedSiteID)
		}
	}
	return selected, nil
}

func filterPortfolioReportSiteIDs(siteIDs []string, availableSiteIDs []string) []string {
	available := make(map[string]bool, len(availableSiteIDs))
	for _, siteID := range availableSiteIDs {
		trimmedSiteID := strings.TrimSpace(siteID)
		if trimmedSiteID != "" {
			available[trimmedSiteID] = true
		}
	}
	filtered := make([]string, 0, len(siteIDs))
	for _, siteID := range siteIDs {
		trimmedSiteID := strings.TrimSpace(siteID)
		if trimmedSiteID != "" && available[trimmedSiteID] && !stringSliceContains(filtered, trimmedSiteID) {
			filtered = append(filtered, trimmedSiteID)
		}
	}
	return filtered
}

func stringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (handlers *TrafficReportHandlers) resolvePortfolioReportDefinitionSites(ctx context.Context, currentUser *CurrentUser, reportID string) (PortfolioTrafficReportDefinitionResponse, []model.Site, error) {
	availableSites, sitesErr := handlers.portfolioReportSites(ctx, currentUser)
	if sitesErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, nil, sitesErr
	}
	availableSiteIDs := portfolioSiteIDs(availableSites)
	if strings.TrimSpace(reportID) == "" || reportID == model.PortfolioTrafficReportDefaultID {
		return PortfolioTrafficReportDefinitionResponse{
			ID:        model.PortfolioTrafficReportDefaultID,
			Name:      portfolioTrafficReportDefaultName,
			SiteIDs:   availableSiteIDs,
			IsDefault: true,
		}, availableSites, nil
	}
	var definition model.PortfolioTrafficReportDefinition
	if err := handlers.database.WithContext(ctx).First(&definition, "id = ? AND user_email = ?", strings.TrimSpace(reportID), currentUser.normalizedEmail()).Error; err != nil {
		return PortfolioTrafficReportDefinitionResponse{}, nil, err
	}
	siteIDs, siteErr := handlers.portfolioReportDefinitionSiteIDs(ctx, definition.ID)
	if siteErr != nil {
		return PortfolioTrafficReportDefinitionResponse{}, nil, siteErr
	}
	filteredSiteIDs := filterPortfolioReportSiteIDs(siteIDs, availableSiteIDs)
	filteredSites := filterPortfolioReportSitesByID(availableSites, filteredSiteIDs)
	return PortfolioTrafficReportDefinitionResponse{
		ID:        definition.ID,
		Name:      definition.Name,
		SiteIDs:   filteredSiteIDs,
		IsDefault: false,
	}, filteredSites, nil
}

func filterPortfolioReportSitesByID(sites []model.Site, siteIDs []string) []model.Site {
	siteIDSet := make(map[string]bool, len(siteIDs))
	for _, siteID := range siteIDs {
		siteIDSet[siteID] = true
	}
	filteredSites := make([]model.Site, 0, len(siteIDs))
	for _, site := range sites {
		if siteIDSet[site.ID] {
			filteredSites = append(filteredSites, site)
		}
	}
	return filteredSites
}

func (handlers *TrafficReportHandlers) findPortfolioSchedule(ctx context.Context, userEmail string, reportID string) (model.PortfolioTrafficReportSchedule, bool, error) {
	var schedule model.PortfolioTrafficReportSchedule
	err := handlers.database.WithContext(ctx).First(&schedule, "user_email = ? AND report_id = ?", strings.ToLower(strings.TrimSpace(userEmail)), strings.TrimSpace(reportID)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PortfolioTrafficReportSchedule{}, false, nil
	}
	if err != nil {
		return model.PortfolioTrafficReportSchedule{}, false, err
	}
	return schedule, true, nil
}

func (handlers *TrafficReportHandlers) upsertPortfolioSchedule(ctx context.Context, schedule model.PortfolioTrafficReportSchedule) (model.PortfolioTrafficReportSchedule, error) {
	var existing model.PortfolioTrafficReportSchedule
	findErr := handlers.database.WithContext(ctx).First(&existing, "user_email = ? AND report_id = ?", schedule.UserEmail, schedule.ReportID).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		if createErr := handlers.database.WithContext(ctx).Create(&schedule).Error; createErr != nil {
			return model.PortfolioTrafficReportSchedule{}, createErr
		}
		return schedule, nil
	}
	if findErr != nil {
		return model.PortfolioTrafficReportSchedule{}, findErr
	}

	existing.Enabled = schedule.Enabled
	existing.Frequency = schedule.Frequency
	existing.RecipientEmail = schedule.RecipientEmail
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
		return model.PortfolioTrafficReportSchedule{}, saveErr
	}
	return existing, nil
}

func (handlers *TrafficReportHandlers) toPortfolioScheduleResponse(schedule model.PortfolioTrafficReportSchedule, persisted bool) trafficReportScheduleResponse {
	lastStatus := strings.TrimSpace(schedule.LastStatus)
	if lastStatus == "" {
		lastStatus = model.TrafficReportStatusPending
	}
	return trafficReportScheduleResponse{
		SiteID:         "",
		ReportID:       schedule.ReportID,
		Enabled:        schedule.Enabled,
		Frequency:      schedule.Frequency,
		RecipientEmail: schedule.RecipientEmail,
		Timezone:       schedule.Timezone,
		SendHour:       schedule.SendHour,
		SendMinute:     schedule.SendMinute,
		Weekday:        schedule.Weekday,
		MonthDay:       schedule.MonthDay,
		NextSendAt:     unixSeconds(schedule.NextSendAt),
		LastSentAt:     unixSeconds(schedule.LastSentAt),
		LastStatus:     lastStatus,
		LastError:      schedule.LastError,
		EmailEnabled:   handlers.emailEnabled,
		Persisted:      persisted,
	}
}

func defaultPortfolioTrafficReportSchedule(userEmail string, reportID string) model.PortfolioTrafficReportSchedule {
	normalizedReportID := strings.TrimSpace(reportID)
	if normalizedReportID == "" {
		normalizedReportID = model.PortfolioTrafficReportDefaultID
	}
	schedule, scheduleErr := model.NewPortfolioTrafficReportSchedule(model.PortfolioTrafficReportScheduleInput{
		UserEmail:      userEmail,
		ReportID:       normalizedReportID,
		Enabled:        false,
		Frequency:      model.TrafficReportFrequencyWeekly,
		RecipientEmail: userEmail,
		Timezone:       model.DefaultTrafficReportTimezone,
		SendHour:       model.DefaultTrafficReportSendHour,
		SendMinute:     model.DefaultTrafficReportSendMinute,
		Weekday:        model.DefaultTrafficReportWeekday,
		MonthDay:       model.DefaultTrafficReportMonthDay,
	})
	if scheduleErr != nil {
		return model.PortfolioTrafficReportSchedule{
			UserEmail:      strings.ToLower(strings.TrimSpace(userEmail)),
			ReportID:       normalizedReportID,
			Enabled:        false,
			Frequency:      model.TrafficReportFrequencyWeekly,
			RecipientEmail: strings.ToLower(strings.TrimSpace(userEmail)),
			Timezone:       model.DefaultTrafficReportTimezone,
			SendHour:       model.DefaultTrafficReportSendHour,
			SendMinute:     model.DefaultTrafficReportSendMinute,
			Weekday:        model.DefaultTrafficReportWeekday,
			MonthDay:       model.DefaultTrafficReportMonthDay,
			LastStatus:     model.TrafficReportStatusPending,
		}
	}
	return schedule
}

func buildPortfolioTrafficReportScheduleFromRequest(userEmail string, reportID string, payload trafficReportScheduleRequest, referenceTime time.Time) (model.PortfolioTrafficReportSchedule, error) {
	frequency := strings.TrimSpace(payload.Frequency)
	if frequency == "" {
		frequency = model.TrafficReportFrequencyWeekly
	}
	timezoneName := strings.TrimSpace(payload.Timezone)
	if timezoneName == "" {
		timezoneName = model.DefaultTrafficReportTimezone
	}
	return model.NewPortfolioTrafficReportSchedule(model.PortfolioTrafficReportScheduleInput{
		UserEmail:      userEmail,
		ReportID:       reportID,
		Enabled:        payload.Enabled,
		Frequency:      frequency,
		RecipientEmail: userEmail,
		Timezone:       timezoneName,
		SendHour:       intValueOrDefault(payload.SendHour, model.DefaultTrafficReportSendHour),
		SendMinute:     intValueOrDefault(payload.SendMinute, model.DefaultTrafficReportSendMinute),
		Weekday:        intValueOrDefault(payload.Weekday, model.DefaultTrafficReportWeekday),
		MonthDay:       intValueOrDefault(payload.MonthDay, model.DefaultTrafficReportMonthDay),
		ReferenceTime:  referenceTime,
	})
}

func parsePortfolioTrafficReportDays(rawValue string) (int, error) {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return portfolioTrafficReportDefaultDays, nil
	}
	days, parseErr := parseVisitTrendDays(trimmedValue)
	if parseErr != nil {
		return 0, parseErr
	}
	if days > portfolioTrafficReportMaxDays {
		return 0, errors.New("portfolio traffic report days out of range")
	}
	return days, nil
}

func buildPortfolioTrafficReportData(ctx context.Context, database *gorm.DB, sites []model.Site, days int) (portfolioTrafficReportData, error) {
	normalizedDays := days
	if normalizedDays <= 0 {
		normalizedDays = portfolioTrafficReportDefaultDays
	}
	siteIDs := portfolioSiteIDs(sites)
	trend, trendErr := portfolioVisitTrend(ctx, database, siteIDs, normalizedDays)
	if trendErr != nil {
		return portfolioTrafficReportData{}, trendErr
	}
	topPages, topPagesErr := portfolioTopPages(ctx, database, siteIDs, normalizedDays, trafficReportTopPagesLimit)
	if topPagesErr != nil {
		return portfolioTrafficReportData{}, topPagesErr
	}
	siteRows, siteRowsErr := portfolioSiteRows(ctx, database, sites, normalizedDays)
	if siteRowsErr != nil {
		return portfolioTrafficReportData{}, siteRowsErr
	}
	pageViews, uniqueVisitors, totalsErr := portfolioTotals(ctx, database, siteIDs, normalizedDays)
	if totalsErr != nil {
		return portfolioTrafficReportData{}, totalsErr
	}
	return portfolioTrafficReportData{
		WindowDays:     normalizedDays,
		SiteCount:      len(sites),
		PageViews:      pageViews,
		UniqueVisitors: uniqueVisitors,
		Trend:          trend,
		TopPages:       topPages,
		Sites:          siteRows,
	}, nil
}

func portfolioSiteIDs(sites []model.Site) []string {
	siteIDs := make([]string, 0, len(sites))
	for _, site := range sites {
		siteID := strings.TrimSpace(site.ID)
		if siteID != "" {
			siteIDs = append(siteIDs, siteID)
		}
	}
	return siteIDs
}

func portfolioVisitTrend(ctx context.Context, database *gorm.DB, siteIDs []string, days int) ([]VisitTrendPoint, error) {
	startDay := visitWindowStartDay(days)
	rowsByDay := make(map[string]portfolioTrendRow)
	if len(siteIDs) > 0 {
		var rows []portfolioTrendRow
		err := database.WithContext(ctx).
			Model(&model.SiteVisit{}).
			Select("DATE(occurred_at) as day, COUNT(*) as page_views, COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN site_id || ':' || visitor_id END) as unique_visitors").
			Where("site_id IN ? AND occurred_at >= ? AND is_bot = ?", siteIDs, startDay, false).
			Group("DATE(occurred_at)").
			Order("day asc").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			dayKey, _, normalizeErr := normalizeVisitTrendMapKey(row.Day)
			if normalizeErr != nil {
				return nil, normalizeErr
			}
			rowsByDay[dayKey] = row
		}
	}

	trend := make([]VisitTrendPoint, 0, days)
	for dayIndex := 0; dayIndex < days; dayIndex++ {
		dateValue := startDay.AddDate(0, 0, dayIndex)
		dayKey := dateValue.Format(visitTrendDayLayout)
		row := rowsByDay[dayKey]
		trend = append(trend, VisitTrendPoint{
			Date:           dayKey,
			PageViews:      row.PageViews,
			UniqueVisitors: row.UniqueVisitors,
		})
	}
	return trend, nil
}

func portfolioTopPages(ctx context.Context, database *gorm.DB, siteIDs []string, days int, limit int) ([]TopPageEntry, error) {
	if len(siteIDs) == 0 {
		return nil, nil
	}
	startDay := visitWindowStartDay(days)
	var rows []TopPageStat
	err := database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select(topPagesSelectStatement).
		Where("site_id IN ? AND path <> '' AND is_bot = ? AND occurred_at >= ?", siteIDs, false, startDay).
		Group(topPagesCanonicalPathExpression).
		Order("visit_count desc, path asc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	entries := make([]TopPageEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, TopPageEntry(row))
	}
	return entries, nil
}

func portfolioTotals(ctx context.Context, database *gorm.DB, siteIDs []string, days int) (int64, int64, error) {
	if len(siteIDs) == 0 {
		return 0, 0, nil
	}
	startDay := visitWindowStartDay(days)
	var row portfolioCountRow
	err := database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("COUNT(*) as page_views, COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN site_id || ':' || visitor_id END) as unique_visitors").
		Where("site_id IN ? AND occurred_at >= ? AND is_bot = ?", siteIDs, startDay, false).
		Scan(&row).Error
	if err != nil {
		return 0, 0, err
	}
	return row.PageViews, row.UniqueVisitors, nil
}

func portfolioSiteRows(ctx context.Context, database *gorm.DB, sites []model.Site, days int) ([]PortfolioTrafficSiteRecord, error) {
	if len(sites) == 0 {
		return nil, nil
	}
	siteIDs := portfolioSiteIDs(sites)
	startDay := visitWindowStartDay(days)
	var rows []portfolioSiteCountRow
	err := database.WithContext(ctx).
		Model(&model.SiteVisit{}).
		Select("site_id, COUNT(*) as page_views, COUNT(DISTINCT CASE WHEN visitor_id <> '' THEN visitor_id END) as unique_visitors").
		Where("site_id IN ? AND occurred_at >= ? AND is_bot = ?", siteIDs, startDay, false).
		Group("site_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	countsBySiteID := make(map[string]portfolioSiteCountRow, len(rows))
	for _, row := range rows {
		countsBySiteID[row.SiteID] = row
	}

	records := make([]PortfolioTrafficSiteRecord, 0, len(sites))
	for _, site := range sites {
		counts := countsBySiteID[site.ID]
		records = append(records, PortfolioTrafficSiteRecord{
			SiteID:             site.ID,
			SiteName:           strings.TrimSpace(site.Name),
			VisitCount:         counts.PageViews,
			UniqueVisitorCount: counts.UniqueVisitors,
		})
	}
	sort.Slice(records, func(leftIndex int, rightIndex int) bool {
		left := records[leftIndex]
		right := records[rightIndex]
		if left.VisitCount == right.VisitCount {
			return strings.ToLower(left.SiteName) < strings.ToLower(right.SiteName)
		}
		return left.VisitCount > right.VisitCount
	})
	return records, nil
}

func buildPortfolioTrafficReportEmail(ctx context.Context, database *gorm.DB, schedule model.PortfolioTrafficReportSchedule) (trafficReportEmail, error) {
	reportName, sites, sitesErr := portfolioTrafficReportEmailSites(ctx, database, schedule)
	if sitesErr != nil {
		return trafficReportEmail{}, sitesErr
	}
	report, reportErr := buildPortfolioTrafficReportData(ctx, database, sites, schedule.ReportWindowDays())
	if reportErr != nil {
		return trafficReportEmail{}, reportErr
	}
	templateData := portfolioTrafficReportEmailTemplateData{
		FrequencyLabel: trafficReportFrequencyLabel(schedule.Frequency),
		ReportName:     reportName,
		WindowDays:     report.WindowDays,
		SiteCount:      report.SiteCount,
		PageViews:      report.PageViews,
		UniqueVisitors: report.UniqueVisitors,
		TopPages:       report.TopPages,
		Sites:          report.Sites,
	}

	subject, subjectErr := renderPortfolioTrafficReportEmailTemplate("subject", templateData)
	if subjectErr != nil {
		return trafficReportEmail{}, subjectErr
	}
	message, messageErr := renderPortfolioTrafficReportEmailTemplate("body", templateData)
	if messageErr != nil {
		return trafficReportEmail{}, messageErr
	}
	return trafficReportEmail{subject: subject, message: message}, nil
}

func portfolioTrafficReportEmailSites(ctx context.Context, database *gorm.DB, schedule model.PortfolioTrafficReportSchedule) (string, []model.Site, error) {
	userEmail := strings.ToLower(strings.TrimSpace(schedule.UserEmail))
	reportID := strings.TrimSpace(schedule.ReportID)
	if reportID == "" || reportID == model.PortfolioTrafficReportDefaultID {
		var sites []model.Site
		if err := database.WithContext(ctx).
			Where("(LOWER(owner_email) = ? OR LOWER(creator_email) = ?)", userEmail, userEmail).
			Order("created_at desc").
			Find(&sites).Error; err != nil {
			return "", nil, err
		}
		return portfolioTrafficReportDefaultName, sites, nil
	}

	var definition model.PortfolioTrafficReportDefinition
	if err := database.WithContext(ctx).First(&definition, "id = ? AND user_email = ?", reportID, userEmail).Error; err != nil {
		return "", nil, err
	}
	var memberships []model.PortfolioTrafficReportDefinitionSite
	if err := database.WithContext(ctx).Where("report_id = ?", reportID).Find(&memberships).Error; err != nil {
		return "", nil, err
	}
	siteIDs := make([]string, 0, len(memberships))
	for _, membership := range memberships {
		if strings.TrimSpace(membership.SiteID) != "" {
			siteIDs = append(siteIDs, membership.SiteID)
		}
	}
	if len(siteIDs) == 0 {
		return strings.TrimSpace(definition.Name), nil, nil
	}
	var sites []model.Site
	if err := database.WithContext(ctx).
		Where("id IN ? AND (LOWER(owner_email) = ? OR LOWER(creator_email) = ?)", siteIDs, userEmail, userEmail).
		Order("created_at desc").
		Find(&sites).Error; err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(definition.Name), sites, nil
}

func renderPortfolioTrafficReportEmailTemplate(templateName string, data portfolioTrafficReportEmailTemplateData) (string, error) {
	var buffer bytes.Buffer
	if templateErr := portfolioTrafficReportEmailTemplate.ExecuteTemplate(&buffer, templateName, data); templateErr != nil {
		return "", fmt.Errorf("portfolio_traffic_report_email_template %s: %w", templateName, templateErr)
	}
	return strings.TrimSpace(buffer.String()), nil
}
