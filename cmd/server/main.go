package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/gauss"
	"github.com/temirov/GAuss/pkg/session"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/notifications"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	commandUseName                    = "server"
	commandShortDescription           = "Run the feedback server"
	commandLongDescription            = "Launch the feedback collection HTTP server"
	missingConfigurationMessage       = "missing required configuration"
	loggerCreationErrorMessage        = "logger"
	logEventListening                 = "listening"
	logFieldAddress                   = "addr"
	flagNameConfigFile                = "config"
	flagNameApplicationAddress        = "app-addr"
	flagNameDatabaseDriver            = "db-driver"
	flagNameDatabaseDataSourceName    = "db-dsn"
	flagNameGoogleClientID            = "google-client-id"
	flagNameGoogleClientSecret        = "google-client-secret"
	flagNameSessionSecret             = "session-secret"
	flagNameSubscriptionNotifications = "subscription-notifications"
	flagNamePublicBaseURL             = "public-base-url"
	flagNamePinguinAddress            = "pinguin-addr"
	flagNamePinguinAuthToken          = "pinguin-auth-token"
	flagNamePinguinConnectionTimeout  = "pinguin-conn-timeout"
	flagNamePinguinOperationTimeout   = "pinguin-op-timeout"
	flagUsageConfigFile               = "path to configuration file"
	flagUsageApplicationAddress       = "address for the HTTP server to listen on"
	flagUsageDatabaseDriver           = "database driver (e.g. sqlite)"
	flagUsageDatabaseDataSourceName   = "database connection string"
	flagUsageGoogleClientID           = "Google OAuth client ID"
	flagUsageGoogleClientSecret       = "Google OAuth client secret"
	flagUsageSessionSecret            = "session secret for browser sessions"
	flagUsagePublicBaseURL            = "public base URL for OAuth callbacks"
	flagUsagePinguinAddress           = "Pinguin gRPC server address"
	flagUsagePinguinAuthToken         = "Pinguin bearer auth token"
	flagUsagePinguinConnTimeout       = "Pinguin connection timeout in seconds"
	flagUsagePinguinOpTimeout         = "Pinguin operation timeout in seconds"
	flagUsageSubscriptionNotify       = "enable notifications for new subscriptions"
	environmentKeyApplicationAddress  = "APP_ADDR"
	environmentKeyDatabaseDriverName  = "DB_DRIVER"
	environmentKeyDatabaseDataSource  = "DB_DSN"
	environmentKeyAdmins              = "ADMINS"
	environmentKeyGoogleClientID      = "GOOGLE_CLIENT_ID"
	environmentKeyGoogleClientSecret  = "GOOGLE_CLIENT_SECRET"
	environmentKeySessionSecret       = "SESSION_SECRET"
	environmentKeyPublicBaseURL       = "PUBLIC_BASE_URL"
	environmentKeyPinguinAddress      = "PINGUIN_ADDR"
	environmentKeyPinguinAuthToken    = "PINGUIN_AUTH_TOKEN"
	environmentKeyPinguinSharedAuth   = "GRPC_AUTH_TOKEN"
	environmentKeyPinguinConnTimeout  = "PINGUIN_CONNECTION_TIMEOUT_SEC"
	environmentKeyPinguinOpTimeout    = "PINGUIN_OPERATION_TIMEOUT_SEC"
	environmentKeySubscriptionNotify  = "SUBSCRIPTION_NOTIFICATIONS"
	configurationKeyAdmins            = "admins"
	defaultApplicationAddress         = ":8080"
	sqliteFileDataSourceNamePattern   = "file:%s?_foreign_keys=on"
	defaultSQLiteDatabaseFileName     = "loopaware.sqlite"
	defaultConfigFileName             = "config.yaml"
	defaultPublicBaseURL              = "http://localhost:8080"
	defaultPinguinAddress             = "localhost:50051"
	defaultPinguinConnTimeoutSeconds  = 5
	defaultPinguinOpTimeoutSeconds    = 30
	defaultSubscriptionNotify         = true
	publicRouteFeedback               = "/api/feedback"
	publicRouteSubscription           = "/api/subscriptions"
	publicRouteSubscriptionConfirm    = "/api/subscriptions/confirm"
	publicRouteSubscriptionOptOut     = "/api/subscriptions/unsubscribe"
	publicRouteSubscribeWidget        = "/subscribe.js"
	publicRouteSubscribeDemo          = "/subscribe-demo"
	publicRouteVisitPixel             = "/api/visits"
	publicRouteWidget                 = "/widget.js"
	landingRouteRoot                  = constants.LoginPath
	dashboardRoute                    = "/app"
	apiRoutePrefix                    = "/api"
	apiRouteMe                        = "/me"
	apiRouteMeAvatar                  = "/me/avatar"
	apiRouteSites                     = "/sites"
	apiRouteSiteUpdate                = "/sites/:id"
	apiRouteSiteMessages              = "/sites/:id/messages"
	apiRouteSiteVisitStats            = "/sites/:id/visits/stats"
	apiRouteSiteSubscribers           = "/sites/:id/subscribers"
	apiRouteSiteSubscriberUpdate      = "/sites/:id/subscribers/:subscriber_id"
	apiRouteSiteSubscribersExport     = "/sites/:id/subscribers/export"
	apiRouteSiteFavicon               = "/sites/:id/favicon"
	apiRouteSiteFaviconEvents         = "/sites/favicons/events"
	apiRouteSiteFeedbackEvents        = "/sites/feedback/events"
	corsOriginWildcard                = "*"
	corsHeaderAuthorization           = "Authorization"
	corsHeaderContentType             = "Content-Type"
	httpMethodGet                     = "GET"
	httpMethodOptions                 = "OPTIONS"
	httpMethodPost                    = "POST"
	httpMethodPatch                   = "PATCH"
	httpMethodDelete                  = "DELETE"
	loggerContextOpenDatabase         = "open_db"
	loggerContextAutoMigrate          = "migrate"
	loggerContextServer               = "server"
	loggerContextAuthService          = "auth_service"
	readHeaderTimeoutSeconds          = 5
	unexpectedArgumentsMessage        = "unexpected command arguments"
	commandInitializationFailure      = "failed to configure command"
	flagNotDefinedMessage             = "flag %s not defined"
	environmentConfigurationError     = "failed to apply environment configuration"
	configurationFileLoadError        = "failed to load configuration file"
	administratorEmailSeparator       = ","
	logMessageMissingAdministrators   = "running without administrators"
)

var (
	corsAllowedMethods          = []string{httpMethodPost, httpMethodGet, httpMethodOptions, httpMethodPatch, httpMethodDelete}
	corsAllowedHeaders          = []string{corsHeaderAuthorization, corsHeaderContentType}
	corsExposedHeaders          = []string{corsHeaderContentType}
	corsAllowOrigins            = []string{corsOriginWildcard}
	defaultDatabaseDriverName   = storage.DriverNameSQLite
	defaultSQLiteDataSourceName = fmt.Sprintf(sqliteFileDataSourceNamePattern, defaultSQLiteDatabaseFileName)
)

// ServerConfig captures configuration needed to run the server.
type ServerConfig struct {
	ApplicationAddress        string
	DatabaseDriverName        string
	DatabaseDataSourceName    string
	AdminEmailAddresses       []string
	GoogleClientID            string
	GoogleClientSecret        string
	SessionSecret             string
	PublicBaseURL             string
	ConfigFilePath            string
	PinguinAddress            string
	PinguinAuthToken          string
	PinguinConnTimeoutSec     int
	PinguinOpTimeoutSec       int
	SubscriptionNotifications bool
}

// DatabaseOpener opens a database connection using the provided configuration.
type DatabaseOpener func(storage.Config) (*gorm.DB, error)

// ServerApplication constructs and executes the server command.
type ServerApplication struct {
	configurationLoader *viper.Viper
	databaseOpener      DatabaseOpener
}

// NewServerApplication creates a ServerApplication with default dependencies.
func NewServerApplication() *ServerApplication {
	return &ServerApplication{
		configurationLoader: viper.New(),
		databaseOpener:      storage.OpenDatabase,
	}
}

// WithDatabaseOpener overrides the database opener dependency.
func (application *ServerApplication) WithDatabaseOpener(databaseOpener DatabaseOpener) *ServerApplication {
	application.databaseOpener = databaseOpener
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
	application.configurationLoader.SetDefault(environmentKeyApplicationAddress, defaultApplicationAddress)
	application.configurationLoader.SetDefault(environmentKeyDatabaseDriverName, defaultDatabaseDriverName)
	application.configurationLoader.SetDefault(environmentKeyDatabaseDataSource, defaultSQLiteDataSourceName)
	application.configurationLoader.SetDefault(environmentKeyPublicBaseURL, defaultPublicBaseURL)
	application.configurationLoader.SetDefault(environmentKeyGoogleClientID, "")
	application.configurationLoader.SetDefault(environmentKeyGoogleClientSecret, "")
	application.configurationLoader.SetDefault(environmentKeySessionSecret, "")
	application.configurationLoader.SetDefault(environmentKeyPinguinAddress, defaultPinguinAddress)
	application.configurationLoader.SetDefault(environmentKeyPinguinAuthToken, "")
	application.configurationLoader.SetDefault(environmentKeyPinguinConnTimeout, defaultPinguinConnTimeoutSeconds)
	application.configurationLoader.SetDefault(environmentKeyPinguinOpTimeout, defaultPinguinOpTimeoutSeconds)
	application.configurationLoader.SetDefault(environmentKeyPinguinSharedAuth, "")
	application.configurationLoader.SetDefault(environmentKeySubscriptionNotify, defaultSubscriptionNotify)
	application.configurationLoader.AutomaticEnv()

	commandFlags := command.Flags()
	commandFlags.String(flagNameConfigFile, defaultConfigFileName, flagUsageConfigFile)
	commandFlags.String(flagNameApplicationAddress, defaultApplicationAddress, flagUsageApplicationAddress)
	commandFlags.String(flagNameDatabaseDriver, defaultDatabaseDriverName, flagUsageDatabaseDriver)
	commandFlags.String(flagNameDatabaseDataSourceName, defaultSQLiteDataSourceName, flagUsageDatabaseDataSourceName)
	commandFlags.String(flagNameGoogleClientID, "", flagUsageGoogleClientID)
	commandFlags.String(flagNameGoogleClientSecret, "", flagUsageGoogleClientSecret)
	commandFlags.String(flagNameSessionSecret, "", flagUsageSessionSecret)
	commandFlags.String(flagNamePublicBaseURL, defaultPublicBaseURL, flagUsagePublicBaseURL)
	commandFlags.String(flagNamePinguinAddress, defaultPinguinAddress, flagUsagePinguinAddress)
	commandFlags.String(flagNamePinguinAuthToken, "", flagUsagePinguinAuthToken)
	commandFlags.Int(flagNamePinguinConnectionTimeout, defaultPinguinConnTimeoutSeconds, flagUsagePinguinConnTimeout)
	commandFlags.Int(flagNamePinguinOperationTimeout, defaultPinguinOpTimeoutSeconds, flagUsagePinguinOpTimeout)
	commandFlags.Bool(flagNameSubscriptionNotifications, defaultSubscriptionNotify, flagUsageSubscriptionNotify)

	if bindErr := application.bindFlag(commandFlags, environmentKeyApplicationAddress, flagNameApplicationAddress); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyDatabaseDriverName, flagNameDatabaseDriver); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyDatabaseDataSource, flagNameDatabaseDataSourceName); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyGoogleClientID, flagNameGoogleClientID); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyGoogleClientSecret, flagNameGoogleClientSecret); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeySessionSecret, flagNameSessionSecret); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyPublicBaseURL, flagNamePublicBaseURL); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyPinguinAddress, flagNamePinguinAddress); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyPinguinAuthToken, flagNamePinguinAuthToken); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyPinguinConnTimeout, flagNamePinguinConnectionTimeout); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyPinguinOpTimeout, flagNamePinguinOperationTimeout); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeySubscriptionNotify, flagNameSubscriptionNotifications); bindErr != nil {
		return bindErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyApplicationAddress, flagNameApplicationAddress); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyDatabaseDriverName, flagNameDatabaseDriver); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyDatabaseDataSource, flagNameDatabaseDataSourceName); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyGoogleClientID, flagNameGoogleClientID); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyGoogleClientSecret, flagNameGoogleClientSecret); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeySessionSecret, flagNameSessionSecret); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyPublicBaseURL, flagNamePublicBaseURL); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeySubscriptionNotify, flagNameSubscriptionNotifications); environmentErr != nil {
		return environmentErr
	}

	if markErr := command.MarkFlagRequired(flagNameGoogleClientID); markErr != nil {
		return markErr
	}

	if markErr := command.MarkFlagRequired(flagNameGoogleClientSecret); markErr != nil {
		return markErr
	}

	if markErr := command.MarkFlagRequired(flagNameSessionSecret); markErr != nil {
		return markErr
	}

	return nil
}

func (application *ServerApplication) bindFlag(flagSet *pflag.FlagSet, environmentKey string, flagName string) error {
	flag := flagSet.Lookup(flagName)
	if flag == nil {
		return fmt.Errorf(flagNotDefinedMessage, flagName)
	}

	if bindErr := application.configurationLoader.BindPFlag(environmentKey, flag); bindErr != nil {
		return bindErr
	}

	return nil
}

func (application *ServerApplication) applyEnvironmentConfiguration(flagSet *pflag.FlagSet, environmentKey string, flagName string) error {
	environmentValue, environmentFound := os.LookupEnv(environmentKey)
	if !environmentFound {
		return nil
	}

	if setErr := flagSet.Set(flagName, environmentValue); setErr != nil {
		return fmt.Errorf("%s: %w", environmentConfigurationError, setErr)
	}

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

	if validationErr := application.ensureRequiredConfiguration(serverConfig); validationErr != nil {
		return validationErr
	}

	logger, loggerErr := zap.NewProduction()
	if loggerErr != nil {
		return fmt.Errorf("%s: %w", loggerCreationErrorMessage, loggerErr)
	}
	defer func() {
		_ = logger.Sync()
	}()

	application.logAdministratorWarning(logger, serverConfig)

	session.NewSession([]byte(serverConfig.SessionSecret))

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

	authService, authErr := gauss.NewService(
		serverConfig.GoogleClientID,
		serverConfig.GoogleClientSecret,
		serverConfig.PublicBaseURL,
		dashboardRoute,
		gauss.ScopeStrings(gauss.DefaultScopes),
		"",
		gauss.WithLogoutRedirectURL(constants.LoginPath),
	)
	if authErr != nil {
		logger.Fatal(loggerContextAuthService, zap.Error(authErr))
	}

	authHandlers, handlersErr := gauss.NewHandlers(authService)
	if handlersErr != nil {
		logger.Fatal(loggerContextAuthService, zap.Error(handlersErr))
	}

	authMux := http.NewServeMux()
	authHandlers.RegisterRoutes(authMux)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsAllowOrigins,
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	sharedHTTPClient := &http.Client{Timeout: 5 * time.Second}
	authManager := httpapi.NewAuthManager(database, logger, serverConfig.AdminEmailAddresses, sharedHTTPClient, landingRouteRoot)
	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	defer feedbackBroadcaster.Close()
	subscriptionEvents := httpapi.NewSubscriptionTestEventBroadcaster()
	defer subscriptionEvents.Close()
	pinguinNotifier, notifierErr := notifications.NewPinguinNotifier(logger, notifications.PinguinConfig{
		Address:           serverConfig.PinguinAddress,
		AuthToken:         serverConfig.PinguinAuthToken,
		ConnectionTimeout: time.Duration(serverConfig.PinguinConnTimeoutSec) * time.Second,
		OperationTimeout:  time.Duration(serverConfig.PinguinOpTimeoutSec) * time.Second,
	})
	if notifierErr != nil {
		logger.Fatal("pinguin_notifier", zap.Error(notifierErr))
	}
	defer pinguinNotifier.Close()
	var subscriptionNotifier httpapi.SubscriptionNotifier
	if serverConfig.SubscriptionNotifications {
		subscriptionNotifier = pinguinNotifier
	}
	publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, subscriptionEvents, pinguinNotifier, subscriptionNotifier, serverConfig.SubscriptionNotifications)
	faviconResolver := favicon.NewHTTPResolver(sharedHTTPClient, logger)
	faviconService := favicon.NewService(faviconResolver)
	faviconManager := httpapi.NewSiteFaviconManager(database, faviconService, logger)
	faviconManagerContext, faviconManagerCancel := context.WithCancel(context.Background())
	defer faviconManager.Stop()
	defer faviconManagerCancel()
	faviconManager.Start(faviconManagerContext)
	faviconManager.TriggerScheduledRefresh()
	statsProvider := httpapi.NewDatabaseSiteStatisticsProvider(database)
	siteHandlers := httpapi.NewSiteHandlers(database, logger, serverConfig.PublicBaseURL, faviconManager, statsProvider, feedbackBroadcaster)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, landingRouteRoot)
	widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(database, logger, serverConfig.PublicBaseURL, feedbackBroadcaster, pinguinNotifier)
	trafficTestHandlers := httpapi.NewSiteTrafficTestHandlers(database, logger)
	subscribeTestHandlers := httpapi.NewSiteSubscribeTestHandlers(database, logger, subscriptionEvents)
	landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager)
	privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager)
	sitemapHandlers := httpapi.NewSitemapHandlers(serverConfig.PublicBaseURL)

	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, constants.LoginPath)
	})
	router.GET(landingRouteRoot, landingHandlers.RenderLandingPage)
	router.GET("/app/sites/:id/widget-test", authManager.RequireAuthenticatedWeb(), widgetTestHandlers.RenderWidgetTestPage)
	router.POST("/app/sites/:id/widget-test/feedback", authManager.RequireAuthenticatedJSON(), widgetTestHandlers.SubmitWidgetTestFeedback)
	router.GET("/app/sites/:id/traffic-test", authManager.RequireAuthenticatedWeb(), trafficTestHandlers.RenderTrafficTestPage)
	router.GET("/app/sites/:id/subscribe-test", authManager.RequireAuthenticatedWeb(), subscribeTestHandlers.RenderSubscribeTestPage)
	router.GET("/app/sites/:id/subscribe-test/events", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.StreamSubscriptionTestEvents)
	router.GET(httpapi.PrivacyPagePath, privacyHandlers.RenderPrivacyPage)
	router.GET(httpapi.SitemapRoutePath, sitemapHandlers.RenderSitemap)
	router.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	router.POST(publicRouteSubscription, publicHandlers.CreateSubscription)
	router.POST(publicRouteSubscriptionConfirm, publicHandlers.ConfirmSubscription)
	router.POST(publicRouteSubscriptionOptOut, publicHandlers.Unsubscribe)
	router.GET(publicRouteWidget, publicHandlers.WidgetJS)
	router.GET(publicRouteSubscribeWidget, publicHandlers.SubscribeJS)
	router.GET(publicRouteSubscribeDemo, publicHandlers.SubscribeDemo)
	router.GET(publicRouteVisitPixel, publicHandlers.CollectVisit)
	router.GET("/pixel.js", publicHandlers.PixelJS)
	router.GET(dashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)

	authHandler := gin.WrapH(authMux)
	router.GET(constants.GoogleAuthPath, authHandler)
	router.GET(constants.CallbackPath, authHandler)
	router.GET(constants.LogoutPath, authHandler)
	router.POST(constants.LogoutPath, authHandler)

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
	apiGroup.GET(apiRouteSiteFavicon, siteHandlers.SiteFavicon)
	apiGroup.GET(apiRouteSiteFaviconEvents, siteHandlers.StreamFaviconUpdates)
	apiGroup.GET(apiRouteSiteFeedbackEvents, siteHandlers.StreamFeedbackUpdates)
	apiGroup.GET(apiRouteSiteVisitStats, siteHandlers.VisitStats)

	httpServer := &http.Server{
		Addr:              serverConfig.ApplicationAddress,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeoutSeconds * time.Second,
	}

	logger.Info(logEventListening, zap.String(logFieldAddress, serverConfig.ApplicationAddress))
	if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		logger.Fatal(loggerContextServer, zap.Error(serveErr))
	}

	return nil
}

func (application *ServerApplication) loadServerConfig(configFilePath string) (ServerConfig, error) {
	trimmedConfigPath := strings.TrimSpace(configFilePath)
	if trimmedConfigPath == "" {
		trimmedConfigPath = defaultConfigFileName
	}

	application.configurationLoader.SetConfigFile(trimmedConfigPath)
	if readErr := application.configurationLoader.ReadInConfig(); readErr != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(readErr, &configFileNotFoundError) && !errors.Is(readErr, os.ErrNotExist) {
			return ServerConfig{}, fmt.Errorf("%s: %w", configurationFileLoadError, readErr)
		}
	}

	configuredAdministratorEmails := normalizeEmailAddresses(application.configurationLoader.GetStringSlice(configurationKeyAdmins))
	environmentAdministratorValue := strings.TrimSpace(application.configurationLoader.GetString(environmentKeyAdmins))

	administratorEmails := configuredAdministratorEmails
	if environmentAdministratorValue != "" {
		environmentAdministratorEmails := strings.Split(environmentAdministratorValue, administratorEmailSeparator)
		administratorEmails = normalizeEmailAddresses(environmentAdministratorEmails)
	}

	serverConfig := ServerConfig{
		ApplicationAddress:        application.configurationLoader.GetString(environmentKeyApplicationAddress),
		DatabaseDriverName:        strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDriverName)),
		DatabaseDataSourceName:    strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDataSource)),
		AdminEmailAddresses:       administratorEmails,
		GoogleClientID:            strings.TrimSpace(application.configurationLoader.GetString(environmentKeyGoogleClientID)),
		GoogleClientSecret:        strings.TrimSpace(application.configurationLoader.GetString(environmentKeyGoogleClientSecret)),
		SessionSecret:             strings.TrimSpace(application.configurationLoader.GetString(environmentKeySessionSecret)),
		PublicBaseURL:             strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPublicBaseURL)),
		ConfigFilePath:            trimmedConfigPath,
		PinguinAddress:            strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinAddress)),
		PinguinAuthToken:          strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinAuthToken)),
		PinguinConnTimeoutSec:     application.configurationLoader.GetInt(environmentKeyPinguinConnTimeout),
		PinguinOpTimeoutSec:       application.configurationLoader.GetInt(environmentKeyPinguinOpTimeout),
		SubscriptionNotifications: application.configurationLoader.GetBool(environmentKeySubscriptionNotify),
	}

	if serverConfig.PinguinAuthToken == "" {
		sharedAuthToken := strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinSharedAuth))
		if sharedAuthToken != "" {
			serverConfig.PinguinAuthToken = sharedAuthToken
		}
	}

	if serverConfig.DatabaseDriverName == storage.DriverNameSQLite && serverConfig.DatabaseDataSourceName == "" {
		serverConfig.DatabaseDataSourceName = defaultSQLiteDataSourceName
	}

	return serverConfig, nil
}

func normalizeEmailAddresses(rawEmailAddresses []string) []string {
	normalizedEmailAddresses := make([]string, 0, len(rawEmailAddresses))
	for _, rawEmailAddress := range rawEmailAddresses {
		trimmedEmailAddress := strings.TrimSpace(rawEmailAddress)
		if trimmedEmailAddress == "" {
			continue
		}

		normalizedEmailAddresses = append(normalizedEmailAddresses, trimmedEmailAddress)
	}

	return normalizedEmailAddresses
}

func (application *ServerApplication) logAdministratorWarning(logger *zap.Logger, configuration ServerConfig) {
	if len(configuration.AdminEmailAddresses) > 0 {
		return
	}

	logger.Warn(logMessageMissingAdministrators)
}

func (application *ServerApplication) ensureRequiredConfiguration(configuration ServerConfig) error {
	var missingParameters []string

	if configuration.DatabaseDriverName == "" {
		missingParameters = append(missingParameters, flagNameDatabaseDriver)
	}

	if configuration.DatabaseDriverName != storage.DriverNameSQLite && configuration.DatabaseDataSourceName == "" {
		missingParameters = append(missingParameters, flagNameDatabaseDataSourceName)
	}

	if configuration.GoogleClientID == "" {
		missingParameters = append(missingParameters, flagNameGoogleClientID)
	}

	if configuration.GoogleClientSecret == "" {
		missingParameters = append(missingParameters, flagNameGoogleClientSecret)
	}

	if configuration.SessionSecret == "" {
		missingParameters = append(missingParameters, flagNameSessionSecret)
	}

	if configuration.PublicBaseURL == "" {
		missingParameters = append(missingParameters, flagNamePublicBaseURL)
	}

	if configuration.PinguinAddress == "" {
		missingParameters = append(missingParameters, flagNamePinguinAddress)
	}

	if configuration.PinguinAuthToken == "" {
		missingParameters = append(missingParameters, flagNamePinguinAuthToken)
	}

	if configuration.PinguinConnTimeoutSec <= 0 {
		missingParameters = append(missingParameters, flagNamePinguinConnectionTimeout)
	}

	if configuration.PinguinOpTimeoutSec <= 0 {
		missingParameters = append(missingParameters, flagNamePinguinOperationTimeout)
	}

	if len(missingParameters) == 0 {
		return nil
	}

	return fmt.Errorf("%s: %s", missingConfigurationMessage, strings.Join(missingParameters, ", "))
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
