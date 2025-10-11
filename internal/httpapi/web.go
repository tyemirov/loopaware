package httpapi

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	dashboardTemplateName                = "dashboard"
	dashboardHTMLContentType             = "text/html; charset=utf-8"
	dashboardPageTitle                   = "LoopAware Dashboard"
	dashboardStatusLoadingUser           = "Loading account information..."
	dashboardStatusLoadingSites          = "Loading sites..."
	dashboardStatusLoadFailed            = "Failed to load data."
	dashboardStatusSavingSite            = "Saving site..."
	dashboardStatusSiteSaved             = "Site updated."
	dashboardStatusCreatingSite          = "Creating site..."
	dashboardStatusSiteCreated           = "Site created."
	dashboardStatusSelectSite            = "Select a site to see details."
	dashboardStatusNoMessages            = "No feedback yet."
	dashboardStatusNoSites               = "No sites available yet."
	dashboardRoleAdminLabel              = "Administrator"
	dashboardRoleUserLabel               = "User"
	dashboardFeedbackPlaceholder         = "Select a site to load feedback."
	dashboardWidgetCardTitle             = "Site widget"
	dashboardWidgetInstructions          = "Embed this <script> tag on pages served from the allowed origin."
	dashboardWidgetUnavailable           = "Save the site to generate a widget snippet."
	dashboardStatusWidgetCopied          = "Widget snippet copied."
	dashboardStatusWidgetCopyFailed      = "Unable to copy widget snippet."
	dashboardFooterBrandPrefix           = "Built by"
	dashboardFooterBrandName             = "Marco Polo Research Lab"
	dashboardFooterBrandURL              = "https://mprlab.com"
	dashboardFooterToggleButtonID        = "dashboard-footer-toggle"
	dashboardHeaderLogoElementID         = "dashboard-header-logo"
	navbarSettingsButtonLabel            = "Account settings"
	navbarLogoutLabel                    = "Logout"
	navbarThemeToggleLabel               = "Dark mode"
	newSiteOptionValue                   = "__new__"
	newSiteOptionLabel                   = "New site"
	siteFormCreateButtonLabel            = "Create site"
	siteFormUpdateButtonLabel            = "Update site"
	dashboardActionButtonPrimaryClass    = "btn btn-outline-primary btn-sm"
	dashboardActionButtonSuccessClass    = "btn btn-outline-success btn-sm"
	dashboardActionButtonSecondaryClass  = "btn btn-outline-secondary btn-sm"
	dashboardActionButtonDangerClass     = "btn btn-outline-danger btn-sm"
	siteFormCreateButtonClass            = dashboardActionButtonPrimaryClass
	siteFormUpdateButtonClass            = dashboardActionButtonSuccessClass
	userNameElementID                    = "user-name"
	userEmailElementID                   = "user-email"
	userRoleBadgeElementID               = "user-role"
	userAvatarElementID                  = "user-avatar"
	sitesListElementID                   = "sites-list"
	emptySitesMessageElementID           = "empty-sites-message"
	siteFormElementID                    = "site-form"
	editSiteNameInputElementID           = "edit-site-name"
	editSiteOriginInputElementID         = "edit-site-origin"
	editSiteOwnerContainerElementID      = "edit-site-owner-container"
	editSiteOwnerInputElementID          = "edit-site-owner"
	saveSiteButtonElementID              = "save-site-button"
	refreshMessagesButtonElementID       = "refresh-messages-button"
	feedbackTableHeaderElementID         = "feedback-table-header"
	feedbackTableHeaderLightClass        = "table-light"
	feedbackTableHeaderDarkClass         = "table-dark"
	feedbackTableBodyElementID           = "feedback-table-body"
	logoutButtonElementID                = "logout-button"
	widgetSnippetTextareaElementID       = "widget-snippet"
	copyWidgetSnippetButtonElementID     = "copy-widget-snippet"
	settingsButtonElementID              = "settings-button"
	settingsMenuElementID                = "settings-menu"
	settingsThemeToggleElementID         = "settings-theme-toggle"
	settingsAvatarImageElementID         = "settings-avatar-image"
	settingsAvatarFallbackElementID      = "settings-avatar-fallback"
	themeStorageKey                      = "loopaware_dashboard_theme"
	formStatusElementID                  = "site-status"
	widgetStatusElementID                = "widget-status"
	messagesStatusElementID              = "messages-status"
	newSiteButtonElementID               = "new-site-button"
	newSiteButtonClass                   = dashboardActionButtonPrimaryClass
	newSiteButtonActiveClass             = "btn btn-primary btn-sm"
	siteListItemClass                    = "list-group-item list-group-item-action"
	siteListItemActiveClass              = "active"
	clientConfigElementID                = "dashboard-config"
	dashboardStatusDeletingSite          = "Deleting site..."
	dashboardStatusSiteDeleted           = "Site deleted."
	dashboardStatusDeleteFailed          = "Failed to delete site."
	dashboardBootstrapIconsIntegrityAttr = "integrity=\"sha384-XGjxtQfXaH2tnPFa9x+ruJTuLE3Aa6LhHSWRr1XeTyhezb4abCG4ccI5AkVDxqC+\""
	dashboardFaviconDataURI              = `data:image/svg+xml;utf8,
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">
    <rect fill="%230A2540" x="0" y="0" width="256" height="256" rx="28" ry="28"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="10" stroke-linejoin="round"
      d="M 32 128 C 72 56, 184 56, 224 128 C 184 200, 72 200, 32 128 Z"/>
    <circle cx="128" cy="128" r="48" stroke="%23D4AF37" fill="none" stroke-width="8"/>
    <path stroke="%23D4AF37" fill="none" stroke-width="8" stroke-linecap="round" stroke-linejoin="round"
      d="M 96 128 C 96 108, 128 108, 128 128
         C 128 148, 160 148, 160 128
         C 160 108, 128 108, 128 128
         C 128 148, 96 148, 96 128"/>
  </svg>`
	deleteSiteButtonElementID           = "delete-site-button"
	deleteSiteButtonClass               = "btn btn-sm border-0 bg-transparent text-danger opacity-100"
	deleteSiteButtonDisabledClass       = "btn btn-sm border-0 bg-transparent text-danger opacity-100 disabled"
	deleteSiteIconClass                 = "bi bi-trash3-fill text-danger"
	footerElementID                     = "dashboard-footer"
	footerInnerElementID                = "dashboard-footer-inner"
	footerBaseClass                     = "mt-auto py-3 fixed-bottom border-top"
	footerThemeLightClass               = "bg-body text-body-secondary"
	footerThemeDarkClass                = "bg-dark text-light border-light"
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
	formStatusSuccessClass              = "py-1 px-2 small rounded border border-success-subtle text-success-emphasis bg-success-subtle"
	formStatusDangerClass               = "py-1 px-2 small rounded border border-danger-subtle text-danger-emphasis bg-danger-subtle"
	fieldHelpButtonClass                = "btn btn-link p-0 text-secondary"
	fieldHelpIconClass                  = "bi bi-question-circle-fill"
	fieldHelpTextClass                  = "form-text text-muted"
	siteListItemHeaderClass             = "d-flex align-items-center gap-2"
	siteListItemFaviconClass            = "flex-shrink-0 rounded border bg-white"
	siteCreatedAtElementID              = "site-created-at"
	siteCreatedAtContainerElementID     = "site-created-at-container"
	siteCreatedAtPlaceholder            = "Not saved yet."
	feedbackCountElementID              = "feedback-count"
	siteNameHelpButtonElementID         = "site-name-help-button"
	siteNameHelpTitle                   = "Site name"
	siteNameHelpContent                 = "Displayed in the sites list for your team."
	allowedOriginHelpButtonElementID    = "allowed-origin-help-button"
	allowedOriginHelpTitle              = "Allowed origin"
	allowedOriginHelpContent            = "Must match the full protocol, host, and optional port where the widget will run."
	ownerEmailHelpButtonElementID       = "owner-email-help-button"
	ownerEmailHelpTitle                 = "Owner email"
	ownerEmailHelpContent               = "Receives notifications when visitors submit feedback."
	siteSearchToggleButtonElementID     = "site-search-toggle-button"
	siteSearchContainerElementID        = "site-search-container"
	siteSearchInputElementID            = "site-search-input"
	siteSearchToggleLabel               = "Toggle site search"
	siteSearchPlaceholder               = "Search sites"
	messagesSearchToggleButtonElementID = "messages-search-toggle-button"
	messagesSearchContainerElementID    = "messages-search-container"
	messagesSearchInputElementID        = "messages-search-input"
	messagesSearchToggleLabel           = "Toggle feedback search"
	messagesSearchPlaceholder           = "Search feedback"
	searchToggleButtonClass             = "btn btn-link p-0 text-secondary"
	searchInputClass                    = "form-control form-control-sm"
	dashboardStatusNoSiteMatches        = "No sites match your search."
	dashboardStatusNoMessageMatches     = "No feedback matches your search."
	validationMessageNameRequiredKey    = "name_required"
	validationMessageOriginKey          = "origin_invalid"
	validationMessageOwnerKey           = "owner_invalid"
	dashboardValidationNameMessage      = "Site name is required."
	dashboardValidationOriginMessage    = "Allowed origin must include protocol and hostname, for example https://example.com."
	dashboardValidationOwnerMessage     = "Provide a valid owner email address."
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
	BootstrapIconsIntegrityAttr       template.HTMLAttr
	FaviconDataURI                    template.URL
	HeaderLogoDataURI                 template.URL
	HeaderLogoImageID                 string
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
	FooterHTML                        template.HTML
	FooterElementID                   string
	FooterInnerElementID              string
	FooterBaseClass                   string
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
	SiteCreatedAtElementID            string
	SiteCreatedAtContainerID          string
	SiteCreatedAtPlaceholder          string
	SiteSearchToggleButtonID          string
	SiteSearchToggleLabel             string
	SiteSearchContainerID             string
	SiteSearchInputID                 string
	SiteSearchPlaceholder             string
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
	FeedbackTableHeaderID             string
	FeedbackTableHeaderLightClass     string
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
	DeleteSiteIconClass               string
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
	SearchToggleButtonClass           string
	SearchInputClass                  string
	FieldHelpButtonClass              string
	FieldHelpIconClass                string
	FieldHelpTextClass                string
	SiteNameHelpButtonID              string
	SiteNameHelpTitle                 string
	SiteNameHelpContent               string
	AllowedOriginHelpButtonID         string
	AllowedOriginHelpTitle            string
	AllowedOriginHelpContent          string
	OwnerEmailHelpButtonID            string
	OwnerEmailHelpTitle               string
	OwnerEmailHelpContent             string
	MessagesSearchToggleButtonID      string
	MessagesSearchToggleLabel         string
	MessagesSearchContainerID         string
	MessagesSearchInputID             string
	MessagesSearchPlaceholder         string
	FeedbackCountElementID            string
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
	ClientConfigElementID             string
	ClientConfigJSON                  template.JS
}

type dashboardClientConfig struct {
	APIPaths           map[string]string `json:"api_paths"`
	Paths              map[string]string `json:"paths"`
	ElementIDs         map[string]string `json:"element_ids"`
	ButtonClasses      map[string]string `json:"button_classes"`
	ButtonLabels       map[string]string `json:"button_labels"`
	StatusMessages     map[string]string `json:"status_messages"`
	RoleLabels         map[string]string `json:"role_labels"`
	ButtonStyles       map[string]string `json:"button_styles"`
	ComponentClasses   map[string]string `json:"component_classes"`
	WidgetTexts        map[string]string `json:"widget_texts"`
	ThemeStorageKey    string            `json:"theme_storage_key"`
	OptionValues       map[string]string `json:"option_values"`
	FormStatusClasses  map[string]string `json:"form_status_classes"`
	FooterThemeClasses map[string]string `json:"footer_theme_classes"`
	TableThemeClasses  map[string]string `json:"table_theme_classes"`
	ValidationMessages map[string]string `json:"validation_messages"`
}

// DashboardWebHandlers serves the authenticated dashboard UI.
type DashboardWebHandlers struct {
	logger      *zap.Logger
	template    *template.Template
	landingPath string
}

func NewDashboardWebHandlers(logger *zap.Logger, landingPath string) *DashboardWebHandlers {
	compiledTemplate := template.Must(template.New(dashboardTemplateName).Parse(dashboardTemplateHTML))
	normalizedLandingPath := landingPath
	if normalizedLandingPath == "" {
		normalizedLandingPath = "/"
	}
	return &DashboardWebHandlers{
		logger:      logger,
		template:    compiledTemplate,
		landingPath: normalizedLandingPath,
	}
}

func (handlers *DashboardWebHandlers) RenderDashboard(context *gin.Context) {
	footerHTML, footerErr := RenderFooterHTML(FooterConfig{
		ElementID:         footerElementID,
		InnerElementID:    footerInnerElementID,
		BaseClass:         footerBaseClass,
		InnerClass:        "container d-flex justify-content-end text-end small",
		WrapperClass:      "dropup d-inline-flex align-items-center gap-2 text-body-secondary",
		PrefixClass:       "text-body-secondary",
		PrefixText:        dashboardFooterBrandPrefix,
		ToggleButtonID:    dashboardFooterToggleButtonID,
		ToggleButtonClass: "btn btn-link dropdown-toggle text-decoration-none px-0 fw-semibold",
		ToggleLabel:       dashboardFooterBrandName,
		MenuClass:         "dropdown-menu dropdown-menu-end shadow",
		MenuItemClass:     "dropdown-item",
	})
	if footerErr != nil {
		handlers.logger.Warn("render_dashboard_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	data := dashboardTemplateData{
		PageTitle:                         dashboardPageTitle,
		APIMeEndpoint:                     "/api/me",
		APISitesEndpoint:                  "/api/sites",
		APISiteUpdateEndpointPrefix:       "/api/sites/",
		APIMessagesEndpointPrefix:         "/api/sites/",
		APIMessagesEndpointSuffix:         "/messages",
		LogoutPath:                        constants.LogoutPath,
		LoginPath:                         constants.LoginPath,
		BootstrapIconsIntegrityAttr:       template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:                    template.URL(dashboardFaviconDataURI),
		HeaderLogoDataURI:                 landingLogoDataURI,
		HeaderLogoImageID:                 dashboardHeaderLogoElementID,
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
		FooterHTML:                        footerHTML,
		FooterElementID:                   footerElementID,
		FooterInnerElementID:              footerInnerElementID,
		FooterBaseClass:                   footerBaseClass,
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
		SiteCreatedAtElementID:            siteCreatedAtElementID,
		SiteCreatedAtContainerID:          siteCreatedAtContainerElementID,
		SiteCreatedAtPlaceholder:          siteCreatedAtPlaceholder,
		SiteSearchToggleButtonID:          siteSearchToggleButtonElementID,
		SiteSearchToggleLabel:             siteSearchToggleLabel,
		SiteSearchContainerID:             siteSearchContainerElementID,
		SiteSearchInputID:                 siteSearchInputElementID,
		SiteSearchPlaceholder:             siteSearchPlaceholder,
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
		FeedbackTableHeaderID:             feedbackTableHeaderElementID,
		FeedbackTableHeaderLightClass:     feedbackTableHeaderLightClass,
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
		DeleteSiteIconClass:               deleteSiteIconClass,
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
		SearchToggleButtonClass:           searchToggleButtonClass,
		SearchInputClass:                  searchInputClass,
		FieldHelpButtonClass:              fieldHelpButtonClass,
		FieldHelpIconClass:                fieldHelpIconClass,
		FieldHelpTextClass:                fieldHelpTextClass,
		SiteNameHelpButtonID:              siteNameHelpButtonElementID,
		SiteNameHelpTitle:                 siteNameHelpTitle,
		SiteNameHelpContent:               siteNameHelpContent,
		AllowedOriginHelpButtonID:         allowedOriginHelpButtonElementID,
		AllowedOriginHelpTitle:            allowedOriginHelpTitle,
		AllowedOriginHelpContent:          allowedOriginHelpContent,
		OwnerEmailHelpButtonID:            ownerEmailHelpButtonElementID,
		OwnerEmailHelpTitle:               ownerEmailHelpTitle,
		OwnerEmailHelpContent:             ownerEmailHelpContent,
		MessagesSearchToggleButtonID:      messagesSearchToggleButtonElementID,
		MessagesSearchToggleLabel:         messagesSearchToggleLabel,
		MessagesSearchContainerID:         messagesSearchContainerElementID,
		MessagesSearchInputID:             messagesSearchInputElementID,
		MessagesSearchPlaceholder:         messagesSearchPlaceholder,
		FeedbackCountElementID:            feedbackCountElementID,
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

	data.ClientConfigElementID = clientConfigElementID

	clientConfig := dashboardClientConfig{
		APIPaths: map[string]string{
			"me":                   "/api/me",
			"sites":                "/api/sites",
			"site_update_prefix":   "/api/sites/",
			"site_messages_prefix": "/api/sites/",
			"site_messages_suffix": "/messages",
		},
		Paths: map[string]string{
			"logout":  constants.LogoutPath,
			"login":   constants.LoginPath,
			"landing": handlers.landingPath,
		},
		ElementIDs: map[string]string{
			"user_name":                     userNameElementID,
			"user_email":                    userEmailElementID,
			"user_avatar":                   userAvatarElementID,
			"user_role":                     userRoleBadgeElementID,
			"sites_list":                    sitesListElementID,
			"empty_sites_message":           emptySitesMessageElementID,
			"site_form":                     siteFormElementID,
			"edit_site_name":                editSiteNameInputElementID,
			"edit_site_origin":              editSiteOriginInputElementID,
			"edit_site_owner_container":     editSiteOwnerContainerElementID,
			"edit_site_owner":               editSiteOwnerInputElementID,
			"site_created_at":               siteCreatedAtElementID,
			"site_created_at_container":     siteCreatedAtContainerElementID,
			"save_site_button":              saveSiteButtonElementID,
			"refresh_messages_button":       refreshMessagesButtonElementID,
			"feedback_table_header":         feedbackTableHeaderElementID,
			"feedback_table_body":           feedbackTableBodyElementID,
			"logout_button":                 logoutButtonElementID,
			"widget_snippet_textarea":       widgetSnippetTextareaElementID,
			"copy_widget_snippet_button":    copyWidgetSnippetButtonElementID,
			"settings_button":               settingsButtonElementID,
			"settings_menu":                 settingsMenuElementID,
			"settings_theme_toggle":         settingsThemeToggleElementID,
			"settings_avatar_image":         settingsAvatarImageElementID,
			"settings_avatar_fallback":      settingsAvatarFallbackElementID,
			"form_status":                   formStatusElementID,
			"new_site_button":               newSiteButtonElementID,
			"delete_site_button":            deleteSiteButtonElementID,
			"delete_site_modal":             deleteSiteModalElementID,
			"delete_site_confirm_button":    deleteSiteModalConfirmButtonID,
			"delete_site_confirm_input":     deleteSiteModalInputElementID,
			"delete_site_target_name":       deleteSiteTargetNameElementID,
			"footer":                        footerElementID,
			"footer_inner":                  footerInnerElementID,
			"site_name_help_button":         siteNameHelpButtonElementID,
			"allowed_origin_help_button":    allowedOriginHelpButtonElementID,
			"owner_email_help_button":       ownerEmailHelpButtonElementID,
			"site_search_toggle_button":     siteSearchToggleButtonElementID,
			"site_search_container":         siteSearchContainerElementID,
			"site_search_input":             siteSearchInputElementID,
			"messages_search_toggle_button": messagesSearchToggleButtonElementID,
			"messages_search_container":     messagesSearchContainerElementID,
			"messages_search_input":         messagesSearchInputElementID,
			"feedback_count":                feedbackCountElementID,
		},
		ButtonClasses: map[string]string{
			"new_site_default":     newSiteButtonClass,
			"new_site_active":      newSiteButtonActiveClass,
			"create":               siteFormCreateButtonClass,
			"update":               siteFormUpdateButtonClass,
			"save_default":         dashboardActionButtonSuccessClass,
			"copy_default":         dashboardActionButtonPrimaryClass,
			"refresh_default":      dashboardActionButtonSecondaryClass,
			"delete_site_default":  deleteSiteButtonClass,
			"delete_site_disabled": deleteSiteButtonDisabledClass,
		},
		ButtonLabels: map[string]string{
			"create":          siteFormCreateButtonLabel,
			"update":          siteFormUpdateButtonLabel,
			"new_site":        newSiteOptionLabel,
			"copy_default":    "Copy snippet",
			"copy_copied":     "Snippet copied.",
			"copy_failed":     "Copy failed.",
			"refresh_default": "Refresh feedback",
			"refresh_loading": "Refreshing...",
			"refresh_success": "Feedback refreshed.",
			"refresh_failed":  "Refresh failed.",
			"save_saving":     "Saving site...",
			"save_saved":      "Site updated.",
			"save_created":    "Site created.",
			"save_failed":     "Failed to save site.",
		},
		StatusMessages: map[string]string{
			"loading_user":       dashboardStatusLoadingUser,
			"loading_sites":      dashboardStatusLoadingSites,
			"load_failed":        dashboardStatusLoadFailed,
			"saving_site":        dashboardStatusSavingSite,
			"site_saved":         dashboardStatusSiteSaved,
			"creating_site":      dashboardStatusCreatingSite,
			"site_created":       dashboardStatusSiteCreated,
			"deleting_site":      dashboardStatusDeletingSite,
			"site_deleted":       dashboardStatusSiteDeleted,
			"delete_site_failed": dashboardStatusDeleteFailed,
			"select_site":        dashboardStatusSelectSite,
			"no_messages":        dashboardStatusNoMessages,
			"no_sites":           dashboardStatusNoSites,
			"widget_copied":      dashboardStatusWidgetCopied,
			"widget_copy_failed": dashboardStatusWidgetCopyFailed,
			"no_site_matches":    dashboardStatusNoSiteMatches,
			"no_message_matches": dashboardStatusNoMessageMatches,
		},
		RoleLabels: map[string]string{
			"admin": dashboardRoleAdminLabel,
			"user":  dashboardRoleUserLabel,
		},
		ButtonStyles: map[string]string{
			"primary":   dashboardActionButtonPrimaryClass,
			"success":   dashboardActionButtonSuccessClass,
			"secondary": dashboardActionButtonSecondaryClass,
			"danger":    dashboardActionButtonDangerClass,
		},
		ComponentClasses: map[string]string{
			"site_list_item":         siteListItemClass,
			"site_list_item_active":  siteListItemActiveClass,
			"site_list_item_header":  siteListItemHeaderClass,
			"site_list_item_favicon": siteListItemFaviconClass,
		},
		WidgetTexts: map[string]string{
			"unavailable": dashboardWidgetUnavailable,
		},
		ThemeStorageKey: themeStorageKey,
		OptionValues: map[string]string{
			"new_site": newSiteOptionValue,
		},
		FormStatusClasses: map[string]string{
			"base":    formStatusBaseClass,
			"success": formStatusSuccessClass,
			"danger":  formStatusDangerClass,
		},
		FooterThemeClasses: map[string]string{
			"light": footerThemeLightClass,
			"dark":  footerThemeDarkClass,
		},
		TableThemeClasses: map[string]string{
			"light": feedbackTableHeaderLightClass,
			"dark":  feedbackTableHeaderDarkClass,
		},
		ValidationMessages: map[string]string{
			validationMessageNameRequiredKey: dashboardValidationNameMessage,
			validationMessageOriginKey:       dashboardValidationOriginMessage,
			validationMessageOwnerKey:        dashboardValidationOwnerMessage,
		},
	}

	configPayload, marshalErr := json.Marshal(clientConfig)
	if marshalErr != nil {
		handlers.logger.Warn("render_dashboard_config", zap.Error(marshalErr))
		configPayload = []byte("{}")
	}
	data.ClientConfigJSON = template.JS(configPayload)

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_dashboard", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{jsonKeyError: "render_failed"})
		return
	}

	context.Data(http.StatusOK, dashboardHTMLContentType, buffer.Bytes())
}
