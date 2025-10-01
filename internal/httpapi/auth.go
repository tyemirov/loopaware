package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/session"
	"go.uber.org/zap"
)

const (
	contextKeyCurrentUser = "httpapi_current_user"
	authErrorUnauthorized = "unauthorized"
	authErrorForbidden    = "forbidden"
	logEventLoadSession   = "load_session"
)

type CurrentUser struct {
	Email      string
	Name       string
	PictureURL string
	IsAdmin    bool
}

type AuthManager struct {
	logger       *zap.Logger
	sessionStore *sessions.CookieStore
	adminEmails  map[string]struct{}
}

func NewAuthManager(logger *zap.Logger, adminEmails []string) *AuthManager {
	store := session.Store()
	adminMap := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		trimmedEmail := strings.ToLower(strings.TrimSpace(email))
		if trimmedEmail == "" {
			continue
		}
		adminMap[trimmedEmail] = struct{}{}
	}

	return &AuthManager{
		logger:       logger,
		sessionStore: store,
		adminEmails:  adminMap,
	}
}

func (authManager *AuthManager) RequireAuthenticatedJSON() gin.HandlerFunc {
	return func(context *gin.Context) {
		if _, ok := authManager.ensureUser(context); !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
			return
		}
		context.Next()
	}
}

func (authManager *AuthManager) RequireAuthenticatedWeb() gin.HandlerFunc {
	return func(context *gin.Context) {
		if _, ok := authManager.ensureUser(context); !ok {
			context.Redirect(http.StatusFound, constants.LoginPath)
			context.Abort()
			return
		}
		context.Next()
	}
}

func (authManager *AuthManager) RequireAdminJSON() gin.HandlerFunc {
	return func(context *gin.Context) {
		currentUser, ok := authManager.ensureUser(context)
		if !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
			return
		}
		if !currentUser.IsAdmin {
			context.AbortWithStatusJSON(http.StatusForbidden, gin.H{jsonKeyError: authErrorForbidden})
			return
		}
		context.Next()
	}
}

func CurrentUserFromContext(context *gin.Context) (*CurrentUser, bool) {
	value, exists := context.Get(contextKeyCurrentUser)
	if !exists {
		return nil, false
	}
	currentUser, ok := value.(*CurrentUser)
	return currentUser, ok
}

func (authManager *AuthManager) ensureUser(context *gin.Context) (*CurrentUser, bool) {
	if currentUser, exists := CurrentUserFromContext(context); exists {
		return currentUser, true
	}

	sessionInstance, sessionErr := authManager.sessionStore.Get(context.Request, constants.SessionName)
	if sessionErr != nil {
		authManager.logger.Warn(logEventLoadSession, zap.Error(sessionErr))
		return nil, false
	}

	email := extractString(sessionInstance.Values[constants.SessionKeyUserEmail])
	if email == "" {
		return nil, false
	}

	name := extractString(sessionInstance.Values[constants.SessionKeyUserName])
	pictureURL := extractString(sessionInstance.Values[constants.SessionKeyUserPicture])
	lowercaseEmail := strings.ToLower(email)
	_, isAdmin := authManager.adminEmails[lowercaseEmail]

	currentUser := &CurrentUser{
		Email:      email,
		Name:       name,
		PictureURL: pictureURL,
		IsAdmin:    isAdmin,
	}

	context.Set(contextKeyCurrentUser, currentUser)
	return currentUser, true
}

func extractString(value interface{}) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
