package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testCurrentUserEmail = "user@example.com"

func TestSetCurrentUserRequiresContext(testingT *testing.T) {
	err := SetCurrentUser(nil, &CurrentUser{Email: testCurrentUserEmail})
	require.ErrorIs(testingT, err, ErrMissingContext)
}

func TestSetCurrentUserRequiresUser(testingT *testing.T) {
	responseRecorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(responseRecorder)

	err := SetCurrentUser(ginContext, nil)
	require.ErrorIs(testingT, err, ErrMissingCurrentUser)
}

func TestSetCurrentUserStoresUser(testingT *testing.T) {
	responseRecorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(responseRecorder)

	currentUser := &CurrentUser{Email: testCurrentUserEmail, Role: RoleUser}
	err := SetCurrentUser(ginContext, currentUser)
	require.NoError(testingT, err)

	savedUser, ok := CurrentUserFromContext(ginContext)
	require.True(testingT, ok)
	require.Equal(testingT, currentUser, savedUser)
}
