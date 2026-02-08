package api

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrMissingContext reports a nil gin context when setting the current user.
var ErrMissingContext = errors.New("missing context")

// ErrMissingCurrentUser reports a nil current user when setting auth state.
var ErrMissingCurrentUser = errors.New("missing current user")

// SetCurrentUser stores the authenticated user in the request context.
func SetCurrentUser(context *gin.Context, currentUser *CurrentUser) error {
	if context == nil {
		return ErrMissingContext
	}
	if currentUser == nil {
		return ErrMissingCurrentUser
	}
	context.Set(contextKeyCurrentUser, currentUser)
	return nil
}
