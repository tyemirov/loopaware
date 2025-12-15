package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	dashboardTemplateName                      = "dashboard"
	dashboardHTMLContentType                   = "text/html; charset=utf-8"
	dashboardPageTitle                         = "LoopAware Dashboard"
	dashboardStatusLoadingUser                 = "Loading account information..."
	dashboardStatusLoadingSites                = "Loading sites..."
	dashboardStatusLoadFailed                  = "Failed to load data."
	dashboardStatusSavingSite                  = "Saving site..."
	dashboardStatusSiteSaved                   = "Site updated."
	dashboardStatusCreatingSite                = "Creating site..."
	dashboardStatusSiteCreated                 = "Site created."
	dashboardStatusSelectSite                  = "Select a site to see details."
	dashboardStatusNoMessages                  = "No feedback yet."
	dashboardStatusNoSites                     = "No sites available yet."
	dashboardRoleAdminLabel                    = "Administrator"
	dashboardRoleUserLabel                     = "User"
	dashboardRoleAdminValue                    = "admin"
	dashboardRoleUserValue                     = "user"
	dashboardFeedbackPlaceholder               = "Select a site to load feedback."
	dashboardWidgetCardTitle                   = "Feedback widget"
	dashboardWidgetInstructions                = "Embed this <script> tag on pages served from the allowed origin."
	subscribeWidgetCardTitle                   = "Subscribers widget"
	subscribeWidgetInstructions                = "Place this snippet on pages where you want visitors to subscribe."
	trafficWidgetCardTitle                     = "Traffic widget"
	trafficWidgetInstructions                  = "Add this pixel to every page to capture visit counts."
	dashboardWidgetUnavailable                 = "Save the site to generate a widget snippet."
	dashboardWidgetPlacementTitle              = "Widget placement"
	dashboardWidgetPlacementSideLabel          = "Bubble position"
	dashboardWidgetPlacementLeftLabel          = "Left"
	dashboardWidgetPlacementRightLabel         = "Right"
	dashboardWidgetPlacementBottomOffsetLabel  = "Bottom offset (px)"
	dashboardWidgetPlacementBottomOffsetHelp   = "Keeps the bubble above sticky footers."
	dashboardStatusWidgetCopied                = "Widget snippet copied."
	dashboardStatusWidgetCopyFailed            = "Unable to copy widget snippet."
	dashboardFooterBrandPrefix                 = "Built by"
	dashboardFooterBrandName                   = "Marco Polo Research Lab"
	dashboardFooterBrandURL                    = "https://mprlab.com"
	dashboardFooterToggleButtonID              = "dashboard-footer-toggle"
	dashboardHeaderLogoElementID               = "dashboard-header-logo"
	navbarSettingsButtonLabel                  = "Account settings"
	navbarLogoutLabel                          = "Logout"
	newSiteOptionValue                         = "__new__"
	newSiteOptionLabel                         = "New site"
	siteFormCreateButtonLabel                  = "Create site"
	siteFormUpdateButtonLabel                  = "Update site"
	dashboardActionButtonPrimaryClass          = "btn btn-outline-primary btn-sm"
	dashboardActionButtonSuccessClass          = "btn btn-outline-success btn-sm"
	dashboardActionButtonSecondaryClass        = "btn btn-outline-secondary btn-sm"
	dashboardActionButtonDangerClass           = "btn btn-outline-danger btn-sm"
	siteFormCreateButtonClass                  = dashboardActionButtonPrimaryClass
	siteFormUpdateButtonClass                  = dashboardActionButtonSuccessClass
	userNameElementID                          = "user-name"
	userEmailElementID                         = "user-email"
	userRoleBadgeElementID                     = "user-role"
	userAvatarElementID                        = "user-avatar"
	sitesListElementID                         = "sites-list"
	emptySitesMessageElementID                 = "empty-sites-message"
	siteFormElementID                          = "site-form"
	editSiteNameInputElementID                 = "edit-site-name"
	editSiteOriginInputElementID               = "edit-site-origin"
	editSiteOwnerContainerElementID            = "edit-site-owner-container"
	editSiteOwnerInputElementID                = "edit-site-owner"
	saveSiteButtonElementID                    = "save-site-button"
	refreshMessagesButtonElementID             = "refresh-messages-button"
	feedbackTableHeaderElementID               = "feedback-table-header"
	feedbackTableHeaderLightClass              = "table-light"
	feedbackTableHeaderDarkClass               = "table-dark"
	feedbackTableBodyElementID                 = "feedback-table-body"
	logoutButtonElementID                      = "logout-button"
	sessionTimeoutContainerElementID           = "session-timeout-notification"
	sessionTimeoutInnerClass                   = "container d-flex flex-column flex-md-row align-items-center justify-content-between gap-3"
	sessionTimeoutMessageElementID             = "session-timeout-message"
	sessionTimeoutActionsClass                 = "session-timeout-actions d-flex flex-shrink-0 gap-2"
	sessionTimeoutMessageClass                 = "session-timeout-message fw-semibold mb-0"
	sessionTimeoutConfirmButtonElementID       = "session-timeout-confirm-button"
	sessionTimeoutDismissButtonElementID       = "session-timeout-dismiss-button"
	sessionTimeoutContainerBaseClass           = "session-timeout-banner position-fixed start-0 end-0 border-top py-3 w-100 d-none z-3"
	sessionTimeoutContainerVisibleClass        = "d-block"
	sessionTimeoutContainerHiddenClass         = "d-none"
	sessionTimeoutLightThemeClass              = "bg-body-secondary text-dark border-light-subtle"
	sessionTimeoutDarkThemeClass               = "bg-dark-subtle text-light border-secondary-subtle"
	sessionTimeoutPromptText                   = "Log out due to inactivity?"
	sessionTimeoutConfirmButtonLabel           = "Yes"
	sessionTimeoutDismissButtonLabel           = "No"
	sessionTimeoutConfirmButtonClass           = "btn btn-outline-danger btn-sm"
	sessionTimeoutDismissButtonClass           = "btn btn-outline-secondary btn-sm"
	sessionTimeoutPromptDelayMilliseconds      = 60000
	sessionTimeoutAutoLogoutMilliseconds       = 120000
	widgetSnippetTextareaElementID             = "widget-snippet"
	copyWidgetSnippetButtonElementID           = "copy-widget-snippet"
	widgetTestButtonElementID                  = "widget-test-button"
	subscribeTestButtonElementID               = "subscribe-test-button"
	trafficTestButtonElementID                 = "traffic-test-button"
	dashboardWidgetTestButtonLabel             = "Test"
	dashboardWidgetTestPathPrefix              = "/app/sites/"
	dashboardWidgetTestPathSuffix              = "/widget-test"
	dashboardSubscribeTestPathPrefix           = "/app/sites/"
	dashboardSubscribeTestPathSuffix           = "/subscribe-test"
	dashboardTrafficTestPathPrefix             = "/app/sites/"
	dashboardTrafficTestPathSuffix             = "/traffic-test"
	widgetPlacementSideInputName               = "widget-bubble-side"
	widgetPlacementSideLeftInputElementID      = "widget-placement-side-left"
	widgetPlacementSideRightInputElementID     = "widget-placement-side-right"
	widgetPlacementBottomOffsetInputElementID  = "widget-placement-bottom-offset"
	widgetPlacementBottomOffsetHelpElementID   = "widget-placement-bottom-offset-help"
	subscribeWidgetSnippetTextareaElementID    = "subscribe-widget-snippet"
	trafficWidgetSnippetTextareaElementID      = "traffic-widget-snippet"
	copySubscribeWidgetSnippetButtonElementID  = "copy-subscribe-widget-snippet"
	copyTrafficWidgetSnippetButtonElementID    = "copy-traffic-widget-snippet"
	settingsButtonElementID                    = "settings-button"
	settingsMenuElementID                      = "settings-menu"
	settingsAvatarImageElementID               = "settings-avatar-image"
	settingsAvatarFallbackElementID            = "settings-avatar-fallback"
	settingsMenuSettingsButtonElementID        = "settings-menu-settings"
	settingsMenuSettingsLabel                  = "Settings"
	settingsModalElementID                     = "settings-modal"
	settingsModalTitleElementID                = "settings-modal-title"
	settingsModalContentElementID              = "settings-modal-content"
	settingsModalTitle                         = navbarSettingsButtonLabel
	settingsModalIntroText                     = "Manage LoopAware account preferences."
	settingsModalCloseButtonLabel              = "Close"
	settingsAutoLogoutSectionTitle             = "Auto logout"
	settingsAutoLogoutDescription              = "Control how the dashboard handles inactivity."
	settingsAutoLogoutEnableLabel              = "Enable auto logout"
	settingsAutoLogoutPromptLabel              = "Show reminder after (seconds)"
	settingsAutoLogoutLogoutLabel              = "Sign out after (seconds)"
	settingsAutoLogoutHelpText                 = "Synthetic activity such as scripted mouse moves will not dismiss the reminder."
	settingsAutoLogoutPromptError              = "Enter a whole number between %d and %d."
	settingsAutoLogoutLogoutError              = "Enter a whole number between %d and %d."
	settingsAutoLogoutGapError                 = "Choose a sign-out time that is at least %d seconds after the reminder."
	settingsAutoLogoutFieldsContainerElementID = "settings-auto-logout-fields"
	settingsAutoLogoutToggleElementID          = "settings-auto-logout-enabled"
	settingsAutoLogoutPromptInputElementID     = "settings-auto-logout-prompt-seconds"
	settingsAutoLogoutLogoutInputElementID     = "settings-auto-logout-logout-seconds"
	settingsAutoLogoutPromptErrorElementID     = "settings-auto-logout-prompt-error"
	settingsAutoLogoutLogoutErrorElementID     = "settings-auto-logout-logout-error"
	settingsAutoLogoutStorageKey               = "loopaware_dashboard_auto_logout"
	widgetBottomOffsetDecreaseButtonElementID  = "widget-bottom-offset-decrease"
	widgetBottomOffsetIncreaseButtonElementID  = "widget-bottom-offset-increase"
	widgetBottomOffsetDecreaseLabel            = "-10 px"
	widgetBottomOffsetIncreaseLabel            = "+10 px"
	widgetBottomOffsetDecreaseAriaLabel        = "Decrease bottom offset by 10 pixels"
	widgetBottomOffsetIncreaseAriaLabel        = "Increase bottom offset by 10 pixels"
	widgetBottomOffsetStepPixels               = 10
	autoLogoutMinimumPromptSeconds             = 10
	autoLogoutMaximumPromptSeconds             = 3600
	autoLogoutMinimumLogoutSeconds             = 20
	autoLogoutMaximumLogoutSeconds             = 7200
	autoLogoutMinimumGapSeconds                = 5
	themeStorageKey                            = "loopaware_dashboard_theme"
	formStatusElementID                        = "site-status"
	widgetStatusElementID                      = "widget-status"
	messagesStatusElementID                    = "messages-status"
	newSiteButtonElementID                     = "new-site-button"
	newSiteButtonClass                         = dashboardActionButtonPrimaryClass
	newSiteButtonActiveClass                   = "btn btn-primary btn-sm"
	siteListItemClass                          = "list-group-item list-group-item-action"
	siteListItemActiveClass                    = "active"
	subscriberCountElementID                   = "subscriber-count"
	subscribersTableBodyElementID              = "subscribers-table-body"
	exportSubscribersButtonElementID           = "export-subscribers-button"
	exportSubscribersButtonLabel               = "Export CSV"
	exportButtonClass                          = "btn btn-outline-secondary btn-sm"
	subscribersStatusElementID                 = "subscribers-status"
	subscribersPlaceholder                     = "No subscribers yet."
	visitCountElementID                        = "visit-count"
	uniqueVisitorCountElementID                = "unique-visitor-count"
	trafficStatusElementID                     = "traffic-status"
	topPagesTableBodyElementID                 = "top-pages-table-body"
	topPagesPlaceholder                        = "No visits yet."
	clientConfigElementID                      = "dashboard-config"
	dashboardStatusDeletingSite                = "Deleting site..."
	dashboardStatusSiteDeleted                 = "Site deleted."
	dashboardStatusDeleteFailed                = "Failed to delete site."
	dashboardBootstrapIconsIntegrityAttr       = "integrity=\"sha384-XGjxtQfXaH2tnPFa9x+ruJTuLE3Aa6LhHSWRr1XeTyhezb4abCG4ccI5AkVDxqC+\""
	dashboardFaviconDataURI                    = `data:image/svg+xml;utf8,
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
	deleteSiteButtonElementID              = "delete-site-button"
	deleteSiteButtonClass                  = "btn btn-sm border-0 bg-transparent text-danger opacity-100"
	deleteSiteButtonDisabledClass          = "btn btn-sm border-0 bg-transparent text-danger opacity-100 disabled"
	deleteSiteIconClass                    = "bi bi-trash3-fill text-danger"
	footerElementID                        = "dashboard-footer"
	footerInnerElementID                   = "dashboard-footer-inner"
	footerBaseClass                        = "mt-auto py-2 fixed-bottom border-top"
	footerThemeLightClass                  = "bg-body text-body-secondary"
	footerThemeDarkClass                   = "bg-dark text-light border-light"
	deleteSiteModalElementID               = "delete-site-modal"
	deleteSiteModalTitle                   = "Delete site"
	deleteSiteModalDescription             = "This action permanently removes the site and its feedback."
	deleteSiteModalInputElementID          = "delete-site-confirm-name"
	deleteSiteModalInputLabel              = "Type the site name to confirm"
	deleteSiteModalInputPlaceholder        = "Enter the site name"
	deleteSiteModalConfirmButtonID         = "delete-site-confirm-button"
	deleteSiteModalConfirmButtonLabel      = "Delete site"
	deleteSiteModalConfirmButtonClass      = "btn btn-danger"
	deleteSiteModalCancelButtonLabel       = "Cancel"
	deleteSiteModalCancelButtonClass       = "btn btn-secondary"
	deleteSiteTargetNameElementID          = "delete-site-target-name"
	deleteSiteModalHintPrefix              = "Type "
	deleteSiteModalHintSuffix              = " exactly to confirm."
	formStatusBaseClass                    = "d-none py-1 px-2 small rounded"
	formStatusSuccessClass                 = "py-1 px-2 small rounded border border-success-subtle text-success-emphasis bg-success-subtle"
	formStatusDangerClass                  = "py-1 px-2 small rounded border border-danger-subtle text-danger-emphasis bg-danger-subtle"
	fieldHelpButtonClass                   = "btn btn-link p-0 text-secondary"
	fieldHelpButtonTabIndexValue           = "-1"
	fieldHelpIconClass                     = "bi bi-question-circle-fill"
	fieldHelpTextClass                     = "form-text text-muted"
	siteListItemHeaderClass                = "d-flex align-items-center gap-2"
	siteListItemFaviconClass               = "flex-shrink-0 rounded border bg-white"
	siteCreatedAtElementID                 = "site-created-at"
	siteCreatedAtContainerElementID        = "site-created-at-container"
	siteCreatedAtPlaceholder               = "Not saved yet."
	feedbackCountElementID                 = "feedback-count"
	siteNameHelpButtonElementID            = "site-name-help-button"
	siteNameHelpTitle                      = "Site name"
	siteNameHelpContent                    = "Displayed in the sites list for your team."
	allowedOriginHelpButtonElementID       = "allowed-origin-help-button"
	allowedOriginHelpTitle                 = "Allowed origins"
	allowedOriginHelpContent               = "One or more origins (protocol, host, optional port) where the widget will run; separate multiple origins with commas or spaces."
	ownerEmailHelpButtonElementID          = "owner-email-help-button"
	ownerEmailHelpTitle                    = "Owner email"
	ownerEmailHelpContent                  = "Receives notifications when visitors submit feedback."
	siteSearchToggleButtonElementID        = "site-search-toggle-button"
	siteSearchContainerElementID           = "site-search-container"
	siteSearchInputElementID               = "site-search-input"
	siteSearchToggleLabel                  = "Toggle site search"
	siteSearchPlaceholder                  = "Search sites"
	messagesSearchToggleButtonElementID    = "messages-search-toggle-button"
	messagesSearchContainerElementID       = "messages-search-container"
	messagesSearchInputElementID           = "messages-search-input"
	messagesSearchToggleLabel              = "Toggle feedback search"
	messagesSearchPlaceholder              = "Search feedback"
	searchToggleButtonClass                = "btn btn-link p-0 text-secondary"
	searchInputClass                       = "form-control form-control-sm"
	dashboardStatusNoSiteMatches           = "No sites match your search."
	dashboardStatusNoMessageMatches        = "No feedback matches your search."
	validationMessageNameRequiredKey       = "name_required"
	validationMessageOriginKey             = "origin_invalid"
	validationMessageOwnerKey              = "owner_invalid"
	validationMessageWidgetOffsetKey       = "widget_offset_invalid"
	dashboardValidationNameMessage         = "Site name is required."
	dashboardValidationOriginMessage       = "Allowed origins must include protocol and hostname, for example https://example.com http://localhost:8080."
	dashboardValidationOwnerMessage        = "Provide a valid owner email address."
	dashboardValidationWidgetOffsetMessage = "Provide a whole number between 0 and 240."
	dashboardValidationWidgetSideMessage   = "Choose left or right for the widget bubble."
	dashboardErrorMessageSiteExists        = "A site for this allowed origin already exists."
	dashboardErrorMessageInvalidJSON       = "Submitted data could not be parsed."
	dashboardErrorMessageMissingFields     = "Provide site name and allowed origin."
	dashboardErrorMessageInvalidOwner      = dashboardValidationOwnerMessage
	dashboardErrorMessageNotAuthorized     = "You are not allowed to manage that site."
	dashboardErrorMessageSaveFailed        = "Failed to save site."
)

type dashboardTemplateData struct {
	PageTitle                           string
	APIMeEndpoint                       string
	APISitesEndpoint                    string
	APISiteUpdateEndpointPrefix         string
	APIMessagesEndpointPrefix           string
	APIMessagesEndpointSuffix           string
	LogoutPath                          string
	LoginPath                           string
	BootstrapIconsIntegrityAttr         template.HTMLAttr
	FaviconDataURI                      template.URL
	HeaderLogoDataURI                   template.URL
	HeaderLogoImageID                   string
	StatusLoadingUser                   string
	StatusLoadingSites                  string
	StatusLoadFailed                    string
	StatusSavingSite                    string
	StatusSiteSaved                     string
	StatusCreatingSite                  string
	StatusSiteCreated                   string
	StatusDeletingSite                  string
	StatusSiteDeleted                   string
	StatusDeleteSiteFailed              string
	StatusSelectSite                    string
	StatusNoMessages                    string
	StatusNoSites                       string
	RoleAdmin                           string
	RoleUser                            string
	EmptySitesMessage                   string
	FeedbackPlaceholder                 string
	FooterHTML                          template.HTML
	FooterElementID                     string
	FooterInnerElementID                string
	FooterBaseClass                     string
	UserNameID                          string
	UserEmailID                         string
	UserRoleBadgeID                     string
	UserAvatarID                        string
	SitesListID                         string
	EmptySitesMessageID                 string
	SiteFormID                          string
	EditSiteNameInputID                 string
	EditSiteOriginInputID               string
	EditSiteOwnerContainerID            string
	EditSiteOwnerInputID                string
	SiteCreatedAtElementID              string
	SiteCreatedAtContainerID            string
	SiteCreatedAtPlaceholder            string
	SiteSearchToggleButtonID            string
	SiteSearchToggleLabel               string
	SiteSearchContainerID               string
	SiteSearchInputID                   string
	SiteSearchPlaceholder               string
	SaveSiteButtonID                    string
	SaveButtonSaving                    string
	SaveButtonSaved                     string
	SaveButtonCreated                   string
	SaveButtonFailed                    string
	SaveButtonDefaultClass              string
	RefreshMessagesButtonID             string
	RefreshButtonLoading                string
	RefreshButtonSuccess                string
	RefreshButtonFailed                 string
	RefreshButtonDefaultLabel           string
	RefreshButtonDefaultClass           string
	ActionButtonPrimaryClass            string
	ActionButtonSuccessClass            string
	ActionButtonSecondaryClass          string
	ActionButtonDangerClass             string
	FeedbackTableHeaderID               string
	FeedbackTableHeaderLightClass       string
	FeedbackTableBodyID                 string
	SubscriberCountElementID            string
	SubscribersTableBodyID              string
	ExportSubscribersButtonID           string
	ExportSubscribersButtonLabel        string
	ExportButtonClass                   string
	SubscribersStatusID                 string
	SubscribersPlaceholder              string
	VisitCountElementID                 string
	UniqueVisitorCountElementID         string
	TrafficStatusID                     string
	TopPagesTableBodyID                 string
	TopPagesPlaceholder                 string
	LogoutButtonID                      string
	NewSiteOptionValue                  string
	CreateButtonLabel                   string
	UpdateButtonLabel                   string
	CreateButtonClass                   string
	UpdateButtonClass                   string
	NewSiteButtonID                     string
	NewSiteButtonLabel                  string
	NewSiteButtonClass                  string
	NewSiteButtonActiveClass            string
	DeleteSiteButtonID                  string
	DeleteSiteButtonLabel               string
	DeleteSiteButtonClass               string
	DeleteSiteButtonDisabledClass       string
	DeleteSiteIconClass                 string
	SiteListItemClass                   string
	SiteListItemActiveClass             string
	WidgetCardTitle                     string
	WidgetInstructions                  string
	WidgetUnavailableMessage            string
	SubscribeWidgetTitle                string
	SubscribeWidgetInstructions         string
	TrafficWidgetTitle                  string
	TrafficWidgetInstructions           string
	StatusWidgetCopied                  string
	StatusWidgetCopyFailed              string
	WidgetSnippetTextareaID             string
	SubscribeSnippetTextareaID          string
	TrafficWidgetSnippetTextareaID      string
	CopyWidgetSnippetButtonID           string
	CopySubscribeSnippetButtonID        string
	CopyTrafficSnippetButtonID          string
	WidgetTestButtonID                  string
	WidgetTestButtonLabel               string
	WidgetTestButtonClass               string
	SubscribeTestButtonID               string
	SubscribeTestButtonLabel            string
	SubscribeTestButtonClass            string
	TrafficTestButtonID                 string
	TrafficTestButtonLabel              string
	TrafficTestButtonClass              string
	CopyButtonCopied                    string
	CopyButtonFailed                    string
	CopyButtonDefaultLabel              string
	CopyButtonDefaultClass              string
	WidgetPlacementTitle                string
	WidgetPlacementSideLabel            string
	WidgetPlacementLeftLabel            string
	WidgetPlacementRightLabel           string
	WidgetPlacementBottomOffsetLabel    string
	WidgetPlacementBottomOffsetHelp     string
	WidgetTestPagePrefix                string
	WidgetTestPageSuffix                string
	SubscribeTestPagePrefix             string
	SubscribeTestPageSuffix             string
	TrafficTestPagePrefix               string
	TrafficTestPageSuffix               string
	SettingsButtonID                    string
	SettingsButtonLabel                 string
	LogoutLabel                         string
	SettingsMenuID                      string
	SettingsMenuSettingsButtonID        string
	SettingsMenuSettingsLabel           string
	SettingsModalID                     string
	SettingsModalTitleID                string
	SettingsModalTitle                  string
	SettingsModalIntro                  string
	SettingsModalCloseLabel             string
	SettingsModalContentID              string
	SettingsAutoLogoutSectionTitle      string
	SettingsAutoLogoutDescription       string
	SettingsAutoLogoutEnableLabel       string
	SettingsAutoLogoutPromptLabel       string
	SettingsAutoLogoutLogoutLabel       string
	SettingsAutoLogoutHelpText          string
	SettingsAutoLogoutPromptError       string
	SettingsAutoLogoutLogoutError       string
	SettingsAutoLogoutGapError          string
	SettingsAutoLogoutFieldsID          string
	SettingsAutoLogoutToggleID          string
	SettingsAutoLogoutPromptInputID     string
	SettingsAutoLogoutLogoutInputID     string
	SettingsAutoLogoutPromptErrorID     string
	SettingsAutoLogoutLogoutErrorID     string
	SettingsAutoLogoutPromptMin         int
	SettingsAutoLogoutPromptMax         int
	SettingsAutoLogoutLogoutMin         int
	SettingsAutoLogoutLogoutMax         int
	SettingsAutoLogoutGapSeconds        int
	WidgetBottomOffsetDecreaseButtonID  string
	WidgetBottomOffsetIncreaseButtonID  string
	WidgetBottomOffsetDecreaseLabel     string
	WidgetBottomOffsetIncreaseLabel     string
	WidgetBottomOffsetDecreaseAriaLabel string
	WidgetBottomOffsetIncreaseAriaLabel string
	WidgetBottomOffsetStep              int
	ThemeStorageKey                     string
	PublicThemeStorageKey               string
	LandingThemeStorageKey              string
	SettingsAvatarImageID               string
	SettingsAvatarFallbackID            string
	FormStatusID                        string
	FormStatusBaseClass                 string
	FormStatusSuccessClass              string
	FormStatusDangerClass               string
	SearchToggleButtonClass             string
	SearchInputClass                    string
	FieldHelpButtonClass                string
	FieldHelpButtonTabIndex             string
	FieldHelpIconClass                  string
	FieldHelpTextClass                  string
	SiteNameHelpButtonID                string
	SiteNameHelpTitle                   string
	SiteNameHelpContent                 string
	AllowedOriginHelpButtonID           string
	AllowedOriginHelpTitle              string
	AllowedOriginHelpContent            string
	OwnerEmailHelpButtonID              string
	OwnerEmailHelpTitle                 string
	OwnerEmailHelpContent               string
	MessagesSearchToggleButtonID        string
	MessagesSearchToggleLabel           string
	MessagesSearchContainerID           string
	MessagesSearchInputID               string
	MessagesSearchPlaceholder           string
	FeedbackCountElementID              string
	WidgetStatusID                      string
	WidgetPlacementSideLeftID           string
	WidgetPlacementSideRightID          string
	WidgetPlacementSideInputName        string
	WidgetBottomOffsetInputID           string
	WidgetBottomOffsetHelpID            string
	WidgetBottomOffsetMin               string
	WidgetBottomOffsetMax               string
	MessagesStatusID                    string
	SessionTimeoutContainerID           string
	SessionTimeoutContainerClass        string
	SessionTimeoutInnerClass            string
	SessionTimeoutMessageID             string
	SessionTimeoutMessageClass          string
	SessionTimeoutPromptText            string
	SessionTimeoutActionsClass          string
	SessionTimeoutConfirmButtonID       string
	SessionTimeoutConfirmLabel          string
	SessionTimeoutConfirmButtonClass    string
	SessionTimeoutDismissButtonID       string
	SessionTimeoutDismissLabel          string
	SessionTimeoutDismissButtonClass    string
	DeleteSiteModalID                   string
	DeleteSiteModalTitle                string
	DeleteSiteModalDescription          string
	DeleteSiteModalInputID              string
	DeleteSiteModalInputLabel           string
	DeleteSiteModalInputPlaceholder     string
	DeleteSiteModalConfirmButtonID      string
	DeleteSiteModalConfirmButtonLabel   string
	DeleteSiteModalConfirmButtonClass   string
	DeleteSiteModalCancelButtonLabel    string
	DeleteSiteModalCancelButtonClass    string
	DeleteSiteTargetNameID              string
	DeleteSiteModalHintPrefix           string
	DeleteSiteModalHintSuffix           string
	ClientConfigElementID               string
	ClientConfigJSON                    template.JS
}

type sessionTimeoutConfig struct {
	PromptDelayMilliseconds int               `json:"prompt_delay_ms"`
	AutoLogoutMilliseconds  int               `json:"auto_logout_ms"`
	Texts                   map[string]string `json:"texts"`
	ComponentClasses        map[string]string `json:"component_classes"`
	ThemeClasses            map[string]string `json:"theme_classes"`
}

type widgetPlacementClientConfig struct {
	InputName           string            `json:"input_name"`
	DefaultSide         string            `json:"default_side"`
	DefaultBottomOffset int               `json:"default_bottom_offset"`
	Sides               map[string]string `json:"sides"`
	BottomOffset        rangeConfig       `json:"bottom_offset"`
}

type rangeConfig struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type autoLogoutClientConfig struct {
	StorageKey        string `json:"storage_key"`
	MinPromptSeconds  int    `json:"min_prompt_seconds"`
	MaxPromptSeconds  int    `json:"max_prompt_seconds"`
	MinLogoutSeconds  int    `json:"min_logout_seconds"`
	MaxLogoutSeconds  int    `json:"max_logout_seconds"`
	MinimumGapSeconds int    `json:"minimum_gap_seconds"`
}

type dashboardClientConfig struct {
	APIPaths           map[string]string           `json:"api_paths"`
	Paths              map[string]string           `json:"paths"`
	ElementIDs         map[string]string           `json:"element_ids"`
	ButtonClasses      map[string]string           `json:"button_classes"`
	ButtonLabels       map[string]string           `json:"button_labels"`
	StatusMessages     map[string]string           `json:"status_messages"`
	RoleLabels         map[string]string           `json:"role_labels"`
	RoleValues         map[string]string           `json:"role_values"`
	ButtonStyles       map[string]string           `json:"button_styles"`
	ComponentClasses   map[string]string           `json:"component_classes"`
	WidgetTexts        map[string]string           `json:"widget_texts"`
	ThemeStorageKey    string                      `json:"theme_storage_key"`
	OptionValues       map[string]string           `json:"option_values"`
	Placeholders       map[string]string           `json:"placeholders"`
	FormStatusClasses  map[string]string           `json:"form_status_classes"`
	FooterThemeClasses map[string]string           `json:"footer_theme_classes"`
	TableThemeClasses  map[string]string           `json:"table_theme_classes"`
	ValidationMessages map[string]string           `json:"validation_messages"`
	ErrorMessages      map[string]string           `json:"error_messages"`
	WidgetPlacement    widgetPlacementClientConfig `json:"widget_placement"`
	SessionTimeout     sessionTimeoutConfig        `json:"session_timeout"`
	AutoLogout         autoLogoutClientConfig      `json:"auto_logout"`
}

// DashboardWebHandlers serves the authenticated dashboard UI.
type DashboardWebHandlers struct {
	logger      *zap.Logger
	template    *template.Template
	landingPath string
}

func NewDashboardWebHandlers(logger *zap.Logger, landingPath string) *DashboardWebHandlers {
	baseTemplate := template.Must(template.New(dashboardTemplateName).Parse(dashboardHeaderTemplateHTML))
	compiledTemplate := template.Must(baseTemplate.Parse(dashboardTemplateHTML))
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
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantDashboard)
	if footerErr != nil {
		handlers.logger.Warn("render_dashboard_footer", zap.Error(footerErr))
		footerHTML = template.HTML("")
	}

	data := dashboardTemplateData{
		PageTitle:                           dashboardPageTitle,
		APIMeEndpoint:                       "/api/me",
		APISitesEndpoint:                    "/api/sites",
		APISiteUpdateEndpointPrefix:         "/api/sites/",
		APIMessagesEndpointPrefix:           "/api/sites/",
		APIMessagesEndpointSuffix:           "/messages",
		LogoutPath:                          constants.LogoutPath,
		LoginPath:                           constants.LoginPath,
		BootstrapIconsIntegrityAttr:         template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:                      template.URL(dashboardFaviconDataURI),
		HeaderLogoDataURI:                   landingLogoDataURI,
		HeaderLogoImageID:                   dashboardHeaderLogoElementID,
		StatusLoadingUser:                   dashboardStatusLoadingUser,
		StatusLoadingSites:                  dashboardStatusLoadingSites,
		StatusLoadFailed:                    dashboardStatusLoadFailed,
		StatusSavingSite:                    dashboardStatusSavingSite,
		StatusSiteSaved:                     dashboardStatusSiteSaved,
		StatusCreatingSite:                  dashboardStatusCreatingSite,
		StatusSiteCreated:                   dashboardStatusSiteCreated,
		StatusDeletingSite:                  dashboardStatusDeletingSite,
		StatusSiteDeleted:                   dashboardStatusSiteDeleted,
		StatusDeleteSiteFailed:              dashboardStatusDeleteFailed,
		StatusSelectSite:                    dashboardStatusSelectSite,
		StatusNoMessages:                    dashboardStatusNoMessages,
		StatusNoSites:                       dashboardStatusNoSites,
		RoleAdmin:                           dashboardRoleAdminLabel,
		RoleUser:                            dashboardRoleUserLabel,
		EmptySitesMessage:                   dashboardStatusNoSites,
		FeedbackPlaceholder:                 dashboardFeedbackPlaceholder,
		FooterHTML:                          footerHTML,
		FooterElementID:                     footerElementID,
		FooterInnerElementID:                footerInnerElementID,
		FooterBaseClass:                     footerBaseClass,
		UserNameID:                          userNameElementID,
		UserEmailID:                         userEmailElementID,
		UserRoleBadgeID:                     userRoleBadgeElementID,
		UserAvatarID:                        userAvatarElementID,
		SitesListID:                         sitesListElementID,
		EmptySitesMessageID:                 emptySitesMessageElementID,
		SiteFormID:                          siteFormElementID,
		EditSiteNameInputID:                 editSiteNameInputElementID,
		EditSiteOriginInputID:               editSiteOriginInputElementID,
		EditSiteOwnerContainerID:            editSiteOwnerContainerElementID,
		EditSiteOwnerInputID:                editSiteOwnerInputElementID,
		SiteCreatedAtElementID:              siteCreatedAtElementID,
		SiteCreatedAtContainerID:            siteCreatedAtContainerElementID,
		SiteCreatedAtPlaceholder:            siteCreatedAtPlaceholder,
		SiteSearchToggleButtonID:            siteSearchToggleButtonElementID,
		SiteSearchToggleLabel:               siteSearchToggleLabel,
		SiteSearchContainerID:               siteSearchContainerElementID,
		SiteSearchInputID:                   siteSearchInputElementID,
		SiteSearchPlaceholder:               siteSearchPlaceholder,
		SaveSiteButtonID:                    saveSiteButtonElementID,
		SaveButtonSaving:                    "Saving site...",
		SaveButtonSaved:                     "Site updated.",
		SaveButtonCreated:                   "Site created.",
		SaveButtonFailed:                    "Failed to save site.",
		SaveButtonDefaultClass:              dashboardActionButtonSuccessClass,
		RefreshMessagesButtonID:             refreshMessagesButtonElementID,
		RefreshButtonLoading:                "Refreshing...",
		RefreshButtonSuccess:                "Feedback refreshed.",
		RefreshButtonFailed:                 "Refresh failed.",
		RefreshButtonDefaultLabel:           "Refresh feedback",
		RefreshButtonDefaultClass:           dashboardActionButtonSecondaryClass,
		ActionButtonPrimaryClass:            dashboardActionButtonPrimaryClass,
		ActionButtonSuccessClass:            dashboardActionButtonSuccessClass,
		ActionButtonSecondaryClass:          dashboardActionButtonSecondaryClass,
		ActionButtonDangerClass:             dashboardActionButtonDangerClass,
		FeedbackTableHeaderID:               feedbackTableHeaderElementID,
		FeedbackTableHeaderLightClass:       feedbackTableHeaderLightClass,
		FeedbackTableBodyID:                 feedbackTableBodyElementID,
		SubscriberCountElementID:            subscriberCountElementID,
		SubscribersTableBodyID:              subscribersTableBodyElementID,
		ExportSubscribersButtonID:           exportSubscribersButtonElementID,
		ExportSubscribersButtonLabel:        exportSubscribersButtonLabel,
		ExportButtonClass:                   exportButtonClass,
		SubscribersStatusID:                 subscribersStatusElementID,
		SubscribersPlaceholder:              subscribersPlaceholder,
		VisitCountElementID:                 visitCountElementID,
		UniqueVisitorCountElementID:         uniqueVisitorCountElementID,
		TrafficStatusID:                     trafficStatusElementID,
		TopPagesTableBodyID:                 topPagesTableBodyElementID,
		TopPagesPlaceholder:                 topPagesPlaceholder,
		LogoutButtonID:                      logoutButtonElementID,
		NewSiteOptionValue:                  newSiteOptionValue,
		CreateButtonLabel:                   siteFormCreateButtonLabel,
		UpdateButtonLabel:                   siteFormUpdateButtonLabel,
		CreateButtonClass:                   siteFormCreateButtonClass,
		UpdateButtonClass:                   siteFormUpdateButtonClass,
		NewSiteButtonID:                     newSiteButtonElementID,
		NewSiteButtonLabel:                  newSiteOptionLabel,
		NewSiteButtonClass:                  newSiteButtonClass,
		NewSiteButtonActiveClass:            newSiteButtonActiveClass,
		DeleteSiteButtonID:                  deleteSiteButtonElementID,
		DeleteSiteButtonLabel:               deleteSiteModalConfirmButtonLabel,
		DeleteSiteButtonClass:               deleteSiteButtonClass,
		DeleteSiteButtonDisabledClass:       deleteSiteButtonDisabledClass,
		DeleteSiteIconClass:                 deleteSiteIconClass,
		SiteListItemClass:                   siteListItemClass,
		SiteListItemActiveClass:             siteListItemActiveClass,
		WidgetCardTitle:                     dashboardWidgetCardTitle,
		WidgetInstructions:                  dashboardWidgetInstructions,
		WidgetUnavailableMessage:            dashboardWidgetUnavailable,
		SubscribeWidgetTitle:                subscribeWidgetCardTitle,
		SubscribeWidgetInstructions:         subscribeWidgetInstructions,
		TrafficWidgetTitle:                  trafficWidgetCardTitle,
		TrafficWidgetInstructions:           trafficWidgetInstructions,
		StatusWidgetCopied:                  dashboardStatusWidgetCopied,
		StatusWidgetCopyFailed:              dashboardStatusWidgetCopyFailed,
		WidgetSnippetTextareaID:             widgetSnippetTextareaElementID,
		SubscribeSnippetTextareaID:          subscribeWidgetSnippetTextareaElementID,
		TrafficWidgetSnippetTextareaID:      trafficWidgetSnippetTextareaElementID,
		CopyWidgetSnippetButtonID:           copyWidgetSnippetButtonElementID,
		CopySubscribeSnippetButtonID:        copySubscribeWidgetSnippetButtonElementID,
		CopyTrafficSnippetButtonID:          copyTrafficWidgetSnippetButtonElementID,
		WidgetTestButtonID:                  widgetTestButtonElementID,
		WidgetTestButtonLabel:               dashboardWidgetTestButtonLabel,
		WidgetTestButtonClass:               dashboardActionButtonSecondaryClass,
		SubscribeTestButtonID:               subscribeTestButtonElementID,
		SubscribeTestButtonLabel:            dashboardWidgetTestButtonLabel,
		SubscribeTestButtonClass:            dashboardActionButtonSecondaryClass,
		TrafficTestButtonID:                 trafficTestButtonElementID,
		TrafficTestButtonLabel:              dashboardWidgetTestButtonLabel,
		TrafficTestButtonClass:              dashboardActionButtonSecondaryClass,
		CopyButtonCopied:                    "Snippet copied.",
		CopyButtonFailed:                    "Copy failed.",
		CopyButtonDefaultLabel:              "Copy snippet",
		CopyButtonDefaultClass:              dashboardActionButtonPrimaryClass,
		WidgetPlacementTitle:                dashboardWidgetPlacementTitle,
		WidgetPlacementSideLabel:            dashboardWidgetPlacementSideLabel,
		WidgetPlacementLeftLabel:            dashboardWidgetPlacementLeftLabel,
		WidgetPlacementRightLabel:           dashboardWidgetPlacementRightLabel,
		WidgetPlacementBottomOffsetLabel:    dashboardWidgetPlacementBottomOffsetLabel,
		WidgetPlacementBottomOffsetHelp:     dashboardWidgetPlacementBottomOffsetHelp,
		WidgetTestPagePrefix:                dashboardWidgetTestPathPrefix,
		WidgetTestPageSuffix:                dashboardWidgetTestPathSuffix,
		SubscribeTestPagePrefix:             dashboardSubscribeTestPathPrefix,
		SubscribeTestPageSuffix:             dashboardSubscribeTestPathSuffix,
		TrafficTestPagePrefix:               dashboardTrafficTestPathPrefix,
		TrafficTestPageSuffix:               dashboardTrafficTestPathSuffix,
		WidgetPlacementSideLeftID:           widgetPlacementSideLeftInputElementID,
		WidgetPlacementSideRightID:          widgetPlacementSideRightInputElementID,
		WidgetPlacementSideInputName:        widgetPlacementSideInputName,
		WidgetBottomOffsetInputID:           widgetPlacementBottomOffsetInputElementID,
		WidgetBottomOffsetHelpID:            widgetPlacementBottomOffsetHelpElementID,
		WidgetBottomOffsetMin:               strconv.Itoa(minWidgetBubbleBottomOffset),
		WidgetBottomOffsetMax:               strconv.Itoa(maxWidgetBubbleBottomOffset),
		SettingsButtonID:                    settingsButtonElementID,
		SettingsButtonLabel:                 navbarSettingsButtonLabel,
		LogoutLabel:                         navbarLogoutLabel,
		SettingsMenuID:                      settingsMenuElementID,
		SettingsMenuSettingsButtonID:        settingsMenuSettingsButtonElementID,
		SettingsMenuSettingsLabel:           settingsMenuSettingsLabel,
		SettingsModalID:                     settingsModalElementID,
		SettingsModalTitleID:                settingsModalTitleElementID,
		SettingsModalTitle:                  settingsModalTitle,
		SettingsModalIntro:                  settingsModalIntroText,
		SettingsModalCloseLabel:             settingsModalCloseButtonLabel,
		SettingsModalContentID:              settingsModalContentElementID,
		SettingsAutoLogoutSectionTitle:      settingsAutoLogoutSectionTitle,
		SettingsAutoLogoutDescription:       settingsAutoLogoutDescription,
		SettingsAutoLogoutEnableLabel:       settingsAutoLogoutEnableLabel,
		SettingsAutoLogoutPromptLabel:       settingsAutoLogoutPromptLabel,
		SettingsAutoLogoutLogoutLabel:       settingsAutoLogoutLogoutLabel,
		SettingsAutoLogoutHelpText:          settingsAutoLogoutHelpText,
		SettingsAutoLogoutPromptError:       fmt.Sprintf(settingsAutoLogoutPromptError, autoLogoutMinimumPromptSeconds, autoLogoutMaximumPromptSeconds),
		SettingsAutoLogoutLogoutError:       fmt.Sprintf(settingsAutoLogoutLogoutError, autoLogoutMinimumLogoutSeconds, autoLogoutMaximumLogoutSeconds),
		SettingsAutoLogoutGapError:          fmt.Sprintf(settingsAutoLogoutGapError, autoLogoutMinimumGapSeconds),
		SettingsAutoLogoutFieldsID:          settingsAutoLogoutFieldsContainerElementID,
		SettingsAutoLogoutToggleID:          settingsAutoLogoutToggleElementID,
		SettingsAutoLogoutPromptInputID:     settingsAutoLogoutPromptInputElementID,
		SettingsAutoLogoutLogoutInputID:     settingsAutoLogoutLogoutInputElementID,
		SettingsAutoLogoutPromptErrorID:     settingsAutoLogoutPromptErrorElementID,
		SettingsAutoLogoutLogoutErrorID:     settingsAutoLogoutLogoutErrorElementID,
		SettingsAutoLogoutPromptMin:         autoLogoutMinimumPromptSeconds,
		SettingsAutoLogoutPromptMax:         autoLogoutMaximumPromptSeconds,
		SettingsAutoLogoutLogoutMin:         autoLogoutMinimumLogoutSeconds,
		SettingsAutoLogoutLogoutMax:         autoLogoutMaximumLogoutSeconds,
		SettingsAutoLogoutGapSeconds:        autoLogoutMinimumGapSeconds,
		WidgetBottomOffsetDecreaseButtonID:  widgetBottomOffsetDecreaseButtonElementID,
		WidgetBottomOffsetIncreaseButtonID:  widgetBottomOffsetIncreaseButtonElementID,
		WidgetBottomOffsetDecreaseLabel:     widgetBottomOffsetDecreaseLabel,
		WidgetBottomOffsetIncreaseLabel:     widgetBottomOffsetIncreaseLabel,
		WidgetBottomOffsetDecreaseAriaLabel: widgetBottomOffsetDecreaseAriaLabel,
		WidgetBottomOffsetIncreaseAriaLabel: widgetBottomOffsetIncreaseAriaLabel,
		WidgetBottomOffsetStep:              widgetBottomOffsetStepPixels,
		ThemeStorageKey:                     themeStorageKey,
		PublicThemeStorageKey:               publicThemeStorageKey,
		LandingThemeStorageKey:              publicLandingThemeStorageKey,
		SettingsAvatarImageID:               settingsAvatarImageElementID,
		SettingsAvatarFallbackID:            settingsAvatarFallbackElementID,
		FormStatusID:                        formStatusElementID,
		FormStatusBaseClass:                 formStatusBaseClass,
		FormStatusSuccessClass:              formStatusSuccessClass,
		FormStatusDangerClass:               formStatusDangerClass,
		SearchToggleButtonClass:             searchToggleButtonClass,
		SearchInputClass:                    searchInputClass,
		FieldHelpButtonClass:                fieldHelpButtonClass,
		FieldHelpButtonTabIndex:             fieldHelpButtonTabIndexValue,
		FieldHelpIconClass:                  fieldHelpIconClass,
		FieldHelpTextClass:                  fieldHelpTextClass,
		SiteNameHelpButtonID:                siteNameHelpButtonElementID,
		SiteNameHelpTitle:                   siteNameHelpTitle,
		SiteNameHelpContent:                 siteNameHelpContent,
		AllowedOriginHelpButtonID:           allowedOriginHelpButtonElementID,
		AllowedOriginHelpTitle:              allowedOriginHelpTitle,
		AllowedOriginHelpContent:            allowedOriginHelpContent,
		OwnerEmailHelpButtonID:              ownerEmailHelpButtonElementID,
		OwnerEmailHelpTitle:                 ownerEmailHelpTitle,
		OwnerEmailHelpContent:               ownerEmailHelpContent,
		MessagesSearchToggleButtonID:        messagesSearchToggleButtonElementID,
		MessagesSearchToggleLabel:           messagesSearchToggleLabel,
		MessagesSearchContainerID:           messagesSearchContainerElementID,
		MessagesSearchInputID:               messagesSearchInputElementID,
		MessagesSearchPlaceholder:           messagesSearchPlaceholder,
		FeedbackCountElementID:              feedbackCountElementID,
		WidgetStatusID:                      widgetStatusElementID,
		MessagesStatusID:                    messagesStatusElementID,
		SessionTimeoutContainerID:           sessionTimeoutContainerElementID,
		SessionTimeoutContainerClass:        sessionTimeoutContainerBaseClass,
		SessionTimeoutInnerClass:            sessionTimeoutInnerClass,
		SessionTimeoutMessageID:             sessionTimeoutMessageElementID,
		SessionTimeoutMessageClass:          sessionTimeoutMessageClass,
		SessionTimeoutPromptText:            sessionTimeoutPromptText,
		SessionTimeoutActionsClass:          sessionTimeoutActionsClass,
		SessionTimeoutConfirmButtonID:       sessionTimeoutConfirmButtonElementID,
		SessionTimeoutConfirmLabel:          sessionTimeoutConfirmButtonLabel,
		SessionTimeoutConfirmButtonClass:    sessionTimeoutConfirmButtonClass,
		SessionTimeoutDismissButtonID:       sessionTimeoutDismissButtonElementID,
		SessionTimeoutDismissLabel:          sessionTimeoutDismissButtonLabel,
		SessionTimeoutDismissButtonClass:    sessionTimeoutDismissButtonClass,
		DeleteSiteModalID:                   deleteSiteModalElementID,
		DeleteSiteModalTitle:                deleteSiteModalTitle,
		DeleteSiteModalDescription:          deleteSiteModalDescription,
		DeleteSiteModalInputID:              deleteSiteModalInputElementID,
		DeleteSiteModalInputLabel:           deleteSiteModalInputLabel,
		DeleteSiteModalInputPlaceholder:     deleteSiteModalInputPlaceholder,
		DeleteSiteModalConfirmButtonID:      deleteSiteModalConfirmButtonID,
		DeleteSiteModalConfirmButtonLabel:   deleteSiteModalConfirmButtonLabel,
		DeleteSiteModalConfirmButtonClass:   deleteSiteModalConfirmButtonClass,
		DeleteSiteModalCancelButtonLabel:    deleteSiteModalCancelButtonLabel,
		DeleteSiteModalCancelButtonClass:    deleteSiteModalCancelButtonClass,
		DeleteSiteTargetNameID:              deleteSiteTargetNameElementID,
		DeleteSiteModalHintPrefix:           deleteSiteModalHintPrefix,
		DeleteSiteModalHintSuffix:           deleteSiteModalHintSuffix,
	}

	data.ClientConfigElementID = clientConfigElementID

	clientConfig := dashboardClientConfig{
		APIPaths: map[string]string{
			"me":                      "/api/me",
			"sites":                   "/api/sites",
			"site_update_prefix":      "/api/sites/",
			"site_messages_prefix":    "/api/sites/",
			"site_messages_suffix":    "/messages",
			"site_subscribers_prefix": "/api/sites/",
			"site_subscribers_suffix": "/subscribers",
			"site_subscribers_export": "/subscribers/export",
			"site_subscriber_update":  "/subscribers/",
			"site_visit_stats":        "/visits/stats",
			"site_favicon_events":     "/api/sites/favicons/events",
			"feedback_events":         "/api/sites/feedback/events",
		},
		Paths: map[string]string{
			"logout":                constants.LogoutPath,
			"login":                 constants.LoginPath,
			"landing":               handlers.landingPath,
			"widget_test_prefix":    dashboardWidgetTestPathPrefix,
			"widget_test_suffix":    dashboardWidgetTestPathSuffix,
			"subscribe_test_prefix": dashboardSubscribeTestPathPrefix,
			"subscribe_test_suffix": dashboardSubscribeTestPathSuffix,
			"traffic_test_prefix":   dashboardTrafficTestPathPrefix,
			"traffic_test_suffix":   dashboardTrafficTestPathSuffix,
		},
		ElementIDs: map[string]string{
			"user_name":                         userNameElementID,
			"user_email":                        userEmailElementID,
			"user_avatar":                       userAvatarElementID,
			"user_role":                         userRoleBadgeElementID,
			"sites_list":                        sitesListElementID,
			"empty_sites_message":               emptySitesMessageElementID,
			"site_form":                         siteFormElementID,
			"edit_site_name":                    editSiteNameInputElementID,
			"edit_site_origin":                  editSiteOriginInputElementID,
			"edit_site_owner_container":         editSiteOwnerContainerElementID,
			"edit_site_owner":                   editSiteOwnerInputElementID,
			"site_created_at":                   siteCreatedAtElementID,
			"site_created_at_container":         siteCreatedAtContainerElementID,
			"save_site_button":                  saveSiteButtonElementID,
			"refresh_messages_button":           refreshMessagesButtonElementID,
			"feedback_table_header":             feedbackTableHeaderElementID,
			"feedback_table_body":               feedbackTableBodyElementID,
			"subscriber_count":                  subscriberCountElementID,
			"subscribers_table_body":            subscribersTableBodyElementID,
			"export_subscribers_button":         exportSubscribersButtonElementID,
			"subscribers_status":                subscribersStatusElementID,
			"visit_count":                       visitCountElementID,
			"unique_visitor_count":              uniqueVisitorCountElementID,
			"traffic_status":                    trafficStatusElementID,
			"top_pages_table_body":              topPagesTableBodyElementID,
			"logout_button":                     logoutButtonElementID,
			"widget_snippet_textarea":           widgetSnippetTextareaElementID,
			"copy_widget_snippet_button":        copyWidgetSnippetButtonElementID,
			"subscribe_snippet_textarea":        subscribeWidgetSnippetTextareaElementID,
			"copy_subscribe_snippet_button":     copySubscribeWidgetSnippetButtonElementID,
			"traffic_snippet_textarea":          trafficWidgetSnippetTextareaElementID,
			"copy_traffic_snippet_button":       copyTrafficWidgetSnippetButtonElementID,
			"widget_test_button":                widgetTestButtonElementID,
			"subscribe_test_button":             subscribeTestButtonElementID,
			"traffic_test_button":               trafficTestButtonElementID,
			"settings_button":                   settingsButtonElementID,
			"settings_menu":                     settingsMenuElementID,
			"settings_menu_settings":            settingsMenuSettingsButtonElementID,
			"settings_modal":                    settingsModalElementID,
			"settings_modal_title":              settingsModalTitleElementID,
			"settings_modal_content":            settingsModalContentElementID,
			"settings_auto_logout_fields":       settingsAutoLogoutFieldsContainerElementID,
			"settings_auto_logout_toggle":       settingsAutoLogoutToggleElementID,
			"settings_auto_logout_prompt":       settingsAutoLogoutPromptInputElementID,
			"settings_auto_logout_logout":       settingsAutoLogoutLogoutInputElementID,
			"settings_auto_logout_prompt_error": settingsAutoLogoutPromptErrorElementID,
			"settings_auto_logout_logout_error": settingsAutoLogoutLogoutErrorElementID,
			"settings_avatar_image":             settingsAvatarImageElementID,
			"settings_avatar_fallback":          settingsAvatarFallbackElementID,
			"form_status":                       formStatusElementID,
			"new_site_button":                   newSiteButtonElementID,
			"delete_site_button":                deleteSiteButtonElementID,
			"delete_site_modal":                 deleteSiteModalElementID,
			"delete_site_confirm_button":        deleteSiteModalConfirmButtonID,
			"delete_site_confirm_input":         deleteSiteModalInputElementID,
			"delete_site_target_name":           deleteSiteTargetNameElementID,
			"footer":                            footerElementID,
			"footer_inner":                      footerInnerElementID,
			"site_name_help_button":             siteNameHelpButtonElementID,
			"allowed_origin_help_button":        allowedOriginHelpButtonElementID,
			"owner_email_help_button":           ownerEmailHelpButtonElementID,
			"site_search_toggle_button":         siteSearchToggleButtonElementID,
			"site_search_container":             siteSearchContainerElementID,
			"site_search_input":                 siteSearchInputElementID,
			"messages_search_toggle_button":     messagesSearchToggleButtonElementID,
			"messages_search_container":         messagesSearchContainerElementID,
			"messages_search_input":             messagesSearchInputElementID,
			"feedback_count":                    feedbackCountElementID,
			"widget_side_left":                  widgetPlacementSideLeftInputElementID,
			"widget_side_right":                 widgetPlacementSideRightInputElementID,
			"widget_bottom_offset":              widgetPlacementBottomOffsetInputElementID,
			"widget_bottom_offset_decrease":     widgetBottomOffsetDecreaseButtonElementID,
			"widget_bottom_offset_increase":     widgetBottomOffsetIncreaseButtonElementID,
			"session_timeout_container":         sessionTimeoutContainerElementID,
			"session_timeout_message":           sessionTimeoutMessageElementID,
			"session_timeout_confirm_button":    sessionTimeoutConfirmButtonElementID,
			"session_timeout_dismiss_button":    sessionTimeoutDismissButtonElementID,
		},
		ButtonClasses: map[string]string{
			"new_site_default":        newSiteButtonClass,
			"new_site_active":         newSiteButtonActiveClass,
			"create":                  siteFormCreateButtonClass,
			"update":                  siteFormUpdateButtonClass,
			"save_default":            dashboardActionButtonSuccessClass,
			"copy_default":            dashboardActionButtonPrimaryClass,
			"refresh_default":         dashboardActionButtonSecondaryClass,
			"delete_site_default":     deleteSiteButtonClass,
			"delete_site_disabled":    deleteSiteButtonDisabledClass,
			"session_timeout_confirm": sessionTimeoutConfirmButtonClass,
			"session_timeout_dismiss": sessionTimeoutDismissButtonClass,
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
			"save_failed":     dashboardErrorMessageSaveFailed,
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
			dashboardRoleAdminValue: dashboardRoleAdminLabel,
			dashboardRoleUserValue:  dashboardRoleUserLabel,
		},
		RoleValues: map[string]string{
			"admin": dashboardRoleAdminValue,
			"user":  dashboardRoleUserValue,
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
		Placeholders: map[string]string{
			"subscribers": subscribersPlaceholder,
			"top_pages":   topPagesPlaceholder,
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
		WidgetPlacement: widgetPlacementClientConfig{
			InputName:           widgetPlacementSideInputName,
			DefaultSide:         defaultWidgetBubbleSide,
			DefaultBottomOffset: defaultWidgetBubbleBottomOffset,
			Sides: map[string]string{
				widgetBubbleSideLeft:  dashboardWidgetPlacementLeftLabel,
				widgetBubbleSideRight: dashboardWidgetPlacementRightLabel,
			},
			BottomOffset: rangeConfig{
				Min: minWidgetBubbleBottomOffset,
				Max: maxWidgetBubbleBottomOffset,
			},
		},
		ValidationMessages: map[string]string{
			validationMessageNameRequiredKey: dashboardValidationNameMessage,
			validationMessageOriginKey:       dashboardValidationOriginMessage,
			validationMessageOwnerKey:        dashboardValidationOwnerMessage,
			validationMessageWidgetOffsetKey: dashboardValidationWidgetOffsetMessage,
		},
		ErrorMessages: map[string]string{
			errorValueSiteExists:          dashboardErrorMessageSiteExists,
			errorValueInvalidOwner:        dashboardErrorMessageInvalidOwner,
			errorValueMissingFields:       dashboardErrorMessageMissingFields,
			errorValueInvalidJSON:         dashboardErrorMessageInvalidJSON,
			errorValueSaveFailed:          dashboardErrorMessageSaveFailed,
			errorValueNotAuthorized:       dashboardErrorMessageNotAuthorized,
			errorValueInvalidWidgetSide:   dashboardValidationWidgetSideMessage,
			errorValueInvalidWidgetOffset: dashboardValidationWidgetOffsetMessage,
			authErrorForbidden:            dashboardErrorMessageNotAuthorized,
		},
		SessionTimeout: sessionTimeoutConfig{
			PromptDelayMilliseconds: sessionTimeoutPromptDelayMilliseconds,
			AutoLogoutMilliseconds:  sessionTimeoutAutoLogoutMilliseconds,
			Texts: map[string]string{
				"prompt":  sessionTimeoutPromptText,
				"confirm": sessionTimeoutConfirmButtonLabel,
				"dismiss": sessionTimeoutDismissButtonLabel,
			},
			ComponentClasses: map[string]string{
				"container":         sessionTimeoutContainerBaseClass,
				"container_visible": sessionTimeoutContainerVisibleClass,
				"container_hidden":  sessionTimeoutContainerHiddenClass,
				"message":           sessionTimeoutMessageClass,
				"actions":           sessionTimeoutActionsClass,
				"inner":             sessionTimeoutInnerClass,
			},
			ThemeClasses: map[string]string{
				"light": sessionTimeoutLightThemeClass,
				"dark":  sessionTimeoutDarkThemeClass,
			},
		},
		AutoLogout: autoLogoutClientConfig{
			StorageKey:        settingsAutoLogoutStorageKey,
			MinPromptSeconds:  autoLogoutMinimumPromptSeconds,
			MaxPromptSeconds:  autoLogoutMaximumPromptSeconds,
			MinLogoutSeconds:  autoLogoutMinimumLogoutSeconds,
			MaxLogoutSeconds:  autoLogoutMaximumLogoutSeconds,
			MinimumGapSeconds: autoLogoutMinimumGapSeconds,
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
