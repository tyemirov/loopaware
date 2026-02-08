package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func buildTestContext(testingT *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return context, recorder
}

func TestRequireAdminJSONRejectsUnauthenticated(testingT *testing.T) {
	authManager := &AuthManager{logger: zap.NewNop()}
	handler := authManager.RequireAdminJSON()

	context, recorder := buildTestContext(testingT)
	handler(context)

	require.Equal(testingT, http.StatusUnauthorized, recorder.Code)
}

func TestRequireAdminJSONRejectsNonAdmin(testingT *testing.T) {
	authManager := &AuthManager{logger: zap.NewNop()}
	handler := authManager.RequireAdminJSON()

	context, recorder := buildTestContext(testingT)
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: "user@example.com", Role: RoleUser})
	handler(context)

	require.Equal(testingT, http.StatusForbidden, recorder.Code)
}

func TestRequireAdminJSONAllowsAdmin(testingT *testing.T) {
	authManager := &AuthManager{logger: zap.NewNop()}
	handler := authManager.RequireAdminJSON()

	context, recorder := buildTestContext(testingT)
	context.Set(contextKeyCurrentUser, &CurrentUser{Email: "admin@example.com", Role: RoleAdmin})
	handler(context)

	require.Equal(testingT, http.StatusOK, recorder.Code)
}
