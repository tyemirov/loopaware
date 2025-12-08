package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tyemirov/GAuss/pkg/constants"
	"github.com/tyemirov/GAuss/pkg/gauss"
	"github.com/tyemirov/GAuss/pkg/session"
)

func TestGoogleAuthRedirectHonorsForwardedProtocol(t *testing.T) {
	session.NewSession([]byte(testSessionSecret))

	service, serviceErr := gauss.NewService(
		testGoogleClientID,
		testGoogleClientSecret,
		"http://loopaware.mprlab.com",
		dashboardRoute,
		gauss.ScopeStrings(gauss.DefaultScopes),
		"",
		gauss.WithLogoutRedirectURL(constants.LoginPath),
	)
	require.NoError(t, serviceErr)

	oauthHandlers, handlersErr := gauss.NewHandlers(service)
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
