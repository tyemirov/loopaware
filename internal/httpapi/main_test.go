package httpapi_test

import (
	"fmt"
	"net/http"
	"os"
	testingpkg "testing"

	"github.com/stretchr/testify/require"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/testutil"
)

const (
	adminDashboardTitleText = "LoopAware Admin Dashboard"
	authorizationHeaderName = "Authorization"
	bearerTokenPrefix       = "Bearer "
)

func TestMain(m *testingpkg.M) {
	if err := testutil.StartEmbeddedPostgresOnce(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start embedded Postgres: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	testutil.StopEmbeddedPostgresOnce()
	os.Exit(code)
}

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
				authorizationHeaderName: bearerTokenPrefix + "invalid",
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
