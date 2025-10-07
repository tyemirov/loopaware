package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	dashboardTemplateName               = "dashboard"
	dashboardHTMLContentType            = "text/html; charset=utf-8"
	dashboardPageTitle                  = "LoopAware Dashboard"
	dashboardStatusLoadingUser          = "Loading account information..."
	dashboardStatusLoadingSites         = "Loading sites..."
	dashboardStatusLoadFailed           = "Failed to load data."
	dashboardStatusSavingSite           = "Saving site..."
	dashboardStatusSiteSaved            = "Site updated."
	dashboardStatusCreatingSite         = "Creating site..."
	dashboardStatusSiteCreated          = "Site created."
	dashboardStatusSelectSite           = "Select a site to see details."
	dashboardStatusNoMessages           = "No feedback yet."
	dashboardStatusNoSites              = "No sites available yet."
	dashboardRoleAdminLabel             = "Administrator"
	dashboardRoleUserLabel              = "User"
	dashboardFeedbackPlaceholder        = "Select a site to load feedback."
	dashboardWidgetCardTitle            = "Site widget"
	dashboardWidgetInstructions         = "Embed this <script> tag on pages served from the allowed origin."
	dashboardWidgetUnavailable          = "Save the site to generate a widget snippet."
	dashboardStatusWidgetCopied         = "Widget snippet copied."
	dashboardStatusWidgetCopyFailed     = "Unable to copy widget snippet."
	dashboardFooterBrandPrefix          = "Built by"
	dashboardFooterBrandName            = "Marco Polo Research Lab"
	dashboardFooterBrandURL             = "https://mprlab.com"
	navbarSettingsButtonLabel           = "Account settings"
	navbarLogoutLabel                   = "Logout"
	navbarThemeToggleLabel              = "Dark mode"
	newSiteOptionValue                  = "__new__"
	newSiteOptionLabel                  = "New site"
	siteFormCreateButtonLabel           = "Create site"
	siteFormUpdateButtonLabel           = "Update site"
	dashboardActionButtonPrimaryClass   = "btn btn-outline-primary btn-sm"
	dashboardActionButtonSuccessClass   = "btn btn-outline-success btn-sm"
	dashboardActionButtonSecondaryClass = "btn btn-outline-secondary btn-sm"
	dashboardActionButtonDangerClass    = "btn btn-outline-danger btn-sm"
	siteFormCreateButtonClass           = dashboardActionButtonPrimaryClass
	siteFormUpdateButtonClass           = dashboardActionButtonSuccessClass
	userNameElementID                   = "user-name"
	userEmailElementID                  = "user-email"
	userRoleBadgeElementID              = "user-role"
	userAvatarElementID                 = "user-avatar"
	sitesListElementID                  = "sites-list"
	emptySitesMessageElementID          = "empty-sites-message"
	siteFormElementID                   = "site-form"
	editSiteNameInputElementID          = "edit-site-name"
	editSiteOriginInputElementID        = "edit-site-origin"
	editSiteOwnerContainerElementID     = "edit-site-owner-container"
	editSiteOwnerInputElementID         = "edit-site-owner"
	saveSiteButtonElementID             = "save-site-button"
	refreshMessagesButtonElementID      = "refresh-messages-button"
	feedbackTableBodyElementID          = "feedback-table-body"
	logoutButtonElementID               = "logout-button"
	widgetSnippetTextareaElementID      = "widget-snippet"
	copyWidgetSnippetButtonElementID    = "copy-widget-snippet"
	settingsButtonElementID             = "settings-button"
	settingsMenuElementID               = "settings-menu"
	settingsThemeToggleElementID        = "settings-theme-toggle"
	settingsAvatarImageElementID        = "settings-avatar-image"
	settingsAvatarFallbackElementID     = "settings-avatar-fallback"
	themeStorageKey                     = "loopaware_theme"
	formStatusElementID                 = "site-status"
	widgetStatusElementID               = "widget-status"
	messagesStatusElementID             = "messages-status"
	newSiteButtonElementID              = "new-site-button"
	newSiteButtonClass                  = dashboardActionButtonPrimaryClass
	newSiteButtonActiveClass            = "btn btn-primary btn-sm"
	siteListItemClass                   = "list-group-item list-group-item-action"
	siteListItemActiveClass             = "active"
	dashboardStatusDeletingSite         = "Deleting site..."
	dashboardStatusSiteDeleted          = "Site deleted."
	dashboardStatusDeleteFailed         = "Failed to delete site."
	deleteSiteButtonElementID           = "delete-site-button"
	deleteSiteButtonClass               = "btn btn-danger btn-sm"
	deleteSiteButtonDisabledClass       = "btn btn-danger btn-sm disabled"
	deleteSiteModalElementID            = "delete-site-modal"
	deleteSiteModalTitle                = "Delete site"
	deleteSiteModalDescription          = "This action permanently removes the site and its feedback."
	deleteSiteModalInputElementID       = "delete-site-confirm-name"
	deleteSiteModalInputLabel           = "Type the site name to confirm"
	deleteSiteModalInputPlaceholder     = "Enter the site name"
	deleteSiteModalConfirmButtonID      = "delete-site-confirm-button"
	deleteSiteModalConfirmButtonLabel   = "Delete site"
	deleteSiteModalConfirmButtonClass   = "btn btn-danger"
	deleteSiteModalCancelButtonLabel    = "Cancel"
	deleteSiteModalCancelButtonClass    = "btn btn-secondary"
	deleteSiteTargetNameElementID       = "delete-site-target-name"
	deleteSiteModalHintPrefix           = "Type "
	deleteSiteModalHintSuffix           = " exactly to confirm."
	formStatusBaseClass                 = "d-none py-1 px-2 small rounded"
	formStatusSuccessClass              = "py-1 px-2 small rounded bg-white border border-success text-success"
	formStatusDangerClass               = "py-1 px-2 small rounded bg-white border border-danger text-danger"
)

type dashboardTemplateData struct {
	PageTitle                         string
	APIMeEndpoint                     string
	APISitesEndpoint                  string
	APISiteUpdateEndpointPrefix       string
	APIMessagesEndpointPrefix         string
	APIMessagesEndpointSuffix         string
	LogoutPath                        string
	LoginPath                         string
	StatusLoadingUser                 string
	StatusLoadingSites                string
	StatusLoadFailed                  string
	StatusSavingSite                  string
	StatusSiteSaved                   string
	StatusCreatingSite                string
	StatusSiteCreated                 string
	StatusDeletingSite                string
	StatusSiteDeleted                 string
	StatusDeleteSiteFailed            string
	StatusSelectSite                  string
	StatusNoMessages                  string
	StatusNoSites                     string
	RoleAdmin                         string
	RoleUser                          string
	EmptySitesMessage                 string
	FeedbackPlaceholder               string
	FooterBrandPrefix                 string
	FooterBrandName                   string
	FooterBrandURL                    string
	UserNameID                        string
	UserEmailID                       string
	UserRoleBadgeID                   string
	UserAvatarID                      string
	SitesListID                       string
	EmptySitesMessageID               string
	SiteFormID                        string
	EditSiteNameInputID               string
	EditSiteOriginInputID             string
	EditSiteOwnerContainerID          string
	EditSiteOwnerInputID              string
	SaveSiteButtonID                  string
	SaveButtonSaving                  string
	SaveButtonSaved                   string
	SaveButtonCreated                 string
	SaveButtonFailed                  string
	SaveButtonDefaultClass            string
	RefreshMessagesButtonID           string
	RefreshButtonLoading              string
	RefreshButtonSuccess              string
	RefreshButtonFailed               string
	RefreshButtonDefaultLabel         string
	RefreshButtonDefaultClass         string
	ActionButtonPrimaryClass          string
	ActionButtonSuccessClass          string
	ActionButtonSecondaryClass        string
	ActionButtonDangerClass           string
	FeedbackTableBodyID               string
	LogoutButtonID                    string
	NewSiteOptionValue                string
	CreateButtonLabel                 string
	UpdateButtonLabel                 string
	CreateButtonClass                 string
	UpdateButtonClass                 string
	NewSiteButtonID                   string
	NewSiteButtonLabel                string
	NewSiteButtonClass                string
	NewSiteButtonActiveClass          string
	DeleteSiteButtonID                string
	DeleteSiteButtonLabel             string
	DeleteSiteButtonClass             string
	DeleteSiteButtonDisabledClass     string
	SiteListItemClass                 string
	SiteListItemActiveClass           string
	WidgetCardTitle                   string
	WidgetInstructions                string
	WidgetUnavailableMessage          string
	StatusWidgetCopied                string
	StatusWidgetCopyFailed            string
	WidgetSnippetTextareaID           string
	CopyWidgetSnippetButtonID         string
	CopyButtonCopied                  string
	CopyButtonFailed                  string
	CopyButtonDefaultLabel            string
	CopyButtonDefaultClass            string
	SettingsButtonID                  string
	SettingsButtonLabel               string
	LogoutLabel                       string
	ThemeToggleLabel                  string
	SettingsMenuID                    string
	SettingsThemeToggleID             string
	ThemeStorageKey                   string
	SettingsAvatarImageID             string
	SettingsAvatarFallbackID          string
	FormStatusID                      string
	FormStatusBaseClass               string
	FormStatusSuccessClass            string
	FormStatusDangerClass             string
	WidgetStatusID                    string
	MessagesStatusID                  string
	DeleteSiteModalID                 string
	DeleteSiteModalTitle              string
	DeleteSiteModalDescription        string
	DeleteSiteModalInputID            string
	DeleteSiteModalInputLabel         string
	DeleteSiteModalInputPlaceholder   string
	DeleteSiteModalConfirmButtonID    string
	DeleteSiteModalConfirmButtonLabel string
	DeleteSiteModalConfirmButtonClass string
	DeleteSiteModalCancelButtonLabel  string
	DeleteSiteModalCancelButtonClass  string
	DeleteSiteTargetNameID            string
	DeleteSiteModalHintPrefix         string
	DeleteSiteModalHintSuffix         string
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
		PageTitle:                         dashboardPageTitle,
		APIMeEndpoint:                     "/api/me",
		APISitesEndpoint:                  "/api/sites",
		APISiteUpdateEndpointPrefix:       "/api/sites/",
		APIMessagesEndpointPrefix:         "/api/sites/",
		APIMessagesEndpointSuffix:         "/messages",
		LogoutPath:                        constants.LogoutPath,
		LoginPath:                         constants.LoginPath,
		StatusLoadingUser:                 dashboardStatusLoadingUser,
		StatusLoadingSites:                dashboardStatusLoadingSites,
		StatusLoadFailed:                  dashboardStatusLoadFailed,
		StatusSavingSite:                  dashboardStatusSavingSite,
		StatusSiteSaved:                   dashboardStatusSiteSaved,
		StatusCreatingSite:                dashboardStatusCreatingSite,
		StatusSiteCreated:                 dashboardStatusSiteCreated,
		StatusDeletingSite:                dashboardStatusDeletingSite,
		StatusSiteDeleted:                 dashboardStatusSiteDeleted,
		StatusDeleteSiteFailed:            dashboardStatusDeleteFailed,
		StatusSelectSite:                  dashboardStatusSelectSite,
		StatusNoMessages:                  dashboardStatusNoMessages,
		StatusNoSites:                     dashboardStatusNoSites,
		RoleAdmin:                         dashboardRoleAdminLabel,
		RoleUser:                          dashboardRoleUserLabel,
		EmptySitesMessage:                 dashboardStatusNoSites,
		FeedbackPlaceholder:               dashboardFeedbackPlaceholder,
		FooterBrandPrefix:                 dashboardFooterBrandPrefix,
		FooterBrandName:                   dashboardFooterBrandName,
		FooterBrandURL:                    dashboardFooterBrandURL,
		UserNameID:                        userNameElementID,
		UserEmailID:                       userEmailElementID,
		UserRoleBadgeID:                   userRoleBadgeElementID,
		UserAvatarID:                      userAvatarElementID,
		SitesListID:                       sitesListElementID,
		EmptySitesMessageID:               emptySitesMessageElementID,
		SiteFormID:                        siteFormElementID,
		EditSiteNameInputID:               editSiteNameInputElementID,
		EditSiteOriginInputID:             editSiteOriginInputElementID,
		EditSiteOwnerContainerID:          editSiteOwnerContainerElementID,
		EditSiteOwnerInputID:              editSiteOwnerInputElementID,
		SaveSiteButtonID:                  saveSiteButtonElementID,
		SaveButtonSaving:                  "Saving site...",
		SaveButtonSaved:                   "Site updated.",
		SaveButtonCreated:                 "Site created.",
		SaveButtonFailed:                  "Failed to save site.",
		SaveButtonDefaultClass:            dashboardActionButtonSuccessClass,
		RefreshMessagesButtonID:           refreshMessagesButtonElementID,
		RefreshButtonLoading:              "Refreshing...",
		RefreshButtonSuccess:              "Feedback refreshed.",
		RefreshButtonFailed:               "Refresh failed.",
		RefreshButtonDefaultLabel:         "Refresh feedback",
		RefreshButtonDefaultClass:         dashboardActionButtonSecondaryClass,
		ActionButtonPrimaryClass:          dashboardActionButtonPrimaryClass,
		ActionButtonSuccessClass:          dashboardActionButtonSuccessClass,
		ActionButtonSecondaryClass:        dashboardActionButtonSecondaryClass,
		ActionButtonDangerClass:           dashboardActionButtonDangerClass,
		FeedbackTableBodyID:               feedbackTableBodyElementID,
		LogoutButtonID:                    logoutButtonElementID,
		NewSiteOptionValue:                newSiteOptionValue,
		CreateButtonLabel:                 siteFormCreateButtonLabel,
		UpdateButtonLabel:                 siteFormUpdateButtonLabel,
		CreateButtonClass:                 siteFormCreateButtonClass,
		UpdateButtonClass:                 siteFormUpdateButtonClass,
		NewSiteButtonID:                   newSiteButtonElementID,
		NewSiteButtonLabel:                newSiteOptionLabel,
		NewSiteButtonClass:                newSiteButtonClass,
		NewSiteButtonActiveClass:          newSiteButtonActiveClass,
		DeleteSiteButtonID:                deleteSiteButtonElementID,
		DeleteSiteButtonLabel:             deleteSiteModalConfirmButtonLabel,
		DeleteSiteButtonClass:             deleteSiteButtonClass,
		DeleteSiteButtonDisabledClass:     deleteSiteButtonDisabledClass,
		SiteListItemClass:                 siteListItemClass,
		SiteListItemActiveClass:           siteListItemActiveClass,
		WidgetCardTitle:                   dashboardWidgetCardTitle,
		WidgetInstructions:                dashboardWidgetInstructions,
		WidgetUnavailableMessage:          dashboardWidgetUnavailable,
		StatusWidgetCopied:                dashboardStatusWidgetCopied,
		StatusWidgetCopyFailed:            dashboardStatusWidgetCopyFailed,
		WidgetSnippetTextareaID:           widgetSnippetTextareaElementID,
		CopyWidgetSnippetButtonID:         copyWidgetSnippetButtonElementID,
		CopyButtonCopied:                  "Snippet copied.",
		CopyButtonFailed:                  "Copy failed.",
		CopyButtonDefaultLabel:            "Copy snippet",
		CopyButtonDefaultClass:            dashboardActionButtonPrimaryClass,
		SettingsButtonID:                  settingsButtonElementID,
		SettingsButtonLabel:               navbarSettingsButtonLabel,
		LogoutLabel:                       navbarLogoutLabel,
		ThemeToggleLabel:                  navbarThemeToggleLabel,
		SettingsMenuID:                    settingsMenuElementID,
		SettingsThemeToggleID:             settingsThemeToggleElementID,
		ThemeStorageKey:                   themeStorageKey,
		SettingsAvatarImageID:             settingsAvatarImageElementID,
		SettingsAvatarFallbackID:          settingsAvatarFallbackElementID,
		FormStatusID:                      formStatusElementID,
		FormStatusBaseClass:               formStatusBaseClass,
		FormStatusSuccessClass:            formStatusSuccessClass,
		FormStatusDangerClass:             formStatusDangerClass,
		WidgetStatusID:                    widgetStatusElementID,
		MessagesStatusID:                  messagesStatusElementID,
		DeleteSiteModalID:                 deleteSiteModalElementID,
		DeleteSiteModalTitle:              deleteSiteModalTitle,
		DeleteSiteModalDescription:        deleteSiteModalDescription,
		DeleteSiteModalInputID:            deleteSiteModalInputElementID,
		DeleteSiteModalInputLabel:         deleteSiteModalInputLabel,
		DeleteSiteModalInputPlaceholder:   deleteSiteModalInputPlaceholder,
		DeleteSiteModalConfirmButtonID:    deleteSiteModalConfirmButtonID,
		DeleteSiteModalConfirmButtonLabel: deleteSiteModalConfirmButtonLabel,
		DeleteSiteModalConfirmButtonClass: deleteSiteModalConfirmButtonClass,
		DeleteSiteModalCancelButtonLabel:  deleteSiteModalCancelButtonLabel,
		DeleteSiteModalCancelButtonClass:  deleteSiteModalCancelButtonClass,
		DeleteSiteTargetNameID:            deleteSiteTargetNameElementID,
		DeleteSiteModalHintPrefix:         deleteSiteModalHintPrefix,
		DeleteSiteModalHintSuffix:         deleteSiteModalHintSuffix,
	}

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_dashboard", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{jsonKeyError: "render_failed"})
		return
	}

	context.Data(http.StatusOK, dashboardHTMLContentType, buffer.Bytes())
}
