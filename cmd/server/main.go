package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/api"
	"github.com/MarkoPoloResearchLab/loopaware/internal/notifications"
	"github.com/MarkoPoloResearchLab/loopaware/internal/serverconfig"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	commandUseName                    = "server"
	commandShortDescription           = "Run the feedback server"
	commandLongDescription            = "Launch the feedback collection HTTP server"
	loggerCreationErrorMessage        = "logger"
	logEventListening                 = "listening"
	logFieldAddress                   = "addr"
	flagNameConfigFile                = "config"
	flagUsageConfigFile               = "path to configuration file"
	publicRoutePrefix                 = "/public"
	publicRouteFeedback               = "/public/feedback"
	publicRouteMobileFeedback         = "/public/mobile-feedback"
	publicRouteSubscription           = "/public/subscriptions"
	publicRouteVisitPixel             = "/public/visits"
	sentryRouteErrors                 = "/sentry/errors"
	sentryRouteBrowserErrors          = "/sentry/browser-errors"
	apiRoutePrefix                    = "/api"
	apiRouteMe                        = "/me"
	apiRouteMeAvatar                  = "/me/avatar"
	apiRouteSites                     = "/sites"
	apiRouteSiteUpdate                = "/sites/:id"
	apiRouteSiteTeamMembers           = "/sites/:id/team"
	apiRouteSiteTeamMember            = "/sites/:id/team/:member_id"
	apiRouteSiteMobileApps            = "/sites/:id/mobile-apps"
	apiRouteSiteMessages              = "/sites/:id/messages"
	apiRouteSiteVisitStats            = "/sites/:id/visits/stats"
	apiRouteSiteVisitTrend            = "/sites/:id/visits/trend"
	apiRouteSiteVisitAttribution      = "/sites/:id/visits/attribution"
	apiRouteSiteVisitEngagement       = "/sites/:id/visits/engagement"
	apiRouteSiteVisitDevices          = "/sites/:id/visits/devices"
	apiRouteSiteVisitLocations        = "/sites/:id/visits/locations"
	apiRouteSiteVisitExport           = "/sites/:id/visits/export"
	apiRouteSiteTrafficReportSchedule = "/sites/:id/traffic-report-schedule"
	apiRouteSiteTrafficReportTest     = "/sites/:id/traffic-report-schedule/test"
	apiRouteSiteHealthMonitor         = "/sites/:id/health-monitor"
	apiRouteSiteHealthMonitorCheck    = "/sites/:id/health-monitor/check"
	apiRoutePortfolioTrafficReport    = "/reports/traffic/portfolio"
	apiRoutePortfolioTrafficReports   = "/reports/traffic/portfolio/reports"
	apiRoutePortfolioTrafficReportDef = "/reports/traffic/portfolio/reports/:report_id"
	apiRoutePortfolioTrafficSchedule  = "/reports/traffic/portfolio/schedule"
	apiRoutePortfolioTrafficTest      = "/reports/traffic/portfolio/schedule/test"
	healthRoute                       = "/healthz"
	apiRouteSiteSubscribers           = "/sites/:id/subscribers"
	apiRouteSiteSubscriberUpdate      = "/sites/:id/subscribers/:subscriber_id"
	apiRouteSiteSubscribersExport     = "/sites/:id/subscribers/export"
	apiRouteSiteFavicon               = "/sites/:id/favicon"
	apiRouteSiteFaviconEvents         = "/sites/favicons/events"
	apiRouteSiteFeedbackEvents        = "/sites/feedback/events"
	apiRouteSiteSentryIssues          = "/sites/:id/sentry/issues"
	apiRouteSiteSentryIssueDetail     = "/sites/:id/sentry/issues/:issue_id"
	apiRouteSiteSentryToken           = "/sites/:id/sentry/token"
	corsOriginWildcard                = "*"
	corsHeaderAuthorization           = "Authorization"
	corsHeaderContentType             = "Content-Type"
	corsHeaderXTAuthTenant            = "X-TAuth-Tenant"
	httpMethodGet                     = "GET"
	httpMethodOptions                 = "OPTIONS"
	httpMethodPost                    = "POST"
	httpMethodPatch                   = "PATCH"
	httpMethodPut                     = "PUT"
	httpMethodDelete                  = "DELETE"
	loggerContextOpenDatabase         = "open_db"
	loggerContextAutoMigrate          = "migrate"
	loggerContextServer               = "server"
	loggerContextAuthService          = "auth_service"
	readHeaderTimeoutSeconds          = 5
	readTimeoutSeconds                = 10
	idleTimeoutSeconds                = 60
	maximumHeaderBytes                = 64 * 1024
	standardRequestBodyBytes          = 64 * 1024
	sentryRequestBodyBytes            = 1024 * 1024
	unexpectedArgumentsMessage        = "unexpected command arguments"
	commandInitializationFailure      = "failed to configure command"
	configurationFileLoadError        = "failed to load configuration file"
	logMessageMissingAdministrators   = "running without administrators"
	trustedProxyConfigurationFailure  = "failed to configure trusted proxies"
)

var (
	corsAllowedMethods = []string{httpMethodPost, httpMethodGet, httpMethodOptions, httpMethodPatch, httpMethodPut, httpMethodDelete}
	corsAllowedHeaders = []string{corsHeaderAuthorization, corsHeaderContentType, corsHeaderXTAuthTenant}
	corsExposedHeaders = []string{corsHeaderContentType}
)

// ServerConfig captures configuration needed to run the server.
type ServerConfig = serverconfig.Config

// DatabaseOpener opens a database connection using the provided configuration.
type DatabaseOpener func(storage.Config) (*gorm.DB, error)

// ServerRunner executes the HTTP server.
type ServerRunner func(*http.Server) error

// ServerApplication constructs and executes the server command.
type ServerApplication struct {
	databaseOpener DatabaseOpener
	serverRunner   ServerRunner
	pinguinDialer  func(context.Context, string) (net.Conn, error)
}

// NewServerApplication creates a ServerApplication with default dependencies.
func NewServerApplication() *ServerApplication {
	return &ServerApplication{
		databaseOpener: storage.OpenDatabase,
		serverRunner: func(server *http.Server) error {
			return server.ListenAndServe()
		},
	}
}

// WithDatabaseOpener overrides the database opener dependency.
func (application *ServerApplication) WithDatabaseOpener(databaseOpener DatabaseOpener) *ServerApplication {
	application.databaseOpener = databaseOpener
	return application
}

// WithServerRunner overrides the HTTP server runner dependency.
func (application *ServerApplication) WithServerRunner(serverRunner ServerRunner) *ServerApplication {
	application.serverRunner = serverRunner
	return application
}

// WithPinguinDialer overrides the Pinguin gRPC dialer dependency.
func (application *ServerApplication) WithPinguinDialer(dialer func(context.Context, string) (net.Conn, error)) *ServerApplication {
	application.pinguinDialer = dialer
	return application
}

// Command builds the Cobra command for the server.
func (application *ServerApplication) Command() (*cobra.Command, error) {
	rootCommand := &cobra.Command{
		Use:   commandUseName,
		Short: commandShortDescription,
		Long:  commandLongDescription,
		RunE:  application.runCommand,
	}

	if configurationErr := application.configureCommand(rootCommand); configurationErr != nil {
		return nil, configurationErr
	}

	return rootCommand, nil
}

func (application *ServerApplication) configureCommand(command *cobra.Command) error {
	commandFlags := command.Flags()
	commandFlags.String(flagNameConfigFile, serverconfig.DefaultPath, flagUsageConfigFile)
	return nil
}

func (application *ServerApplication) runCommand(command *cobra.Command, arguments []string) error {
	if len(arguments) > 0 {
		return fmt.Errorf("%s: %s", unexpectedArgumentsMessage, strings.Join(arguments, " "))
	}

	configFilePath := strings.TrimSpace(command.Flag(flagNameConfigFile).Value.String())
	serverConfig, serverConfigErr := application.loadServerConfig(configFilePath)
	if serverConfigErr != nil {
		return serverConfigErr
	}

	logger, loggerErr := zap.NewProduction()
	if loggerErr != nil {
		return fmt.Errorf("%s: %w", loggerCreationErrorMessage, loggerErr)
	}
	defer func() {
		_ = logger.Sync()
	}()

	application.logAdministratorWarning(logger, serverConfig)

	router := gin.New()
	trustedProxyCIDRs := make([]string, 0, len(serverConfig.TrustedProxyCIDRs))
	for _, trustedProxyCIDR := range serverConfig.TrustedProxyCIDRs {
		trustedProxyCIDRs = append(trustedProxyCIDRs, trustedProxyCIDR.String())
	}
	if trustedProxyErr := router.SetTrustedProxies(trustedProxyCIDRs); trustedProxyErr != nil {
		return fmt.Errorf("%s: %w", trustedProxyConfigurationFailure, trustedProxyErr)
	}
	router.RemoteIPHeaders = []string{"X-Forwarded-For"}
	router.Use(gin.Recovery())
	router.Use(api.TrustedProxyHeaders(serverConfig.TrustedProxyCIDRs, serverConfig.TrustedEdgeGeoProxyCIDRs))
	router.Use(api.SecurityHeaders())
	router.Use(api.RequestLogger(logger))
	router.Use(api.RequestBodyLimit(requestBodyLimit))

	sharedHTTPClient := &http.Client{Timeout: 5 * time.Second}
	database, databaseErr := application.databaseOpener(storage.Config{
		DriverName:     serverConfig.DatabaseDriverName,
		DataSourceName: serverConfig.DatabaseDataSourceName,
	})
	if databaseErr != nil {
		logger.Fatal(loggerContextOpenDatabase, zap.Error(databaseErr))
	}

	if migrateErr := storage.AutoMigrate(database); migrateErr != nil {
		logger.Fatal(loggerContextAutoMigrate, zap.Error(migrateErr))
	}

	authManager, authManagerErr := api.NewAuthManager(database, logger, serverConfig.AdminEmailAddresses, sharedHTTPClient, api.AuthConfig{
		SigningKey: serverConfig.TauthSigningKey,
		CookieName: serverConfig.TauthSessionCookieName,
		TenantID:   serverConfig.TauthTenantID,
	})
	if authManagerErr != nil {
		logger.Fatal(loggerContextAuthService, zap.Error(authManagerErr))
	}

	feedbackBroadcaster := api.NewFeedbackEventBroadcaster()
	defer feedbackBroadcaster.Close()
	subscriptionEvents := api.NewSubscriptionTestEventBroadcaster()
	defer subscriptionEvents.Close()
	pinguinNotifier, notifierErr := notifications.NewPinguinNotifier(logger, notifications.PinguinConfig{
		Address:           serverConfig.PinguinAddress,
		AuthToken:         serverConfig.PinguinAuthToken,
		TenantID:          serverConfig.PinguinTenantID,
		ConnectionTimeout: time.Duration(serverConfig.PinguinConnTimeoutSec) * time.Second,
		OperationTimeout:  time.Duration(serverConfig.PinguinOpTimeoutSec) * time.Second,
		Dialer:            application.pinguinDialer,
	})
	if notifierErr != nil {
		logger.Fatal("pinguin_notifier", zap.Error(notifierErr))
	}
	defer pinguinNotifier.Close()
	var subscriptionNotifier api.SubscriptionNotifier
	if serverConfig.SubscriptionNotifications {
		subscriptionNotifier = pinguinNotifier
	}
	var trafficReportEmailSender api.EmailSender
	if serverConfig.TrafficReportEmails {
		trafficReportEmailSender = pinguinNotifier
	}
	publicHandlers := api.NewPublicHandlers(database, logger, feedbackBroadcaster, subscriptionEvents, pinguinNotifier, subscriptionNotifier, serverConfig.SubscriptionNotifications, serverConfig.PublicBaseURL, serverConfig.SessionSecret, pinguinNotifier)
	faviconResolver := favicon.NewHTTPResolver(sharedHTTPClient, logger)
	faviconService := favicon.NewService(faviconResolver)
	faviconManager := api.NewSiteFaviconManager(database, faviconService, logger)
	faviconManagerContext, faviconManagerCancel := context.WithCancel(context.Background())
	defer faviconManager.Stop()
	defer faviconManagerCancel()
	faviconManager.Start(faviconManagerContext)
	faviconManager.TriggerScheduledRefresh()
	statsProvider := api.NewDatabaseSiteStatisticsProvider(database)
	siteHandlers := api.NewSiteHandlers(database, logger, serverConfig.PublicBaseURL, faviconManager, statsProvider, feedbackBroadcaster)
	trafficReportHandlers := api.NewTrafficReportHandlers(database, logger, statsProvider, trafficReportEmailSender, serverConfig.TrafficReportEmails)
	siteHealthProber := api.NewHTTPHealthProber(nil)
	siteHealthManager := api.NewSiteHealthManager(database, logger, siteHealthProber, pinguinNotifier, true)
	siteHealthManagerContext, siteHealthManagerCancel := context.WithCancel(context.Background())
	defer siteHealthManager.Stop()
	defer siteHealthManagerCancel()
	siteHealthManager.Start(siteHealthManagerContext)
	siteHealthHandlers := api.NewSiteHealthHandlers(database, logger, siteHealthManager, true)
	sentryHandlers := api.NewSentryHandlers(database, logger, pinguinNotifier, serverConfig.PublicBaseURL)
	if serverConfig.TrafficReportEmails {
		trafficReportScheduler, schedulerErr := api.NewTrafficReportScheduler(database, logger, statsProvider, trafficReportEmailSender, 0, 0)
		if schedulerErr != nil {
			logger.Fatal("traffic_report_scheduler", zap.Error(schedulerErr))
		}
		trafficReportSchedulerContext, trafficReportSchedulerCancel := context.WithCancel(context.Background())
		defer trafficReportScheduler.Stop()
		defer trafficReportSchedulerCancel()
		trafficReportScheduler.Start(trafficReportSchedulerContext)
	}
	widgetTestHandlers := api.NewSiteWidgetTestHandlers(database, logger, feedbackBroadcaster, pinguinNotifier)
	subscribeTestHandlers := api.NewSiteSubscribeTestHandlers(database, logger, subscriptionEvents, subscriptionNotifier, serverConfig.SubscriptionNotifications, serverConfig.PublicBaseURL, serverConfig.SessionSecret, pinguinNotifier)
	authenticatedOrigin, originErr := resolveOrigin(serverConfig.PublicBaseURL)
	if originErr != nil {
		logger.Fatal("cors_origin", zap.Error(originErr))
	}
	registerBackendRoutes(router, authManager, publicHandlers, siteHandlers, trafficReportHandlers, siteHealthHandlers, sentryHandlers, widgetTestHandlers, subscribeTestHandlers, authenticatedOrigin)

	httpServer := &http.Server{
		Addr:              serverConfig.ApplicationAddress,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeoutSeconds * time.Second,
		ReadTimeout:       readTimeoutSeconds * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       idleTimeoutSeconds * time.Second,
		MaxHeaderBytes:    maximumHeaderBytes,
	}

	logger.Info(logEventListening, zap.String(logFieldAddress, serverConfig.ApplicationAddress))
	if serveErr := application.serverRunner(httpServer); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		logger.Fatal(loggerContextServer, zap.Error(serveErr))
	}

	return nil
}

func requestBodyLimit(request *http.Request) int64 {
	if request.Method == http.MethodPost && request.URL.Path == sentryRouteErrors {
		return sentryRequestBodyBytes
	}
	if request.URL.Path == publicRouteVisitPixel {
		return 0
	}
	switch request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return standardRequestBodyBytes
	default:
		return 0
	}
}

func resolveOrigin(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", errors.New("missing base url")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base url: %s", trimmed)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func (application *ServerApplication) loadServerConfig(configFilePath string) (ServerConfig, error) {
	serverConfig, loadError := serverconfig.Load(configFilePath)
	if loadError != nil {
		return ServerConfig{}, fmt.Errorf("%s: %w", configurationFileLoadError, loadError)
	}
	return serverConfig, nil
}

func (application *ServerApplication) logAdministratorWarning(logger *zap.Logger, configuration ServerConfig) {
	if len(configuration.AdminEmailAddresses) > 0 {
		return
	}

	logger.Warn(logMessageMissingAdministrators)
}

func main() {
	application := NewServerApplication()
	rootCommand, commandErr := application.Command()
	if commandErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", commandInitializationFailure, commandErr)
		os.Exit(1)
	}

	if executeErr := rootCommand.Execute(); executeErr != nil {
		os.Exit(1)
	}
}
