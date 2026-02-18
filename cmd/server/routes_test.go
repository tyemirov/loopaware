package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsPublicAPIPathClassification(testingT *testing.T) {
	testCases := []struct {
		name       string
		path       string
		expectOpen bool
	}{
		{
			name:       "empty",
			path:       "",
			expectOpen: false,
		},
		{
			name:       "feedback",
			path:       "/api/feedback",
			expectOpen: true,
		},
		{
			name:       "widget_config",
			path:       "/api/widget-config",
			expectOpen: true,
		},
		{
			name:       "visits",
			path:       "/api/visits",
			expectOpen: true,
		},
		{
			name:       "subscription_prefix",
			path:       "/api/subscriptions/confirm-link",
			expectOpen: true,
		},
		{
			name:       "authenticated_route",
			path:       "/api/sites",
			expectOpen: false,
		},
	}

	for _, testCase := range testCases {
		require.Equal(testingT, testCase.expectOpen, isPublicAPIPath(testCase.path), testCase.name)
	}
}

func TestAPIPreflightRoutesReturnCORSHeadersForAuthenticatedRequests(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	publicCORS := cors.New(cors.Config{
		AllowOrigins:     []string{corsOriginWildcard},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
	authenticatedOrigin := "http://localhost:8090"
	authenticatedCORS := cors.New(cors.Config{
		AllowOrigins:     []string{authenticatedOrigin},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
	registerAPIPreflightRoutes(router, publicCORS, authenticatedCORS)

	request := httptest.NewRequest(http.MethodOptions, "/api/sites", nil)
	request.Header.Set("Origin", authenticatedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusNoContent, recorder.Code)
	require.Equal(testingT, authenticatedOrigin, recorder.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(testingT, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
}

func TestAPIPreflightRoutesUseWildcardCORSForPublicRequests(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	publicCORS := cors.New(cors.Config{
		AllowOrigins:     []string{corsOriginWildcard},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
	authenticatedCORS := cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8090"},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
	registerAPIPreflightRoutes(router, publicCORS, authenticatedCORS)

	request := httptest.NewRequest(http.MethodOptions, "/api/feedback", nil)
	request.Header.Set("Origin", "http://widget.example")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusNoContent, recorder.Code)
	require.Equal(testingT, corsOriginWildcard, recorder.Header().Get("Access-Control-Allow-Origin"))
	require.Empty(testingT, recorder.Header().Get("Access-Control-Allow-Credentials"))
}

func TestAPIPreflightRoutesPreserveAbortStatusForRejectedOrigin(testingT *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	publicCORS := cors.New(cors.Config{
		AllowOrigins:     []string{corsOriginWildcard},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
	authenticatedCORS := cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8090"},
		AllowMethods:     corsAllowedMethods,
		AllowHeaders:     corsAllowedHeaders,
		ExposeHeaders:    corsExposedHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
	registerAPIPreflightRoutes(router, publicCORS, authenticatedCORS)

	request := httptest.NewRequest(http.MethodOptions, "/api/sites", nil)
	request.Header.Set("Origin", "http://unknown-origin.test")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(testingT, http.StatusForbidden, recorder.Code)
	require.NotEqual(testingT, http.StatusNoContent, recorder.Code)
}
