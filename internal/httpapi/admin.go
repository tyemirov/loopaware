package httpapi

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	errorValueMissingFields    = "missing_fields"
	errorValueUnknownSite      = "unknown_site"
	errorValueQueryFailed      = "query_failed"
	errorValueNotAuthorized    = "not_authorized"
	errorValueInvalidOwner     = "invalid_owner"
	errorValueNothingToUpdate  = "nothing_to_update"
	errorValueInvalidOperation = "invalid_operation"
	errorValueDeleteFailed     = "delete_failed"
	errorValueSaveFailed       = "save_failed"

	widgetScriptTemplate = "<script src=\"%s/widget.js?site_id=%s\"></script>"

	dashboardNoticeCreated = "site_created"
	dashboardNoticeUpdated = "site_updated"
	dashboardNoticeDeleted = "site_deleted"
)

var (
	errSiteUnknown     = &siteError{code: errorValueUnknownSite}
	errSiteNotAllowed  = &siteError{code: errorValueNotAuthorized}
	errSiteInvalidForm = &siteError{code: errorValueMissingFields}
)

type siteError struct {
	code string
}

func (err *siteError) Error() string {
	return err.code
}

func (err *siteError) Code() string {
	return err.code
}

type createSiteRequest struct {
	Name          string
	AllowedOrigin string
	OwnerEmail    string
}

type updateSiteRequest struct {
	Name          *string
	AllowedOrigin *string
	OwnerEmail    *string
}

type siteResponse struct {
	ID            string
	Name          string
	AllowedOrigin string
	OwnerEmail    string
	Widget        string
	CreatedAt     int64
}

type feedbackMessageResponse struct {
	ID        string
	Contact   string
	Message   string
	IP        string
	UserAgent string
	CreatedAt int64
}

// SiteService coordinates data access and validation for site management.
type SiteService struct {
	database      *gorm.DB
	logger        *zap.Logger
	widgetBaseURL string
}

// NewSiteService constructs SiteService with dependencies.
func NewSiteService(database *gorm.DB, logger *zap.Logger, widgetBaseURL string) *SiteService {
	return &SiteService{
		database:      database,
		logger:        logger,
		widgetBaseURL: normalizeWidgetBaseURL(widgetBaseURL),
	}
}

func (service *SiteService) ListSitesForUser(currentUser *CurrentUser) ([]siteResponse, error) {
	var sites []model.Site

	query := service.database.Model(&model.Site{})
	if !currentUser.IsAdmin {
		query = query.Where("owner_email = ?", strings.ToLower(strings.TrimSpace(currentUser.Email)))
	}

	if err := query.Order("created_at desc").Find(&sites).Error; err != nil {
		return nil, fmt.Errorf("%s: %w", errorValueQueryFailed, err)
	}

	responses := make([]siteResponse, 0, len(sites))
	for _, site := range sites {
		responses = append(responses, service.toSiteResponse(site))
	}
	return responses, nil
}

func (service *SiteService) LoadSiteForUser(siteID string, currentUser *CurrentUser) (siteResponse, error) {
	site, err := service.loadSite(siteID)
	if err != nil {
		return siteResponse{}, err
	}
	if !canManageSite(currentUser, site) {
		return siteResponse{}, errSiteNotAllowed
	}
	return service.toSiteResponse(site), nil
}

func (service *SiteService) ListMessagesForSite(siteID string, currentUser *CurrentUser) ([]feedbackMessageResponse, error) {
	site, err := service.loadSite(siteID)
	if err != nil {
		return nil, err
	}
	if !canManageSite(currentUser, site) {
		return nil, errSiteNotAllowed
	}

	var feedbacks []model.Feedback
	if err := service.database.
		Where("site_id = ?", site.ID).
		Order("created_at desc").
		Find(&feedbacks).Error; err != nil {
		return nil, fmt.Errorf("%s: %w", errorValueQueryFailed, err)
	}

	responses := make([]feedbackMessageResponse, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		responses = append(responses, feedbackMessageResponse{
			ID:        feedback.ID,
			Contact:   feedback.Contact,
			Message:   feedback.Message,
			IP:        feedback.IP,
			UserAgent: feedback.UserAgent,
			CreatedAt: feedback.CreatedAt.Unix(),
		})
	}

	return responses, nil
}

func (service *SiteService) CreateSite(currentUser *CurrentUser, payload createSiteRequest) (siteResponse, error) {
	name := strings.TrimSpace(payload.Name)
	allowedOrigin := strings.TrimSpace(payload.AllowedOrigin)
	desiredOwner := strings.ToLower(strings.TrimSpace(payload.OwnerEmail))
	currentEmail := strings.ToLower(strings.TrimSpace(currentUser.Email))

	if !currentUser.IsAdmin {
		if desiredOwner != "" && !strings.EqualFold(desiredOwner, currentEmail) {
			return siteResponse{}, &siteError{code: errorValueInvalidOperation}
		}
		desiredOwner = currentEmail
	}

	if name == "" || allowedOrigin == "" {
		return siteResponse{}, errSiteInvalidForm
	}
	if desiredOwner == "" {
		return siteResponse{}, &siteError{code: errorValueInvalidOwner}
	}

	site := model.Site{
		ID:            storage.NewID(),
		Name:          name,
		AllowedOrigin: allowedOrigin,
		OwnerEmail:    desiredOwner,
	}

	if err := service.database.Create(&site).Error; err != nil {
		service.logger.Warn("create_site", zap.Error(err))
		return siteResponse{}, fmt.Errorf("%s: %w", errorValueSaveFailed, err)
	}

	return service.toSiteResponse(site), nil
}

func (service *SiteService) UpdateSite(currentUser *CurrentUser, siteID string, payload updateSiteRequest) (siteResponse, error) {
	if payload.Name == nil && payload.AllowedOrigin == nil && payload.OwnerEmail == nil {
		return siteResponse{}, &siteError{code: errorValueNothingToUpdate}
	}

	site, err := service.loadSite(siteID)
	if err != nil {
		return siteResponse{}, err
	}
	if !canManageSite(currentUser, site) {
		return siteResponse{}, errSiteNotAllowed
	}

	if payload.Name != nil {
		trimmed := strings.TrimSpace(*payload.Name)
		if trimmed == "" {
			return siteResponse{}, errSiteInvalidForm
		}
		site.Name = trimmed
	}

	if payload.AllowedOrigin != nil {
		trimmed := strings.TrimSpace(*payload.AllowedOrigin)
		if trimmed == "" {
			return siteResponse{}, errSiteInvalidForm
		}
		site.AllowedOrigin = trimmed
	}

	if payload.OwnerEmail != nil {
		if !currentUser.IsAdmin {
			return siteResponse{}, &siteError{code: errorValueInvalidOperation}
		}
		trimmed := strings.ToLower(strings.TrimSpace(*payload.OwnerEmail))
		if trimmed == "" {
			return siteResponse{}, &siteError{code: errorValueInvalidOwner}
		}
		site.OwnerEmail = trimmed
	}

	if err := service.database.Save(&site).Error; err != nil {
		service.logger.Warn("update_site", zap.Error(err))
		return siteResponse{}, fmt.Errorf("%s: %w", errorValueSaveFailed, err)
	}

	return service.toSiteResponse(site), nil
}

func (service *SiteService) DeleteSite(currentUser *CurrentUser, siteID string) error {
	site, err := service.loadSite(siteID)
	if err != nil {
		return err
	}
	if !canManageSite(currentUser, site) {
		return errSiteNotAllowed
	}

	deleteErr := service.database.Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Where("site_id = ?", site.ID).Delete(&model.Feedback{}).Error; err != nil {
			return err
		}
		if err := transaction.Delete(&model.Site{ID: site.ID}).Error; err != nil {
			return err
		}
		return nil
	})
	if deleteErr != nil {
		service.logger.Warn("delete_site", zap.Error(deleteErr))
		return fmt.Errorf("%s: %w", errorValueDeleteFailed, deleteErr)
	}

	return nil
}

func (service *SiteService) loadSite(siteID string) (model.Site, error) {
	trimmed := strings.TrimSpace(siteID)
	if trimmed == "" {
		return model.Site{}, errSiteUnknown
	}

	var site model.Site
	if err := service.database.First(&site, "id = ?", trimmed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Site{}, errSiteUnknown
		}
		return model.Site{}, fmt.Errorf("%s: %w", errorValueQueryFailed, err)
	}

	return site, nil
}

func (service *SiteService) toSiteResponse(site model.Site) siteResponse {
	widgetBase := service.widgetBaseURL
	if widgetBase == "" {
		widgetBase = normalizeWidgetBaseURL(site.AllowedOrigin)
	}

	return siteResponse{
		ID:            site.ID,
		Name:          site.Name,
		AllowedOrigin: site.AllowedOrigin,
		OwnerEmail:    site.OwnerEmail,
		Widget:        fmt.Sprintf(widgetScriptTemplate, widgetBase, site.ID),
		CreatedAt:     site.CreatedAt.UTC().Unix(),
	}
}

func normalizeWidgetBaseURL(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimRight(trimmed, "/")
}

func canManageSite(currentUser *CurrentUser, site model.Site) bool {
	if currentUser.IsAdmin {
		return true
	}
	return strings.EqualFold(site.OwnerEmail, strings.TrimSpace(currentUser.Email))
}
