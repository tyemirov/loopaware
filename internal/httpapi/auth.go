package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/temirov/GAuss/pkg/constants"
	"github.com/temirov/GAuss/pkg/session"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/feedback_svc/internal/model"
)

const (
	contextKeyCurrentUser = "httpapi_current_user"
	authErrorUnauthorized = "unauthorized"
	authErrorForbidden    = "forbidden"
	logEventLoadSession   = "load_session"
	logEventPersistUser   = "persist_user"
	logEventFetchAvatar   = "fetch_avatar"
	avatarEndpointPath    = "/api/me/avatar"
	defaultAvatarMimeType = "application/octet-stream"
	maxAvatarBytes        = 1 << 20
)

var defaultAvatarFetchTimeout = 5 * time.Second

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type CurrentUser struct {
	Email      string
	Name       string
	PictureURL string
	IsAdmin    bool
}

type AuthManager struct {
	database     *gorm.DB
	logger       *zap.Logger
	sessionStore *sessions.CookieStore
	adminEmails  map[string]struct{}
	httpClient   HTTPClient
}

func NewAuthManager(database *gorm.DB, logger *zap.Logger, adminEmails []string, httpClient HTTPClient) *AuthManager {
	store := session.Store()
	adminMap := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		trimmedEmail := strings.ToLower(strings.TrimSpace(email))
		if trimmedEmail == "" {
			continue
		}
		adminMap[trimmedEmail] = struct{}{}
	}

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: defaultAvatarFetchTimeout}
	}

	return &AuthManager{
		database:     database,
		logger:       logger,
		sessionStore: store,
		adminEmails:  adminMap,
		httpClient:   client,
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

	localAvatarPath := ""
	if authManager.database != nil {
		persistedPath, persistErr := authManager.persistUser(context.Request.Context(), lowercaseEmail, name, pictureURL)
		if persistErr != nil {
			authManager.logger.Warn(logEventPersistUser, zap.Error(persistErr))
		} else {
			localAvatarPath = persistedPath
		}
	}

	currentUser := &CurrentUser{
		Email:   email,
		Name:    name,
		IsAdmin: isAdmin,
	}
	if localAvatarPath != "" {
		currentUser.PictureURL = localAvatarPath
	} else {
		currentUser.PictureURL = pictureURL
	}

	context.Set(contextKeyCurrentUser, currentUser)
	return currentUser, true
}

func (authManager *AuthManager) persistUser(ctx context.Context, lowercaseEmail string, name string, pictureURL string) (string, error) {
	var user model.User
	result := authManager.database.First(&user, "email = ?", lowercaseEmail)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		user = model.User{Email: lowercaseEmail}
	} else if result.Error != nil {
		return "", result.Error
	}

	user.Name = strings.TrimSpace(name)
	trimmedPictureURL := strings.TrimSpace(pictureURL)
	shouldFetchAvatar := false
	if trimmedPictureURL != "" {
		if user.PictureSourceURL == "" || user.PictureSourceURL != trimmedPictureURL {
			shouldFetchAvatar = true
		}
	}

	if shouldFetchAvatar {
		avatarData, contentType, fetchErr := authManager.fetchAvatar(ctx, trimmedPictureURL)
		if fetchErr != nil {
			authManager.logger.Warn(logEventFetchAvatar, zap.Error(fetchErr))
		} else {
			user.AvatarData = avatarData
			user.AvatarContentType = contentType
			user.PictureSourceURL = trimmedPictureURL
		}
	}

	if user.AvatarContentType == "" && len(user.AvatarData) > 0 {
		user.AvatarContentType = defaultAvatarMimeType
	}

	if saveErr := authManager.database.Save(&user).Error; saveErr != nil {
		return "", saveErr
	}

	if len(user.AvatarData) == 0 {
		return "", nil
	}

	return fmt.Sprintf("%s?v=%d", avatarEndpointPath, user.UpdatedAt.Unix()), nil
}

func (authManager *AuthManager) fetchAvatar(ctx context.Context, sourceURL string) ([]byte, string, error) {
	if authManager.httpClient == nil {
		return nil, "", errors.New("http client is not configured")
	}
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if reqErr != nil {
		return nil, "", reqErr
	}
	resp, respErr := authManager.httpClient.Do(req)
	if respErr != nil {
		return nil, "", respErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected avatar status: %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxAvatarBytes+1)
	data, readErr := io.ReadAll(limited)
	if readErr != nil {
		return nil, "", readErr
	}
	if len(data) > maxAvatarBytes {
		return nil, "", fmt.Errorf("avatar exceeds %d bytes", maxAvatarBytes)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = defaultAvatarMimeType
	}
	return data, contentType, nil
}

func extractString(value interface{}) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
