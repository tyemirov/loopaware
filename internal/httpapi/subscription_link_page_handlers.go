package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscriptionLinkPageHandlers struct {
	logger     *zap.Logger
	authConfig AuthClientConfig
	apiBaseURL string
}

func NewSubscriptionLinkPageHandlers(logger *zap.Logger, authConfig AuthClientConfig, apiBaseURL string) *SubscriptionLinkPageHandlers {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SubscriptionLinkPageHandlers{
		logger:     logger,
		authConfig: authConfig,
		apiBaseURL: normalizeBaseURL(apiBaseURL),
	}
}

func (handlers *SubscriptionLinkPageHandlers) RenderConfirmSubscriptionLink(context *gin.Context) {
	handlers.renderSubscriptionLinkPage(context, "Subscription confirmation", "Preparing confirmation...", joinBaseURL(handlers.apiBaseURL, "/api/subscriptions/confirm-link"))
}

func (handlers *SubscriptionLinkPageHandlers) RenderUnsubscribeSubscriptionLink(context *gin.Context) {
	handlers.renderSubscriptionLinkPage(context, "Unsubscribe", "Preparing unsubscribe...", joinBaseURL(handlers.apiBaseURL, "/api/subscriptions/unsubscribe-link"))
}

func (handlers *SubscriptionLinkPageHandlers) renderSubscriptionLinkPage(context *gin.Context, heading string, message string, actionEndpoint string) {
	footerHTML, footerErr := renderFooterHTMLForVariant(footerVariantLanding)
	if footerErr != nil {
		handlers.logger.Warn("render_subscription_link_footer", zap.Error(footerErr))
		footerHTML = ""
	}

	headerHTML, headerErr := renderPublicHeader(landingLogoDataURI, false, publicPageLanding, handlers.authConfig, false)
	if headerErr != nil {
		handlers.logger.Warn("render_subscription_link_header", zap.Error(headerErr))
		headerHTML = ""
	}

	themeScript, themeErr := renderPublicThemeScript()
	if themeErr != nil {
		handlers.logger.Warn("render_subscription_link_theme_script", zap.Error(themeErr))
		themeScript = ""
	}
	authScript, authErr := renderPublicAuthScript()
	if authErr != nil {
		handlers.logger.Warn("render_subscription_link_auth_script", zap.Error(authErr))
		authScript = ""
	}

	payload := subscriptionConfirmedTemplateData{
		PageTitle:      heading + " — LoopAware",
		SharedStyles:   sharedPublicStyles(),
		ThemeScript:    themeScript,
		AuthScript:     authScript,
		FaviconDataURI: template.URL(dashboardFaviconDataURI),
		HeaderHTML:     headerHTML,
		FooterHTML:     footerHTML,
		TauthScriptURL: template.URL(handlers.authConfig.TauthScriptURL),
		LandingPath:    LandingPagePath,
		Heading:        heading,
		Message:        message,
		ActionEndpoint: actionEndpoint,
	}

	var buffer bytes.Buffer
	if err := subscriptionConfirmedTemplate.Execute(&buffer, payload); err != nil {
		handlers.logger.Warn("render_subscription_link_page", zap.Error(err))
		context.Data(http.StatusInternalServerError, "text/html; charset=utf-8", []byte(message))
		return
	}
	context.Data(http.StatusOK, "text/html; charset=utf-8", buffer.Bytes())
}
