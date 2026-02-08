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
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	flagNameSessionSecret             = "session-secret"
	flagNameTauthBaseURL              = "tauth-base-url"
	flagNameTauthTenantID             = "tauth-tenant-id"
	flagNameTauthSigningKey           = "tauth-signing-key"
	flagNameTauthSessionCookieName    = "tauth-session-cookie-name"
	flagNameServeMode                 = "serve-mode"
	flagNameSubscriptionNotifications = "subscription-notifications"
	flagNamePublicBaseURL             = "public-base-url"
	flagNamePinguinAddress            = "pinguin-addr"
	flagNamePinguinAuthToken          = "pinguin-auth-token"
	flagNamePinguinTenantID           = "pinguin-tenant-id"
	flagNamePinguinConnectionTimeout  = "pinguin-conn-timeout"
	flagNamePinguinOperationTimeout   = "pinguin-op-timeout"
	flagUsageConfigFile               = "path to configuration file"
	flagUsageApplicationAddress       = "address for the HTTP server to listen on"
	flagUsageDatabaseDriver           = "database driver (e.g. sqlite)"
	flagUsageDatabaseDataSourceName   = "database connection string"
	flagUsageGoogleClientID           = "Google OAuth client ID"
	flagUsageSessionSecret            = "secret for subscription confirmation tokens"
	flagUsageTauthBaseURL             = "base URL for the TAuth service"
	flagUsageTauthTenantID            = "tenant identifier configured in TAuth"
	flagUsageTauthSigningKey          = "JWT signing key for validating TAuth sessions"
	flagUsageTauthSessionCookieName   = "session cookie name used by TAuth"
	flagUsageServeMode                = "server routing mode (monolith, web, api)"
	flagUsagePublicBaseURL            = "public base URL for landing pages and sitemap"
	flagUsagePinguinAddress           = "Pinguin gRPC server address"
	flagUsagePinguinAuthToken         = "Pinguin bearer auth token"
	flagUsagePinguinTenantID          = "Pinguin tenant identifier"
	flagUsagePinguinConnTimeout       = "Pinguin connection timeout in seconds"
	flagUsagePinguinOpTimeout         = "Pinguin operation timeout in seconds"
	flagUsageSubscriptionNotify       = "enable notifications for new subscriptions"
	environmentKeyApplicationAddress  = "APP_ADDR"
	environmentKeyDatabaseDriverName  = "DB_DRIVER"
	environmentKeyDatabaseDataSource  = "DB_DSN"
	environmentKeyAdmins              = "ADMINS"
	environmentKeyGoogleClientID      = "GOOGLE_CLIENT_ID"
	environmentKeySessionSecret       = "SESSION_SECRET"
	environmentKeyTauthBaseURL        = "TAUTH_BASE_URL"
	environmentKeyTauthTenantID       = "TAUTH_TENANT_ID"
	environmentKeyTauthSigningKey     = "TAUTH_JWT_SIGNING_KEY"
	environmentKeyTauthSessionCookie  = "TAUTH_SESSION_COOKIE_NAME"
	environmentKeyServeMode           = "SERVE_MODE"
	environmentKeyPublicBaseURL       = "PUBLIC_BASE_URL"
	environmentKeyPinguinAddress      = "PINGUIN_ADDR"
	environmentKeyPinguinAuthToken    = "PINGUIN_AUTH_TOKEN"
	environmentKeyPinguinTenantID     = "PINGUIN_TENANT_ID"
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
	defaultTauthSessionCookieName     = "app_session"
	defaultServeMode                  = string(ServeModeMonolith)
	defaultPinguinAddress             = "localhost:50051"
	defaultPinguinConnTimeoutSeconds  = 5
	defaultPinguinOpTimeoutSeconds    = 30
	defaultSubscriptionNotify         = true
	publicRouteFeedback               = "/api/feedback"
	publicRouteSubscription           = "/api/subscriptions"
	publicRouteSubscriptionConfirm    = "/api/subscriptions/confirm"
	publicRouteSubscriptionOptOut     = "/api/subscriptions/unsubscribe"
	publicRouteSubscriptionConfirmWeb = "/subscriptions/confirm"
	publicRouteSubscriptionOptOutWeb  = "/subscriptions/unsubscribe"
	publicRouteSubscribeWidget        = "/subscribe.js"
	publicRouteSubscribeDemo          = "/subscribe-demo"
	publicRouteVisitPixel             = "/api/visits"
	publicRouteWidget                 = "/widget.js"
	landingRouteRoot                  = httpapi.LandingPagePath
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
	corsHeaderXTAuthTenant            = "X-TAuth-Tenant"
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
	corsAllowedHeaders          = []string{corsHeaderAuthorization, corsHeaderContentType, corsHeaderXTAuthTenant}
	corsExposedHeaders          = []string{corsHeaderContentType}
	defaultDatabaseDriverName   = storage.DriverNameSQLite
	defaultSQLiteDataSourceName = fmt.Sprintf(sqliteFileDataSourceNamePattern, defaultSQLiteDatabaseFileName)
)

// ServerConfig captures configuration needed to run the server.
type ServerConfig struct {
	ApplicationAddress        string
	ServeMode                 ServeMode
	DatabaseDriverName        string
	DatabaseDataSourceName    string
	AdminEmailAddresses       []string
	GoogleClientID            string
	SessionSecret             string
	TauthBaseURL              string
	TauthTenantID             string
	TauthSigningKey           string
	TauthSessionCookieName    string
	PublicBaseURL             string
	ConfigFilePath            string
	PinguinAddress            string
	PinguinAuthToken          string
	PinguinTenantID           string
	PinguinConnTimeoutSec     int
	PinguinOpTimeoutSec       int
	SubscriptionNotifications bool
}

// DatabaseOpener opens a database connection using the provided configuration.
type DatabaseOpener func(storage.Config) (*gorm.DB, error)

// ServerRunner executes the HTTP server.
type ServerRunner func(*http.Server) error

// ServerApplication constructs and executes the server command.
type ServerApplication struct {
	configurationLoader *viper.Viper
	databaseOpener      DatabaseOpener
	serverRunner        ServerRunner
	pinguinDialer       func(context.Context, string) (net.Conn, error)
}

// NewServerApplication creates a ServerApplication with default dependencies.
func NewServerApplication() *ServerApplication {
	return &ServerApplication{
		configurationLoader: viper.New(),
		databaseOpener:      storage.OpenDatabase,
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
	defaults := []struct {
		environmentKey string
		value          any
	}{
		{environmentKeyApplicationAddress, defaultApplicationAddress},
		{environmentKeyServeMode, defaultServeMode},
		{environmentKeyDatabaseDriverName, defaultDatabaseDriverName},
		{environmentKeyDatabaseDataSource, defaultSQLiteDataSourceName},
		{environmentKeyPublicBaseURL, defaultPublicBaseURL},
		{environmentKeyGoogleClientID, ""},
		{environmentKeySessionSecret, ""},
		{environmentKeyTauthBaseURL, ""},
		{environmentKeyTauthTenantID, ""},
		{environmentKeyTauthSigningKey, ""},
		{environmentKeyTauthSessionCookie, defaultTauthSessionCookieName},
		{environmentKeyPinguinAddress, defaultPinguinAddress},
		{environmentKeyPinguinAuthToken, ""},
		{environmentKeyPinguinTenantID, ""},
		{environmentKeyPinguinConnTimeout, defaultPinguinConnTimeoutSeconds},
		{environmentKeyPinguinOpTimeout, defaultPinguinOpTimeoutSeconds},
		{environmentKeyPinguinSharedAuth, ""},
		{environmentKeySubscriptionNotify, defaultSubscriptionNotify},
	}
	for _, entry := range defaults {
		application.configurationLoader.SetDefault(entry.environmentKey, entry.value)
	}
	application.configurationLoader.AutomaticEnv()

	commandFlags := command.Flags()
	stringFlags := []struct {
		flagName     string
		defaultValue string
		usage        string
	}{
		{flagNameConfigFile, defaultConfigFileName, flagUsageConfigFile},
		{flagNameApplicationAddress, defaultApplicationAddress, flagUsageApplicationAddress},
		{flagNameServeMode, defaultServeMode, flagUsageServeMode},
		{flagNameDatabaseDriver, defaultDatabaseDriverName, flagUsageDatabaseDriver},
		{flagNameDatabaseDataSourceName, defaultSQLiteDataSourceName, flagUsageDatabaseDataSourceName},
		{flagNameGoogleClientID, "", flagUsageGoogleClientID},
		{flagNameSessionSecret, "", flagUsageSessionSecret},
		{flagNameTauthBaseURL, "", flagUsageTauthBaseURL},
		{flagNameTauthTenantID, "", flagUsageTauthTenantID},
		{flagNameTauthSigningKey, "", flagUsageTauthSigningKey},
		{flagNameTauthSessionCookieName, defaultTauthSessionCookieName, flagUsageTauthSessionCookieName},
		{flagNamePublicBaseURL, defaultPublicBaseURL, flagUsagePublicBaseURL},
		{flagNamePinguinAddress, defaultPinguinAddress, flagUsagePinguinAddress},
		{flagNamePinguinAuthToken, "", flagUsagePinguinAuthToken},
		{flagNamePinguinTenantID, "", flagUsagePinguinTenantID},
	}
	for _, flagEntry := range stringFlags {
		commandFlags.String(flagEntry.flagName, flagEntry.defaultValue, flagEntry.usage)
	}

	intFlags := []struct {
		flagName     string
		defaultValue int
		usage        string
	}{
		{flagNamePinguinConnectionTimeout, defaultPinguinConnTimeoutSeconds, flagUsagePinguinConnTimeout},
		{flagNamePinguinOperationTimeout, defaultPinguinOpTimeoutSeconds, flagUsagePinguinOpTimeout},
	}
	for _, flagEntry := range intFlags {
		commandFlags.Int(flagEntry.flagName, flagEntry.defaultValue, flagEntry.usage)
	}

	boolFlags := []struct {
		flagName     string
		defaultValue bool
		usage        string
	}{
		{flagNameSubscriptionNotifications, defaultSubscriptionNotify, flagUsageSubscriptionNotify},
	}
	for _, flagEntry := range boolFlags {
		commandFlags.Bool(flagEntry.flagName, flagEntry.defaultValue, flagEntry.usage)
	}

	flagBindings := []struct {
		environmentKey string
		flagName       string
	}{
		{environmentKeyApplicationAddress, flagNameApplicationAddress},
		{environmentKeyServeMode, flagNameServeMode},
		{environmentKeyDatabaseDriverName, flagNameDatabaseDriver},
		{environmentKeyDatabaseDataSource, flagNameDatabaseDataSourceName},
		{environmentKeyGoogleClientID, flagNameGoogleClientID},
		{environmentKeySessionSecret, flagNameSessionSecret},
		{environmentKeyTauthBaseURL, flagNameTauthBaseURL},
		{environmentKeyTauthTenantID, flagNameTauthTenantID},
		{environmentKeyTauthSigningKey, flagNameTauthSigningKey},
		{environmentKeyTauthSessionCookie, flagNameTauthSessionCookieName},
		{environmentKeyPublicBaseURL, flagNamePublicBaseURL},
		{environmentKeyPinguinAddress, flagNamePinguinAddress},
		{environmentKeyPinguinAuthToken, flagNamePinguinAuthToken},
		{environmentKeyPinguinTenantID, flagNamePinguinTenantID},
		{environmentKeyPinguinConnTimeout, flagNamePinguinConnectionTimeout},
		{environmentKeyPinguinOpTimeout, flagNamePinguinOperationTimeout},
		{environmentKeySubscriptionNotify, flagNameSubscriptionNotifications},
	}
	for _, binding := range flagBindings {
		if bindErr := application.bindFlag(commandFlags, binding.environmentKey, binding.flagName); bindErr != nil {
			return bindErr
		}
	}
	for _, binding := range flagBindings {
		if environmentErr := application.applyEnvironmentConfiguration(commandFlags, binding.environmentKey, binding.flagName); environmentErr != nil {
			return environmentErr
		}
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

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	sharedHTTPClient := &http.Client{Timeout: 5 * time.Second}
	authClientConfig := httpapi.NewAuthClientConfig(serverConfig.GoogleClientID, serverConfig.TauthBaseURL, serverConfig.TauthTenantID)
	if serverConfig.ServeMode == ServeModeWeb {
		authManager, authManagerErr := httpapi.NewAuthManager(nil, logger, serverConfig.AdminEmailAddresses, sharedHTTPClient, landingRouteRoot, httpapi.AuthConfig{
			SigningKey: serverConfig.TauthSigningKey,
			CookieName: serverConfig.TauthSessionCookieName,
			TenantID:   serverConfig.TauthTenantID,
		})
		if authManagerErr != nil {
			logger.Fatal(loggerContextAuthService, zap.Error(authManagerErr))
		}

		dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, landingRouteRoot, authClientConfig, "")
		landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager, authClientConfig, "")
		privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager, authClientConfig)
		sitemapHandlers := httpapi.NewSitemapHandlers(serverConfig.PublicBaseURL)
		publicJavaScriptHandlers := httpapi.NewPublicJavaScriptHandlers()
		subscribeDemoHandlers := httpapi.NewSubscribeDemoPageHandlers(logger)
		subscriptionLinkHandlers := httpapi.NewSubscriptionLinkPageHandlers(logger, authClientConfig, "")
		widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(nil, logger, serverConfig.PublicBaseURL, nil, nil, authClientConfig)
		trafficTestHandlers := httpapi.NewSiteTrafficTestHandlers(nil, logger, authClientConfig)
		subscribeTestHandlers := httpapi.NewSiteSubscribeTestHandlers(nil, logger, nil, nil, false, serverConfig.PublicBaseURL, "", nil, authClientConfig)
		registerFrontendRoutes(router, authManager, landingHandlers, privacyHandlers, sitemapHandlers, dashboardHandlers, publicJavaScriptHandlers, subscribeDemoHandlers, subscriptionLinkHandlers, widgetTestHandlers, trafficTestHandlers, subscribeTestHandlers)
	} else {
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

		authManager, authManagerErr := httpapi.NewAuthManager(database, logger, serverConfig.AdminEmailAddresses, sharedHTTPClient, landingRouteRoot, httpapi.AuthConfig{
			SigningKey: serverConfig.TauthSigningKey,
			CookieName: serverConfig.TauthSessionCookieName,
			TenantID:   serverConfig.TauthTenantID,
		})
		if authManagerErr != nil {
			logger.Fatal(loggerContextAuthService, zap.Error(authManagerErr))
		}

		feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
		defer feedbackBroadcaster.Close()
		subscriptionEvents := httpapi.NewSubscriptionTestEventBroadcaster()
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
		var subscriptionNotifier httpapi.SubscriptionNotifier
		if serverConfig.SubscriptionNotifications {
			subscriptionNotifier = pinguinNotifier
		}
		publicHandlers := httpapi.NewPublicHandlers(database, logger, feedbackBroadcaster, subscriptionEvents, pinguinNotifier, subscriptionNotifier, serverConfig.SubscriptionNotifications, serverConfig.PublicBaseURL, serverConfig.SessionSecret, pinguinNotifier, authClientConfig)
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
		widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(database, logger, serverConfig.PublicBaseURL, feedbackBroadcaster, pinguinNotifier, authClientConfig)
		trafficTestHandlers := httpapi.NewSiteTrafficTestHandlers(database, logger, authClientConfig)
		subscribeTestHandlers := httpapi.NewSiteSubscribeTestHandlers(database, logger, subscriptionEvents, subscriptionNotifier, serverConfig.SubscriptionNotifications, serverConfig.PublicBaseURL, serverConfig.SessionSecret, pinguinNotifier, authClientConfig)
		authenticatedOrigin, originErr := resolveOrigin(serverConfig.PublicBaseURL)
		if originErr != nil {
			logger.Fatal("cors_origin", zap.Error(originErr))
		}
		registerBackendRoutes(router, authManager, publicHandlers, siteHandlers, widgetTestHandlers, subscribeTestHandlers, authenticatedOrigin)

		if serverConfig.ServeMode == ServeModeMonolith {
			dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, landingRouteRoot, authClientConfig, "")
			landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager, authClientConfig, "")
			privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager, authClientConfig)
			sitemapHandlers := httpapi.NewSitemapHandlers(serverConfig.PublicBaseURL)
			publicJavaScriptHandlers := httpapi.NewPublicJavaScriptHandlers()
			subscribeDemoHandlers := httpapi.NewSubscribeDemoPageHandlers(logger)
			subscriptionLinkHandlers := httpapi.NewSubscriptionLinkPageHandlers(logger, authClientConfig, "")
			registerFrontendRoutes(router, authManager, landingHandlers, privacyHandlers, sitemapHandlers, dashboardHandlers, publicJavaScriptHandlers, subscribeDemoHandlers, subscriptionLinkHandlers, widgetTestHandlers, trafficTestHandlers, subscribeTestHandlers)
		}
	}

	httpServer := &http.Server{
		Addr:              serverConfig.ApplicationAddress,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeoutSeconds * time.Second,
	}

	logger.Info(logEventListening, zap.String(logFieldAddress, serverConfig.ApplicationAddress))
	if serveErr := application.serverRunner(httpServer); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		logger.Fatal(loggerContextServer, zap.Error(serveErr))
	}

	return nil
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

	serveMode, serveModeErr := ParseServeMode(application.configurationLoader.GetString(environmentKeyServeMode))
	if serveModeErr != nil {
		return ServerConfig{}, serveModeErr
	}

	serverConfig := ServerConfig{
		ApplicationAddress:        application.configurationLoader.GetString(environmentKeyApplicationAddress),
		ServeMode:                 serveMode,
		DatabaseDriverName:        strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDriverName)),
		DatabaseDataSourceName:    strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDataSource)),
		AdminEmailAddresses:       administratorEmails,
		GoogleClientID:            strings.TrimSpace(application.configurationLoader.GetString(environmentKeyGoogleClientID)),
		SessionSecret:             strings.TrimSpace(application.configurationLoader.GetString(environmentKeySessionSecret)),
		TauthBaseURL:              strings.TrimSpace(application.configurationLoader.GetString(environmentKeyTauthBaseURL)),
		TauthTenantID:             strings.TrimSpace(application.configurationLoader.GetString(environmentKeyTauthTenantID)),
		TauthSigningKey:           strings.TrimSpace(application.configurationLoader.GetString(environmentKeyTauthSigningKey)),
		TauthSessionCookieName:    strings.TrimSpace(application.configurationLoader.GetString(environmentKeyTauthSessionCookie)),
		PublicBaseURL:             strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPublicBaseURL)),
		ConfigFilePath:            trimmedConfigPath,
		PinguinAddress:            strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinAddress)),
		PinguinAuthToken:          strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinAuthToken)),
		PinguinTenantID:           strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPinguinTenantID)),
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

	serveMode := configuration.ServeMode
	if serveMode == "" {
		serveMode = ServeModeMonolith
	}
	if configuration.GoogleClientID == "" {
		missingParameters = append(missingParameters, flagNameGoogleClientID)
	}

	if configuration.TauthBaseURL == "" {
		missingParameters = append(missingParameters, flagNameTauthBaseURL)
	}

	if configuration.TauthTenantID == "" {
		missingParameters = append(missingParameters, flagNameTauthTenantID)
	}

	if configuration.TauthSigningKey == "" {
		missingParameters = append(missingParameters, flagNameTauthSigningKey)
	}

	if configuration.PublicBaseURL == "" {
		missingParameters = append(missingParameters, flagNamePublicBaseURL)
	}

	if serveMode != ServeModeWeb {
		if configuration.DatabaseDriverName == "" {
			missingParameters = append(missingParameters, flagNameDatabaseDriver)
		}

		if configuration.DatabaseDriverName != storage.DriverNameSQLite && configuration.DatabaseDataSourceName == "" {
			missingParameters = append(missingParameters, flagNameDatabaseDataSourceName)
		}

		if configuration.SessionSecret == "" {
			missingParameters = append(missingParameters, flagNameSessionSecret)
		}

		if configuration.PinguinAddress == "" {
			missingParameters = append(missingParameters, flagNamePinguinAddress)
		}

		if configuration.PinguinAuthToken == "" {
			missingParameters = append(missingParameters, flagNamePinguinAuthToken)
		}

		if configuration.PinguinTenantID == "" {
			missingParameters = append(missingParameters, flagNamePinguinTenantID)
		}

		if configuration.PinguinConnTimeoutSec <= 0 {
			missingParameters = append(missingParameters, flagNamePinguinConnectionTimeout)
		}

		if configuration.PinguinOpTimeoutSec <= 0 {
			missingParameters = append(missingParameters, flagNamePinguinOperationTimeout)
		}
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
