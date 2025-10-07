package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	dashboardTemplateName            = "dashboard"
	dashboardHTMLContentType         = "text/html; charset=utf-8"
	dashboardPageTitle               = "LoopAware Dashboard"
	dashboardStatusLoadingUser       = "Loading account information..."
	dashboardStatusLoadingSites      = "Loading sites..."
	dashboardStatusLoadFailed        = "Failed to load data."
	dashboardStatusSavingSite        = "Saving site..."
	dashboardStatusSiteSaved         = "Site updated."
	dashboardStatusCreatingSite      = "Creating site..."
	dashboardStatusSiteCreated       = "Site created."
	dashboardStatusSelectSite        = "Select a site to see details."
	dashboardStatusNoMessages        = "No feedback yet."
	dashboardStatusNoSites           = "No sites available yet."
	dashboardRoleAdminLabel          = "Administrator"
	dashboardRoleUserLabel           = "User"
	dashboardFeedbackPlaceholder     = "Select a site to load feedback."
	dashboardWidgetCardTitle         = "Site widget"
	dashboardWidgetInstructions      = "Embed this <script> tag on pages served from the allowed origin."
	dashboardWidgetUnavailable       = "Save the site to generate a widget snippet."
	dashboardStatusWidgetCopied      = "Widget snippet copied."
	dashboardStatusWidgetCopyFailed  = "Unable to copy widget snippet."
	dashboardFooterBrandPrefix       = "Built by"
	dashboardFooterBrandName         = "Marco Polo Research Lab"
	dashboardFooterBrandURL          = "https://mprlab.com"
	navbarSettingsButtonLabel        = "Account settings"
	navbarLogoutLabel                = "Logout"
	navbarThemeToggleLabel           = "Dark mode"
	newSiteOptionValue               = "__new__"
	newSiteOptionLabel               = "New site"
	siteFormCreateButtonLabel        = "Create site"
	siteFormUpdateButtonLabel        = "Update site"
	siteFormCreateButtonClass        = "btn btn-primary"
	siteFormUpdateButtonClass        = "btn btn-success"
	userNameElementID                = "user-name"
	userEmailElementID               = "user-email"
	userRoleBadgeElementID           = "user-role"
	userAvatarElementID              = "user-avatar"
	sitesListElementID               = "sites-list"
	emptySitesMessageElementID       = "empty-sites-message"
	siteFormElementID                = "site-form"
	editSiteNameInputElementID       = "edit-site-name"
	editSiteOriginInputElementID     = "edit-site-origin"
	editSiteOwnerContainerElementID  = "edit-site-owner-container"
	editSiteOwnerInputElementID      = "edit-site-owner"
	saveSiteButtonElementID          = "save-site-button"
	refreshMessagesButtonElementID   = "refresh-messages-button"
	feedbackTableBodyElementID       = "feedback-table-body"
	logoutButtonElementID            = "logout-button"
	widgetSnippetTextareaElementID   = "widget-snippet"
	copyWidgetSnippetButtonElementID = "copy-widget-snippet"
	settingsButtonElementID          = "settings-button"
	settingsMenuElementID            = "settings-menu"
	settingsThemeToggleElementID     = "settings-theme-toggle"
	settingsAvatarImageElementID     = "settings-avatar-image"
	settingsAvatarFallbackElementID  = "settings-avatar-fallback"
	themeStorageKey                  = "loopaware_theme"
	formStatusElementID              = "site-status"
	widgetStatusElementID            = "widget-status"
	messagesStatusElementID          = "messages-status"
	newSiteButtonElementID           = "new-site-button"
	newSiteButtonClass               = "btn btn-outline-primary btn-sm"
	newSiteButtonActiveClass         = "btn btn-primary btn-sm"
	siteListItemClass                = "list-group-item list-group-item-action"
	siteListItemActiveClass          = "active"
)

type dashboardTemplateData struct {
	PageTitle                   string
	APIMeEndpoint               string
	APISitesEndpoint            string
	APISiteUpdateEndpointPrefix string
	APIMessagesEndpointPrefix   string
	APIMessagesEndpointSuffix   string
	LogoutPath                  string
	LoginPath                   string
	StatusLoadingUser           string
	StatusLoadingSites          string
	StatusLoadFailed            string
	StatusSavingSite            string
	StatusSiteSaved             string
	StatusCreatingSite          string
	StatusSiteCreated           string
	StatusSelectSite            string
	StatusNoMessages            string
	StatusNoSites               string
	RoleAdmin                   string
	RoleUser                    string
	EmptySitesMessage           string
	FeedbackPlaceholder         string
	CurrentYear                 int
	FooterBrandPrefix           string
	FooterBrandName             string
	FooterBrandURL              string
	UserNameID                  string
	UserEmailID                 string
	UserRoleBadgeID             string
	UserAvatarID                string
	SitesListID                 string
	EmptySitesMessageID         string
	SiteFormID                  string
	EditSiteNameInputID         string
	EditSiteOriginInputID       string
	EditSiteOwnerContainerID    string
	EditSiteOwnerInputID        string
	SaveSiteButtonID            string
	SaveButtonSaving            string
	SaveButtonSaved             string
	SaveButtonCreated           string
	SaveButtonFailed            string
	SaveButtonDefaultClass      string
	RefreshMessagesButtonID     string
	RefreshButtonLoading        string
	RefreshButtonSuccess        string
	RefreshButtonFailed         string
	RefreshButtonDefaultLabel   string
	RefreshButtonDefaultClass   string
	FeedbackTableBodyID         string
	LogoutButtonID              string
	NewSiteOptionValue          string
	CreateButtonLabel           string
	UpdateButtonLabel           string
	CreateButtonClass           string
	UpdateButtonClass           string
	NewSiteButtonID             string
	NewSiteButtonLabel          string
	NewSiteButtonClass          string
	NewSiteButtonActiveClass    string
	SiteListItemClass           string
	SiteListItemActiveClass     string
	WidgetCardTitle             string
	WidgetInstructions          string
	WidgetUnavailableMessage    string
	StatusWidgetCopied          string
	StatusWidgetCopyFailed      string
	WidgetSnippetTextareaID     string
	CopyWidgetSnippetButtonID   string
	CopyButtonCopied            string
	CopyButtonFailed            string
	CopyButtonDefaultLabel      string
	CopyButtonDefaultClass      string
	SettingsButtonID            string
	SettingsButtonLabel         string
	LogoutLabel                 string
	ThemeToggleLabel            string
	SettingsMenuID              string
	SettingsThemeToggleID       string
	ThemeStorageKey             string
	SettingsAvatarImageID       string
	SettingsAvatarFallbackID    string
	FormStatusID                string
	WidgetStatusID              string
	MessagesStatusID            string
}

// DashboardWebHandlers serves the authenticated dashboard UI.
type DashboardWebHandlers struct {
	logger   *zap.Logger
	template *template.Template
}

func NewDashboardWebHandlers(logger *zap.Logger) *DashboardWebHandlers {
	compiledTemplate := template.Must(template.New(dashboardTemplateName).Parse(dashboardTemplateHTML))
	return &DashboardWebHandlers{
		logger:   logger,
		template: compiledTemplate,
	}
}

func (handlers *DashboardWebHandlers) RenderDashboard(context *gin.Context) {
	data := dashboardTemplateData{
		PageTitle:                   dashboardPageTitle,
		APIMeEndpoint:               "/api/me",
		APISitesEndpoint:            "/api/sites",
		APISiteUpdateEndpointPrefix: "/api/sites/",
		APIMessagesEndpointPrefix:   "/api/sites/",
		APIMessagesEndpointSuffix:   "/messages",
		LogoutPath:                  constants.LogoutPath,
		LoginPath:                   constants.LoginPath,
		StatusLoadingUser:           dashboardStatusLoadingUser,
		StatusLoadingSites:          dashboardStatusLoadingSites,
		StatusLoadFailed:            dashboardStatusLoadFailed,
		StatusSavingSite:            dashboardStatusSavingSite,
		StatusSiteSaved:             dashboardStatusSiteSaved,
		StatusCreatingSite:          dashboardStatusCreatingSite,
		StatusSiteCreated:           dashboardStatusSiteCreated,
		StatusSelectSite:            dashboardStatusSelectSite,
		StatusNoMessages:            dashboardStatusNoMessages,
		StatusNoSites:               dashboardStatusNoSites,
		RoleAdmin:                   dashboardRoleAdminLabel,
		RoleUser:                    dashboardRoleUserLabel,
		EmptySitesMessage:           dashboardStatusNoSites,
		FeedbackPlaceholder:         dashboardFeedbackPlaceholder,
		CurrentYear:                 time.Now().Year(),
		FooterBrandPrefix:           dashboardFooterBrandPrefix,
		FooterBrandName:             dashboardFooterBrandName,
		FooterBrandURL:              dashboardFooterBrandURL,
		UserNameID:                  userNameElementID,
		UserEmailID:                 userEmailElementID,
		UserRoleBadgeID:             userRoleBadgeElementID,
		UserAvatarID:                userAvatarElementID,
		SitesListID:                 sitesListElementID,
		EmptySitesMessageID:         emptySitesMessageElementID,
		SiteFormID:                  siteFormElementID,
		EditSiteNameInputID:         editSiteNameInputElementID,
		EditSiteOriginInputID:       editSiteOriginInputElementID,
		EditSiteOwnerContainerID:    editSiteOwnerContainerElementID,
		EditSiteOwnerInputID:        editSiteOwnerInputElementID,
		SaveSiteButtonID:            saveSiteButtonElementID,
		SaveButtonSaving:            "Saving site...",
		SaveButtonSaved:             "Site updated.",
		SaveButtonCreated:           "Site created.",
		SaveButtonFailed:            "Failed to save site.",
		SaveButtonDefaultClass:      "btn btn-outline-success",
		RefreshMessagesButtonID:     refreshMessagesButtonElementID,
		RefreshButtonLoading:        "Refreshing...",
		RefreshButtonSuccess:        "Feedback refreshed.",
		RefreshButtonFailed:         "Refresh failed.",
		RefreshButtonDefaultLabel:   "Refresh feedback",
		RefreshButtonDefaultClass:   "btn btn-outline-secondary btn-sm",
		FeedbackTableBodyID:         feedbackTableBodyElementID,
		LogoutButtonID:              logoutButtonElementID,
		NewSiteOptionValue:          newSiteOptionValue,
		CreateButtonLabel:           siteFormCreateButtonLabel,
		UpdateButtonLabel:           siteFormUpdateButtonLabel,
		CreateButtonClass:           siteFormCreateButtonClass,
		UpdateButtonClass:           siteFormUpdateButtonClass,
		NewSiteButtonID:             newSiteButtonElementID,
		NewSiteButtonLabel:          newSiteOptionLabel,
		NewSiteButtonClass:          newSiteButtonClass,
		NewSiteButtonActiveClass:    newSiteButtonActiveClass,
		SiteListItemClass:           siteListItemClass,
		SiteListItemActiveClass:     siteListItemActiveClass,
		WidgetCardTitle:             dashboardWidgetCardTitle,
		WidgetInstructions:          dashboardWidgetInstructions,
		WidgetUnavailableMessage:    dashboardWidgetUnavailable,
		StatusWidgetCopied:          dashboardStatusWidgetCopied,
		StatusWidgetCopyFailed:      dashboardStatusWidgetCopyFailed,
		WidgetSnippetTextareaID:     widgetSnippetTextareaElementID,
		CopyWidgetSnippetButtonID:   copyWidgetSnippetButtonElementID,
		CopyButtonCopied:            "Snippet copied.",
		CopyButtonFailed:            "Copy failed.",
		CopyButtonDefaultLabel:      "Copy snippet",
		CopyButtonDefaultClass:      "btn btn-outline-primary btn-sm",
		SettingsButtonID:            settingsButtonElementID,
		SettingsButtonLabel:         navbarSettingsButtonLabel,
		LogoutLabel:                 navbarLogoutLabel,
		ThemeToggleLabel:            navbarThemeToggleLabel,
		SettingsMenuID:              settingsMenuElementID,
		SettingsThemeToggleID:       settingsThemeToggleElementID,
		ThemeStorageKey:             themeStorageKey,
		SettingsAvatarImageID:       settingsAvatarImageElementID,
		SettingsAvatarFallbackID:    settingsAvatarFallbackElementID,
		FormStatusID:                formStatusElementID,
		WidgetStatusID:              widgetStatusElementID,
		MessagesStatusID:            messagesStatusElementID,
	}

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_dashboard", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{jsonKeyError: "render_failed"})
		return
	}

	context.Data(http.StatusOK, dashboardHTMLContentType, buffer.Bytes())
}
