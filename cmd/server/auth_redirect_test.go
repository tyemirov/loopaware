package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/auth"
	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/gauss"
	"github.com/temirov/GAuss/pkg/session"
)

func TestGoogleAuthRedirectHonorsForwardedProtocol(t *testing.T) {
	handlersConfiguration := auth.Config{
		GoogleClientID:     testGoogleClientID,
		GoogleClientSecret: testGoogleClientSecret,
		PublicBaseURL:      "http://loopaware.mprlab.com",
		LocalRedirectPath:  dashboardRoute,
		Scopes:             gauss.ScopeStrings(gauss.DefaultScopes),
		LoginTemplate:      "",
	}

	session.NewSession([]byte(testSessionSecret))

	oauthHandlers, handlersErr := auth.NewHandlers(handlersConfiguration)
	require.NoError(t, handlersErr)

	serveMux := http.NewServeMux()
	oauthHandlers.RegisterRoutes(serveMux)

	request := httptest.NewRequest(http.MethodGet, constants.GoogleAuthPath, nil)
	request.Host = "loopaware.mprlab.com"
	request.Header.Set("X-Forwarded-Proto", "https")

	recorder := httptest.NewRecorder()
	serveMux.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)

	redirectLocation := recorder.Header().Get("Location")
	require.NotEmpty(t, redirectLocation)

	redirectURL, parseErr := url.Parse(redirectLocation)
	require.NoError(t, parseErr)

	redirectURIValue := redirectURL.Query().Get("redirect_uri")
	require.Equal(t, "https://loopaware.mprlab.com/auth/google/callback", redirectURIValue)
}
