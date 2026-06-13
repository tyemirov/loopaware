package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
)

func isPublicAPIPath(path string) bool {
	if path == "" {
		return false
	}
	if path == publicRouteFeedback || path == publicRouteMobileFeedback || path == "/public/widget-config" || path == publicRouteVisitPixel {
		return true
	}
	return strings.HasPrefix(path, publicRouteSubscription)
}

func registerAPIPreflightRoutes(router *gin.Engine, publicCORS gin.HandlerFunc, authenticatedCORS gin.HandlerFunc) {
	preflightHandler := func(context *gin.Context) {
		requestPath := context.Request.URL.Path
		if isPublicAPIPath(requestPath) {
			publicCORS(context)
		} else {
			authenticatedCORS(context)
		}
		if context.IsAborted() {
			return
		}
		context.Status(http.StatusNoContent)
	}

	router.OPTIONS(apiRoutePrefix+"/*path", preflightHandler)
	router.OPTIONS(publicRoutePrefix+"/*path", preflightHandler)
	router.OPTIONS(sentryRouteBrowserErrors, func(context *gin.Context) {
		publicCORS(context)
		if context.IsAborted() {
			return
		}
		context.Status(http.StatusNoContent)
	})
}

func registerBackendRoutes(
	router *gin.Engine,
	authManager *api.AuthManager,
	publicHandlers *api.PublicHandlers,
	siteHandlers *api.SiteHandlers,
	trafficReportHandlers *api.TrafficReportHandlers,
	sentryHandlers *api.SentryHandlers,
	widgetTestHandlers *api.SiteWidgetTestHandlers,
	subscribeTestHandlers *api.SiteSubscribeTestHandlers,
	authenticatedOrigin string,
) {
	publicCORS := cors.New(cors.Config{
		AllowOrigins:     []string{corsOriginWildcard},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
	authenticatedCORS := cors.New(cors.Config{
		AllowOrigins:     []string{authenticatedOrigin},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})

	registerAPIPreflightRoutes(router, publicCORS, authenticatedCORS)

	publicGroup := router.Group("/")
	publicGroup.Use(publicCORS)
	publicGroup.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	publicGroup.POST(publicRouteMobileFeedback, publicHandlers.CreateMobileFeedback)
	publicGroup.POST(publicRouteSubscription, publicHandlers.CreateSubscription)
	publicGroup.POST(publicRouteSubscriptionStatus, publicHandlers.SubscriptionStatus)
	publicGroup.POST(publicRouteSubscriptionConfirm, publicHandlers.ConfirmSubscription)
	publicGroup.POST(publicRouteSubscriptionOptOut, publicHandlers.Unsubscribe)
	publicGroup.GET("/public/widget-config", publicHandlers.WidgetConfig)
	publicGroup.GET("/public/subscriptions/confirm-link", publicHandlers.ConfirmSubscriptionLinkJSON)
	publicGroup.GET("/public/subscriptions/unsubscribe-link", publicHandlers.UnsubscribeSubscriptionLinkJSON)
	publicGroup.GET(publicRouteVisitPixel, publicHandlers.CollectVisit)
	publicGroup.POST(publicRouteVisitPixel, publicHandlers.CollectVisit)
	publicGroup.POST(sentryRouteBrowserErrors, sentryHandlers.CaptureBrowserError)

	router.POST(sentryRouteErrors, sentryHandlers.CaptureError)

	apiGroup := router.Group(apiRoutePrefix)
	apiGroup.Use(authenticatedCORS)
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET(apiRouteMe, siteHandlers.CurrentUser)
	apiGroup.GET(apiRouteMeAvatar, siteHandlers.UserAvatar)
	apiGroup.GET(apiRouteSites, siteHandlers.ListSites)
	apiGroup.POST(apiRouteSites, siteHandlers.CreateSite)
	apiGroup.PATCH(apiRouteSiteUpdate, siteHandlers.UpdateSite)
	apiGroup.DELETE(apiRouteSiteUpdate, siteHandlers.DeleteSite)
	apiGroup.GET(apiRouteSiteTeamMembers, siteHandlers.ListTeamMembers)
	apiGroup.POST(apiRouteSiteTeamMembers, siteHandlers.CreateTeamMember)
	apiGroup.DELETE(apiRouteSiteTeamMember, siteHandlers.DeleteTeamMember)
	apiGroup.GET(apiRouteSiteMobileApps, siteHandlers.ListMobileApps)
	apiGroup.POST(apiRouteSiteMobileApps, siteHandlers.CreateMobileApp)
	apiGroup.GET(apiRouteSiteMessages, siteHandlers.ListMessagesBySite)
	apiGroup.GET(apiRouteSiteSubscribers, siteHandlers.ListSubscribers)
	apiGroup.GET(apiRouteSiteSubscribersExport, siteHandlers.ExportSubscribers)
	apiGroup.PATCH(apiRouteSiteSubscriberUpdate, siteHandlers.UpdateSubscriberStatus)
	apiGroup.DELETE(apiRouteSiteSubscriberUpdate, siteHandlers.DeleteSubscriber)
	apiGroup.GET(apiRouteSiteFavicon, siteHandlers.SiteFavicon)
	apiGroup.GET(apiRouteSiteFaviconEvents, siteHandlers.StreamFaviconUpdates)
	apiGroup.GET(apiRouteSiteFeedbackEvents, siteHandlers.StreamFeedbackUpdates)
	apiGroup.GET(apiRouteSiteVisitStats, siteHandlers.VisitStats)
	apiGroup.GET(apiRouteSiteVisitTrend, siteHandlers.VisitTrend)
	apiGroup.GET(apiRouteSiteVisitAttribution, siteHandlers.VisitAttribution)
	apiGroup.GET(apiRouteSiteVisitEngagement, siteHandlers.VisitEngagement)
	apiGroup.GET(apiRouteSiteVisitDevices, siteHandlers.DeviceBreakdown)
	apiGroup.GET(apiRouteSiteVisitLocations, siteHandlers.LocationDistribution)
	apiGroup.GET(apiRouteSiteVisitExport, siteHandlers.ExportTraffic)
	apiGroup.GET(apiRouteSiteTrafficReportSchedule, trafficReportHandlers.GetSchedule)
	apiGroup.PUT(apiRouteSiteTrafficReportSchedule, trafficReportHandlers.SaveSchedule)
	apiGroup.POST(apiRouteSiteTrafficReportTest, trafficReportHandlers.SendTestReport)
	apiGroup.GET(apiRoutePortfolioTrafficReport, trafficReportHandlers.GetPortfolioReport)
	apiGroup.GET(apiRoutePortfolioTrafficReports, trafficReportHandlers.ListPortfolioReports)
	apiGroup.POST(apiRoutePortfolioTrafficReports, trafficReportHandlers.CreatePortfolioReport)
	apiGroup.PUT(apiRoutePortfolioTrafficReportDef, trafficReportHandlers.UpdatePortfolioReport)
	apiGroup.GET(apiRoutePortfolioTrafficSchedule, trafficReportHandlers.GetPortfolioSchedule)
	apiGroup.PUT(apiRoutePortfolioTrafficSchedule, trafficReportHandlers.SavePortfolioSchedule)
	apiGroup.POST(apiRoutePortfolioTrafficTest, trafficReportHandlers.SendPortfolioTestReport)
	apiGroup.GET(apiRouteSiteSentryIssues, sentryHandlers.ListIssues)
	apiGroup.GET(apiRouteSiteSentryIssueDetail, sentryHandlers.IssueDetail)
	apiGroup.PATCH(apiRouteSiteSentryIssueDetail, sentryHandlers.UpdateIssueStatus)
	apiGroup.POST(apiRouteSiteSentryToken, sentryHandlers.RotateToken)

	apiGroup.POST("/sites/:id/widget-test/feedback", widgetTestHandlers.SubmitWidgetTestFeedback)
	apiGroup.GET("/sites/:id/subscribe-test/events", subscribeTestHandlers.StreamSubscriptionTestEvents)
	apiGroup.POST("/sites/:id/subscribe-test/subscriptions", subscribeTestHandlers.CreateSubscription)
}
