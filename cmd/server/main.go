package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
	"github.com/MarkoPoloResearchLab/feedback_svc/internal/storage"
)

func main() {
	logger, loggerErr := zap.NewProduction()
	if loggerErr != nil {
		panic(fmt.Errorf("logger: %w", loggerErr))
	}
	defer logger.Sync()

	viper.SetDefault("APP_ADDR", ":8080")
	viper.SetDefault("DB_DSN", "")
	viper.SetDefault("ADMIN_BEARER_TOKEN", "")
	viper.AutomaticEnv()

	appAddr := viper.GetString("APP_ADDR")
	databaseDSN := viper.GetString("DB_DSN")
	adminBearerToken := strings.TrimSpace(viper.GetString("ADMIN_BEARER_TOKEN"))

	database, dbErr := storage.OpenPostgres(databaseDSN)
	if dbErr != nil {
		logger.Fatal("open_db", zap.Error(dbErr))
	}

	if migrateErr := storage.AutoMigrate(database); migrateErr != nil {
		logger.Fatal("migrate", zap.Error(migrateErr))
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	public := httpapi.NewPublicHandlers(database, logger)
	admin := httpapi.NewAdminHandlers(database, logger, adminBearerToken)
	adminWeb := httpapi.NewAdminWebHandlers(logger)

	router.POST("/api/feedback", public.CreateFeedback)
	router.GET("/widget.js", public.WidgetJS)
	router.GET("/admin", httpapi.AdminAuthMiddleware(adminBearerToken), adminWeb.RenderAdminInterface)

	adminGroup := router.Group("/api/admin")
	adminGroup.Use(httpapi.AdminAuthMiddleware(adminBearerToken))
	adminGroup.POST("/sites", admin.CreateSite)
	adminGroup.GET("/sites/:id/messages", admin.ListMessagesBySite)

	server := &http.Server{
		Addr:              appAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("listening", zap.String("addr", appAddr))
	if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		logger.Fatal("server", zap.Error(serveErr))
	}
}
