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
) {
	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, landingRouteRoot)
	})
	router.GET(landingRouteRoot, landingHandlers.RenderLandingPage)
	router.GET(httpapi.PrivacyPagePath, privacyHandlers.RenderPrivacyPage)
	router.GET(httpapi.SitemapRoutePath, sitemapHandlers.RenderSitemap)
	router.GET(dashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)
}

func registerBackendRoutes(
	router *gin.Engine,
	authManager *httpapi.AuthManager,
	publicHandlers *httpapi.PublicHandlers,
	siteHandlers *httpapi.SiteHandlers,
	widgetTestHandlers *httpapi.SiteWidgetTestHandlers,
	trafficTestHandlers *httpapi.SiteTrafficTestHandlers,
	subscribeTestHandlers *httpapi.SiteSubscribeTestHandlers,
) {
	router.GET("/app/sites/:id/widget-test", authManager.RequireAuthenticatedWeb(), widgetTestHandlers.RenderWidgetTestPage)
	router.POST("/app/sites/:id/widget-test/feedback", authManager.RequireAuthenticatedJSON(), widgetTestHandlers.SubmitWidgetTestFeedback)
	router.GET("/app/sites/:id/traffic-test", authManager.RequireAuthenticatedWeb(), trafficTestHandlers.RenderTrafficTestPage)
	router.GET("/app/sites/:id/subscribe-test", authManager.RequireAuthenticatedWeb(), subscribeTestHandlers.RenderSubscribeTestPage)
	router.GET("/app/sites/:id/subscribe-test/events", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.StreamSubscriptionTestEvents)
	router.POST("/app/sites/:id/subscribe-test/subscriptions", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.CreateSubscription)

	router.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	router.POST(publicRouteSubscription, publicHandlers.CreateSubscription)
	router.POST(publicRouteSubscriptionConfirm, publicHandlers.ConfirmSubscription)
	router.POST(publicRouteSubscriptionOptOut, publicHandlers.Unsubscribe)
	router.GET(publicRouteSubscriptionConfirmWeb, publicHandlers.ConfirmSubscriptionLink)
	router.GET(publicRouteSubscriptionOptOutWeb, publicHandlers.UnsubscribeSubscriptionLink)
	router.GET(publicRouteWidget, publicHandlers.WidgetJS)
	router.GET(publicRouteSubscribeWidget, publicHandlers.SubscribeJS)
	router.GET(publicRouteSubscribeDemo, publicHandlers.SubscribeDemo)
	router.GET(publicRouteVisitPixel, publicHandlers.CollectVisit)
	router.GET("/pixel.js", publicHandlers.PixelJS)

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
}
