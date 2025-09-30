package httpapi_test

import (
	"net/http"
	testingpkg "testing"

	"github.com/stretchr/testify/require"
)

const (
	adminDashboardTitleText = "LoopAware Admin Dashboard"
	authorizationHeaderName = "Authorization"
	bearerTokenPrefix       = "Bearer "
	invalidBearerTokenValue = "invalid"
)

func TestAdminPageAccessibility(t *testingpkg.T) {
	apiHarness := buildAPIHarness(t)
	testCases := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name: "with bearer token",
			headers: map[string]string{
				authorizationHeaderName: bearerTokenPrefix + apiHarness.adminBearerToken,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing bearer token",
			headers:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid bearer token",
			headers: map[string]string{
				authorizationHeaderName: bearerTokenPrefix + invalidBearerTokenValue,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(testingT *testingpkg.T) {
			recorder := performJSONRequest(testingT, apiHarness.router, http.MethodGet, "/admin", nil, testCase.headers)
			require.Equal(testingT, testCase.expectedStatus, recorder.Code)
			require.Contains(testingT, recorder.Header().Get("Content-Type"), "text/html")
			require.Contains(testingT, recorder.Body.String(), adminDashboardTitleText)
		})
	}
}
