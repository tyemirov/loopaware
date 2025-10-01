package main

import (
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

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

const (
	commandUseName                   = "server"
	commandShortDescription          = "Run the feedback server"
	commandLongDescription           = "Launch the feedback collection HTTP server"
	missingConfigurationMessage      = "missing required configuration"
	loggerCreationErrorMessage       = "logger"
	logEventListening                = "listening"
	logFieldAddress                  = "addr"
	flagNameConfigFile               = "config"
	flagNameApplicationAddress       = "app-addr"
	flagNameDatabaseDriver           = "db-driver"
	flagNameDatabaseDataSourceName   = "db-dsn"
	flagNameGoogleClientID           = "google-client-id"
	flagNameGoogleClientSecret       = "google-client-secret"
	flagNameSessionSecret            = "session-secret"
	flagNamePublicBaseURL            = "public-base-url"
	flagUsageConfigFile              = "path to configuration file"
	flagUsageApplicationAddress      = "address for the HTTP server to listen on"
	flagUsageDatabaseDriver          = "database driver (e.g. sqlite)"
	flagUsageDatabaseDataSourceName  = "database connection string"
	flagUsageGoogleClientID          = "Google OAuth client ID"
	flagUsageGoogleClientSecret      = "Google OAuth client secret"
	flagUsageSessionSecret           = "session secret for browser sessions"
	flagUsagePublicBaseURL           = "public base URL for OAuth callbacks"
	environmentKeyApplicationAddress = "APP_ADDR"
	environmentKeyDatabaseDriverName = "DB_DRIVER"
	environmentKeyDatabaseDataSource = "DB_DSN"
	environmentKeyGoogleClientID     = "GOOGLE_CLIENT_ID"
	environmentKeyGoogleClientSecret = "GOOGLE_CLIENT_SECRET"
	environmentKeySessionSecret      = "SESSION_SECRET"
	environmentKeyPublicBaseURL      = "PUBLIC_BASE_URL"
	configurationKeyAdmins           = "admins"
	defaultApplicationAddress        = ":8080"
	sqliteFileDataSourceNamePattern  = "file:%s?_foreign_keys=on"
	defaultSQLiteDatabaseFileName    = "loopaware.sqlite"
	defaultConfigFileName            = "config.yaml"
	defaultPublicBaseURL             = "http://localhost:8080"
	publicRouteFeedback              = "/api/feedback"
	publicRouteWidget                = "/widget.js"
	dashboardRoute                   = "/app"
	apiRoutePrefix                   = "/api"
	apiRouteMe                       = "/me"
	apiRouteSites                    = "/sites"
	apiRouteSiteUpdate               = "/sites/:id"
	apiRouteSiteMessages             = "/sites/:id/messages"
	corsOriginWildcard               = "*"
	corsHeaderAuthorization          = "Authorization"
	corsHeaderContentType            = "Content-Type"
	httpMethodGet                    = "GET"
	httpMethodOptions                = "OPTIONS"
	httpMethodPost                   = "POST"
	httpMethodPatch                  = "PATCH"
	loggerContextOpenDatabase        = "open_db"
	loggerContextAutoMigrate         = "migrate"
	loggerContextServer              = "server"
	loggerContextAuthService         = "auth_service"
	loggerContextTemplate            = "template"
	readHeaderTimeoutSeconds         = 5
	unexpectedArgumentsMessage       = "unexpected command arguments"
	commandInitializationFailure     = "failed to configure command"
	flagNotDefinedMessage            = "flag %s not defined"
	environmentConfigurationError    = "failed to apply environment configuration"
	configurationFileLoadError       = "failed to load configuration file"
)

var (
	corsAllowedMethods          = []string{httpMethodPost, httpMethodGet, httpMethodOptions, httpMethodPatch}
	corsAllowedHeaders          = []string{corsHeaderAuthorization, corsHeaderContentType}
	corsExposedHeaders          = []string{corsHeaderContentType}
	corsAllowOrigins            = []string{corsOriginWildcard}
	defaultDatabaseDriverName   = storage.DriverNameSQLite
	defaultSQLiteDataSourceName = fmt.Sprintf(sqliteFileDataSourceNamePattern, defaultSQLiteDatabaseFileName)
)

// ServerConfig captures configuration needed to run the server.
type ServerConfig struct {
	ApplicationAddress     string
	DatabaseDriverName     string
	DatabaseDataSourceName string
	AdminEmailAddresses    []string
	GoogleClientID         string
	GoogleClientSecret     string
	SessionSecret          string
	PublicBaseURL          string
	ConfigFilePath         string
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
	if configFilePath == "" {
		configFilePath = defaultConfigFileName
	}

	application.configurationLoader.SetConfigFile(configFilePath)
	if readErr := application.configurationLoader.ReadInConfig(); readErr != nil {
		return fmt.Errorf("%s: %w", configurationFileLoadError, readErr)
	}

	adminEmails := application.configurationLoader.GetStringSlice(configurationKeyAdmins)
	for index := range adminEmails {
		adminEmails[index] = strings.TrimSpace(adminEmails[index])
	}

	serverConfig := ServerConfig{
		ApplicationAddress:     application.configurationLoader.GetString(environmentKeyApplicationAddress),
		DatabaseDriverName:     strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDriverName)),
		DatabaseDataSourceName: strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDataSource)),
		AdminEmailAddresses:    adminEmails,
		GoogleClientID:         strings.TrimSpace(application.configurationLoader.GetString(environmentKeyGoogleClientID)),
		GoogleClientSecret:     strings.TrimSpace(application.configurationLoader.GetString(environmentKeyGoogleClientSecret)),
		SessionSecret:          strings.TrimSpace(application.configurationLoader.GetString(environmentKeySessionSecret)),
		PublicBaseURL:          strings.TrimSpace(application.configurationLoader.GetString(environmentKeyPublicBaseURL)),
		ConfigFilePath:         configFilePath,
	}

	if serverConfig.DatabaseDriverName == storage.DriverNameSQLite && serverConfig.DatabaseDataSourceName == "" {
		serverConfig.DatabaseDataSourceName = defaultSQLiteDataSourceName
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
	)
	if authErr != nil {
		logger.Fatal(loggerContextAuthService, zap.Error(authErr))
	}

	authHandlers, handlersErr := gauss.NewHandlers(authService)
	if handlersErr != nil {
		logger.Fatal(loggerContextTemplate, zap.Error(handlersErr))
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

	publicHandlers := httpapi.NewPublicHandlers(database, logger)
	siteHandlers := httpapi.NewSiteHandlers(database, logger)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger)
	authManager := httpapi.NewAuthManager(logger, serverConfig.AdminEmailAddresses)

	router.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	router.GET(publicRouteWidget, publicHandlers.WidgetJS)
	router.GET(dashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)

	authHandler := gin.WrapH(authMux)
	router.GET(constants.LoginPath, authHandler)
	router.GET(constants.GoogleAuthPath, authHandler)
	router.GET(constants.CallbackPath, authHandler)
	router.GET(constants.LogoutPath, authHandler)
	router.POST(constants.LogoutPath, authHandler)

	apiGroup := router.Group(apiRoutePrefix)
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET(apiRouteMe, siteHandlers.CurrentUser)
	apiGroup.GET(apiRouteSites, siteHandlers.ListSites)
	apiGroup.PATCH(apiRouteSiteUpdate, siteHandlers.UpdateSite)
	apiGroup.GET(apiRouteSiteMessages, siteHandlers.ListMessagesBySite)

	adminSitesGroup := apiGroup.Group(apiRouteSites)
	adminSitesGroup.Use(authManager.RequireAdminJSON())
	adminSitesGroup.POST("", siteHandlers.CreateSite)

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

func (application *ServerApplication) ensureRequiredConfiguration(configuration ServerConfig) error {
	var missingParameters []string

	if configuration.DatabaseDriverName == "" {
		missingParameters = append(missingParameters, flagNameDatabaseDriver)
	}

	if configuration.DatabaseDriverName != storage.DriverNameSQLite && configuration.DatabaseDataSourceName == "" {
		missingParameters = append(missingParameters, flagNameDatabaseDataSourceName)
	}

	if len(configuration.AdminEmailAddresses) == 0 {
		missingParameters = append(missingParameters, configurationKeyAdmins)
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
