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
	flagNameApplicationAddress       = "app-addr"
	flagNameDatabaseDataSourceName   = "db-dsn"
	flagNameAdminBearerToken         = "admin-bearer-token"
	flagUsageApplicationAddress      = "address for the HTTP server to listen on"
	flagUsageDatabaseDataSourceName  = "PostgreSQL connection string"
	flagUsageAdminBearerToken        = "bearer token required for admin API access"
	environmentKeyApplicationAddress = "APP_ADDR"
	environmentKeyDatabaseDataSource = "DB_DSN"
	environmentKeyAdminBearerToken   = "ADMIN_BEARER_TOKEN"
	defaultApplicationAddress        = ":8080"
	adminRoutePrefix                 = "/api/admin"
	adminRouteSites                  = "/sites"
	adminRouteMessagesBySite         = "/sites/:id/messages"
	publicRouteFeedback              = "/api/feedback"
	publicRouteWidget                = "/widget.js"
	corsOriginWildcard               = "*"
	corsHeaderAuthorization          = "Authorization"
	corsHeaderContentType            = "Content-Type"
	httpMethodGet                    = "GET"
	httpMethodOptions                = "OPTIONS"
	httpMethodPost                   = "POST"
	loggerContextOpenDatabase        = "open_db"
	loggerContextAutoMigrate         = "migrate"
	loggerContextServer              = "server"
	readHeaderTimeoutSeconds         = 5
	unexpectedArgumentsMessage       = "unexpected command arguments"
	commandInitializationFailure     = "failed to configure command"
	flagNotDefinedMessage            = "flag %s not defined"
	environmentConfigurationError    = "failed to apply environment configuration"
)

var (
	corsAllowedMethods = []string{httpMethodPost, httpMethodGet, httpMethodOptions}
	corsAllowedHeaders = []string{corsHeaderAuthorization, corsHeaderContentType}
	corsExposedHeaders = []string{corsHeaderContentType}
	corsAllowOrigins   = []string{corsOriginWildcard}
)

// ServerConfig captures configuration needed to run the server.
type ServerConfig struct {
	ApplicationAddress     string
	DatabaseDataSourceName string
	AdminBearerToken       string
}

// DatabaseOpener opens a database connection using the provided data source name.
type DatabaseOpener func(string) (*gorm.DB, error)

// ServerApplication constructs and executes the server command.
type ServerApplication struct {
	configurationLoader *viper.Viper
	databaseOpener      DatabaseOpener
}

// NewServerApplication creates a ServerApplication with default dependencies.
func NewServerApplication() *ServerApplication {
	return &ServerApplication{
		configurationLoader: viper.New(),
		databaseOpener:      storage.OpenPostgres,
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
	application.configurationLoader.SetDefault(environmentKeyDatabaseDataSource, "")
	application.configurationLoader.SetDefault(environmentKeyAdminBearerToken, "")
	application.configurationLoader.AutomaticEnv()

	commandFlags := command.Flags()
	commandFlags.String(flagNameApplicationAddress, defaultApplicationAddress, flagUsageApplicationAddress)
	commandFlags.String(flagNameDatabaseDataSourceName, "", flagUsageDatabaseDataSourceName)
	commandFlags.String(flagNameAdminBearerToken, "", flagUsageAdminBearerToken)

	if bindErr := application.bindFlag(commandFlags, environmentKeyApplicationAddress, flagNameApplicationAddress); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyDatabaseDataSource, flagNameDatabaseDataSourceName); bindErr != nil {
		return bindErr
	}

	if bindErr := application.bindFlag(commandFlags, environmentKeyAdminBearerToken, flagNameAdminBearerToken); bindErr != nil {
		return bindErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyApplicationAddress, flagNameApplicationAddress); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyDatabaseDataSource, flagNameDatabaseDataSourceName); environmentErr != nil {
		return environmentErr
	}

	if environmentErr := application.applyEnvironmentConfiguration(commandFlags, environmentKeyAdminBearerToken, flagNameAdminBearerToken); environmentErr != nil {
		return environmentErr
	}

	if markErr := command.MarkFlagRequired(flagNameDatabaseDataSourceName); markErr != nil {
		return markErr
	}

	if markErr := command.MarkFlagRequired(flagNameAdminBearerToken); markErr != nil {
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

	serverConfig := ServerConfig{
		ApplicationAddress:     application.configurationLoader.GetString(environmentKeyApplicationAddress),
		DatabaseDataSourceName: strings.TrimSpace(application.configurationLoader.GetString(environmentKeyDatabaseDataSource)),
		AdminBearerToken:       strings.TrimSpace(application.configurationLoader.GetString(environmentKeyAdminBearerToken)),
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

	database, databaseErr := application.databaseOpener(serverConfig.DatabaseDataSourceName)
	if databaseErr != nil {
		logger.Fatal(loggerContextOpenDatabase, zap.Error(databaseErr))
	}

	if migrateErr := storage.AutoMigrate(database); migrateErr != nil {
		logger.Fatal(loggerContextAutoMigrate, zap.Error(migrateErr))
	}

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
	adminHandlers := httpapi.NewAdminHandlers(database, logger, serverConfig.AdminBearerToken)

	router.POST(publicRouteFeedback, publicHandlers.CreateFeedback)
	router.GET(publicRouteWidget, publicHandlers.WidgetJS)

	adminGroup := router.Group(adminRoutePrefix)
	adminGroup.Use(httpapi.AdminAuthMiddleware(serverConfig.AdminBearerToken))
	adminGroup.POST(adminRouteSites, adminHandlers.CreateSite)
	adminGroup.GET(adminRouteMessagesBySite, adminHandlers.ListMessagesBySite)

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

	if configuration.DatabaseDataSourceName == "" {
		missingParameters = append(missingParameters, flagNameDatabaseDataSourceName)
	}

	if configuration.AdminBearerToken == "" {
		missingParameters = append(missingParameters, flagNameAdminBearerToken)
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
		fmt.Fprintf(os.Stderr, "%s: %v\n", commandInitializationFailure, commandErr)
		os.Exit(1)
	}

	if executeErr := rootCommand.Execute(); executeErr != nil {
		os.Exit(1)
	}
}
