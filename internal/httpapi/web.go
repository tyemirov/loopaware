package httpapi

import (
	"bytes"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
)

const (
	dashboardTemplateName         = "dashboard"
	dashboardHTMLContentType      = "text/html; charset=utf-8"
	dashboardPageTitle            = "LoopAware Dashboard"
	dashboardTimestampLayout      = "02 Jan 2006 15:04 MST"
	queryParamSiteID              = "site_id"
	queryParamNotice              = "notice"
	queryParamRefresh             = "refresh"
	queryValueNewSite             = "new"
	formFieldName                 = "name"
	formFieldAllowedOrigin        = "allowed_origin"
	formFieldOwnerEmail           = "owner_email"
	dashboardNoticeMessageCreated = "Site created."
	dashboardNoticeMessageUpdated = "Site updated."
	dashboardNoticeMessageDeleted = "Site deleted."
	dashboardErrorGeneral         = "Something went wrong. Please try again."
	dashboardErrorMissingFields   = "All fields are required."
	dashboardErrorInvalidOwner    = "Owner email must be provided."
	dashboardErrorUnauthorized    = "You are not allowed to perform that action."
	dashboardErrorUnknownSite     = "Site not found."
	dashboardErrorNothingToUpdate = "No changes to apply."
	siteFormCreateLabel           = "Create site"
	siteFormUpdateLabel           = "Update site"
	siteFormCreateClass           = "btn btn-outline-primary btn-sm"
	siteFormUpdateClass           = "btn btn-outline-success btn-sm"
	widgetUnavailableMessage      = "Save the site to generate a widget snippet."
	dashboardThemeStorageKey      = "loopaware_theme"
	dashboardThemeToggleID        = "dashboard-theme-toggle"
	dashboardBrandName            = "Marco Polo Research Lab"
	dashboardBrandURL             = "https://mprlab.com"
	dashboardFaviconDataURI       = `data:image/svg+xml;utf8,
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">
    <rect fill="%230A2540" x="0" y="0" width="256" height="256" rx="28" ry="28"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linejoin="round"
      d="M 32 128 C 72 56, 184 56, 224 128 C 184 200, 72 200, 32 128 Z"/>
    <circle cx="128" cy="128" r="48" stroke="%23D4AF37" fill="none" stroke-width="8"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linecap="round"
      d="M 54 82 L 32 32"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linecap="round"
      d="M 202 82 L 224 32"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linecap="round"
      d="M 54 174 L 32 224"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linecap="round"
      d="M 202 174 L 224 224"/>
  </svg>`
)

// DashboardRoute exposes the authenticated dashboard path.
const DashboardRoute = "/app"

var noticeMessages = map[string]string{
	dashboardNoticeCreated: dashboardNoticeMessageCreated,
	dashboardNoticeUpdated: dashboardNoticeMessageUpdated,
	dashboardNoticeDeleted: dashboardNoticeMessageDeleted,
}

type dashboardPageData struct {
	PageTitle                  string
	CurrentUser                dashboardUserView
	DeleteError                string
	Sites                      []dashboardSiteView
	SelectedSite               *dashboardSiteView
	SiteForm                   dashboardFormState
	SiteStatus                 dashboardStatus
	SiteStatusDefault          string
	SiteStatusDefaultClass     string
	WidgetSnippet              string
	WidgetStatus               dashboardStatus
	WidgetStatusDefault        string
	WidgetStatusDefaultClass   string
	WidgetCopyEnabled          bool
	FeedbackMessages           []dashboardMessageView
	FeedbackStatus             dashboardStatus
	FeedbackStatusDefault      string
	FeedbackStatusDefaultClass string
	FeedbackRefresh            string
	DeleteActionURL            string
	DeleteAllowed              bool
	IsCreateMode               bool
	LogoutPath                 string
	BrandName                  string
	BrandURL                   string
	FaviconDataURL             htmltemplate.URL
}

type dashboardUserView struct {
	Name           string
	Email          string
	PictureURL     string
	IsAdmin        bool
	AvatarInitial  string
	ThemeToggleID  string
	ThemeStorageID string
}

type dashboardStatus struct {
	Message   string
	Class     string
	Ephemeral bool
}

type dashboardSiteView struct {
	ID            string
	Name          string
	AllowedOrigin string
	OwnerEmail    string
	Widget        string
	CreatedAt     string
	IsActive      bool
}

type dashboardMessageView struct {
	Contact   string
	Message   string
	CreatedAt string
}

type dashboardFormState struct {
	ActionURL     string
	Name          string
	AllowedOrigin string
	OwnerEmail    string
	SubmitLabel   string
	SubmitClass   string
}

// DashboardWebHandlers renders and processes the authenticated dashboard.
type DashboardWebHandlers struct {
	logger      *zap.Logger
	template    *htmltemplate.Template
	siteService *SiteService
}

// NewDashboardWebHandlers compiles the dashboard template with dependencies.
func NewDashboardWebHandlers(logger *zap.Logger, siteService *SiteService) *DashboardWebHandlers {
	compiledTemplate := htmltemplate.Must(htmltemplate.New(dashboardTemplateName).Parse(dashboardTemplateHTML))
	return &DashboardWebHandlers{
		logger:      logger,
		template:    compiledTemplate,
		siteService: siteService,
	}
}

func (handlers *DashboardWebHandlers) RenderDashboard(context *gin.Context) {
	currentUser, ok := handlers.currentUser(context)
	if !ok {
		context.Redirect(http.StatusFound, "/login")
		context.Writer.WriteHeaderNow()
		return
	}

	selectedSiteID := strings.TrimSpace(context.Query(queryParamSiteID))
	data, err := handlers.buildDashboardData(currentUser, selectedSiteID)
	if err != nil {
		handlers.logger.Error("render_dashboard", zap.Error(err))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if noticeKey := strings.TrimSpace(context.Query(queryParamNotice)); noticeKey != "" {
		if message, ok := noticeMessages[noticeKey]; ok {
			data.SiteStatus = successStatus(message)
		}
	}

	if selectedSiteID != "" && data.SelectedSite == nil && !data.IsCreateMode {
		data.SiteStatus = errorStatus(dashboardErrorUnknownSite)
	}

	if data.DeleteError != "" {
		data.SiteStatus = errorStatus(data.DeleteError)
	}

	handlers.renderDashboard(context, http.StatusOK, data)
}

func (handlers *DashboardWebHandlers) CreateSite(context *gin.Context) {
	currentUser, ok := handlers.currentUser(context)
	if !ok {
		context.Redirect(http.StatusFound, "/login")
		context.Writer.WriteHeaderNow()
		return
	}

	request := createSiteRequest{
		Name:          context.PostForm(formFieldName),
		AllowedOrigin: context.PostForm(formFieldAllowedOrigin),
		OwnerEmail:    context.PostForm(formFieldOwnerEmail),
	}

	response, err := handlers.siteService.CreateSite(currentUser, request)
	if err != nil {
		handlers.renderCreateError(context, currentUser, request, err)
		return
	}

	redirectURL := fmt.Sprintf("%s?%s=%s&%s=%s", DashboardRoute, queryParamSiteID, response.ID, queryParamNotice, dashboardNoticeCreated)
	context.Redirect(http.StatusSeeOther, redirectURL)
	context.Writer.WriteHeaderNow()
}

func (handlers *DashboardWebHandlers) UpdateSite(context *gin.Context) {
	currentUser, ok := handlers.currentUser(context)
	if !ok {
		context.Redirect(http.StatusFound, "/login")
		context.Writer.WriteHeaderNow()
		return
	}

	siteID := strings.TrimSpace(context.Param("id"))
	name := context.PostForm(formFieldName)
	allowedOrigin := context.PostForm(formFieldAllowedOrigin)
	ownerEmail := context.PostForm(formFieldOwnerEmail)

	payload := updateSiteRequest{}
	if name != "" {
		payload.Name = &name
	}
	if allowedOrigin != "" {
		payload.AllowedOrigin = &allowedOrigin
	}
	if ownerEmail != "" {
		payload.OwnerEmail = &ownerEmail
	}

	response, err := handlers.siteService.UpdateSite(currentUser, siteID, payload)
	if err != nil {
		handlers.renderUpdateError(context, currentUser, siteID, payload, err)
		return
	}

	redirectURL := fmt.Sprintf("%s?%s=%s&%s=%s", DashboardRoute, queryParamSiteID, response.ID, queryParamNotice, dashboardNoticeUpdated)
	context.Redirect(http.StatusSeeOther, redirectURL)
	context.Writer.WriteHeaderNow()
}

func (handlers *DashboardWebHandlers) DeleteSite(context *gin.Context) {
	currentUser, ok := handlers.currentUser(context)
	if !ok {
		context.Redirect(http.StatusFound, "/login")
		context.Writer.WriteHeaderNow()
		return
	}

	siteID := strings.TrimSpace(context.Param("id"))
	if err := handlers.siteService.DeleteSite(currentUser, siteID); err != nil {
		handlers.renderDeleteError(context, currentUser, siteID, err)
		return
	}

	context.Redirect(http.StatusSeeOther, fmt.Sprintf("%s?%s=%s", DashboardRoute, queryParamNotice, dashboardNoticeDeleted))
	context.Writer.WriteHeaderNow()
}

func (handlers *DashboardWebHandlers) UserAvatar(context *gin.Context) {
	currentUser, ok := handlers.currentUser(context)
	if !ok {
		context.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	trimmedEmail := strings.ToLower(strings.TrimSpace(currentUser.Email))
	if trimmedEmail == "" {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}

	var user model.User
	if err := handlers.siteService.database.First(&user, "email = ?", trimmedEmail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			context.AbortWithStatus(http.StatusNotFound)
			return
		}
		handlers.logger.Warn("load_user_avatar", zap.Error(err))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(user.AvatarData) == 0 {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}

	contentType := user.AvatarContentType
	if contentType == "" {
		contentType = defaultAvatarMimeType
	}

	context.Header("Cache-Control", "no-cache")
	context.Data(http.StatusOK, contentType, user.AvatarData)
}

func (handlers *DashboardWebHandlers) currentUser(context *gin.Context) (*CurrentUser, bool) {
	currentUser, ok := CurrentUserFromContext(context)
	return currentUser, ok
}

func (handlers *DashboardWebHandlers) buildDashboardData(currentUser *CurrentUser, requestedSiteID string) (dashboardPageData, error) {
	data := dashboardPageData{
		PageTitle: dashboardPageTitle,
		CurrentUser: dashboardUserView{
			Name:           currentUser.Name,
			Email:          currentUser.Email,
			PictureURL:     currentUser.PictureURL,
			IsAdmin:        currentUser.IsAdmin,
			AvatarInitial:  computeAvatarInitial(currentUser),
			ThemeToggleID:  dashboardThemeToggleID,
			ThemeStorageID: dashboardThemeStorageKey,
		},
		LogoutPath:     constants.LogoutPath,
		BrandName:      dashboardBrandName,
		BrandURL:       dashboardBrandURL,
		FaviconDataURL: htmltemplate.URL(dashboardFaviconDataURI),
	}

	siteResponses, err := handlers.siteService.ListSitesForUser(currentUser)
	if err != nil {
		return data, err
	}

	data.Sites = make([]dashboardSiteView, 0, len(siteResponses))
	for _, site := range siteResponses {
		data.Sites = append(data.Sites, toDashboardSiteView(site))
	}

	selectedSiteID := strings.TrimSpace(requestedSiteID)
	if selectedSiteID == "" && len(data.Sites) > 0 {
		selectedSiteID = data.Sites[0].ID
	}

	isCreateMode := selectedSiteID == "" || selectedSiteID == queryValueNewSite
	var selectedSite *dashboardSiteView
	if !isCreateMode {
		for index := range data.Sites {
			if data.Sites[index].ID == selectedSiteID {
				data.Sites[index].IsActive = true
				selectedSite = &data.Sites[index]
				break
			}
		}
		if selectedSite == nil && len(data.Sites) > 0 {
			data.Sites[0].IsActive = true
			selectedSite = &data.Sites[0]
		}
		if selectedSite == nil {
			isCreateMode = true
		}
	}

	data.IsCreateMode = isCreateMode
	data.SiteForm = handlers.buildSiteForm(currentUser, selectedSite, isCreateMode)
	baseSiteStatus := defaultSiteStatus(isCreateMode)
	data.SiteStatus = baseSiteStatus
	data.SiteStatusDefault = baseSiteStatus.Message
	data.SiteStatusDefaultClass = baseSiteStatus.Class

	baseWidgetStatus := defaultWidgetStatus(isCreateMode)
	data.WidgetStatus = baseWidgetStatus
	data.WidgetStatusDefault = baseWidgetStatus.Message
	data.WidgetStatusDefaultClass = baseWidgetStatus.Class

	baseFeedbackStatus := defaultFeedbackStatus()
	data.FeedbackStatus = baseFeedbackStatus
	data.FeedbackStatusDefault = baseFeedbackStatus.Message
	data.FeedbackStatusDefaultClass = baseFeedbackStatus.Class

	if isCreateMode || selectedSite == nil {
		data.WidgetSnippet = widgetUnavailableMessage
		data.WidgetCopyEnabled = false
		data.DeleteAllowed = false
		data.FeedbackMessages = nil
		return data, nil
	}

	data.SelectedSite = selectedSite
	data.WidgetSnippet = selectedSite.Widget
	data.WidgetCopyEnabled = true
	data.DeleteAllowed = true
	data.DeleteActionURL = fmt.Sprintf("/app/sites/%s/delete", selectedSite.ID)
	data.FeedbackRefresh = buildRefreshURL(selectedSite.ID)
	messages, err := handlers.siteService.ListMessagesForSite(selectedSite.ID, currentUser)
	if err != nil {
		return data, err
	}
	data.FeedbackMessages = make([]dashboardMessageView, 0, len(messages))
	for _, message := range messages {
		data.FeedbackMessages = append(data.FeedbackMessages, toDashboardMessageView(message))
	}
	return data, nil
}

func (handlers *DashboardWebHandlers) buildSiteForm(currentUser *CurrentUser, selectedSite *dashboardSiteView, isCreateMode bool) dashboardFormState {
	if isCreateMode || selectedSite == nil {
		defaultOwner := strings.TrimSpace(currentUser.Email)
		if currentUser.IsAdmin {
			defaultOwner = ""
		}
		return dashboardFormState{
			ActionURL:     "/app/sites",
			Name:          "",
			AllowedOrigin: "",
			OwnerEmail:    defaultOwner,
			SubmitLabel:   siteFormCreateLabel,
			SubmitClass:   siteFormCreateClass,
		}
	}

	return dashboardFormState{
		ActionURL:     fmt.Sprintf("/app/sites/%s", selectedSite.ID),
		Name:          selectedSite.Name,
		AllowedOrigin: selectedSite.AllowedOrigin,
		OwnerEmail:    selectedSite.OwnerEmail,
		SubmitLabel:   siteFormUpdateLabel,
		SubmitClass:   siteFormUpdateClass,
	}
}

func (handlers *DashboardWebHandlers) renderDashboard(context *gin.Context, status int, data dashboardPageData) {
	var buffer bytes.Buffer
	if err := handlers.template.ExecuteTemplate(&buffer, dashboardTemplateName, data); err != nil {
		handlers.logger.Error("execute_dashboard_template", zap.Error(err))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	context.Data(status, dashboardHTMLContentType, buffer.Bytes())
}

func (handlers *DashboardWebHandlers) renderCreateError(context *gin.Context, currentUser *CurrentUser, request createSiteRequest, err error) {
	data, buildErr := handlers.buildDashboardData(currentUser, queryValueNewSite)
	if buildErr != nil {
		handlers.logger.Error("create_render_dashboard", zap.Error(buildErr))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	data.SiteForm.Name = request.Name
	data.SiteForm.AllowedOrigin = request.AllowedOrigin
	data.SiteForm.OwnerEmail = request.OwnerEmail
	data.SiteStatus = errorStatus(siteErrorMessage(err))
	data.WidgetStatus = infoStatus(widgetUnavailableMessage)
	data.FeedbackStatus = infoStatus("No feedback yet.")

	handlers.renderDashboard(context, http.StatusBadRequest, data)
}

func (handlers *DashboardWebHandlers) renderUpdateError(context *gin.Context, currentUser *CurrentUser, siteID string, payload updateSiteRequest, err error) {
	data, buildErr := handlers.buildDashboardData(currentUser, siteID)
	if buildErr != nil {
		handlers.logger.Error("update_render_dashboard", zap.Error(buildErr))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	data.SiteStatus = errorStatus(siteErrorMessage(err))
	if payload.Name != nil {
		data.SiteForm.Name = *payload.Name
	}
	if payload.AllowedOrigin != nil {
		data.SiteForm.AllowedOrigin = *payload.AllowedOrigin
	}
	if payload.OwnerEmail != nil {
		data.SiteForm.OwnerEmail = *payload.OwnerEmail
	}

	handlers.renderDashboard(context, http.StatusBadRequest, data)
}

func (handlers *DashboardWebHandlers) renderDeleteError(context *gin.Context, currentUser *CurrentUser, siteID string, err error) {
	data, buildErr := handlers.buildDashboardData(currentUser, siteID)
	if buildErr != nil {
		handlers.logger.Error("delete_render_dashboard", zap.Error(buildErr))
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	data.SiteStatus = errorStatus(siteErrorMessage(err))

	handlers.renderDashboard(context, http.StatusBadRequest, data)
}

func toDashboardSiteView(site siteResponse) dashboardSiteView {
	return dashboardSiteView{
		ID:            site.ID,
		Name:          site.Name,
		AllowedOrigin: site.AllowedOrigin,
		OwnerEmail:    site.OwnerEmail,
		Widget:        site.Widget,
		CreatedAt:     formatTimestamp(site.CreatedAt),
	}
}

func toDashboardMessageView(message feedbackMessageResponse) dashboardMessageView {
	return dashboardMessageView{
		Contact:   message.Contact,
		Message:   message.Message,
		CreatedAt: formatTimestamp(message.CreatedAt),
	}
}

func formatTimestamp(unixSeconds int64) string {
	if unixSeconds == 0 {
		return ""
	}
	return time.Unix(unixSeconds, 0).UTC().Format(dashboardTimestampLayout)
}

func siteErrorMessage(err error) string {
	var domainErr *siteError
	if errors.As(err, &domainErr) {
		switch domainErr.Code() {
		case errorValueMissingFields:
			return dashboardErrorMissingFields
		case errorValueInvalidOwner:
			return dashboardErrorInvalidOwner
		case errorValueNotAuthorized, errorValueInvalidOperation:
			return dashboardErrorUnauthorized
		case errorValueUnknownSite:
			return dashboardErrorUnknownSite
		case errorValueNothingToUpdate:
			return dashboardErrorNothingToUpdate
		default:
			return dashboardErrorGeneral
		}
	}
	return dashboardErrorGeneral
}

func computeAvatarInitial(currentUser *CurrentUser) string {
	name := strings.TrimSpace(currentUser.Name)
	if name == "" {
		name = strings.TrimSpace(currentUser.Email)
	}
	if name == "" {
		return ""
	}
	runes := []rune(name)
	return strings.ToUpper(string(runes[0]))
}

func buildRefreshURL(siteID string) string {
	if siteID == "" {
		return DashboardRoute
	}
	values := url.Values{}
	values.Set(queryParamSiteID, siteID)
	values.Set(queryParamRefresh, fmt.Sprintf("%d", time.Now().Unix()))
	return fmt.Sprintf("%s?%s", DashboardRoute, values.Encode())
}

func defaultSiteStatus(createMode bool) dashboardStatus {
	return dashboardStatus{Message: "", Class: "", Ephemeral: false}
}

func defaultWidgetStatus(createMode bool) dashboardStatus {
	return dashboardStatus{Message: "", Class: "", Ephemeral: false}
}

func successStatus(message string) dashboardStatus {
	return dashboardStatus{Message: message, Class: "text-success", Ephemeral: true}
}

func errorStatus(message string) dashboardStatus {
	return dashboardStatus{Message: message, Class: "text-danger", Ephemeral: true}
}

func infoStatus(message string) dashboardStatus {
	return dashboardStatus{Message: message, Class: "text-muted", Ephemeral: false}
}

func defaultFeedbackStatus() dashboardStatus {
	return dashboardStatus{Message: "", Class: "", Ephemeral: false}
}
