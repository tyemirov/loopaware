package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
)

func registerFrontendRoutes(
	router *gin.Engine,
	authManager *httpapi.AuthManager,
	landingHandlers *httpapi.LandingPageHandlers,
	privacyHandlers *httpapi.PrivacyPageHandlers,
	sitemapHandlers *httpapi.SitemapHandlers,
	dashboardHandlers *httpapi.DashboardWebHandlers,
	publicJavaScriptHandlers *httpapi.PublicJavaScriptHandlers,
	subscribeDemoHandlers *httpapi.SubscribeDemoPageHandlers,
	subscriptionLinkHandlers *httpapi.SubscriptionLinkPageHandlers,
	widgetTestHandlers *httpapi.SiteWidgetTestHandlers,
	trafficTestHandlers *httpapi.SiteTrafficTestHandlers,
	subscribeTestHandlers *httpapi.SiteSubscribeTestHandlers,
) {
	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, landingRouteRoot)
	})
	router.GET(landingRouteRoot, landingHandlers.RenderLandingPage)
	router.GET(httpapi.PrivacyPagePath, privacyHandlers.RenderPrivacyPage)
	router.GET(httpapi.SitemapRoutePath, sitemapHandlers.RenderSitemap)
	router.GET(dashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)

	router.GET(publicRouteWidget, publicJavaScriptHandlers.WidgetJS)
	router.GET(publicRouteSubscribeWidget, publicJavaScriptHandlers.SubscribeJS)
	router.GET("/pixel.js", publicJavaScriptHandlers.PixelJS)
	router.GET(publicRouteSubscribeDemo, subscribeDemoHandlers.RenderSubscribeDemo)
	router.GET(publicRouteSubscriptionConfirmWeb, subscriptionLinkHandlers.RenderConfirmSubscriptionLink)
	router.GET(publicRouteSubscriptionOptOutWeb, subscriptionLinkHandlers.RenderUnsubscribeSubscriptionLink)

	router.GET("/app/sites/:id/widget-test", authManager.RequireAuthenticatedWeb(), widgetTestHandlers.RenderWidgetTestPage)
	router.GET("/app/sites/:id/traffic-test", authManager.RequireAuthenticatedWeb(), trafficTestHandlers.RenderTrafficTestPage)
	router.GET("/app/sites/:id/subscribe-test", authManager.RequireAuthenticatedWeb(), subscribeTestHandlers.RenderSubscribeTestPage)
}

func registerBackendRoutes(
	router *gin.Engine,
	authManager *httpapi.AuthManager,
	publicHandlers *httpapi.PublicHandlers,
	siteHandlers *httpapi.SiteHandlers,
	widgetTestHandlers *httpapi.SiteWidgetTestHandlers,
	subscribeTestHandlers *httpapi.SiteSubscribeTestHandlers,
) {
	router.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	router.POST(publicRouteSubscription, publicHandlers.CreateSubscription)
	router.POST(publicRouteSubscriptionConfirm, publicHandlers.ConfirmSubscription)
	router.POST(publicRouteSubscriptionOptOut, publicHandlers.Unsubscribe)
	router.GET("/api/widget-config", publicHandlers.WidgetConfig)
	router.GET("/api/subscriptions/confirm-link", publicHandlers.ConfirmSubscriptionLinkJSON)
	router.GET("/api/subscriptions/unsubscribe-link", publicHandlers.UnsubscribeSubscriptionLinkJSON)
	router.GET(publicRouteVisitPixel, publicHandlers.CollectVisit)

	apiGroup := router.Group(apiRoutePrefix)
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET(apiRouteMe, siteHandlers.CurrentUser)
	apiGroup.GET(apiRouteMeAvatar, siteHandlers.UserAvatar)
	apiGroup.GET(apiRouteSites, siteHandlers.ListSites)
	apiGroup.POST(apiRouteSites, siteHandlers.CreateSite)
	apiGroup.PATCH(apiRouteSiteUpdate, siteHandlers.UpdateSite)
	apiGroup.DELETE(apiRouteSiteUpdate, siteHandlers.DeleteSite)
	apiGroup.GET(apiRouteSiteMessages, siteHandlers.ListMessagesBySite)
	apiGroup.GET(apiRouteSiteSubscribers, siteHandlers.ListSubscribers)
	apiGroup.GET(apiRouteSiteSubscribersExport, siteHandlers.ExportSubscribers)
	apiGroup.PATCH(apiRouteSiteSubscriberUpdate, siteHandlers.UpdateSubscriberStatus)
	apiGroup.DELETE(apiRouteSiteSubscriberUpdate, siteHandlers.DeleteSubscriber)
	apiGroup.GET(apiRouteSiteFavicon, siteHandlers.SiteFavicon)
	apiGroup.GET(apiRouteSiteFaviconEvents, siteHandlers.StreamFaviconUpdates)
	apiGroup.GET(apiRouteSiteFeedbackEvents, siteHandlers.StreamFeedbackUpdates)
	apiGroup.GET(apiRouteSiteVisitStats, siteHandlers.VisitStats)

	apiGroup.POST("/sites/:id/widget-test/feedback", widgetTestHandlers.SubmitWidgetTestFeedback)
	apiGroup.GET("/sites/:id/subscribe-test/events", subscribeTestHandlers.StreamSubscriptionTestEvents)
	apiGroup.POST("/sites/:id/subscribe-test/subscriptions", subscribeTestHandlers.CreateSubscription)
}
