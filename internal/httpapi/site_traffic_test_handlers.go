package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

type SiteTrafficTestHandlers struct {
	database   *gorm.DB
	logger     *zap.Logger
	template   *template.Template
	authConfig AuthClientConfig
}

func NewSiteTrafficTestHandlers(database *gorm.DB, logger *zap.Logger, authConfig AuthClientConfig) *SiteTrafficTestHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	baseTemplate := template.Must(template.New("traffic_test").Parse(dashboardHeaderTemplateHTML))
	compiled := template.Must(baseTemplate.Parse(trafficTestTemplateHTML))
	return &SiteTrafficTestHandlers{
		database:   database,
		logger:     logger,
		template:   compiled,
		authConfig: authConfig,
	}
}

type trafficTestTemplateData struct {
	PageTitle               string
	Header                  dashboardHeaderTemplateData
	LandingPath             string
	BootstrapIconsIntegrity template.HTMLAttr
	FaviconDataURI          template.URL
	TauthScriptURL          template.URL
	SiteID                  string
	SiteName                string
	VisitsEndpoint          template.URL
	StatsEndpoint           template.URL
	DefaultURL              string
	StatusElementID         string
	StatusLogElementID      string
	URLInputID              string
	SendButtonID            string
	RefreshButtonID         string
	TotalCounterID          string
	UniqueCounterID         string
	TopPagesTableID         string
	SharedStyles            template.CSS
	AuthScript              template.JS
	FooterHTML              template.HTML
	FooterElementID         string
	FooterInnerElementID    string
	FooterBaseClass         string
	FooterThemeLightClass   string
	FooterThemeDarkClass    string
	ThemeStorageKey         string
	PublicThemeStorageKey   string
	LandingThemeStorageKey  string
	LegacyThemeStorageKey   string
	DashboardPath           string
}

func (handlers *SiteTrafficTestHandlers) RenderTrafficTestPage(context *gin.Context) {
	siteIdentifier := strings.TrimSpace(context.Param("id"))
	if siteIdentifier == "" {
		context.AbortWithStatus(http.StatusBadRequest)
		return
	}

	currentUser, ok := CurrentUserFromContext(context)
	if !ok {
		context.Redirect(http.StatusFound, LandingPagePath)
		return
	}

	var site model.Site
	if handlers.database == nil || handlers.database.First(&site, "id = ?", siteIdentifier).Error != nil {
		context.AbortWithStatus(http.StatusNotFound)
		return
	}
	if !currentUser.canManageSite(site) {
		context.AbortWithStatus(http.StatusForbidden)
		return
	}

	headerData := dashboardHeaderTemplateData{
		PageTitle:               dashboardPageTitle,
		HeaderLogoDataURI:       landingLogoDataURI,
		HeaderLogoImageID:       dashboardHeaderLogoElementID,
		HeaderGoogleClientID:    handlers.authConfig.GoogleClientID,
		HeaderTauthBaseURL:      handlers.authConfig.TauthBaseURL,
		HeaderTauthTenantID:     handlers.authConfig.TauthTenantID,
		HeaderTauthLoginPath:    TauthLoginPath,
		HeaderTauthLogoutPath:   TauthLogoutPath,
		HeaderTauthNoncePath:    TauthNoncePath,
		HeaderSignInLabel:       publicSignInLabel,
		HeaderSignOutLabel:      publicSignOutLabel,
		SettingsButtonID:        settingsButtonElementID,
		SettingsButtonLabel:     navbarSettingsButtonLabel,
		SettingsModalID:         settingsModalElementID,
		SettingsModalTitleID:    settingsModalTitleElementID,
		SettingsModalTitle:      settingsModalTitle,
		SettingsModalIntro:      settingsModalIntroText,
		SettingsModalCloseLabel: settingsModalCloseButtonLabel,
		SettingsModalContentID:  settingsModalContentElementID,
	}

	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantDashboard)
	if footerErr != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_traffic_test_footer", zap.Error(footerErr))
		}
		footerHTML = template.HTML("")
	}
	authScript, authErr := renderPublicAuthScript()
	if authErr != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_traffic_test_auth_script", zap.Error(authErr))
		}
		authScript = template.JS("")
	}

	statsEndpoint := "/api/sites/" + site.ID + "/visits/stats"
	visitsEndpoint := "/api/visits"
	defaultURL := primaryAllowedOrigin(site.AllowedOrigin)
	if strings.TrimSpace(defaultURL) == "" {
		defaultURL = handlers.defaultSampleURL(context.Request)
	}

	data := trafficTestTemplateData{
		PageTitle:               "Traffic Widget Test — " + site.Name,
		Header:                  headerData,
		LandingPath:             LandingPagePath,
		BootstrapIconsIntegrity: template.HTMLAttr(dashboardBootstrapIconsIntegrityAttr),
		FaviconDataURI:          template.URL(dashboardFaviconDataURI),
		TauthScriptURL:          template.URL(handlers.authConfig.TauthScriptURL),
		SiteID:                  site.ID,
		SiteName:                site.Name,
		VisitsEndpoint:          template.URL(visitsEndpoint),
		StatsEndpoint:           template.URL(statsEndpoint),
		DefaultURL:              defaultURL,
		StatusElementID:         "traffic-test-status",
		StatusLogElementID:      "traffic-test-log",
		URLInputID:              "traffic-test-url",
		SendButtonID:            "traffic-test-send-hit",
		RefreshButtonID:         "traffic-test-refresh-stats",
		TotalCounterID:          "traffic-test-visit-total",
		UniqueCounterID:         "traffic-test-visit-unique",
		TopPagesTableID:         "traffic-test-top-pages",
		SharedStyles:            sharedPublicStyles(),
		AuthScript:              authScript,
		FooterHTML:              footerHTML,
		FooterElementID:         footerElementID,
		FooterInnerElementID:    footerInnerElementID,
		FooterBaseClass:         footerBaseClass,
		FooterThemeLightClass:   footerThemeLightClass,
		FooterThemeDarkClass:    footerThemeDarkClass,
		ThemeStorageKey:         themeStorageKey,
		PublicThemeStorageKey:   publicThemeStorageKey,
		LandingThemeStorageKey:  publicLandingThemeStorageKey,
		LegacyThemeStorageKey:   publicLegacyThemeStorageKey,
		DashboardPath:           publicDashboardPath,
	}

	var buffer bytes.Buffer
	if err := handlers.template.Execute(&buffer, data); err != nil {
		if handlers.logger != nil {
			handlers.logger.Warn("render_traffic_test_page", zap.Error(err))
		}
		context.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}

func (handlers *SiteTrafficTestHandlers) defaultSampleURL(request *http.Request) string {
	if request == nil || request.URL == nil {
		return "https://example.com/"
	}
	base := request.URL.Scheme + "://" + request.Host
	if base == "://" {
		return "https://example.com/"
	}
	return base
}
