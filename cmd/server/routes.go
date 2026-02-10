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
	if path == "/api/feedback" || path == "/api/widget-config" || path == "/api/visits" {
		return true
	}
	return strings.HasPrefix(path, "/api/subscriptions")
}

func registerAPIPreflightRoutes(router *gin.Engine, publicCORS gin.HandlerFunc, authenticatedCORS gin.HandlerFunc) {
	router.OPTIONS(apiRoutePrefix+"/*path", func(context *gin.Context) {
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
	})
}

func registerBackendRoutes(
	router *gin.Engine,
	authManager *api.AuthManager,
	publicHandlers *api.PublicHandlers,
	siteHandlers *api.SiteHandlers,
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
	publicGroup.POST(publicRouteSubscription, publicHandlers.CreateSubscription)
	publicGroup.POST(publicRouteSubscriptionConfirm, publicHandlers.ConfirmSubscription)
	publicGroup.POST(publicRouteSubscriptionOptOut, publicHandlers.Unsubscribe)
	publicGroup.GET("/api/widget-config", publicHandlers.WidgetConfig)
	publicGroup.GET("/api/subscriptions/confirm-link", publicHandlers.ConfirmSubscriptionLinkJSON)
	publicGroup.GET("/api/subscriptions/unsubscribe-link", publicHandlers.UnsubscribeSubscriptionLinkJSON)
	publicGroup.GET(publicRouteVisitPixel, publicHandlers.CollectVisit)
	publicGroup.POST(publicRouteVisitPixel, publicHandlers.CollectVisit)

	apiGroup := router.Group(apiRoutePrefix)
	apiGroup.Use(authenticatedCORS)
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
