package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tyemirov/tauth/pkg/sessionvalidator"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
)

const (
	contextKeyCurrentUser = "api_current_user"
	authErrorUnauthorized = "unauthorized"
	authErrorForbidden    = "forbidden"
	logEventLoadSession   = "load_session"
	logEventPersistUser   = "persist_user"
	logEventFetchAvatar   = "fetch_avatar"
	avatarEndpointPath    = "/api/me/avatar"
	defaultAvatarMimeType = "application/octet-stream"
	defaultAvatarDataURI  = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI2NCIgaGVpZ2h0PSI2NCIgdmlld0JveD0iMCAwIDY0IDY0IiByb2xlPSJpbWciIGFyaWEtbGFiZWw9IlVzZXIiPgogIDxyZWN0IHdpZHRoPSI2NCIgaGVpZ2h0PSI2NCIgcng9IjMyIiBmaWxsPSIjMzM0MTU1Ii8+CiAgPHBhdGggZmlsbD0iI2UyZThmMCIgZD0iTTMyIDM0YzYuNjI3IDAgMTItNS4zNzMgMTItMTJTMzguNjI3IDEwIDMyIDEwIDIwIDE1LjM3MyAyMCAyMnM1LjM3MyAxMiAxMiAxMnptMCA0Yy0xMC40OTMgMC0xOSA2LjUwNy0xOSAxNC41VjU2aDM4di0zLjVDNTEgNDQuNTA3IDQyLjQ5MyAzOCAzMiAzOHoiLz4KPC9zdmc+Cg=="
	maxAvatarBytes        = 1 << 20
)

var defaultAvatarFetchTimeout = 5 * time.Second

type persistedUserSnapshot struct {
	Email             string
	Name              string
	PictureSourceURL  string
	AvatarContentType string
	AvatarSize        int64 `gorm:"column:avatar_size"`
	UpdatedAt         time.Time
}

// UserRole enumerates the supported access levels for authenticated dashboard users.
type UserRole string

const (
	// RoleAdmin grants full access to every site in the system.
	RoleAdmin UserRole = "admin"
	// RoleUser restricts access to sites created or owned by the caller.
	RoleUser UserRole = "user"
)

// HTTPClient executes outbound HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// CurrentUser captures authenticated account metadata made available to handlers.
type CurrentUser struct {
	Email      string
	Name       string
	PictureURL string
	Role       UserRole
}

func (currentUser *CurrentUser) hasRole(role UserRole) bool {
	if currentUser == nil {
		return false
	}
	return currentUser.Role == role
}

func (currentUser *CurrentUser) normalizedEmail() string {
	if currentUser == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(currentUser.Email))
}

func (currentUser *CurrentUser) canManageSite(site model.Site) bool {
	if currentUser == nil {
		return false
	}
	if currentUser.hasRole(RoleAdmin) {
		return true
	}
	normalized := currentUser.normalizedEmail()
	if normalized == "" {
		return false
	}
	if strings.EqualFold(site.OwnerEmail, normalized) {
		return true
	}
	return strings.EqualFold(site.CreatorEmail, normalized)
}

// AuthManager resolves authenticated dashboard users and enforces access policies.
type AuthManager struct {
	database         *gorm.DB
	logger           *zap.Logger
	adminEmails      map[string]struct{}
	httpClient       HTTPClient
	sessionValidator *sessionvalidator.Validator
	expectedTenantID string
}

// AuthConfig captures the TAuth validation configuration.
type AuthConfig struct {
	SigningKey string
	CookieName string
	TenantID   string
}

// NewAuthManager constructs an AuthManager with the provided dependencies.
func NewAuthManager(database *gorm.DB, logger *zap.Logger, adminEmails []string, httpClient HTTPClient, authConfig AuthConfig) (*AuthManager, error) {
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

	validatorConfig := sessionvalidator.Config{
		SigningKey: []byte(strings.TrimSpace(authConfig.SigningKey)),
		CookieName: strings.TrimSpace(authConfig.CookieName),
	}
	sessionValidator, validatorErr := sessionvalidator.New(validatorConfig)
	if validatorErr != nil {
		return nil, validatorErr
	}

	return &AuthManager{
		database:         database,
		logger:           logger,
		adminEmails:      adminMap,
		httpClient:       client,
		sessionValidator: sessionValidator,
		expectedTenantID: strings.TrimSpace(authConfig.TenantID),
	}, nil
}

// RequireAuthenticatedJSON enforces authentication for JSON API routes.
func (authManager *AuthManager) RequireAuthenticatedJSON() gin.HandlerFunc {
	return func(context *gin.Context) {
		if _, ok := authManager.ensureUser(context); !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
			return
		}
		context.Next()
	}
}

// CurrentUser returns the authenticated account associated with the request if available.
func (authManager *AuthManager) CurrentUser(context *gin.Context) (*CurrentUser, bool) {
	return authManager.ensureUser(context)
}

// RequireAdminJSON enforces administrator access for JSON API routes.
func (authManager *AuthManager) RequireAdminJSON() gin.HandlerFunc {
	return func(context *gin.Context) {
		currentUser, ok := authManager.ensureUser(context)
		if !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{jsonKeyError: authErrorUnauthorized})
			return
		}
		if !currentUser.hasRole(RoleAdmin) {
			context.AbortWithStatusJSON(http.StatusForbidden, gin.H{jsonKeyError: authErrorForbidden})
			return
		}
		context.Next()
	}
}

// CurrentUserFromContext loads the current user from the request context.
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

	if authManager.sessionValidator == nil {
		authManager.logger.Warn(logEventLoadSession, zap.Error(sessionvalidator.ErrMissingSigningKey))
		return nil, false
	}

	claims, validationErr := authManager.sessionValidator.ValidateRequest(context.Request)
	if validationErr != nil {
		authManager.logger.Warn(logEventLoadSession, zap.Error(validationErr))
		return nil, false
	}
	expectedTenantID := authManager.expectedTenantID
	if expectedTenantID != "" && !strings.EqualFold(claims.GetTenantID(), expectedTenantID) {
		authManager.logger.Warn(logEventLoadSession, zap.Error(sessionvalidator.ErrInvalidToken))
		return nil, false
	}

	email := strings.TrimSpace(claims.GetUserEmail())
	if email == "" {
		return nil, false
	}

	name := strings.TrimSpace(claims.GetUserDisplayName())
	pictureURL := strings.TrimSpace(claims.GetUserAvatarURL())
	lowercaseEmail := strings.ToLower(email)
	userRole := RoleUser
	if _, isPrivileged := authManager.adminEmails[lowercaseEmail]; isPrivileged {
		userRole = RoleAdmin
	}

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
		Email: email,
		Name:  name,
		Role:  userRole,
	}
	if localAvatarPath != "" {
		currentUser.PictureURL = localAvatarPath
	} else {
		currentUser.PictureURL = pictureURL
	}
	if currentUser.PictureURL == "" {
		currentUser.PictureURL = defaultAvatarDataURI
	}

	context.Set(contextKeyCurrentUser, currentUser)
	return currentUser, true
}

func (authManager *AuthManager) persistUser(ctx context.Context, lowercaseEmail string, name string, pictureURL string) (string, error) {
	trimmedName := strings.TrimSpace(name)
	trimmedPictureURL := strings.TrimSpace(pictureURL)
	database := authManager.database.WithContext(ctx)

	var snapshot persistedUserSnapshot
	loadErr := database.Model(&model.User{}).
		Select("email", "name", "picture_source_url", "avatar_content_type", "coalesce(length(avatar_data), 0) as avatar_size", "updated_at").
		First(&snapshot, "email = ?", lowercaseEmail).Error
	if errors.Is(loadErr, gorm.ErrRecordNotFound) {
		user := model.User{
			Email: lowercaseEmail,
			Name:  trimmedName,
		}
		if trimmedPictureURL != "" {
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
		if createErr := database.Create(&user).Error; createErr != nil {
			return "", createErr
		}
		if len(user.AvatarData) == 0 {
			return "", nil
		}
		return fmt.Sprintf("%s?v=%d", avatarEndpointPath, user.UpdatedAt.Unix()), nil
	}
	if loadErr != nil {
		return "", loadErr
	}

	updates := make(map[string]any)
	if trimmedName != snapshot.Name {
		updates["name"] = trimmedName
	}

	shouldFetchAvatar := trimmedPictureURL != "" && (snapshot.PictureSourceURL == "" || snapshot.PictureSourceURL != trimmedPictureURL)
	avatarUpdated := false
	if shouldFetchAvatar {
		avatarData, contentType, fetchErr := authManager.fetchAvatar(ctx, trimmedPictureURL)
		if fetchErr != nil {
			authManager.logger.Warn(logEventFetchAvatar, zap.Error(fetchErr))
		} else {
			updates["avatar_data"] = avatarData
			updates["avatar_content_type"] = contentType
			updates["picture_source_url"] = trimmedPictureURL
			avatarUpdated = true
		}
	}

	hasAvatar := snapshot.AvatarSize > 0
	if avatarUpdated {
		if avatarContentTypeValue, ok := updates["avatar_content_type"].(string); ok {
			if avatarContentTypeValue == "" {
				updates["avatar_content_type"] = defaultAvatarMimeType
			}
		}
		if avatarDataValue, ok := updates["avatar_data"].([]byte); ok {
			hasAvatar = len(avatarDataValue) > 0
		}
	}

	if !avatarUpdated && snapshot.AvatarContentType == "" && hasAvatar {
		updates["avatar_content_type"] = defaultAvatarMimeType
		avatarUpdated = true
	}

	avatarVersionUnix := snapshot.UpdatedAt.Unix()
	if avatarUpdated {
		updateTimestamp := time.Now().UTC()
		updates["updated_at"] = updateTimestamp
		avatarVersionUnix = updateTimestamp.Unix()
	}

	if len(updates) == 0 {
		if !hasAvatar {
			return "", nil
		}
		return fmt.Sprintf("%s?v=%d", avatarEndpointPath, avatarVersionUnix), nil
	}

	if updateErr := database.Model(&model.User{}).Where("email = ?", lowercaseEmail).Updates(updates).Error; updateErr != nil {
		return "", updateErr
	}

	if !hasAvatar {
		return "", nil
	}

	return fmt.Sprintf("%s?v=%d", avatarEndpointPath, avatarVersionUnix), nil
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
