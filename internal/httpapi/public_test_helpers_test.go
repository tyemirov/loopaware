package httpapi_test

import (
	"github.com/gin-gonic/gin"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/httpapi"
)

type stubCurrentUserProvider struct {
	authenticated bool
}

func (provider *stubCurrentUserProvider) CurrentUser(context *gin.Context) (*httpapi.CurrentUser, bool) {
	if provider == nil || !provider.authenticated {
		return nil, false
	}
	return &httpapi.CurrentUser{
		Email: "user@example.com",
		Name:  "Authenticated User",
		Role:  httpapi.RoleUser,
	}, true
}
